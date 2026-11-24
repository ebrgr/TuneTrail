#!/usr/bin/env python3
"""Audit which tables and columns of a TuneTrail database are used, populated or repeated.

Evidence per column:
  * Go: how many SQL statements of the production Go code read it and how many write it
    (statement level parse; test files and the schema layout list are ignored).
  * DB: whether an index, trigger, view, generated column, foreign key or CHECK of another
    object depends on it.
  * Data: non-null count, distinct values and whether it is constant, on a populated database.
  * Repeat: columns of the same table that hold the same value in every row.

Usage: audit_db_usage.py DATABASE REPO_ROOT OUTPUT_JSON
Read-only: the database is opened with mode=ro and nothing is written next to it.
"""
import json
import os
import re
import sqlite3
import subprocess
import sys

db_path, repo, out_path = sys.argv[1:4]
db = sqlite3.connect(f'file:{db_path}?mode=ro', uri=True)

tables = [r[0] for r in db.execute("select name from sqlite_master where type='table' and name not like 'sqlite_%' order by name")]
columns = {t: [r[1] for r in db.execute(f'pragma table_xinfo("{t}")')] for t in tables}
generated = {t: {r[1] for r in db.execute(f'pragma table_xinfo("{t}")') if r[6] in (2, 3)} for t in tables}
rowcount = {t: db.execute(f'select count(*) from "{t}"').fetchone()[0] for t in tables}

# ---- Go: statement level reads and writes -------------------------------------------------
files = subprocess.check_output(
    "find . -name '*.go' -not -name '*_test.go' -not -path './backlog/*' -not -path './node_modules/*' -not -path './scripts/*'",
    shell=True, text=True, cwd=repo).split()
SKIP = ('internal/db/sqlite/ingest_schema.go', 'internal/db/sqlite/bootstrap.go')
literals = []
for f in files:
    if f.endswith(SKIP):
        continue
    src = open(os.path.join(repo, f), errors='ignore').read()
    for m in re.finditer(r'`([^`]*)`|"((?:\\.|[^"\\])*)"', src, re.S):
        text = m.group(1) if m.group(1) is not None else m.group(2)
        if re.search(r'\b(SELECT|INSERT|UPDATE|DELETE|REPLACE)\b', text, re.I) and re.search(r'\b(FROM|INTO|SET|JOIN)\b', text, re.I):
            literals.append((f, text))

reads = {t: {c: 0 for c in columns[t]} for t in tables}
writes = {t: {c: 0 for c in columns[t]} for t in tables}
token_refs = {t: {c: 0 for c in columns[t]} for t in tables}
alltext = '\n'.join(l for _, l in literals)
src_all = '\n'.join(open(os.path.join(repo, f), errors='ignore').read() for f in files if not f.endswith(SKIP))
for t in tables:
    for c in columns[t]:
        token_refs[t][c] = len(re.findall(r'(?<![A-Za-z0-9_])' + re.escape(c) + r'(?![A-Za-z0-9_])', src_all))

IDENT = re.compile(r'[A-Za-z_][A-Za-z0-9_]*')
for _, text in literals:
    for stmt in re.split(r';', text):
        s = ' '.join(stmt.split())
        if not s:
            continue
        alias = {}
        for m in re.finditer(r'\b(?:FROM|JOIN|INTO|UPDATE)\s+([A-Za-z_]\w*)(?:\s+(?:AS\s+)?([A-Za-z_]\w*))?', s, re.I):
            tbl = m.group(1)
            if tbl in tables:
                alias[tbl] = tbl
                a = m.group(2)
                if a and a.upper() not in ('WHERE', 'SET', 'ON', 'JOIN', 'LEFT', 'INNER', 'VALUES', 'SELECT', 'ORDER', 'GROUP', 'LIMIT', 'USING', 'AS', 'RETURNING', 'CROSS', 'NOT', 'DEFAULT'):
                    alias[a] = tbl
        involved = sorted(set(alias.values()))
        if not involved:
            continue
        written = {}
        ins = re.match(r'(?:INSERT|REPLACE)(?:\s+OR\s+\w+)?\s+INTO\s+([A-Za-z_]\w*)\s*(\(([^)]*)\))?', s, re.I)
        if ins and ins.group(1) in tables:
            t = ins.group(1)
            cols = [c.strip() for c in ins.group(3).split(',')] if ins.group(3) else columns[t]
            written[t] = set(cols)
        for m in re.finditer(r'\bSET\s+(.*?)(?:\bWHERE\b|\bFROM\b|\bRETURNING\b|$)', s, re.I):
            target = [t for t in involved][:1]
            upd = re.match(r'UPDATE\s+([A-Za-z_]\w*)', s, re.I)
            t = upd.group(1) if upd and upd.group(1) in tables else (target[0] if target else None)
            if t:
                for lhs in re.finditer(r'(?:^|,)\s*(?:\w+\.)?([A-Za-z_]\w*)\s*=', m.group(1)):
                    written.setdefault(t, set()).add(lhs.group(1))
        for m in re.finditer(r'ON CONFLICT[^)]*\)?\s*DO UPDATE SET\s+(.*?)(?:\bWHERE\b|$)', s, re.I):
            t = ins.group(1) if ins and ins.group(1) in tables else involved[0]
            for lhs in re.finditer(r'(?:^|,)\s*([A-Za-z_]\w*)\s*=', m.group(1)):
                written.setdefault(t, set()).add(lhs.group(1))
        for t, cols in written.items():
            for c in cols:
                if c in writes[t]:
                    writes[t][c] += 1
        # reads: every other column token of an involved table
        for m in IDENT.finditer(s):
            tok = m.group(0)
            prefix = s[:m.start()]
            qual = re.search(r'([A-Za-z_]\w*)\.$', prefix)
            if qual and qual.group(1) in alias:
                t = alias[qual.group(1)]
                cands = [t] if tok in reads[t] else []
            elif prefix.endswith('.'):
                continue
            else:
                cands = [t for t in involved if tok in reads[t]]
            for t in cands:
                if tok in written.get(t, ()) and re.match(r'(?:INSERT|REPLACE|UPDATE)', s, re.I) and (
                        (ins and ins.group(3) and tok in [x.strip() for x in ins.group(3).split(',')] and m.start() < (ins.end())) or
                        re.search(r'\bSET\b[^=]*\b' + re.escape(tok) + r'\s*=', s[:m.end() + 2], re.I) and s[m.end():m.end() + 3].strip().startswith('=')):
                    continue
                reads[t][tok] += 1

# ---- DB objects that depend on a column -----------------------------------------------------
objects = db.execute("select type,name,tbl_name,sql from sqlite_master where sql is not null and type in ('index','trigger','view')").fetchall()
fk_cols = {t: {r[3] for r in db.execute(f'pragma foreign_key_list("{t}")')} for t in tables}
fk_targets = {}
for t in tables:
    for r in db.execute(f'pragma foreign_key_list("{t}")'):
        fk_targets.setdefault(r[2], set()).add(r[4])


def dbuse(t, c):
    use = []
    for typ, name, tbl, sql in objects:
        if re.search(r'(?<![A-Za-z0-9_])' + re.escape(c) + r'(?![A-Za-z0-9_])', sql) and (tbl == t or t in sql):
            use.append(typ)
    if c in fk_cols[t]:
        use.append('fk')
    if c in fk_targets.get(t, ()):
        use.append('fk-alvo')
    if c in generated[t]:
        use.append('gerada')
    sql = db.execute("select sql from sqlite_master where name=?", (t,)).fetchone()[0]
    for m in re.finditer(r'\bGENERATED\b.*?\bAS\b\s*\((.*?)\)', sql, re.S | re.I):
        pass
    return sorted(set(use))


# generated columns depend on other columns: record which
gen_deps = {}
for t in tables:
    sql = db.execute("select sql from sqlite_master where name=?", (t,)).fetchone()[0]
    for g in generated[t]:
        m = re.search(r'\b' + re.escape(g) + r'\b[^,]*?\bAS\s*\(', sql, re.S)
        if m:
            seg = sql[m.end() - 1:]
            depth = 0
            for i, ch in enumerate(seg):
                depth += (ch == '(') - (ch == ')')
                if depth == 0:
                    expr = seg[:i + 1]
                    break
            for c in columns[t]:
                if c != g and re.search(r'(?<![A-Za-z0-9_])' + re.escape(c) + r'(?![A-Za-z0-9_])', expr):
                    gen_deps.setdefault(t, set()).add(c)

# ---- data: population and distinct values ---------------------------------------------------
population = {}
for t in tables:
    n = rowcount[t]
    population[t] = {}
    for c in columns[t]:
        if n == 0:
            population[t][c] = None
            continue
        nn, dist = db.execute(f'select count("{c}"), count(distinct "{c}") from "{t}"').fetchone()
        avg = db.execute(f'select avg(length("{c}")) from "{t}" where typeof("{c}") in (\'text\',\'blob\')').fetchone()[0]
        population[t][c] = dict(nonnull=nn, distinct=dist, avg_len=round(avg, 1) if avg else None)

# ---- repeats: identical columns of the same table --------------------------------------------
repeats = {}
for t in tables:
    if rowcount[t] == 0:
        continue
    cols = [c for c in columns[t] if population[t][c] and population[t][c]['nonnull'] > 0]
    for i, a in enumerate(cols):
        for b in cols[i + 1:]:
            same = db.execute(f'select count(*) from "{t}" where "{a}" is not "{b}"').fetchone()[0]
            if same == 0 and population[t][a]['distinct'] > 1:
                repeats.setdefault(t, []).append((a, b))

# ---- size ------------------------------------------------------------------------------------
size = {}
try:
    for name, s in db.execute('select name, sum(pgsize) from dbstat group by name'):
        size[name] = s
except Exception:
    pass
owner = {r[0]: r[1] for r in db.execute("select name,tbl_name from sqlite_master where type in ('table','index')")}
by_table = {}
for name, s in size.items():
    t = owner.get(name)
    if t is None and name.startswith('sqlite_autoindex_'):
        t = name[len('sqlite_autoindex_'):].rsplit('_', 1)[0]
    if t:
        d = by_table.setdefault(t, {'table': 0, 'indexes': 0})
        d['table' if name == t else 'indexes'] += s

result = dict(tables=tables, columns=columns, rowcount=rowcount, reads=reads, writes=writes, token_refs=token_refs,
              dbuse={t: {c: dbuse(t, c) for c in columns[t]} for t in tables}, gen_deps={t: sorted(v) for t, v in gen_deps.items()},
              population=population, repeats=repeats, size=by_table, generated={t: sorted(v) for t, v in generated.items()})
json.dump(result, open(out_path, 'w'), indent=1, default=list)
print('tabelas', len(tables), '| colunas', sum(len(c) for c in columns.values()), '| literais SQL no Go', len(literals), '| escritos em', out_path)
