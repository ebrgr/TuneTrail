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
	raw, token, err := relayDraft(ctx, store, id, raw, token)
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
	raw, token, err := relayDraft(ctx, store, id, raw, token)
	if err != nil {
		return "", err
	}
	// Tests can inject a bounded fake transport. Live configuration always uses the policy dialer.
	if client.Transport != nil {
		return client.Test(ctx, raw, token)
	}
	var permissions relay.Permissions
	if scoped, ok := store.(interface {
		RelayPermissions(context.Context, int32) (relay.Permissions, error)
	}); ok {
		permissions, err = scoped.RelayPermissions(ctx, id)
		if err != nil {
			return "", err
		}
	}
	return relay.NewDeliverer(permissions, nil).Test(ctx, raw, token)
}

// relayDraft reuses a saved credential only for the same endpoint.
func relayDraft(ctx context.Context, store db.ProfileStore, id int32, raw, token string) (string, string, error) {
	if token != "" {
		return relay.Normalize(raw, token)
	}
	base, _, err := relay.Normalize(raw, "placeholder")
	if err != nil {
		return "", "", err
	}
	saved, secret, err := store.GetProfileRelay(ctx, id)
	if err != nil {
		return "", "", err
	}
	if base != saved || secret == "" {
		return "", "", db.ErrInvalidInput
	}
	return relay.Normalize(base, secret)
}
