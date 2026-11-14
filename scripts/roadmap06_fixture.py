#!/usr/bin/env python3
"""Create only new, fictional SQLite v3 comparison fixtures in a new directory."""

import argparse
from contextlib import closing
import hashlib
import json
import sqlite3
from datetime import datetime, timezone
from pathlib import Path
from zoneinfo import ZoneInfo

from roadmap05_fixture import BASE, IMAGE, ROOT, build_fixture

FIXTURE_VERSION = 1
REFERENCE_NOW = "2025-06-01T12:00:00Z"  # Documentation only; not injected into the server.
LONG_NAME = "長い曲名 — " + "Long fictional title " * 8
# Fixed bcrypt hash for the public fictional password "validation-password".
FIXTURE_PASSWORD_HASH = "$2a$10$8v5K53tXJgGKfBra.QNXmutgDII0KmVr62LAVwvD7Je9Ia2f0g8J."
TRANSITIONS = [
    ("2018-11-04T02:59:00Z", "2018-11-03 23:59 -0300"),
    ("2018-11-04T03:01:00Z", "2018-11-04 01:01 -0200"),
    ("2019-02-17T01:30:00Z", "2019-02-16 23:30 -0200"),
    ("2019-02-17T02:30:00Z", "2019-02-16 23:30 -0300"),
]


def timestamp(value):
    """Parse an explicit UTC fixture instant without relying on the host timezone."""
    return int(datetime.fromisoformat(value.replace("Z", "+00:00")).timestamp())


def counts(conn):
    """Return the number of listens for every profile, including empty profiles."""
    return dict(conn.execute("SELECT p.slug,count(l.track_id) FROM listening_profiles p LEFT JOIN listens l ON l.profile_id=p.id GROUP BY p.id ORDER BY p.id"))


def check_database(path, expected):
    """Check schema, literal profile counts and integrity on a disposable database."""
    with closing(sqlite3.connect(path)) as conn:
        conn.execute("PRAGMA foreign_keys=ON")
        actual = counts(conn)
        if actual != expected:
            raise AssertionError((actual, expected))
        if conn.execute("SELECT MAX(version_id) FROM goose_db_version WHERE is_applied=1").fetchone() != (3,):
            raise AssertionError("fixture schema is not v3")
        if conn.execute("PRAGMA integrity_check").fetchall() != [("ok",)] or conn.execute("PRAGMA foreign_key_check").fetchall():
            raise AssertionError("fixture integrity failed")
    return actual


def backup(source, destination):
    """Use SQLite backup so each new destination contains an independent snapshot."""
    if destination.exists():
        raise FileExistsError(destination)
    with closing(sqlite3.connect(f"{source.resolve().as_uri()}?mode=ro", uri=True)) as src, closing(sqlite3.connect(destination)) as dst:
        src.backup(dst)


def create(output):
    """Create the named fixture set, refusing any existing destination directory."""
    output.mkdir(parents=False, exist_ok=False, mode=0o700)
    canonical = output / "base.sqlite3"
    with closing(sqlite3.connect(canonical)) as conn:
        build_fixture(conn)
        # The legacy trigger creates a profile for a new user; remove only that
        # synthetic profile so server startup cannot add a fourth profile.
        conn.execute("INSERT INTO users(id,username,role,password) VALUES(1,'validation-admin','admin',?)", (FIXTURE_PASSWORD_HASH.encode("ascii"),))
        conn.execute("DELETE FROM listening_profiles WHERE id=(SELECT profile_id FROM legacy_user_profiles WHERE user_id=1)")
        conn.execute("UPDATE track_aliases SET alias=? WHERE track_id=4 AND is_primary=1", (LONG_NAME,))
        conn.execute("UPDATE releases SET image='22222222-2222-4222-8222-222222222222' WHERE id=2")
        conn.executemany("INSERT INTO listens(profile_id,track_id,listened_at,client) VALUES(1,1,?,'fixture-dst')", [(timestamp(value),) for value, _ in TRANSITIONS])
        conn.commit()
    expected = {"alpha": 7, "beta": 2, "empty": 0}
    check_database(canonical, expected)
    with closing(sqlite3.connect(canonical)) as conn:
        for value, local in TRANSITIONS:
            instant = timestamp(value)
            if conn.execute("SELECT COUNT(*) FROM listens WHERE profile_id=1 AND track_id=1 AND listened_at=?", (instant,)).fetchone() != (1,):
                raise AssertionError("DST fixture instant missing")
            if datetime.fromtimestamp(instant, ZoneInfo("America/Sao_Paulo")).strftime("%Y-%m-%d %H:%M %z") != local:
                raise AssertionError("unexpected Sao Paulo timezone data")
        if conn.execute("SELECT COUNT(*) FROM listens WHERE track_id=1 AND listened_at=?", (BASE,)).fetchone() != (2,):
            raise AssertionError("shared same-second fixture missing")
        if conn.execute("SELECT COUNT(*) FROM listens WHERE track_id=3").fetchone() != (0,):
            raise AssertionError("unplayed track fixture changed")
        if conn.execute("SELECT alias FROM track_aliases WHERE track_id=4 AND is_primary=1").fetchone() != (LONG_NAME,):
            raise AssertionError("long Unicode fixture missing")
    for name in ("go", "react", "mutation"):
        folder = output / name
        folder.mkdir(mode=0o700)
        backup(canonical, folder / "koito.db")
        cache = folder / "image_cache" / IMAGE[:2] / IMAGE
        cache.mkdir(parents=True, mode=0o700)
        (cache / "source.webp").write_bytes((ROOT / "assets/preview/avatar.webp").read_bytes())
        check_database(folder / "koito.db", expected)
    for name, remove, want in (("zero", (1, 2, 3), {}), ("one", (2, 3), {"alpha": 7})):
        folder = output / name
        folder.mkdir(mode=0o700)
        backup(canonical, folder / "koito.db")
        with closing(sqlite3.connect(folder / "koito.db")) as conn:
            conn.execute("PRAGMA foreign_keys=ON")
            conn.executemany("DELETE FROM listening_profiles WHERE id=?", [(x,) for x in remove])
            conn.commit()
        check_database(folder / "koito.db", want)
    folder = output / "large"
    folder.mkdir(mode=0o700)
    backup(canonical, folder / "koito.db")
    with closing(sqlite3.connect(folder / "koito.db")) as conn:
        conn.executemany("INSERT INTO listens(profile_id,track_id,listened_at,client) VALUES(1,?,?,'fixture-large')", ((1 if i % 2 == 0 else 4, BASE + 100 + i // 2) for i in range(1002)))
        conn.commit()
        ordered = conn.execute("SELECT listened_at,track_id FROM listens WHERE profile_id=1 ORDER BY listened_at,track_id").fetchall()
        if len(ordered) != 1009 or ordered[999][0] != ordered[1000][0]:
            raise AssertionError("large export boundary no longer crosses a tie")
    check_database(folder / "koito.db", {"alpha": 1009, "beta": 2, "empty": 0})
    digest = hashlib.sha256(canonical.read_bytes()).hexdigest()
    report = {"fixture_version": FIXTURE_VERSION, "schema_version": 3, "reference_now_not_injected": REFERENCE_NOW,
              "timezone": "America/Sao_Paulo", "sqlite_version": sqlite3.sqlite_version,
              "base_counts": expected, "zero_counts": {}, "one_counts": {"alpha": 7},
              "large_counts": {"alpha": 1009, "beta": 2, "empty": 0},
              "dst_utc_local": TRANSITIONS, "base_sha256": digest,
              "images": "one fictional local shared WebP; beta album points to a deliberately missing image",
              "copies": ["go/koito.db", "react/koito.db", "mutation/koito.db", "zero/koito.db", "one/koito.db", "large/koito.db"],
              "credentials": "fictional administrator and keys; never use this fixture in production"}
    (output / "fixture-report.json").write_text(json.dumps(report, indent=2, ensure_ascii=False) + "\n")
    return report


def main():
    """Write a new disposable fixture directory chosen explicitly by the operator."""
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--output-dir", type=Path, required=True, help="new directory inside a disposable workspace")
    args = parser.parse_args()
    print(json.dumps(create(args.output_dir), indent=2, ensure_ascii=False))


if __name__ == "__main__":
    main()
