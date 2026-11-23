package engine

import (
	"bytes"
	"context"
	"net/http"
	"testing"

	"github.com/ebrgr/TuneTrail/internal/admin"
	"github.com/ebrgr/TuneTrail/internal/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// TestSessionTransitionMatrix164 checks that the same root cookie and service
// identity lose access across all boundaries when the account changes.
func TestSessionTransitionMatrix164(t *testing.T) {
	a := taskApp139(t)
	_, owner := owner160(t, a)
	ctx := context.Background()
	rootSession, err := uuid.Parse(a.cookie)
	require.NoError(t, err)
	root, err := admin.Authenticate(ctx, a.store, rootSession)
	require.NoError(t, err)

	protected := true
	_, err = admin.UpdateProfile(ctx, a.store, root.User, owner.ID, owner.Slug, owner.DisplayName, owner.Slug, &protected)
	require.NoError(t, err)
	require.NoError(t, a.store.Exec("UPDATE users SET role='admin' WHERE id=?", owner.UserID))

	path := "/apis/web/v1/profiles/" + owner.Slug
	for _, state := range []struct {
		name         string
		read, manage int
		service      error
	}{
		{"root", http.StatusOK, http.StatusOK, nil},
		{"demoted", http.StatusNotFound, http.StatusForbidden, admin.ErrForbidden},
		{"expired", http.StatusNotFound, http.StatusUnauthorized, admin.ErrUnauthorized},
	} {
		t.Run(state.name, func(t *testing.T) {
			switch state.name {
			case "demoted":
				require.NoError(t, admin.ChangeProfileAccountRole(ctx, a.store, rootSession, 1, models.UserRoleUser))
			case "expired":
				require.NoError(t, a.store.Exec("UPDATE sessions SET expires_at=1 WHERE id=?", rootSession.String()))
			}

			require.Equal(t, state.read, taskRequest139(a, "GET", "/"+owner.Slug, a.cookie, nil, nil).Code)
			require.Equal(t, state.read, json160(a, "GET", path+"/stats", "", a.cookie, "").Code)
			require.Equal(t, state.manage, json160(a, "GET", path+"/management", "", a.cookie, "").Code)

			var archive bytes.Buffer
			err := admin.ExportHistory(ctx, a.store, root.User, owner.ID, &archive)
			if state.service == nil {
				require.NoError(t, err)
				require.NotEmpty(t, archive.Bytes())
			} else {
				require.ErrorIs(t, err, state.service)
				require.Empty(t, archive.Bytes())
			}
		})
	}
}

// TestSharedLoginLimit164 verifies that HTML and JSON consume the same
// existing allowance across different account names.
func TestSharedLoginLimit164(t *testing.T) {
	a := newProfileTestApp(t)
	for range 5 {
		html := signIn128(t, a, "missing-html-user", "wrong-password", "")
		require.Equal(t, http.StatusUnauthorized, html.Code)
		api := json160(a, "POST", "/apis/web/v1/login", `{"username":"missing-api-user","password":"wrong-password"}`, "", "")
		require.Equal(t, http.StatusUnauthorized, api.Code)
	}
	blocked := signIn128(t, a, "missing-eleventh-user", "wrong-password", "")
	require.Equal(t, http.StatusTooManyRequests, blocked.Code)
	require.Contains(t, blocked.Header().Get("Cache-Control"), "no-store")
}
