#!/usr/bin/env python3
"""Run the React helper browser regression on isolated 0/1/N profile fixtures."""
import argparse
import subprocess
import tempfile
from pathlib import Path

from roadmap06_fixture import create, check_database
from validate_multiuser import Server


def main():
    """Start disposable local servers and always close them after the browser run."""
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--binary', required=True, type=Path)
    args = parser.parse_args()
    with tempfile.TemporaryDirectory(prefix='koito-028-') as temp:
        root = Path(temp) / 'fixtures'
        create(root)
        servers = []
        try:
            for name in ('zero', 'one', 'react'):
                server = Server(args.binary.resolve(), root / name, environment={'KOITO_WEB_UI': 'react'})
                servers.append(server)
                server.start()
            subprocess.run(['node', str(Path(__file__).with_suffix('.cjs')), *(server.origin for server in servers)], check=True)
        finally:
            for server in reversed(servers):
                server.stop()
        for name, counts in [('zero', {}), ('one', {'alpha': 7}), ('react', {'alpha': 7, 'beta': 2, 'empty': 0})]:
            check_database(root / name / 'koito.db', counts)
        print('SQLite integrity and profile counts OK')


if __name__ == '__main__':
    main()
