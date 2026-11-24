package engine

import (
	"context"
	"time"

	"github.com/ebrgr/TuneTrail/internal/cfg"
	"github.com/ebrgr/TuneTrail/internal/enrichment"
	"github.com/ebrgr/TuneTrail/internal/logger"
)

type enrichmentResolver interface {
	CountUnverifiedMetadata(context.Context) (int64, error)
	ConfigureEnrichment(context.Context, bool) error
	ResolveNextEnrichment(context.Context, enrichment.Provider) (bool, error)
}

// startEnrichment owns a separate worker so external calls never delay admission or receipt recovery.
func startEnrichment(ctx context.Context, store any) <-chan struct{} {
	done := make(chan struct{})
	resolver, ok := store.(enrichmentResolver)
	if !ok {
		close(done)
		return done
	}
	// Surface the migration result even while provider enrichment is disabled.
	if count, err := resolver.CountUnverifiedMetadata(ctx); err != nil {
		logger.FromContext(ctx).Warn().Msg("Could not check archived metadata provenance")
	} else if count > 0 {
		logger.FromContext(ctx).Warn().Int64("count", count).Msg("Historical metadata without a verified listen is retained outside the active catalog")
	}
	var provider enrichment.Provider
	if cfg.SubsonicEnabled() {
		client, err := enrichment.NewSubsonic(cfg.SubsonicUrl(), cfg.SubsonicParams(), cfg.UserAgent(), 5*time.Second, time.Second)
		if err == nil {
			provider = client
		} else {
			logger.FromContext(ctx).Warn().Msg("Subsonic enrichment configuration unavailable")
		}
	}
	if err := resolver.ConfigureEnrichment(ctx, provider != nil); err != nil {
		logger.FromContext(ctx).Warn().Msg("Enrichment queue configuration unavailable")
		close(done)
		return done
	}
	if provider == nil {
		close(done)
		return done
	}
	go func() {
		defer close(done)
		timer := time.NewTicker(time.Second)
		defer timer.Stop()
		for ctx.Err() == nil {
			found, err := resolver.ResolveNextEnrichment(ctx, provider)
			if err != nil && ctx.Err() == nil {
				logger.FromContext(ctx).Warn().Msg("Enrichment work could not be stored")
			}
			if err != nil || !found {
				select {
				case <-ctx.Done():
					return
				case <-timer.C:
				}
			}
		}
	}()
	return done
}
