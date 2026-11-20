package importer

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"github.com/ebrgr/TuneTrail/internal/cfg"
	"io"
	"os"
	"time"

	"github.com/ebrgr/TuneTrail/internal/catalog"
	"github.com/ebrgr/TuneTrail/internal/ingest"
	"github.com/ebrgr/TuneTrail/internal/logger"
	"github.com/google/uuid"
)

type shadowKey struct{}

const shadowBatchSize = 200

type fileShadow struct {
	receiver  ingest.Receiver
	admission ingest.Admission
	active    bool
	batch     ingest.Batch
	ordinal   int
}

// withFileShadow hashes exact file bytes using an independent reader position.
func withFileShadow(ctx context.Context, store importStore, profile int32, format string, file *os.File) (context.Context, error) {
	receiver, ok := store.(ingest.Receiver)
	if !ok {
		return ctx, nil
	}
	admission, _ := store.(ingest.Admission)
	active := false
	// Check the profile cut even with the master switch off.
	if admission != nil {
		var err error
		active, err = admission.BeginImportIngest(ctx, profile, cfg.ActiveIngestEnabled())
		if err != nil {
			return ctx, err
		}
	}
	if result, ok := ctx.Value(resultKey{}).(*Result); ok {
		result.Active = active
	}
	info, err := file.Stat()
	if err != nil {
		if active {
			return ctx, fmt.Errorf("stat active import: %w", err)
		}
		logger.FromContext(ctx).Warn().Msg("Could not stat import for shadow receipt")
		return ctx, nil
	}
	hash := sha256.New()
	if _, err := io.Copy(hash, &contextReader{ctx: ctx, reader: io.NewSectionReader(file, 0, info.Size())}); err != nil {
		if active {
			return ctx, fmt.Errorf("hash active import: %w", err)
		}
		logger.FromContext(ctx).Warn().Msg("Could not hash import for shadow receipt")
		return ctx, nil
	}
	return context.WithValue(ctx, shadowKey{}, &fileShadow{
		receiver: receiver, admission: admission, active: active,
		batch:   ingest.Batch{ProfileID: profile, Source: ingest.ShadowSourceFile, Operation: "import", FileDigest: hex.EncodeToString(hash.Sum(nil)), FileFormat: format, Items: make([]ingest.Item, 0, shadowBatchSize)},
		ordinal: -1,
	}), nil
}

// activeFileIngest reports the mode pinned before reading this job's items.
func activeFileIngest(ctx context.Context) bool {
	job, ok := ctx.Value(shadowKey{}).(*fileShadow)
	return ok && job.active
}

// captureFileListen buffers a bounded page with absolute ordinals.
func captureFileListen(ctx context.Context, snapshot ingest.Snapshot, at time.Time) error {
	shadow, ok := ctx.Value(shadowKey{}).(*fileShadow)
	if !ok {
		return nil
	}
	if len(shadow.batch.Items) == 0 {
		shadow.batch.ReceivedAt = time.Now()
	}
	item := ingest.Item{Ordinal: shadow.ordinal, Snapshot: snapshot, ListenedAt: at}
	if shadow.active {
		if snapshot.Artist == "" || snapshot.Title == "" {
			return invalidFile(fmt.Errorf("listen at ordinal %d requires an artist and title", item.Ordinal))
		}
		encoded, _ := item.Encode()
		if len(encoded) > 10<<10 {
			return invalidFile(fmt.Errorf("listen at ordinal %d exceeds 10 KiB", item.Ordinal))
		}
	}
	shadow.batch.Items = append(shadow.batch.Items, item)
	if len(shadow.batch.Items) == shadowBatchSize {
		return flushFileShadow(ctx)
	}
	return nil
}

// flushLegacyFileShadow preserves the partial-page behavior of the legacy parser.
func flushLegacyFileShadow(ctx context.Context) {
	if !activeFileIngest(ctx) {
		_ = flushFileShadow(ctx) // Shadow errors are logged and never affect the import.
	}
}

// flushFileShadow admits a page, with active failures returned to the importer.
func flushFileShadow(ctx context.Context) error {
	shadow, ok := ctx.Value(shadowKey{}).(*fileShadow)
	if !ok || len(shadow.batch.Items) == 0 {
		return nil
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	var results []ingest.Result
	var err error
	if shadow.active {
		_, results, err = shadow.admission.AdmitIngest(ctx, shadow.batch, true, &shadow.active, nil)
	} else {
		results, err = shadow.receiver.ReceiveBatch(ctx, shadow.batch)
	}
	clear(shadow.batch.Items)
	shadow.batch.Items = shadow.batch.Items[:0]
	if err != nil {
		if shadow.active {
			return err
		}
		logger.FromContext(ctx).Warn().Msg("Import shadow receipt batch failed")
		return nil
	}
	if result, ok := ctx.Value(resultKey{}).(*Result); ok {
		for _, receipt := range results {
			if receipt.Pending {
				result.Pending++
			} else if shadow.active && receipt.Replay {
				result.Duplicates++
			}
		}
	}
	return nil
}

// submitFileListen records the observed fields before catalog enrichment.
func submitFileListen(ctx context.Context, store importStore, opts catalog.SubmitListenOpts, originalTime string) error {
	snapshot := ingest.Snapshot{Artist: opts.Artist, ArtistNames: opts.ArtistNames, Title: opts.TrackTitle,
		Release: opts.ReleaseTitle, DurationMS: int64(opts.Duration) * 1000, Client: opts.Client, OriginalTime: originalTime, SkipCacheImage: opts.SkipCacheImage}
	// Preserve the same identifiers and named credits passed to the legacy catalog.
	for _, id := range opts.ArtistMbzIDs {
		snapshot.ArtistMBIDs = append(snapshot.ArtistMBIDs, id.String())
	}
	for _, credit := range opts.ArtistMbidMappings {
		snapshot.ArtistMbidMappings = append(snapshot.ArtistMbidMappings, ingest.ArtistMapping{Name: credit.Artist, MBID: credit.Mbid.String()})
	}
	if opts.ReleaseMbzID != uuid.Nil {
		snapshot.ReleaseMBID = opts.ReleaseMbzID.String()
	}
	if opts.ReleaseGroupMbzID != uuid.Nil {
		snapshot.ReleaseGroupMBID = opts.ReleaseGroupMbzID.String()
	}
	if opts.RecordingMbzID != uuid.Nil {
		snapshot.Recording = opts.RecordingMbzID.String()
	}
	if err := captureFileListen(ctx, snapshot, opts.Time); err != nil || activeFileIngest(ctx) {
		return err
	}
	return catalog.SubmitListen(ctx, store, opts)
}
