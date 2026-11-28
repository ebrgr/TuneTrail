"""Trigger cost on a skewed library with foreign keys enforced, as the application runs."""
import sqlite3, random, time, sys, re
# Usage: python3 medir_gatilhos_fk.py [schema.sql]
schema=open(sys.argv[1] if len(sys.argv)>1 else __file__.rsplit('/',1)[0]+'/esquema-gatilhos.sql').read()
def build(n, triggers=True, retries=False, same_order=False):
    r=random.Random(7); db=sqlite3.connect(':memory:')
    s=schema if triggers else re.sub(r'CREATE TRIGGER.*?^END;\n','',schema,flags=re.S|re.M)
    db.executescript("PRAGMA foreign_keys=ON;CREATE TABLE artists(id INTEGER PRIMARY KEY);CREATE TABLE releases(id INTEGER PRIMARY KEY);CREATE TABLE recordings(id INTEGER PRIMARY KEY);CREATE TABLE listening_profiles(id INTEGER PRIMARY KEY);CREATE TABLE event_origins(origin TEXT PRIMARY KEY);"+s)
    na,nr=max(2,n//50),max(2,n//10)
    db.execute('INSERT INTO recordings VALUES(1)'); db.execute('INSERT INTO listening_profiles VALUES(1)'); db.execute("INSERT INTO event_origins VALUES('legacy')")
    db.executemany('INSERT INTO artists VALUES(?)',((i,) for i in range(1,na+1))); db.executemany('INSERT INTO releases VALUES(?)',((i,) for i in range(1,nr+1)))
    rel={i:r.randint(1,nr) for i in range(1,n+1)}
    # one artist holds 20% of the tracks
    art={i:1 if r.random()<.2 else r.randint(2,na) for i in range(1,n+1)}
    db.executemany('INSERT INTO tracks(id,release_id) VALUES(?,?)',((i,rel[i]) for i in rel))
    db.executemany('INSERT INTO occurrences(id,recording_id,release_id) VALUES(?,1,?)',((i,rel[i]) for i in rel))
    db.executemany('INSERT INTO artist_tracks(artist_id,track_id) VALUES(?,?)',((art[i],i) for i in rel))
    db.executemany("INSERT INTO occurrence_credits(occurrence_id,position,name,artist_id) VALUES(?,0,'A',?)",((i,art[i]) for i in rel))
    order=list(rel); r.shuffle(order)
    db.executemany('INSERT INTO listens(profile_id,track_id,listened_at,event_id) VALUES(1,?,?,?)',((t,k,str(k)) for k,t in enumerate(order)))
    db.execute("INSERT INTO metadata_sources VALUES(1,'s','i')")
    q='INSERT OR IGNORE INTO descriptive_metadata(profile_id,{},source_id,kind,value,observed_at) VALUES(1,?,1,?,?,?)'
    db.executemany(q.format('occurrence_id'),((i,'genre','g','t') for i in rel))
    db.executemany(q.format('release_id'),((x,'genre','g','t') for x in set(rel.values())))
    db.executemany(q.format('artist_id'),((x,'genre','g','t') for x in set(art.values())))
    if retries: db.executemany('INSERT INTO metadata_retries VALUES(1,?,1)',((i,) for i in rel))
    db.commit(); return db,order
def avg(db,sql,ids):
    t=time.perf_counter()
    for i in ids: db.execute(sql,(i,))
    db.commit(); return (time.perf_counter()-t)/len(ids)*1000
for n in (1000,10000,100000):
    for label,kw in (('com gatilhos',{}),('sem gatilhos',{'triggers':False}),('com gatilhos + marcadores',{'retries':True})):
        db,order=build(n,**kw); pick=order[:100]
        listen=avg(db,'DELETE FROM listens WHERE track_id=?',pick)
        db,order=build(n,**kw); occ=avg(db,'DELETE FROM occurrences WHERE id=?',order[:100])
        db,order=build(n,**kw); t=time.perf_counter(); db.execute('DELETE FROM listens WHERE profile_id=1'); db.commit(); bulk=time.perf_counter()-t
        left=db.execute('SELECT count(*) FROM descriptive_metadata').fetchone()[0]
        db,order=build(n,**kw); t=time.perf_counter(); db.execute('DELETE FROM listening_profiles WHERE id=1'); db.commit(); prof=time.perf_counter()-t
        print(f'{n:>7} {label:<26} escuta {listen:.3f} ms | ocorrência {occ:.3f} ms | apagar escutas {bulk:.2f} s (sobram {left}) | apagar perfil {prof:.2f} s',flush=True)
