#!/usr/bin/env python3
"""Exercise relay shutdown with an intentionally stalled loopback receiver."""
import argparse
import json
import signal
import sqlite3
import tempfile
import threading
import time
from http.server import BaseHTTPRequestHandler,ThreadingHTTPServer
from pathlib import Path

from validate_admin_webui import approve_saved_relay
from validate_multiuser import Server,WEB,LBZ,scrobble


def main():
    """Confirm a durable local listen and bounded process shutdown during relay I/O."""
    parser=argparse.ArgumentParser();parser.add_argument('--binary',type=Path,required=True);args=parser.parse_args()
    entered=threading.Event();release=threading.Event();finished=threading.Event()
    received=[]
    class Receiver(BaseHTTPRequestHandler):
        def do_POST(self):
            body=json.loads(self.rfile.read(int(self.headers.get('Content-Length','0'))))
            received.extend((item['listened_at'],item['track_metadata']['track_name'],'fixture-user')
                            for item in body['payload'])
            entered.set();release.wait(15)
            try:self.send_response(200);self.end_headers()
            except OSError:pass
            finally:finished.set()
        def log_message(self,*_):pass
    relay=ThreadingHTTPServer(('127.0.0.1',0),Receiver);thread=threading.Thread(target=relay.serve_forever,daemon=True);thread.start()
    try:
        with tempfile.TemporaryDirectory(prefix='koito-shutdown-020-') as root:
            server=Server(args.binary.resolve(),Path(root)/'data')
            try:
                server.start();server.login();key=server.request('GET',WEB+'/profiles/default/user/apikeys',admin=True)[0]['key']
                path=WEB+'/profiles/default'
                host=f'127.0.0.1:{relay.server_port}'
                server.request('PUT',path+'/relay/permissions',{'host_port':host,'allow_http':True},admin=True,expected=204)
                server.request('PUT',path+'/relay/permissions',{'host_port':host,'cidr':'127.0.0.1/32','allow_private':True},admin=True,expected=204)
                server.request('PUT',path+'/relay',{'url':f'http://{host}','token':'fictional-relay-token'},admin=True,expected=204)
                approve_saved_relay(server, 1)
                stamp=int(time.time())-60;server.request('POST',LBZ+'/submit-listens',scrobble(stamp),token=key)
                assert entered.wait(5),'relay request did not reach local receiver'
                start=time.monotonic();server.process.send_signal(signal.SIGTERM)
                returncode=server.process.wait(timeout=8.5);duration=time.monotonic()-start
                assert returncode==0,'server did not exit cleanly'
                with sqlite3.connect(server.directory/'tunetrail.db') as db:
                    assert db.execute('SELECT count(*) FROM listens').fetchone()[0]==1
                    assert db.execute("SELECT state,attempt_count,lease_owner FROM outbox").fetchall()==[('retryable',0,None)]
                    assert db.execute("SELECT pause_class FROM relay_destinations").fetchall()==[(None,)]
                    assert db.execute('PRAGMA integrity_check').fetchall()==[('ok',)]
                    assert db.execute('PRAGMA foreign_key_check').fetchall()==[]
                server.stop()
                release.set()
                server.start()
                deadline=time.monotonic()+8
                while True:
                    with sqlite3.connect(server.directory/'tunetrail.db') as db:
                        delivered=db.execute("SELECT count(*) FROM outbox WHERE state='sent'").fetchone()[0]
                    if delivered:break
                    assert time.monotonic()<deadline, 'persisted delivery was not retried'
                    time.sleep(.1)
                assert len(received)==2 and len(set(received))==1
                print(json.dumps({'posts':2,'duplicate_attempts':1,'unique_mock_keys':1,'local_listens':1,'relay':'resumed after SIGTERM','shutdown_seconds':round(duration,3),'integrity':'ok'}))
            finally:server.stop()
    finally:
        release.set();finished.wait(2);relay.shutdown();relay.server_close();thread.join()


if __name__=='__main__':main()
