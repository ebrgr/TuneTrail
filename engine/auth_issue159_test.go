package engine

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ebrgr/TuneTrail/internal/admin"
	"github.com/ebrgr/TuneTrail/internal/db"
	"github.com/ebrgr/TuneTrail/internal/models"
	"github.com/stretchr/testify/require"
)

// TestUserAuthenticationKeepsAdminRoutesClosed159 exercises the production HTML and JSON boundaries.
func TestUserAuthenticationKeepsAdminRoutesClosed159(t *testing.T) {
	a := newProfileTestApp(t)
	_, err := a.store.SaveUser(context.Background(), db.SaveUserOpts{Username: "listener", Password: "test-password", Role: models.UserRoleUser})
	require.NoError(t, err)
	login := a.request("POST", "/apis/web/v1/login", `{"username":"listener","password":"test-password"}`, "", false)
	require.Equal(t, 204, login.Code)
	cookie := login.Result().Cookies()[0]
	for _, path := range []string{"/admin", "/admin/settings", "/admin/profile/1/delete-review", "/admin/profile/1/export", "/apis/web/v1/user", "/apis/web/v1/profiles/default/management"} {
		r := httptest.NewRequest("GET", "http://koito.test"+path, nil)
		r.AddCookie(cookie)
		w := httptest.NewRecorder()
		a.router.ServeHTTP(w, r)
		require.Equal(t, 403, w.Code, path)
	}
	r := httptest.NewRequest("POST", "http://koito.test/admin/profile/1/save", strings.NewReader(""))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.AddCookie(cookie)
	w := httptest.NewRecorder()
	a.router.ServeHTTP(w, r)
	require.Equal(t, 403, w.Code)
	// The JSON logout path is already available to either account role.
	r = httptest.NewRequest("POST", "http://koito.test/apis/web/v1/logout", nil)
	r.AddCookie(cookie)
	w = httptest.NewRecorder()
	a.router.ServeHTTP(w, r)
	require.Equal(t, 204, w.Code)
	_, err = admin.Login(context.Background(), a.store, "listener", "wrong-password", false)
	require.ErrorIs(t, err, admin.ErrUnauthorized)
}

// TestSharedLoginLimiter159 proves HTML and JSON still share the original ten-attempt budget.
func TestSharedLoginLimiter159(t *testing.T) {
	a := newProfileTestApp(t)
	for i := 0; i < 10; i++ {
		path := "/apis/web/v1/login"
		if i%2 == 1 {
			path = "/admin/login"
		}
		// Invalid bodies still count across endpoints; the budget is not per account or IP.
		r := httptest.NewRequest("POST", "http://koito.test"+path, strings.NewReader(`{}`))
		r.RemoteAddr = "192.0.2.1:1234"
		w := httptest.NewRecorder()
		a.router.ServeHTTP(w, r)
		require.NotEqual(t, http.StatusTooManyRequests, w.Code)
	}
	for _, path := range []string{"/apis/web/v1/login", "/admin/login"} {
		r := httptest.NewRequest("POST", "http://koito.test"+path, strings.NewReader(`{"username":"another-account"}`))
		r.RemoteAddr = "198.51.100.2:9876"
		w := httptest.NewRecorder()
		a.router.ServeHTTP(w, r)
		require.Equal(t, http.StatusTooManyRequests, w.Code, path)
	}
}
