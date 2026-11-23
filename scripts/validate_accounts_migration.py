#!/usr/bin/env python3
"""Verify account migration and restoration using only disposable component copies.

Sources are never opened by SQLite. Reports contain counts, never credentials.
"""
import argparse
import hashlib
import json
from pathlib import Path
import sqlite3
import tempfile

from validate_ingest_migration import (check, component_hashes, connect,
                                       copy_components, profile_counts)
from validate_multiuser import LATEST_SCHEMA_VERSION, Server


def inventory(path):
    """Keep original rows and column projections private for exact preservation checks."""
    with connect(path) as conn:
        result = {}
        for (table,) in conn.execute("SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%' AND name NOT IN ('goose_db_version','legacy_user_profiles')"):
            columns = [r[1] for r in conn.execute(f'PRAGMA table_info("{table}")')]
            rows = conn.execute(f'SELECT * FROM "{table}"').fetchall()
            ids = [r[columns.index('id')] for r in rows] if table in ('users', 'listening_profiles') else None
            result[table] = (columns, ids, hashlib.sha256(repr(sorted(map(repr, rows))).encode()).digest())
        return result


def preserved(path, before):
    """Compare original rows while allowing only newly allocated account/profile pairs."""
    with connect(path) as conn:
        for table, (columns, ids, digest) in before.items():
            query = 'SELECT ' + ','.join(f'"{c}"' for c in columns) + f' FROM "{table}"'
            if ids is not None:
                query += ' WHERE id IN (' + ','.join('?' for _ in ids) + ')'
            rows = conn.execute(query, ids or ()).fetchall()
            assert hashlib.sha256(repr(sorted(map(repr, rows))).encode()).digest() == digest, f'preservation failed: {table}'


def accounts(path):
    """Check required ownership, uniqueness, public defaults and retired bridge."""
    check(path, LATEST_SCHEMA_VERSION)
    with connect(path) as conn:
        assert conn.execute('SELECT COUNT(*) FROM listening_profiles WHERE user_id IS NULL OR require_login<>0').fetchone() == (0,)
        assert conn.execute("SELECT COUNT(*) FROM users WHERE role='admin' AND status='active'").fetchone()[0] > 0
        assert conn.execute('SELECT COUNT(*) FROM users u WHERE NOT EXISTS(SELECT 1 FROM listening_profiles p WHERE p.user_id=u.id)').fetchone() == (0,)
        assert conn.execute('SELECT COUNT(*) FROM listening_profiles').fetchone() == conn.execute('SELECT COUNT(DISTINCT user_id) FROM listening_profiles').fetchone()
        assert conn.execute("SELECT COUNT(*) FROM sqlite_master WHERE name IN ('legacy_user_profiles','trg_legacy_user_profile')").fetchone() == (0,)
        return dict(conn.execute('SELECT status,COUNT(*) FROM users GROUP BY status'))


def validate(binary, source, root, index, stage_v4=False):
    """Migrate a copied database, restore its backup independently, and restart twice."""
    original = component_hashes(source)
    work = root / f'copy-{index}'
    work.mkdir(mode=0o700)
    database = work / 'tunetrail.db'
    copy_components(source, database)
    with connect(database) as conn:
        version = conn.execute('SELECT MAX(version_id) FROM goose_db_version WHERE is_applied=1').fetchone()[0]
    assert version in (3, 4), 'provide a pre-account v3/v4 source'
    if stage_v4 and version == 3:
        # Stage the published 004 only on the private copy, then exercise v4-to-current.
        with connect(database) as conn:
            ddl = (Path(__file__).resolve().parents[1] / 'db/migrations_sqlite/004_ingest_infrastructure.sql').read_text().split('-- +goose Down')[0]
            conn.executescript(ddl)
            conn.execute('INSERT INTO goose_db_version(version_id,is_applied) VALUES(4,1)')
            conn.commit()
        version = 4
    before, counts = inventory(database), profile_counts(database)
    server = Server(binary, work)
    try:
        server.start()
        status = accounts(database)
        preserved(database, before)
        assert profile_counts(database) == counts
        server.stop()
        backup, = work.glob('tunetrail.db.pre-' + ('accounts' if version == 4 else 'ingest') + '-*.bak')
        check(backup, version)
        assert backup.stat().st_mode & 0o777 == 0o600
        assert inventory(backup) == before
        restored = root / f'restored-{index}'
        restored.mkdir(mode=0o700)
        copy_components(backup, restored / 'tunetrail.db')
        check(restored / 'tunetrail.db', version)
        restore_server = Server(binary, restored)
        try:
            restore_server.start()
            accounts(restored / 'tunetrail.db')
            preserved(restored / 'tunetrail.db', before)
            assert profile_counts(restored / 'tunetrail.db') == counts
        finally:
            restore_server.stop()
        stable = inventory(database)
        for _ in range(2):
            server.start()
            server.stop()
            assert inventory(database) == stable
    finally:
        server.stop()
    assert component_hashes(source) == original, 'source components changed independently'
    return dict(case=index, source_version=version, target_version=LATEST_SCHEMA_VERSION, counts_before=counts,
                counts_after=profile_counts(database), account_states=status,
                preserved=True, backup_restore=True, restart=True, source_unchanged=True)


def main():
    """Require explicit sources and discard all temporary data after aggregate reporting."""
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--binary', type=Path, required=True)
    parser.add_argument('--source', type=Path, action='append', required=True)
    parser.add_argument('--stage-v4', action='store_true', help='Stage published 004 on the private copy first')
    args = parser.parse_args()
    with tempfile.TemporaryDirectory(prefix='tunetrail-accounts-v5-') as tmp:
        reports = [validate(args.binary.resolve(), p.resolve(), Path(tmp), i, args.stage_v4)
                   for i, p in enumerate(args.source, 1)]
        print(json.dumps(dict(result='PASS', copies=reports), sort_keys=True))


if __name__ == '__main__':
    main()
