package importer

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/ebrgr/TuneTrail/internal/catalog"
	"github.com/ebrgr/TuneTrail/internal/cfg"
	"github.com/ebrgr/TuneTrail/internal/db"
	"github.com/ebrgr/TuneTrail/internal/logger"
	"github.com/ebrgr/TuneTrail/internal/mbz"
	"github.com/google/uuid"
)

type LastFMExportPage struct {
	Track []LastFMTrack `json:"track"`
}
type LastFMTrack struct {
	Artist LastFMItem    `json:"artist"`
	Images []LastFMImage `json:"image"`
	MBID   string        `json:"mbid"`
	Album  LastFMItem    `json:"album"`
	Name   string        `json:"name"`
	Date   LastFMDate    `json:"date"`
}
type LastFMItem struct {
	MBID string `json:"mbid"`
	Text string `json:"#text"`
}
type LastFMDate struct {
	Unix string `json:"uts"`
	Text string `json:"#text"`
}
type LastFMImage struct {
	Size string `json:"size"`
	Url  string `json:"#text"`
}

func ImportLastFMFile(ctx context.Context, store importStore, mbzc mbz.MusicBrainzCaller, filename string) error {
	if db.ProfileID(ctx) <= 0 {
		return fmt.Errorf("explicit import profile required")
	}
	l := logger.FromContext(ctx)
	l.Info().Msgf("Beginning LastFM import on file: %s", filename)
	file, err := os.Open(importPath(ctx, filename))
	if err != nil {
		l.Err(err).Msgf("Failed to read import file: %s", filename)
		return fmt.Errorf("ImportLastFMFile: %w", err)
	}
	defer file.Close()

	export := make([]LastFMExportPage, 0)
	err = decodeJSON(ctx, file, &export)
	if err != nil {
		return fmt.Errorf("ImportLastFMFile: %w", err)
	}
	count := 0
	total := 0
	if len(export) > MaxImportItems {
		return invalidFile(fmt.Errorf("too many import items"))
	}
	for _, item := range export {
		if err := ctx.Err(); err != nil {
			return err
		}
		total += len(item.Track)
		if total > MaxImportItems {
			return invalidFile(fmt.Errorf("too many import items"))
		}
		for _, track := range item.Track {
			seen(ctx)
			if err := ctx.Err(); err != nil {
				return err
			}
			album := track.Album.Text
			if album == "" {
				album = track.Name
			}
			if track.Name == "" || track.Artist.Text == "" {
				l.Debug().Msg("Skipping invalid LastFM import item")
				ignored(ctx)
				continue
			}
			albumMbzID, err := uuid.Parse(track.Album.MBID)
			if err != nil {
				albumMbzID = uuid.Nil
			}
			artistMbzID, err := uuid.Parse(track.Artist.MBID)
			if err != nil {
				artistMbzID = uuid.Nil
			}
			trackMbzID, err := uuid.Parse(track.MBID)
			if err != nil {
				trackMbzID = uuid.Nil
			}
			var ts time.Time
			unix, err := strconv.ParseInt(track.Date.Unix, 10, 64)
			if err != nil {
				ts, err = time.Parse("02 Jan 2006, 15:04", track.Date.Text)
				if err != nil {
					l.Err(err).Msg("Could not parse time from listen activity, skipping...")
					ignored(ctx)
					continue
				}
			} else {
				ts = time.Unix(unix, 0).UTC()
			}
			if !inImportTimeWindow(ts) {
				l.Debug().Msgf("Skipping import due to import time rules")
				ignored(ctx)
				continue
			}

			var artistMbidMap []catalog.ArtistMbidMap
			if artistMbzID != uuid.Nil {
				artistMbidMap = append(artistMbidMap, catalog.ArtistMbidMap{Artist: track.Artist.Text, Mbid: artistMbzID})
			}

			opts := catalog.SubmitListenOpts{
				MbzCaller:          mbzc,
				Artist:             track.Artist.Text,
				ArtistMbzIDs:       []uuid.UUID{artistMbzID},
				TrackTitle:         track.Name,
				RecordingMbzID:     trackMbzID,
				ReleaseTitle:       album,
				ReleaseMbzID:       albumMbzID,
				ArtistMbidMappings: artistMbidMap,
				Client:             "lastfm",
				Time:               ts,
				ProfileID:          db.ProfileID(ctx),
				SkipCacheImage:     !cfg.FetchImagesDuringImport(),
			}
			err = catalog.SubmitListen(ctx, store, opts)
			if err != nil {
				l.Err(err).Msg("Failed to import LastFM playback item")
				return fmt.Errorf("ImportLastFMFile: %w", err)
			}
			count++
			if err := throttle(ctx); err != nil {
				return err
			}
		}
	}
	return finishImport(ctx, filename, count)
}
