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

from validate_multiuser import Server,WEB,LBZ,scrobble


def main():
    """Confirm a durable local listen and bounded process shutdown during relay I/O."""
    parser=argparse.ArgumentParser();parser.add_argument('--binary',type=Path,required=True);args=parser.parse_args()
    entered=threading.Event();release=threading.Event();finished=threading.Event()
    class Receiver(BaseHTTPRequestHandler):
        def do_POST(self):
            self.rfile.read(int(self.headers.get('Content-Length','0')))
            entered.set();release.wait(15)
            try:self.send_response(200);self.end_headers()
            except OSError:pass
            finally:finished.set()
        def log_message(self,*_):pass
    relay=ThreadingHTTPServer(('127.0.0.1',0),Receiver);thread=threading.Thread(target=relay.serve_forever,daemon=True);thread.start()
    try:
        with tempfile.TemporaryDirectory(prefix='koito-shutdown-020-') as root:
            server=Server(args.binary.resolve(),Path(root)/'data',environment={'KOITO_WEB_UI':'go'})
            try:
                server.start();server.login();key=server.request('GET',WEB+'/profiles/default/user/apikeys',admin=True)[0]['key']
                server.request('PUT',WEB+'/profiles/default/relay',{'url':f'http://127.0.0.1:{relay.server_port}','token':'fictional-relay-token'},admin=True,expected=204)
                stamp=int(time.time())-60;server.request('POST',LBZ+'/submit-listens',scrobble(stamp),token=key)
                assert entered.wait(5),'relay request did not reach local receiver'
                start=time.monotonic();server.process.send_signal(signal.SIGINT)
                returncode=server.process.wait(timeout=11);duration=time.monotonic()-start
                assert returncode==0,'server did not exit cleanly'
                with sqlite3.connect(server.directory/'tunetrail.db') as db:
                    assert db.execute('SELECT count(*) FROM listens').fetchone()[0]==1
                    assert db.execute('PRAGMA integrity_check').fetchall()==[('ok',)]
                    assert db.execute('PRAGMA foreign_key_check').fetchall()==[]
                print(json.dumps({'local_listens':1,'relay':'in-flight at shutdown; delivery not guaranteed','shutdown_seconds':round(duration,3),'integrity':'ok'}))
            finally:server.stop()
    finally:
        release.set();finished.wait(2);relay.shutdown();relay.server_close();thread.join()


if __name__=='__main__':main()
