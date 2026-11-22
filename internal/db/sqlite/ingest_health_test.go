package sqlite

import (
	"slices"
	"testing"
	"time"

	"github.com/ebrgr/TuneTrail/internal/db"
	"github.com/ebrgr/TuneTrail/internal/ingest"
	"github.com/stretchr/testify/require"
)

// TestIngestHealthBeforeDurableRelay guards the normal v11 startup path.
func TestIngestHealthBeforeDurableRelay(t *testing.T) {
	s, _, _ := migrationDB(t, 11)
	seedIdentityHistory(t, s)
	health, err := s.IngestHealth(t.Context(), 1)
	require.NoError(t, err)
	require.False(t, health.RelayTracked)
	require.Len(t, health.Sources, 1)
	require.Equal(t, "untracked", health.Sources[0].Delivery)
	require.Len(t, health.Decisions, 1)

	provenance, err := s.ListenProvenance(t.Context(), 1, 1, 42)
	require.NoError(t, err)
	require.Len(t, provenance, 1)
	require.Equal(t, "legacy", provenance[0].Origin)
	require.Equal(t, "untracked", provenance[0].Delivery)
	require.Nil(t, provenance[0].Received)
}

// TestIngestHealthSourceSummary covers backfill, admission ordering, pagination and deletion fallback.
func TestIngestHealthSourceSummary(t *testing.T) {
	ctx := t.Context()
	s, provider, _ := migrationDB(t, 16)
	seedIdentityHistory(t, s)
	batch := ingest.Batch{ProfileID: 1, Source: ingest.ShadowSourceListenBrainz, Operation: "single",
		ReceivedAt: time.Unix(1, 0), Items: []ingest.Item{{Snapshot: ingest.Snapshot{Client: "rare client"}}}}
	rare, err := s.ReceiveBatch(ctx, batch)
	require.NoError(t, err)
	_, err = provider.UpTo(ctx, 17)
	require.NoError(t, err)
	require.NoError(t, checkDatabase(s.db))
	require.Equal(t, 1, scalar(t, s, `SELECT count(*) FROM ingest_source_summary WHERE profile_id=1 AND client='rare client' AND receipt_id=?`, rare[0].ReceiptID))

	// Admission is the only application write needed to index a new source.
	batch.Items[0].Snapshot.Client = "active client"
	for _, stamp := range []int64{1000, 500, 1000} {
		batch.ReceivedAt = time.Unix(stamp, 0)
		_, err := s.ReceiveBatch(ctx, batch)
		require.NoError(t, err)
	}
	require.Equal(t, 1, scalar(t, s, `SELECT count(*) FROM ingest_source_summary x WHERE client='active client' AND receipt_id=(
		SELECT receipt_id FROM ingest_events e WHERE e.profile_id=x.profile_id AND e.health_client=x.client
		ORDER BY received_at DESC,receipt_id DESC LIMIT 1)`))
	var latest string
	require.NoError(t, s.db.QueryRow(`SELECT receipt_id FROM ingest_source_summary WHERE client='active client'`).Scan(&latest))
	execSQL(t, s, `DELETE FROM ingest_events WHERE receipt_id=?`, latest)
	require.Equal(t, 1, scalar(t, s, `SELECT count(*) FROM ingest_source_summary WHERE client='active client' AND received_at=1000 AND receipt_id<>?`, latest))

	batch.ProfileID = 2
	_, err = s.ReceiveBatch(ctx, batch)
	require.NoError(t, err)
	require.Equal(t, 2, scalar(t, s, `SELECT count(*) FROM ingest_source_summary WHERE client='active client'`))
	execSQL(t, s, `WITH RECURSIVE sample(n) AS (VALUES(1) UNION ALL SELECT n+1 FROM sample WHERE n<101)
		INSERT INTO ingest_events(receipt_id,profile_id,source_instance_id,schema_version,source_type,adapter_version,batch_id,
			item_ordinal,operation,received_at,time_precision,time_semantics,original_snapshot,snapshot_digest,digest_version)
		SELECT 'source-summary-'||n,1,'source',1,'test','1','batch',n,'single',n+10,'second','start',
			json_object('client','client-'||n),'digest','1' FROM sample`)
	first, err := s.IngestHealth(ctx, 1)
	require.NoError(t, err)
	require.Len(t, first.Sources, healthSourcePageSize)
	require.Equal(t, 2, first.SourceNext)
	second, err := s.IngestHealth(ctx, 1, 1, 2)
	require.NoError(t, err)
	require.Equal(t, 1, second.SourcePrevious)
	require.True(t, slices.ContainsFunc(second.Sources, func(source db.IngestSource) bool {
		return source.Client == "rare client"
	}))
	execSQL(t, s, `DELETE FROM ingest_events WHERE receipt_id=?`, rare[0].ReceiptID)
	require.Zero(t, scalar(t, s, `SELECT count(*) FROM ingest_source_summary WHERE client='rare client'`))
	execSQL(t, s, `DELETE FROM listening_profiles WHERE id=2`)
	require.Zero(t, scalar(t, s, `SELECT count(*) FROM ingest_source_summary WHERE profile_id=2`))
}
