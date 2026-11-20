package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/ebrgr/TuneTrail/internal/ingest"
	"github.com/stretchr/testify/require"
)

// TestIngestActivationUpgrade verifies the v8 snapshot and the v9 receipt boundary.
func TestIngestActivationUpgrade(t *testing.T) {
	s, _, source := migrationDB(t, 8)
	seedIdentityHistory(t, s)
	execSQL(t, s, `UPDATE ingest_events SET state='pending_duplicate' WHERE receipt_id='shadow'`)
	before := legacyDigests(t, s.db)
	dir := t.TempDir()
	target := filepath.Join(dir, "tunetrail.db")
	require.NoError(t, copyDatabase(source, target))
	_, err := bootstrap(dir, false)
	require.NoError(t, err)

	backups, err := filepath.Glob(target + ".pre-ingest-activation-*.bak")
	require.NoError(t, err)
	require.Len(t, backups, 1)
	backup, err := openExisting(backups[0])
	require.NoError(t, err)
	require.NoError(t, checkDatabase(backup))
	version, err := schemaVersion(backup)
	require.NoError(t, err)
	require.Equal(t, 8, version)
	require.Equal(t, before, legacyDigests(t, backup))
	require.NoError(t, backup.Close())

	conn, err := openExisting(target)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, conn.Close()) })
	s = &Sqlite{db: conn}
	require.NoError(t, checkDatabase(conn))
	version, err = schemaVersion(conn)
	require.NoError(t, err)
	require.Equal(t, currentSchemaVersion, version)
	for name, digest := range before {
		if name == "event_origins" {
			// Migration 010 appends origins while keeping the legacy one.
			require.Equal(t, 7, scalar(t, s, `SELECT count(*) FROM event_origins`))
			require.Equal(t, 1, scalar(t, s, `SELECT count(*) FROM event_origins WHERE origin='legacy'`))
			continue
		}
		require.Equal(t, digest, digestTable(t, conn, name, digest.columns), name)
	}
	require.Equal(t, 1, scalar(t, s, `SELECT count(*) FROM ingest_events WHERE mode='shadow'`))
	require.Zero(t, scalar(t, s, `SELECT count(*) FROM profile_ingest_flags`))
	execSQL(t, s, `INSERT INTO profile_ingest_flags(profile_id) VALUES(1)`)
	require.Equal(t, 1, scalar(t, s, `SELECT count(*) FROM profile_ingest_flags WHERE active=0 AND review=0`))

	for _, tc := range []struct{ query, guard string }{
		{`UPDATE profile_ingest_flags SET review=2`, "CHECK constraint failed: review IN (0,1)"},
		{`UPDATE ingest_events SET mode='active' WHERE receipt_id='shadow'`, "Receipt mode is immutable"},
		{`UPDATE ingest_events SET state='accepted',event_id=(SELECT event_id FROM listens WHERE rowid=10)`, "shadow receipts cannot materialize decisions"},
		{`INSERT INTO ingest_revisions VALUES(1,'shadow',1,'fixture','review','["fixture"]','{}','1',0)`, "Receipt revision requires an active receipt"},
		{`INSERT INTO outbox(id,profile_id,destination_id,event_id,operation,delivery_key,created_at)
		  SELECT 'blocked',1,'source',event_id,'listen','blocked',0 FROM listens WHERE rowid=10`, "Outbox intent requires an accepted active receipt"},
	} {
		_, err := conn.Exec(tc.query)
		require.ErrorContains(t, err, tc.guard, tc.query)
	}

	const insert = `INSERT INTO ingest_events(receipt_id,profile_id,source_instance_id,schema_version,source_type,adapter_version,
		batch_id,item_ordinal,operation,received_at,time_precision,time_semantics,original_snapshot,snapshot_digest,digest_version,mode,state)
		VALUES('admitted',1,'source',1,'test','1','batch',0,'submit',0,'second','start','{}','digest','1',?,'pending_duplicate')`
	_, err = conn.Exec(insert, "unknown")
	require.ErrorContains(t, err, "CHECK constraint failed: mode IN ('shadow','active')")
	execSQL(t, s, insert, "active")
	_, err = conn.Exec(`UPDATE ingest_events SET mode='shadow' WHERE receipt_id='admitted'`)
	require.ErrorContains(t, err, "Receipt mode is immutable")

	const queue = `SELECT receipt_id FROM ingest_events WHERE mode='active' AND closed_at IS NULL AND state='pending_duplicate' ORDER BY next_attempt_at,received_at`
	var receiptID, plan string
	var id, parent, unused int
	require.NoError(t, conn.QueryRow(queue).Scan(&receiptID))
	require.Equal(t, "admitted", receiptID)
	require.NoError(t, conn.QueryRow("EXPLAIN QUERY PLAN "+queue).Scan(&id, &parent, &unused, &plan))
	require.Contains(t, plan, "idx_ingest_active_queue")

	// Resolution and outbox writes succeed with admission still disabled.
	execSQL(t, s, `UPDATE ingest_events SET state='accepted',identity_status='resolved',resolution_version=1,
		event_id=(SELECT event_id FROM listens WHERE rowid=10),occurrence_id=(SELECT track_id FROM listens WHERE rowid=10),
		recording_id=(SELECT o.recording_id FROM listens l JOIN occurrences o ON o.id=l.track_id WHERE l.rowid=10),
		closed_at=1 WHERE receipt_id='admitted';
		INSERT INTO ingest_revisions VALUES(1,'admitted',1,'fixture','different','["fixture"]','{}','1',1);
		INSERT INTO outbox(id,profile_id,destination_id,event_id,operation,delivery_key,created_at,state,closed_at,result)
		SELECT 'o',1,'source',event_id,'listen','key',1,'discarded',1,'{"reason":"pre_082_best_effort"}' FROM listens WHERE rowid=10`)
	require.Zero(t, scalar(t, s, `SELECT count(*) FROM (`+queue+`)`))
	require.NoError(t, checkIntegrity(conn))
}

// TestResolveIngestLegacyCollision covers both review and the legacy fallback.
func TestResolveIngestLegacyCollision(t *testing.T) {
	for _, review := range []bool{false, true} {
		t.Run(fmt.Sprint(review), func(t *testing.T) {
			s, _, _ := migrationDB(t, currentSchemaVersion)
			seedIdentityHistory(t, s)
			execSQL(t, s, `INSERT INTO profile_ingest_flags(profile_id,review) VALUES(1,?)`, review)
			execSQL(t, s, `INSERT INTO ingest_events(receipt_id,profile_id,source_instance_id,schema_version,source_type,adapter_version,batch_id,
				item_ordinal,operation,received_at,time_precision,time_semantics,original_snapshot,snapshot_digest,digest_version,mode,lease_owner,lease_until)
				VALUES('active',1,'source',1,'test','1','batch',0,'submit',0,'second','start','{}','digest','1','active','worker',?)`, time.Now().Add(time.Minute).Unix())
			receipt := claimedReceipt{id: "active", lease: "worker", profile: 1, track: 1, unix: 42}
			require.NoError(t, s.finishIngest(context.Background(), receipt))
			require.Equal(t, 3, scalar(t, s, `SELECT count(*) FROM listens`))
			require.Equal(t, 1, scalar(t, s, `SELECT count(*) FROM ingest_revisions WHERE receipt_id='active'`))
			var state, reason string
			require.NoError(t, s.db.QueryRow(`SELECT state,json_extract(reason_codes,'$[0]') FROM ingest_events WHERE receipt_id='active'`).Scan(&state, &reason))
			if review {
				require.Equal(t, "pending_duplicate", state)
				require.Equal(t, "legacy_identity_collision", reason)
			} else {
				require.Equal(t, "duplicate", state)
				require.Equal(t, "review_deferred", reason)
			}
			require.ErrorIs(t, s.finishIngest(context.Background(), receipt), sql.ErrNoRows)
		})
	}
}

// TestIngestAdmissionModeReplay exercises all shared key indexes across mode switches.
func TestIngestAdmissionModeReplay(t *testing.T) {
	ctx := context.Background()
	for _, key := range []string{"external", "client", "file"} {
		t.Run(key, func(t *testing.T) {
			s, _, _ := migrationDB(t, currentSchemaVersion)
			seedIdentityHistory(t, s)
			batch := ingest.Batch{ProfileID: 1, Source: "adapter", Operation: "single", ReceivedAt: time.Now(),
				Items: []ingest.Item{{Snapshot: ingest.Snapshot{Artist: "Artist", Title: "Track"}}}}
			switch key {
			case "external":
				batch.Items[0].ExternalNamespace, batch.Items[0].ExternalID = "provider", "one"
			case "client":
				batch.ClientKey = "one"
			case "file":
				batch.FileDigest, batch.FileFormat = ingest.Digest([]byte("one")), "fixture"
			}
			shadow, err := s.ReceiveBatch(ctx, batch)
			require.NoError(t, err)
			execSQL(t, s, `INSERT INTO profile_ingest_flags(profile_id,active) VALUES(1,1)`)
			activeMode, replay, err := s.AdmitIngest(ctx, batch, true, nil, nil)
			require.NoError(t, err)
			require.True(t, activeMode)
			require.False(t, replay[0].Replay)
			require.NotEqual(t, shadow[0].ReceiptID, replay[0].ReceiptID)
			require.Zero(t, scalar(t, s, `SELECT count(*) FROM ingest_events WHERE receipt_id=?`, shadow[0].ReceiptID))
			require.Equal(t, 1, scalar(t, s, `SELECT count(*) FROM ingest_events WHERE receipt_id=? AND mode='active' AND state='received'`, replay[0].ReceiptID))

			pinned, err := s.BeginImportIngest(ctx, 1, true)
			require.NoError(t, err)
			require.True(t, pinned)
			execSQL(t, s, `UPDATE profile_ingest_flags SET active=0 WHERE profile_id=1`)
			switch key {
			case "external":
				batch.Items[0].ExternalID = "two"
			case "client":
				batch.ClientKey = "two"
			case "file":
				batch.FileDigest = ingest.Digest([]byte("two"))
			}
			activeMode, active, err := s.AdmitIngest(ctx, batch, true, &pinned, nil)
			require.NoError(t, err)
			require.True(t, activeMode)
			require.False(t, active[0].Replay)
			before := legacyDigests(t, s.db)
			batch.Items[0].Snapshot.Title = "Variant"
			replay, err = s.ReceiveBatch(ctx, batch)
			require.NoError(t, err)
			require.True(t, replay[0].Replay)
			require.Equal(t, active[0].ReceiptID, replay[0].ReceiptID)
			require.Equal(t, before, legacyDigests(t, s.db))
		})
	}
}

// TestIngestDigestUpgrade preserves v1 keyed retries while detecting changes to v1 fields.
func TestIngestDigestUpgrade(t *testing.T) {
	ctx := context.Background()
	s, _, _ := migrationDB(t, currentSchemaVersion)
	seedIdentityHistory(t, s)
	// Store the v1 bytes and batch digest exactly as the old adapter wrote them.
	legacySnapshot := `{"artist":"A","title":"T"}`
	oldDigest := ingest.Digest([]byte(legacySnapshot))
	batchDigest := ingest.Digest([]byte(fmt.Sprintf(`["%s"]`, oldDigest)))
	execSQL(t, s, `INSERT INTO integrations(id,profile_id,source_type,created_at) VALUES('listenbrainz:1',1,'listenbrainz',0);
		INSERT INTO ingest_events(receipt_id,profile_id,source_instance_id,schema_version,source_type,adapter_version,batch_id,item_ordinal,
			operation,received_at,time_precision,time_semantics,original_snapshot,snapshot_digest,digest_version,client_idempotency_key,batch_digest)
		VALUES('v1',1,'listenbrainz:1',1,'listenbrainz','1','batch',0,'single',0,'estimated','unknown',?,?,'sha256-json-v1','key',?)`,
		legacySnapshot, oldDigest, batchDigest)
	batch := ingest.Batch{ProfileID: 1, Source: ingest.ShadowSourceListenBrainz, Operation: "single", ClientKey: "key", ReceivedAt: time.Now(),
		Items: []ingest.Item{{Snapshot: ingest.Snapshot{Artist: "A", Title: "T",
			ArtistMbidMappings: []ingest.ArtistMapping{{Name: "A", MBID: "12345678-1234-4321-abcd-000000000001"}}}}}}
	retry, err := s.ReceiveBatch(ctx, batch)
	require.NoError(t, err)
	require.Equal(t, "v1", retry[0].ReceiptID)
	require.Equal(t, "same", retry[0].Decision)
	require.Zero(t, scalar(t, s, `SELECT count(*) FROM ingest_observations WHERE receipt_id='v1'`))
	batch.Items[0].Snapshot.Title = "Changed"
	retry, err = s.ReceiveBatch(ctx, batch)
	require.NoError(t, err)
	require.Equal(t, "review", retry[0].Decision)
	require.Equal(t, 1, scalar(t, s, `SELECT count(*) FROM ingest_observations WHERE receipt_id='v1' AND digest_version=?`, ingest.DigestVersion))
}

// TestProfileDeleteClosesActiveIngest closes only work matching the deleted tracks.
func TestProfileDeleteClosesActiveIngest(t *testing.T) {
	ctx := context.Background()
	s, _, _ := migrationDB(t, currentSchemaVersion)
	seedIdentityHistory(t, s)
	execSQL(t, s, `INSERT INTO tracks(id,release_id,duration) VALUES(71,1,100);
		INSERT INTO track_aliases VALUES(71,'U','Canonical',1)`)
	at := time.Unix(100, 0)
	var related, unrelated string
	for _, profile := range []int32{1, 2} {
		batch := ingest.Batch{ProfileID: profile, Source: "file", Operation: "import", ReceivedAt: time.Now(),
			Items: []ingest.Item{
				{Ordinal: 0, ListenedAt: at, Snapshot: ingest.Snapshot{Artist: "different", Title: "different", CatalogTrackID: 70}},
				{Ordinal: 1, ListenedAt: at, Snapshot: ingest.Snapshot{Artist: " ARTIST ", Title: "opening"}},
				{Ordinal: 2, ListenedAt: at, Snapshot: ingest.Snapshot{Artist: "B", Title: "U", CatalogTrackID: 71}},
				{Ordinal: 3, ListenedAt: at, Snapshot: ingest.Snapshot{Artist: "other artist", Title: "Intro"}},
			}}
		pinned := true
		_, results, err := s.AdmitIngest(ctx, batch, true, &pinned, nil)
		require.NoError(t, err)
		if profile == 1 {
			related, unrelated = results[0].ReceiptID, results[2].ReceiptID
		}
	}
	execSQL(t, s, `UPDATE ingest_events SET state='retryable',lease_owner='old',lease_until=unixepoch()+60
		WHERE receipt_id=?`, related)
	plan, err := s.PreviewProfileDelete(ctx, 1, "track", 1)
	require.NoError(t, err)
	require.EqualValues(t, 2, plan.Count)
	deleted, err := s.ApplyProfileDelete(ctx, plan)
	require.NoError(t, err)
	require.Equal(t, plan.Count, deleted)
	require.Equal(t, 2, scalar(t, s, `SELECT count(*) FROM ingest_events WHERE profile_id=1 AND state='invalid'
		AND closed_at IS NOT NULL AND lease_owner IS NULL AND resolution_version=1`))
	require.Equal(t, 2, scalar(t, s, `SELECT count(*) FROM ingest_revisions WHERE profile_id=1 AND decision='invalid'`))
	require.Equal(t, 4, scalar(t, s, `SELECT count(*) FROM ingest_events WHERE profile_id=2 AND closed_at IS NULL`))
	require.Equal(t, 2, scalar(t, s, `SELECT count(*) FROM ingest_events WHERE profile_id=1 AND closed_at IS NULL`))
	require.ErrorIs(t, s.finishIngest(ctx, claimedReceipt{id: related, profile: 1, track: 70, lease: "old", unix: 100}), sql.ErrNoRows)
	// Leave only the unrelated native receipt eligible for this resolver pass.
	execSQL(t, s, `UPDATE ingest_events SET next_attempt_at=unixepoch()+60
		WHERE closed_at IS NULL AND receipt_id<>?`, unrelated)
	found, err := s.ResolveNextIngest(ctx, nil)
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, 1, scalar(t, s, `SELECT count(*) FROM ingest_events e JOIN listens l ON l.event_id=e.event_id
		WHERE e.receipt_id=? AND e.state='accepted' AND l.track_id=71`, unrelated))
}

// TestExpiredIngestLeaseCanBeReclaimed checks CAS while the new claim is still open.
func TestExpiredIngestLeaseCanBeReclaimed(t *testing.T) {
	ctx := context.Background()
	s, _, path := migrationDB(t, currentSchemaVersion)
	seedIdentityHistory(t, s)
	pinned := true
	_, results, err := s.AdmitIngest(ctx, ingest.Batch{
		ProfileID: 1, Source: "file", Operation: "import", ReceivedAt: time.Now(),
		Items: []ingest.Item{{ListenedAt: time.Unix(100, 0), Snapshot: ingest.Snapshot{Artist: "A", Title: "T", CatalogTrackID: 1}}},
	}, true, &pinned, nil)
	require.NoError(t, err)
	first, _, _, err := s.claimNextIngest(ctx)
	require.NoError(t, err)
	require.Equal(t, results[0].ReceiptID, first.id)

	conn, err := openExisting(path)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, conn.Close()) })
	worker := &Sqlite{db: conn}
	unavailable, _, _, err := worker.claimNextIngest(ctx)
	require.NoError(t, err)
	require.Empty(t, unavailable.id)
	execSQL(t, s, `UPDATE ingest_events SET lease_until=unixepoch()-1 WHERE receipt_id=?`, first.id)

	second, _, _, err := worker.claimNextIngest(ctx)
	require.NoError(t, err)
	require.Equal(t, first.id, second.id)
	require.NotEqual(t, first.lease, second.lease)
	require.Equal(t, first.work+1, second.work)
	first.track, first.unix = 1, 100
	second.track, second.unix = 1, 100
	require.ErrorIs(t, s.finishIngest(ctx, first), sql.ErrNoRows)
	require.NoError(t, worker.finishIngest(ctx, second))
	require.Equal(t, 1, scalar(t, s, `SELECT count(*) FROM listens WHERE profile_id=1 AND listened_at=100`))
}

// TestIngestReclaimsShadowBatch verifies atomic replacement, preservation, and resolution.
func TestIngestReclaimsShadowBatch(t *testing.T) {
	ctx := context.Background()
	s, _, _ := migrationDB(t, currentSchemaVersion)
	seedIdentityHistory(t, s)
	at := time.Unix(900, 0)
	snapshot := ingest.Snapshot{Artist: "Artist", Title: "Intro", CatalogTrackID: 1}
	batch := ingest.Batch{ProfileID: 1, Source: ingest.ShadowSourceFile, Operation: "import", ReceivedAt: at,
		FileFormat: "koito", FileDigest: ingest.Digest([]byte("fixture")), Items: []ingest.Item{
			{Ordinal: 0, Snapshot: snapshot}, // The shadow estimated time must survive a later retry.
			{Ordinal: 1, Snapshot: snapshot, ListenedAt: time.Unix(42, 0)},
			{Ordinal: 2, Snapshot: snapshot, ListenedAt: time.Unix(42, 0)},
		}}
	shadow, err := s.ReceiveBatch(ctx, batch)
	require.NoError(t, err)
	other := batch
	other.ProfileID = 2
	otherShadow, err := s.ReceiveBatch(ctx, other)
	require.NoError(t, err)

	// Seed the linked compatibility case; normal v9 shadow writers cannot create it.
	execSQL(t, s, `DROP TRIGGER trg_ingest_event_mode_update;
		UPDATE ingest_events SET state='accepted',duplicate_decision='different',event_id=(SELECT event_id FROM listens WHERE rowid=10)
		WHERE receipt_id=?`, shadow[2].ReceiptID)
	execSQL(t, s, `INSERT INTO ingest_observations
		(id,profile_id,receipt_id,schema_version,snapshot_digest,digest_version,original_snapshot,received_at)
		VALUES('retired-variant',1,?,1,'variant','1','{}',0);
		INSERT INTO ingest_event_details(profile_id,receipt_id,resolution_version,kind,payload)
		VALUES(1,?,0,'diagnostic','{}')`, shadow[0].ReceiptID, shadow[0].ReceiptID)
	execSQL(t, s, `INSERT INTO profile_ingest_flags(profile_id,active) VALUES(1,1)`)
	batch.Items = append(batch.Items, ingest.Item{Ordinal: 3, Snapshot: snapshot, ListenedAt: time.Unix(950, 0)})
	batch.ReceivedAt = at.Add(time.Hour)

	execSQL(t, s, `CREATE TABLE reclaim_commit_fk(id INTEGER REFERENCES listening_profiles(id) DEFERRABLE INITIALLY DEFERRED);
		CREATE TRIGGER reclaim_commit_failure AFTER INSERT ON ingest_events WHEN NEW.mode='active' AND NEW.item_ordinal=3
		BEGIN INSERT INTO reclaim_commit_fk VALUES(-1); END`)
	_, results, err := s.AdmitIngest(ctx, batch, true, nil, nil)
	var unavailable *ingest.ErrUnavailable
	require.ErrorAs(t, err, &unavailable)
	require.Empty(t, results)
	require.Equal(t, 3, scalar(t, s, `SELECT count(*) FROM ingest_events WHERE profile_id=1 AND file_format='koito' AND mode='shadow'`))
	require.Equal(t, 1, scalar(t, s, `SELECT count(*) FROM ingest_observations WHERE id='retired-variant'`))
	execSQL(t, s, `DROP TRIGGER reclaim_commit_failure`)

	active, results, err := s.AdmitIngest(ctx, batch, true, nil, nil)
	require.NoError(t, err)
	require.True(t, active)
	require.Len(t, results, 4)
	for _, i := range []int{0, 1, 3} {
		require.False(t, results[i].Replay)
		require.True(t, results[i].Pending)
	}
	require.Equal(t, ingest.Result{ReceiptID: shadow[2].ReceiptID, Decision: "different", Replay: true}, results[2])
	require.Zero(t, scalar(t, s, `SELECT count(*) FROM ingest_events WHERE receipt_id IN (?,?)`, shadow[0].ReceiptID, shadow[1].ReceiptID))
	require.Zero(t, scalar(t, s, `SELECT count(*) FROM ingest_observations WHERE id='retired-variant'`))
	require.Zero(t, scalar(t, s, `SELECT count(*) FROM ingest_event_details WHERE receipt_id=?`, shadow[0].ReceiptID))
	require.Equal(t, 3, scalar(t, s, `SELECT count(*) FROM ingest_events WHERE profile_id=2 AND mode='shadow'
		AND receipt_id IN (?,?,?)`, otherShadow[0].ReceiptID, otherShadow[1].ReceiptID, otherShadow[2].ReceiptID))

	for range 3 {
		found, err := s.ResolveNextIngest(ctx, nil)
		require.NoError(t, err)
		require.True(t, found)
	}
	require.Equal(t, 2, scalar(t, s, `SELECT count(*) FROM ingest_events WHERE receipt_id IN (?,?) AND state='accepted'`, results[0].ReceiptID, results[3].ReceiptID))
	require.Equal(t, 1, scalar(t, s, `SELECT count(*) FROM ingest_events WHERE receipt_id=? AND state='duplicate'
		AND event_id=(SELECT event_id FROM listens WHERE rowid=10)`, results[1].ReceiptID))
	require.Equal(t, 1, scalar(t, s, `SELECT count(*) FROM ingest_events WHERE receipt_id=? AND mode='shadow' AND state='accepted'
		AND event_id=(SELECT event_id FROM listens WHERE rowid=10) AND resolution_version=0`, results[2].ReceiptID))
	require.Equal(t, 5, scalar(t, s, `SELECT count(*) FROM listens`))
	require.Equal(t, 1, scalar(t, s, `SELECT count(*) FROM listens WHERE profile_id=1 AND track_id=1 AND listened_at=900`))
}
