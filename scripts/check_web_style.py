#!/usr/bin/env python3
"""Keep web templates and stylesheets readable: no line longer than 160 characters,
and no stylesheet part longer than 400 lines (ISSUE-068).

Exceptions are lines that cannot be broken without changing the output: a single
attribute whose value has no spaces (a long URL), or a single Go string literal
(a long translation default) on its own line.
"""
import gzip
import re
import sys
from pathlib import Path

LIMIT = 160
ROOT = Path(__file__).resolve().parent.parent
# Only for long URLs and translation defaults; anything else must be wrapped.
UNBREAKABLE = re.compile(r'^\s*([\w:-]+="[^"\s]*"|"[^"]*"(\}\})?)\s*$')


def offenders(root=ROOT):
    """Yield (path, line number, length) for every line over the limit."""
    files = sorted((root / 'assets/templates').rglob('*.html')) + sorted((root / 'assets/css').rglob('*.css'))
    for path in files:
        for number, line in enumerate(path.read_text().splitlines(), 1):
            if len(line) > LIMIT and not UNBREAKABLE.match(line):
                yield path.relative_to(root), number, len(line)


CSS_LINES = 400


def long_stylesheets(root=ROOT):
    """Yield (path, line count) for every stylesheet part, nested folders included, above CSS_LINES."""
    for path in sorted((root / 'assets/css').rglob('*.css')):
        count = len(path.read_text().splitlines())
        if count > CSS_LINES:
            yield path.relative_to(root), count


# Compressed size budgets for scripts that must stay small (ISSUE-107).
JS_BUDGETS = {'assets/js/search.js': 4096, 'assets/js/enhance.js': 4096}


def over_budget(root=ROOT):
    """Yield (path, gzip size, budget) for every budgeted script above its limit."""
    for name, budget in JS_BUDGETS.items():
        if not (root / name).exists():
            continue  # a missing script is reported by the manifest and node --check
        size = len(gzip.compress((root / name).read_bytes(), compresslevel=9))
        if size > budget:
            yield name, size, budget


def main():
    found = list(offenders())
    for path, number, length in found:
        print(f'{path}:{number}: {length} characters (limit {LIMIT})')
    long = list(long_stylesheets())
    for path, count in long:
        print(f'{path}: {count} lines (limit {CSS_LINES}); split it by function')
    heavy = list(over_budget())
    for path, size, budget in heavy:
        print(f'{path}: {size} bytes gzip (budget {budget})')
    return 1 if found or long or heavy else 0


if __name__ == '__main__':
    sys.exit(main())
