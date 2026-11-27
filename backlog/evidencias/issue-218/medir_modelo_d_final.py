"""Measure final tables and transient successful runs without vacuum after deletion."""
import os
import sqlite3
import tempfile
from pathlib import Path

N = 10000
migration = Path('db/migrations_sqlite/026_descriptive_metadata.sql').read_text().split('-- A profile may only')[0]
runs = Path('db/migrations_sqlite/025_enrichment_registry.sql').read_text().split('CREATE TABLE enrichment_provider_runs',1)[1].split('-- +goose Down',1)[0]
runs = 'CREATE TABLE enrichment_provider_runs' + runs
values = [('genre', 'Rock', 'track'), ('genre', 'Alternative Rock', 'track'),
          ('genre', 'Indie Rock', 'track'), ('recording_year', '2005', 'track'),
          ('release_date', '2005-10-04', 'release'), ('first_release_date', '2005-09-30', 'release'),
          ('isrc', 'USABC0500001', 'track'), ('bpm', '120', 'track')]
for repeats in (1, 3, 10):
    with tempfile.TemporaryDirectory() as temp:
        path = Path(temp) / 'measure.sqlite'
        db = sqlite3.connect(path)
        db.executescript(migration + runs)
        db.execute('VACUUM')
        base = os.path.getsize(path)
        db.execute('INSERT INTO metadata_sources VALUES (1,?,?)', ('navidrome-subsonic', 'a' * 64))
        tracks = N // repeats
        releases = max(1, tracks // 10)
        for i in range(N):
            receipt = f'{i:08x}-0000-4000-8000-{i:012x}'
            for chain in ('identity',):
                db.execute('INSERT INTO enrichment_provider_runs VALUES(?,?,?,?,?,?)', (receipt, 'navidrome-subsonic', chain, 'found', 'a'*64, 1795815000))
                db.commit()
            track, release = 1 + i % tracks, 1 + i % releases
            for kind, value, entity in values:
                db.execute('INSERT OR IGNORE INTO descriptive_metadata(profile_id,occurrence_id,release_id,source_id,kind,value,observed_at) VALUES (1,?,?,?,?,?,?)',
                    (track if entity == 'track' else None, release if entity == 'release' else None, 1, kind, value, '2026-11-27T21:30:00.123456789Z'))
            db.commit()
            db.execute("DELETE FROM enrichment_provider_runs WHERE receipt_id=? AND state='found'", (receipt,))
            db.commit()
        db.commit()
        size = os.path.getsize(path)
        print(f'{repeats} escuta(s)/faixa: {(size-base)/N:.2f} bytes/escuta; {db.execute("SELECT count(*) FROM descriptive_metadata").fetchone()[0]} valores')
        db.close()
