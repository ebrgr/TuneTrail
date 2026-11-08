package importer

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/ebrgr/TuneTrail/internal/catalog"
	"github.com/ebrgr/TuneTrail/internal/cfg"
	"github.com/ebrgr/TuneTrail/internal/db"
	"github.com/ebrgr/TuneTrail/internal/logger"
	mbz "github.com/ebrgr/TuneTrail/internal/mbz"
)

type SpotifyExportItem struct {
	Timestamp  time.Time `json:"ts"`
	TrackName  string    `json:"master_metadata_track_name"`
	ArtistName string    `json:"master_metadata_album_artist_name"`
	AlbumName  string    `json:"master_metadata_album_album_name"`
	ReasonEnd  string    `json:"reason_end"`
	MsPlayed   int32     `json:"ms_played"`
}

func ImportSpotifyFile(ctx context.Context, store importStore, mbzc mbz.MusicBrainzCaller, filename string) error {
	if db.ProfileID(ctx) <= 0 {
		return fmt.Errorf("explicit import profile required")
	}
	l := logger.FromContext(ctx)
	l.Info().Msgf("Beginning spotify import on file: %s", filename)
	file, err := os.Open(importPath(ctx, filename))
	if err != nil {
		l.Err(err).Msgf("Failed to read import file: %s", filename)
		return fmt.Errorf("ImportSpotifyFile: %w", err)
	}
	defer file.Close()

	export := make([]SpotifyExportItem, 0)
	err = decodeJSON(ctx, file, &export)
	if err != nil {
		return fmt.Errorf("ImportSpotifyFile: %w", err)
	}

	if len(export) > MaxImportItems {
		return invalidFile(fmt.Errorf("too many import items"))
	}
	for _, item := range export {
		seen(ctx)
		if err := ctx.Err(); err != nil {
			return err
		}
		if item.ReasonEnd != "trackdone" {
			ignored(ctx)
			continue
		}
		if !inImportTimeWindow(item.Timestamp) {
			l.Debug().Msgf("Skipping import due to import time rules")
			ignored(ctx)
			continue
		}
		dur := item.MsPlayed
		if item.TrackName == "" || item.ArtistName == "" {
			l.Debug().Msg("Skipping non-track item")
			ignored(ctx)
			continue
		}
		opts := catalog.SubmitListenOpts{
			MbzCaller:      mbzc,
			Artist:         item.ArtistName,
			TrackTitle:     item.TrackName,
			ReleaseTitle:   item.AlbumName,
			Duration:       dur / 1000,
			Time:           item.Timestamp,
			Client:         "spotify",
			ProfileID:      db.ProfileID(ctx),
			SkipCacheImage: !cfg.FetchImagesDuringImport(),
		}
		err = catalog.SubmitListen(ctx, store, opts)
		if err != nil {
			l.Err(err).Msg("Failed to import spotify playback item")
			return fmt.Errorf("ImportSpotifyFile: %w", err)
		}
		if err := throttle(ctx); err != nil {
			return err
		}
	}
	return finishImport(ctx, filename, len(export))
}
