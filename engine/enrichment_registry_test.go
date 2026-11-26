package engine

import (
	"testing"

	"github.com/ebrgr/TuneTrail/internal/cfg"
	"github.com/ebrgr/TuneTrail/internal/enrichment"
	"github.com/ebrgr/TuneTrail/internal/mbz"
	"github.com/stretchr/testify/require"
)

// TestEnrichmentRegistryKnowsEverySource keeps unconfigured sources listed as unavailable, so the
// stored registry always shows what could be turned on (ISSUE-214).
func TestEnrichmentRegistryKnowsEverySource(t *testing.T) {
	registry := enrichmentRegistry(t.Context(), &mbz.MbzErrorCaller{})
	var ids []string
	for _, d := range registry.Descriptors() {
		ids = append(ids, d.ID)
	}
	require.Equal(t, []string{enrichment.SubsonicDescriptor.ID, enrichment.MusicBrainzDescriptor.ID, enrichment.ListenBrainzDescriptor.ID}, ids)
	require.True(t, registry.Available(enrichment.ListenBrainzDescriptor.ID), "each receipt still needs an approved relay")
	if !cfg.SubsonicEnabled() {
		require.False(t, registry.Available(enrichment.SubsonicDescriptor.ID))
	}
}

// TestEnrichmentRegistrySharesTheMusicBrainzQueue registers the real client's queue as the budget
// and nothing when MusicBrainz is disabled.
func TestEnrichmentRegistrySharesTheMusicBrainzQueue(t *testing.T) {
	_, ok := enrichmentRegistry(t.Context(), &mbz.MbzErrorCaller{}).Budget(enrichment.BudgetMusicBrainz)
	require.False(t, ok, "a disabled MusicBrainz has no budget to share")
	require.False(t, enrichmentRegistry(t.Context(), &mbz.MbzErrorCaller{}).Available(enrichment.MusicBrainzDescriptor.ID))

	client := mbz.NewMusicBrainzClient()
	defer client.Shutdown()
	enabled := enrichmentRegistry(t.Context(), client)
	require.True(t, enabled.Available(enrichment.MusicBrainzDescriptor.ID))
	budget, ok := enabled.Budget(enrichment.BudgetMusicBrainz)
	require.True(t, ok)
	require.Same(t, client, budget, "the resolver and the providers draw from the same queue")
}
