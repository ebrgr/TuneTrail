package sqlite

import (
	"context"
	"strings"
	"testing"

	"github.com/ebrgr/TuneTrail/internal/db"
)

func seedMediaRead(t *testing.T, s *Sqlite) {
	t.Helper()
	seedMusic(t, s)
	execSQL(t, s, `INSERT INTO release_aliases VALUES(1,'Album','User',1)`)
	execSQL(t, s, `INSERT INTO track_aliases VALUES(1,'Track','User',1)`)
	execSQL(t, s, `INSERT INTO artist_tracks VALUES(1,1,1)`)
}
func TestMediaReadsPropagateCountFailures(t *testing.T) {
	for _, kind := range []string{"artist", "album", "track"} {
		t.Run(kind, func(t *testing.T) {
			s, _, _ := migrationDB(t, 3)
			seedMediaRead(t, s)
			// Metadata remains valid. Only the personal aggregate is unavailable.
			execSQL(t, s, `ALTER TABLE listens RENAME TO unavailable_listens`)
			ctx := context.Background()
			var err error
			switch kind {
			case "artist":
				_, err = s.GetArtist(ctx, db.GetArtistOpts{ID: 1})
			case "album":
				_, err = s.GetAlbum(ctx, db.GetAlbumOpts{ID: 1})
			case "track":
				_, err = s.GetTrack(ctx, db.GetTrackOpts{ID: 1})
			}
			if err == nil || !strings.Contains(err.Error(), "count listens") {
				t.Fatalf("count failure became zero: %v", err)
			}
		})
	}
}
func TestArtistAndAlbumPropagateDurationAggregateFailures(t *testing.T) {
	for _, kind := range []string{"artist", "album"} {
		t.Run(kind, func(t *testing.T) {
			s, _, _ := migrationDB(t, 3)
			seedMediaRead(t, s)
			execSQL(t, s, `ALTER TABLE tracks DROP COLUMN duration`)
			var err error
			if kind == "artist" {
				_, err = s.GetArtist(context.Background(), db.GetArtistOpts{ID: 1})
			} else {
				_, err = s.GetAlbum(context.Background(), db.GetAlbumOpts{ID: 1})
			}
			if err == nil || !strings.Contains(err.Error(), "time listened") {
				t.Fatalf("duration failure became zero: %v", err)
			}
		})
	}
}
func TestUnplayedCatalogMetadataRetainsZeroRank(t *testing.T) {
	s, _, _ := migrationDB(t, 3)
	seedMediaRead(t, s)
	a, err := s.GetArtist(context.Background(), db.GetArtistOpts{ID: 1})
	if err != nil || a.AllTimeRank != 0 {
		t.Fatalf("unplayed artist: %+v %v", a, err)
	}
	b, err := s.GetAlbum(context.Background(), db.GetAlbumOpts{ID: 1})
	if err != nil || b.AllTimeRank != 0 {
		t.Fatalf("unplayed album: %+v %v", b, err)
	}
	tr, err := s.GetTrack(context.Background(), db.GetTrackOpts{ID: 1})
	if err != nil || tr.AllTimeRank != 0 {
		t.Fatalf("unplayed track: %+v %v", tr, err)
	}
}
