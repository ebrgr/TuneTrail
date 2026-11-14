#!/usr/bin/env python3
"""Package a local binary with its matching external UI and catalogs; never publish."""
import argparse
from pathlib import Path
import tarfile

ROOT = Path(__file__).resolve().parent.parent


def main():
    """Create a new archive, refusing to overwrite an existing distribution."""
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--binary', type=Path, required=True)
    parser.add_argument('--output', type=Path, required=True)
    args = parser.parse_args()
    binary = args.binary.resolve(strict=True)
    version = (ROOT/'internal/version/VERSION').read_text().strip()
    prefix = f'tunetrail-{version}'
    # The installation archive declares root-owned, read-only resources; executable remains runnable.
    def permissions(entry):
        entry.uid = entry.gid = 0
        entry.uname = entry.gname = 'root'
        entry.mode = 0o555 if entry.isdir() or entry.name == f'{prefix}/tunetrail' else 0o444
        return entry
    with tarfile.open(args.output, 'x:gz') as archive:
        archive.add(binary, arcname=f'{prefix}/tunetrail', filter=permissions)
        for name in ('assets', 'langs', 'LICENSE'):
            archive.add(ROOT/name, arcname=f'{prefix}/{name}', filter=permissions)
    print(args.output)


if __name__ == '__main__':
    main()
