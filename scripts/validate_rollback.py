#!/usr/bin/env python3
"""Exercise rollback from TuneTrail 0.6+ (tunetrail.db) to the previous release (koito.db).

Manual tool for ISSUE-064, run on fictional disposable data only:
  python3 scripts/validate_rollback.py --binary ./tunetrail --previous-binary <0.5 build>
Build the previous release without touching the checkout, for example:
  git archive 6b13514 | tar -x -C /tmp/tt-0.5 && (cd /tmp/tt-0.5 && CGO_ENABLED=0 go build -o tunetrail-0.5 ./cmd/api)
"""
import argparse
import contextlib
import json
import shutil
import sqlite3
import tempfile
import time
from pathlib import Path

from roadmap06_fixture import FIXTURE_PASSWORD_HASH
from validate_multiuser import Server, WEB, LBZ, create_legacy

TOKEN = 'legacy-validation-token'


def listens(server):
    # Public listen count of the single legacy profile.
    return server.request('GET', WEB + '/profiles/default/stats?period=all_time')['listen_count']


def start(binary, directory, stack):
    server = Server(binary, directory, environment={'KOITO_FORCE_TZ': 'UTC'})
    server.start()
    stack.callback(server.stop)
    return server


def main():
    parser = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    parser.add_argument('--binary', type=Path, required=True, help='current build (uses tunetrail.db)')
    parser.add_argument('--previous-binary', type=Path, required=True, help='previous release build (uses koito.db)')
    args = parser.parse_args()
    current, previous = args.binary.resolve(strict=True), args.previous_binary.resolve(strict=True)
    report = {}
    with tempfile.TemporaryDirectory(prefix='tunetrail-rollback-') as tmp:
        root = Path(tmp)
        legacy_dir = root / 'in-place'
        legacy_dir.mkdir()
        create_legacy(legacy_dir / 'koito.db', FIXTURE_PASSWORD_HASH.encode('ascii'))

        # 1. Upgrade: the current build migrates koito.db into tunetrail.db.
        with contextlib.ExitStack() as stack:
            new = start(current, legacy_dir, stack)
            assert listens(new) == 1
            # A listen recorded after the upgrade exists only in tunetrail.db.
            new.request('POST', LBZ + '/submit-listens', {'listen_type': 'single', 'payload': [{'listened_at': int(time.time()) - 60, 'track_metadata': {'artist_name': 'After upgrade', 'track_name': 'Only in tunetrail.db'}}]}, token=TOKEN)
            assert listens(new) == 2
        backups = sorted(legacy_dir.glob('koito.db.pre-tunetrail-*.bak'))
        assert len(backups) == 1, backups
        backup = backups[0]
        report['upgrade'] = 'current build migrated koito.db; one listen added afterwards'

        # 3 (prepared first, before the previous release touches koito.db): copy the .bak.
        from_backup = root / 'from-backup'
        from_backup.mkdir()
        shutil.copy2(backup, from_backup / 'koito.db')
        forward = root / 'forward-again'
        forward.mkdir()
        shutil.copy2(backup, forward / 'koito.db')

        # 2. Rollback in place: the previous release reads the untouched koito.db.
        with contextlib.ExitStack() as stack:
            old = start(previous, legacy_dir, stack)
            assert listens(old) == 1, 'previous release must see the pre-upgrade history'
            assert old.request('GET', LBZ + '/validate-token', token=TOKEN)['valid'] is True
        report['rollback_in_place'] = 'previous release started on the legacy koito.db: 1 listen; the listen recorded after the upgrade is not visible'

        # The previous release migrates koito.db in place, so the current build now
        # refuses the directory conservatively instead of guessing which copy is right.
        with contextlib.ExitStack() as stack:
            again = Server(current, legacy_dir, environment={'KOITO_FORCE_TZ': 'UTC'})
            try:
                again.start()
                raise AssertionError('current build must refuse a koito.db changed after migration')
            except RuntimeError:
                log = (legacy_dir / 'server.log').read_text(errors='ignore')
                assert 'legacy database changed after migration' in log, log[-500:]
            finally:
                again.stop()
        report['upgrade_after_rollback'] = 'refused: legacy database changed after migration (manual reconciliation required)'

        # 3. Rollback from the retained backup into an empty directory.
        with contextlib.ExitStack() as stack:
            old = start(previous, from_backup, stack)
            assert listens(old) == 1
            assert old.request('GET', LBZ + '/validate-token', token=TOKEN)['valid'] is True
        report['rollback_from_backup'] = 'previous release started from the .bak copied as koito.db: 1 listen, legacy key valid'

        # 4. Forward again from the backup: the current build migrates the copy.
        with contextlib.ExitStack() as stack:
            new = start(current, forward, stack)
            assert listens(new) == 1
        with contextlib.closing(sqlite3.connect(forward / 'tunetrail.db')) as conn:
            assert conn.execute('PRAGMA integrity_check').fetchall() == [('ok',)]
        report['upgrade_from_backup'] = 'current build migrated the .bak copy again: 1 listen, integrity ok'
    print(json.dumps(report, indent=2, ensure_ascii=False))


if __name__ == '__main__':
    main()
