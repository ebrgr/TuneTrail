#!/usr/bin/env python3
"""Validate profile isolation against real servers and disposable v3 fixtures."""
import argparse
import contextlib
import json
import queue
import re
import shutil
import sqlite3
import tempfile
import threading
import time
import urllib.parse
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from pathlib import Path

from roadmap05_fixture import BASE, disposable_fixture
from validate_multiuser import LBZ, WEB, Server
from validate_admin_webui import Forms


@contextlib.contextmanager
def relay_receiver():
    """Capture fictional relay requests in one local process."""
    messages=queue.Queue()
    class Handler(BaseHTTPRequestHandler):
        def do_POST(self):
            content=self.rfile.read(int(self.headers.get('Content-Length','0')))
            messages.put((self.headers.get('Authorization'),json.loads(content)))
            self.send_response(200);self.end_headers()
        def log_message(self,*_): pass
    server=ThreadingHTTPServer(('127.0.0.1',0),Handler)
    thread=threading.Thread(target=server.serve_forever,daemon=True);thread.start()
    try: yield f'http://127.0.0.1:{server.server_port}',messages
    finally: server.shutdown();server.server_close();thread.join()


def seed_server(binary,root,mode,large=False):
    """Copy a fixture into a fresh directory before starting the application."""
    directory=root/(mode+('-large' if large else ''))
    directory.mkdir()
    with disposable_fixture(large) as source:
        shutil.copyfile(source,directory/'koito.db')
    return Server(binary,directory,environment={'KOITO_WEB_UI':mode,'TUNETRAIL_FORCE_TZ':'UTC'})


def scrobble(track,stamp,kind='single'):
    """Build a fictional ListenBrainz payload with a conflicting claimed profile."""
    return {'listen_type':kind,'profile_id':2,'payload':[{'listened_at':stamp,'profile_id':2,
      'track_metadata':{'artist_name':'Artista comum','release_name':'Álbum comum',
                        'track_name':track,'additional_info':{'duration':180}}}]}


def checked_json(server,slug,suffix):
    """Read a JSON endpoint using only the explicit profile URL."""
    return server.request('GET',WEB+'/profiles/'+slug+suffix)


def run_fixture(binary,root,mode):
    """Check independent per-profile read oracles before and after mutations."""
    server=seed_server(binary,root,mode)
    try:
        server.start();server.login()
        rankings={
          'alpha':{'artists':{1:3,2:2},'albums':{1:3},'tracks':{1:2,4:1}},
          'beta':{'artists':{1:1,2:2},'albums':{1:1,2:1},'tracks':{1:1,2:1}},
          'empty':{'artists':{},'albums':{},'tracks':{}},
        }
        for slug,count in [('alpha',3),('beta',2),('empty',0)]:
            assert checked_json(server,slug,'/stats?period=all_time')['listen_count']==count
            assert checked_json(server,slug,'/listens?period=all_time')['total_record_count']==count
            assert checked_json(server,slug,'/summary?period=all_time')['plays']==count
            for kind in ('artists','albums','tracks'):
                result=checked_json(server,slug,'/top/'+kind+'?period=all_time')
                assert {x['item']['id']:x['item']['listen_count'] for x in result['items']} == rankings[slug][kind]
            result=checked_json(server,slug,'/listen-activity?year=2024&month=5&tz=UTC')
            assert sum(x['listens'] for x in result['activity'])==count
        # Cookies and a player key for Beta do not override Alpha's URL scope.
        mixed=server.request('GET',WEB+'/profiles/alpha/stats?period=all_time',admin=True,token='fixture-beta-key')
        assert mixed['listen_count']==3
        assert checked_json(server,'alpha','/listens?track_id=2&period=all_time')['total_record_count']==0
        assert checked_json(server,'beta','/listens?track_id=2&period=all_time')['total_record_count']==1
        # IDs 2/3 have no Alpha listens; unplayed catalog is never public search.
        for kind,id_ in [('track',2),('track',3),('album',2),('album',3),('artist',3)]:
            server.request('GET',WEB+f'/profiles/alpha/{kind}/{id_}',expected=404)
        assert not checked_json(server,'alpha','/search?q=S%C3%B3%20Beta')['tracks']
        assert len(checked_json(server,'beta','/search?q=S%C3%B3%20Beta')['tracks'])==1
        assert not checked_json(server,'alpha','/search?q=id%3A2')['tracks']
        assert not checked_json(server,'empty','/search?q=Faixa')['tracks']
        for slug,expected in [('alpha',2),('beta',1)]:
            track=checked_json(server,slug,'/track/1');assert track['listen_count']==expected
            aliases=checked_json(server,slug,'/track/1/aliases');assert aliases
            interest=checked_json(server,slug,'/track/1/interest?buckets=4')
            assert sum(x['listen_count'] for x in interest)==expected
            for suffix in ('/track/1/artists','/album/1/artists','/album/1/aliases','/artist/1/aliases'):
                assert checked_json(server,slug,suffix)
        for path in ['/track/2/artists','/track/2/aliases','/album/2/artists','/album/2/aliases','/artist/3/aliases']:
            server.request('GET',WEB+'/profiles/alpha'+path,expected=404)
        if mode=='go':
            for slug,count in [('alpha',3),('beta',2),('empty',0)]:
                home=server.request('GET',f'/{slug}?period=all_time')
                assert f'data-listen-count="{count}"'.encode() in home
            for path in ['/alpha?period=all_time','/alpha/chart/top/tracks','/alpha/listens','/alpha/track/1','/empty']:
                page=server.request('GET',path);assert b'S\xc3\xb3 Beta' not in page
            assert b'S\xc3\xb3 Beta' in server.request('GET','/beta/listens')
            server.request('GET','/alpha/track/1?track_id=2',expected=400)
            assert b'/alpha/track/2' not in server.request('GET','/alpha/search?q=S%C3%B3%20Beta')
            assert b'/beta/track/2' in server.request('GET','/beta/search?q=S%C3%B3%20Beta')
            management='/alpha/manage/track/1?merge_query=S%C3%B3%20Beta'
            assert b'Review merge into' not in server.request('GET',management)
            assert b'Review merge into S\xc3\xb3 Beta' in server.request('GET',management,admin=True)
        for slug,count in [('alpha',3),('beta',2),('empty',0)]:
            export=server.request('GET',WEB+f'/profiles/{slug}/export',admin=True)
            assert len(export['listens'])==count
        server.request('GET',WEB+'/stats',expected=409)
        # Same profile/track/timestamp is ignored; a claimed profile ID cannot reassign the key.
        server.request('POST',LBZ+'/submit-listens',scrobble('Faixa comum',BASE),token='fixture-alpha-key')
        assert checked_json(server,'alpha','/stats')['listen_count']==3
        assert checked_json(server,'beta','/stats')['listen_count']==2
        server.request('POST',LBZ+'/submit-listens',scrobble('Faixa comum',BASE+10),token='fixture-alpha-key',admin=True)
        assert checked_json(server,'alpha','/stats')['listen_count']==4
        assert checked_json(server,'beta','/stats')['listen_count']==2
        server.request('POST',LBZ+'/submit-listens',scrobble('Faixa comum',0,'playing_now'),token='fixture-beta-key')
        assert checked_json(server,'beta','/now-playing')['currently_playing']
        assert not checked_json(server,'alpha','/now-playing')['currently_playing']
        stale=None
        if mode=='go':
            forms=Forms(server);page=forms.request('GET','/admin?profile=1')[0]
            csrf=re.search(rb'name="csrf_token" value="([^"]+)"',page)[1].decode()
            ticket=re.search(rb'name="submission_token" value="([^"]+)"',page)[1].decode()
            stale=urllib.parse.urlencode({'csrf_token':csrf,'submission_token':ticket,'profile_slug':'alpha','display_name':'Wrong','slug':'alpha'}).encode()
        server.request('PATCH',WEB+'/profiles/alpha',{'display_name':'Alpha Renamed','slug':'alpha-new'},admin=True)
        if stale is not None:
            forms.request('POST','/admin/profile/1/save',stale,expected=409,headers={'Content-Type':'application/x-www-form-urlencoded'})
            assert checked_json(server,'alpha-new','')['display_name']=='Alpha Renamed'
        server.request('GET',WEB+'/profiles/alpha/stats',expected=404)
        assert checked_json(server,'alpha-new','/stats')['listen_count']==4
        assert server.request('GET',LBZ+'/validate-token',token='fixture-alpha-key')['user_name']=='alpha-new'
        with sqlite3.connect(server.directory/'tunetrail.db') as db:
            assert db.execute('PRAGMA integrity_check').fetchall()==[('ok',)]
            assert db.execute('PRAGMA foreign_key_check').fetchall()==[]
            assert db.execute('SELECT profile_id,count(*) FROM listens GROUP BY profile_id ORDER BY profile_id').fetchall()==[(1,4),(2,2)]
        server.stop();server.start()
        assert checked_json(server,'alpha-new','/stats')['listen_count']==4
        assert not checked_json(server,'beta','/now-playing')['currently_playing']
        return {'mode':mode,'profile_counts':{'alpha':4,'beta':2,'empty':0},'integrity':'ok','restart':'ok'}
    finally: server.stop()


def run_large_export(binary,root):
    """Verify the literal export count and uniqueness at a tied page boundary."""
    server=seed_server(binary,root,'go',large=True)
    try:
        server.start();server.login()
        doc=server.request('GET',WEB+'/profiles/alpha/export',admin=True)
        assert len(doc['listens'])==1005
        keys=[(x['listened_at'],x['track']['aliases'][0]['alias']) for x in doc['listens']]
        assert len(set(keys))==1005
        assert len(server.request('GET',WEB+'/profiles/beta/export',admin=True)['listens'])==2
        return {'export_alpha':len(keys),'export_unique':len(set(keys)),'export_beta':2}
    finally:server.stop()


def run_import_relay_and_cache(binary,root):
    """Use separate servers for destructive import, relay and cache scenarios."""
    (root/'separate').mkdir()
    server=seed_server(binary,root/'separate','go')
    try:
        server.start();server.login()
        # Forms opened in two tabs retain their own profile, token and destination.
        forms=Forms(server)
        pages=[forms.request('GET',f'/admin?profile={i}')[0] for i in (1,2)]
        tokens=[]
        for page in pages:
            csrf=re.search(rb'name="csrf_token" value="([^"]+)"',page)[1].decode()
            ticket=re.search(rb'name="submission_token" value="([^"]+)"',page)[1].decode()
            tokens.append((csrf,ticket))
        assert tokens[0][1]!=tokens[1][1]
        for i,slug,name in [(1,'alpha','Alpha Tab'),(2,'beta','Beta Tab')]:
            csrf,ticket=tokens[i-1]
            body=urllib.parse.urlencode({'csrf_token':csrf,'submission_token':ticket,'profile_slug':slug,'display_name':name,'slug':slug}).encode()
            forms.request('POST',f'/admin/profile/{i}/save',body,expected=303,headers={'Content-Type':'application/x-www-form-urlencoded'})
        assert checked_json(server,'alpha','')['display_name']=='Alpha Tab'
        assert checked_json(server,'beta','')['display_name']=='Beta Tab'
        exported=server.request('GET',WEB+'/profiles/alpha/export',admin=True)
        server.upload(WEB+'/profiles/empty/import','koito',json.dumps(exported).encode())
        assert checked_json(server,'empty','/stats')['listen_count']==3
        assert checked_json(server,'alpha','/stats')['listen_count']==3
        assert checked_json(server,'beta','/stats')['listen_count']==2
        with relay_receiver() as (url_a,msg_a), relay_receiver() as (url_b,msg_b):
            server.request('PUT',WEB+'/profiles/alpha/relay',{'url':url_a,'token':'fictional-relay-alpha'},admin=True,expected=204)
            server.request('PUT',WEB+'/profiles/beta/relay',{'url':url_b,'token':'fictional-relay-beta'},admin=True,expected=204)
            server.request('POST',LBZ+'/submit-listens',scrobble('Faixa comum',BASE+20),token='fixture-alpha-key')
            server.request('POST',LBZ+'/submit-listens',scrobble('Faixa comum',BASE+21),token='fixture-beta-key')
            header_a,payload_a=msg_a.get(timeout=5);header_b,payload_b=msg_b.get(timeout=5)
            assert header_a=='Token fictional-relay-alpha' and header_b=='Token fictional-relay-beta'
            assert payload_a['payload'][0]['listened_at']==BASE+20
            assert payload_b['payload'][0]['listened_at']==BASE+21
            assert msg_a.empty() and msg_b.empty()
            server.request('DELETE',WEB+'/profiles/alpha/relay',admin=True,expected=204)
            server.request('DELETE',WEB+'/profiles/beta/relay',admin=True,expected=204)
        # A deleted global track is lazily removed from that profile's cache only.
        server.request('POST',LBZ+'/submit-listens',scrobble('Faixa comum',0,'playing_now'),token='fixture-beta-key')
        server.request('POST',LBZ+'/submit-listens',scrobble('Outra Alpha',0,'playing_now'),token='fixture-alpha-key')
        assert checked_json(server,'alpha','/now-playing')['track']['id']==4
        assert checked_json(server,'beta','/now-playing')['track']['id']==1
        server.request('DELETE',WEB+'/track/4',admin=True,expected=204)
        assert not checked_json(server,'alpha','/now-playing')['currently_playing']
        assert checked_json(server,'beta','/now-playing')['track']['id']==1
        # Revocation affects the player's own profile without invalidating the other key.
        server.request('DELETE',WEB+'/profiles/alpha/user/apikeys/1',admin=True,expected=204)
        server.request('GET',LBZ+'/validate-token',token='fixture-alpha-key',expected=401)
        assert server.request('GET',LBZ+'/validate-token',token='fixture-beta-key')['valid']
        with sqlite3.connect(server.directory/'tunetrail.db') as db:
            assert db.execute('PRAGMA integrity_check').fetchall()==[('ok',)]
            assert db.execute('PRAGMA foreign_key_check').fetchall()==[]
        return {'two_tabs':'ok','import_to_empty':3,'isolated_relays':2,'cache_delete':'ok','key_revocation':'ok'}
    finally: server.stop()


def run_legacy_and_merge(binary,root):
    """Exercise profile lifecycle, legacy routing and cache after merge."""
    (root/'legacy').mkdir()
    server=seed_server(binary,root/'legacy','go')
    try:
        server.start();server.login()
        # Merge removes one cached source while another profile keeps its destination.
        server.request('POST',LBZ+'/submit-listens',scrobble('Outra Alpha',0,'playing_now'),token='fixture-alpha-key')
        server.request('POST',LBZ+'/submit-listens',scrobble('Faixa comum',0,'playing_now'),token='fixture-beta-key')
        server.request('POST',WEB+'/track/1/merge',{'merge_from_id':4},admin=True,expected=204)
        assert not checked_json(server,'alpha','/now-playing')['currently_playing']
        assert checked_json(server,'beta','/now-playing')['track']['id']==1
        server.request('POST',LBZ+'/submit-listens',scrobble('Faixa comum',0,'playing_now'),token='fixture-alpha-key')
        forms=Forms(server);page=forms.request('GET','/admin?profile=2')[0]
        csrf=re.search(rb'name="csrf_token" value="([^"]+)"',page)[1].decode()
        ticket=re.search(rb'name="submission_token" value="([^"]+)"',page)[1].decode()
        stale=urllib.parse.urlencode({'csrf_token':csrf,'submission_token':ticket,'profile_slug':'beta','display_name':'Wrong','slug':'beta'}).encode()
        profiles=server.request('GET',WEB+'/profiles')
        server.request('GET',WEB+'/stats',expected=409)
        for profile in profiles:
            slug=profile['slug']
            if slug=='alpha':continue
            count=checked_json(server,slug,'/stats')['listen_count']
            server.request('DELETE',WEB+'/profiles/'+slug,{'display_name':profile['display_name'],'listen_count':count},admin=True,expected=204)
        server.request('GET',WEB+'/profiles/beta/stats',expected=404)
        forms.request('POST','/admin/profile/2/save',stale,expected=404,headers={'Content-Type':'application/x-www-form-urlencoded'})
        server.request('GET',LBZ+'/validate-token',token='fixture-beta-key',expected=401)
        assert checked_json(server,'alpha','/now-playing')['currently_playing']
        assert server.request('GET',WEB+'/stats')['listen_count']==3
        server.request('POST',WEB+'/profiles',{'display_name':'Reserved','slug':'admin'},admin=True,expected=400)
        server.request('DELETE',WEB+'/profiles/alpha',{'display_name':'Alpha','listen_count':3},admin=True,expected=204)
        server.request('GET',WEB+'/stats',expected=409)
        assert server.request('GET',WEB+'/profiles')==[]
        with sqlite3.connect(server.directory/'tunetrail.db') as db:
            assert db.execute('PRAGMA integrity_check').fetchall()==[('ok',)]
            assert db.execute('PRAGMA foreign_key_check').fetchall()==[]
        return {'legacy_profile_counts':'many/one/zero','merge_cache':'ok','profile_deletion':'ok','reserved_slug':'rejected'}
    finally: server.stop()


def main():
    """Run both modes in fresh directories and remove all test data."""
    parser=argparse.ArgumentParser();parser.add_argument('--binary',type=Path,required=True);args=parser.parse_args()
    with tempfile.TemporaryDirectory(prefix='koito-isolation-019-') as work:
        root=Path(work);report=[run_fixture(args.binary.resolve(),root,mode) for mode in ('go',)]
        report.append(run_large_export(args.binary.resolve(),root))
        report.append(run_import_relay_and_cache(args.binary.resolve(),root))
        report.append(run_legacy_and_merge(args.binary.resolve(),root))
    print(json.dumps(report,indent=2))


if __name__=='__main__':main()
