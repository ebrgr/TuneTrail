#!/usr/bin/env python3
"""Exercise authorization and secret handling on disposable real local servers."""
import hashlib
import json
import re
import sqlite3
import tempfile
import time
import urllib.error
import urllib.parse
import urllib.request
import uuid
from pathlib import Path

from validate_multiuser import Server, WEB, PASSWORD
from validate_admin_webui import NoRedirect


class Client:
    """Send requests without following redirects or printing response secrets."""
    def __init__(self, server):
        self.server = server
        self.opener = urllib.request.build_opener(NoRedirect())

    def request(self, method, path, *, cookie=None, token=None, data=None, headers=None):
        fields = dict(headers or {})
        if cookie is not None:
            fields['Cookie'] = cookie
        if token:
            fields['Authorization'] = 'Token ' + token
        req = urllib.request.Request(self.server.origin + path, data=data, headers=fields, method=method)
        try:
            response = self.opener.open(req, timeout=20)
        except urllib.error.HTTPError as err:
            response = err
        with response:
            return response.status, response.headers, response.read()


def expect(status, wanted, what):
    """Report only route and status, never body or credentials."""
    if status != wanted:
        raise AssertionError(f'{what}: expected {wanted}, got {status}')


def staged_files(directory):
    """Hash application file contents without comparing database WAL or access logs."""
    return sorted((str(p.relative_to(directory)), hashlib.sha256(p.read_bytes()).hexdigest())
                  for p in directory.rglob('*') if p.is_file()
                  and p.name not in ('tunetrail.db', 'tunetrail.db-wal', 'tunetrail.db-shm', 'server.log'))


def full_snapshot(path):
    """Hash all columns, including profile ownership, without disclosing rows."""
    result = {}
    with sqlite3.connect(path) as db:
        tables = db.execute("SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%' ORDER BY name").fetchall()
        for (name,) in tables:
            quoted = '"' + name.replace('"', '""') + '"'
            rows = sorted(repr(row) for row in db.execute('SELECT * FROM ' + quoted))
            result[name] = hashlib.sha256('\n'.join(rows).encode()).hexdigest()
    return result


def run_mode(binary, root, mode):
    """Test all protected JSON families in both modes and HTML in Go mode."""
    server = Server(binary, root / mode, environment={'KOITO_WEB_UI': mode, 'KOITO_LOG_LEVEL': 'debug'})
    checks = 0
    try:
        server.start()
        c = Client(server)
        raw = json.dumps({'username': 'validation-admin', 'password': PASSWORD}).encode()
        status, headers, _ = c.request('POST', WEB+'/login', data=raw, headers={'Content-Type':'application/json'})
        expect(status, 204, 'login')
        admin_cookie = headers['Set-Cookie'].split(';', 1)[0]
        session = admin_cookie.split('=', 1)[1]
        assert 'HttpOnly' in headers['Set-Cookie'] and 'SameSite=Strict' in headers['Set-Cookie']
        assert 'Secure' not in headers['Set-Cookie']
        with sqlite3.connect(server.directory/'tunetrail.db') as db:
            db.execute("INSERT INTO users(username,role,password) VALUES('fictional-user','user',x'00')")
            nonadmin = db.execute("SELECT id FROM users WHERE username='fictional-user'").fetchone()[0]
            now = int(time.time())
            nonadmin_session = str(uuid.uuid4())
            expired_session = str(uuid.uuid4())
            db.execute('INSERT INTO sessions(id,user_id,created_at,expires_at,persistent) VALUES(?,?,?,?,0)', (nonadmin_session,nonadmin,now,now+3600))
            db.execute('INSERT INTO sessions(id,user_id,created_at,expires_at,persistent) VALUES(?,?,?,?,0)', (expired_session,1,now-7200,now-3600))
            db.execute("INSERT INTO api_keys(key,profile_id,created_at,label) VALUES('fixture-second-player',2,?,'fixture')", (now,))
            db.commit()
            key = db.execute('SELECT key FROM api_keys WHERE profile_id=1 LIMIT 1').fetchone()[0]
        actors = [
            ('visitor', None, None, 401), ('key', None, key, 401), ('invalid-key', None, 'invalid-fixture-key', 401),
            ('invalid-cookie', 'koito_session=invalid', None, 401),
            ('duplicate-cookie', admin_cookie+'; koito_session=invalid', None, 401),
            ('duplicate-cookie-reverse', 'koito_session=invalid; '+admin_cookie, None, 401),
            ('expired', 'koito_session='+expired_session, None, 401),
            ('revoked-session', 'koito_session='+str(uuid.uuid4()), None, 401),
            ('other-profile-key', None, 'fixture-second-player', 401),
            ('non-admin', 'koito_session='+nonadmin_session, None, 403),
        ]
        prefix = WEB+'/profiles/default'
        routes = [
            ('POST', WEB+'/profiles'), ('PATCH',prefix), ('DELETE',prefix), ('GET',prefix+'/management'),
            ('PUT',prefix+'/avatar'), ('DELETE',prefix+'/avatar'), ('GET',prefix+'/export'), ('HEAD',prefix+'/export'),
            ('POST',prefix+'/import'), ('GET',prefix+'/relay'), ('PUT',prefix+'/relay'),
            ('DELETE',prefix+'/relay'), ('POST',prefix+'/relay/test'), ('POST',prefix+'/listens'),
            ('DELETE',prefix+'/listens'), ('GET',prefix+'/user/apikeys'), ('POST',prefix+'/user/apikeys'),
            ('PATCH',prefix+'/user/apikeys/1'), ('DELETE',prefix+'/user/apikeys/1'),
            ('GET',WEB+'/user'), ('PATCH',WEB+'/user'), ('DELETE',WEB+'/data'),
        ]
        for kind in ('artist','album','track'):
            base=WEB+'/'+kind+'/1'
            routes += [('DELETE',base),('PATCH',base),('POST',base+'/merge'),
                       ('GET',WEB+'/user') ,('POST',base+'/aliases'),('DELETE',base+'/aliases'),
                       ('PATCH',base+'/aliases/primary')]
            if kind != 'track': routes.append(('PATCH',base+'/image'))
        routes += [('PATCH',WEB+'/album/1/artists/1'),('POST',WEB+'/track/1/artists'),
                   ('PATCH',WEB+'/track/1/artists/1'),('DELETE',WEB+'/track/1/artists/1')]
        if mode == 'go':
            routes += [('GET','/admin/profile/1/export'),('HEAD','/admin/profile/1/export'),
                       ('POST','/admin/profile/create'),('POST','/admin/profile/1/save'),
                       ('POST','/admin/profile/1/avatar'),('POST','/admin/profile/1/avatar-clear'),
                       ('POST','/admin/profile/1/delete'),('POST','/admin/profile/1/import'),
                       ('POST','/admin/profile/1/keys/create'),('POST','/admin/profile/1/keys/1/label'),
                       ('POST','/admin/profile/1/keys/1/revoke'),('POST','/admin/profile/1/relay/save'),
                       ('POST','/admin/profile/1/relay/test'),('POST','/admin/profile/1/relay/remove'),
                       ('POST','/admin/account/password'),('POST','/admin/account/username')]
            for kind in ('artist', 'album', 'track'):
                for action in ('alias-add','alias-primary','alias-delete','metadata','artist-add','artist-remove','primary-artist','listen-add','listen-delete','artwork','confirm'):
                    routes.append(('POST','/default/manage/'+kind+'/1/'+action))
        baseline = full_snapshot(server.directory/'tunetrail.db')
        file_baseline = staged_files(server.directory)
        for method,path in routes:
            for actor,cookie,token,want in actors:
                status, headers, body = c.request(method,path,cookie=cookie,token=token)
                expect(status,want,f'{mode} {actor} {method} {path}')
                assert 'no-store' in headers.get('Cache-Control',''), (mode,method,path,actor)
                assert PASSWORD.encode() not in body and key.encode() not in body
                checks += 1
        assert full_snapshot(server.directory/'tunetrail.db') == baseline, 'negative request mutated SQLite'
        assert staged_files(server.directory) == file_baseline, 'negative request left files behind'
        # The valid session remains administrative even with a conflicting player key.
        status,headers,_=c.request('GET',WEB+'/user',cookie=admin_cookie,token='fixture-second-player')
        expect(status,200,'admin session plus another player key')
        assert 'no-store' in headers.get('Cache-Control','')
        # Origin absent is accepted; malformed, mismatched and cross-site requests are rejected first.
        for origin,site,want in [('', '', 401), (server.origin, '', 401), ('null', '', 403),
                                  ('https://foreign.invalid','',403), (server.origin+'.evil.invalid','',403),
                                  (server.origin.replace('http:', 'https:'),'',403),
                                  ('http://127.0.0.1:1','',403),
                                  (server.origin,'cross-site',403)]:
            hdr={'Content-Type':'application/json'}
            if origin: hdr['Origin']=origin
            if site: hdr['Sec-Fetch-Site']=site
            status,headers,_=c.request('PATCH',WEB+'/user',data=b'{}',headers=hdr)
            expect(status,want,'origin/fetch-metadata')
            assert 'no-store' in headers.get('Cache-Control','')
            checks += 1
        # Secure cookies follow the documented trusted-proxy X-Forwarded-Proto contract.
        status,headers,_=c.request('POST',WEB+'/login',data=raw,
                                   headers={'Content-Type':'application/json','X-Forwarded-Proto':'https'})
        expect(status,204,'proxy login')
        assert 'Secure' in headers['Set-Cookie'] and 'HttpOnly' in headers['Set-Cookie']
        proxy_cookie=headers['Set-Cookie'].split(';',1)[0]
        expect(c.request('POST',WEB+'/logout',cookie=proxy_cookie,headers={'X-Forwarded-Proto':'https'})[0],204,'proxy logout')
        form_payload = None
        if mode == 'go':
            # A form is bound to the session that rendered it, and duplicate tokens fail.
            status,_,body=c.request('GET','/admin',cookie=admin_cookie)
            expect(status,200,'admin form')
            csrf=re.search(rb'name="csrf_token" value="([^"]+)"',body)[1].decode()
            ticket=re.search(rb'name="submission_token" value="([^"]+)"',body)[1].decode()
            form_payload=urllib.parse.urlencode({'csrf_token':csrf,'submission_token':ticket,
                                                  'display_name':'Denied','slug':'denied'}).encode()
            status,headers,_=c.request('POST',WEB+'/login',data=raw,headers={'Content-Type':'application/json'})
            expect(status,204,'second login')
            second_cookie=headers['Set-Cookie'].split(';',1)[0]
            form_headers={'Content-Type':'application/x-www-form-urlencoded'}
            before_csrf=full_snapshot(server.directory/'tunetrail.db')
            before_csrf_files=staged_files(server.directory)
            expect(c.request('POST','/admin/profile/create',cookie=second_cookie,data=form_payload,headers=form_headers)[0],403,'cross-session CSRF')
            expect(c.request('POST','/admin/profile/create',cookie=admin_cookie,data=form_payload+b'&csrf_token='+csrf.encode(),headers=form_headers)[0],400,'duplicate CSRF')
            assert full_snapshot(server.directory/'tunetrail.db') == before_csrf
            assert staged_files(server.directory) == before_csrf_files
            expect(c.request('POST',WEB+'/logout',cookie=second_cookie)[0],204,'second logout')
            checks += 2
        # Login/logout deliberately mutate sessions; reset the baseline afterwards.
        baseline = full_snapshot(server.directory/'tunetrail.db')
        # Existing user credentials in any query parameter must not enter the debug/access log.
        for route in (WEB+'/profiles/default/search',):
            status,_,_ = c.request('GET',route+'?q=test&token=log-fixture-secret&next=%2F%3Fsecret%3Dnested-fixture-secret')
            expect(status,200,'search')
        if mode == 'go':
            # GET of a POST-only action and HEAD never execute it.
            expect(c.request('GET','/admin/logout',cookie=admin_cookie)[0],404,'GET logout')
            status,headers,body=c.request('HEAD','/admin/profile/1/export',cookie=admin_cookie)
            expect(status,200,'HEAD export'); assert not body and 'no-store' in headers.get('Cache-Control','')
            # An authenticated form without CSRF, with a wrong token, or from another origin cannot mutate.
            for payload,origin in [(b'display_name=Denied&slug=denied',server.origin),
                                   (b'csrf_token=wrong&display_name=Denied&slug=denied',server.origin),
                                   (b'csrf_token=wrong&display_name=Denied&slug=denied','https://foreign.invalid')]:
                status,_,_=c.request('POST','/admin/profile/create',cookie=admin_cookie,data=payload,
                    headers={'Origin':origin,'Content-Type':'application/x-www-form-urlencoded'})
                expect(status,403,'CSRF/origin');checks+=1
        assert full_snapshot(server.directory/'tunetrail.db') == baseline, 'checks changed SQLite'
        assert staged_files(server.directory) == file_baseline, 'checks left files behind'
        with sqlite3.connect(server.directory/'tunetrail.db') as db:
            assert db.execute('PRAGMA integrity_check').fetchall()==[('ok',)]
            assert db.execute('PRAGMA foreign_key_check').fetchall()==[]
        server.log.flush()
        log=(server.directory/'server.log').read_text(errors='replace')
        for secret in (PASSWORD,key,'log-fixture-secret','nested-fixture-secret'):
            assert secret not in log, 'credential or query marker leaked into server logs'
        if mode == 'go':
            # A process restart invalidates previously rendered CSRF tokens and tickets.
            server.stop()
            server.start()
            status,_,_=c.request('POST','/admin/profile/create',cookie=admin_cookie,data=form_payload,
                                  headers={'Content-Type':'application/x-www-form-urlencoded'})
            expect(status,403,'CSRF after restart')
            assert full_snapshot(server.directory/'tunetrail.db') == baseline
            assert staged_files(server.directory) == file_baseline
            checks += 1
        return {'mode': mode, 'negative_checks': checks, 'integrity':'ok', 'log_secrets':'absent'}
    finally:
        server.stop()


def main():
    """Use fresh private temporary directories and always stop both servers."""
    import argparse
    parser=argparse.ArgumentParser()
    parser.add_argument('--binary',type=Path,required=True)
    args=parser.parse_args()
    with tempfile.TemporaryDirectory(prefix='koito-permissions-017-') as work:
        report=[run_mode(args.binary.resolve(),Path(work),mode) for mode in ('go',)]
    print(json.dumps(report,indent=2))


if __name__ == '__main__':
    main()
