#!/usr/bin/env python3
"""Replay shadow dedup on a synthetic database and optionally a disposable real copy."""

import argparse
import json
from pathlib import Path
import subprocess
import tempfile

from validate_ingest_migration import ROOT, check, component_hashes, connect, copy_components, digests, profile_counts


# Schema version the shadow receiver leaves behind after bootstrap.
LATEST_SCHEMA = 8


def run(mode, directory):
    """Pass only a temporary path to the Go receiver and return its aggregates."""
    (Path(directory) / ".shadow-replay-copy").touch(exist_ok=False)
    proc = subprocess.run(
        ["go", "run", "./scripts/ingest_shadow_replay", mode, directory],
        cwd=ROOT, capture_output=True, text=True, check=True,
    )
    return json.loads(proc.stdout)


def synthetic():
    """Verify cross-source ambiguity at 0–2 s, and preserve a 3 s negative control."""
    with tempfile.TemporaryDirectory(prefix="tunetrail-shadow-synthetic-") as tmp:
        report = run("synthetic", tmp)
        expected = {
            "cross_source_0s": 1, "cross_source_1s": 1, "cross_source_2s": 1,
            "same_source_same_second": 1, "cross_source_3s_control": 0,
        }
        assert set(report["groups"]) == set(expected)
        for group, reviews in expected.items():
            assert report["groups"][group] == {
                "different": 2 - reviews, "review": reviews, "same": 0, "skipped": 0,
            }
        database = Path(tmp) / "tunetrail.db"
        assert profile_counts(database) == {}
        check(database, LATEST_SCHEMA)
        return report


def historical(source):
    """Hash source components in memory and never pass the source to SQLite."""
    before = component_hashes(source)
    with tempfile.TemporaryDirectory(prefix="tunetrail-shadow-history-") as tmp:
        database = Path(tmp) / "tunetrail.db"
        copy_components(source, database)
        if component_hashes(source) != before or component_hashes(database) != before:
            raise RuntimeError("source or sidecars changed during copy")
        with connect(database) as conn:
            schema = conn.execute(
                "SELECT max(version_id) FROM goose_db_version WHERE is_applied=1"
            ).fetchone()[0]
            if schema < 3:
                raise ValueError("the historical replay expects schema 3 or later")
        check(database, schema)
        counts = profile_counts(database)
        listens = digests(database)["listens"]
        report = run("copy", tmp)
        assert {profile: sum(group.values()) for profile, group in report["groups"].items() if sum(group.values())} == {
            str(profile): count for profile, count in counts.items()
        }
        assert profile_counts(database) == counts
        assert digests(database)["listens"] == listens
        check(database, LATEST_SCHEMA)
    assert component_hashes(source) == before
    return {"input_schema": schema, "listen_counts": counts, "source_components_unchanged": True,
            "legacy_listens_unchanged": True, **report}


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--source", type=Path, help="copy this quiescent SQLite file and its sidecars first")
    args = parser.parse_args()
    report = {"synthetic": synthetic()}
    if args.source is not None:
        report["historical"] = historical(args.source.resolve())
    print(json.dumps(report, sort_keys=True))


if __name__ == "__main__":
    main()
