#!/usr/bin/env python3
"""Check v0.44 parity and time the additive v10 upgrade on disposable data."""

import argparse
from contextlib import closing
import json
from pathlib import Path
import shutil
import sqlite3
import tempfile
import time

from rehearse_issue081 import PROFILE, boot, database, file_fixtures, init, submit
from validate_multiuser import ROOT


def export(server):
    payload = server.request("GET", PROFILE + "/export", admin=True)
    del payload["exported_at"]  # Export time is wall-clock metadata.
    return payload


def replay(binary, directory):
    server = boot(binary, directory)
    try:
        server.login()
        for stamp in (1780000000, 1780000000, 1780000004):
            assert submit(server, stamp, key=f"e1-{stamp}") == {"status": "ok"}
        for form, _, payload in file_fixtures(1780010000, 3):
            server.upload(PROFILE + "/import", form, payload)
        exported = export(server)
    finally:
        server.stop()
    with database(directory) as conn:
        listens = conn.execute("""SELECT profile_id,track_id,listened_at,client,origin
            FROM listens ORDER BY profile_id,listened_at,track_id""").fetchall()
        shadow = conn.execute("""SELECT source_type,operation,item_ordinal,original_snapshot,
            snapshot_digest,state,duplicate_decision,rule_version,mode
            FROM ingest_events ORDER BY source_type,operation,item_ordinal,snapshot_digest""").fetchall()
        assert all(row[-1] == "legacy" for row in listens)
        assert all(row[-1] == "shadow" for row in shadow)
        assert conn.execute("PRAGMA foreign_key_check").fetchall() == []
    return exported, listens, shadow


def verify_replay(binary, baseline, root):
    seed, old, new, restored = (root / name for name in ("seed", "baseline", "upgraded", "restored"))
    init(baseline, seed)
    seeded = boot(baseline, seed)
    try:
        seeded.login()
        submit(seeded, 1779999999, key="before-upgrade")
        seed_export = export(seeded)
    finally:
        seeded.stop()
    for directory in (old, new, restored):
        directory.mkdir()
    for directory in (old, new):
        shutil.copyfile(seed / "tunetrail.db", directory / "tunetrail.db")
    original = replay(baseline, old)
    upgraded = replay(binary, new)
    for name, before, after in zip(("export", "legacy listens", "shadow receipts"), original, upgraded):
        assert before == after, f"{name} projection differs"
    backup, = new.glob("tunetrail.db.pre-occurrence-ingest-*.bak")
    shutil.copyfile(backup, restored / "tunetrail.db")
    with database(restored) as conn:
        assert conn.execute("SELECT max(version_id) FROM goose_db_version").fetchone() == (9,)
    server = boot(baseline, restored)
    try:
        server.login()
        assert export(server) == seed_export
    finally:
        server.stop()
    assert replay(baseline, restored) == original
    return {"listens": len(original[1]), "shadow_receipts": len(original[2]), "restore_v044": True}


def measure(binary, baseline, root, count):
    fixture = root / "bulk"
    init(baseline, fixture)
    with database(fixture) as conn:
        conn.execute("INSERT INTO integrations(id,profile_id,source_type,created_at) VALUES('bulk',1,'fixture',0)")
        conn.execute("""WITH RECURSIVE n(i) AS (SELECT 1 UNION ALL SELECT i+1 FROM n WHERE i<?)
            INSERT INTO ingest_events(receipt_id,profile_id,source_instance_id,schema_version,source_type,
                adapter_version,batch_id,item_ordinal,operation,received_at,time_precision,time_semantics,
                original_snapshot,snapshot_digest,digest_version)
            SELECT 'bulk-'||i,1,'bulk',1,'fixture','1','bulk',i,'import',0,'second','unknown','{}','fixture','1' FROM n""", (count - 1,))
        conn.executescript("""
            INSERT INTO releases(id) VALUES(100);
            INSERT INTO tracks(id,release_id) VALUES(100,100);
            INSERT INTO listens(profile_id,track_id,listened_at,event_id) VALUES(1,100,42,'bulk-event');
            INSERT INTO ingest_events(receipt_id,profile_id,source_instance_id,schema_version,source_type,
                adapter_version,batch_id,item_ordinal,operation,received_at,time_precision,time_semantics,
                original_snapshot,snapshot_digest,digest_version,mode,state,identity_status,duplicate_decision,
                resolution_version,event_id)
            VALUES('bulk-drift',1,'bulk',1,'fixture','1','bulk',0,'import',0,'second','unknown',
                   '{}','fixture','1','active','accepted','resolved','different',1,'bulk-event');
        """)
        conn.commit()
        conn.execute("PRAGMA wal_checkpoint(TRUNCATE)")

    # Time the same SQL separately from process startup and bootstrap integrity checks.
    measured = root / "measured.db"
    ddl = (ROOT / "db/migrations_sqlite/010_occurrence_ingest.sql").read_text().split("-- +goose Down", 1)[0]
    timestamps = {}

    def trace(statement):
        prefixes = {"CREATE TEMP TABLE occurrence_link_repairs": "repair_start",
                    "CREATE TRIGGER trg_occurrence_flag_one_way": "repair_end"}
        for prefix, label in prefixes.items():
            if any(line.startswith(prefix) for line in statement.splitlines()):
                timestamps[label] = time.monotonic()

    with closing(sqlite3.connect(fixture / "tunetrail.db")) as source:
        start = time.monotonic()
        source.execute("VACUUM INTO ?", (str(measured),))
        backup = time.monotonic() - start
    with closing(sqlite3.connect(measured)) as conn:
        conn.execute("PRAGMA foreign_keys=ON")
        conn.set_trace_callback(trace)
        start = time.monotonic()
        conn.executescript("BEGIN IMMEDIATE;\n" + ddl + "\nCOMMIT;")
        duration = time.monotonic() - start
        repair = timestamps["repair_end"] - timestamps["repair_start"]
        assert conn.execute("SELECT count(*) FROM ingest_events WHERE resolver_variant='v9'").fetchone() == (count,)
    start = time.monotonic()
    server = boot(binary, fixture)
    try:
        startup = time.monotonic() - start
        with database(fixture) as conn:
            assert conn.execute("SELECT max(version_id) FROM goose_db_version").fetchone() == (10,)
            assert conn.execute("SELECT count(*) FROM ingest_events").fetchone() == (count,)
            assert conn.execute("SELECT receipt_id,author FROM ingest_revisions").fetchall() == [("bulk-drift", "migration")]
    finally:
        server.stop()
    return {"receipts": count, "backup_ms": round(backup * 1000, 1), "ddl_commit_ms": round((duration - repair) * 1000, 1),
            "repair_scan_and_update_ms": round(repair * 1000, 1), "bootstrap_to_health_ms": round(startup * 1000, 1),
            "backup_bytes": (next(fixture.glob("*.pre-occurrence-ingest-*.bak"))).stat().st_size}


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--binary", type=Path, required=True)
    parser.add_argument("--baseline", type=Path, required=True)
    parser.add_argument("--receipts", type=int, default=300000)
    args = parser.parse_args()
    if args.receipts < 2:
        parser.error("--receipts must be at least 2")
    with tempfile.TemporaryDirectory(prefix="tunetrail-e1-") as directory:
        root = Path(directory)
        print(json.dumps(verify_replay(args.binary.resolve(), args.baseline.resolve(), root)), flush=True)
        print(json.dumps(measure(args.binary.resolve(), args.baseline.resolve(), root, args.receipts)), flush=True)


if __name__ == "__main__":
    main()
