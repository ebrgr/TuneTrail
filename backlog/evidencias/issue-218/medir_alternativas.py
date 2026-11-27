"""Compare storage layouts for descriptive metadata with the same synthetic values as ISSUE-218."""
import os, sqlite3, tempfile, sys
from pathlib import Path

W = Path(__file__).resolve().parents[3]
full = (W / 'db/migrations_sqlite/024_enrichment.sql').read_text()
current = full[full.index('CREATE TABLE metadata_values'):full.index('-- Defensive guards')]
current = current.replace("'artist_origin_country'", "'artist_origin_country','isrc','bpm','first_release_date','first_recording_date'", 1)
N = 10000
VALUES = [('genre','Rock','rec'),('genre','Alternative Rock','rec'),('genre','Indie Rock','rec'),('recording_year','2005','rec'),
          ('release_date','2005-10-04','rel'),('first_release_date','2005-09-30','rel'),('isrc','USABC0500001','rec'),('bpm','120','rec')]

def schema(per_entity, ref):
    s = current
    if ref:
        s = s.replace('    instance TEXT NOT NULL CHECK(length(instance)>0),\n    source TEXT NOT NULL CHECK(length(source)>0),',
                      '    instance INTEGER NOT NULL,')
        s = s.replace('kind,instance,value,source', 'kind,instance,value').replace('kind,instance,value,source', 'kind,instance,value')
        s = s.replace(',source,event_id)', ',event_id)')
    if per_entity:
        cols = 'kind,instance,value' + ('' if ref else ',source')
        s = s[:s.index('CREATE UNIQUE INDEX')]
        s = s.replace('    occurrence_id INTEGER REFERENCES occurrences(id) ON DELETE CASCADE,\n', '')
        s = s.replace('    event_id TEXT NOT NULL REFERENCES listens(event_id) ON DELETE CASCADE,\n', '')
        s = s.replace('    CHECK((recording_id IS NULL)=(occurrence_id IS NULL)),\n', '')
        for e in ('artist', 'recording', 'release'):
            s += f'CREATE UNIQUE INDEX u_{e} ON metadata_values(profile_id,{e}_id,{cols}) WHERE {e}_id IS NOT NULL;\n'
            s += f'CREATE INDEX f_{e} ON metadata_values({e}_id) WHERE {e}_id IS NOT NULL;\n'
    return s

def measure(per_entity, ref, repeats):
    recordings = N // repeats
    releases = max(1, recordings // 10)
    with tempfile.TemporaryDirectory() as temp:
        path = Path(temp) / 'x.sqlite'
        db = sqlite3.connect(path)
        db.executescript(schema(per_entity, ref)); db.execute('VACUUM')
        base = os.path.getsize(path)
        for i in range(1, N + 1):
            event = f'{i:08x}-0000-4000-8000-{i:012x}'
            rec, rel = 1 + (i - 1) % recordings, 1 + (i - 1) % releases
            for kind, value, entity in VALUES:
                row = {'profile_id': 1, 'kind': kind, 'value': value, 'observed_at': '2026-11-27T21:30:00Z',
                       'recording_id': rec if entity == 'rec' else None, 'release_id': rel if entity == 'rel' else None}
                row.update({'instance': 1} if ref else {'instance': 'a' * 64, 'source': 'navidrome-subsonic'})
                if not per_entity:
                    row.update(event_id=event, occurrence_id=row['recording_id'])
                db.execute(f'INSERT OR IGNORE INTO metadata_values({",".join(row)}) VALUES ({",".join("?"*len(row))})', list(row.values()))
        db.commit(); db.execute('VACUUM')
        rows = db.execute('SELECT count(*) FROM metadata_values').fetchone()[0]
        size = os.path.getsize(path); db.close()
    return (size - base) / N, rows

for per_entity, ref, name in ((False, False, 'A por escuta (atual)'), (False, True, 'B por escuta, origem por referência'),
                              (True, False, 'C por entidade'), (True, True, 'D por entidade, origem por referência')):
    out = []
    for repeats in (1, 3, 10):
        b, rows = measure(per_entity, ref, repeats)
        out.append(f'{repeats:>2} escutas/faixa: {b:7.0f} B ({rows} linhas)')
    print(f'{name:40s}', ' | '.join(out))
