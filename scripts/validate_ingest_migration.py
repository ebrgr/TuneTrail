#!/usr/bin/env python3
"""Exercise normal v3-to-v5 startup, backup and restore on disposable copies.

Optional --source files are copied with all SQLite sidecars before SQLite opens
anything. Only temporary copies are used; stdout contains aggregate counts only.
"""
import argparse
import contextlib
import hashlib
import json
from pathlib import Path
import shutil
import sqlite3
import tempfile

from roadmap06_fixture import FIXTURE_PASSWORD_HASH
from validate_multiuser import Server, WEB, ROOT, create_legacy


def connect(path):
    """Open an existing temporary database without implicit creation."""
    return contextlib.closing(sqlite3.connect(path.resolve().as_uri() + '?mode=rw', uri=True))


def digests(path):
    """Compare every existing table/column in memory without exposing its rows."""
    with connect(path) as conn:
        result = {}
        tables = conn.execute("SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%' AND name NOT IN ('goose_db_version','legacy_user_profiles')").fetchall()
        for (table,) in tables:
            columns = {'users':'id,username,role,password','listening_profiles':'id,slug,display_name,avatar_image_id,avatar_icon,created_at'}.get(table,'*')
            clause = " WHERE status='active'" if table=='users' and 'status' in [r[1] for r in conn.execute('PRAGMA table_info(users)')] else ''
            rows = sorted(repr(row) for row in conn.execute(f'SELECT {columns} FROM "{table}"{clause}'))
            result[table] = hashlib.sha256('\n'.join(rows).encode()).hexdigest()
        return result


def profile_counts(path):
    """Report counts indexed by numeric profile only, without names or content."""
    with connect(path) as conn:
        return dict(conn.execute('SELECT profile_id,COUNT(*) FROM listens GROUP BY profile_id'))


def check(path, version):
    """Verify migration history, pages and referential integrity."""
    with connect(path) as conn:
        assert conn.execute('SELECT MAX(version_id) FROM goose_db_version WHERE is_applied=1').fetchone() == (version,)
        assert conn.execute('PRAGMA integrity_check').fetchall() == [('ok',)]
        assert conn.execute('PRAGMA foreign_key_check').fetchall() == []


def copy_components(source, destination):
    """Copy database and recovery components before making any SQLite connection."""
    for suffix in ('', '-wal', '-shm', '-journal'):
        candidate = Path(str(source) + suffix)
        if candidate.exists():
            shutil.copyfile(candidate, Path(str(destination) + suffix))


def component_hashes(source):
    """Check source bytes without allowing SQLite to open the source."""
    return {suffix: hashlib.sha256(Path(str(source) + suffix).read_bytes()).digest()
            for suffix in ('', '-wal', '-shm', '-journal') if Path(str(source) + suffix).exists()}


def validate(binary, source, root, name):
    """Migrate both canonical and legacy filenames and rehearse backup restoration."""
    source_hashes = component_hashes(source)
    reports = []
    filenames = (source.name,) if source.name in ('tunetrail.db', 'koito.db') else ('tunetrail.db', 'koito.db')
    for filename in filenames:
        work = root / f'{name}-{filename}'
        work.mkdir(mode=0o700)
        database = work / filename
        copy_components(source, database)
        check(database, 3)
        before, counts = digests(database), profile_counts(database)
        server = Server(binary, work)
        try:
            server.start()
            target = work / 'tunetrail.db'
            check(target, 7)
            after = digests(target)
            assert all(after[table] == digest for table, digest in before.items()), 'existing rows changed'
            assert profile_counts(target) == counts
            profiles = server.request('GET', WEB + '/profiles')
            for profile in profiles:
                stats = server.request('GET', WEB + f'/profiles/{profile["slug"]}/stats?period=all_time')
                assert stats['listen_count'] == counts.get(profile['id'], 0)
            backups = list(work.glob(filename + ('.pre-ingest-*.bak' if filename == 'tunetrail.db' else '.pre-tunetrail-*.bak')))
            assert len(backups) == 1
            backup = backups[0]
            check(backup, 3)
            assert digests(backup) == before
            assert backup.stat().st_mode & 0o777 == 0o600
            server.stop()
            stable = digests(target)
            server.start()
            check(target, 7)
            server.stop()
            assert digests(target) == stable, 'restart changed rows'
            # Restore to an empty isolated directory, then migrate the restored v3 again.
            restored = root / f'{name}-{filename}-restore'
            restored.mkdir(mode=0o700)
            with connect(backup) as src, contextlib.closing(sqlite3.connect(restored / 'tunetrail.db')) as dst:
                src.backup(dst)
            check(restored / 'tunetrail.db', 3)
            assert digests(restored / 'tunetrail.db') == before
            restored_server = Server(binary, restored)
            try:
                restored_server.start()
                check(restored / 'tunetrail.db', 7)
                assert profile_counts(restored / 'tunetrail.db') == counts
            finally:
                restored_server.stop()
            reports.append({'case': name, 'entry': filename, 'from': 3, 'to': 5,
                            'counts_before': counts, 'counts_after': profile_counts(target),
                            'all_existing_table_digests_equal': True, 'backup_restore': 'ok',
                            'integrity': 'ok', 'restart': 'ok', 'public_stats': 'ok'})
        finally:
            server.stop()
    assert component_hashes(source) == source_hashes, "source components changed"
    return reports


def main():
    """Generate fictional multi-profile v3 or copy explicitly supplied examples."""
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--binary', type=Path, required=True)
    parser.add_argument('--source', type=Path, action='append', default=[])
    args = parser.parse_args()
    with tempfile.TemporaryDirectory(prefix='tunetrail-ingest-v4-') as tmp:
        root = Path(tmp)
        sources = args.source
        if not sources:
            fixture = root / 'fictional-v3.db'
            create_legacy(fixture, FIXTURE_PASSWORD_HASH.encode('ascii'))
            with connect(fixture) as conn:
                ddl = (ROOT / 'db/migrations_sqlite/003_listening_profiles.sql').read_text().split('-- +goose Down')[0]
                conn.executescript(ddl)
                conn.execute('INSERT INTO goose_db_version(version_id,is_applied) VALUES(3,1)')
                conn.execute("INSERT INTO listening_profiles(id,slug,display_name,created_at) VALUES(2,'second','Second',0)")
                conn.execute("INSERT INTO listens(profile_id,track_id,listened_at,client) VALUES(2,1,1700000002,'fictional')")
                conn.commit()
            sources = [fixture]
        reports = []
        for index, source in enumerate(sources):
            reports += validate(args.binary.resolve(), source.resolve(), root, f'copy-{index+1}')
        print(json.dumps({'result': 'PASS', 'copies': reports}, sort_keys=True))


if __name__ == '__main__':
    main()
