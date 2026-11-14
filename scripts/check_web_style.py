#!/usr/bin/env python3
"""Keep web templates and stylesheets readable: no line longer than 160 characters.

Exceptions are lines that cannot be broken without changing the output: a single
attribute whose value has no spaces (a long URL), or a single Go string literal
(a long translation default) on its own line.
"""
import re
import sys
from pathlib import Path

LIMIT = 160
ROOT = Path(__file__).resolve().parent.parent
# Only for long URLs and translation defaults; anything else must be wrapped.
UNBREAKABLE = re.compile(r'^\s*([\w:-]+="[^"\s]*"|"[^"]*"(\}\})?)\s*$')


def offenders(root=ROOT):
    """Yield (path, line number, length) for every line over the limit."""
    files = sorted((root / 'assets/templates').rglob('*.html')) + sorted((root / 'assets/css').glob('*.css'))
    for path in files:
        for number, line in enumerate(path.read_text().splitlines(), 1):
            if len(line) > LIMIT and not UNBREAKABLE.match(line):
                yield path.relative_to(root), number, len(line)


def main():
    found = list(offenders())
    for path, number, length in found:
        print(f'{path}:{number}: {length} characters (limit {LIMIT})')
    return 1 if found else 0


if __name__ == '__main__':
    sys.exit(main())
