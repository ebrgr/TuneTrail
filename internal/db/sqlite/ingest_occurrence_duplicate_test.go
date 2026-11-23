package sqlite

import (
	"fmt"
	"testing"
	"time"

	"github.com/ebrgr/TuneTrail/internal/db"
	"github.com/ebrgr/TuneTrail/internal/ingest"
	"github.com/stretchr/testify/require"
)

// TestOccurrenceDistinctSameSecond preserves known separate deliveries in both review modes.
func TestOccurrenceDistinctSameSecond(t *testing.T) {
	for _, review := range []bool{false, true} {
		for _, source := range []string{ingest.ShadowSourceFile, ingest.ShadowSourceListenBrainz} {
			t.Run(fmt.Sprintf("%s/review=%t", source, review), func(t *testing.T) {
				ctx := t.Context()
				s := convertedOccurrenceFixture(t)
				require.NoError(t, s.SetIngestReview(ctx, 1, review))
				snapshot := ingest.Snapshot{Title: "Identical tracks", Artist: "Artist", Release: "Album"}
				item := ingest.Item{ListenedAt: time.Unix(5000, 0), Snapshot: snapshot}
				batch := ingest.Batch{ProfileID: 1, Source: source, Operation: "import", ReceivedAt: time.Now(),
					Items: []ingest.Item{item, item}}
				if source == ingest.ShadowSourceFile {
					batch.FileFormat, batch.FileDigest = "lastfm", ingest.Digest([]byte("two identical lines"))
					batch.Items[1].Ordinal = 1
				} else {
					batch.Items[0].ExternalNamespace, batch.Items[1].ExternalNamespace = "fixture", "fixture"
					batch.Items[0].ExternalID, batch.Items[1].ExternalID = "first", "second"
				}
				admitted, err := s.AdmitOccurrence(ctx, batch)
				require.NoError(t, err)
				require.Len(t, admitted, 2)
				for range admitted {
					resolveOccurrenceFixture(t, s, nil)
				}
				requireReceiptState(t, s, admitted[0].ReceiptID, "accepted")
				requireReceiptState(t, s, admitted[1].ReceiptID, "accepted")
				require.Equal(t, 2, scalar(t, s, `SELECT count(*) FROM listens WHERE profile_id=1 AND listened_at=5000`))
			})
		}
	}
}

// TestOccurrenceTrustedReplay preserves the key and bounds the review queue across resends.
func TestOccurrenceTrustedReplay(t *testing.T) {
	for _, review := range []bool{false, true} {
		t.Run(fmt.Sprintf("review=%t", review), func(t *testing.T) {
			ctx := t.Context()
			s := convertedOccurrenceFixture(t)
			require.NoError(t, s.SetIngestReview(ctx, 1, review))
			batch := ingest.Batch{ProfileID: 1, Source: ingest.ShadowSourceListenBrainz, Operation: "single", ReceivedAt: time.Now(),
				Items: []ingest.Item{{ExternalNamespace: "fixture", ExternalID: "stable", ListenedAt: time.Unix(5000, 0),
					Snapshot: ingest.Snapshot{Title: "Trusted replay", Artist: "Artist", Release: "Album"}}}}
			admit := func() ingest.Result {
				t.Helper()
				result, err := s.AdmitOccurrence(ctx, batch)
				require.NoError(t, err)
				require.Len(t, result, 1)
				return result[0]
			}
			original := admit()
			resolveOccurrenceFixture(t, s, nil)
			replay := admit()
			require.Equal(t, replay.ReceiptID, admit().ReceiptID)
			require.Equal(t, 1, scalar(t, s, `SELECT count(*) FROM listens WHERE profile_id=1 AND listened_at=5000`))
			require.Equal(t, 1, scalar(t, s, `SELECT count(*) FROM ingest_events WHERE external_id_trusted=1 AND external_event_id='stable'`))
			requireReceiptState(t, s, original.ReceiptID, "accepted")
			if !review {
				require.Equal(t, original.ReceiptID, replay.ReceiptID)
				require.False(t, replay.Pending)
				return
			}
			require.NotEqual(t, original.ReceiptID, replay.ReceiptID)
			require.True(t, replay.Pending)
			requireReceiptState(t, s, replay.ReceiptID, "pending_duplicate")
			health, err := s.IngestHealth(ctx, 1)
			require.NoError(t, err)
			require.Len(t, health.Reviews, 1)
			require.Len(t, health.Reviews[0].Candidates, 1)
			require.NoError(t, s.ResolveIngestReview(ctx, 1, 1, db.IngestReviewDecision{
				Receipt: replay.ReceiptID, Candidate: health.Reviews[0].Candidates[0].Event, Value: "same", Version: 1,
			}))
			requireReceiptState(t, s, replay.ReceiptID, "duplicate")
			require.False(t, admit().Pending)
			require.Equal(t, 1, scalar(t, s, `SELECT count(*) FROM listens WHERE profile_id=1 AND listened_at=5000`))
		})
	}
}
