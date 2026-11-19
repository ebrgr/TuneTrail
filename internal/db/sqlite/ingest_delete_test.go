package sqlite

import (
	"context"
	"testing"
	"time"

	"github.com/ebrgr/TuneTrail/internal/db"
	"github.com/ebrgr/TuneTrail/internal/ingest"
	"github.com/stretchr/testify/require"
)

// TestShadowDeletePaths checks atomic rollback, child rows, scope and reimport.
func TestShadowDeletePaths(t *testing.T) {
	for _, kind := range []string{"listen", "profile catalog", "profile account", "global"} {
		t.Run(kind, func(t *testing.T) {
			s := mergeFixture(t)
			ctx := db.WithProfile(context.Background(), 1)
			start := time.Unix(100, 0)
			file := ingest.Batch{ProfileID: 1, Source: "file", Operation: "import", FileFormat: "test",
				FileDigest: ingest.Digest([]byte("test")), ReceivedAt: time.Now(), Items: []ingest.Item{{
					Ordinal: 4, ListenedAt: start.Add(500 * time.Millisecond), Snapshot: ingest.Snapshot{Artist: " ONE ", Title: "one"},
				}}}
			record := func(b ingest.Batch) ingest.Result {
				t.Helper()
				results, err := s.ReceiveBatch(ctx, b)
				require.NoError(t, err)
				require.Len(t, results, 1)
				return results[0]
			}
			first := record(file)
			require.Equal(t, "same", record(file).Decision)
			original := file.Items[0]
			file.Items[0].Snapshot.Release = "variant"
			require.Equal(t, "review", record(file).Decision)
			file.Items[0] = original
			execSQL(t, s, `INSERT INTO ingest_event_details(profile_id,receipt_id,resolution_version,kind,payload) VALUES(1,?,0,'test','{}')`, first.ReceiptID)
			secondary := file
			secondary.Source, secondary.FileDigest, secondary.FileFormat = "listenbrainz", "", ""
			secondary.Items = []ingest.Item{{ListenedAt: start, Snapshot: ingest.Snapshot{Artist: "Two", Title: "Two"}}}
			unrelated := record(secondary)
			secondary.Items[0].ListenedAt = start.Add(time.Second)
			secondary.Items[0].Snapshot = file.Items[0].Snapshot
			nextSecond := record(secondary)
			other := file
			other.ProfileID = 2
			otherReceipt := record(other)
			execSQL(t, s, `INSERT INTO integrations(id,profile_id,source_type,created_at) VALUES('file:configured-source',1,'external',0),('unrelated-integration',2,'other',0)`)
			remove := func() error {
				switch kind {
				case "listen":
					return s.DeleteListen(ctx, 1, start)
				case "profile catalog":
					plan, err := s.PreviewProfileDelete(ctx, 1, "track", 1)
					require.NoError(t, err)
					_, err = s.ApplyProfileDelete(ctx, plan)
					return err
				case "profile account":
					return s.DeleteListeningProfile(ctx, 1, "Alpha", 2)
				default:
					return s.PurgeAllData(ctx)
				}
			}
			execSQL(t, s, `CREATE TRIGGER shadow_delete_failure BEFORE DELETE ON ingest_events
				WHEN OLD.profile_id=1 BEGIN SELECT RAISE(ABORT,'shadow delete failure'); END`)
			require.Error(t, remove())
			require.Equal(t, 4, scalar(t, s, `SELECT count(*) FROM listens`))
			require.Equal(t, 4, scalar(t, s, `SELECT count(*) FROM ingest_events`))
			execSQL(t, s, `DROP TRIGGER shadow_delete_failure`)
			require.NoError(t, remove())
			require.Zero(t, scalar(t, s, `SELECT count(*) FROM ingest_events WHERE receipt_id=?`, first.ReceiptID))
			require.Zero(t, scalar(t, s, `SELECT count(*) FROM ingest_observations`))
			require.Zero(t, scalar(t, s, `SELECT count(*) FROM ingest_event_details`))
			require.Equal(t, 1, scalar(t, s, `SELECT count(*) FROM integrations WHERE id='unrelated-integration'`))
			configured := 1
			if kind == "profile account" {
				configured = 0
			}
			require.Equal(t, configured, scalar(t, s, `SELECT count(*) FROM integrations WHERE id='file:configured-source' AND profile_id=1`))
			if kind == "global" {
				require.Zero(t, scalar(t, s, `SELECT count(*) FROM listens`))
				require.Equal(t, 2, scalar(t, s, `SELECT count(*) FROM integrations`))
			} else {
				require.Equal(t, 2, scalar(t, s, `SELECT count(*) FROM listens WHERE profile_id=2`))
				require.Equal(t, 1, scalar(t, s, `SELECT count(*) FROM ingest_events WHERE receipt_id=?`, otherReceipt.ReceiptID))
				require.Equal(t, 1, scalar(t, s, `SELECT count(*) FROM integrations WHERE id='file:2' AND profile_id=2`))
			}
			if kind == "listen" {
				require.Equal(t, 2, scalar(t, s, `SELECT count(*) FROM ingest_events WHERE receipt_id IN (?,?)`, unrelated.ReceiptID, nextSecond.ReceiptID))
			} else {
				require.Equal(t, configured, scalar(t, s, `SELECT count(*) FROM integrations WHERE profile_id=1`))
				require.Zero(t, scalar(t, s, `SELECT count(*) FROM ingest_events WHERE profile_id=1`))
			}
			if kind != "profile account" {
				reimport := record(file)
				require.NotEqual(t, "same", reimport.Decision)
				require.NotEqual(t, first.ReceiptID, reimport.ReceiptID)
			}
		})
	}
}
