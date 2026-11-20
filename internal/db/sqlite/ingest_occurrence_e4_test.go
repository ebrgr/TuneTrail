package sqlite

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/ebrgr/TuneTrail/internal/ingest"
	"github.com/stretchr/testify/require"
)

// TestOccurrenceHistoryDeleteWithUnresolvedLease records the boundary of occurrence-based deletion.
func TestOccurrenceHistoryDeleteWithUnresolvedLease(t *testing.T) {
	ctx := context.Background()
	store := convertedOccurrenceFixture(t)
	const stamp = 1783021000
	snapshot := ingest.Snapshot{Title: "In transit", Artist: "Artist", Release: "Album"}
	old := admitOccurrenceFixture(t, store, snapshot, stamp, "earlier-event")
	resolveOccurrenceFixture(t, store, nil)
	var occurrence int32
	var oldEvent string
	require.NoError(t, store.db.QueryRow(`SELECT occurrence_id,event_id FROM ingest_events WHERE receipt_id=?`, old).Scan(&occurrence, &oldEvent))

	pending := admitOccurrenceFixture(t, store, snapshot, stamp, "in-transit")
	claim, _, _, err := store.claimNextIngest(ctx)
	require.NoError(t, err)
	require.Equal(t, pending, claim.id)
	claim.snapshot, claim.unix = snapshot, stamp

	plan, err := store.PreviewProfileDelete(ctx, 1, "track", occurrence)
	require.NoError(t, err)
	removed, err := store.ApplyProfileDelete(ctx, plan)
	require.NoError(t, err)
	require.EqualValues(t, 1, removed)
	require.Zero(t, scalar(t, store, `SELECT count(*) FROM listens WHERE event_id=?`, oldEvent))

	// An unresolved receipt cannot be selected by occurrence, so the original claim may finish.
	require.Equal(t, 1, scalar(t, store, `SELECT count(*) FROM ingest_events
		WHERE receipt_id=? AND lease_owner=? AND occurrence_id IS NULL`, pending, claim.lease))
	require.NoError(t, store.finishOccurrence(ctx, claim, occurrenceProof{}))
	requireReceiptState(t, store, pending, "accepted")
	require.Equal(t, 1, scalar(t, store, `SELECT count(*) FROM listens WHERE profile_id=1
		AND listened_at=? AND event_id<>?`, stamp, oldEvent))
}

// TestOccurrenceAdmissionSQLiteFull simulates a full SQLite file with the active writer.
func TestOccurrenceAdmissionSQLiteFull(t *testing.T) {
	ctx := context.Background()
	store := convertedOccurrenceFixture(t)
	limit := scalar(t, store, `PRAGMA page_count`)
	execSQL(t, store, fmt.Sprintf("PRAGMA max_page_count=%d", limit))
	batch := ingest.Batch{ProfileID: 1, Source: ingest.ShadowSourceListenBrainz,
		Operation: "import", ClientKey: "full-fixture", ReceivedAt: time.Now()}
	for i := range 70 {
		batch.Items = append(batch.Items, ingest.Item{Ordinal: i,
			Snapshot: ingest.Snapshot{Artist: "Artist", Title: strings.Repeat("x", 7000), Release: "Album"}})
	}
	active, results, err := store.AdmitIngest(ctx, batch, true, nil, nil)
	var unavailable *ingest.ErrUnavailable
	require.True(t, active)
	require.ErrorAs(t, err, &unavailable)
	require.ErrorContains(t, err, "database or disk is full")
	require.Empty(t, results)
	require.Zero(t, scalar(t, store, `SELECT count(*) FROM ingest_events WHERE resolver_variant='occurrence'`))
	require.Zero(t, scalar(t, store, `SELECT count(*) FROM integrations WHERE id='listenbrainz:1'`))

	execSQL(t, store, fmt.Sprintf("PRAGMA max_page_count=%d", limit+2000))
	_, results, err = store.AdmitIngest(ctx, batch, true, nil, nil)
	require.NoError(t, err)
	require.Len(t, results, len(batch.Items))
	require.Equal(t, len(batch.Items), scalar(t, store, `SELECT count(*) FROM ingest_events WHERE resolver_variant='occurrence'`))
}
