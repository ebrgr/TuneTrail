package importer

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/ebrgr/TuneTrail/internal/catalog"
	"github.com/ebrgr/TuneTrail/internal/cfg"
	"github.com/ebrgr/TuneTrail/internal/db"
	"github.com/ebrgr/TuneTrail/internal/ingest"
	"github.com/ebrgr/TuneTrail/internal/logger"
	mbz "github.com/ebrgr/TuneTrail/internal/mbz"
)

type SpotifyExportItem struct {
	rawTimestamp string
	Timestamp    time.Time `json:"ts"`
	TrackName    string    `json:"master_metadata_track_name"`
	ArtistName   string    `json:"master_metadata_album_artist_name"`
	AlbumName    string    `json:"master_metadata_album_album_name"`
	ReasonEnd    string    `json:"reason_end"`
	MsPlayed     int32     `json:"ms_played"`
}

// UnmarshalJSON captures the source timestamp before time.Time normalizes its spelling.
func (item *SpotifyExportItem) UnmarshalJSON(data []byte) error {
	type row SpotifyExportItem
	var observed struct {
		*row
		Timestamp string `json:"ts"`
	}
	observed.row = (*row)(item)
	if err := json.Unmarshal(data, &observed); err != nil {
		return err
	}
	item.rawTimestamp = observed.Timestamp
	if observed.Timestamp == "" {
		return nil
	}
	at, err := time.Parse(time.RFC3339Nano, observed.Timestamp)
	item.Timestamp = at
	return err
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
	ctx, err = withFileShadow(ctx, store, db.ProfileID(ctx), "spotify", file)
	if err != nil {
		return err
	}
	defer flushLegacyFileShadow(ctx)
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
		err = submitFileListen(ctx, store, opts, item.Timestamp.Format(time.RFC3339Nano), func(snapshot *ingest.Snapshot) {
			snapshot.ArtistNames = []string{item.ArtistName}
			snapshot.OriginalTime = item.rawTimestamp
			snapshot.DurationMS = int64(item.MsPlayed)
			snapshot.TimeSemantics = "end"
		})
		if err != nil {
			l.Err(err).Msg("Failed to import spotify playback item")
			return fmt.Errorf("ImportSpotifyFile: %w", err)
		}
		if err := throttle(ctx); err != nil {
			return err
		}
	}
	if err := flushFileShadow(ctx); err != nil {
		return err
	}
	return finishImport(ctx, filename, len(export))
}
