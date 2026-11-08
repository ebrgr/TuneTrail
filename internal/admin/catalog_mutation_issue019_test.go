package admin

import (
	"context"
	"github.com/ebrgr/TuneTrail/internal/db"
	"github.com/ebrgr/TuneTrail/internal/memkv"
	"github.com/ebrgr/TuneTrail/internal/models"
	"github.com/stretchr/testify/require"
	"testing"
)

type cacheMutationStore struct{ replacement int32 }

// PreviewCatalogMutation satisfies the review interface without performing I/O.
func (s cacheMutationStore) PreviewCatalogMutation(_ context.Context, p db.CatalogPlan) (db.CatalogPlan, error) {
	return p, nil
}

// ApplyCatalogMutation simulates a new playback update during the transaction.
func (s cacheMutationStore) ApplyCatalogMutation(context.Context, db.CatalogPlan) error {
	memkv.Store.Set("now-playing:19001", s.replacement)
	return nil
}

// ListListeningProfiles returns two isolated profiles for cache invalidation.
func (s cacheMutationStore) ListListeningProfiles(context.Context) ([]models.ListeningProfile, error) {
	return []models.ListeningProfile{{ID: 19001}, {ID: 19002}}, nil
}

// TestCatalogMutationKeepsNewPlayback protects even a replacement with the same track ID.
func TestCatalogMutationKeepsNewPlayback(t *testing.T) {
	for _, replacement := range []int32{7, 8} {
		memkv.Store.Set("now-playing:19001", int32(7))
		memkv.Store.Set("now-playing:19002", int32(9))
		plan := db.CatalogPlan{Kind: "track", Action: "delete", AffectedTrackIDs: []int32{7}}
		require.NoError(t, ApplyCatalogMutation(context.Background(), cacheMutationStore{replacement}, &models.User{Role: models.UserRoleAdmin}, plan))
		first, ok := memkv.Store.Get("now-playing:19001")
		require.True(t, ok)
		require.Equal(t, replacement, first)
		second, ok := memkv.Store.Get("now-playing:19002")
		require.True(t, ok)
		require.Equal(t, int32(9), second)
	}
	memkv.Store.Delete("now-playing:19001")
	memkv.Store.Delete("now-playing:19002")
}
