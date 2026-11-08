package listening

import (
	"context"
	"testing"

	"github.com/ebrgr/TuneTrail/internal/db"
	"github.com/ebrgr/TuneTrail/internal/memkv"
	"github.com/ebrgr/TuneTrail/internal/models"
	"github.com/stretchr/testify/require"
)

type delayedMissingTrack struct {
	db.TrackStore
	started, resume chan struct{}
}

// GetTrack pauses a stale lookup while another request updates the cache.
func (s delayedMissingTrack) GetTrack(context.Context, db.GetTrackOpts) (*models.Track, error) {
	close(s.started)
	<-s.resume
	return nil, db.ErrNotFound
}

// TestStaleNowPlayingLookupDoesNotEraseNewUpdate covers different and repeated track IDs.
func TestStaleNowPlayingLookupDoesNotEraseNewUpdate(t *testing.T) {
	for _, replacement := range []int32{7, 8} {
		t.Run(string(rune('0'+replacement)), func(t *testing.T) {
			key := "now-playing:17919"
			memkv.Store.Set(key, int32(7))
			t.Cleanup(func() { memkv.Store.Delete(key) })
			store := delayedMissingTrack{started: make(chan struct{}), resume: make(chan struct{})}
			done := make(chan error, 1)
			go func() { _, err := ReadNowPlaying(db.WithProfile(context.Background(), 17919), store); done <- err }()
			<-store.started
			memkv.Store.Set(key, replacement)
			close(store.resume)
			require.NoError(t, <-done)
			got, ok := memkv.Store.Get(key)
			require.True(t, ok)
			require.Equal(t, replacement, got)
		})
	}
}
