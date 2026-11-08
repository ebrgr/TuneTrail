package engine

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/ebrgr/TuneTrail/engine/handlers"
	"github.com/ebrgr/TuneTrail/engine/middleware"
	"github.com/ebrgr/TuneTrail/internal/db"
	"github.com/ebrgr/TuneTrail/internal/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// TestLoginFixedLifetimes validates both expiration contracts against the stored session.
func TestLoginFixedLifetimes(t *testing.T) {
	a := newProfileTestApp(t)
	for _, tc := range []struct {
		body     string
		lifetime time.Duration
	}{
		{`{"username":"administrator","password":"test-password","remember_me":false}`, 24 * time.Hour},
		{`{"username":"administrator","password":"test-password","remember_me":true}`, 30 * 24 * time.Hour},
	} {
		before := time.Now()
		w := a.request("POST", "/apis/web/v1/login", tc.body, "", false)
		after := time.Now()
		require.Equal(t, http.StatusNoContent, w.Code)
		require.Len(t, w.Result().Cookies(), 1)
		cookie := w.Result().Cookies()[0]
		require.WithinRange(t, cookie.Expires, before.Add(tc.lifetime-time.Second), after.Add(tc.lifetime))
		var expiry int64
		require.NoError(t, a.store.QueryRow(`SELECT expires_at FROM sessions WHERE id=?`, cookie.Value).Scan(&expiry))
		require.Equal(t, cookie.Expires.Unix(), expiry)
	}
}

type failedSessionStore struct{ db.UserStore }

// GetUserBySession injects an infrastructure error without a real database failure.
func (s failedSessionStore) GetUserBySession(context.Context, uuid.UUID) (*models.User, error) {
	return nil, errors.New("private session store failure")
}

// DeleteSession injects a revocation failure before any cookie is cleared.
func (s failedSessionStore) DeleteSession(context.Context, uuid.UUID) error {
	return errors.New("private session store failure")
}

// GetUserByUsername simulates an unavailable login store.
func (s failedSessionStore) GetUserByUsername(context.Context, string) (*models.User, error) {
	return nil, errors.New("private session store failure")
}

// TestSessionStoreFailureIsNotReportedAsInvalidOrSuccessful ensures failures stay private.
func TestSessionStoreFailureIsNotReportedAsInvalidOrSuccessful(t *testing.T) {
	store := failedSessionStore{}
	for _, handler := range []http.Handler{
		middleware.RequireAdmin(store)(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { t.Fatal("unexpected authorization") })),
		handlers.LogoutHandler(store),
	} {
		r := httptest.NewRequest("POST", "http://koito.test/apis/web/v1/user", nil)
		r.AddCookie(&http.Cookie{Name: "koito_session", Value: uuid.NewString()})
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		require.Equal(t, 500, w.Code)
		require.Equal(t, "no-store", w.Header().Get("Cache-Control"))
		require.NotContains(t, w.Body.String(), "private session store failure")
		require.Empty(t, w.Header().Values("Set-Cookie"))
	}
}

// TestLoginStoreFailureIsNotReportedAsBadCredentials keeps infrastructure errors distinct.
func TestLoginStoreFailureIsNotReportedAsBadCredentials(t *testing.T) {
	r := httptest.NewRequest("POST", "http://koito.test/apis/web/v1/login", strings.NewReader(`{"username":"fixture","password":"fictional-password"}`))
	w := httptest.NewRecorder()
	handlers.LoginHandler(failedSessionStore{}).ServeHTTP(w, r)
	require.Equal(t, http.StatusInternalServerError, w.Code)
	require.Equal(t, "no-store", w.Header().Get("Cache-Control"))
	require.NotContains(t, w.Body.String(), "private session store failure")
	require.NotContains(t, w.Body.String(), "fictional-password")
	require.Empty(t, w.Header().Values("Set-Cookie"))
}

// TestTLSLoginCookieUsesSecureFlag exercises real TLS rather than a proxy header alone.
func TestTLSLoginCookieUsesSecureFlag(t *testing.T) {
	a := newProfileTestApp(t)
	server := httptest.NewTLSServer(a.router)
	defer server.Close()
	resp, err := server.Client().Post(server.URL+"/apis/web/v1/login", "application/json", strings.NewReader(`{"username":"administrator","password":"test-password"}`))
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusNoContent, resp.StatusCode)
	require.Len(t, resp.Cookies(), 1)
	require.True(t, resp.Cookies()[0].Secure)
	require.True(t, resp.Cookies()[0].HttpOnly)
	require.Equal(t, http.SameSiteStrictMode, resp.Cookies()[0].SameSite)
}

// TestAmbiguousLogoutDoesNotRevokeEitherSession protects against selecting a duplicate cookie.
func TestAmbiguousLogoutDoesNotRevokeEitherSession(t *testing.T) {
	for _, reverse := range []bool{false, true} {
		a := newProfileTestApp(t)
		pair := []string{"koito_session=" + a.cookie, "koito_session=invalid"}
		if reverse {
			pair[0], pair[1] = pair[1], pair[0]
		}
		r := httptest.NewRequest("POST", "http://koito.test/apis/web/v1/logout", nil)
		r.Header.Set("Cookie", strings.Join(pair, "; "))
		w := httptest.NewRecorder()
		a.router.ServeHTTP(w, r)
		require.Equal(t, http.StatusBadRequest, w.Code)
		require.Equal(t, http.StatusOK, a.request("GET", "/apis/web/v1/user", "", "", true).Code)
	}
}

// TestCrossOriginRejectionIsNotCached checks the outer JSON middleware's error response.
func TestCrossOriginRejectionIsNotCached(t *testing.T) {
	a := newProfileTestApp(t)
	r := httptest.NewRequest("POST", "http://koito.test/apis/web/v1/login", strings.NewReader("{}"))
	r.Header.Set("Origin", "https://evil.invalid")
	w := httptest.NewRecorder()
	a.router.ServeHTTP(w, r)
	require.Equal(t, http.StatusForbidden, w.Code)
	require.Equal(t, "no-store", w.Header().Get("Cache-Control"))
}
