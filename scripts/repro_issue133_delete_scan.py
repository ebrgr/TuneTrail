#!/usr/bin/env python3
"""Reproduce the ISSUE-133 foreign-key scan using only throwaway synthetic data."""

import argparse
import json
from pathlib import Path
import sqlite3
import tempfile
import time

from validate_ingest_migration import check, component_hashes, copy_components
from validate_multiuser import Server


def start_and_stop(binary, directory):
    server = Server(binary, directory)
    try:
        server.start()
    finally:
        server.stop()


def run(v7_binary, v8_binary, receipts):
    with tempfile.TemporaryDirectory(prefix="tunetrail-133-scan-", dir="/tmp") as temporary:
        root = Path(temporary)
        source_dir = root / "source"
        start_and_stop(v7_binary, source_dir)
        source = source_dir / "tunetrail.db"
        check(source, 7)
        with sqlite3.connect(source) as writer:
            writer.execute("PRAGMA foreign_keys=ON")
            writer.execute("PRAGMA journal_mode=WAL")
            writer.execute("PRAGMA wal_autocheckpoint=0")
            writer.executescript("""
                INSERT INTO releases(id) VALUES(1);
                INSERT INTO tracks(id,release_id) VALUES(1,1);
                INSERT INTO integrations(id,profile_id,source_type,created_at)
                SELECT 'fixture',id,'listenbrainz',0 FROM listening_profiles;
            """)
            writer.execute("""
                WITH RECURSIVE items(n) AS (
                    SELECT 1 UNION ALL SELECT n+1 FROM items WHERE n < ?
                )
                INSERT INTO ingest_events (
                    receipt_id,profile_id,source_instance_id,schema_version,source_type,
                    adapter_version,batch_id,item_ordinal,operation,received_at,time_precision,
                    time_semantics,original_snapshot,snapshot_digest,digest_version,duplicate_decision
                )
                SELECT printf('receipt-%d',n),profile_id,'fixture',1,'listenbrainz',
                       'fixture','batch',n,'submit',0,'estimated','start','{}','fixture','1','review'
                FROM items CROSS JOIN integrations WHERE integrations.id='fixture'
            """, (receipts,))
            writer.commit()

            # Keep the idle writer open so the WAL and SHM survive the component copy.
            before = component_hashes(source)
            assert {"", "-wal", "-shm"} <= before.keys()
            target_dir = root / "copy"
            target_dir.mkdir()
            target = target_dir / "tunetrail.db"
            copy_components(source, target)
            assert component_hashes(target) == before
            start_and_stop(v8_binary, target_dir)
            check(target, 8)
            assert component_hashes(source) == before

        with sqlite3.connect(target) as conn:
            conn.execute("PRAGMA foreign_keys=ON")
            count, linked = conn.execute(
                "SELECT count(*), count(occurrence_id)+count(recording_id) FROM ingest_events"
            ).fetchone()
            assert (count, linked) == (receipts, 0)
            plans = {}
            for table in ("occurrences", "recordings"):
                details = conn.execute(
                    f"EXPLAIN QUERY PLAN DELETE FROM {table} WHERE id=1"
                )
                plans[table] = [row[3] for row in details if "ingest_events" in row[3]]
            conn.execute("BEGIN IMMEDIATE")
            started = time.perf_counter()
            conn.execute("DELETE FROM tracks WHERE id=1")
            elapsed = time.perf_counter() - started
            assert conn.execute("SELECT count(*) FROM occurrences").fetchone() == (0,)
            assert conn.execute("SELECT count(*) FROM recordings").fetchone() == (0,)
            assert conn.execute("SELECT count(*) FROM ingest_events").fetchone() == (receipts,)
            conn.rollback()
            return {
                "receipts": receipts,
                "receipts_linked": linked,
                "source_components_unchanged": True,
                "sqlite_version": sqlite3.sqlite_version,
                "v8_fk_delete_plans": plans,
                "legacy_track_delete_ms": round(elapsed * 1000, 3),
                "confirmed": all("SCAN ingest_events" in plan for plan in plans.values()),
            }


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--v7", type=Path, required=True, help="TuneTrail v0.42 binary")
    parser.add_argument("--v8", type=Path, required=True, help="TuneTrail built at ae09b13")
    parser.add_argument("--receipts", type=int, default=1)
    args = parser.parse_args()
    if args.receipts < 1:
        parser.error("--receipts must be positive")
    print(json.dumps(run(args.v7.resolve(strict=True), args.v8.resolve(strict=True), args.receipts), sort_keys=True))


if __name__ == "__main__":
    main()
