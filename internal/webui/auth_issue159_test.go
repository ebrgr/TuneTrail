package webui

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ebrgr/TuneTrail/internal/admin"
	"github.com/ebrgr/TuneTrail/internal/db"
	"github.com/ebrgr/TuneTrail/internal/models"
	"github.com/stretchr/testify/require"
)

// TestAdminSessionRejectsOrdinaryAccount159 keeps public management controls exclusive to admins.
func TestAdminSessionRejectsOrdinaryAccount159(t *testing.T) {
	store := pageStore(t)
	h, err := NewHandler(store, Options{Sessions: store})
	require.NoError(t, err)
	for _, role := range []models.UserRole{models.UserRoleUser, models.UserRoleAdmin} {
		_, err = store.SaveUser(context.Background(), db.SaveUserOpts{Username: string(role), Password: "test-password", Role: role})
		require.NoError(t, err)
		session, err := admin.Login(context.Background(), store, string(role), "test-password", false)
		require.NoError(t, err)
		r := httptest.NewRequest("GET", "http://koito.test/alpha/listens", nil)
		r.AddCookie(&http.Cookie{Name: "tunetrail_session", Value: session.ID.String()})
		require.Equal(t, role == models.UserRoleAdmin, h.adminSession(r))
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		require.Equal(t, 200, w.Code)
		if role == models.UserRoleUser {
			require.NotContains(t, w.Body.String(), "/manage/")
		} else {
			require.Contains(t, w.Body.String(), "/manage/")
		}
	}
}
