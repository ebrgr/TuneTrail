"""Time the new triggers of migration 026 on its real tables and indexes, with fictitious data."""
import sqlite3, time, uuid, re, sys
# Usage: python3 medir_gatilhos.py [schema.sql] [--shared] [--selective]; selective keeps 16 unrelated listens during deletion.

shared = '--shared' in sys.argv[1:]
selective = '--selective' in sys.argv[1:]
paths = [arg for arg in sys.argv[1:] if arg not in ('--shared', '--selective')]
schema = open(paths[0] if paths else __file__.rsplit('/', 1)[0] + '/esquema-gatilhos.sql').read()

def entity(i):
    return ((2 if i <= 16 else 1) if selective else 1) if shared else 1 + (i - 1) // 10

def build(n, triggers=True):
    db = sqlite3.connect(':memory:')
    script = schema if triggers else re.sub(r'CREATE TRIGGER.*?^END;\n', '', schema, flags=re.S | re.M)
    db.executescript(script)
    cols = {r[1]: (r[2], r[3], r[4]) for r in db.execute('PRAGMA table_info(occurrences)')}
    need = [c for c, (t, notnull, default) in cols.items() if notnull and default is None and c not in ('id', 'release_id')]
    names = ['id', 'release_id'] + need
    values = lambda i: [i, entity(i)] + ['x' if 'TEXT' in cols[c][0].upper() else 1 for c in need]
    db.executemany('INSERT INTO tracks(id,release_id) VALUES(?,?)', ((i, entity(i)) for i in range(1, n + 1)))
    db.executemany(f'INSERT INTO occurrences({",".join(names)}) VALUES({",".join("?"*len(names))})', (values(i) for i in range(1, n + 1)))
    db.executemany('INSERT INTO artist_tracks(artist_id,track_id) VALUES(?,?)', ((entity(i), i) for i in range(1,n+1)))
    db.executemany("INSERT INTO occurrence_credits(occurrence_id,position,name,artist_id) VALUES(?,0,'Artist',?)", ((i,entity(i)) for i in range(1,n+1)))
    db.executemany('INSERT INTO listens(profile_id,track_id,listened_at,event_id) VALUES(1,?,?,?)', ((i, i, str(uuid.uuid4())) for i in range(1, n + 1)))
    db.execute("INSERT INTO metadata_sources(id,source,instance) VALUES(1,'navidrome-subsonic',?)", ('a' * 64,))
    db.commit()
    return db

def insert(db, tracks):
    start = time.perf_counter()
    for i in tracks:
        release = entity(i)
        for kind, value, occ, rel in [('genre', 'A', i, None), ('genre', 'B', i, None), ('recording_year', '2005', i, None), ('isrc', 'X', i, None),
                                      ('bpm', '120', i, None), ('release_date', '2005', None, release), ('first_release_date', '2004', None, release), ('genre', 'C', None, release)]:
            db.execute('INSERT OR IGNORE INTO descriptive_metadata(profile_id,occurrence_id,release_id,source_id,kind,value,observed_at) VALUES(1,?,?,1,?,?,?)', (occ, rel, kind, value, 't'))
        db.execute("INSERT OR IGNORE INTO descriptive_metadata(profile_id,artist_id,source_id,kind,value,observed_at) VALUES(1,?,1,'genre','A','t')", (entity(i),))
    db.commit()
    return (time.perf_counter() - start) / len(tracks) * 1000

for n in (1000, 10000, 100000):
    k = 100
    tracks = [n - j * (n // k) for j in range(k)]  # spread over the whole history
    db = build(n)
    ins = insert(db, tracks)
    again = insert(db, tracks)  # repeated observation: every row is ignored
    occ = sqlite3.connect(':memory:'); db.backup(occ)
    legacy = sqlite3.connect(':memory:'); db.backup(legacy)
    start = time.perf_counter()
    for i in tracks: occ.execute('DELETE FROM occurrences WHERE id=?', (i,))
    occ.commit(); occurrence_delete = (time.perf_counter()-start)/k*1000
    start = time.perf_counter()
    for i in tracks: legacy.execute('DELETE FROM tracks WHERE id=?', (i,))
    legacy.commit(); track_delete = (time.perf_counter()-start)/k*1000
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
    batch = build(n); insert(batch, range(1,n+1))
    start=time.perf_counter(); batch.execute('DELETE FROM listens WHERE profile_id=1' + (' AND track_id>16' if selective else '')); batch.commit()
    batch_per = (time.perf_counter()-start)/(n - 16 if selective else n)*1000
    print(f'{n:>7} escutas: inserir {ins:.3f} ms, repetir {again:.3f} ms, sem gatilho {base:.3f} ms | escuta {delete:.3f} ms, ocorrência {occurrence_delete:.3f} ms, faixa {track_delete:.3f} ms, sem gatilho {base_delete:.3f} ms | lote {batch_per:.3f} ms/escuta',flush=True)
