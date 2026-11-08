package sqlite

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/ebrgr/TuneTrail/internal/db"
)

func TestListensPropagatesCountCancellationForEveryFilter(t *testing.T) {
	s, _, _ := migrationDB(t, 3)
	ctx, cancel := context.WithCancel(db.WithProfile(context.Background(), 1))
	cancel()
	for _, opts := range []db.GetItemsOpts{{}, {TrackID: 1}, {AlbumID: 1}, {ArtistID: 1}} {
		opts.Page, opts.Limit = 1, 100
		result, err := s.GetListensPaginated(ctx, opts)
		if result != nil || !errors.Is(err, context.Canceled) || !strings.Contains(err.Error(), "count listens") {
			t.Fatalf("count failure hidden for %+v: result=%v error=%v", opts, result, err)
		}
	}
}
