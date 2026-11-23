package importer

import (
	"archive/zip"
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/ebrgr/TuneTrail/internal/cfg"
	"github.com/ebrgr/TuneTrail/internal/db"
	"github.com/ebrgr/TuneTrail/internal/ingest"
	"github.com/ebrgr/TuneTrail/internal/listenbrainz"
	"github.com/ebrgr/TuneTrail/internal/logger"
	"github.com/ebrgr/TuneTrail/internal/mbz"
)

// ImportListenBrainzExport uses one ordinal stream across the archive members.
func ImportListenBrainzExport(ctx context.Context, store importStore, mbzc mbz.MusicBrainzCaller, filename string) error {
	if db.ProfileID(ctx) <= 0 {
		return fmt.Errorf("explicit import profile required")
	}
	l := logger.FromContext(ctx)

	file, err := os.Open(importPath(ctx, filename))
	if err != nil {
		return err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return err
	}
	r, err := zip.NewReader(file, info.Size())
	if err != nil {
		if errors.Is(err, zip.ErrFormat) {
			return invalidFile(err)
		}
		return err
	}

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
	ctx, err = withFileIngest(ctx, store, db.ProfileID(ctx), "listenbrainz", file)
	if err != nil {
		return err
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
	if err := flushFileIngest(ctx); err != nil {
		return err
	}
	return finishImport(ctx, filename, 0)
}

type archiveLinesKey struct{}

// ImportListenBrainzFile preserves the source timestamp in the shadow snapshot.
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
		ts := time.Unix(int64(payload.ListenedAt), 0)
		if !inImportTimeWindow(ts) {
			l.Debug().Msgf("Skipping import due to import time rules")
			ignored(ctx)
			continue
		}
		var observedAt time.Time
		if payload.ListenedAt != 0 {
			observedAt = time.Unix(int64(payload.ListenedAt), 0)
		}
		// Active files write only through durable receipt admission.
		snapshot := ingest.ListenBrainzSnapshot(payload.TrackMeta)
		// Export mapping can carry individual credits when additional_info omits them.
		if len(snapshot.ArtistNames) == 0 {
			for _, credit := range snapshot.ArtistMbidMappings {
				snapshot.ArtistNames = append(snapshot.ArtistNames, credit.Name)
			}
		}
		// An explicitly identified release can use the legacy display fallback.
		if snapshot.Release == "" && (snapshot.ReleaseMBID != "" || snapshot.MappedReleaseMBID != "") {
			snapshot.Release = snapshot.Title
		}
		// Keep zero-value timestamps missing in receipts, as in the legacy shadow writer.
		if payload.ListenedAt != 0 {
			snapshot.OriginalTime = payload.OriginalListenedAt
		}
		snapshot.SkipCacheImage = !cfg.FetchImagesDuringImport()
		err = captureFileListen(ctx, snapshot, observedAt)
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
