"""Time the new triggers of migration 026 on its real tables and indexes, with fictitious data."""
import sqlite3, time, uuid, re, sys
# Usage: python3 medir_gatilhos.py [schema.sql]; the default schema was exported from migration 026 before any fix.

schema = open(sys.argv[1] if len(sys.argv) > 1 else __file__.rsplit('/', 1)[0] + '/esquema-gatilhos.sql').read()

def build(n, triggers=True):
    db = sqlite3.connect(':memory:')
    script = schema if triggers else re.sub(r'CREATE TRIGGER.*?END;\n', '', schema, flags=re.S)
    db.executescript(script)
    cols = {r[1]: (r[2], r[3], r[4]) for r in db.execute('PRAGMA table_info(occurrences)')}
    need = [c for c, (t, notnull, default) in cols.items() if notnull and default is None and c not in ('id', 'release_id')]
    names = ['id', 'release_id'] + need
    values = lambda i: [i, 1 + (i - 1) // 10] + ['x' if 'TEXT' in cols[c][0].upper() else 1 for c in need]
    db.executemany(f'INSERT INTO occurrences({",".join(names)}) VALUES({",".join("?"*len(names))})', (values(i) for i in range(1, n + 1)))
    db.executemany('INSERT INTO listens(profile_id,track_id,listened_at,event_id) VALUES(1,?,?,?)', ((i, i, str(uuid.uuid4())) for i in range(1, n + 1)))
    db.execute("INSERT INTO metadata_sources(id,source,instance) VALUES(1,'navidrome-subsonic',?)", ('a' * 64,))
    db.commit()
    return db

def insert(db, tracks):
    start = time.perf_counter()
    for i in tracks:
        release = 1 + (i - 1) // 10
        for kind, value, occ, rel in [('genre', 'A', i, None), ('genre', 'B', i, None), ('recording_year', '2005', i, None), ('isrc', 'X', i, None),
                                      ('bpm', '120', i, None), ('release_date', '2005', None, release), ('first_release_date', '2004', None, release), ('genre', 'C', None, release)]:
            db.execute('INSERT OR IGNORE INTO descriptive_metadata(profile_id,occurrence_id,release_id,source_id,kind,value,observed_at) VALUES(1,?,?,1,?,?,?)', (occ, rel, kind, value, 't'))
    db.commit()
    return (time.perf_counter() - start) / len(tracks) * 1000

for n in (1000, 10000, 100000):
    k = 100
    tracks = [n - j * (n // k) for j in range(k)]  # spread over the whole history
    db = build(n)
    ins = insert(db, tracks)
    again = insert(db, tracks)  # repeated observation: every row is ignored
    start = time.perf_counter()
    for i in tracks:
        db.execute('DELETE FROM listens WHERE track_id=?', (i,))
    db.commit()
    delete = (time.perf_counter() - start) / k * 1000
    plain = build(n, triggers=False)
    base = insert(plain, tracks)
    start = time.perf_counter()
    for i in tracks:
        plain.execute('DELETE FROM listens WHERE track_id=?', (i,))
    plain.commit()
    base_delete = (time.perf_counter() - start) / k * 1000
    print(f'{n:>7} escutas: gravar metadados de 1 escuta {ins:8.2f} ms (repetida {again:8.2f}) sem gatilho {base:5.2f} | apagar 1 escuta {delete:8.2f} ms, sem gatilho {base_delete:5.2f}')
