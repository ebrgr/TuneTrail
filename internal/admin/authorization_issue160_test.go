package admin_test

import (
	"context"
	"github.com/ebrgr/TuneTrail/internal/admin"
	"github.com/ebrgr/TuneTrail/internal/db"
	"github.com/ebrgr/TuneTrail/internal/models"
	"github.com/stretchr/testify/require"
	"io"
	"testing"
	"time"
)

// TestManagementPolicy160 covers every capability and unknown actions for each actor.
func TestManagementPolicy160(t *testing.T) {
	own := &models.ListeningProfile{ID: 10, UserID: 2}
	for _, actor := range []struct {
		name        string
		user        *models.User
		own, global bool
	}{
		{"visitor", nil, false, false},
		{"owner", &models.User{ID: 2, Role: models.UserRoleUser, Status: "active"}, true, false},
		{"other", &models.User{ID: 3, Role: models.UserRoleUser, Status: "active"}, false, false},
		{"root", &models.User{ID: 1, Role: models.UserRoleAdmin, Status: "active"}, true, true},
		{"pending", &models.User{ID: 2, Role: models.UserRoleAdmin, Status: "pending"}, false, false},
		{"unknown-role", &models.User{ID: 2, Role: "superuser", Status: "active"}, false, false},
	} {
		t.Run(actor.name, func(t *testing.T) {
			for _, action := range []admin.Action{admin.ManageProfile, admin.ManageAccount, admin.ManageAccounts, admin.ManageCatalog, admin.ManageServer, "unlisted"} {
				allowed := actor.global && action != "unlisted"
				if action == admin.ManageProfile {
					allowed = actor.own
				}
				if action == admin.ManageAccount {
					allowed = actor.user != nil && actor.user.Status == "active" && (actor.user.Role == models.UserRoleUser || actor.user.Role == models.UserRoleAdmin)
				}
				err := admin.AuthorizeAction(actor.user, action, own)
				if allowed {
					require.NoError(t, err, string(action))
				} else {
					require.Error(t, err, string(action))
				}
			}
			require.Error(t, admin.AuthorizeAction(actor.user, admin.ManageProfile, nil))
		})
	}
}

// TestPersonalServices160 proves direct calls cannot widen the server-selected profile.
func TestPersonalServices160(t *testing.T) {
	store, root := accountFixture(t)
	ctx := context.Background()
	owner, err := store.SaveUser(ctx, db.SaveUserOpts{Username: "owner", Password: "test-password"})
	require.NoError(t, err)
	session, err := admin.Login(ctx, store, owner.Username, "test-password", false)
	require.NoError(t, err)
	identity, err := admin.Authenticate(ctx, store, session.ID)
	require.NoError(t, err)
	owner = identity.User
	profiles, err := store.ListListeningProfiles(ctx)
	require.NoError(t, err)
	var own, other models.ListeningProfile
	for _, p := range profiles {
		if p.UserID == owner.ID {
			own = p
		} else {
			other = p
		}
	}
	key, err := admin.CreateKey(ctx, store, owner, own.ID, own.Slug, "own")
	require.NoError(t, err)
	require.NoError(t, admin.RenameKey(ctx, store, owner, own.ID, own.Slug, key.ID, "renamed"))
	require.NoError(t, admin.RevokeKey(ctx, store, owner, own.ID, own.Slug, key.ID))
	_, err = admin.UpdateProfile(ctx, store, owner, own.ID, own.Slug, "Owner", own.Slug)
	require.NoError(t, err)
	_, err = admin.SetAvatar(ctx, store, owner, own.ID, own.Slug, "music", nil)
	require.NoError(t, err)
	require.NoError(t, admin.RemoveRelay(ctx, store, owner, own.ID, own.Slug))
	for _, user := range []*models.User{nil, owner} {
		_, err = admin.Keys(ctx, store, user, other.ID, other.Slug)
		require.Error(t, err)
		_, err = admin.CreateKey(ctx, store, user, other.ID, other.Slug, "bad")
		require.Error(t, err)
		_, err = admin.UpdateProfile(ctx, store, user, other.ID, other.Slug, "stolen", other.Slug)
		require.Error(t, err)
		_, err = admin.SetAvatar(ctx, store, user, other.ID, other.Slug, "music", nil)
		require.Error(t, err)
		require.Error(t, admin.RemoveRelay(ctx, store, user, other.ID, other.Slug))
		require.Error(t, admin.ExportHistory(ctx, store, user, other.ID, io.Discard))
		_, err = admin.ImportHistory(ctx, store, user, nil, other.ID, "koito", "missing")
		require.Error(t, err)
		_, err = admin.ManualListen(ctx, store, user, other.ID, 1, time.Now().Add(-time.Hour), "")
		require.Error(t, err)
		require.Error(t, admin.RemovePersonalListen(ctx, store, user, other.ID, 1, time.Now()))
		_, err = admin.PreviewPersonalDelete(ctx, store, user, other.ID, "track", 1)
		require.Error(t, err)
		_, err = admin.ApplyPersonalDelete(ctx, store, user, db.ProfileDeletePlan{ProfileID: other.ID})
		require.Error(t, err)
		require.Error(t, admin.DeleteProfile(ctx, store, user, own.ID, own.Slug, "Owner", 0))
		_, err = admin.CreateProfile(ctx, store, user, "bad", "bad", "")
		require.Error(t, err)
		require.Error(t, admin.MediaExists(ctx, store, user, "track", 1))
		require.Error(t, admin.EditAlias(ctx, store, user, "track", 1, "add", "bad"))
		require.Error(t, admin.MergeTracks(ctx, store, user, 1, 2))
		_, err = admin.PreviewCatalogMutation(ctx, store, user, db.CatalogPlan{})
		require.Error(t, err)
	}
	_, err = admin.Keys(ctx, store, root, own.ID, own.Slug)
	require.NoError(t, err)
	require.NoError(t, admin.ExportHistory(ctx, store, owner, own.ID, io.Discard))
}

// TestServicesRequireHumanSession160 rejects forged roles, ingestion attribution,
// expired sessions and role snapshots retained after a server-side demotion.
func TestServicesRequireHumanSession160(t *testing.T) {
	store, root := accountFixture(t)
	ctx := context.Background()
	_, err := store.SaveApiKey(ctx, db.SaveApiKeyOpts{UserID: root.ID, Key: "machine-key", Label: "machine"})
	require.NoError(t, err)
	attributed, err := store.GetUserByApiKey(ctx, "machine-key")
	require.NoError(t, err)
	require.ErrorIs(t, admin.PurgeData(ctx, store, attributed), admin.ErrUnauthorized)
	_, err = admin.Keys(ctx, store, attributed, 1, "default")
	require.ErrorIs(t, err, admin.ErrUnauthorized)
	_, err = store.SaveUser(ctx, db.SaveUserOpts{Username: "other-root", Password: "test-password", Role: models.UserRoleAdmin})
	require.NoError(t, err)
	require.NoError(t, store.Exec(`UPDATE users SET role='user' WHERE id=?`, root.ID))
	require.ErrorIs(t, admin.DeleteCatalogItem(ctx, store, root, "track", 1), admin.ErrForbidden)
	require.ErrorIs(t, admin.PurgeData(ctx, store, root), admin.ErrForbidden)
	// An owner retained as root in memory still manages only the canonical own profile.
	_, err = admin.Keys(ctx, store, root, 1, "default")
	require.NoError(t, err)
	require.NoError(t, store.DeleteSession(ctx, root.SessionID))
	_, err = admin.Keys(ctx, store, root, 1, "default")
	require.ErrorIs(t, err, admin.ErrUnauthorized)
}
