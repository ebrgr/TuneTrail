package sqlite

import (
	"fmt"
	"testing"
	"time"

	"github.com/ebrgr/TuneTrail/internal/db"
	"github.com/stretchr/testify/require"
)

// TestIngestReviewBatch checks eligibility, replay, isolation and atomic relay intent creation.
func TestIngestReviewBatch(t *testing.T) {
	s, _ := relayStoreFixture(t)
	ctx := t.Context()
	execSQL(t, s, `INSERT INTO profile_ingest_flags(profile_id,active,review) VALUES(1,1,1),(2,1,1)`)
	stamp := time.Unix(42, 0)
	for _, id := range []string{"ambiguous", "foreign", "invalid-time", "missing-occurrence", "truncated"} {
		relayReceipt(t, s, 1, id, "v9", 1, stamp)
		resolveRelayV9(t, s)
	}
	execSQL(t, s, `UPDATE ingest_events SET evidence=json_object('candidates',json_array(
		'event:'||(SELECT event_id FROM listens WHERE rowid=10),'event:'||(SELECT event_id FROM listens WHERE rowid=20)))
		WHERE receipt_id='ambiguous';
		UPDATE ingest_events SET evidence=json_object('candidate_event_id',(SELECT event_id FROM listens WHERE rowid=30))
		WHERE receipt_id='foreign';
		UPDATE ingest_events SET evidence='{}',listened_at='invalid' WHERE receipt_id='invalid-time';
		UPDATE ingest_events SET evidence='{}',occurrence_id=NULL WHERE receipt_id='missing-occurrence';
		UPDATE ingest_events SET evidence=json_object('truncated',json('true'),'candidates',
			json_array('event:'||(SELECT event_id FROM listens WHERE rowid=10))) WHERE receipt_id='truncated'`)
	for i := range ingestReviewBatchSize + 1 {
		relayReceipt(t, s, 1, fmt.Sprintf("single-%02d", i), "v9", 1, stamp)
		resolveRelayV9(t, s)
	}
	relayReceipt(t, s, 2, "other-profile", "v9", 1, stamp)
	resolveRelayV9(t, s)
	health, err := s.IngestHealth(ctx, 1)
	require.NoError(t, err)
	require.Equal(t, ingestReviewBatchSize, health.ReviewSame.Count)
	require.ErrorIs(t, s.ResolveIngestReviewBatch(ctx, 2, 1, "same", health.ReviewSame.Digest), db.ErrConflict)
	require.NoError(t, s.ResolveIngestReviewBatch(ctx, 1, 1, "same", health.ReviewSame.Digest))
	// An identical POST cannot pick up the remaining eligible receipt.
	require.ErrorIs(t, s.ResolveIngestReviewBatch(ctx, 1, 1, "same", health.ReviewSame.Digest), db.ErrConflict)
	require.Equal(t, ingestReviewBatchSize, scalar(t, s, `SELECT count(*) FROM ingest_events WHERE profile_id=1 AND state='duplicate' AND receipt_id LIKE 'single-%'`))
	require.Zero(t, scalar(t, s, `SELECT count(*) FROM outbox`))

	health, err = s.IngestHealth(ctx, 1)
	require.NoError(t, err)
	require.Equal(t, 1, health.ReviewSame.Count)
	require.Equal(t, 4, health.ReviewDifferent.Count)
	execSQL(t, s, `CREATE TRIGGER fail_second_intent BEFORE INSERT ON outbox WHEN (SELECT count(*) FROM outbox)=1
		BEGIN SELECT RAISE(ABORT,'fixture second intent failure'); END`)
	require.Error(t, s.ResolveIngestReviewBatch(ctx, 1, 2, "different", health.ReviewDifferent.Digest))
	require.Zero(t, scalar(t, s, `SELECT count(*) FROM outbox`))
	require.Equal(t, 3, scalar(t, s, `SELECT count(*) FROM listens`))
	require.Equal(t, ingestReviewBatchSize, scalar(t, s, `SELECT count(*) FROM ingest_revisions WHERE author='user'`))
	execSQL(t, s, `DROP TRIGGER fail_second_intent`)
	require.NoError(t, s.ResolveIngestReviewBatch(ctx, 1, 2, "different", health.ReviewDifferent.Digest))
	require.ErrorIs(t, s.ResolveIngestReviewBatch(ctx, 1, 2, "different", health.ReviewDifferent.Digest), db.ErrConflict)
	require.Equal(t, 7, scalar(t, s, `SELECT count(*) FROM listens`))
	require.Equal(t, 4, scalar(t, s, `SELECT count(*) FROM outbox JOIN ingest_events USING(profile_id,event_id)
		WHERE ingest_events.state='accepted' AND outbox.format_version=1`))
	require.Equal(t, 4, scalar(t, s, `SELECT count(DISTINCT event_id) FROM outbox`))
	require.Equal(t, 3, scalar(t, s, `SELECT count(*) FROM ingest_events WHERE state='pending_duplicate'
		AND receipt_id IN ('invalid-time','missing-occurrence','other-profile')`))
	require.Equal(t, 4, scalar(t, s, `SELECT count(*) FROM ingest_revisions WHERE author='user' AND decision='different'
		AND json_extract(evidence,'$.decided_by')=2`))
}

// TestIngestReviewBatchRace starts opposing actions against the same receipt on independent pools.
func TestIngestReviewBatchRace(t *testing.T) {
	s, path := relayStoreFixture(t)
	execSQL(t, s, `PRAGMA busy_timeout=5000`)
	execSQL(t, s, `INSERT INTO profile_ingest_flags(profile_id,active,review) VALUES(1,1,1)`)
	relayReceipt(t, s, 1, "race", "v9", 1, time.Unix(42, 0))
	resolveRelayV9(t, s)
	health, err := s.IngestHealth(t.Context(), 1)
	require.NoError(t, err)
	conn, err := openExisting(path)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, conn.Close()) })
	other := &Sqlite{db: conn}

	start := make(chan struct{})
	same, different := make(chan error, 1), make(chan error, 1)
	go func() {
		<-start
		same <- s.ResolveIngestReviewBatch(t.Context(), 1, 1, "same", health.ReviewSame.Digest)
	}()
	go func() {
		<-start
		different <- other.ResolveIngestReviewBatch(t.Context(), 1, 2, "different", health.ReviewDifferent.Digest)
	}()
	close(start)
	sameErr, differentErr := <-same, <-different
	wantOutbox := 0
	if sameErr == nil {
		require.ErrorIs(t, differentErr, db.ErrConflict)
	} else {
		require.ErrorIs(t, sameErr, db.ErrConflict)
		require.NoError(t, differentErr)
		wantOutbox = 1
	}
	require.Equal(t, wantOutbox, scalar(t, s, `SELECT count(*) FROM outbox`))
	require.Equal(t, 1, scalar(t, s, `SELECT count(*) FROM ingest_revisions WHERE receipt_id='race' AND author='user'`))
}
