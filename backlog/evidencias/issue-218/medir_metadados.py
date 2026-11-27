#!/usr/bin/env python3
"""Measure incremental SQLite bytes using synthetic observations and the v25 indexes."""
import os
import sqlite3
import tempfile
from pathlib import Path

schema = Path('db/migrations_sqlite/024_enrichment.sql').read_text()
schema = schema[schema.index('CREATE TABLE metadata_values'):schema.index('-- Defensive guards')]
schema = schema.replace("'artist_origin_country'", "'artist_origin_country','isrc','bpm','first_release_date','first_recording_date'", 1)
N = 10000
with tempfile.TemporaryDirectory() as temp:
    path = Path(temp) / 'synthetic.sqlite'
    db = sqlite3.connect(path)
    db.executescript(schema)
    db.execute('VACUUM')
    baseline = os.path.getsize(path)
    for i in range(1, N + 1):
        event = f'{i:08x}-0000-4000-8000-{i:012x}'
        recording = 1 + (i - 1) % 1000
        release = 1 + (i - 1) % 100
        values = [('genre', 'Rock', 'recording'), ('genre', 'Alternative Rock', 'recording'),
                  ('genre', 'Indie Rock', 'recording'), ('recording_year', '2005', 'recording'),
                  ('release_date', '2005-10-04', 'release'), ('first_release_date', '2005-09-30', 'release'),
                  ('isrc', 'USABC0500001', 'recording'), ('bpm', '120', 'recording')]
        for kind, value, entity in values:
            rec = recording if entity == 'recording' else None
            rel = release if entity == 'release' else None
            occ = recording if entity == 'recording' else None
            db.execute('INSERT INTO metadata_values(profile_id,recording_id,release_id,occurrence_id,event_id,kind,value,instance,source,observed_at) VALUES (?,?,?,?,?,?,?,?,?,?)',
                       (1, rec, rel, occ, event, kind, value, 'a' * 64, 'navidrome-subsonic', '2026-11-27T21:30:00Z'))
    db.commit()
    db.execute('VACUUM')
    size = os.path.getsize(path)
    print(f'escutas={N}, valores_por_escuta=8, bytes_iniciais={baseline}, bytes_finais={size}, bytes_por_escuta={(size-baseline)/N:.2f}')
    db.close()
