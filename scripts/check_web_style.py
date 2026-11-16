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


def width_media_offenders(root=ROOT):
    """Reject viewport size queries except the three mobile-first width thresholds.

    Strip comments and strings before scanning, and decode CSS escapes so spelling
    a feature differently cannot bypass the policy. Device/accessibility queries
    remain independent of the layout breakpoint scale.
    """
    allowed = {'(min-width:40rem)', '(min-width:64rem)', '(min-width:90rem)'}
    for path in sorted((root / 'assets/css').rglob('*.css')):
        text = path.read_text()
        text = re.sub(r'/\*.*?\*/', lambda m: '\n' * m[0].count('\n'), text, flags=re.S)
        text = re.sub(r'"(?:\\.|[^"\\])*"|\'(?:\\.|[^\'\\])*\'', '""', text)
        text = re.sub(r'\\([0-9a-fA-F]{1,6})[ \t]?', lambda m: chr(int(m[1], 16) or 0xfffd), text)
        text = re.sub(r'\\([^\n])', r'\1', text)
        for match in re.finditer(r'@media\b([^{};]*)', text, re.I):
            condition = match[1].lower()
            # Height/device-width/orientation queries would create another layout scale.
            size = re.search(r'width|height|orientation|aspect-ratio', condition)
            if not size:
                continue
            canonical = re.sub(r'\s+', '', condition)
            if canonical not in allowed:
                yield path.relative_to(root), text.count('\n', 0, match.start()) + 1, condition.strip()


def main():
    """Run readability, breakpoint and script-budget checks together for ui.check."""
    found = list(offenders())
    for path, number, length in found:
        print(f'{path}:{number}: {length} characters (limit {LIMIT})')
    long = list(long_stylesheets())
    for path, count in long:
        print(f'{path}: {count} lines (limit {CSS_LINES}); split it by function')
    heavy = list(over_budget())
    for path, size, budget in heavy:
        print(f'{path}: {size} bytes gzip (budget {budget})')
    media = list(width_media_offenders())
    for path, number, condition in media:
        print(f'{path}:{number}: unsupported media query {condition}; use min-width 40rem, 64rem or 90rem')
    return 1 if found or long or heavy or media else 0


if __name__ == '__main__':
    sys.exit(main())
