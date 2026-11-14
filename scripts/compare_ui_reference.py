#!/usr/bin/env python3
"""Compare exported UI bytes; normalized equality is advisory, never visual approval."""
import argparse
import hashlib
import json
import re
from pathlib import Path


def compare(base, candidate):
    """Keep asset order and inline spaces; normalize only indentation between tags."""
    def inventory(root):
        return {str(p.relative_to(root)): p.read_bytes() for p in root.rglob('*')
                if p.is_file()}
    left, right = inventory(base), inventory(candidate)
    differences = []
    for name in sorted(left.keys() | right.keys()):
        a, b = left.get(name), right.get(name)
        if a == b:
            continue
        normalized = False
        if a is not None and b is not None and name.endswith('.html'):
            normalized = re.sub(rb'>[ \t]*\n\s*<', b'><', a) == re.sub(rb'>[ \t]*\n\s*<', b'><', b)
        differences.append({'file': name, 'result': 'normalized' if normalized else 'different',
                            'base_sha256': hashlib.sha256(a).hexdigest() if a is not None else None,
                            'new_sha256': hashlib.sha256(b).hexdigest() if b is not None else None})
    return {'files': len(left), 'differences': differences,
            'exit_code': 2 if any(x['result'] == 'different' for x in differences) else (1 if differences else 0)}


def main():
    """Exit zero for identical bytes, one for indentation-only changes, two for other differences."""
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('base', type=Path)
    parser.add_argument('candidate', type=Path)
    args = parser.parse_args()
    if not args.base.is_dir() or not args.candidate.is_dir():
        parser.error('both reference directories must exist')
    report = compare(args.base, args.candidate)
    print(json.dumps(report, indent=2))
    raise SystemExit(report['exit_code'])


if __name__ == '__main__':
    main()
