package sqlite

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestIngestHealthBeforeDurableRelay guards the normal v11 startup path.
func TestIngestHealthBeforeDurableRelay(t *testing.T) {
	s, _, _ := migrationDB(t, 11)
	seedIdentityHistory(t, s)
	health, err := s.IngestHealth(t.Context(), 1)
	require.NoError(t, err)
	require.False(t, health.RelayTracked)
	require.Len(t, health.Sources, 1)
	require.Equal(t, "untracked", health.Sources[0].Delivery)
	require.Len(t, health.Decisions, 1)

	provenance, err := s.ListenProvenance(t.Context(), 1, 1, 42)
	require.NoError(t, err)
	require.Len(t, provenance, 1)
	require.Equal(t, "legacy", provenance[0].Origin)
	require.Equal(t, "untracked", provenance[0].Delivery)
	require.Nil(t, provenance[0].Received)
}
