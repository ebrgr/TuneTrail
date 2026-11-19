package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"

	"fmt"
	"github.com/ebrgr/TuneTrail/internal/dedup"
	"strings"
	"testing"
	"time"

	"github.com/ebrgr/TuneTrail/internal/ingest"
	"github.com/stretchr/testify/require"
)

// TestShadowWALRetry exercises the receiver on two independently opened pools against schema v4.
func TestShadowWALRetry(t *testing.T) {
	s, _, path := migrationDB(t, 4)
	execSQL(t, s, `PRAGMA journal_mode=WAL`)
	execSQL(t, s, `PRAGMA busy_timeout=5000`)
	execSQL(t, s, `INSERT INTO listening_profiles(id,slug,display_name,created_at) VALUES(1,'one','One',0)`)
	other, err := sql.Open("sqlite", sqliteURI(path)+"?mode=rw&_pragma=journal_mode(WAL)&_pragma=foreign_keys(ON)&_pragma=busy_timeout(5000)")
	require.NoError(t, err)
	defer other.Close()
	for _, key := range []string{"external", "client", "file"} {
		t.Run(key, func(t *testing.T) {
			item := ingest.Item{Ordinal: 7, ListenedAt: time.Unix(1700000000, 0), Snapshot: ingest.Snapshot{Artist: "Artist", Title: key}}
			batch := ingest.Batch{ProfileID: 1, Source: "listenbrainz", Operation: "single", ReceivedAt: time.Now(), Items: []ingest.Item{item}}
			switch key {
			case "external":
				batch.Items[0].ExternalNamespace, batch.Items[0].ExternalID = "test-v1", "event"
			case "client":
				batch.ClientKey = ingest.Digest([]byte("client-key"))
			case "file":
				batch.Source, batch.FileFormat, batch.FileDigest = "file", "test", ingest.Digest([]byte("file"))
			}
			type outcome struct {
				result []ingest.Result
				err    error
			}
			start := make(chan struct{})
			outcomes := make(chan outcome, 2)
			for _, store := range []*Sqlite{s, {db: other}} {
				go func() {
					<-start
					result, err := store.ReceiveBatch(context.Background(), batch)
					outcomes <- outcome{result, err}
				}()
			}
			close(start)
			a, b := <-outcomes, <-outcomes
			require.NoError(t, a.err)
			require.NoError(t, b.err)
			require.Len(t, a.result, 1)
			require.Len(t, b.result, 1)
			id := a.result[0].ReceiptID
			require.Equal(t, id, b.result[0].ReceiptID)
			require.ElementsMatch(t, []string{"different", "same"}, []string{a.result[0].Decision, b.result[0].Decision})
			require.Equal(t, 1, scalar(t, s, `SELECT count(*) FROM ingest_events WHERE receipt_id=? AND
				duplicate_decision='different' AND reason_codes='["no_candidate"]' AND rule_version='shadow-v2'
				AND json_extract(evidence,'$.candidate_receipt_id') IS NULL`, id))
			if key == "external" {
				batch.Items[0].Snapshot.Title = "Changed"
				changed, err := s.ReceiveBatch(context.Background(), batch)
				require.NoError(t, err)
				require.Equal(t, "review", changed[0].Decision)
				require.Equal(t, 1, scalar(t, s, `SELECT count(*) FROM ingest_observations WHERE receipt_id=?`, id))
				require.Equal(t, 1, scalar(t, s, `SELECT count(*) FROM ingest_events WHERE receipt_id=? AND state='pending_duplicate'
					AND original_snapshot LIKE '%"external"%'`, id))
				execSQL(t, s, `CREATE TRIGGER shadow_rewrite_guard BEFORE UPDATE ON ingest_events
					BEGIN SELECT RAISE(ABORT,'retry rewrote receipt'); END`)
				again, err := s.ReceiveBatch(context.Background(), batch)
				require.NoError(t, err)
				require.Equal(t, changed, again)
				require.Equal(t, 1, scalar(t, s, `SELECT count(*) FROM ingest_observations WHERE receipt_id=?`, id))
				execSQL(t, s, `DROP TRIGGER shadow_rewrite_guard`)
			}
		})
	}
	require.Equal(t, 3, scalar(t, s, `SELECT count(*) FROM ingest_events`))
}

// TestShadowRepetitions covers offset equivalence, the DST fold and independent file ordinals.
func TestShadowRepetitions(t *testing.T) {
	s := ingestFixture(t)
	first, err := time.Parse(time.RFC3339, "2026-11-01T01:30:00-04:00")
	require.NoError(t, err)
	second, err := time.Parse(time.RFC3339, "2026-11-01T01:30:00-05:00")
	require.NoError(t, err)
	track := ingest.Snapshot{Title: "Track", Artist: "Artist"}
	batch := ingest.Batch{ProfileID: 1, Source: "listenbrainz", Operation: "import", ReceivedAt: time.Now(), Items: []ingest.Item{
		{Ordinal: 0, ListenedAt: first, Snapshot: track},
		{Ordinal: 1, ListenedAt: first.UTC(), Snapshot: track},
		{Ordinal: 2, ListenedAt: second, Snapshot: track},
	}}
	store := func(b ingest.Batch, decisions ...string) []ingest.Result {
		results, err := s.ReceiveBatch(context.Background(), b)
		require.NoError(t, err)
		require.Len(t, results, len(decisions))
		for i, result := range results {
			require.Equal(t, decisions[i], result.Decision)
		}
		return results
	}
	store(batch, "different", "review", "different")
	batch.Source, batch.FileFormat, batch.FileDigest = "file", "test", ingest.Digest([]byte("file"))
	a := store(batch, "review", "review", "review")
	b := store(batch, "same", "same", "same")
	require.NotEqual(t, a[0].ReceiptID, a[1].ReceiptID)
	for i := range a {
		require.Equal(t, a[i].ReceiptID, b[i].ReceiptID)
	}
	batch = ingest.Batch{ProfileID: 2, Source: "listenbrainz", Operation: "single", ReceivedAt: first, Items: []ingest.Item{{Snapshot: track}}}
	store(batch, "different")
	require.Equal(t, 1, scalar(t, s, `SELECT count(*) FROM ingest_events WHERE profile_id=2 AND time_precision='estimated' AND original_time IS NULL`))
	require.Zero(t, scalar(t, s, `SELECT count(*) FROM listens`))
}

// TestShadowDiskFull leaves the entire batch available to retry.
func TestShadowDiskFull(t *testing.T) {
	s, _, _ := migrationDB(t, 4)
	execSQL(t, s, `INSERT INTO listening_profiles(id,slug,display_name,created_at) VALUES(1,'one','One',0)`)
	limit := scalar(t, s, `PRAGMA page_count`)
	execSQL(t, s, fmt.Sprintf("PRAGMA max_page_count=%d", limit))
	batch := ingest.Batch{ProfileID: 1, Source: "listenbrainz", Operation: "import", ReceivedAt: time.Now(), Items: []ingest.Item{
		{Ordinal: 0, Snapshot: ingest.Snapshot{Artist: "Artist", Title: "Small"}},
		{Ordinal: 1, Snapshot: ingest.Snapshot{Artist: "Artist", Title: strings.Repeat("large", 50000)}},
	}}
	_, err := s.ReceiveBatch(context.Background(), batch)
	var unavailable *ingest.ErrUnavailable
	require.ErrorAs(t, err, &unavailable)
	require.Zero(t, scalar(t, s, `SELECT count(*) FROM ingest_events`))
	require.Zero(t, scalar(t, s, `SELECT count(*) FROM integrations`))
	execSQL(t, s, fmt.Sprintf("PRAGMA max_page_count=%d", limit+1000))
	results, err := s.ReceiveBatch(context.Background(), batch)
	require.NoError(t, err)
	require.Len(t, results, 2)
	require.Equal(t, 2, scalar(t, s, `SELECT count(*) FROM ingest_events`))
}

// TestShadowClientKey scopes batch conflicts to an operation and profile.
func TestShadowClientKey(t *testing.T) {
	s := ingestFixture(t)
	ctx := context.Background()
	batch := ingest.Batch{ProfileID: 1, Source: "listenbrainz", Operation: "import", ReceivedAt: time.Now(), ClientKey: "client-key",
		Items: []ingest.Item{{Ordinal: 0, Snapshot: ingest.Snapshot{Artist: "Artist", Title: "First"}}}}
	original, err := s.ReceiveBatch(ctx, batch)
	require.NoError(t, err)
	retry, err := s.ReceiveBatch(ctx, batch)
	require.NoError(t, err)
	require.Equal(t, original[0].ReceiptID, retry[0].ReceiptID)
	require.Equal(t, "same", retry[0].Decision)
	batch.Items = append(batch.Items, ingest.Item{Ordinal: 1, Snapshot: ingest.Snapshot{Artist: "Artist", Title: "Added"}})
	conflict, err := s.ReceiveBatch(ctx, batch)
	require.NoError(t, err)
	require.Equal(t, "review", conflict[0].Decision)
	require.Equal(t, "review", conflict[1].Decision)
	batch.ProfileID = 2
	other, err := s.ReceiveBatch(ctx, batch)
	require.NoError(t, err)
	require.NotEqual(t, original[0].ReceiptID, other[0].ReceiptID)
	require.Equal(t, "different", other[0].Decision)
	require.Equal(t, "different", other[1].Decision)
}

// TestShadowV4AcceptsSame inserts a schema fixture using a real trusted-key verdict.
// The fixture omits the key on the second row because the runtime reuses its unique receipt.
func TestShadowV4AcceptsSame(t *testing.T) {
	s, _, _ := migrationDB(t, 4)
	execSQL(t, s, `INSERT INTO listening_profiles(id,slug,display_name,created_at) VALUES(1,'one','One',0)`)
	batch := ingest.Batch{ProfileID: 1, Source: "test", Operation: "single", ReceivedAt: time.Now(),
		Items: []ingest.Item{{ExternalNamespace: "test", ExternalID: "stable", Snapshot: ingest.Snapshot{Artist: "A", Title: "T"}}}}
	result, err := s.ReceiveBatch(context.Background(), batch)
	require.NoError(t, err)
	candidate, err := scanReceipt(s.QueryRow(`SELECT `+receiptColumns+` FROM ingest_events WHERE receipt_id=?`, result[0].ReceiptID))
	require.NoError(t, err)
	_, digest := batch.Items[0].Encode()
	incoming := dedup.Receipt{Digest: digest, Time: batch.ReceivedAt, Precision: "estimated"}
	decision := dedup.Compare(dedup.Shadow, incoming, &candidate, true)
	require.Equal(t, "same", decision.Value)
	reasons, err := json.Marshal([]string{decision.Reason})
	require.NoError(t, err)
	evidence, err := json.Marshal(decision.Evidence)
	require.NoError(t, err)
	execSQL(t, s, `INSERT INTO ingest_events(receipt_id,profile_id,source_instance_id,schema_version,source_type,adapter_version,
		batch_id,item_ordinal,operation,received_at,time_precision,time_semantics,original_snapshot,snapshot_digest,digest_version,
		duplicate_decision,reason_codes,evidence,rule_version)
		SELECT 'same-fixture',profile_id,source_instance_id,schema_version,source_type,adapter_version,'other',0,operation,
		received_at,time_precision,time_semantics,original_snapshot,snapshot_digest,digest_version,?,?,?,? FROM ingest_events WHERE receipt_id=?`,
		decision.Value, string(reasons), string(evidence), decision.RuleVersion, candidate.ID)
	require.Equal(t, 1, scalar(t, s, `SELECT count(*) FROM ingest_events WHERE receipt_id='same-fixture'
		AND duplicate_decision='same' AND event_id IS NULL AND json_extract(evidence,'$.candidate_receipt_id')=?`, candidate.ID))
	require.Equal(t, 1, scalar(t, s, `SELECT count(*) FROM ingest_events WHERE receipt_id=? AND duplicate_decision='different'`, candidate.ID))
}
