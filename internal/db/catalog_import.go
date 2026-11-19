package db

import (
	"context"
	"fmt"
)

type importedCatalogKey struct{}
type catalogProfileKey struct{}
type editorialCatalogKey struct{}

const ImportedAliasPrefix = "ProfileImport:"

// WithImportedCatalog preserves readable imported metadata while scoping alias resolution.
func WithImportedCatalog(ctx context.Context, profileID int32) context.Context {
	return context.WithValue(context.WithValue(ctx, importedCatalogKey{}, profileID), catalogProfileKey{}, profileID)
}

// ImportedCatalogProfile identifies metadata created by an owner import, not ingestion in general.
func ImportedCatalogProfile(ctx context.Context) int32 {
	id, _ := ctx.Value(importedCatalogKey{}).(int32)
	return id
}

// CatalogProfile survives removal of listening-read scope during shared-catalog association.
func CatalogProfile(ctx context.Context) int32 {
	id, ok := ctx.Value(catalogProfileKey{}).(int32)
	if ok {
		return id
	}
	return ProfileID(ctx)
}

// ImportedAliasSource uses existing provenance storage without changing display names or schema.
func ImportedAliasSource(ctx context.Context, fallback string) string {
	if id := ImportedCatalogProfile(ctx); id > 0 {
		return fmt.Sprintf("%s%d", ImportedAliasPrefix, id)
	}
	return fallback
}

// CatalogAliasSource identifies the only owner-import aliases eligible for this destination.
func CatalogAliasSource(ctx context.Context) string {
	return fmt.Sprintf("%s%d", ImportedAliasPrefix, CatalogProfile(ctx))
}

// WithCatalogEditorial is used only after root authorization of an explicit catalog edit.
func WithCatalogEditorial(ctx context.Context) context.Context {
	return context.WithValue(ctx, editorialCatalogKey{}, true)
}

// CatalogEditorial permits root to validate an imported identity's MBID explicitly.
func CatalogEditorial(ctx context.Context) bool {
	value, _ := ctx.Value(editorialCatalogKey{}).(bool)
	return value
}
