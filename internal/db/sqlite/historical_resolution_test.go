package sqlite

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// relayReceipt seeds a resolver input without calling a historical writer.
func relayReceipt(t *testing.T, s *Sqlite, profile int32, id string, instant time.Time) {
	t.Helper()
	execSQL(t, s, `INSERT INTO integrations(id,profile_id,source_type,created_at) VALUES('origin:'||?,?,'listenbrainz',0) ON CONFLICT DO NOTHING`, profile, profile)
	execSQL(t, s, `INSERT INTO ingest_events(receipt_id,profile_id,source_instance_id,schema_version,source_type,adapter_version,batch_id,item_ordinal,operation,
		received_at,listened_at,time_precision,time_semantics,original_snapshot,snapshot_digest,digest_version,mode,resolver_variant,delivery_policy)
		VALUES(?,?,'origin:'||?,1,'listenbrainz','1',?,0,'single',? ,?,'second','start',
		json_object('artist','Artist & Co','title','Relay Track','release','Fixture','client','fixture'),'fixture','1','active','occurrence','durable')`,
		id, profile, profile, id, instant.Unix(), instant.UTC().Format(receiptTime))
}

// relayStoreFixture supplies catalog rows and profiles for the two independent writers.
func relayStoreFixture(t *testing.T) (*Sqlite, string) {
	t.Helper()
	s, provider, path := migrationDB(t, 20)
	seedIdentityHistory(t, s)
	execSQL(t, s, `DELETE FROM ingest_events WHERE mode='shadow';
		UPDATE relay_delivery_gate SET policy='durable';
		UPDATE artist_aliases SET alias='Artist & Co' WHERE artist_id=1;
		UPDATE track_aliases SET alias='Relay Track' WHERE track_id=1;
		UPDATE occurrences SET presented_title='Relay Track' WHERE id=1;
		INSERT INTO release_aliases VALUES(1,'Fixture','Canonical',1)`)
	require.NoError(t, s.SetProfileRelay(t.Context(), 1, "https://RELAY.test:443/prefix", "relay-fixture"))
	require.NoError(t, s.SetProfileRelay(t.Context(), 2, "https://else.test", "relay-fixture-two"))
	// Pre-existing relays stay approved on upgrade. New configurations start as drafts.
	_, err := provider.UpTo(t.Context(), currentSchemaVersion)
	require.NoError(t, err)
	require.NoError(t, s.SetIngestReview(t.Context(), 1, false))
	require.NoError(t, s.SetIngestReview(t.Context(), 2, false))
	return s, path
}

// TestResolveIngestLegacyCollision checks current decisions against a pre-upgrade listen.
func TestResolveIngestLegacyCollision(t *testing.T) {
	for _, tc := range []struct {
		name, state, reason string
		review              bool
	}{
		{"disabled", "duplicate", "review_deferred", false},
		{"enabled", "pending_duplicate", "legacy_identity_collision", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s, _ := relayStoreFixture(t)
			require.NoError(t, s.SetIngestReview(t.Context(), 1, tc.review))
			relayReceipt(t, s, 1, "collision", time.Unix(42, 0))
			claim, snapshot, instant, err := s.claimNextIngest(t.Context())
			require.NoError(t, err)
			require.Equal(t, "collision", claim.id)
			for range 2 {
				require.NoError(t, s.resolveIngestClaim(t.Context(), claim, snapshot, instant, nil))
				require.Equal(t, 3, scalar(t, s, `SELECT count(*) FROM listens`))
				require.Equal(t, 1, scalar(t, s, `SELECT count(*) FROM ingest_revisions WHERE receipt_id='collision'`))
				require.Equal(t, 1, scalar(t, s, `SELECT count(*) FROM ingest_events WHERE receipt_id='collision'
					AND state=? AND json_extract(reason_codes,'$[0]')=?`, tc.state, tc.reason))
			}
		})
	}
}
