package ingest

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestLegacyDigestOmitsCatalogTrack checks the v1 bytes for a native snapshot.
func TestLegacyDigestOmitsCatalogTrack(t *testing.T) {
	item := Item{Snapshot: Snapshot{Artist: "A", Title: "T", CatalogTrackID: 42}}
	_, digest := item.Encode()
	legacy := Digest([]byte(`{"artist":"A","title":"T"}`))
	require.NotEqual(t, legacy, digest)
	require.Equal(t, legacy, item.LegacyDigest(ShadowSourceFile, "koito"))
}
