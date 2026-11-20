package ingest

import (
	"encoding/json"
	"testing"

	"github.com/ebrgr/TuneTrail/internal/listenbrainz"
	"github.com/stretchr/testify/require"
)

// TestListenBrainzOccurrenceFixture preserves declared IDs, ordered credits and the raw timestamp.
func TestListenBrainzOccurrenceFixture(t *testing.T) {
	var item listenbrainz.LbzSubmitListenPayload
	require.NoError(t, json.Unmarshal([]byte(`{"listened_at":1780000000.250,"track_metadata":{"artist_name":"A & B","track_name":"Intro","additional_info":{"artist_names":["A","B"],"discnumber":"2","tracknumber":1,"track_mbid":"fa000000-0000-4000-8000-000000000001","mbid_source":"fixture","mbid_declared_at":"2026-11-20T00:00:00Z"}}}`), &item))
	snapshot := ListenBrainzSnapshot(item.TrackMeta)
	require.Equal(t, "1780000000.250", item.OriginalListenedAt)
	require.Equal(t, []string{"A", "B"}, snapshot.ArtistNames)
	require.Equal(t, 2, snapshot.DiscNumber)
	require.Equal(t, 1, snapshot.TrackNumber)
	require.NotEmpty(t, snapshot.TrackMBID)
	require.Equal(t, "fixture", snapshot.MBIDSource)
	require.Empty(t, snapshot.Recording)
}
