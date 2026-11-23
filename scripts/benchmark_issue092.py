#!/usr/bin/env python3
"""Measure v0.47 reads and an experimental index on disposable synthetic history."""

import argparse
from datetime import datetime, timezone
import json
from pathlib import Path
import platform
import re
import sqlite3
import tempfile
import time

from rehearsal_support import SUBMIT, database, percentiles, sample_resources, wait_for
from rehearse_issue133 import fill
from validate_multiuser import ROOT, WEB, Server, scrobble

SQLITE = ROOT / "internal/db/sqlite"
START = int(datetime(2023, 1, 1, tzinfo=timezone.utc).timestamp())
END = int(datetime(2024, 1, 1, tzinfo=timezone.utc).timestamp()) - 1
INDEX = "issue092_profile_time_track"
ENV = {"TUNETRAIL_FORCE_TZ": "UTC", "TZ": "UTC", "TUNETRAIL_LOG_LEVEL": "error"}


def sql_literal(file, first, last, *, ends=False):
    """Read the exact SQL literal bracketed by two fragments in the checkout."""
    candidates = re.findall(r"`([^`]+)`", (SQLITE / file).read_text())
    return next(query.strip() for query in reversed(candidates)
                if first in query and (query.strip().endswith(last) if ends else last in query))


def scoped(query):
    """Apply the production read scope for the public synthetic profile."""
    scope = "WITH listens AS (SELECT * FROM main.listens WHERE profile_id = 1)"
    return scope + (", " + query[5:] if query.startswith("WITH ") else " " + query)


def queries():
    """Build plans for representative statements exercised by the HTTP scenarios."""
    bounds = (START, END)
    base = {
        "plays": (sql_literal("counts.go", "SELECT COUNT(*) FROM listens WHERE", "BETWEEN"), bounds),
        "minutes": (sql_literal("counts.go", "SUM(t.duration)", "WHERE l.listened_at BETWEEN ? AND ?", ends=True), bounds),
        "unique_artists": (sql_literal("artist.go", "COUNT(DISTINCT at2.artist_id)", "BETWEEN"), bounds),
        "active_days": (sql_literal("listen_activity.go", "DISTINCT listened_at", "listened_at < ?"), (END + 1,)),
        "longest_streak": (sql_literal("listen_activity.go", "GROUP BY hour_bucket ORDER BY", "listened_at >= ?"), bounds),
        "activity": (sql_literal("listen_activity.go", "COUNT(*) AS listen_count", "GROUP BY hour_bucket"), bounds),
        "top_tracks": (sql_literal("track.go", "WITH TrackCounts", "GROUP BY track_id"), (*bounds, 25, 0)),
        "top_albums": (sql_literal("album.go", "WITH AlbumCounts", "GROUP BY t.release_id"), (*bounds, 25, 0)),
        "top_artists": (sql_literal("artist.go", "WITH ArtistCounts", "GROUP BY at2.artist_id"), (*bounds, 25, 0)),
        "history_page_400": (sql_literal("listen.go", "COALESCE(l.client", "ORDER BY l.listened_at DESC"), (*bounds, 25, 9975)),
        "new_tracks": (sql_literal("track.go", "MIN(listened_at)", "GROUP BY track_id"), bounds),
        "profile_counts": ("SELECT profile_id, count(*) FROM listens GROUP BY profile_id", ()),
    }
    # Compare only the aggregation portion when replacing the catalog join.
    base["release_legacy"] = ("""SELECT t.release_id, count(*) FROM listens l
        JOIN tracks t ON t.id=l.track_id WHERE l.listened_at BETWEEN ? AND ?
        GROUP BY t.release_id ORDER BY count(*) DESC, t.release_id""", bounds)
    base["release_occurrence"] = ("""SELECT o.release_id, count(*) FROM listens l
        JOIN occurrences o ON o.id=l.track_id WHERE l.listened_at BETWEEN ? AND ?
        GROUP BY o.release_id ORDER BY count(*) DESC, o.release_id""", bounds)
    base["recording_occurrence"] = ("""SELECT o.recording_id, count(*) FROM listens l
        JOIN occurrences o ON o.id=l.track_id WHERE l.listened_at BETWEEN ? AND ?
        GROUP BY o.recording_id ORDER BY count(*) DESC, o.recording_id""", bounds)
    # Active days in /stats has a historical upper bound independent of the selected year.
    base["active_days"] = (base["active_days"][0], ((int(time.time()) // 86400 + 1) * 86400,))
    return {name: (query if name == "profile_counts" else scoped(query), args)
            for name, (query, args) in base.items()}


def measure(call, repeats):
    """Warm the workload and time complete result consumption."""
    for _ in range(3):
        call()
    samples = []
    for _ in range(repeats):
        start = time.perf_counter()
        call()
        samples.append(time.perf_counter() - start)
    return percentiles(samples)


def read_workload(server, repeats):
    """Read real HTTP endpoints and record resource peaks for the phase."""
    prefix = WEB + "/profiles/bench-fixture"
    routes = {
        "stats_year": prefix + "/stats?year=2023",
        "stats_all": prefix + "/stats?period=all_time",
        "top_tracks": prefix + "/top/tracks?year=2023&limit=25",
        "top_albums": prefix + "/top/albums?year=2023&limit=25",
        "top_artists": prefix + "/top/artists?year=2023&limit=25",
        "history_first": prefix + "/listens?year=2023&limit=25&page=1",
        "history_page_400": prefix + "/listens?year=2023&limit=25&page=400",
        "heatmap_data_recent": prefix + "/listen-activity?range=252&step=day",
        "heatmap_data_year": prefix + "/listen-activity?year=2023&step=day",
        "rewind_api": prefix + "/summary?year=2023",
        "rewind_html": "/bench-fixture/rewind?year=2023",
        "home_all": "/bench-fixture?period=all_time",
        "home_week": "/bench-fixture?period=week",
    }
    with sample_resources(server) as peaks:
        latency = {}
        for name, route in routes.items():
            latency[name] = measure(lambda: server.request("GET", route), repeats)
            print(f"  {name}: p50 {latency[name]['p50_ms']} ms", flush=True)
    return {"latency": latency, "peaks": peaks}


def write_workload(server, directory, start, repeats):
    """Measure serial live legacy submissions and verify every write landed."""
    def send(offset):
        server.request("POST", SUBMIT, scrobble(start + offset * 4), token="fixture-api")
    with sample_resources(server) as peaks:
        samples = []
        for i in range(repeats + 3):
            before = time.perf_counter()
            send(i)
            if i >= 3:
                samples.append(time.perf_counter() - before)
    with database(directory) as conn:
        actual = conn.execute("SELECT count(*) FROM listens WHERE profile_id=1 AND listened_at BETWEEN ? AND ?",
                              (start, start + (repeats + 2) * 4)).fetchone()[0]
        assert actual == repeats + 3, (actual, repeats + 3)
    return {"latency": percentiles(samples), "peaks": peaks, "written": actual}


def benchmark(args, directory):
    """Generate the ISSUE-133 fixture, migrate it, and isolate the index experiment."""
    baseline = Server(args.baseline, directory, environment=ENV)
    try:
        baseline.start()
    finally:
        baseline.stop()
    with database(directory) as conn:
        fill(conn, args.listens)
        conn.execute("UPDATE listening_profiles SET slug='bench-fixture' WHERE id=1")
        # Add traffic to both rolling Home windows without changing fixture size.
        latest = conn.execute("SELECT max(listened_at) FROM listens").fetchone()[0]
        shift = int(time.time()) - 86400 - latest
        conn.execute("UPDATE listens SET listened_at=listened_at+? WHERE listened_at>?",
                     (shift, latest - 14 * 86400))
        conn.commit()
        conn.execute("PRAGMA wal_checkpoint(TRUNCATE)")
    server = Server(args.binary, directory, environment=ENV)
    result = {"machine": platform.platform(), "sqlite_explain": sqlite3.sqlite_version,
              "samples": {"http": args.repeats, "sql": args.repeats, "ingest": args.writes}, "phases": {}}
    try:
        try:
            server.start()
        except RuntimeError:
            # The stock fixture helper times out after 30 seconds on large migrations.
            if server.process is None or server.process.poll() is not None:
                raise
            def ready():
                if server.process.poll() is not None:
                    raise RuntimeError("migration server exited")
                try:
                    server.request("GET", WEB + "/health")
                    return True
                except OSError:
                    return False
            wait_for(ready, timeout=300)
        with database(directory) as conn:
            assert conn.execute("SELECT max(version_id) FROM goose_db_version WHERE is_applied=1").fetchone() == (11,)
            result["profile_counts"] = dict(conn.execute("""SELECT p.id, count(l.event_id) FROM listening_profiles p
                LEFT JOIN listens l ON l.profile_id=p.id GROUP BY p.id"""))
            assert sum(result["profile_counts"].values()) == args.listens
            assert result["profile_counts"][4] == 0
        stamp = int(time.time()) - 30 * 86400
        for name in ("baseline", "indexed"):
            with database(directory) as conn:
                if name == "indexed":
                    conn.execute(f"CREATE INDEX {INDEX} ON listens(profile_id,listened_at,track_id)")
                    conn.commit()
                conn.execute("PRAGMA wal_checkpoint(TRUNCATE)")
                disk = conn.execute("PRAGMA page_count").fetchone()[0] * conn.execute("PRAGMA page_size").fetchone()[0]
                statements = {}
                for label, (query, params) in queries().items():
                    plan = [row[3] for row in conn.execute("EXPLAIN QUERY PLAN " + query, params)]
                    rows = conn.execute(query, params).fetchall()
                    statements[label] = {"plan": plan, "rows": len(rows),
                                         **measure(lambda: conn.execute(query, params).fetchall(), args.repeats)}
                    if label == "release_legacy":
                        legacy = rows
                    if label == "release_occurrence":
                        assert rows == legacy
            print(f"{args.listens}: {name} SQL complete", flush=True)
            result["phases"][name] = {"db_bytes": disk, "sql": statements,
                                      "http": read_workload(server, args.repeats),
                                      "ingest": write_workload(server, directory, stamp, args.writes)}
            stamp -= 7 * 86400
            print(f"{args.listens}: {name} complete", flush=True)
    finally:
        server.stop()
    return result


def main():
    """Run one volume per process and emit a portable JSON result."""
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--binary", type=lambda v: Path(v).resolve(), required=True, help="v0.47 binary")
    parser.add_argument("--baseline", type=lambda v: Path(v).resolve(), required=True, help="v0.42 binary")
    parser.add_argument("--listens", type=int, choices=(300_000, 1_000_000), required=True)
    parser.add_argument("--repeats", type=int, help="samples per read (default: 100 at 300k, 40 at 1M)")
    parser.add_argument("--writes", type=int, default=100)
    parser.add_argument("--output", type=Path, required=True)
    args = parser.parse_args()
    if args.repeats is None:
        args.repeats = 100 if args.listens == 300_000 else 40
    if args.repeats < 1 or args.writes < 1 or args.writes > 1000:
        parser.error("require positive sample counts and at most 1000 writes")
    with tempfile.TemporaryDirectory(prefix="tunetrail-092-") as directory:
        result = benchmark(args, Path(directory))
    args.output.write_text(json.dumps(result, indent=2) + "\n")


if __name__ == "__main__":
    main()
