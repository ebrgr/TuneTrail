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
		batch:    ingest.Batch{ProfileID: profile, Source: "file", Operation: "import", FileDigest: hex.EncodeToString(hash.Sum(nil)), FileFormat: format},
		ordinal:  -1,
	})
}

// captureFileListen checkpoints the absolute ordinal without influencing legacy writes.
func captureFileListen(ctx context.Context, snapshot ingest.Snapshot, at time.Time) {
	shadow, ok := ctx.Value(shadowKey{}).(*fileShadow)
	if !ok {
		return
	}
	batch := shadow.batch
	batch.ReceivedAt = time.Now()
	batch.Items = []ingest.Item{{Ordinal: shadow.ordinal, Snapshot: snapshot, ListenedAt: at}}
	results, err := shadow.receiver.ReceiveBatch(ctx, batch)
	if err != nil {
		logger.FromContext(ctx).Warn().Msg("Import shadow receipt failed")
		return
	}
	if result, ok := ctx.Value(resultKey{}).(*Result); ok && len(results) == 1 && results[0].Pending {
		result.Pending++
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
