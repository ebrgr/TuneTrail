package admin

import (
	"context"
	"github.com/ebrgr/TuneTrail/internal/db"
	"github.com/ebrgr/TuneTrail/internal/models"
)

// ValidateMerge rejects malformed identities before the transactional existence checks.
func ValidateMerge(user *models.User, from, to int32) error {
	if err := Authorize(user); err != nil {
		return err
	}
	if from <= 0 || to <= 0 || from == to {
		return db.ErrInvalidInput
	}
	return nil
}

// MergeArtists applies the same actor and identity contract for JSON and future HTML.
func MergeArtists(ctx context.Context, store interface {
	MergeArtists(context.Context, int32, int32, bool) error
}, user *models.User, from, to int32, replace bool) error {
	if err := ValidateMerge(user, from, to); err != nil {
		return err
	}
	return store.MergeArtists(ctx, from, to, replace)
}

// MergeAlbums delegates atomic catalog changes without applying a personal read scope.
func MergeAlbums(ctx context.Context, store interface {
	MergeAlbums(context.Context, int32, int32, bool) error
}, user *models.User, from, to int32, replace bool) error {
	if err := ValidateMerge(user, from, to); err != nil {
		return err
	}
	return store.MergeAlbums(ctx, from, to, replace)
}

// MergeTracks preserves the store's per-profile duplicate handling.
func MergeTracks(ctx context.Context, store interface {
	MergeTracks(context.Context, int32, int32) error
}, user *models.User, from, to int32) error {
	if err := ValidateMerge(user, from, to); err != nil {
		return err
	}
	return store.MergeTracks(ctx, from, to)
}
