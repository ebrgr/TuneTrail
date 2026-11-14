#!/usr/bin/env python3
"""Capture the navigation using an isolated fictional SQLite fixture."""
import argparse
import subprocess
import tempfile
from pathlib import Path
from roadmap06_fixture import create
from validate_multiuser import Server


def main():
    """Start a local server for each capture run and close it on exit."""
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--binary', type=Path, required=True)
    parser.add_argument('--output', type=Path, required=True)
    parser.add_argument('--stage', choices=['before', 'after'], required=True)
    args = parser.parse_args()
    args.output.mkdir(parents=True, exist_ok=True)
    with tempfile.TemporaryDirectory(prefix='koito-sidebar-029-') as tmp:
        root = Path(tmp) / 'fixtures'
        create(root)
        server = Server(args.binary.resolve(), root / 'go', environment={'KOITO_WEB_UI':'go'})
        try:
            server.start()
            subprocess.run(['node', str(Path(__file__).with_suffix('.cjs')), server.origin, str(args.output.resolve()), args.stage], check=True)
        finally:
            server.stop()


if __name__ == '__main__':
    main()
