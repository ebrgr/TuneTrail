package admin

import (
	"context"
	"github.com/ebrgr/TuneTrail/internal/db"
	"github.com/ebrgr/TuneTrail/internal/models"
)

// DeleteCatalogItem preserves the legacy JSON deletion contract behind the shared policy.
func DeleteCatalogItem(ctx context.Context, store interface {
	DeleteArtist(context.Context, int32) error
	DeleteAlbum(context.Context, int32) error
	DeleteTrack(context.Context, int32) error
}, user *models.User, kind string, id int32) error {
	if err := AuthorizeService(ctx, store, user, ManageCatalog, nil); err != nil {
		return err
	}
	if id <= 0 {
		return db.ErrInvalidInput
	}
	ctx = db.WithoutProfile(ctx)
	switch kind {
	case "artist":
		return store.DeleteArtist(ctx, id)
	case "album":
		return store.DeleteAlbum(ctx, id)
	case "track":
		return store.DeleteTrack(ctx, id)
	}
	return db.ErrInvalidInput
}

// PurgeData requires root for the global server operation, including direct service calls.
func PurgeData(ctx context.Context, store interface{ PurgeAllData(context.Context) error }, user *models.User) error {
	if err := AuthorizeService(ctx, store, user, ManageServer, nil); err != nil {
		return err
	}
	return store.PurgeAllData(ctx)
}
