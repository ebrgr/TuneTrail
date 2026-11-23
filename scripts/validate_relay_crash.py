#!/usr/bin/env python3
"""Exercise process crashes on loopback at the three durable relay boundaries."""
import argparse
import json
import signal
import sqlite3
import tempfile
import threading
import time
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from pathlib import Path

from validate_admin_webui import approve_saved_relay
from validate_multiuser import LBZ, WEB, Server, scrobble


def wait_for(query, path):
    deadline = time.monotonic() + 8
    while time.monotonic() < deadline:
        with sqlite3.connect(path) as db:
            if db.execute(query).fetchone()[0]:
                return
        time.sleep(.05)
    raise AssertionError('fixture state did not become visible')


def run_case(binary, directory, boundary):
    arrived, release = threading.Event(), threading.Event()
    accepted = []
    repeated = threading.Event()

    class Receiver(BaseHTTPRequestHandler):
        def do_POST(self):
            items = json.loads(self.rfile.read(int(self.headers['Content-Length'])))['payload']
            if boundary in ('remote', 'commit') or release.is_set():
                accepted.extend((item['listened_at'], item['track_metadata']['track_name'], 'fixture-user') for item in items)
            arrived.set()
            if boundary != 'commit' or len(accepted) > 1:
                if boundary == 'commit':
                    repeated.set()
                release.wait(12)
            try:
                self.send_response(200)
                self.end_headers()
            except OSError:
                pass

        def log_message(self, *_):
            pass

    receiver = ThreadingHTTPServer(('127.0.0.1', 0), Receiver)
    thread = threading.Thread(target=receiver.serve_forever)
    thread.start()
    app = Server(binary, directory)
    path = directory/'tunetrail.db'
    try:
        app.start()
        app.login()
        profile = WEB+'/profiles/default'
        host = f'127.0.0.1:{receiver.server_port}'
        app.request('PUT', profile+'/relay/permissions', {'host_port':host,'allow_http':True}, admin=True, expected=204)
        app.request('PUT', profile+'/relay/permissions',
                    {'host_port':host,'cidr':'127.0.0.1/32','allow_private':True}, admin=True, expected=204)
        app.request('PUT', profile+'/relay', {'url':'http://'+host,'token':'fictional-crash-token'},
                    admin=True, expected=204)
        approve_saved_relay(app, 1)
        key = app.request('GET', profile+'/user/apikeys', admin=True)[0]['key']
        if boundary == 'admission':
            with sqlite3.connect(path) as db:
                db.execute("""CREATE TRIGGER test_relay_stop_resolution BEFORE UPDATE OF state ON ingest_events
                    WHEN NEW.state='accepted' BEGIN SELECT RAISE(ABORT,'fixture'); END""")
        if boundary == 'commit':
            with sqlite3.connect(path) as db:
                db.execute("CREATE TRIGGER test_stop_settlement BEFORE UPDATE OF state ON outbox "
                           "WHEN NEW.state='sent' BEGIN SELECT RAISE(ABORT,'fixture'); END")
        app.request('POST', LBZ+'/submit-listens', scrobble(int(time.time())-60), token=key,
                    headers={'Idempotency-Key':'durable-crash-fixture'})
        if boundary == 'commit':
            assert repeated.wait(8), '2xx did not repeat after settlement rollback'
            with sqlite3.connect(path) as db:
                db.execute('DROP TRIGGER test_stop_settlement')
            release.set()
        else:
            if boundary != 'admission':
                assert arrived.wait(8), 'relay did not reach stub'
                wait_for("SELECT count(*) FROM outbox WHERE lease_owner IS NOT NULL AND state='pending'", path)

            app.process.send_signal(signal.SIGKILL)
            assert app.process.wait(timeout=8) == -signal.SIGKILL
            app.stop()
            with sqlite3.connect(path) as db:
                if boundary == 'admission':
                    assert db.execute("SELECT count(*) FROM ingest_events WHERE delivery_policy='durable'").fetchone()[0] == 1
                    assert db.execute('SELECT count(*) FROM outbox').fetchone()[0] == 0
                    db.execute('DROP TRIGGER test_relay_stop_resolution')
                    db.execute("UPDATE ingest_events SET lease_until=1 WHERE lease_owner IS NOT NULL")
                    db.execute("UPDATE ingest_events SET next_attempt_at=1 WHERE next_attempt_at IS NOT NULL")
                else:
                    assert db.execute('SELECT state FROM outbox').fetchall() == [('pending',)]
                    # Move the persisted deadline into the past instead of sleeping for the real lease.
                    db.execute('UPDATE outbox SET lease_until=1 WHERE lease_owner IS NOT NULL')
                    db.execute('UPDATE relay_destinations SET lease_until=1 WHERE lease_owner IS NOT NULL')
            release.set()
            app.start()
        wait_for("SELECT count(*) FROM outbox WHERE state='sent'", path)
        app.login()
        exported = app.request('GET', WEB+'/profiles/default/export', admin=True)
        assert 'fictional-crash-token' not in str(exported)
        with sqlite3.connect(path) as db:
            assert db.execute('SELECT count(*) FROM listens').fetchone()[0] == 1
            assert db.execute('PRAGMA foreign_key_check').fetchall() == []
        assert len(accepted) == (2 if boundary in ('remote', 'commit') else 1), boundary
        assert len(set(accepted)) == 1
        print(json.dumps(dict(boundary=boundary, accepted=len(accepted), unique=len(set(accepted)),
                              duplicate_attempts=len(accepted) - len(set(accepted)))))
    finally:
        app.stop()
        release.set()
        receiver.shutdown()
        receiver.server_close()
        thread.join()


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('--binary', type=Path, required=True)
    args = parser.parse_args()
    with tempfile.TemporaryDirectory(prefix='tunetrail-relay-crash-') as root:
        for boundary in ('admission', 'claim', 'remote', 'commit'):
            run_case(args.binary.resolve(), Path(root)/boundary, boundary)


if __name__ == '__main__':
    main()
