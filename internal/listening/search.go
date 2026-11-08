package listening

import (
	"context"
	"errors"
	"github.com/ebrgr/TuneTrail/internal/db"
	"github.com/ebrgr/TuneTrail/internal/models"
	"strconv"
	"strings"
)

type SearchResults struct {
	Artists []*models.Artist `json:"artists"`
	Albums  []*models.Album  `json:"albums"`
	Tracks  []*models.Track  `json:"tracks"`
}

func ReadSearch(ctx context.Context, s StatsStore, q string) (SearchResults, error) {
	result := SearchResults{Artists: []*models.Artist{}, Albums: []*models.Album{}, Tracks: []*models.Track{}}
	if q == "" {
		return result, nil
	}
	if strings.HasPrefix(q, "id:") {
		id, e := strconv.ParseInt(strings.TrimPrefix(q, "id:"), 10, 32)
		if e != nil || id <= 0 {
			return result, nil
		}
		a, e := s.GetArtist(ctx, db.GetArtistOpts{ID: int32(id)})
		if e != nil && !errors.Is(e, db.ErrNotFound) {
			return result, e
		}
		if a != nil {
			result.Artists = append(result.Artists, a)
		}
		b, e := s.GetAlbum(ctx, db.GetAlbumOpts{ID: int32(id)})
		if e != nil && !errors.Is(e, db.ErrNotFound) {
			return result, e
		}
		if b != nil {
			result.Albums = append(result.Albums, b)
		}
		tr, e := s.GetTrack(ctx, db.GetTrackOpts{ID: int32(id)})
		if e != nil && !errors.Is(e, db.ErrNotFound) {
			return result, e
		}
		if tr != nil {
			result.Tracks = append(result.Tracks, tr)
		}
		return result, nil
	}
	var e error
	result.Artists, e = s.SearchArtists(ctx, q)
	if e != nil {
		return result, e
	}
	result.Albums, e = s.SearchAlbums(ctx, q)
	if e != nil {
		return result, e
	}
	result.Tracks, e = s.SearchTracks(ctx, q)
	return result, e
}
