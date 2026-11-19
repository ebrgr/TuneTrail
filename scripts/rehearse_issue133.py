#!/usr/bin/env python3
"""Generate and rehearse the v7-to-v8 migration on synthetic data only."""

import argparse
import hashlib
import json
from pathlib import Path
import random
import shutil
import sqlite3
import statistics
import subprocess
import tempfile
import threading
import time

from validate_ingest_migration import FIXTURE_PASSWORD_HASH, check, component_hashes, copy_components
from validate_multiuser import LBZ, WEB, Server
from repro_issue133_delete_scan import delete_plans

ARTISTS, ALBUMS, TRACKS = 600, 1503, 9018
SEED = 133
IMAGE = "00000000-0000-4000-a000-000000000133"


def mbid(kind, number):
    return f"00000000-0000-4{kind:03d}-a000-{number:012d}"


def fill(conn, n):
    """Populate a real bootstrapped database with a repeatable catalog and history."""
    rng = random.Random(SEED)
    conn.executescript("""
        PRAGMA foreign_keys=ON;
        PRAGMA journal_mode=WAL;
        PRAGMA wal_autocheckpoint=0;
        UPDATE listening_profiles SET slug='public-fixture',created_at=0 WHERE id=1;
        INSERT INTO users(id,username,role,password,status) VALUES
            (2,'protected-fixture','user',x'','pending'),
            (3,'relay-fixture','user',x'','pending'),
            (4,'empty-fixture','user',x'','pending');
        INSERT INTO listening_profiles(id,slug,display_name,created_at,user_id,require_login) VALUES
            (2,'protected-fixture','Protected fixture',0,2,1),
            (3,'relay-fixture','Relay fixture',0,3,0),
            (4,'empty-fixture','Empty fixture',0,4,0);
        INSERT INTO api_keys(key,user_id,profile_id,created_at,label) VALUES
            ('fixture-api',1,1,0,'Synthetic API'),('fixture-relay',3,3,0,'Synthetic relay');
        INSERT INTO profile_relays(profile_id,url,token) VALUES(3,'http://127.0.0.1:1','fixture-only');
        INSERT INTO integrations(id,profile_id,source_type,created_at) VALUES
            ('listenbrainz:1',1,'listenbrainz',0),('file:1',1,'file',0),
            ('listenbrainz:2',2,'listenbrainz',0),('file:2',2,'file',0),
            ('listenbrainz:3',3,'listenbrainz',0),('file:3',3,'file',0),
            ('custom:1',1,'configuration',0);
    """)
    conn.execute("UPDATE users SET password=? WHERE id=1", (FIXTURE_PASSWORD_HASH.encode(),))
    conn.executemany("INSERT INTO artists(id,musicbrainz_id,image) VALUES(?,?,?)",
                     ((i, mbid(1, i) if i % 2 else None, IMAGE if i == 1 else None)
                      for i in range(1, ARTISTS + 1)))
    conn.executemany("INSERT INTO artist_aliases VALUES(?,?,?,?)",
                     ((i, label, "Canonical" if primary else "Import", primary)
                      for i in range(1, ARTISTS + 1)
                      for label, primary in [(f"Artist {i:04}", 1), (f"Alias {i:04}", 0)]))
    conn.executemany("INSERT INTO releases(id,musicbrainz_id,image) VALUES(?,?,?)",
                     ((i, mbid(2, i) if i % 2 else None, IMAGE if i == 1 else None)
                      for i in range(1, ALBUMS + 1)))
    conn.executemany("INSERT INTO artist_releases VALUES(?,?,1)",
                     (((i - 1) % ARTISTS + 1, i) for i in range(1, ALBUMS + 1)))
    conn.executemany("INSERT INTO release_aliases VALUES(?,?,? ,?)",
                     ((i, label, "Canonical" if primary else "Import", primary)
                      for i in range(1, ALBUMS + 1)
                      for label, primary in [(("Night Sessions", "Night Sessions", "Night Sessions Deluxe",
                                               "Night Sessions Remaster", "Unknown album")[i - 1]
                                              if i <= 5 else f"Album {i:04}", 1),
                                             (f"Edition {i:04}", 0)]))
    conn.executemany("INSERT INTO tracks(id,musicbrainz_id,release_id,duration) VALUES(?,?,?,?)",
                     ((i, mbid(3, i) if i % 2 and i not in (1, 7, 13, 19) else None,
                       (i - 1) // 6 + 1, (100 + i % 390)) for i in range(1, TRACKS + 1)))
    conn.executemany("INSERT INTO track_aliases VALUES(?,?,?,?)",
                     ((i, label, "Canonical" if primary else "Import", primary)
                      for i in range(1, TRACKS + 1)
                      for label, primary in [("Intro" if i % 6 == 1 else f"Track {i:05}", 1),
                                             (f"Track alias {i:05}", 0)]))
    conn.executemany("INSERT INTO artist_tracks VALUES(?,?,1)",
                     ((((i - 1) // 6) % ARTISTS + 1, i) for i in range(1, TRACKS + 1)))
    conn.executemany("INSERT INTO artist_tracks VALUES(?,?,0)",
                     ((((i - 1) // 6 + 1) % ARTISTS + 1, i) for i in range(1, TRACKS + 1, 4)))
    zones = ("+00:00", "-03:00", "+05:30", "+09:00")
    listens = [(1, 1, 1514764800, "API"),
               (2, 1, 1514764800, "File"),
               (1, 1, 1514764801, "File")]
    for i in range(3, n):
        profile = i % 3 + 1
        track = rng.randrange(1, TRACKS - 17)
        # Monotone second component avoids collisions while spanning several years.
        timestamp = 1514764802 + i * (6 * 365 * 86400 // n)
        listens.append((profile, track, timestamp, ("API", "File")[i % 2]))
    conn.executemany("INSERT INTO listens(profile_id,track_id,listened_at,client) VALUES(?,?,?,?)", listens)
    conn.executemany("""
        INSERT INTO ingest_events(
            receipt_id, profile_id, source_instance_id, schema_version, source_type, adapter_version,
            batch_id,item_ordinal,operation,received_at,listened_at,original_offset,time_precision,time_semantics,
            original_snapshot,snapshot_digest,digest_version,duplicate_decision,rule_version)
        VALUES(?,?,?,1,?,'fixture','batch',?,'submit',?,datetime(?,'unixepoch'),?,?,'start',
            ?,?,'1',?,'shadow-v2')
    """, ((f"shadow-{i:08}", profile, f"{kind}:{profile}", kind, i, ts, ts, zones[i % 4],
           "estimated" if i % 7 == 0 else "second", json.dumps({"track": track, "client": client}),
           f"fixture-{i}", ("same", "different", "review")[i % 3])
          for i, (profile, track, ts, client) in enumerate(listens)
          for kind in ["listenbrainz" if client == "API" else "file"]))
    conn.commit()


def stat(conn, sql, params=()):
    return conn.execute(sql, params).fetchone()[0]


def logical_digest(conn, table, columns):
    """Hash each ordered row using the same v7 column projection on both schemas."""
    digest = hashlib.sha256()
    query = f'SELECT {columns} FROM "{table}" ORDER BY {columns}'
    for row in conn.execute(query):
        digest.update(repr(row).encode())
        digest.update(b"\n")
    return digest.hexdigest()


def inventory(path, projection=None):
    with sqlite3.connect(path) as conn:
        tables = [row[0] for row in conn.execute(
            "SELECT name FROM sqlite_schema WHERE type='table' AND name NOT LIKE 'sqlite_%' "
            "AND name!='goose_db_version' ORDER BY name"
        )]
        if projection is None:
            projection = {name: ",".join(f'"{row[1]}"' for row in conn.execute(f'PRAGMA table_info("{name}")'))
                          for name in tables}
        return {
            "counts": {name: stat(conn, f'SELECT count(*) FROM "{name}"') for name in tables},
            "profiles": dict(conn.execute(
                "SELECT p.slug,count(l.track_id) FROM listening_profiles p "
                "LEFT JOIN listens l ON l.profile_id=p.id GROUP BY p.id"
            )),
            "digests": {name: logical_digest(conn, name, columns) for name, columns in projection.items()},
            "page_count": stat(conn, "PRAGMA page_count"),
            "free_pages": stat(conn, "PRAGMA freelist_count"),
        }, projection


def benchmark_writes(path):
    """Exercise the same legacy UPSERT on both versions inside a rolled-back transaction."""
    with sqlite3.connect(path) as conn:
        conn.execute("PRAGMA foreign_keys=ON")
        conn.execute("BEGIN IMMEDIATE")
        rows = [(1, 1451600000 + i, None, 1, "benchmark") for i in range(1000)]
        start = time.perf_counter()
        conn.executemany(
            "INSERT INTO listens(track_id,listened_at,user_id,profile_id,client) VALUES(?,?,?,?,?) "
            "ON CONFLICT(profile_id,track_id,listened_at) DO NOTHING", rows)
        elapsed = time.perf_counter() - start
        conn.rollback()
        plan = [row[3] for row in conn.execute(
            "EXPLAIN QUERY PLAN SELECT listened_at FROM listens WHERE profile_id=1 "
            "ORDER BY listened_at DESC LIMIT 25")]
        return {"ms_per_1000": round(elapsed * 1000, 3), "profile_time_plan": plan}


def benchmark_origin_lookup():
    """Isolate the origin FK using two otherwise identical in-memory tables."""
    timings = {"without_origin_fk": [], "with_origin_fk": []}
    with sqlite3.connect(":memory:") as conn:
        conn.executescript("""
            PRAGMA foreign_keys=ON;
            CREATE TABLE event_origins(origin TEXT NOT NULL PRIMARY KEY, description TEXT NOT NULL);
            INSERT INTO event_origins VALUES ('legacy','Synthetic reference');
            CREATE TABLE without_origin_fk(id INTEGER PRIMARY KEY,origin TEXT NOT NULL DEFAULT 'legacy');
            CREATE TABLE with_origin_fk(id INTEGER PRIMARY KEY,
                origin TEXT NOT NULL DEFAULT 'legacy' REFERENCES event_origins(origin));
        """)
        for index in range(8):
            for table in sorted(timings, reverse=bool(index % 2)):
                conn.execute("BEGIN")
                start = time.perf_counter()
                conn.executemany(f"INSERT INTO {table}(id) VALUES(?)", ((n,) for n in range(30000)))
                timings[table].append((time.perf_counter() - start) * 1000)
                conn.rollback()
    return {name: round(statistics.median(samples), 3) for name, samples in timings.items()}


def timed(call):
    start = time.monotonic()
    result = call()
    return result, round(time.monotonic() - start, 4)


def measured_start(server, temp_root):
    peaks = {"rss_kib": 0, "disk_bytes": 0, "wal_bytes": 0}
    finished = threading.Event()

    def sample():
        while not finished.is_set():
            paths = list(temp_root.rglob("*")) + list(server.directory.glob("tunetrail.db*"))
            size = 0
            for path in paths:
                try:
                    if path.is_file():
                        size += path.stat().st_size
                except FileNotFoundError:
                    pass
            peaks["disk_bytes"] = max(peaks["disk_bytes"], size)
            try:
                wal = (server.directory / "tunetrail.db-wal").stat().st_size
            except FileNotFoundError:
                wal = 0
            peaks["wal_bytes"] = max(peaks["wal_bytes"], wal)
            if server.process:
                rss = subprocess.run(["ps", "-o", "rss=", "-p", str(server.process.pid)],
                                     capture_output=True, text=True).stdout.strip()
                if rss.isdigit():
                    peaks["rss_kib"] = max(peaks["rss_kib"], int(rss))
            finished.wait(0.015)

    sampler = threading.Thread(target=sample)
    sampler.start()
    try:
        _, elapsed = timed(server.start)
    finally:
        finished.set()
        sampler.join()
    return {"seconds": elapsed, **peaks}


def differing_fields(left, right, prefix=""):
    """Report changed response fields without printing response payloads."""
    if type(left) is not type(right):
        return [prefix + ":type"]
    if isinstance(left, dict):
        if left.keys() != right.keys():
            return [prefix + ":keys"]
        return [change for key in left for change in differing_fields(left[key], right[key], prefix + "." + key)]
    if isinstance(left, list):
        if len(left) != len(right):
            return [prefix + ":length"]
        return [change for a, b in zip(left, right) for change in differing_fields(a, b, prefix + "[]")]
    return [] if left == right else [prefix]


def comparable(route, payload):
    """Remove only response timestamps generated at request time."""
    if route == "export":
        payload.pop("exported_at", None)
    if "range" in payload and isinstance(payload["range"], dict):
        payload["range"].pop("to", None)
    return payload


def compare_reads(server, track=1):
    server.login()
    result = {}
    for slug in ("public-fixture", "protected-fixture", "relay-fixture", "empty-fixture"):
        prefix = f"{WEB}/profiles/{slug}"
        routes = ["stats?period=all_time", "listens?page=1", "listens?page=2",
                  "top/artists", "top/albums", "top/tracks", "search?q=Intro", "export"]
        if slug == "public-fixture":
            routes += [f"track/{track}", f"track/{track}/aliases"]
        for route in routes:
            result[f"{slug}/{route}"] = comparable(route, server.request("GET", f"{prefix}/{route}", admin=True))
    return result


def exercise_writes(server):
    server.login()
    prefix = f"{WEB}/profiles/public-fixture"
    details = {"track_id": 1, "unix": 1451606400, "client": "rehearsal"}
    operations = [
        ("insert", lambda: server.request("POST", prefix + "/listens", details, admin=True, expected=201)),
        ("repeat", lambda: server.request("POST", prefix + "/listens", details, admin=True, expected=201)),
        ("delete_listen", lambda: server.request("DELETE", prefix + "/listens?track_id=1&unix=1451606400",
                                                admin=True, expected=204)),
    ]
    timings = {}
    for name, call in operations:
        _, timings[name] = timed(call)
        with sqlite3.connect(server.directory / "tunetrail.db") as conn:
            assert stat(conn, "SELECT count(*) FROM listens WHERE profile_id=1 "
                             "AND track_id=1 AND listened_at=1451606400") == (name != "delete_listen")
    payload = {"listen_type": "single", "payload": [{"listened_at": 1451606401, "track_metadata": {
        "artist_name": "Artist 0001", "track_name": "Intro", "release_name": "Night Sessions"}}]}
    _, timings["shadow_api"] = timed(lambda: server.request(
        "POST", LBZ + "/submit-listens", payload, token="fixture-api"))
    _, timings["merge_track"] = timed(lambda: server.request(
        "POST", WEB + "/track/1/merge", {"merge_from_id": TRACKS - 7}, admin=True, expected=204))
    _, timings["delete_release"] = timed(lambda: server.request(
        "DELETE", f"{WEB}/album/{ALBUMS-1}", admin=True, expected=204))
    path = server.directory / "tunetrail.db"
    with sqlite3.connect(path) as conn:
        assert stat(conn, "SELECT count(*) FROM ingest_events WHERE rule_version LIKE 'shadow-%' "
                          "AND received_at>1500000000") > 0
        removed_count = stat(conn, "SELECT count(*) FROM listens WHERE profile_id=2")
    _, timings["delete_profile"] = timed(lambda: server.request(
        "DELETE", WEB + "/profiles/protected-fixture", {"display_name": "Protected fixture",
                                                        "listen_count": removed_count},
        admin=True, expected=204))
    with sqlite3.connect(path) as conn:
        assert stat(conn, "SELECT count(*) FROM listens WHERE profile_id=2") == 0
        assert stat(conn, "SELECT count(*) FROM ingest_events WHERE profile_id=2") == 0
    _, timings["purge"] = timed(lambda: server.request("DELETE", WEB + "/data", admin=True, expected=204))
    with sqlite3.connect(path) as conn:
        for table in ("listens", "ingest_events", "recordings", "occurrences"):
            assert stat(conn, f"SELECT count(*) FROM {table}") == 0
    return timings


def rehearse(root, args):
    source_dir = root / "source"
    source_server = Server(args.v7, source_dir)
    try:
        source_server.start()
    finally:
        source_server.stop()
    source = source_dir / "tunetrail.db"
    check(source, 7)
    with sqlite3.connect(source) as writer:
        fill(writer, args.listens)
        # Apply a real v7 merge and release deletion before freezing the fixture.
        try:
            source_server.start()
            source_server.login()
            source_server.request("POST", WEB + "/track/1/merge", {"merge_from_id": TRACKS},
                                  admin=True, expected=204)
            source_server.request("DELETE", f"{WEB}/album/{ALBUMS}", admin=True, expected=204)
        finally:
            source_server.stop()

        before = component_hashes(source)
        assert {"", "-wal", "-shm"} <= before.keys()
        assert shutil.disk_usage(root).free >= 3 * sum(
            (source_dir / f"tunetrail.db{suffix}").stat().st_size for suffix in before)
        dirs = {}
        for name in ("baseline", "upgrade"):
            dirs[name] = root / name
            dirs[name].mkdir()
            target = dirs[name] / "tunetrail.db"
            copy_components(source, target)
            assert component_hashes(target) == before

        pre, projection = inventory(dirs["baseline"] / "tunetrail.db")
        baseline = Server(args.v7, dirs["baseline"])
        try:
            baseline.start()
            before_reads = compare_reads(baseline)
        finally:
            baseline.stop()

        rollback = None
        if args.fault_probe:
            failure_dir = root / "failure"
            failure_dir.mkdir()
            failed_db = failure_dir / "tunetrail.db"
            copy_components(source, failed_db)
            _, rollback = timed(lambda: subprocess.run([args.fault_probe, failed_db], check=True))
            check(failed_db, 7)
            assert inventory(failed_db, projection)[0]["digests"] == pre["digests"]
            # Reuse this rolled-back copy for the real bootstrap retry and its backup checks.
            dirs["upgrade"] = failure_dir

        SQLite_tmp = root / "sqlite-temp"
        SQLite_tmp.mkdir()
        upgrade = Server(args.v8, dirs["upgrade"],
                         environment={"TMPDIR": str(SQLite_tmp), "SQLITE_TMPDIR": str(SQLite_tmp)})
        path = dirs["upgrade"] / "tunetrail.db"
        try:
            measured = measured_start(upgrade, SQLite_tmp)
            post, _ = inventory(path, projection)
            after_reads = compare_reads(upgrade)
            mismatches = [key for key in before_reads if before_reads[key] != after_reads[key]]
            assert not mismatches, {key: sorted(set(differing_fields(before_reads[key], after_reads[key])))
                                    for key in mismatches}
        finally:
            upgrade.stop()
        check(path, 8)
        assert pre["digests"] == post["digests"]
        assert pre["profiles"] == post["profiles"]
        assert post["counts"]["listens"] == args.listens
        with sqlite3.connect(path) as conn:
            assert stat(conn, "SELECT count(*) FROM tracks t JOIN occurrences o "
                              "ON o.legacy_track_id=t.id AND o.id=t.id") == pre["counts"]["tracks"]
            assert stat(conn, "SELECT count(DISTINCT recording_id) FROM occurrences") == pre["counts"]["tracks"]
            assert stat(conn, "SELECT count(*) FROM listens WHERE length(event_id)=32 AND origin='legacy'") == args.listens
        (backup,) = dirs["upgrade"].glob("tunetrail.db.pre-recording-identity-*.bak")
        check(backup, 7)
        assert inventory(backup, projection)[0]["digests"] == pre["digests"]
        assert component_hashes(source) == before

        restored = root / "restored"
        restored.mkdir()
        restored_db = restored / "tunetrail.db"
        shutil.copyfile(backup, restored_db)
        restore = Server(args.v7, restored)
        try:
            restore.start()
        finally:
            restore.stop()
        assert inventory(restored_db, projection)[0]["digests"] == pre["digests"]
        stable, _ = inventory(path)
        try:
            upgrade.start()
        finally:
            upgrade.stop()
        assert inventory(path)[0]["digests"] == stable["digests"]
        for binary in (args.v7, args.v6):
            refusal = Server(binary, dirs["upgrade"])
            logfile = refusal.directory / "server.log"
            offset = logfile.stat().st_size
            try:
                refusal.start()
                raise AssertionError("Older binary accepted schema v8")
            except RuntimeError:
                assert "unsupported schema version 8" in logfile.read_text()[offset:]
            finally:
                refusal.stop()
        assert inventory(path)[0]["digests"] == stable["digests"]
        benchmarks = {label: benchmark_writes(database) for label, database in
                      [("v7", dirs["baseline"] / "tunetrail.db"), ("v8", path)]}
        with sqlite3.connect(path) as conn:
            conn.execute("PRAGMA foreign_keys=ON")
            plans = delete_plans(conn)
        assert all("SCAN ingest_events" not in plan for plan in plans.values())
        alternative = None
        if args.alternative:
            variant = root / "variant"
            variant.mkdir()
            copy_components(source, variant / "tunetrail.db")
            variant_server = Server(args.alternative, variant,
                                    environment={"TMPDIR": str(SQLite_tmp), "SQLITE_TMPDIR": str(SQLite_tmp)})
            try:
                alternative = measured_start(variant_server, SQLite_tmp)
            finally:
                variant_server.stop()
            check(variant / "tunetrail.db", 8)
            assert inventory(variant / "tunetrail.db", projection)[0]["digests"] == pre["digests"]
        mutations = root / "mutations"
        mutations.mkdir()
        copy_components(path, mutations / "tunetrail.db")
        mutable = Server(args.v8, mutations)
        try:
            mutable.start()
            writes = exercise_writes(mutable)
        finally:
            mutable.stop()
        check(mutations / "tunetrail.db", 8)
        return {"seed": SEED, "source_hashes": {key or "db": value.hex() for key, value in before.items()},
                "source_components_unchanged": True, "before": pre, "after": post,
                "db_bytes": path.stat().st_size, "backup_bytes": backup.stat().st_size,
                "bootstrap": measured, "alternative": alternative, "rollback_seconds": rollback, "compared_http_routes": len(before_reads),
                "legacy_write_seconds": writes, "upsert": benchmarks, "delete_plans": plans,
                "origin_fk_ms_per_30000": benchmark_origin_lookup()}


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    for name in ("v6", "v7", "v8"):
        parser.add_argument(f"--{name}", type=Path, required=True)
    parser.add_argument("--fault-probe", type=Path, help="Optional issue133_fault binary for a late 008 failure")
    parser.add_argument("--alternative", type=Path, help="Optional test binary without the TEMP listens copy")
    parser.add_argument("--listens", type=int, default=30000)
    args = parser.parse_args()
    if args.listens < 30000:
        parser.error("--listens must be at least 30000")
    for name in ("v6", "v7", "v8"):
        setattr(args, name, getattr(args, name).resolve(strict=True))
    if args.fault_probe:
        args.fault_probe = args.fault_probe.resolve(strict=True)
    if args.alternative:
        args.alternative = args.alternative.resolve(strict=True)
    with tempfile.TemporaryDirectory(prefix="tt133-rehearsal-", dir="/tmp") as tmp:
        print(json.dumps(rehearse(Path(tmp), args), indent=2, sort_keys=True))


if __name__ == "__main__":
    main()
