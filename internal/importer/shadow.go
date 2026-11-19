package importer

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
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
	receiver ingest.Receiver
	batch    ingest.Batch
	ordinal  int
}

// withFileShadow hashes exact file bytes using an independent reader position.
func withFileShadow(ctx context.Context, store importStore, profile int32, format string, file *os.File) context.Context {
	receiver, ok := store.(ingest.Receiver)
	if !ok {
		return ctx
	}
	info, err := file.Stat()
	if err != nil {
		logger.FromContext(ctx).Warn().Msg("Could not stat import for shadow receipt")
		return ctx
	}
	hash := sha256.New()
	if _, err := io.Copy(hash, &contextReader{ctx: ctx, reader: io.NewSectionReader(file, 0, info.Size())}); err != nil {
		logger.FromContext(ctx).Warn().Msg("Could not hash import for shadow receipt")
		return ctx
	}
	return context.WithValue(ctx, shadowKey{}, &fileShadow{
		receiver: receiver,
		batch:    ingest.Batch{ProfileID: profile, Source: "file", Operation: "import", FileDigest: hex.EncodeToString(hash.Sum(nil)), FileFormat: format, Items: make([]ingest.Item, 0, shadowBatchSize)},
		ordinal:  -1,
	})
}

// captureFileListen buffers a bounded page with absolute ordinals.
func captureFileListen(ctx context.Context, snapshot ingest.Snapshot, at time.Time) {
	shadow, ok := ctx.Value(shadowKey{}).(*fileShadow)
	if !ok {
		return
	}
	if len(shadow.batch.Items) == 0 {
		shadow.batch.ReceivedAt = time.Now()
	}
	shadow.batch.Items = append(shadow.batch.Items, ingest.Item{Ordinal: shadow.ordinal, Snapshot: snapshot, ListenedAt: at})
	if len(shadow.batch.Items) == shadowBatchSize {
		flushFileShadow(ctx)
	}
}

// flushFileShadow commits full pages or the final partial page without affecting legacy status.
func flushFileShadow(ctx context.Context) {
	shadow, ok := ctx.Value(shadowKey{}).(*fileShadow)
	if !ok || len(shadow.batch.Items) == 0 || ctx.Err() != nil {
		return
	}
	results, err := shadow.receiver.ReceiveBatch(ctx, shadow.batch)
	clear(shadow.batch.Items)
	shadow.batch.Items = shadow.batch.Items[:0]
	if err != nil {
		logger.FromContext(ctx).Warn().Msg("Import shadow receipt batch failed")
		return
	}
	if result, ok := ctx.Value(resultKey{}).(*Result); ok {
		for _, receipt := range results {
			if receipt.Pending {
				result.Pending++
			}
		}
	}
}

// submitFileListen records the observed fields before catalog enrichment.
func submitFileListen(ctx context.Context, store importStore, opts catalog.SubmitListenOpts, originalTime string) error {
	snapshot := ingest.Snapshot{Artist: opts.Artist, ArtistNames: opts.ArtistNames, Title: opts.TrackTitle,
		Release: opts.ReleaseTitle, DurationMS: int64(opts.Duration) * 1000, Client: opts.Client, OriginalTime: originalTime}
	if opts.RecordingMbzID != uuid.Nil {
		snapshot.Recording = opts.RecordingMbzID.String()
	}
	captureFileListen(ctx, snapshot, opts.Time)
	return catalog.SubmitListen(ctx, store, opts)
}
