package sqlite

import (
	"testing"
	"time"

	"github.com/ebrgr/TuneTrail/internal/db"
	"github.com/ebrgr/TuneTrail/internal/ingest"
	"github.com/stretchr/testify/require"
)

// TestIngestReviewDecisions covers scope, candidate membership, version fencing and relay cardinality.
func TestIngestReviewDecisions(t *testing.T) {
	s, _ := relayStoreFixture(t)
	ctx := t.Context()
	require.ErrorIs(t, s.SetIngestReview(ctx, 2, true), db.ErrConflict)
	execSQL(t, s, `INSERT INTO profile_ingest_flags(profile_id,active) VALUES(1,1)`)
	health, err := s.IngestHealth(ctx, 1)
	require.NoError(t, err)
	require.True(t, health.ReviewAvailable)
	require.False(t, health.ReviewEnabled)

	now := time.Unix(42, 0).UTC()
	relayReceipt(t, s, 1, "default-off", "v9", 1, now)
	resolveRelayV9(t, s)
	require.Equal(t, 1, scalar(t, s, `SELECT count(*) FROM ingest_events WHERE receipt_id='default-off' AND state='duplicate'`))
	require.NoError(t, s.SetIngestReview(ctx, 1, true))
	for _, id := range []string{"same", "different", "foreign"} {
		relayReceipt(t, s, 1, id, "v9", 1, now)
		resolveRelayV9(t, s)
	}
	var event, foreign string
	require.NoError(t, s.db.QueryRow(`SELECT event_id FROM listens WHERE rowid=10`).Scan(&event))
	require.NoError(t, s.db.QueryRow(`SELECT event_id FROM listens WHERE rowid=30`).Scan(&foreign))
	execSQL(t, s, `UPDATE ingest_events SET evidence=json_object('candidate_event_id',?) WHERE receipt_id='foreign'`, foreign)

	// Disabling the setting still allows decisions on existing receipts.
	require.NoError(t, s.SetIngestReview(ctx, 1, false))
	health, err = s.IngestHealth(ctx, 1)
	require.NoError(t, err)
	require.False(t, health.ReviewEnabled)
	require.Equal(t, 3, health.ReviewCount)
	for _, item := range health.Reviews {
		if item.Receipt == "foreign" {
			require.Empty(t, item.Candidates)
		} else {
			require.Len(t, item.Candidates, 1)
			require.Equal(t, event, item.Candidates[0].Event)
		}
	}
	other, err := s.IngestHealth(ctx, 2)
	require.NoError(t, err)
	require.Empty(t, other.Reviews)

	same := db.IngestReviewDecision{Receipt: "same", Version: 1, Value: "same", Candidate: event}
	require.ErrorIs(t, s.ResolveIngestReview(ctx, 2, 1, same), db.ErrNotFound)
	fake := db.IngestReviewDecision{Receipt: "foreign", Version: 1, Value: "same", Candidate: foreign}
	require.ErrorIs(t, s.ResolveIngestReview(ctx, 1, 1, fake), db.ErrNotFound)
	unlisted := same
	require.NoError(t, s.db.QueryRow(`SELECT event_id FROM listens WHERE rowid=20`).Scan(&unlisted.Candidate))
	require.ErrorIs(t, s.ResolveIngestReview(ctx, 1, 1, unlisted), db.ErrNotFound)
	require.NoError(t, s.ResolveIngestReview(ctx, 1, 1, same))
	require.ErrorIs(t, s.ResolveIngestReview(ctx, 1, 1, same), db.ErrConflict)
	require.Zero(t, scalar(t, s, `SELECT count(*) FROM outbox`))
	require.Equal(t, 3, scalar(t, s, `SELECT count(*) FROM listens`))
	require.Equal(t, 1, scalar(t, s, `SELECT count(*) FROM ingest_events WHERE receipt_id='same' AND state='duplicate' AND event_id=? AND closed_at IS NOT NULL`, event))

	separate := db.IngestReviewDecision{Receipt: "different", Version: 1, Value: "different"}
	// Failure after the decision rolls the listen, version and revision back together.
	execSQL(t, s, `CREATE TRIGGER fail_review_intent BEFORE INSERT ON outbox BEGIN SELECT RAISE(ABORT,'fixture'); END`)
	require.Error(t, s.ResolveIngestReview(ctx, 1, 1, separate))
	require.Equal(t, 3, scalar(t, s, `SELECT count(*) FROM listens`))
	require.Equal(t, 1, scalar(t, s, `SELECT count(*) FROM ingest_revisions WHERE receipt_id='different'`))
	execSQL(t, s, `DROP TRIGGER fail_review_intent`)
	require.NoError(t, s.ResolveIngestReview(ctx, 1, 1, separate))
	require.ErrorIs(t, s.ResolveIngestReview(ctx, 1, 1, separate), db.ErrConflict)
	require.Equal(t, 4, scalar(t, s, `SELECT count(*) FROM listens`))
	require.Equal(t, 1, scalar(t, s, `SELECT count(*) FROM outbox JOIN ingest_events USING(profile_id,event_id)
		WHERE receipt_id='different' AND ingest_events.state='accepted' AND outbox.format_version=1`))
	require.Equal(t, 2, scalar(t, s, `SELECT count(*) FROM ingest_revisions WHERE author='user' AND resolution_version=2`))
}

// TestIngestReviewRace runs opposing decisions through independent SQLite connections.
func TestIngestReviewRace(t *testing.T) {
	s, path := relayStoreFixture(t)
	execSQL(t, s, `PRAGMA busy_timeout=5000`)
	execSQL(t, s, `INSERT INTO profile_ingest_flags(profile_id,active,review) VALUES(1,1,1)`)
	relayReceipt(t, s, 1, "race", "v9", 1, time.Unix(42, 0))
	resolveRelayV9(t, s)
	var candidate string
	require.NoError(t, s.db.QueryRow(`SELECT event_id FROM listens WHERE rowid=10`).Scan(&candidate))
	conn, err := openExisting(path)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, conn.Close()) })
	other := &Sqlite{db: conn}

	start := make(chan struct{})
	same, different := make(chan error, 1), make(chan error, 1)
	go func() {
		<-start
		same <- s.ResolveIngestReview(t.Context(), 1, 1, db.IngestReviewDecision{Receipt: "race", Version: 1, Value: "same", Candidate: candidate})
	}()
	go func() {
		<-start
		different <- other.ResolveIngestReview(t.Context(), 1, 1, db.IngestReviewDecision{Receipt: "race", Version: 1, Value: "different"})
	}()
	close(start)
	sameErr, differentErr := <-same, <-different
	wantNew := 0
	if sameErr == nil {
		require.ErrorIs(t, differentErr, db.ErrConflict)
	} else {
		require.ErrorIs(t, sameErr, db.ErrConflict)
		require.NoError(t, differentErr)
		wantNew = 1
	}
	require.Equal(t, 3+wantNew, scalar(t, s, `SELECT count(*) FROM listens`))
	require.Equal(t, wantNew, scalar(t, s, `SELECT count(*) FROM outbox`))
	require.Equal(t, 1, scalar(t, s, `SELECT count(*) FROM ingest_revisions WHERE receipt_id='race' AND author='user'`))
}

// TestOccurrenceReviewSetting checks exact repeats, the ±2 s window and profile scope.
func TestOccurrenceReviewSetting(t *testing.T) {
	s := convertedOccurrenceFixture(t)
	ctx := t.Context()
	snapshot := ingest.Snapshot{Title: "Review track", Artist: "Artist", Release: "Album"}
	first := admitOccurrenceFixture(t, s, snapshot, 1000, "")
	resolveOccurrenceFixture(t, s, nil)
	requireReceiptState(t, s, first, "accepted")
	repeat := admitOccurrenceFixture(t, s, snapshot, 1000, "")
	resolveOccurrenceFixture(t, s, nil)
	requireReceiptState(t, s, repeat, "duplicate")
	require.Equal(t, 1, scalar(t, s, `SELECT count(*) FROM listens WHERE profile_id=1 AND origin='listenbrainz_api' AND listened_at=1000`))
	require.Equal(t, 1, scalar(t, s, `SELECT count(*) FROM ingest_events e JOIN ingest_events original USING(profile_id,event_id)
		WHERE e.receipt_id=? AND original.receipt_id=? AND e.duplicate_decision='same' AND e.reason_codes='["review_deferred"]'`, repeat, first))

	nearby := admitOccurrenceFixture(t, s, snapshot, 1002, "nearby")
	resolveOccurrenceFixture(t, s, nil)
	requireReceiptState(t, s, nearby, "accepted")
	require.Equal(t, 1, scalar(t, s, `SELECT count(*) FROM ingest_events WHERE receipt_id=? AND reason_codes='["review_deferred"]'`, nearby))
	execSQL(t, s, `INSERT INTO profile_ingest_flags(profile_id,active,occurrence_write,review) VALUES(2,1,1,1)`)
	active, other, err := s.AdmitIngest(ctx, ingest.Batch{ProfileID: 2, Source: ingest.ShadowSourceListenBrainz,
		Operation: "single", ReceivedAt: time.Now(),
		Items: []ingest.Item{{ListenedAt: time.Unix(1000, 0), Snapshot: snapshot}}}, true, nil, nil)
	require.NoError(t, err)
	require.True(t, active)
	require.Len(t, other, 1)
	resolveOccurrenceFixture(t, s, nil)
	requireReceiptState(t, s, other[0].ReceiptID, "accepted")

	require.NoError(t, s.SetIngestReview(ctx, 1, true))
	id := admitOccurrenceFixture(t, s, snapshot, 1000, "")
	resolveOccurrenceFixture(t, s, nil)
	requireReceiptState(t, s, id, "pending_duplicate")
	nearReview := admitOccurrenceFixture(t, s, snapshot, 1004, "near-review")
	resolveOccurrenceFixture(t, s, nil)
	requireReceiptState(t, s, nearReview, "pending_duplicate")
	require.Equal(t, 1, scalar(t, s, `SELECT count(*) FROM ingest_events WHERE receipt_id=? AND reason_codes='["ambiguous_time_candidate"]'`, nearReview))
	health, err := s.IngestHealth(ctx, 1)
	require.NoError(t, err)
	require.Len(t, health.Reviews, 2)
	for _, item := range health.Reviews {
		if item.Receipt == id {
			require.Len(t, item.Candidates, 2)
			require.True(t, item.CanKeep)
			require.Equal(t, "Review track", item.Title)
		}
	}
	require.NoError(t, s.ResolveIngestReview(ctx, 1, 1, db.IngestReviewDecision{Receipt: id, Version: 1, Value: "different"}))
	requireReceiptState(t, s, id, "accepted")
	require.Equal(t, 2, scalar(t, s, `SELECT count(*) FROM listens WHERE profile_id=1 AND origin='listenbrainz_api' AND listened_at=1000`))
	require.Zero(t, scalar(t, s, `SELECT count(*) FROM outbox`))
}
