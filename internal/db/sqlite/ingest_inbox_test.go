package sqlite

import (
	"testing"
	"time"

	"github.com/ebrgr/TuneTrail/internal/db"
	"github.com/ebrgr/TuneTrail/internal/ingest"
	"github.com/stretchr/testify/require"
)

func TestIngestInboxMigrationAndCounts(t *testing.T) {
	s, provider, _ := migrationDB(t, 18)
	ctx := t.Context()
	seedIdentityHistory(t, s)
	now := time.Date(2026, 11, 22, 17, 0, 0, 0, time.UTC)
	execSQL(t, s, `INSERT INTO ingest_events(receipt_id,profile_id,source_instance_id,schema_version,source_type,
		adapter_version,batch_id,item_ordinal,operation,received_at,time_precision,time_semantics,original_snapshot,snapshot_digest,digest_version)
		VALUES('legacy-inbox',1,'source',1,'listenbrainz','1','legacy',0,'single',?,'second','start',
			'{"title":{"private":"hidden"},"artist":"Public artist"}','legacy','1')`, now.Add(-2*time.Hour).Unix())
	_, err := provider.UpTo(ctx, 19)
	require.NoError(t, err)
	_, err = provider.Down(ctx)
	require.Error(t, err)
	execSQL(t, s, `INSERT INTO profile_ingest_flags(profile_id,active) VALUES(1,1)`)

	first, err := s.SaveApiKey(ctx, db.SaveApiKeyOpts{Key: "unused-secret-a", ProfileID: 1, Label: "Player A"})
	require.NoError(t, err)
	second, err := s.SaveApiKey(ctx, db.SaveApiKeyOpts{Key: "unused-secret-b", ProfileID: 1, Label: "Player B"})
	require.NoError(t, err)
	receive := func(profile int32, key int64, source string, age time.Duration, snapshot ingest.Snapshot) string {
		t.Helper()
		at := now.Add(-age)
		_, receipts, err := s.AdmitIngest(ctx, ingest.Batch{
			ProfileID: profile, APIKeyID: key, Source: source, Operation: "single", ReceivedAt: at,
			Items: []ingest.Item{{Snapshot: snapshot, ListenedAt: at}},
		}, true, nil, nil)
		require.NoError(t, err)
		require.Len(t, receipts, 1)
		return receipts[0].ReceiptID
	}
	missing := ingest.Snapshot{Title: "Missing artist", Client: "player"}
	incomplete := ingest.Snapshot{Title: "Incomplete track", Artist: "Artist", Client: "player"}
	complete := ingest.Snapshot{
		Title: "Complete track", Artist: "Artist", Release: "Album", Client: "player", DurationMS: 1000,
		Recording: "00000000-0000-4000-8000-000000000001", ReleaseMBID: "00000000-0000-4000-8000-000000000002",
		ArtistMBIDs: []string{"00000000-0000-4000-8000-000000000003"},
	}
	ref := receive(1, int64(first.ID), ingest.ShadowSourceListenBrainz, 24*time.Hour, missing)
	accepted := receive(1, int64(first.ID), ingest.ShadowSourceListenBrainz, 3*24*time.Hour, incomplete)
	duplicate := receive(1, int64(first.ID), ingest.ShadowSourceListenBrainz, 29*24*time.Hour, complete)
	receive(1, int64(first.ID), ingest.ShadowSourceListenBrainz, 31*24*time.Hour, complete)
	receive(1, int64(second.ID), ingest.ShadowSourceListenBrainz, time.Hour, incomplete)
	receive(1, 0, ingest.ShadowSourceFile, time.Hour, complete)
	receive(2, 0, ingest.ShadowSourceListenBrainz, time.Hour, complete)
	execSQL(t, s, `UPDATE ingest_events SET state='invalid',rule_version='fixture',reason_codes='["invalid"]' WHERE receipt_id=?`, ref)
	execSQL(t, s, `UPDATE ingest_events SET state=CASE WHEN receipt_id=? THEN 'accepted' ELSE 'duplicate' END,
		identity_status='resolved',resolution_version=1,
		(event_id,occurrence_id,recording_id)=(SELECT l.event_id,o.id,o.recording_id
			FROM listens l JOIN occurrences o ON o.id=l.track_id WHERE l.profile_id=1 LIMIT 1)
		WHERE receipt_id IN (?,?)`, accepted, accepted, duplicate)
	require.NoError(t, s.DeleteApiKey(ctx, second.ID))
	replacement, err := s.SaveApiKey(ctx, db.SaveApiKeyOpts{Key: "unused-secret-c", ProfileID: 1})
	require.NoError(t, err)
	require.Greater(t, replacement.ID, second.ID)

	key := int64(first.ID)
	for _, tc := range []struct{ days, total, incomplete int64 }{{1, 1, 0}, {7, 2, 1}, {30, 3, 1}} {
		groups, err := s.IngestInbox(ctx, 1, &key, now.Add(-time.Duration(tc.days)*24*time.Hour), now)
		require.NoError(t, err)
		require.Len(t, groups, 1)
		group := groups[0]
		require.Equal(t, "Player A", group.Label)
		require.Equal(t, tc.total, group.Total)
		require.EqualValues(t, 1, group.Invalid)
		require.Zero(t, group.Pending)
		require.EqualValues(t, tc.incomplete, group.Accepted)
		require.EqualValues(t, tc.total-1-tc.incomplete, group.Duplicate)
		require.EqualValues(t, 1, group.Missing)
		require.Equal(t, tc.incomplete, group.Incomplete)
		require.Equal(t, "Missing artist", group.Examples[0].Title)
		if tc.incomplete != 0 {
			require.Equal(t, "Incomplete track", group.Examples[1].Title)
		}
	}
	groups, err := s.IngestInbox(ctx, 1, nil, now.Add(-24*time.Hour), now)
	require.NoError(t, err)
	require.Len(t, groups, 4)
	require.EqualValues(t, -1, groups[0].Key)
	require.EqualValues(t, 0, groups[1].Key)
	require.EqualValues(t, 1, groups[1].Missing)
	require.Empty(t, groups[1].Examples[0].Title)
	require.Equal(t, "Public artist", groups[1].Examples[0].Artist)
	require.EqualValues(t, second.ID, groups[3].Key)
	require.False(t, groups[3].KeyPresent)
	require.Empty(t, groups[3].Label)
	other, err := s.IngestInbox(ctx, 2, &key, now.Add(-24*time.Hour), now)
	require.NoError(t, err)
	require.Empty(t, other)
	execSQL(t, s, `DELETE FROM ingest_events WHERE receipt_id=?`, ref)
	groups, err = s.IngestInbox(ctx, 1, &key, now.Add(-24*time.Hour), now)
	require.NoError(t, err)
	require.Empty(t, groups)
}
