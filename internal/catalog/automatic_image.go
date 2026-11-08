package catalog

import (
	"context"
	"fmt"

	"github.com/ebrgr/TuneTrail/imagecache"
	"github.com/ebrgr/TuneTrail/internal/logger"
	"github.com/google/uuid"
)

// saveWithAutomaticImage protects an in-flight ID until the database operation
// finishes. Downloads happen outside the publication gate; failed downloads keep
// their source association for later recovery. Failed saves are left for the
// collector to check, since the database may have committed before returning an error.
func saveWithAutomaticImage[T any](ctx context.Context, id uuid.UUID, source string, skip bool, save func() (T, error)) (T, error) {
	var zero T
	if id == uuid.Nil || source == "" || skip {
		return save()
	}
	release, err := imagecache.ReservePublication(ctx, id)
	if err != nil {
		return zero, fmt.Errorf("reserve automatic image: %w", err)
	}
	defer release()
	if err := imagecache.DownloadImageContext(ctx, id, source); err != nil {
		logger.FromContext(ctx).Warn().Err(err).Msg("Failed to cache automatic image; the source remains available for retry")
		if ctx.Err() != nil {
			return zero, ctx.Err()
		}
	}
	if err := ctx.Err(); err != nil {
		return zero, err
	}
	return save()
}
