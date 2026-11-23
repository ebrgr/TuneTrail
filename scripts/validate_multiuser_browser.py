#!/usr/bin/env python3
"""Exercise the multiuser browser journey on disposable SQLite instances."""

import argparse
from contextlib import closing
from pathlib import Path
import sqlite3
import subprocess
import tempfile

from roadmap06_fixture import FIXTURE_PASSWORD_HASH
from validate_multiuser import PASSWORD, Server


def run(binary, directory, mode):
    """Start a fresh server for each mode so the second run has its own sessions."""
    server = Server(binary, directory)
    try:
        server.start()
        database = directory / 'tunetrail.db'
        with closing(sqlite3.connect(database)) as conn, conn:
            for slug, protected in [('listener', 0), ('other', 1)]:
                user = conn.execute(
                    "INSERT INTO users(username,role,password) VALUES(?,'user',?)",
                    (slug, FIXTURE_PASSWORD_HASH.encode()),
                )
                conn.execute(
                    'INSERT INTO listening_profiles(user_id,slug,display_name,created_at,require_login) '
                    'VALUES(?,?,?,0,?)', (user.lastrowid, slug, slug, protected),
                )
        runner = Path(__file__).with_suffix('.cjs')
        subprocess.run(['node', str(runner), server.origin, str(database), PASSWORD, mode], check=True)
    finally:
        server.stop()


if __name__ == '__main__':
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--binary', type=Path, required=True)
    args = parser.parse_args()
    with tempfile.TemporaryDirectory(prefix='tunetrail-browser-') as tmp:
        for mode in ('on', 'off'):
            run(args.binary.resolve(strict=True), Path(tmp) / mode, mode)
