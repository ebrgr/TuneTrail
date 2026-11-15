"""List rule pairs whose winner may change under @layer: lower-layer rule that used to win (higher specificity,
or equal specificity and later in source) against a higher-layer rule touching the same property and a shared key
(rightmost compound class/tag/attribute)."""
import re, sys, os, subprocess
sys.path.insert(0, os.path.dirname(__file__))
from cssparse import chunks
ROOT = __import__('subprocess').run(['git', '-C', __import__('os').path.dirname(__import__('os').path.abspath(__file__)), 'rev-parse', '--show-toplevel'], capture_output=True, text=True, check=True).stdout.strip() + '/'
ORDER = ['reset', 'tokens', 'base', 'layout', 'components', 'pages', 'utilities']

def spec(sel):
    sel = re.sub(r'::?(before|after|backdrop|placeholder)', ' ', sel)
    s = sel
    a = len(re.findall(r'#[\w-]+', s))
    b = len(re.findall(r'\.[\w-]+|\[[^\]]*\]|:(?!is|not|has|where)[\w-]+', s))
    c = len(re.findall(r'(?:^|[\s>+~(,])([a-z][\w-]*)', s))
    return (a, b, c)

def keys(sel):
    last = re.split(r'[\s>+~]+', sel.strip())[-1]
    last = re.sub(r'::?[\w-]+(\([^)]*\))?', '', last)
    ks = set(re.findall(r'\.[\w-]+', last)) | set(re.findall(r'\[[\w-]+', last))
    tag = re.match(r'[a-z][\w-]*', last)
    if tag: ks.add(tag.group(0))
    if last in ('*', ''): ks.add('*')
    return ks

def props(body):
    body = re.sub(r'/\*.*?\*/', '', body, flags=re.S)
    return {p.strip().lower() for p in re.findall(r'(?:^|;)\s*([\w-]+)\s*:', body)}

def rules(text, layer, origin, ctx=''):
    for raw, pre, body, line in chunks(text):
        if pre is None: continue
        if pre.startswith('@layer') and not pre.endswith(';'):
            yield from rules(body, pre.split()[1], origin, ctx)
        elif pre.startswith('@media'):
            yield from rules(body, layer, origin, pre)
        elif pre.startswith('@'):
            continue
        else:
            for sel in pre.split(','):
                yield dict(sel=sel.strip(), layer=layer, props=props(body), media=ctx, origin=origin)

orig = subprocess.run(['git', '-C', ROOT, 'show', 'cfd6717:assets/css/ui.css'], capture_output=True, text=True).stdout
admin = subprocess.run(['git', '-C', ROOT, 'show', 'cfd6717:assets/css/admin.css'], capture_output=True, text=True).stdout
# original source position, used for the unlayered tie-break
pos = {}
for i, r in enumerate(list(rules(orig, None, 'ui')) + list(rules(admin, None, 'admin'))):
    pos.setdefault((r['origin'], r['sel'], r['media'], frozenset(r['props'])), i)
new = []
import json
manifest = json.load(open(ROOT + 'assets/manifest.json'))
for part in manifest['public/static/ui.css']:
    new += list(rules(re.sub(r'@layer [a-z, ]+;', '', open(ROOT + 'assets/' + part).read()), None, 'ui'))
new += list(rules(open(ROOT + 'assets/' + manifest['public/static/admin.css']).read(), None, 'admin'))
for i, r in enumerate(new):
    r['bundle'] = i
    r['pos'] = pos[(r['origin'], r['sel'], r['media'], frozenset(r['props']))]
found = 0
for A in new:
    for B in new:
        la, lb = ORDER.index(A['layer']), ORDER.index(B['layer'])
        # cross-layer: A below B; same layer: A now before B in the bundle
        if la > lb or (la == lb and A['bundle'] >= B['bundle']): continue
        shared = A['props'] & B['props']
        if not shared: continue
        if not (keys(A['sel']) & keys(B['sel']) or '*' in keys(B['sel']) or '*' in keys(A['sel'])): continue
        sa, sb = spec(A['sel']), spec(B['sel'])
        if (la < lb and sa > sb) or (sa == sb and A['pos'] > B['pos']):
            found += 1
            print(f"{A['layer']:10} {A['sel']!r:60} {A['media'][:30]:30} beats? {B['layer']:10} {B['sel']!r} {B['media'][:30]} props={sorted(shared)}")
print('candidates', found)
