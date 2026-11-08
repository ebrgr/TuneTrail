package importer

import (
	"archive/zip"
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/ebrgr/TuneTrail/internal/catalog"
	"github.com/ebrgr/TuneTrail/internal/cfg"
	"github.com/ebrgr/TuneTrail/internal/db"
	"github.com/ebrgr/TuneTrail/internal/listenbrainz"
	"github.com/ebrgr/TuneTrail/internal/logger"
	"github.com/ebrgr/TuneTrail/internal/mbz"
	"github.com/ebrgr/TuneTrail/internal/utils"
	"github.com/google/uuid"
)

func ImportListenBrainzExport(ctx context.Context, store importStore, mbzc mbz.MusicBrainzCaller, filename string) error {
	if db.ProfileID(ctx) <= 0 {
		return fmt.Errorf("explicit import profile required")
	}
	l := logger.FromContext(ctx)

	r, err := zip.OpenReader(importPath(ctx, filename))
	if err != nil {
		if errors.Is(err, zip.ErrFormat) {
			return invalidFile(err)
		}
		return err
	}
	defer r.Close()

	const maxMembers = 1000
	const maxExpanded = 256 << 20
	if len(r.File) > maxMembers {
		return invalidFile(fmt.Errorf("too many archive members"))
	}
	matched := 0
	var declared uint64
	for _, f := range r.File {
		if f.UncompressedSize64 > maxExpanded || declared > maxExpanded-f.UncompressedSize64 {
			return invalidFile(fmt.Errorf("archive exceeds expanded limit"))
		}
		declared += f.UncompressedSize64
	}
	budget := &io.LimitedReader{N: maxExpanded + 1}
	totalLines := 0
	ctx = context.WithValue(ctx, archiveLinesKey{}, &totalLines)
	for _, f := range r.File {
		if err := ctx.Err(); err != nil {
			return err
		}

		if f.FileInfo().IsDir() {
			l.Debug().Msgf("File %s is dir, skipping...", f.Name)
			continue
		}

		if strings.HasPrefix(f.Name, "listens/") && strings.HasSuffix(f.Name, ".jsonl") {
			matched++
			l.Info().Msgf("Found: %s\n", f.Name)

			rc, err := f.Open()
			if err != nil {
				if errors.Is(err, zip.ErrAlgorithm) || errors.Is(err, zip.ErrFormat) {
					return invalidFile(err)
				}
				return fmt.Errorf("open archive member: %w", err)
			}

			budget.R = rc
			err = ImportListenBrainzFile(ctx, store, mbzc, &contextReader{ctx: ctx, reader: budget}, f.Name)
			if err == nil && budget.N <= 0 {
				err = invalidFile(fmt.Errorf("archive exceeds expanded limit"))
			}
			if err != nil {
				rc.Close()
				return fmt.Errorf("import archive member: %w", err)
			}

			rc.Close()
		}
	}
	if matched == 0 {
		return invalidFile(fmt.Errorf("archive has no ListenBrainz listen files"))
	}
	return finishImport(ctx, filename, 0)
}

type archiveLinesKey struct{}

func ImportListenBrainzFile(ctx context.Context, store importStore, mbzc mbz.MusicBrainzCaller, r io.Reader, filename string) error {
	if db.ProfileID(ctx) <= 0 {
		return fmt.Errorf("explicit import profile required")
	}
	l := logger.FromContext(ctx)
	l.Info().Msgf("Beginning ListenBrainz import on file: %s", filename)

	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 4096), 1<<20)
	count := 0
	for scanner.Scan() {
		seen(ctx)
		if err := ctx.Err(); err != nil {
			return err
		}
		count++
		if total, ok := ctx.Value(archiveLinesKey{}).(*int); ok {
			*total++
			if *total > MaxImportItems {
				return invalidFile(fmt.Errorf("too many archive lines"))
			}
		}
		if count > MaxImportItems {
			return invalidFile(fmt.Errorf("too many archive lines"))
		}
		line := scanner.Bytes()
		payload := new(listenbrainz.LbzSubmitListenPayload)
		err := json.Unmarshal(line, payload)
		if err != nil {
			l.Err(err).Msg("Error unmarshaling JSON")
			ignored(ctx)
			continue
		}
		if strings.TrimSpace(payload.TrackMeta.ArtistName) == "" || strings.TrimSpace(payload.TrackMeta.TrackName) == "" {
			ignored(ctx)
			continue
		}
		ts := time.Unix(payload.ListenedAt, 0)
		if !inImportTimeWindow(ts) {
			l.Debug().Msgf("Skipping import due to import time rules")
			ignored(ctx)
			continue
		}
		artistMbzIDs, err := utils.ParseUUIDSlice(payload.TrackMeta.AdditionalInfo.ArtistMBIDs)
		if err != nil {
			l.Debug().AnErr("error", err).Msg("ImportListenBrainzFile: Failed to parse one or more UUIDs")
		}
		if len(artistMbzIDs) < 1 {
			l.Debug().AnErr("error", err).Msg("ImportListenBrainzFile: Attempting to parse artist UUIDs from mbid_mapping")
			artistMbzIDs, err = utils.ParseUUIDSlice(payload.TrackMeta.MBIDMapping.ArtistMBIDs)
			if err != nil {
				l.Debug().AnErr("error", err).Msg("ImportListenBrainzFile: Failed to parse one or more UUIDs")
			}
		}
		rgMbzID, err := uuid.Parse(payload.TrackMeta.AdditionalInfo.ReleaseGroupMBID)
		if err != nil {
			rgMbzID = uuid.Nil
		}
		releaseMbzID, err := uuid.Parse(payload.TrackMeta.AdditionalInfo.ReleaseMBID)
		if err != nil {
			releaseMbzID, err = uuid.Parse(payload.TrackMeta.MBIDMapping.ReleaseMBID)
			if err != nil {
				releaseMbzID = uuid.Nil
			}
		}
		recordingMbzID, err := uuid.Parse(payload.TrackMeta.AdditionalInfo.RecordingMBID)
		if err != nil {
			recordingMbzID, err = uuid.Parse(payload.TrackMeta.MBIDMapping.RecordingMBID)
			if err != nil {
				recordingMbzID = uuid.Nil
			}
		}

		var client string
		if payload.TrackMeta.AdditionalInfo.MediaPlayer != "" {
			client = payload.TrackMeta.AdditionalInfo.MediaPlayer
		} else if payload.TrackMeta.AdditionalInfo.SubmissionClient != "" {
			client = payload.TrackMeta.AdditionalInfo.SubmissionClient
		}

		var duration int32
		if payload.TrackMeta.AdditionalInfo.Duration != 0 {
			duration = payload.TrackMeta.AdditionalInfo.Duration
		} else if payload.TrackMeta.AdditionalInfo.DurationMs != 0 {
			duration = payload.TrackMeta.AdditionalInfo.DurationMs / 1000
		}

		var artistMbidMap []catalog.ArtistMbidMap
		for _, a := range payload.TrackMeta.MBIDMapping.Artists {
			if a.ArtistMBID == "" || a.ArtistName == "" {
				continue
			}
			mbid, err := uuid.Parse(a.ArtistMBID)
			if err != nil {
				l.Err(err).Msgf("LbzSubmitListenHandler: Failed to parse UUID for artist '%s'", a.ArtistName)
			}
			artistMbidMap = append(artistMbidMap, catalog.ArtistMbidMap{Artist: a.ArtistName, Mbid: mbid})
		}

		opts := catalog.SubmitListenOpts{
			MbzCaller:          mbzc,
			ArtistNames:        payload.TrackMeta.AdditionalInfo.ArtistNames,
			Artist:             payload.TrackMeta.ArtistName,
			ArtistMbzIDs:       artistMbzIDs,
			TrackTitle:         payload.TrackMeta.TrackName,
			RecordingMbzID:     recordingMbzID,
			ReleaseTitle:       payload.TrackMeta.ReleaseName,
			ReleaseMbzID:       releaseMbzID,
			ReleaseGroupMbzID:  rgMbzID,
			ArtistMbidMappings: artistMbidMap,
			Duration:           duration,
			Time:               ts,
			ProfileID:          db.ProfileID(ctx),
			Client:             client,
			SkipCacheImage:     !cfg.FetchImagesDuringImport(),
		}
		err = catalog.SubmitListen(ctx, store, opts)
		if err != nil {
			l.Err(err).Msg("Failed to import LastFM playback item")
			return fmt.Errorf("ImportListenBrainzFile: %w", err)
		}
		if err := throttle(ctx); err != nil {
			return err
		}
	}
	if err := scanner.Err(); err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return invalidFile(err)
	}
	l.Info().Msgf("Finished importing %s; imported %d items", filename, count)
	return nil
}
