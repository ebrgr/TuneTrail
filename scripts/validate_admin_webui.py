#!/usr/bin/env python3
"""Exercise integrated HTML actions on disposable SQLite v3; never reads .env.
Build cmd/api first, then pass --binary and a new --work-dir. No real services.
"""
import argparse
import contextlib
import io
import json
import re
import sqlite3
import threading
import urllib.error
import urllib.parse
import urllib.request
import uuid
import zipfile
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from pathlib import Path
from validate_multiuser import Server, WEB, LBZ, PASSWORD

class NoRedirect(urllib.request.HTTPRedirectHandler):
    def redirect_request(self, *_):
        return None

def csrf(body):
    match = re.search(rb'name="csrf_token" value="([^"]+)"', body)
    assert match, 'Missing form protection'
    return match[1].decode()

def multipart(fields, data, field='file', name='fictional.json'):
    boundary = uuid.uuid4().hex
    body = b''
    for key, value in fields.items():
        body += f'--{boundary}\r\nContent-Disposition: form-data; name="{key}"\r\n\r\n{value}\r\n'.encode()
    body += f'--{boundary}\r\nContent-Disposition: form-data; name="{field}"; filename="{name}"\r\nContent-Type: application/octet-stream\r\n\r\n'.encode() + data + f'\r\n--{boundary}--\r\n'.encode()
    return body, 'multipart/form-data; boundary=' + boundary

class Forms:
    def __init__(self, server):
        self.server = server
        self.opener = urllib.request.build_opener(urllib.request.HTTPCookieProcessor(server.cookies), NoRedirect())
    def request(self, method, path, values=None, expected=200, headers=None):
        if isinstance(values,dict) and 'csrf_token' in values and method=='POST' and path!='/admin/login' and not path.endswith('/confirm'):
            values=dict(values)
            values.setdefault('submission_token',self.ticket())
        body = urllib.parse.urlencode(values).encode() if isinstance(values, dict) else values
        request_headers = {'Origin': self.server.origin}
        if isinstance(values, dict): request_headers['Content-Type'] = 'application/x-www-form-urlencoded'
        request_headers.update(headers or {})
        req = urllib.request.Request(self.server.origin + path, body, request_headers, method=method)
        try: response = self.opener.open(req, timeout=60)
        except urllib.error.HTTPError as error: response = error
        with response:
            data = response.read()
            assert response.status == expected, f'{method} {path}: expected {expected}, got {response.status}'
            assert 'no-store' in response.headers.get('Cache-Control', ''), f'Cache policy: {path}'
            return data, response.headers
    def ticket(self):
        body=self.request('GET','/admin')[0]
        match=re.search(rb'name="submission_token" value="([^"]+)"',body)
        return match[1].decode() if match else ''
    def token(self, path='/admin'):
        return csrf(self.request('GET', path)[0])
    def login(self, next_path='/admin', password=PASSWORD):
        token = self.token('/admin?next=' + urllib.parse.quote(next_path, safe=''))
        _, headers = self.request('POST', '/admin/login', {'csrf_token': token, 'username':'validation-admin', 'password':password, 'next':next_path}, expected=303)
        assert headers['Location'] == next_path

@contextlib.contextmanager
def relay_server():
    class Handler(BaseHTTPRequestHandler):
        def do_GET(self):
            assert self.path == '/validate-token'
            valid = self.headers.get('Authorization') == 'Token fictional-token'
            self.send_response(200); self.send_header('Content-Type','application/json'); self.end_headers()
            self.wfile.write(json.dumps({'valid':valid,'user_name':'fictional-listener'}).encode())
        def log_message(self, *_): pass
    server = ThreadingHTTPServer(('127.0.0.1',0),Handler)
    thread = threading.Thread(target=server.serve_forever,daemon=True); thread.start()
    try: yield f'http://127.0.0.1:{server.server_port}'
    finally: server.shutdown();server.server_close();thread.join()

def validate(server):
    f = Forms(server)
    token = f.token()
    f.request('POST','/admin/login',{'username':'validation-admin','password':PASSWORD},expected=403)
    f.login('/admin?profile=1')
    token=f.token('/admin?profile=1')
    # Documents are distinct from actions, including HEAD and expired sessions.
    for path in ['/admin','/admin?profile=1','/admin/profile/1/export','/admin/profile/1/delete-review']:
        body, _=f.request('HEAD',path);assert not body
    f.request('POST','/admin/profile/1/save',{'csrf_token':token,'profile_slug':'default','display_name':'Cross origin','slug':'default'},expected=403,headers={'Origin':'https://elsewhere.invalid'})
    f.request('GET','/admin/logout',expected=404)
    # A production form ticket is required and can be used only once.
    token=f.token();raw={'csrf_token':token,'display_name':'Replay Fictional','slug':'replay-fictional'}
    encoded=urllib.parse.urlencode(raw).encode()
    f.request('POST','/admin/profile/create',encoded,expected=409,headers={'Content-Type':'application/x-www-form-urlencoded'})
    raw['submission_token']=f.ticket();encoded=urllib.parse.urlencode(raw).encode()
    f.request('POST','/admin/profile/create',encoded,expected=303,headers={'Content-Type':'application/x-www-form-urlencoded'})
    f.request('POST','/admin/profile/create',encoded,expected=409,headers={'Content-Type':'application/x-www-form-urlencoded'})
    assert sum(p['slug']=='replay-fictional' for p in server.request('GET',WEB+'/profiles'))==1
    # Create and rename profiles without changing the administrative account.
    for name,slug in [('Mara Silva','mara'),('Luna Paz','luna')]:
        f.request('POST','/admin/profile/create',{'csrf_token':token,'display_name':name,'slug':slug},expected=303)
    profiles=server.request('GET',WEB+'/profiles');by_slug={p['slug']:p for p in profiles}
    p=by_slug['mara'];other=by_slug['luna'];prefix=f'/admin/profile/{p["id"]}/';common={'csrf_token':token,'profile_slug':'mara'}
    f.request('POST',prefix+'save',dict(common,display_name='Mara Fictional',slug='mara'),expected=303)
    # Keys remain personal; repeated or cross-profile revocation is rejected.
    f.request('POST',prefix+'keys/create',dict(common,label='Fictional player'),expected=303)
    keys=server.request('GET',WEB+'/profiles/mara/user/apikeys',admin=True);key=keys[-1]
    f.request('POST',prefix+f'keys/{key["id"]}/label',dict(common,label='Renamed fictional player'),expected=303)
    f.request('POST',f'/admin/profile/{other["id"]}/keys/{key["id"]}/revoke',{'csrf_token':token,'profile_slug':'luna','confirm':'1'},expected=404)
    f.request('POST',prefix+f'keys/{key["id"]}/revoke',common,expected=400)
    f.request('POST',prefix+f'keys/{key["id"]}/revoke',dict(common,confirm='1'),expected=303)
    server.request('GET',LBZ+'/validate-token',token=key['key'],expected=401)
    with relay_server() as origin:
        draft=dict(common,url=origin,token='fictional-token')
        f.request('POST',prefix+'relay/test',draft,expected=303)
        assert not server.request('GET',WEB+'/profiles/mara/relay',admin=True)['enabled']
        f.request('POST',prefix+'relay/save',draft,expected=303)
        f.request('POST',prefix+'relay/remove',common,expected=303)
    # Five fictional source formats; file identity cannot change the target.
    fixtures={
        'spotify':b'[{"ts":"2024-01-01T12:00:00Z","master_metadata_track_name":"S Track","master_metadata_album_artist_name":"S Artist","master_metadata_album_album_name":"S Album","reason_end":"trackdone"}]',
        'maloja':b'{"scrobbles":[{"time":1704110400,"track":{"artists":["M Artist"],"title":"M Track","album":{"albumtitle":"M Album"}}}]}',
        'lastfm':b'[{"track":[{"artist":{"#text":"F Artist"},"album":{"#text":"F Album"},"name":"F Track","date":{"uts":"1704110400"}}]}]',
        'koito':b'{"version":"1","user":"unrelated-name","listens":[{"listened_at":"2024-01-01T12:00:00Z","track":{"aliases":[{"alias":"K Track","is_primary":true}]},"album":{"aliases":[{"alias":"K Album","is_primary":true}]},"artists":[{"aliases":[{"alias":"K Artist","is_primary":true}]}]}]}',
    }
    zipped=io.BytesIO()
    with zipfile.ZipFile(zipped,'w') as z:z.writestr('listens/data.jsonl',json.dumps({'listened_at':1704110400,'track_metadata':{'artist_name':'L Artist','track_name':'L Track','release_name':'L Album'}})+'\n')
    fixtures['listenbrainz']=zipped.getvalue()
    for format,data in fixtures.items():
        fields=dict(common,submission_token=f.ticket(),profile_name='Mara Fictional',confirm='1',format=format)
        body,ctype=multipart(fields,data)
        f.request('POST',prefix+'import',body,expected=303,headers={'Content-Type':ctype})
    data,headers=f.request('GET',prefix+'export');doc=json.loads(data);assert len(doc['listens'])==5;assert 'attachment' in headers['Content-Disposition']
    body,ctype=multipart({'csrf_token':token,'submission_token':f.ticket(),'profile_slug':'luna','profile_name':'Luna Paz','confirm':'1','format':'koito'},data)
    f.request('POST',f'/admin/profile/{other["id"]}/import',body,expected=303,headers={'Content-Type':ctype})
    assert server.request('GET',WEB+'/profiles/luna/stats')['listen_count']==5
    tracks=server.request('GET',WEB+'/profiles/mara/top/tracks?period=all_time')['items'];track=tracks[0]['item'];tid=track['id']
    manage=f'/mara/manage/track/{tid}'
    mt=f.token(manage);mc={'csrf_token':mt,'profile_id':str(p['id']),'profile_slug':'mara'}
    f.request('HEAD',manage)
    f.request('HEAD',manage+'/review?action=delete')
    f.request('POST',manage+'/alias-add',dict(mc,alias='Fictional Unicode 音'),expected=303)
    f.request('POST',manage+'/alias-primary',dict(mc,alias='Fictional Unicode 音'),expected=303)
    body,_=f.request('POST',manage+'/metadata',dict(mc,mbid='bad-uuid'),expected=400);assert b'value="bad-uuid"' in body
    body,_=f.request('POST',manage+'/listen-add',dict(mc,local_time='2024-11-03T01:30',timezone='America/New_York'),expected=409);assert b'occurs twice' in body
    f.request('POST',manage+'/listen-add',dict(mc,local_time='2024-11-03T01:30',timezone='America/New_York',offset='-04:00'),expected=303)
    back='/mara/listens?year=2024&week=44&page=2&limit=1&track_id='+str(tid)
    selected=manage+'?listen_at=1730611800&history_return='+urllib.parse.quote(back,safe='')
    focused,_=f.request('GET',selected);assert b'id="selected-listen"' in focused
    _,returned=f.request('POST',manage+'/listen-delete',dict(mc,unix='1730611800',confirm='1',return_to=back),expected=303)
    params=urllib.parse.parse_qs(urllib.parse.urlsplit(returned['Location']).query);assert params['week']==['44'] and params['page']==['1'] and params['track_id']==[str(tid)]
    # Source/destination review, stale plan and final confirmation.
    searched,_=f.request('GET',manage+'?merge_query=Track');assert b'Review merge into' in searched
    dest=tracks[1]['item']['id'];review=manage+f'/review?action=merge&to={dest}'
    body,_=f.request('GET',review);assert b'Destination and surviving ID' in body
    plan=re.search(rb'name="plan" value="([^"]+)"',body)[1].decode();confirm=dict(mc,csrf_token=csrf(body),plan=plan,confirm='1')
    f.request('POST',manage+'/alias-add',dict(mc,alias='Change after review'),expected=303)
    f.request('POST',manage+'/confirm',confirm,expected=409)
    body,_=f.request('GET',review);confirm.update(plan=re.search(rb'name="plan" value="([^"]+)"',body)[1].decode(),csrf_token=csrf(body))
    f.request('POST',manage+'/confirm',confirm,expected=303)
    f.request('POST',manage+'/confirm',confirm,expected=409)
    # All profiles may be deleted while the administrator and its session survive.
    for profile in server.request('GET',WEB+'/profiles'):
        count=server.request('GET',WEB+f'/profiles/{profile["slug"]}/stats')['listen_count']
        path=f'/admin/profile/{profile["id"]}/';pt=f.token(path+'delete-review')
        f.request('POST',path+'delete',{'csrf_token':pt,'profile_slug':profile['slug'],'display_name':profile['display_name'],'listen_count':str(count),'confirm':'1'},expected=303)
    assert not server.request('GET',WEB+'/profiles');body,_=f.request('GET','/admin');assert b'No profiles yet' in body
    token=csrf(body);f.request('POST','/admin/account/password',{'csrf_token':token,'password':'replacement-fictional-password','confirm_password':'replacement-fictional-password'},expected=303)
    f.request('POST','/admin/logout',{'csrf_token':token},expected=401)
    f.login(password='replacement-fictional-password')
    with sqlite3.connect(server.directory/'tunetrail.db') as conn:
        assert conn.execute('PRAGMA integrity_check').fetchone()[0]=='ok';assert not conn.execute('PRAGMA foreign_key_check').fetchall()
    return {'schema':3,'formats':list(fixtures),'checks':['HTML login/return','HEAD','CSRF/origin','profiles','keys/revocation','local relay','five import formats','export/round-trip','metadata/form state','DST','personal deletion','stale merge/repetition','zero profiles','password revocation','SQLite integrity']}

def rate_limit(binary,root,html_first):
    server=Server(binary,root,environment={'KOITO_WEB_UI':'go','KOITO_DISABLE_RATE_LIMIT':'false'})
    try:
        server.start();f=Forms(server);token=f.token()
        order=['html']*6+['json']*4 if html_first else ['json']*4+['html']*6
        for mode in order:
            if mode=='html':f.request('POST','/admin/login',{'csrf_token':token,'username':'validation-admin','password':'wrong-password'},expected=401)
            else:server.request('POST',WEB+'/login',{'username':'validation-admin','password':'wrong-password'},expected=401)
        f.request('POST','/admin/login',{'csrf_token':token,'username':'validation-admin','password':'wrong-password'},expected=429)
        req=urllib.request.Request(server.origin+WEB+'/login',json.dumps({'username':'validation-admin','password':'wrong-password'}).encode(),{'Content-Type':'application/json'},method='POST')
        try:response=urllib.request.urlopen(req)
        except urllib.error.HTTPError as e:response=e
        with response:assert response.status==429;assert 'no-store' in response.headers.get('Cache-Control','')
    finally:server.stop()

def main():
    parser=argparse.ArgumentParser(description=__doc__);parser.add_argument('--binary',type=Path,required=True);parser.add_argument('--work-dir',type=Path,required=True);args=parser.parse_args();root=args.work_dir.resolve();root.mkdir(parents=True,exist_ok=False);root.chmod(0o700)
    server=Server(args.binary.resolve(),root/'go',environment={'KOITO_WEB_UI':'go'})
    try:server.start();report=validate(server)
    finally:server.stop()
    for first in [False,True]:rate_limit(args.binary.resolve(),root/('rate-html-first' if first else 'rate-json-first'),first)
    report['checks'].append('shared HTML/JSON login budget both orders');(root/'report.json').write_text(json.dumps(report,indent=2)+'\n');print(json.dumps(report))
if __name__=='__main__':main()
