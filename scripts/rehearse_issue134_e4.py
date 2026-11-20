#!/usr/bin/env python3
"""Measure ISSUE-134 on disposable v0.46 and v11 servers."""

import argparse
from concurrent.futures import ThreadPoolExecutor
from contextlib import closing, contextmanager, suppress
from datetime import datetime, timezone
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
import io
import json
from pathlib import Path
import shutil
import sqlite3
import tempfile
import threading
import time
import urllib.error
import urllib.request
import zipfile

from rehearse_issue081 import PROFILE, SUBMIT, TOKEN, boot, database, init, percentiles, sample_resources, wait_for
from rehearse_issue134_e1 import verify_replay
from rehearse_issue134_e3 import catalog_query_plans
from validate_multiuser import ROOT, Server, scrobble


def post(server, payload, key):
    request = urllib.request.Request(server.origin + SUBMIT, json.dumps(payload).encode(),
                                     headers={"Content-Type": "application/json", "Authorization": "Token " + TOKEN,
                                              "Idempotency-Key": key})
    try:
        response = urllib.request.urlopen(request, timeout=30)
    except urllib.error.HTTPError as error:
        response = error
    with response:
        return response.status, response.headers.get("Retry-After"), response.read()


def large_history(conn):
    conn.executescript("""
        PRAGMA foreign_keys=ON;
        PRAGMA wal_autocheckpoint=0;
        CREATE TEMP TABLE sequence(n INTEGER PRIMARY KEY);
        INSERT INTO sequence WITH RECURSIVE seq(n) AS (SELECT 1 UNION ALL SELECT n+1 FROM seq WHERE n<300000) SELECT n FROM seq;
        INSERT INTO artists(id) VALUES(1);
        INSERT INTO artist_aliases VALUES(1,'Archive Artist','Fixture',1);
        INSERT INTO releases(id) SELECT n FROM sequence WHERE n<=200;
        INSERT INTO release_aliases SELECT n,'Archive Album '||n,'Fixture',1 FROM sequence WHERE n<=200;
        INSERT INTO artist_releases SELECT 1,n,1 FROM sequence WHERE n<=200;
        INSERT INTO tracks(id,release_id,duration) SELECT n,n,180 FROM sequence WHERE n<=200;
        INSERT INTO track_aliases SELECT n,'Archive Song '||n,'Fixture',1 FROM sequence WHERE n<=200;
        INSERT INTO artist_tracks SELECT 1,n,1 FROM sequence WHERE n<=200;
        INSERT INTO integrations(id,profile_id,source_type,created_at) VALUES('listenbrainz:1',1,'listenbrainz',0);
        INSERT INTO listens(profile_id,track_id,listened_at,event_id,client)
        SELECT 1,(n-1)%200+1,1640995200+n*4,'history-'||n,'Fixture' FROM sequence;
        INSERT INTO ingest_events(receipt_id,profile_id,source_instance_id,schema_version,source_type,adapter_version,
            batch_id,item_ordinal,operation,received_at,listened_at,original_time,time_precision,time_semantics,
            original_snapshot,snapshot_digest,digest_version,mode,state,identity_status,duplicate_decision,
            resolution_version,rule_version,occurrence_id,recording_id,event_id,closed_at)
        SELECT 'receipt-'||n,1,'listenbrainz:1',1,'listenbrainz','1','imported-history',n,'import',l.listened_at,
            strftime('%Y-%m-%dT%H:%M:%S',l.listened_at,'unixepoch')||'.000000000Z',CAST(l.listened_at AS TEXT),
            'second','start',json_object('title','Archive Song '||l.track_id,'artist','Archive Artist',
            'release','Archive Album '||l.track_id,'original_time',CAST(l.listened_at AS TEXT),'duration_ms',180000),
            'history-digest-'||n,'sha256-json-v2','active','accepted','resolved','different',1,'active-v1',
            l.track_id,o.recording_id,l.event_id,l.listened_at
        FROM sequence JOIN listens l ON l.event_id='history-'||n JOIN occurrences o ON o.id=l.track_id;
        INSERT INTO ingest_revisions
        SELECT profile_id,receipt_id,1,'automatic','different','["fixture"]','{}','active-v1',received_at FROM ingest_events;
        DROP TABLE sequence;
    """)


def benchmark_load(server, directory, duration, rate):
    first = 1781000000
    count = duration * rate
    arrivals, acknowledgements, visible, statuses = {}, {}, {}, {}
    stop = threading.Event()

    def observe():
        with database(directory) as conn:
            while not stop.is_set():
                now = time.monotonic()
                for (stamp,) in conn.execute("SELECT listened_at FROM listens WHERE profile_id=1 AND listened_at>=?", (first,)):
                    visible.setdefault(stamp, now)
                stop.wait(.025)

    def send(stamp):
        arrivals[stamp] = time.monotonic()
        status, _, _ = post(server, scrobble(stamp), f"load-{stamp}")
        acknowledgements[stamp] = time.monotonic()
        statuses[stamp] = status

    observer = threading.Thread(target=observe)
    observer.start()
    initial_wal = (directory / "tunetrail.db-wal").stat().st_size
    try:
        with sample_resources(server) as peaks:
            beginning = time.monotonic()
            with ThreadPoolExecutor(max_workers=16) as pool:
                futures = []
                for i in range(count):
                    remaining = beginning + i / rate - time.monotonic()
                    if remaining > 0:
                        time.sleep(remaining)
                    futures.append(pool.submit(send, first + i * 4))
                for future in futures:
                    future.result()
            assert set(statuses.values()) == {200}, statuses
            wait_for(lambda: len(visible) == count)
            elapsed = max(visible.values()) - beginning
    finally:
        stop.set()
        observer.join()
    with database(directory, timeout=0) as lock:
        lock.execute("BEGIN IMMEDIATE")
        before_busy = time.monotonic()
        status, retry, _ = post(server, scrobble(first + count * 4), "busy-probe")
        busy_ms = (time.monotonic() - before_busy) * 1000
        assert (status, retry) == (503, "1")
        lock.rollback()
    assert post(server, scrobble(first + count * 4), "busy-probe")[0] == 200
    wait_for(lambda: event_count(directory, first) == count + 1)
    return {"count": count, "rate_target_per_second": rate, "duration_target_s": duration,
            "visible_per_second": round(count / elapsed, 2), "elapsed_s": round(elapsed, 3),
            "ack": percentiles([acknowledgements[s] - arrivals[s] for s in arrivals]),
            "visibility_since_send": percentiles([visible[s] - arrivals[s] for s in arrivals]),
            "visibility_since_ack": percentiles([max(0, visible[s] - acknowledgements[s]) for s in arrivals]),
            "busy_ms": round(busy_ms, 1), "retry_after_seconds": retry, "initial_wal_bytes": initial_wal, **peaks}


def event_count(directory, first):
    with database(directory) as conn:
        return conn.execute("SELECT count(*) FROM listens WHERE profile_id=1 AND listened_at>=?", (first,)).fetchone()[0]


def sustained(binary, baseline, root, duration, rate):
    directory = root / "load"
    init(baseline, directory)
    with closing(sqlite3.connect(directory / "tunetrail.db")) as conn:
        large_history(conn)
        conn.commit()
        original_wal = (directory / "tunetrail.db-wal").stat().st_size
        assert original_wal > 0
        start = time.monotonic()
        server = boot(binary, directory, True)
        upgrade_seconds = time.monotonic() - start
        upgraded_bytes = sum(path.stat().st_size for path in directory.iterdir() if path.is_file())
    try:
        with database(directory) as conn:
            assert conn.execute("SELECT count(*) FROM ingest_events WHERE profile_id=1").fetchone() == (300000,)
            conn.execute("INSERT INTO profile_ingest_flags(profile_id,active,occurrence_write) VALUES(1,1,1)")
            plans = catalog_query_plans(conn)
            code = (ROOT / "internal/db/sqlite/ingest_resolve.go").read_text()
            claim = code.split("const nextReceipt = `", 1)[1].split("`", 1)[0]
            plans["claim_query_plan"] = [row[3] for row in conn.execute("EXPLAIN QUERY PLAN " + claim, (0, 0))]
        workload = benchmark_load(server, directory, duration, rate)
    finally:
        server.stop()
    backup, = directory.glob("*.pre-occurrence-ingest-*.bak")
    restored = root / "restored"
    restored.mkdir()
    shutil.copyfile(backup, restored / "tunetrail.db")
    server = boot(baseline, restored)
    try:
        with database(restored) as conn:
            assert conn.execute("SELECT max(version_id) FROM goose_db_version").fetchone() == (9,)
            assert conn.execute("SELECT count(*) FROM ingest_events WHERE profile_id=1").fetchone() == (300000,)
            assert conn.execute("PRAGMA foreign_key_check").fetchall() == []
    finally:
        server.stop()
    old = Server(baseline, directory)
    try:
        old.start()
    except RuntimeError:
        assert "unsupported schema version 11" in (directory / "server.log").read_text()
    else:
        raise AssertionError("v0.46 opened schema v11")
    finally:
        old.stop()
    return {"retained_same_profile": 300000, "upgrade_health_s": round(upgrade_seconds, 3),
            "v9_wal_bytes": original_wal, "upgrade_bytes_at_health": upgraded_bytes,
            "backup_bytes": backup.stat().st_size, "restore_v046": True, "v046_refuses_v11": True,
            "workload": workload, **plans}


@contextmanager
def loopback(handler):
    server = ThreadingHTTPServer(("127.0.0.1", 0), handler)
    worker = threading.Thread(target=server.serve_forever, daemon=True)
    worker.start()
    try:
        yield f"http://127.0.0.1:{server.server_port}"
    finally:
        server.shutdown()
        server.server_close()
        worker.join()


def parity(binary, baseline, root):
    projection = verify_replay(binary, baseline, root)
    relayed = []

    class Relay(BaseHTTPRequestHandler):
        def log_message(self, *_):
            pass

        def do_POST(self):
            body = self.rfile.read(int(self.headers["Content-Length"]))
            relayed.append((self.path, self.headers.get("Authorization"), body))
            self.send_response(200)
            self.end_headers()

    payload = scrobble(1783000000)
    with loopback(Relay) as url:
        for index, executable in enumerate((baseline, binary)):
            directory = root / f"relay-{index}"
            init(executable, directory)
            with database(directory) as conn:
                conn.execute("INSERT INTO profile_relays VALUES(1,?,'fixture-relay')", (url,))
            server = boot(executable, directory)
            try:
                assert post(server, payload, "legacy-relay")[0] == 200
                wait_for(lambda: len(relayed) == index + 1)
            finally:
                server.stop()
    assert relayed[0] == relayed[1]
    assert relayed[1] == ("/submit-listens", "Token fixture-relay", json.dumps(payload).encode())
    projection["restore_v046"] = projection.pop("restore_v044")
    return {**projection, "relay_body_and_headers_equal": True}


def overlap_files(stamp):
    artist, album, title = "Shared Artist", "Shared Album", "Shared Song"
    meta = {"artist_name": artist, "release_name": album, "track_name": title}
    archive = io.BytesIO()
    with zipfile.ZipFile(archive, "w") as zipped:
        zipped.writestr("listens/fixture.jsonl", '{"listened_at":' + str(stamp) + '.125,"track_metadata":'
                       + json.dumps(meta) + "}\n")
    yield "listenbrainz", archive.getvalue()
    yield "lastfm", json.dumps([{"track": [{"artist": {"#text": artist}, "album": {"#text": album},
                                          "name": title, "date": {"uts": str(stamp)}}]}]).encode()
    clock = datetime.fromtimestamp(stamp, timezone.utc).strftime("%Y-%m-%dT%H:%M:%S")
    yield "spotify", json.dumps([{"ts": clock + ".2500Z", "master_metadata_track_name": title,
                                  "master_metadata_album_artist_name": artist,
                                  "master_metadata_album_album_name": album, "reason_end": "trackdone",
                                  "ms_played": 180000}]).encode()
    yield "maloja", json.dumps({"scrobbles": [{"time": stamp, "track": {"artists": [artist], "title": title,
                                  "album": {"albumtitle": album}}}]}).encode()

    def aliases(value):
        return {"aliases": [{"alias": value, "is_primary": True}]}

    yield "koito", json.dumps({"version": "1", "listens": [{"listened_at": clock + ".100Z",
                                "track": aliases(title), "album": aliases(album),
                                "artists": [aliases(artist)]}]}).encode()


def occurrence_fixture(binary, directory, **env):
    init(binary, directory)
    with database(directory) as conn:
        conn.execute("INSERT INTO profile_ingest_flags(profile_id,active,occurrence_write) VALUES(1,1,1)")
    return boot(binary, directory, True, **env)


def overlap(binary, root):
    directory = root / "overlap"
    server = occurrence_fixture(binary, directory)
    try:
        server.login()
        stamp = 1783001000
        for index in range(20):
            first = stamp + index * 10
            payload = scrobble(first)
            payload["payload"][0]["track_metadata"].update(artist_name="Shared Artist", release_name="Shared Album", track_name="Shared Song")
            assert post(server, payload, f"primary-{index}")[0] == 200
        wait_for(lambda: event_count(directory, stamp) == 20)
        grouped = {}
        for index in range(20):
            for format, data in overlap_files(stamp + index * 10):
                grouped.setdefault(format, []).append(data)
        # Upload separate files to retain the genuine source digest and time representation.
        for format, files in grouped.items():
            for data in files:
                server.upload(PROFILE + "/import", format, data)

        def pending():
            with database(directory) as conn:
                return conn.execute("SELECT count(*) FROM ingest_events WHERE state='pending_duplicate'").fetchone()[0]

        wait_for(lambda: pending() == 100)
        assert event_count(directory, stamp) == 20
        with database(directory) as conn:
            distribution = conn.execute("""SELECT source_type,coalesce(file_format,'api'),state,time_precision,reason_codes,count(*)
                FROM ingest_events GROUP BY 1,2,3,4,5""").fetchall()
            evidence = conn.execute("""SELECT count(*) FROM ingest_events WHERE state='pending_duplicate'
                AND json_array_length(json_extract(evidence,'$.candidates'))=1
                AND json_extract(evidence,'$.candidates[0]') LIKE 'event:%'""").fetchone()[0]
            assert evidence == 100
            assert conn.execute("SELECT count(*) FROM outbox").fetchone() == (0,)
        return {"events": 20, "pending": 100, "one_candidate_each": evidence, "distribution": distribution}
    finally:
        server.stop()


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--binary", type=Path, required=True)
    parser.add_argument("--baseline", type=Path, required=True)
    parser.add_argument("--scenario", choices=("load", "parity", "overlap"), default="load")
    parser.add_argument("--duration", type=int, default=30)
    parser.add_argument("--rate", type=int, default=20)
    args = parser.parse_args()
    with tempfile.TemporaryDirectory(prefix="tunetrail-e4-") as tmp:
        root = Path(tmp)
        binary, baseline = args.binary.resolve(), args.baseline.resolve()
        if args.scenario == "load":
            result = sustained(binary, baseline, root, args.duration, args.rate)
        elif args.scenario == "parity":
            result = parity(binary, baseline, root)
        else:
            result = overlap(binary, root)
        print(json.dumps(result), flush=True)


if __name__ == "__main__":
    main()
