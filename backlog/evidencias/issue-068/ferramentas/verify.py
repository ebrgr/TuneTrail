"""Prove the bundle holds exactly the original rules: same multiset of (media, rule) after whitespace normalisation."""
import re, sys, collections, json
sys.path.insert(0, __import__('os').path.dirname(__file__))
from cssparse import chunks

def norm(s):
    s = re.sub(r'/\*.*?\*/', '', s, flags=re.S)
    return re.sub(r'\s+', '', s)

def flatten(text, ctx=()):
    for raw, pre, body, _ in chunks(text):
        if pre is None:
            continue
        if pre.startswith('@layer') and body is not None and not pre.endswith(';'):
            yield from flatten(body, ctx)
        elif pre.startswith('@media'):
            for r in flatten(body, ctx + (norm(pre),)):
                yield r
        else:
            yield ctx + (norm(pre) + '{' + norm(body) + '}',)

def strip_statement(t):
    return re.sub(r'@layer [a-z, ]+;', '', t)

orig = collections.Counter(flatten(open(sys.argv[1]).read()))
new = collections.Counter()
for f in sys.argv[2:]:
    new.update(flatten(strip_statement(open(f).read())))
print('original rules', sum(orig.values()), 'bundle rules', sum(new.values()), 'missing', list(orig - new)[:5], 'extra', list(new - orig)[:5])
sys.exit(0 if orig == new else 1)
