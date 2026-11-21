#!/usr/bin/env python3
"""Measure durable relay delivery using disposable profiles and loopback receivers."""
import argparse
from collections import Counter
from concurrent.futures import ThreadPoolExecutor
from contextlib import closing
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
import json
from pathlib import Path
import shutil
import socket
import sqlite3
import subprocess
import tempfile
import threading
import time

from rehearse_issue081 import percentiles, sample_resources
from rehearse_issue134_e4 import large_history
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
            port = receiver.server_port if scenario not in ('refused', 'capacity') else refused.getsockname()[1]
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
                    if scenario in ('refused', 'capacity') and row and row[0] and sum(state.values()) == count:
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
                final = settle(db, files * size + 40, 400)
                duration = time.monotonic() - start
                if scenario not in ('refused', 'capacity'):
                    assert final == ({'sent': 4039, 'failed': 1} if scenario == 'healthy' else {'sent': 160}), final
                    assert len(set(accepted)) == final['sent']
                    assert posts and max(posts) <= 100
                assert db.execute('SELECT count(*) FROM listens').fetchone()[0] == files * size + 40
                saturated_ack = None
                storage_ack = None
                if scenario == 'capacity':
                    fill_capacity(db)
                    saturated_ack = percentiles([submit(files * size + 40 + n, 1) for n in range(40)])
                    final = settle(db, 100040, 60)
                    assert final.get('failed') == 40
                    assert db.execute("SELECT count(*) FROM outbox WHERE json_extract(result,'$.reason')='queue_capacity'").fetchone()[0] == 40
                    assert db.execute('SELECT open_count FROM relay_destinations').fetchone()[0] == 100000
                    assert db.execute('SELECT count(*) FROM listens').fetchone()[0] == 100040
                    assert db.execute("SELECT count(*) FROM outbox WHERE instr(coalesce(result,''),'e4-fictional')").fetchone()[0] == 0
                    # Model an unavailable local writer separately from delivery capacity.
                    db.execute('BEGIN IMMEDIATE')
                    try:
                        unavailable_at = time.monotonic()
                        app.request('POST', LBZ + '/submit-listens', scrobble(start_stamp + 1000),
                                    token=key, expected=503)
                        storage_ack = round(time.monotonic() - unavailable_at, 3)
                    finally:
                        db.rollback()
                before = cpu_time(app)
                time.sleep(3)
                idle_cpu = cpu_time(app) - before
            return dict(scenario=scenario, seconds=round(duration, 3), posts=len(posts),
                        batch_sizes=dict(sorted(Counter(posts).items())), states=final, pauses=sorted(pauses),
                        ack=percentiles(acks), saturated_ack=saturated_ack, local_writer_503_seconds=storage_ack, imports=percentiles(imports), idle_cpu_seconds=round(idle_cpu, 3),
                        duplicates=len(accepted) - len(set(accepted)),
                        inversions=sum(a > b for a, b in zip(accepted, accepted[1:])), **peaks)
        finally:
            ready.set()
            app.stop()
            receiver.shutdown()
            thread.join()


def fill_capacity(db):
    # Seed no-send history while the refused destination remains paused.
    db.execute("UPDATE relay_destinations SET paused_until=unixepoch()+3600,pause_class='transient'")
    db.executescript("""
        CREATE TEMP TABLE e4_sequence(n INTEGER PRIMARY KEY);
        INSERT INTO e4_sequence WITH RECURSIVE seq(n) AS
            (VALUES(1) UNION ALL SELECT n+1 FROM seq WHERE n<99840) SELECT n FROM seq;
        INSERT INTO listens(profile_id,track_id,listened_at,event_id)
            SELECT 1,(SELECT min(track_id) FROM listens),100000+n,'e4-capacity-'||n FROM e4_sequence;
        INSERT INTO ingest_events(receipt_id,profile_id,source_instance_id,schema_version,source_type,adapter_version,
            batch_id,item_ordinal,operation,received_at,listened_at,time_precision,time_semantics,original_snapshot,
            snapshot_digest,digest_version,state,identity_status,event_id,occurrence_id,recording_id,mode,delivery_policy,closed_at)
            SELECT l.event_id,1,e.source_instance_id,1,'listenbrainz','1',l.event_id,0,'import',unixepoch(),
                e.listened_at,'second','start',e.original_snapshot,'fixture','1','accepted','resolved',
                l.event_id,e.occurrence_id,e.recording_id,'active','durable',unixepoch()
            FROM listens l CROSS JOIN (SELECT * FROM ingest_events WHERE state='accepted' LIMIT 1) e
            WHERE l.event_id LIKE 'e4-capacity-%';
        INSERT INTO outbox(id,profile_id,destination_id,event_id,operation,delivery_key,created_at,next_attempt_at,format_version)
            SELECT event_id,1,'relay:1',event_id,'listen',event_id,unixepoch(),unixepoch(),1
            FROM ingest_events WHERE receipt_id LIKE 'e4-capacity-%';
    """)
    assert db.execute('SELECT open_count FROM relay_destinations').fetchone()[0] == 100000



def upgrade(binary, baseline, root):
    app = Server(baseline, root / 'upgrade')
    app.start()
    try:
        app.login()
        profile = WEB + '/profiles/'
        app.request('POST', WEB + '/profiles', {'slug': 'fixture-member', 'display_name': 'Fixture member'},
                    admin=True, expected=201)
        for slug in ('default', 'fixture-member'):
            app.request('PUT', profile + slug + '/relay',
                        {'url': 'http://127.0.0.1:1/fixture', 'token': 'upgrade-fictional-token'},
                        admin=True, expected=204)
    finally:
        app.stop()

    current = Server(binary, app.directory)
    path = app.directory / 'tunetrail.db'
    with closing(sqlite3.connect(path)) as db:
        db.execute("UPDATE users SET role='user' WHERE id=(SELECT user_id FROM listening_profiles WHERE slug='fixture-member')")
        large_history(db)
        db.commit()
        wal_bytes = Path(str(path) + '-wal').stat().st_size
        assert wal_bytes > 0
        start = time.monotonic()
        try:
            current.start()
            seconds = time.monotonic() - start
            with closing(sqlite3.connect(path)) as upgraded:
                assert upgraded.execute('SELECT max(version_id) FROM goose_db_version').fetchone()[0] == 12
                assert upgraded.execute('SELECT count(*) FROM ingest_events').fetchone()[0] == 300000
                pauses = dict(upgraded.execute('SELECT p.slug,d.pause_class FROM relay_destinations d JOIN listening_profiles p ON p.id=d.profile_id'))
                assert pauses['default'] is None and pauses['fixture-member'].startswith('policy_blocked')
                assert upgraded.execute("SELECT count(*) FROM relay_allowlist WHERE origin='seeded'").fetchone()[0] == 2
            current.login()
            panel = current.request('GET', '/admin/settings', admin=True, headers={'Accept-Language': 'en'})
            assert b'Ask an instance administrator to add the required exception.' in panel
            assert b'upgrade-fictional-token' not in panel
            member_relay = profile + 'fixture-member/relay'
            response = current.request('GET', member_relay, admin=True)
            assert 'token' not in response
            for permission in ({'host_port': '127.0.0.1:1', 'allow_http': True},
                               {'host_port': '127.0.0.1:1', 'cidr': '127.0.0.1/32', 'allow_private': True}):
                current.request('PUT', member_relay + '/permissions', permission, admin=True, expected=204)
            with closing(sqlite3.connect(path)) as upgraded:
                assert upgraded.execute("""SELECT d.pause_class FROM relay_destinations d
                    JOIN listening_profiles p ON p.id=d.profile_id WHERE p.slug='fixture-member'""").fetchone() == (None,)
            log = (app.directory / 'server.log').read_text()
            assert 'seed_skipped_non_admin' in log and 'upgrade-fictional-token' not in log
            backup, = app.directory.glob('tunetrail.db.pre-relay-delivery-*.bak')
            backup_bytes = backup.stat().st_size
            # Physical backups preserve credentials needed to restore v11.
            with closing(sqlite3.connect(backup)) as saved:
                assert saved.execute('SELECT count(*) FROM ingest_events').fetchone()[0] == 300000
                assert saved.execute('SELECT count(*) FROM profile_relays WHERE token=?', ('upgrade-fictional-token',)).fetchone()[0] == 2
        finally:
            current.stop()
    restore = Server(baseline, root / 'restore')
    restore.directory.mkdir()
    shutil.copy2(backup, restore.directory / 'tunetrail.db')
    try:
        restore.start()
        with closing(sqlite3.connect(restore.directory / 'tunetrail.db')) as saved:
            assert saved.execute('SELECT max(version_id) FROM goose_db_version').fetchone()[0] == 11
            assert saved.execute('SELECT count(*) FROM ingest_events').fetchone()[0] == 300000
    finally:
        restore.stop()
    return dict(scenario='upgrade', seconds=round(seconds, 3), wal_v11_bytes=wal_bytes,
                backup_bytes=backup_bytes, database_v12_bytes=path.stat().st_size,
                member_paused=True, admin_seeded=True, restore_v11=True)



def cpu_time(server):
    elapsed = subprocess.check_output(['ps', '-o', 'time=', '-p', str(server.process.pid)], text=True)
    total = 0.0
    for field in elapsed.strip().split(':'):
        total = 60 * total + float(field)
    return total


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--binary', type=Path, required=True)
    parser.add_argument('--baseline', type=Path)
    parser.add_argument('--cases', nargs='+', choices=['healthy', 'slow', 'refused', 'oscillating', 'capacity', 'upgrade'],
                        default=['healthy', 'slow', 'refused', 'oscillating'])
    args = parser.parse_args()
    with tempfile.TemporaryDirectory(prefix='tunetrail-relay-e4-') as root:
        for case in args.cases:
            binary = args.binary.resolve(strict=True)
            if case == 'upgrade':
                if args.baseline is None:
                    parser.error('--cases upgrade requires --baseline')
                result = upgrade(binary, args.baseline.resolve(strict=True), Path(root))
            else:
                result = exercise(binary, Path(root) / case, case)
            print(json.dumps(result), flush=True)


if __name__ == '__main__':
    main()
