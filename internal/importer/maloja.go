package importer

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/ebrgr/TuneTrail/internal/catalog"
	"github.com/ebrgr/TuneTrail/internal/cfg"
	"github.com/ebrgr/TuneTrail/internal/db"
	"github.com/ebrgr/TuneTrail/internal/logger"
	"github.com/ebrgr/TuneTrail/internal/mbz"
	"github.com/ebrgr/TuneTrail/internal/utils"
)

type MalojaExport struct {
	Scrobbles []MalojaExportItem `json:"scrobbles"`
}
type MalojaExportItem struct {
	Time  int64       `json:"time"`
	Track MalojaTrack `json:"track"`
}
type MalojaTrack struct {
	Artists []string `json:"artists"`
	Title   string   `json:"title"`
	Album   struct {
		Title string `json:"albumtitle"`
	} `json:"album"`
}

func ImportMalojaFile(ctx context.Context, store importStore, mbzc mbz.MusicBrainzCaller, filename string) error {
	if db.ProfileID(ctx) <= 0 {
		return fmt.Errorf("explicit import profile required")
	}
	l := logger.FromContext(ctx)
	l.Info().Msgf("Beginning maloja import on file: %s", filename)
	file, err := os.Open(importPath(ctx, filename))
	if err != nil {
		l.Err(err).Msgf("Failed to read import file: %s", filename)
		return fmt.Errorf("ImportMalojaFile: %w", err)
	}
	defer file.Close()

	export := new(MalojaExport)
	err = decodeJSON(ctx, file, &export)
	if err != nil {
		return fmt.Errorf("ImportMalojaFile: %w", err)
	}
	if len(export.Scrobbles) > MaxImportItems {
		return invalidFile(fmt.Errorf("too many import items"))
	}
	for _, item := range export.Scrobbles {
		seen(ctx)
		if err := ctx.Err(); err != nil {
			return err
		}
		martists := make([]string, 0)
		// Maloja has a tendency to have the the artist order ['feature', 'main \u2022 feature'], so
		// here we try to turn that artist array into ['main', 'feature']
		item.Track.Artists = utils.MoveFirstMatchToFront(item.Track.Artists, " \u2022 ")
		for _, an := range item.Track.Artists {
			ans := strings.Split(an, " \u2022 ")
			martists = append(martists, ans...)
		}
		artists := utils.UniqueIgnoringCase(martists)
		if len(item.Track.Artists) < 1 || item.Track.Title == "" {
			l.Debug().Msg("Skipping invalid maloja import item")
			ignored(ctx)
			continue
		}
		ts := time.Unix(item.Time, 0)
		if !inImportTimeWindow(ts) {
			l.Debug().Msgf("Skipping import due to import time rules")
			ignored(ctx)
			continue
		}
		opts := catalog.SubmitListenOpts{
			MbzCaller:      mbzc,
			Artist:         item.Track.Artists[0],
			ArtistNames:    artists,
			TrackTitle:     item.Track.Title,
			ReleaseTitle:   item.Track.Album.Title,
			Time:           ts.Local(),
			Client:         "maloja",
			ProfileID:      db.ProfileID(ctx),
			SkipCacheImage: !cfg.FetchImagesDuringImport(),
		}
		err = catalog.SubmitListen(ctx, store, opts)
		if err != nil {
			l.Err(err).Msg("Failed to import maloja playback item")
			return fmt.Errorf("ImportMalojaFile: %w", err)
		}
		if err := throttle(ctx); err != nil {
			return err
		}
	}
	return finishImport(ctx, filename, len(export.Scrobbles))
}
