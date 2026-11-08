// Package listening composes profile-scoped reads independently of HTTP or HTML.
package listening

import (
	"context"
	"fmt"
	"github.com/ebrgr/TuneTrail/internal/db"
)

type Stats struct {
	ListenCount     int64   `json:"listen_count"`
	TrackCount      int64   `json:"track_count"`
	AlbumCount      int64   `json:"album_count"`
	ArtistCount     int64   `json:"artist_count"`
	MinutesListened int64   `json:"minutes_listened"`
	DaysActive      int     `json:"days_active"`
	LongestStreak   int     `json:"longest_streak"`
	AvgDailyPlays   float32 `json:"avg_daily_plays"`
	TracksPerArtist float32 `json:"tracks_per_artist"`
	AlbumsPerArtist float32 `json:"albums_per_artist"`
}
type StatsStore interface {
	db.ListenStore
	db.TrackStore
	db.AlbumStore
	db.ArtistStore
}

func Ratio(a, b int64) float32 {
	if b == 0 {
		return 0
	}
	return float32(a) / float32(b)
}
func ReadStats(ctx context.Context, s StatsStore, tf db.Timeframe) (Stats, error) {
	var v Stats
	counts := []struct {
		dst *int64
		get func(context.Context, db.Timeframe) (int64, error)
	}{{&v.ListenCount, s.CountListens}, {&v.TrackCount, s.CountTracks}, {&v.AlbumCount, s.CountAlbums}, {&v.ArtistCount, s.CountArtists}}
	for _, c := range counts {
		n, err := c.get(ctx, tf)
		if err != nil {
			return v, fmt.Errorf("statistics: %w", err)
		}
		*c.dst = n
	}
	seconds, err := s.CountTimeListened(ctx, tf)
	if err != nil {
		return v, err
	}
	v.MinutesListened = seconds / 60
	v.DaysActive, err = s.GetActiveDays(ctx, tf.Timezone)
	if err != nil {
		return v, err
	}
	v.LongestStreak, err = s.GetLongestListenStreak(ctx, db.ListenActivityOpts{Timezone: tf.Timezone})
	if err != nil {
		return v, err
	}
	v.AvgDailyPlays = Ratio(v.ListenCount, int64(v.DaysActive))
	v.TracksPerArtist = Ratio(v.TrackCount, v.ArtistCount)
	v.AlbumsPerArtist = Ratio(v.AlbumCount, v.ArtistCount)
	return v, nil
}
