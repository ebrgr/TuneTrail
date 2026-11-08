package admin

import (
	"context"
	"github.com/ebrgr/TuneTrail/internal/db"
	"github.com/ebrgr/TuneTrail/internal/models"
	"github.com/ebrgr/TuneTrail/internal/relay"
)

// Relay returns sensitive values only to an authenticated administrator.
func Relay(ctx context.Context, store db.ProfileStore, user *models.User, id int32, slug string) (string, string, error) {
	if _, err := Profile(ctx, store, user, id, slug); err != nil {
		return "", "", err
	}
	return store.GetProfileRelay(ctx, id)
}

// SaveRelay validates the same destination used by both test and forwarding.
func SaveRelay(ctx context.Context, store db.ProfileStore, user *models.User, id int32, slug, raw, token string) error {
	profileMu.Lock()
	defer profileMu.Unlock()
	if _, err := Profile(ctx, store, user, id, slug); err != nil {
		return err
	}
	raw, token, err := relay.Normalize(raw, token)
	if err != nil {
		return err
	}
	return store.SetProfileRelay(ctx, id, raw, token)
}

// RemoveRelay prevents future submissions from loading the removed configuration.
func RemoveRelay(ctx context.Context, store db.ProfileStore, user *models.User, id int32, slug string) error {
	profileMu.Lock()
	defer profileMu.Unlock()
	if _, err := Profile(ctx, store, user, id, slug); err != nil {
		return err
	}
	return store.SetProfileRelay(ctx, id, "", "")
}

// TestRelay checks a draft without saving it or emitting a listening event.
func TestRelay(ctx context.Context, store db.ProfileStore, user *models.User, id int32, slug, raw, token string, client relay.Client) (string, error) {
	if _, err := Profile(ctx, store, user, id, slug); err != nil {
		return "", err
	}
	return client.Test(ctx, raw, token)
}
