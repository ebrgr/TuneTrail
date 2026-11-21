#!/usr/bin/env python3
"""Measure durable relay delivery using disposable profiles and loopback receivers."""
import argparse
from collections import Counter
from concurrent.futures import ThreadPoolExecutor
from contextlib import closing
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
import json
from pathlib import Path
import socket
import sqlite3
import subprocess
import tempfile
import threading
import time

from rehearse_issue081 import percentiles, sample_resources
from validate_multiuser import LBZ, WEB, Server, scrobble


def exercise(binary, directory, scenario):
    posts, accepted, pauses = [], [], set()
    lock = threading.Lock()
    ready = threading.Event()
    ready.set()

    class Receiver(BaseHTTPRequestHandler):
        def do_POST(self):
            payload = json.loads(self.rfile.read(int(self.headers['Content-Length'])))['payload']
            with lock:
                posts.append(len(payload))
                attempt = len(posts)
            ready.wait(30)
            status = 200
            if scenario == 'slow':
                time.sleep(7.7)
            if scenario == 'oscillating' and attempt in (1, 2):
                status = (503, 429)[attempt - 1]
            if any(item['track_metadata']['track_name'] == 'Rejected fixture' for item in payload):
                status = 400
            if status == 200:
                with lock:
                    accepted.extend((item['listened_at'], item['track_metadata']['track_name'], 'fictional') for item in payload)
            self.send_response(status)
            if status == 429:
                self.send_header('Retry-After', '2')
            if scenario == 'healthy' and attempt == 1:
                self.send_header('X-RateLimit-Remaining', '0')
                self.send_header('X-RateLimit-Reset-In', '2')
            self.end_headers()

        def log_message(self, *_):
            pass

    with ThreadingHTTPServer(('127.0.0.1', 0), Receiver) as receiver, socket.socket() as refused:
        refused.bind(('127.0.0.1', 0))
        thread = threading.Thread(target=receiver.serve_forever)
        thread.start()
        app = Server(binary, directory, environment={'TUNETRAIL_INGEST_ACTIVE': 'true'})
        try:
            app.start()
            app.login()
            profile = WEB + '/profiles/default'
            key = app.request('GET', profile + '/user/apikeys', admin=True)[0]['key']
            preview = app.request('GET', profile + '/ingest-activation', admin=True)
            app.request('PUT', profile + '/ingest-activation',
                        {'active': True, 'review': False, 'digest': preview['digest']}, admin=True, expected=204)
            port = receiver.server_port if scenario != 'refused' else refused.getsockname()[1]
            host = f'127.0.0.1:{port}'
            for permission in ({'host_port': host, 'allow_http': True},
                               {'host_port': host, 'cidr': '127.0.0.1/32', 'allow_private': True}):
                app.request('PUT', profile + '/relay/permissions', permission, admin=True, expected=204)
            app.request('PUT', profile + '/relay', {'url': f'http://{host}/fixture', 'token': 'e4-fictional'},
                        admin=True, expected=204)
            start_stamp = int(time.time()) - 10000
            files, size = (4, 1000) if scenario == 'healthy' else (2, 60)

            def submit(index, count, reject=False):
                items = [scrobble(start_stamp + index + n)['payload'][0] for n in range(count)]
                if reject:
                    items[-1]['track_metadata']['track_name'] = 'Rejected fixture'
                start = time.monotonic()
                app.request('POST', LBZ + '/submit-listens', {'listen_type': 'import', 'payload': items}, token=key)
                return time.monotonic() - start

            def states(db):
                return dict(db.execute('SELECT state,count(*) FROM outbox GROUP BY state'))

            def settle(db, count, timeout):
                deadline = time.monotonic() + timeout
                while time.monotonic() < deadline:
                    row = db.execute('SELECT pause_class FROM relay_destinations').fetchone()
                    if row and row[0]:
                        pauses.add(row[0])
                    state = states(db)
                    if state.get('sent', 0) + state.get('failed', 0) == count:
                        return state
                    if scenario == 'refused' and row and row[0] and sum(state.values()) == count:
                        return state
                    time.sleep(.05)
                raise AssertionError(f'{scenario} did not settle: {states(db)}')

            with closing(sqlite3.connect(directory / 'tunetrail.db')) as db, sample_resources(app) as peaks:
                ready.clear()
                start = time.monotonic()
                with ThreadPoolExecutor(max_workers=files) as pool:
                    jobs = [pool.submit(submit, n * size, size, scenario == 'healthy' and n == 0)
                            for n in range(files)]
                    imports = [job.result() for job in jobs]
                ready.set()
                acks = [submit(files * size + i, 1) for i in range(40)]
                final = settle(db, files * size + 40, 115)
                duration = time.monotonic() - start
                if scenario != 'refused':
                    assert final == ({'sent': 4039, 'failed': 1} if scenario == 'healthy' else {'sent': 160}), final
                    assert len(set(accepted)) == final['sent']
                    assert posts and max(posts) <= 100
                assert db.execute('SELECT count(*) FROM listens').fetchone()[0] == files * size + 40
                before = cpu_time(app)
                time.sleep(3)
                idle_cpu = cpu_time(app) - before
            return dict(scenario=scenario, seconds=round(duration, 3), posts=len(posts),
                        batch_sizes=dict(sorted(Counter(posts).items())), states=final, pauses=sorted(pauses),
                        ack=percentiles(acks), imports=percentiles(imports), idle_cpu_seconds=round(idle_cpu, 3),
                        duplicates=len(accepted) - len(set(accepted)),
                        inversions=sum(a > b for a, b in zip(accepted, accepted[1:])), **peaks)
        finally:
            ready.set()
            app.stop()
            receiver.shutdown()
            thread.join()


def cpu_time(server):
    elapsed = subprocess.check_output(['ps', '-o', 'time=', '-p', str(server.process.pid)], text=True)
    total = 0.0
    for field in elapsed.strip().split(':'):
        total = 60 * total + float(field)
    return total


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--binary', type=Path, required=True)
    parser.add_argument('--cases', nargs='+', choices=['healthy', 'slow', 'refused', 'oscillating'],
                        default=['healthy', 'slow', 'refused', 'oscillating'])
    args = parser.parse_args()
    with tempfile.TemporaryDirectory(prefix='tunetrail-relay-e4-') as root:
        for case in args.cases:
            print(json.dumps(exercise(args.binary.resolve(strict=True), Path(root) / case, case)), flush=True)


if __name__ == '__main__':
    main()
