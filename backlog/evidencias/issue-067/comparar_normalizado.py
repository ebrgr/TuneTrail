#!/usr/bin/env python3
"""ISSUE-067 evidence: compare two UI reference exports after removing changes with no
rendering effect (asset hashes, UI generation, whitespace between tags, CSS whitespace).
Prints each file that still differs, with the first differing excerpt. With --strict,
whitespace is collapsed but never removed, for steps that must not change text spacing."""
import json
import re
import sys
from pathlib import Path


def norm_html(t):
    t = re.sub(r'\?v=[0-9a-f]+', '?v=H', t)
    t = re.sub(r'data-ui-generation="[^"]*"', 'data-ui-generation="G"', t)
    t = re.sub(r'>\s+<', '><', t)
    t = re.sub(r'\s+', ' ', t)
    # Whitespace just inside a start tag's attribute list ("<link\n rel" / '"\n>').
    t = re.sub(r'\s+>', '>', t)
    return t.strip()


def norm_strict(t):
    """Keep every whitespace run (collapsed to one space): a removed or added space
    between inline elements would be visible, so it must still count as a difference."""
    t = re.sub(r'\?v=[0-9a-f]+', '?v=H', t)
    t = re.sub(r'data-ui-generation="[^"]*"', 'data-ui-generation="G"', t)
    return re.sub(r'\s+', ' ', t)


def norm_css(t):
    t = re.sub(r'\?v=[0-9a-f]+', '?v=H', t)
    t = re.sub(r'/\*.*?\*/', '', t, flags=re.S)
    t = re.sub(r'\s+', ' ', t)
    t = re.sub(r'\s*([{};:,>])\s*', r'\1', t)
    return t.replace(';}', '}').strip()


def main(base, new, strict=False):
    base, new = Path(base), Path(new)
    names = sorted({str(p.relative_to(base)) for p in base.rglob('*') if p.is_file()} |
                   {str(p.relative_to(new)) for p in new.rglob('*') if p.is_file()})
    report = {'files': len(names), 'identical': 0, 'normalized': 0, 'different': []}
    for n in names:
        a, b = base / n, new / n
        if not a.exists() or not b.exists():
            report['different'].append({'file': n, 'reason': 'missing'})
            continue
        x, y = a.read_bytes(), b.read_bytes()
        if x == y:
            report['identical'] += 1
            continue
        f = (norm_strict if strict else norm_html) if n.endswith('.html') else norm_css if n.endswith('.css') else None
        if f and f(x.decode()) == f(y.decode()):
            report['normalized'] += 1
            continue
        if f:
            s, t = f(x.decode()), f(y.decode())
            i = next((k for k in range(min(len(s), len(t))) if s[k] != t[k]), min(len(s), len(t)))
            report['different'].append({'file': n, 'base': s[max(0, i - 80):i + 120], 'new': t[max(0, i - 80):i + 120]})
        else:
            report['different'].append({'file': n, 'reason': 'binary or other'})
    print(json.dumps(report, ensure_ascii=False, indent=1))
    return 1 if report['different'] else 0


if __name__ == '__main__':
    args = [a for a in sys.argv[1:] if a != '--strict']
    sys.exit(main(args[0], args[1], strict='--strict' in sys.argv))
