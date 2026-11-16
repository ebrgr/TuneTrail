-- Conceptual contract fixture only; never apply to an application database.
PRAGMA foreign_keys=ON;
CREATE TABLE profiles(id TEXT PRIMARY KEY);
CREATE TABLE recordings(id TEXT PRIMARY KEY);
CREATE TABLE releases(id TEXT PRIMARY KEY);
CREATE TABLE occurrences(
 id TEXT PRIMARY KEY, recording_id TEXT NOT NULL REFERENCES recordings(id),
 release_id TEXT REFERENCES releases(id), position INTEGER
);
CREATE TABLE events(
 id TEXT PRIMARY KEY, profile_id TEXT NOT NULL REFERENCES profiles(id),
 occurrence_id TEXT NOT NULL REFERENCES occurrences(id), listened_at INTEGER NOT NULL,
 UNIQUE(profile_id,id)
);
CREATE TABLE receipts(
 id TEXT PRIMARY KEY, profile_id TEXT NOT NULL REFERENCES profiles(id),
 source_instance TEXT NOT NULL, external_id TEXT NOT NULL,
 status TEXT NOT NULL CHECK(status IN ('accepted','duplicate','pending_identity','pending_duplicate')),
 event_id TEXT, reason TEXT, version INTEGER NOT NULL DEFAULT 1,
 UNIQUE(profile_id,source_instance,external_id),
 FOREIGN KEY(profile_id,event_id) REFERENCES events(profile_id,id),
 CHECK((status IN ('accepted','duplicate') AND event_id IS NOT NULL)
    OR (status IN ('pending_identity','pending_duplicate') AND event_id IS NULL))
);
INSERT INTO profiles VALUES('P1'),('P2');
INSERT INTO recordings VALUES('R'),('Intro1'),('Intro2');
INSERT INTO releases VALUES('A'),('B');
INSERT INTO occurrences VALUES
 ('RA','R','A',1),('RB','R','B',1),('RU','R',NULL,NULL),
 ('IA1','Intro1','A',2),('IA2','Intro2','A',3);
INSERT INTO events VALUES
 ('E1','P1','RA',1000),('E2','P1','RB',1001),('E3','P2','RA',1000),
 ('E4','P1','RA',2000),('E5','P1','RA',2000),('E6','P1','RU',3000),
 ('E7','P1','IA1',4000),('E8','P1','IA2',4000);
INSERT INTO receipts(id,profile_id,source_instance,external_id,status,event_id,reason) VALUES
 ('C1','P1','player-1','source-1','accepted','E1',NULL),
 ('C2','P1','player-1','source-2','accepted','E2',NULL),
 ('C3','P2','player-1','source-1','accepted','E3',NULL),
 ('C4','P1','player-1','repeat-1','accepted','E4',NULL),
 ('C5','P1','player-1','repeat-2','accepted','E5',NULL),
 ('C6','P1','player-1','unknown-release','accepted','E6',NULL),
 ('C7','P1','player-1','intro-1','accepted','E7',NULL),
 ('C8','P1','player-1','intro-2','accepted','E8',NULL),
 ('C9','P1','import-1','digest:ordinal-9','pending_identity',NULL,'alias_ambiguous'),
 ('C10','P1','import-1','digest:ordinal-10','pending_identity',NULL,'recording_conflict'),
 ('C11','P1','api-1','receipt-without-external-id-1','pending_duplicate',NULL,'indistinguishable_delivery'),
 ('C12','P1','another-source','confirmed-same-event','duplicate','E4','review_equal');
