package admin

import (
	"context"
	"github.com/ebrgr/TuneTrail/internal/db"
	"github.com/ebrgr/TuneTrail/internal/export"
	"github.com/ebrgr/TuneTrail/internal/importer"
	"github.com/ebrgr/TuneTrail/internal/mbz"
	"github.com/ebrgr/TuneTrail/internal/models"
	"io"
)

// ExportHistory authorizes before the serializer can write even the first byte.
func ExportHistory(ctx context.Context, store interface {
	db.ProfileStore
	db.ExportStore
}, user *models.User, id int32, out io.Writer) error {
	profile, err := AuthorizeProfile(ctx, store, user, id)
	if err != nil {
		return err
	}
	return export.ExportData(ctx, profile, store, out)
}

// ImportHistory authorizes a server-selected destination before staging or parsing.
// Creating new musical identities with artwork/aliases is permitted by the user;
// it does not grant access to the shared catalog's editorial operations.
func ImportHistory(ctx context.Context, store interface {
	db.ProfileStore
	importer.UploadStore
}, user *models.User, mb mbz.MusicBrainzCaller, id int32, format, filename string) (importer.Result, error) {
	if _, err := AuthorizeProfile(ctx, store, user, id); err != nil {
		return importer.Result{}, err
	}
	return importer.Upload(ctx, store, mb, id, format, filename)
}

// PreviewPersonalDelete limits the review to the authorized profile.
func PreviewPersonalDelete(ctx context.Context, store interface {
	db.ProfileStore
	db.ProfileDeleteStore
}, user *models.User, id int32, kind string, item int32) (db.ProfileDeletePlan, error) {
	if _, err := AuthorizeProfile(ctx, store, user, id); err != nil {
		return db.ProfileDeletePlan{}, err
	}
	return store.PreviewProfileDelete(ctx, id, kind, item)
}

// ApplyPersonalDelete reauthorizes the reviewed destination before transactional revalidation.
func ApplyPersonalDelete(ctx context.Context, store interface {
	db.ProfileStore
	db.ProfileDeleteStore
}, user *models.User, plan db.ProfileDeletePlan) (int64, error) {
	if _, err := AuthorizeProfile(ctx, store, user, plan.ProfileID); err != nil {
		return 0, err
	}
	return store.ApplyProfileDelete(ctx, plan)
}
