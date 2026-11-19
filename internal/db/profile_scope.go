package db

import "context"

type profileScopeKey struct{}

// WithProfile scopes listening reads to a stable internal ID, never a mutable slug.
func WithProfile(ctx context.Context, id int32) context.Context {
	return context.WithValue(context.WithValue(ctx, profileScopeKey{}, id), catalogProfileKey{}, id)
}
func ProfileID(ctx context.Context) int32 {
	id, _ := ctx.Value(profileScopeKey{}).(int32)
	return id
}

// WithoutProfile is for shared catalog maintenance, not public reads.
func WithoutProfile(ctx context.Context) context.Context {
	// Catalog resolution retains its destination even when listening reads become global.
	return context.WithValue(context.WithValue(ctx, catalogProfileKey{}, CatalogProfile(ctx)), profileScopeKey{}, int32(0))
}
