#!/usr/bin/env python3
"""Repeat ISSUE-133's HTTP comparison across schema v13/v14 and the writer cut."""

import argparse
import hashlib
import json
from pathlib import Path
import sqlite3
import tempfile

from rehearse_issue133 import comparable, differing_fields, fill
from validate_multiuser import Server, WEB


PROFILES = ("public-fixture", "protected-fixture", "relay-fixture", "empty-fixture")
ROUTES = (
    "stats?period=all_time", "listens?page=1", "listens?page=2", "top/artists",
    "top/albums", "top/tracks", "top/tracks?group_by=release", "search?q=Intro", "export",
)


def canonical(value):
    return json.dumps(value, ensure_ascii=False, sort_keys=True, separators=(",", ":"))


def normalize_archive(value):
    """Koito v1 aliases are a set and archive order changed to event ID order."""
    for listen in value["listens"]:
        for entity in [listen["track"], listen["album"], *(listen["artists"] or [])]:
            entity["aliases"] = sorted(entity["aliases"] or [], key=canonical)
    value["listens"].sort(key=canonical)
    return value


def read(server, legacy):
    result = {}
    server.login()
    for slug in PROFILES:
        routes = ROUTES
        if slug == "public-fixture":
            routes += ("track/1", "track/1/aliases", "album/1")
        for route in routes:
            suffix = "export?version=1" if route == "export" and not legacy else route
            response = server.request("GET", f"{WEB}/profiles/{slug}/{suffix}", admin=True)
            value = comparable(route, response)
            result[f"{slug}/{route}"] = normalize_archive(value) if route == "export" else value
    return result


def check_schema(db, version):
    with sqlite3.connect(db) as conn:
        assert conn.execute("SELECT max(version_id) FROM goose_db_version WHERE is_applied=1").fetchone() == (version,)
        assert conn.execute("PRAGMA foreign_key_check").fetchall() == []


def run(baseline, current, listens):
    with tempfile.TemporaryDirectory(prefix="tunetrail-issue-183-") as tmp:
        root = Path(tmp)
        db = root / "tunetrail.db"
        old = Server(baseline, root)
        new = Server(current, root)
        try:
            old.start()
            check_schema(db, 13)
            with sqlite3.connect(db) as conn:
                fill(conn, listens)
            before = read(old, legacy=True)
        finally:
            old.stop()
        try:
            new.start()
            check_schema(db, 14)
            upgraded = read(new, legacy=False)
            with sqlite3.connect(db) as conn:
                # Remove synthetic unresolved shadow receipts before the guarded cut.
                conn.execute("DELETE FROM ingest_events WHERE profile_id IN (1,2) AND mode='shadow'")
                conn.execute("""
                    INSERT INTO profile_ingest_flags(profile_id,active,occurrence_write)
                    VALUES(1,1,1),(2,1,1)
                    ON CONFLICT(profile_id) DO UPDATE SET active=1,occurrence_write=1
                """)
            converted = read(new, legacy=False)
        finally:
            new.stop()
        for label, result in [("upgrade", upgraded), ("occurrence writer cut", converted)]:
            differences = differing_fields(before, result)
            if differences:
                raise AssertionError(f"{label}: {differences[:25]}")
        return {
            "source_schema": 13,
            "target_schema": 14,
            "listens": listens,
            "read_responses_per_pass": len(before),
            "activated_profiles": [1, 2],
            "normalized_sha256": hashlib.sha256(canonical(before).encode()).hexdigest(),
            "differences": [],
        }


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--baseline", type=Path, required=True)
    parser.add_argument("--current", type=Path, required=True)
    parser.add_argument("--listens", type=int, default=3000)
    args = parser.parse_args()
    print(json.dumps(run(args.baseline.resolve(), args.current.resolve(), args.listens), indent=2))
