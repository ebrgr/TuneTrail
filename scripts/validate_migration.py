#!/usr/bin/env python3
"""Verify independent v2-to-v3 legacy migrations and the Go/API journeys after them.

Formerly validate_migration_030.py; the migrate() and route_matrix() helpers come
from the historical validate_migration_025.py, reduced to the Go interface.
"""
import argparse
import contextlib
import json
import sqlite3
import tempfile
import urllib.error
import urllib.request
from pathlib import Path

from roadmap06_fixture import FIXTURE_PASSWORD_HASH
from validate_admin_webui import validate as validate_admin
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


def migrate(binary, root, name, stack, wal=False):
    """Start one independently created legacy database and verify its migration."""
    directory = root / name
    directory.mkdir()
    database = directory / 'koito.db'
    create_legacy(database, FIXTURE_PASSWORD_HASH.encode('ascii'))
    if wal:
        # A committed row that exists only in the WAL must survive the snapshot.
        legacy_conn = sqlite3.connect(database)
        stack.callback(legacy_conn.close)
        assert legacy_conn.execute('PRAGMA journal_mode=WAL').fetchone()[0] == 'wal'
        legacy_conn.execute("INSERT INTO listens(track_id,listened_at,user_id,client) VALUES(1,1700000001,1,'fictional-wal')")
        legacy_conn.commit()
        assert Path(str(database)+'-wal').exists(), 'committed WAL was not present'
    before = snapshot(database)
    with contextlib.closing(sqlite3.connect(database)) as conn:
        assert conn.execute('SELECT MAX(version_id) FROM goose_db_version WHERE is_applied=1').fetchone() == (2,)
    server = Server(binary, directory, environment={'KOITO_FORCE_TZ': 'UTC'})
    stack.callback(server.stop)
    server.start()
    after = snapshot(directory / "tunetrail.db")
    assert all(after[table] == digest for table, digest in before.items()), 'legacy columns changed during migration'
    backups = list(directory.glob('koito.db.pre-tunetrail-*.bak'))
    assert len(backups) == 1 and snapshot(backups[0]) == before, 'missing or incorrect v2 backup'
    # Restore the retained backup into an isolated copy and compare it with the source.
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


def route_matrix(server):
    """Check actual method, content and authentication behavior after migration."""
    rows = []
    for path, expected in [('/',200), ('/default',200), ('/admin',200), ('/favicon.svg',200),
                           ('/public/yuu.jpg',200), ('/apis/web/v1/profiles',200),
                           ('/apis/web/v1/health',200), ('/web-assets/ui.css',200),
                           ('/theme-helper',404), ('/assets/not-a-real-file.js',404),
                           ('/default/not-a-real-page',404), ('/listens',302)]:
        for method in ('GET','HEAD'):
            status, headers, body = http(server, method, path)
            assert status == expected, (method,path,status,expected)
            if method == 'HEAD': assert not body, path
            if path == '/' and method == 'GET':
                assert b'data-assets="/web-assets"' in body
            if path == '/favicon.svg': assert 'image/svg+xml' in headers.get('Content-Type','')
            if path == '/web-assets/ui.css' and method == 'GET':
                assert b'data-assets="/web-assets"' not in body
                assert 'text/css' in headers.get('Content-Type','')
            if path == '/listens': assert headers.get('Location') == '/default/listens'
            rows.append({'method':method,'path':path,'status':status,'content_type':headers.get('Content-Type','').split(';')[0]})
    status, _, _ = http(server, 'POST', WEB+'/profiles')
    assert status in (401,403), ('unauthorized profile create',status)
    rows.append({'method':'POST','path':WEB+'/profiles','status':status,'actor':'visitor'})
    return rows


def main():
    """Use independent v2 databases, including a committed WAL snapshot."""
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--binary', type=Path, required=True)
    args = parser.parse_args()
    with tempfile.TemporaryDirectory(prefix='tunetrail-migration-') as tmp, contextlib.ExitStack() as stack:
        root, binary = Path(tmp), args.binary.resolve(strict=True)
        web = migrate(binary, root, 'web', stack)
        matrix = route_matrix(web)
        admin = migrate(binary, root, 'admin', stack)
        validate_admin(admin)
        api = migrate(binary, root, 'api-wal', stack, wal=True)
        validate_journey(api)
        for server in (web, admin, api):
            with contextlib.closing(sqlite3.connect(server.directory / 'tunetrail.db')) as conn:
                assert conn.execute('PRAGMA integrity_check').fetchall() == [('ok',)]
                assert not conn.execute('PRAGMA foreign_key_check').fetchall()
        print(json.dumps({'migrations':3, 'wal': 'ok', 'backup_restore': 'ok', 'route_checks':len(matrix), 'admin_forms':'ok', 'api':'ok', 'integrity':'ok'}))


if __name__ == '__main__':
    main()
