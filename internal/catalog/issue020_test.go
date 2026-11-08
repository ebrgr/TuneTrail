package catalog_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ebrgr/TuneTrail/internal/catalog"
	"github.com/ebrgr/TuneTrail/internal/db"
	"github.com/ebrgr/TuneTrail/internal/db/sqlite"
	"github.com/ebrgr/TuneTrail/internal/mbz"
	"github.com/stretchr/testify/require"
)

var errAssociation = errors.New("simulated album association failure")

type failedAlbumLink struct{ *sqlite.Sqlite }

// AddArtistsToAlbum injects a failure after catalog association.
func (s failedAlbumLink) AddArtistsToAlbum(context.Context, db.AddArtistsToAlbumOpts) error {
	return errAssociation
}

// TestSubmitListenReportsAlbumAssociationFailure rejects success without a required relation.
func TestSubmitListenReportsAlbumAssociationFailure(t *testing.T) {
	s, err := sqlite.NewInMemory()
	require.NoError(t, err)
	defer s.Close(context.Background())
	require.NoError(t, s.Exec(`INSERT INTO listening_profiles(id,slug,display_name,created_at) VALUES(1,'alpha','Alpha',1)`))
	err = catalog.SubmitListen(context.Background(), failedAlbumLink{s}, catalog.SubmitListenOpts{MbzCaller: &mbz.MbzErrorCaller{}, Artist: "Test artist", TrackTitle: "Test track", ReleaseTitle: "Test album", Time: time.Unix(1700000000, 0), ProfileID: 1, SkipCacheImage: true})
	require.ErrorIs(t, err, errAssociation)
	n, err := s.ProfileListenCount(context.Background(), 1)
	require.NoError(t, err)
	require.Zero(t, n)
}

// TestAssociationGateWaitRespectsCancellation proves one blocked request cannot prevent shutdown forever.
func TestAssociationGateWaitRespectsCancellation(t *testing.T) {
	unlock, err := catalog.LockAssociations(context.Background())
	require.NoError(t, err)
	defer unlock()
	ctx, cancel := context.WithCancel(context.Background())
	started := make(chan struct{})
	done := make(chan error, 1)
	go func() {
		close(started)
		release, err := catalog.LockAssociations(ctx)
		if err == nil {
			release()
		}
		done <- err
	}()
	<-started
	cancel()
	select {
	case err := <-done:
		require.ErrorIs(t, err, context.Canceled)
	case <-time.After(time.Second):
		t.Fatal("waiting association gate did not cancel")
	}
}
