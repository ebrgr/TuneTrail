package ingest

import (
	"encoding/json"
	"math"
	"testing"
	"time"

	"github.com/ebrgr/TuneTrail/internal/listenbrainz"
	"github.com/stretchr/testify/require"
)

// TestListenBrainzOccurrenceFixture preserves declared IDs, ordered credits and the raw timestamp.
func TestListenBrainzOccurrenceFixture(t *testing.T) {
	var item listenbrainz.LbzSubmitListenPayload
	require.NoError(t, json.Unmarshal([]byte(`{"listened_at":1780000000.250,"track_metadata":{"artist_name":"A & B","track_name":"Intro","additional_info":{"artist_names":["A","B"],"discnumber":"2","tracknumber":1,"track_mbid":"fa000000-0000-4000-8000-000000000001","mbid_source":"navidrome-subsonic","mbid_declared_at":"2026-11-20T00:00:00Z"}}}`), &item))
	snapshot := ListenBrainzSnapshot(item.TrackMeta)
	require.Equal(t, "1780000000.250", item.OriginalListenedAt)
	require.Equal(t, []string{"A", "B"}, snapshot.ArtistNames)
	require.Equal(t, 2, snapshot.DiscNumber)
	require.Equal(t, 1, snapshot.TrackNumber)
	require.NotEmpty(t, snapshot.TrackMBID)
	require.Equal(t, "navidrome-subsonic", snapshot.MBIDSource)
	require.Equal(t, "2026-11-20T00:00:00Z", snapshot.MBIDDeclaredAt)
	item.TrackMeta.AdditionalInfo.MBIDSource = "arbitrary-source"
	unrecognized := ListenBrainzSnapshot(item.TrackMeta)
	require.Empty(t, unrecognized.MBIDSource)
	require.Empty(t, unrecognized.MBIDDeclaredAt)
	require.Empty(t, snapshot.Recording)
}

// TestOccurrenceTimestamp covers exact decimal time and a declared offset.
func TestOccurrenceTimestamp(t *testing.T) {
	item := Item{ListenedAt: time.Unix(1780000000, 0), Snapshot: Snapshot{OriginalTime: "1780000000.250"}}
	at, precision, offset, semantics := item.OccurrenceTime()
	require.EqualValues(t, 250000000, at.Nanosecond())
	require.Equal(t, "millisecond", precision)
	require.Empty(t, offset)
	require.Equal(t, "unknown", semantics)
	item.Snapshot.OriginalTime = "2026-05-28T21:46:40.2500-03:00"
	item.Snapshot.TimeSemantics = "end"
	at, precision, offset, semantics = item.OccurrenceTime()
	require.EqualValues(t, 250000000, at.Nanosecond())
	require.Equal(t, "microsecond", precision)
	require.Equal(t, "-03:00", offset)
	require.Equal(t, "end", semantics)
}

func TestListenBrainzPlaybackTime(t *testing.T) {
	for _, tc := range []struct {
		name string
		info string
		want *int64
	}{
		{"seconds as text", `{"duration_played":"120"}`, PlayedSeconds(120)},
		{"milliseconds as text", `{"duration_played_ms":"90500"}`, PlayedSeconds(90)},
		{"zero seconds fall back to milliseconds", `{"duration_played":"0","duration_played_ms":90500}`, PlayedSeconds(90)},
		{"valid fallback", `{"duration_played":"unknown","duration_played_ms":"90500"}`, PlayedSeconds(90)},
		{"nonnumeric", `{"duration_played":"unknown","duration_played_ms":"unknown"}`, nil},
		{"null and absent", `{"duration_played":null}`, nil},
		{"empty and boolean", `{"duration_played":"","duration_played_ms":true}`, nil},
		{"object and list", `{"duration_played":{},"duration_played_ms":[]}`, nil},
		{"negative and NaN", `{"duration_played":-1,"duration_played_ms":"NaN"}`, nil},
		{"huge values", `{"duration_played":1e999,"duration_played_ms":"1e30"}`, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var meta listenbrainz.LbzTrackMeta
			require.NoError(t, json.Unmarshal([]byte(`{"additional_info":`+tc.info+`}`), &meta))
			require.Equal(t, tc.want, ListenBrainzSnapshot(meta).DurationPlayed)
		})
	}
}

// TestPlayedSecondsZeroIsUnknown makes zero, sub-second and invalid values unknown, so the track length is used (ISSUE-190).
func TestPlayedSecondsZeroIsUnknown(t *testing.T) {
	for _, unknown := range []float64{0, 0.4, 0.999, -1, math.NaN(), math.Inf(1), float64(math.MaxInt32) + 1} {
		if got := PlayedSeconds(unknown); got != nil {
			t.Errorf("PlayedSeconds(%v) = %d, want unknown", unknown, *got)
		}
	}
	if got := PlayedSeconds(1); got == nil || *got != 1 {
		t.Error("one second is the smallest measurement")
	}
	if got := PlayedSeconds(90.9); got == nil || *got != 90 {
		t.Error("measurements truncate to whole seconds")
	}
}
