#!/usr/bin/env python3
"""Verify independent legacy migrations and Go/API journeys without React."""
import argparse
import contextlib
import json
import sqlite3
import tempfile
from pathlib import Path
from validate_migration_025 import migrate, route_matrix
from validate_admin_webui import validate as validate_admin
from validate_multiuser import validate_journey


def main():
    """Use independent v2 databases, including a committed WAL snapshot."""
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--binary', type=Path, required=True)
    args = parser.parse_args()
    with tempfile.TemporaryDirectory(prefix='koito-030-') as tmp, contextlib.ExitStack() as stack:
        root, binary = Path(tmp), args.binary.resolve(strict=True)
        web = migrate(binary, root, 'web', 'go', stack)
        matrix = route_matrix(web, 'go')
        admin = migrate(binary, root, 'admin', 'go', stack)
        validate_admin(admin)
        api = migrate(binary, root, 'api-wal', 'go', stack, wal=True)
        validate_journey(api)
        for server in (web, admin, api):
            with contextlib.closing(sqlite3.connect(server.directory / 'tunetrail.db')) as conn:
                assert conn.execute('PRAGMA integrity_check').fetchall() == [('ok',)]
                assert not conn.execute('PRAGMA foreign_key_check').fetchall()
        print(json.dumps({'migrations':3, 'wal': 'ok', 'backup_restore': 'ok', 'route_checks':len(matrix), 'admin_forms':'ok', 'api':'ok', 'integrity':'ok'}))


if __name__ == '__main__':
    main()
