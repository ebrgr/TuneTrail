package sqlite

import (
	"testing"
	"time"

	"github.com/ebrgr/TuneTrail/internal/db"
	"github.com/stretchr/testify/require"
)

// TestRecordingMergeLargeFixture measures atomic reconciliation with active receipt history.
func TestRecordingMergeLargeFixture(t *testing.T) {
	const occurrences = 5000
	s := convertedOccurrenceFixture(t)
	execSQL(t, s, `WITH RECURSIVE seq(n) AS (SELECT 1 UNION ALL SELECT n+1 FROM seq WHERE n<?)
		INSERT INTO occurrences(id,recording_id,presented_title)
		SELECT 100+n,(SELECT recording_id FROM occurrences WHERE id=70),'Edition '||n FROM seq`, occurrences)
	execSQL(t, s, `INSERT INTO listens(profile_id,track_id,listened_at,event_id,origin)
		SELECT 1,id,1000,id||':first','listenbrainz_api' FROM occurrences WHERE id>100
		UNION ALL SELECT 1,id,1000,id||':second','listenbrainz_api' FROM occurrences WHERE id>100;
		INSERT INTO integrations(id,profile_id,source_type,created_at) VALUES('bench',1,'listenbrainz',0);
		INSERT INTO ingest_events(receipt_id,profile_id,source_instance_id,schema_version,source_type,adapter_version,batch_id,item_ordinal,
			operation,received_at,time_precision,time_semantics,original_snapshot,snapshot_digest,digest_version,mode,resolver_variant,
			state,identity_status,duplicate_decision,event_id,occurrence_id,recording_id,resolution_version)
		SELECT 'bench:'||l.event_id,1,'bench',1,'listenbrainz','1','bench',l.rowid,'single',0,'second','start','{}','digest','1',
			'active','occurrence','accepted','resolved','different',l.event_id,o.id,o.recording_id,1
		FROM listens l JOIN occurrences o ON o.id=l.track_id WHERE o.id>100;
		INSERT INTO ingest_revisions SELECT profile_id,receipt_id,1,'fixture','different','["fixture"]','{}','fixture-v1',0
			FROM ingest_events WHERE source_instance_id='bench';
		ANALYZE`)
	start := time.Now()
	require.NoError(t, s.MergeTracks(t.Context(), 70, 1))
	t.Logf("merge %d occurrences and %d accepted receipts: %s", occurrences+1, occurrences*2, time.Since(start))
	require.Equal(t, occurrences*2, scalar(t, s, `SELECT count(*) FROM ingest_events e JOIN occurrences o ON o.id=e.occurrence_id
		JOIN ingest_revisions r ON (r.profile_id,r.receipt_id,r.resolution_version)=(e.profile_id,e.receipt_id,e.resolution_version)
		WHERE e.source_instance_id='bench' AND r.resolution_version=2 AND e.recording_id=o.recording_id
		AND r.rule_version='recording-reconciliation-v1'`))
	start = time.Now()
	require.NoError(t, s.UndoTrackMerge(t.Context(), 70, 1))
	t.Logf("undo: %s", time.Since(start))
	require.Equal(t, occurrences*2, scalar(t, s, `SELECT count(*) FROM ingest_revisions WHERE resolution_version=3 AND rule_version='recording-reconciliation-v1'`))
}

// TestRecordingUndoRetention checks the deadline and pruning after an expired request.
func TestRecordingUndoRetention(t *testing.T) {
	s := mergeFixture(t)
	ctx := t.Context()
	start := time.Now().Unix()
	require.NoError(t, s.MergeTracks(ctx, 1, 2))
	require.Equal(t, 1, scalar(t, s, `SELECT count(*) FROM recording_moves WHERE stage=1 AND expires_at>=?`, start+recordingUndoSeconds))
	execSQL(t, s, `UPDATE recording_moves SET expires_at=unixepoch()-1`)
	require.ErrorIs(t, s.UndoTrackMerge(ctx, 1, 2), db.ErrNotFound)
	require.Zero(t, scalar(t, s, `SELECT count(*) FROM recording_moves`))
	require.Equal(t, 1, scalar(t, s, `SELECT count(DISTINCT recording_id) FROM occurrences`))
	require.Equal(t, 4, scalar(t, s, `SELECT count(*) FROM listens`))
}

// TestArchiveBoundaryUpgrade checks existing v14 history and undo entries through v15.
func TestArchiveBoundaryUpgrade(t *testing.T) {
	s, provider, _ := migrationDB(t, 14)
	seedIdentityHistory(t, s)
	execSQL(t, s, `INSERT INTO recording_moves(source_track,destination_track,occurrence_id,original_recording,merged_recording,stage)
		SELECT 70,1,70,(SELECT recording_id FROM occurrences WHERE id=70),
			(SELECT recording_id FROM occurrences WHERE id=1),1;
		UPDATE occurrences SET recording_id=(SELECT recording_id FROM occurrences WHERE id=1) WHERE id=70`)
	start := time.Now().Unix()
	_, err := provider.UpTo(t.Context(), 20)
	require.NoError(t, err)
	require.Equal(t, 3, scalar(t, s, `SELECT count(*) FROM listen_export_order JOIN listens USING(event_id)`))
	require.Equal(t, 1, scalar(t, s, `SELECT count(*) FROM recording_moves WHERE stage=1 AND expires_at>=?`, start+recordingUndoSeconds))
	require.NoError(t, s.UndoTrackMerge(t.Context(), 70, 1))
	require.Equal(t, 2, scalar(t, s, `SELECT count(DISTINCT recording_id) FROM occurrences`))
	assertForeignKeys(t, s)
}
