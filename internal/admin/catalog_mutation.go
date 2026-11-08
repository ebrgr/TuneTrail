package admin

import (
	"context"
	"fmt"

	"github.com/ebrgr/TuneTrail/internal/db"
	"github.com/ebrgr/TuneTrail/internal/memkv"
	"github.com/ebrgr/TuneTrail/internal/models"
)

// CatalogMutationStore supplies a review and an atomic revalidated mutation.
type CatalogMutationStore interface {
	PreviewCatalogMutation(context.Context, db.CatalogPlan) (db.CatalogPlan, error)
	ApplyCatalogMutation(context.Context, db.CatalogPlan) error
	ListListeningProfiles(context.Context) ([]models.ListeningProfile, error)
}

// PreviewCatalogMutation requires an administrator and computes global impact.
func PreviewCatalogMutation(ctx context.Context, s CatalogMutationStore, user *models.User, plan db.CatalogPlan) (db.CatalogPlan, error) {
	if err := Authorize(user); err != nil {
		return plan, err
	}
	return s.PreviewCatalogMutation(ctx, plan)
}

// ApplyCatalogMutation changes the catalog only after transactional revalidation.
func ApplyCatalogMutation(ctx context.Context, s CatalogMutationStore, user *models.User, plan db.CatalogPlan) error {
	if err := Authorize(user); err != nil {
		return err
	}
	profiles, err := s.ListListeningProfiles(ctx)
	if err != nil {
		return err
	}
	affected := map[int32]bool{}
	for _, id := range plan.AffectedTrackIDs {
		affected[id] = true
	}
	if plan.Kind == "track" && plan.Action == "merge" {
		affected[plan.To] = true
	}
	// Snapshot cache entries before commit; a later update must not be deleted.
	versions := map[string]uint64{}
	for _, p := range profiles {
		key := fmt.Sprintf("now-playing:%d", p.ID)
		if v, version, ok := memkv.Store.GetVersioned(key); ok {
			if id, ok := v.(int32); ok && affected[id] {
				versions[key] = version
			}
		}
	}
	if err = s.ApplyCatalogMutation(ctx, plan); err != nil {
		return err
	}
	for key, version := range versions {
		memkv.Store.DeleteVersion(key, version)
	}
	// Orphan image cleanup is deferred to the global collector; committed changes are never reported as rolled back.
	return nil
}
