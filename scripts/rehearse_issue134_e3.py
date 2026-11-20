#!/usr/bin/env python3
"""Rehearse the occurrence adapters and recovery against a large disposable ledger."""

import argparse
from concurrent.futures import ThreadPoolExecutor
import json
from pathlib import Path
import tempfile
import time

from rehearse_issue081 import PROFILE, boot, database, file_fixtures, init, submit, wait_for
from rehearse_issue134_e1 import verify_replay
from validate_multiuser import ROOT


def rehearse(binary, baseline, root):
    directory = root / "load"
    init(baseline, directory)
    with database(directory) as conn:
        conn.execute("INSERT INTO integrations(id,profile_id,source_type,created_at) VALUES('bulk',1,'fixture',0)")
        conn.execute("""WITH RECURSIVE n(i) AS (SELECT 1 UNION ALL SELECT i+1 FROM n WHERE i<300000)
            INSERT INTO ingest_events(receipt_id,profile_id,source_instance_id,schema_version,source_type,
                adapter_version,batch_id,item_ordinal,operation,received_at,time_precision,time_semantics,
                original_snapshot,snapshot_digest,digest_version)
            SELECT 'bulk-'||i,1,'bulk',1,'fixture','1','bulk',i,'import',0,'second','unknown','{}','fixture','1' FROM n""")
        conn.commit()
        conn.execute("PRAGMA wal_checkpoint(TRUNCATE)")

    start = time.monotonic()
    server = boot(binary, directory, True)
    upgrade = time.monotonic() - start
    try:
        with database(directory) as conn:
            assert conn.execute("SELECT count(*) FROM ingest_events").fetchone() == (300000,)
            sql = (ROOT / "internal/db/sqlite/ingest_occurrence_catalog.go").read_text()
            query = sql.split("QueryContext(ctx, `SELECT o.id", 1)[1].split("`", 1)[0]
            plan = [row[3] for row in conn.execute("EXPLAIN QUERY PLAN SELECT o.id" + query,
                                                    ("Song", "Song", '["Artist"]', 1, False, ""))]
            start = time.monotonic()
            conn.execute("DELETE FROM ingest_events WHERE profile_id=1 AND mode='shadow'")
            conn.commit()
            purge = time.monotonic() - start
            conn.execute("INSERT INTO profile_ingest_flags(profile_id,active,occurrence_write) VALUES(1,1,1)")

        server.login()
        arrivals = {}
        first = 1780000000
        started = time.monotonic()
        with ThreadPoolExecutor(max_workers=2) as workers:
            list(workers.map(lambda _: submit(server, first, key="ack-replay"), range(2)))
        arrivals[first] = time.monotonic()
        with database(directory) as conn:
            assert conn.execute("SELECT count(*) FROM ingest_events WHERE mode='active'").fetchone() == (1,)
        submit(server, first, key="ack-replay", title="Changed observation")
        with database(directory) as conn:
            assert conn.execute("SELECT count(*) FROM ingest_observations").fetchone() == (1,)
        files = list(file_fixtures(first + 10000, 20))
        for form, stamps, payload in files:
            server.upload(PROFILE + "/import", form, payload)
            arrivals.update(dict.fromkeys(stamps, time.monotonic()))

        def states():
            with database(directory) as conn:
                return dict(conn.execute("SELECT state,count(*) FROM ingest_events GROUP BY state"))

        wait_for(lambda: states().get("accepted") == 101)
        visible_at = time.monotonic()
        # The same file is a replay independent of staged filename.
        form, _, payload = files[1]
        server.upload(PROFILE + "/import", form, payload)
        assert states() == {"accepted": 101}
        with database(directory) as conn:
            assert conn.execute("SELECT count(DISTINCT origin) FROM listens").fetchone() == (6,)
        # Change exact file bytes while keeping the same Last.fm history.
        server.upload(PROFILE + "/import", form, payload + b"\n")
        wait_for(lambda: states().get("pending_duplicate") == 20)
        assert states() == {"accepted": 101, "pending_duplicate": 20}

        with database(directory) as conn:
            conn.executescript("""CREATE TABLE injected_e3_failure(id INTEGER REFERENCES listening_profiles(id) DEFERRABLE INITIALLY DEFERRED);
                CREATE TRIGGER inject_e3_commit_failure AFTER INSERT ON ingest_revisions
                WHEN NEW.rule_version='occurrence-v1' BEGIN INSERT INTO injected_e3_failure VALUES(-1); END;""")
        faulty_stamp = first + 50000
        form, _, data = next(file_fixtures(faulty_stamp, 1))
        server.upload(PROFILE + "/import", form, data)
        wait_for(lambda: states().get("retryable") == 1)
        with database(directory) as conn:
            assert conn.execute("SELECT count(*) FROM listens WHERE listened_at=?", (faulty_stamp,)).fetchone() == (0,)
            conn.execute("DROP TRIGGER inject_e3_commit_failure")
    finally:
        server.stop()

    with database(directory) as conn:
        conn.execute("UPDATE ingest_events SET next_attempt_at=0 WHERE state='retryable'")
    server = boot(binary, directory, False)
    try:
        wait_for(lambda: states().get("accepted") == 102)
        submit(server, first + 60000, key="master-off", expected=503)
        assert states() == {"accepted": 102, "pending_duplicate": 20}
    finally:
        server.stop()

    return {"seed_shadow_receipts": 300000, "upgrade_health_seconds": round(upgrade, 3),
            "shadow_delete_commit_seconds": round(purge, 3), "accepted_first_wave": 101,
            "first_wave_seconds": round(visible_at - started, 3),
            "last_upload_ack_to_all_visible_ms": round(max(0, visible_at - max(arrivals.values())) * 1000, 1),
            "overlap_pending": 20, "commit_rollback_and_restart": True, "occurrence_query_plan": plan,
            "backup_bytes": next(directory.glob("*.pre-occurrence-ingest-*.bak")).stat().st_size}


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--binary", required=True, type=Path)
    parser.add_argument("--baseline", required=True, type=Path)
    args = parser.parse_args()
    with tempfile.TemporaryDirectory(prefix="tunetrail-e3-") as directory:
        root = Path(directory)
        print(json.dumps(verify_replay(args.binary.resolve(), args.baseline.resolve(), root)), flush=True)
        print(json.dumps(rehearse(args.binary.resolve(), args.baseline.resolve(), root)), flush=True)


if __name__ == "__main__":
    main()
