package importer

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/ebrgr/TuneTrail/internal/export"
	"io"
	"os"
	"time"

	"github.com/ebrgr/TuneTrail/internal/catalog"
	"github.com/ebrgr/TuneTrail/internal/ingest"
	"github.com/google/uuid"
)

type shadowKey struct{}

const shadowBatchSize = 200

type fileShadow struct {
	admission interface {
		AdmitOccurrence(context.Context, ingest.Batch) ([]ingest.Result, error)
	}
	batch   ingest.Batch
	ordinal int
}

// withFileIngest hashes exact file bytes using an independent reader position.
func withFileIngest(ctx context.Context, store importStore, profile int32, format string, file *os.File) (context.Context, error) {
	info, err := file.Stat()
	if err != nil {
		return ctx, fmt.Errorf("stat import file: %w", err)
	}
	hash := sha256.New()
	if _, err := io.Copy(hash, &contextReader{ctx: ctx, reader: io.NewSectionReader(file, 0, info.Size())}); err != nil {
		return ctx, fmt.Errorf("hash import file: %w", err)
	}
	if result, ok := ctx.Value(resultKey{}).(*Result); ok {
		result.Active = true
	}
	return context.WithValue(ctx, shadowKey{}, &fileShadow{
		admission: store,
		batch: ingest.Batch{ProfileID: profile, Source: ingest.ShadowSourceFile, Operation: "import",
			FileDigest: hex.EncodeToString(hash.Sum(nil)), FileFormat: format, Items: make([]ingest.Item, 0, shadowBatchSize)},
		ordinal: -1,
	}), nil
}

// captureFileListen buffers a bounded page with absolute ordinals.
func captureFileListen(ctx context.Context, snapshot ingest.Snapshot, at time.Time, native ...export.KoitoListen) error {
	shadow, ok := ctx.Value(shadowKey{}).(*fileShadow)
	if !ok {
		return nil
	}
	if len(shadow.batch.Items) == 0 {
		shadow.batch.ReceivedAt = time.Now()
	}
	// Exclude local admission time from replay identity but retain it in the receipt.
	if snapshot.ReleaseMBID != "" || snapshot.MappedReleaseMBID != "" || len(snapshot.ArtistMBIDs) != 0 || len(snapshot.MappedArtistMBIDs) != 0 || len(snapshot.ArtistMbidMappings) != 0 {
		snapshot.MBIDSource = shadow.batch.FileFormat
		snapshot.MBIDDeclaredAt = shadow.batch.ReceivedAt.UTC().Format(time.RFC3339Nano)
	}
	item, err := prepareFileItem(shadow.ordinal, snapshot, at, native)
	if errors.Is(err, errFileItemEnvelopeTooLarge) {
		// Keep absolute ordinals even when this item cannot be admitted.
		if result, ok := ctx.Value(resultKey{}).(*Result); ok {
			result.RejectedOversizedItems++
		}
		return nil
	}
	if err != nil {
		return err
	}
	shadow.batch.Items = append(shadow.batch.Items, item)
	if len(shadow.batch.Items) == shadowBatchSize {
		return flushFileIngest(ctx)
	}
	return nil
}

// flushFileIngest commits a file page before the staging cursor can advance.
func flushFileIngest(ctx context.Context) error {
	job, ok := ctx.Value(shadowKey{}).(*fileShadow)
	if !ok || len(job.batch.Items) == 0 {
		return nil
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	results, err := job.admission.AdmitOccurrence(ctx, job.batch)
	if err != nil {
		return err
	}
	if result, ok := ctx.Value(resultKey{}).(*Result); ok {
		for _, item := range job.batch.Items {
			result.TruncatedFields += item.Snapshot.TruncatedFields
			result.IgnoredAliasLimit += item.Snapshot.IgnoredAliasLimit
		}
		for _, receipt := range results {
			result.IgnoredAliasConflicts += receipt.IgnoredAliasConflicts
			result.IgnoredMBIDConflicts += receipt.IgnoredMBIDConflicts
			if receipt.Pending {
				result.Pending++
			} else if receipt.Replay {
				result.Duplicates++
			}
		}
	}
	clear(job.batch.Items)
	job.batch.Items = job.batch.Items[:0]
	return nil
}

// submitFileListen records the observed fields before catalog enrichment.
func submitFileListen(ctx context.Context, opts catalog.SubmitListenOpts, originalTime string, enrich ...func(*ingest.Snapshot)) error {
	snapshot := ingest.Snapshot{Artist: opts.Artist, ArtistNames: opts.ArtistNames, Title: opts.TrackTitle,
		Release: opts.ReleaseTitle, DurationMS: int64(opts.Duration) * 1000, Client: opts.Client, OriginalTime: originalTime, SkipCacheImage: opts.SkipCacheImage}
	// Preserve the same identifiers and named credits passed to the legacy catalog.
	for _, id := range opts.ArtistMbzIDs {
		if id != uuid.Nil {
			snapshot.ArtistMBIDs = append(snapshot.ArtistMBIDs, id.String())
		}
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
	for _, apply := range enrich {
		apply(&snapshot)
	}
	return captureFileListen(ctx, snapshot, opts.Time)
}
