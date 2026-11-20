package engine

import (
	"context"
	"time"

	"github.com/ebrgr/TuneTrail/internal/logger"
	"github.com/ebrgr/TuneTrail/internal/mbz"
)

type ingestResolver interface {
	ResolveNextIngest(context.Context, mbz.MusicBrainzCaller) (bool, error)
}

// startIngestRecovery drains bounded bursts and lets the engine join it on shutdown.
func startIngestRecovery(ctx context.Context, store any, mbzc mbz.MusicBrainzCaller) <-chan struct{} {
	done := make(chan struct{})
	resolver, ok := store.(ingestResolver)
	if !ok {
		close(done)
		return done
	}
	go func() {
		defer close(done)
		timer := time.NewTicker(time.Second)
		defer timer.Stop()
		for ctx.Err() == nil {
			for attempt := 0; attempt < 32 && ctx.Err() == nil; attempt++ {
				found, err := resolver.ResolveNextIngest(ctx, mbzc)
				if err != nil {
					if ctx.Err() == nil {
						logger.FromContext(ctx).Warn().Msg("Active ingest resolution failed; receipt remains durable")
					}
					break
				}
				if !found {
					break
				}
			}
			select {
			case <-ctx.Done():
				return
			case <-timer.C:
			}
		}
	}()
	return done
}
