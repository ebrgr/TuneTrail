package engine

import (
	"context"
	"time"

	"github.com/ebrgr/TuneTrail/internal/cfg"
	"github.com/ebrgr/TuneTrail/internal/enrichment"
	"github.com/ebrgr/TuneTrail/internal/logger"
	"github.com/ebrgr/TuneTrail/internal/mbz"
)

type enrichmentResolver interface {
	ConfigureEnrichmentProviders(context.Context, *enrichment.Registry) error
	ResolveNextEnrichmentChain(context.Context, *enrichment.Registry) (bool, error)
}

// enrichmentRegistry lists every source this process knows. A source that is not configured is
// registered as unavailable, so the stored registry still lists it; its order and enabled flag are
// owner choices kept in the database (ISSUE-214).
func enrichmentRegistry(ctx context.Context, mbzc mbz.MusicBrainzCaller) *enrichment.Registry {
	registry := enrichment.NewRegistry()
	var subsonic enrichment.Provider
	if cfg.SubsonicEnabled() {
		client, err := enrichment.NewSubsonic(cfg.SubsonicUrl(), cfg.SubsonicParams(), cfg.UserAgent(), 5*time.Second, time.Second)
		if err == nil {
			subsonic = client
		} else {
			logger.FromContext(ctx).Warn().Msg("Subsonic enrichment configuration unavailable")
		}
	}
	// The descriptors are constants and valid, so registration cannot fail.
	_ = registry.Register(enrichment.SubsonicDescriptor, subsonic)
	// Relay approval is per receipt and may change without restarting the engine.
	_ = registry.Register(enrichment.ListenBrainzDescriptor, enrichment.NewListenBrainzProvider(enrichment.NewListenBrainz(cfg.UserAgent(), nil)))
	// The resolver and every MusicBrainz provider draw from the one queue of the real client.
	if budget, ok := mbzc.(enrichment.Budget); ok {
		registry.SetBudget(enrichment.BudgetMusicBrainz, budget)
	}
	// A missing budget leaves the provider visible but unavailable.
	var musicBrainz enrichment.Provider
	if budget, ok := registry.Budget(enrichment.BudgetMusicBrainz); ok {
		musicBrainz = enrichment.NewMusicBrainz(cfg.MusicBrainzUrl(), cfg.UserAgent(), budget)
	}
	_ = registry.Register(enrichment.MusicBrainzDescriptor, musicBrainz)
	return registry
}

// startEnrichment owns a separate worker so external calls never delay admission or receipt recovery.
func startEnrichment(ctx context.Context, store any, mbzc mbz.MusicBrainzCaller) <-chan struct{} {
	done := make(chan struct{})
	resolver, ok := store.(enrichmentResolver)
	if !ok {
		close(done)
		return done
	}
	registry := enrichmentRegistry(ctx, mbzc)
	if err := resolver.ConfigureEnrichmentProviders(ctx, registry); err != nil {
		logger.FromContext(ctx).Warn().Msg("Enrichment queue configuration unavailable")
		close(done)
		return done
	}
	go func() {
		defer close(done)
		timer := time.NewTicker(time.Second)
		defer timer.Stop()
		for ctx.Err() == nil {
			found, err := resolver.ResolveNextEnrichmentChain(ctx, registry)
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
