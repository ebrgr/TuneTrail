#!/usr/bin/env python3
"""Exercise independent v2-to-v3 upgrades in Go and React/API on local fixtures."""
import argparse
import contextlib
import hashlib
import json
import os
import sqlite3
import subprocess
import tempfile
import urllib.error
import urllib.request
from pathlib import Path

from roadmap06_fixture import FIXTURE_PASSWORD_HASH
from validate_admin_webui import validate as validate_go_admin
from validate_multiuser import Server, WEB, LBZ, create_legacy, snapshot, validate_journey


class NoRedirect(urllib.request.HTTPRedirectHandler):
    """Keep redirects observable instead of following them automatically."""
    def redirect_request(self, *_):
        return None


def http(server, method, path):
    """Read a local route and retain status, headers and body for the matrix."""
    req = urllib.request.Request(server.origin + path, method=method)
    try:
        response = urllib.request.build_opener(NoRedirect()).open(req, timeout=10)
    except urllib.error.HTTPError as err:
        response = err
    with response:
        return response.status, response.headers, response.read()


def migrate(binary, root, name, mode, stack, wal=False):
    """Start one independently created legacy database and verify its migration."""
    directory = root / name
    directory.mkdir()
    database = directory / 'koito.db'
    create_legacy(database, FIXTURE_PASSWORD_HASH.encode('ascii'))
    if wal:
        legacy_conn = sqlite3.connect(database)
        stack.callback(legacy_conn.close)
        assert legacy_conn.execute('PRAGMA journal_mode=WAL').fetchone()[0] == 'wal'
        legacy_conn.execute("INSERT INTO listens(track_id,listened_at,user_id,client) VALUES(1,1700000001,1,'fictional-wal')")
        legacy_conn.commit()
        assert Path(str(database)+'-wal').exists(), 'committed WAL was not present'
    before = snapshot(database)
    with contextlib.closing(sqlite3.connect(database)) as conn:
        assert conn.execute('SELECT MAX(version_id) FROM goose_db_version WHERE is_applied=1').fetchone() == (2,)
    server = Server(binary, directory, environment={'KOITO_WEB_UI': mode, 'KOITO_FORCE_TZ': 'UTC'})
    stack.callback(server.stop)
    server.start()
    after = snapshot(directory / "tunetrail.db")
    assert all(after[table] == digest for table, digest in before.items()), 'legacy columns changed during migration'
    backups = list(directory.glob('koito.db.pre-tunetrail-*.bak'))
    assert len(backups) == 1 and snapshot(backups[0]) == before, 'missing or incorrect v2 backup'
    restored = directory / 'restored-test.db'
    with contextlib.closing(sqlite3.connect(f'{backups[0].resolve().as_uri()}?mode=ro',uri=True)) as src, contextlib.closing(sqlite3.connect(restored)) as dst:
        src.backup(dst)
    assert snapshot(restored) == before
    with contextlib.closing(sqlite3.connect(restored)) as conn:
        assert conn.execute('SELECT MAX(version_id) FROM goose_db_version WHERE is_applied=1').fetchone() == (2,)
        assert conn.execute('PRAGMA integrity_check').fetchall() == [('ok',)]
    with contextlib.closing(sqlite3.connect(directory / 'tunetrail.db')) as conn:
        assert conn.execute('SELECT MAX(version_id) FROM goose_db_version WHERE is_applied=1').fetchone() == (3,)
        assert conn.execute('PRAGMA integrity_check').fetchall() == [('ok',)]
        assert not conn.execute('PRAGMA foreign_key_check').fetchall()
    assert server.request('GET', WEB + '/profiles/default/stats?period=all_time')['listen_count'] == (2 if wal else 1)
    assert server.request('GET', LBZ + '/validate-token', token='legacy-validation-token')['valid'] is True
    server.legacy_hashes = before
    return server


def route_matrix(server, mode):
    """Check actual method, content and authentication behavior after migration."""
    rows = []
    for path, expected in [('/',200), ('/default',200), ('/admin',200), ('/favicon.svg',200),
                           ('/public/yuu.jpg',200), ('/apis/web/v1/profiles',200),
                           ('/apis/web/v1/health',200), ('/web-assets/ui.css',200 if mode=='go' else 200),
                           ('/theme-helper',404 if mode=='go' else 200),
                           ('/assets/not-a-real-file.js',404 if mode=='go' else 200),
                           ('/default/not-a-real-page',404 if mode=='go' else 200),
                           ('/listens',302 if mode=='go' else 200)]:
        for method in ('GET','HEAD'):
            status, headers, body = http(server, method, path)
            assert status == expected, (mode,method,path,status,expected)
            if method == 'HEAD': assert not body, (mode,path)
            if path == '/' and method == 'GET':
                assert (b'data-assets="/web-assets"' in body) == (mode=='go')
            if path == '/favicon.svg': assert 'image/svg+xml' in headers.get('Content-Type','')
            if path == '/web-assets/ui.css' and method == 'GET':
                assert (b'data-assets="/web-assets"' not in body)
                assert ('text/css' in headers.get('Content-Type','')) == (mode=='go')
            if path == '/listens' and mode == 'go': assert headers.get('Location') == '/default/listens'
            rows.append({'mode':mode,'method':method,'path':path,'status':status,'content_type':headers.get('Content-Type','').split(';')[0]})
    status, _, _ = http(server, 'POST', WEB+'/profiles')
    assert status in (401,403), (mode,'unauthorized profile create',status)
    rows.append({'mode':mode,'method':'POST','path':WEB+'/profiles','status':status,'actor':'visitor'})
    return rows


def main():
    """Run four isolated migrations, two web sessions and two admin journeys."""
    parser=argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--binary',type=Path,required=True)
    parser.add_argument('--report',type=Path,required=True,help='new output report; contains no credentials')
    args=parser.parse_args()
    if args.report.exists(): raise FileExistsError(args.report)
    binary=args.binary.resolve(strict=True)
    with tempfile.TemporaryDirectory(prefix='koito-025-') as tmp, contextlib.ExitStack() as stack:
        root=Path(tmp)
        web_go=migrate(binary,root,'web-go','go',stack)
        web_react=migrate(binary,root,'web-react','react',stack)
        matrix=route_matrix(web_go,'go')+route_matrix(web_react,'react')
        for server in (web_go,web_react):
            server.login()
            assert len(server.request('GET',WEB+'/profiles/default/export',admin=True)['listens'])==1
        env=os.environ.copy()
        browser = subprocess.run(['node','scripts/validate_migration_browser_025.cjs',web_go.origin,web_react.origin],env=env,check=True,capture_output=True,text=True)
        print(browser.stdout, end='')
        helper_status = next((line.split('=',1)[1] for line in browser.stdout.splitlines() if line.startswith('THEME_HELPER=')), None)
        assert helper_status in ('context-error','editor-visible')
        # Use independent copies because both admin journeys mutate and delete data.
        admin_go=migrate(binary,root,'admin-go','go',stack)
        go_report=validate_go_admin(admin_go)
        admin_react=migrate(binary,root,'admin-react','react',stack,wal=True)
        validate_journey(admin_react)
        for server in (web_go,web_react,admin_go,admin_react):
            with contextlib.closing(sqlite3.connect(server.directory/'koito.db')) as conn:
                assert conn.execute('PRAGMA integrity_check').fetchall()==[('ok',)]
                assert not conn.execute('PRAGMA foreign_key_check').fetchall()
        report={'migration':'four independent v2 databases migrated at startup to v3',
                'legacy_fixture':'one admin, one profile, one key, one track and one listen; second committed listen in WAL for admin-react',
                'legacy_columns':'hashes preserved before authentication', 'legacy_hashes':web_go.legacy_hashes,
                'wal_legacy_hashes':admin_react.legacy_hashes,
                'binary_sha256':hashlib.sha256(binary.read_bytes()).hexdigest(),
                'source_revision':subprocess.check_output(['git','rev-parse','HEAD'],text=True).strip(),
                'source_state':'local uncommitted changes present' if subprocess.check_output(['git','status','--porcelain'],text=True).strip() else 'clean working tree',
                'backups':'four v2 backups matched source hashes and were restored into isolated copies',
                'web':'Go/React browser with hydrated UI, legacy redirect and scoped profile',
                'theme_helper':helper_status,
                'admin_go':go_report, 'admin_react':'validate_journey JSON/API plus independent browser login',
                'integrity':'ok','routes':matrix}
        args.report.write_text(json.dumps(report,indent=2,ensure_ascii=False)+'\n')
        print(json.dumps({'migrations':4,'backup':'ok','route_checks':len(matrix),'browser':'go/react','admin':'go forms/react API','integrity':'ok'}))


if __name__=='__main__': main()
