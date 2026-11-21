#!/usr/bin/env python3
"""Replay legacy ListenBrainz requests against two builds on loopback."""
import argparse
import queue
import tempfile
import threading
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from pathlib import Path

from validate_multiuser import LBZ, WEB, Server


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('--baseline', type=Path, required=True)
    parser.add_argument('--binary', type=Path, required=True)
    args = parser.parse_args()
    calls = queue.Queue()

    class Receiver(BaseHTTPRequestHandler):
        def do_POST(self):
            body = self.rfile.read(int(self.headers['Content-Length']))
            calls.put((self.path, self.command, dict(self.headers), body))
            self.send_response(200)
            self.end_headers()

        def log_message(self, *_):
            pass

    receiver = ThreadingHTTPServer(('127.0.0.1', 0), Receiver)
    runner = threading.Thread(target=receiver.serve_forever)
    runner.start()
    try:
        url = f'http://127.0.0.1:{receiver.server_port}/fixture'
        profiles = WEB+'/profiles/default'
        bodies = [
            b'{ "listen_type":"single" , "payload":[{"listened_at":1700000100,"track_metadata":{"artist_name":"A","track_name":"B"}}] }\n',
            b'{"listen_type":"import","payload":[{"listened_at":1700000101,"track_metadata":{"artist_name":"A","track_name":"C"}},{"listened_at":1700000102,"track_metadata":{"artist_name":"A","track_name":"D"}}]}',
            b'{ "listen_type":"playing_now", "payload":[{"track_metadata":{"artist_name":"A","track_name":"E"}}] }',
        ]
        traces = []
        with tempfile.TemporaryDirectory(prefix='tunetrail-relay-parity-') as temp:
            for name, binary in [('baseline', args.baseline), ('candidate', args.binary)]:
                app = Server(binary.resolve(), Path(temp)/name)
                app.start()
                try:
                    app.login()
                    key = app.request('GET', profiles+'/user/apikeys', admin=True)[0]['key']
                    app.request('PUT', profiles+'/relay', {'url':url,'token':'fixture-outbound'}, admin=True, expected=204)
                    trace = []
                    for body in bodies:
                        ack = app.request('POST', LBZ+'/submit-listens', body, token=key,
                                          headers={'Content-Type':'application/json'})
                        request = calls.get(timeout=5)
                        assert request[0] == '/fixture/submit-listens' and request[-1] == body
                        trace.append((ack, request))
                    errors = [app.request('POST', LBZ+'/submit-listens', b'{}', token=key, expected=400),
                              app.request('POST', LBZ+'/submit-listens', bodies[0], token='invalid', expected=401)]
                    traces.append((trace, errors))
                    assert calls.empty()
                finally:
                    app.stop()
            old_build = Server(args.baseline.resolve(), Path(temp)/'candidate')
            try:
                old_build.start()
            except RuntimeError:
                assert 'unsupported schema version 12' in (Path(temp)/'candidate'/'server.log').read_text()
            else:
                raise AssertionError('v0.48 opened a v12 database')
            finally:
                old_build.stop()
        assert traces[0] == traces[1], 'legacy wire replay differs'
        print('legacy relay replay: single, import, playing_now and 2 errors match')
    finally:
        receiver.shutdown()
        receiver.server_close()
        runner.join()


if __name__ == '__main__':
    main()
