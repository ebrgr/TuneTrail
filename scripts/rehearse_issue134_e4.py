#!/usr/bin/env python3
"""Measure ISSUE-134 on disposable v0.46 and v11 servers."""

import argparse
from concurrent.futures import ThreadPoolExecutor
from contextlib import closing
import json
from pathlib import Path
import shutil
import sqlite3
import tempfile
import threading
import time
import urllib.error
import urllib.request

from rehearse_issue081 import SUBMIT, TOKEN, boot, database, init, percentiles, sample_resources, wait_for
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


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--binary", type=Path, required=True)
    parser.add_argument("--baseline", type=Path, required=True)
    parser.add_argument("--duration", type=int, default=30)
    parser.add_argument("--rate", type=int, default=20)
    args = parser.parse_args()
    with tempfile.TemporaryDirectory(prefix="tunetrail-e4-") as tmp:
        print(json.dumps(sustained(args.binary.resolve(), args.baseline.resolve(), Path(tmp),
                                   args.duration, args.rate)), flush=True)


if __name__ == "__main__":
    main()
