package middleware

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ebrgr/TuneTrail/internal/db"
	"github.com/ebrgr/TuneTrail/internal/models"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"
)

type failingProfileStore struct {
	db.ProfileStore
	err error
}

// GetListeningProfile injects missing or infrastructure errors without touching a database.
func (s failingProfileStore) GetListeningProfile(context.Context, string) (*models.ListeningProfile, error) {
	return nil, s.err
}

// GetProfileByAPIKey injects missing or infrastructure errors without exposing credentials.
func (s failingProfileStore) GetProfileByAPIKey(context.Context, string) (*models.ListeningProfile, error) {
	return nil, s.err
}

// TestProfileAndIngestionLookupFailures distinguishes absence from an unavailable store.
func TestProfileAndIngestionLookupFailures(t *testing.T) {
	for _, tc := range []struct {
		name                     string
		err                      error
		profileStatus, keyStatus int
	}{
		{"missing", db.ErrNotFound, 404, 401},
		{"wrapped missing", errors.Join(errors.New("wrapped"), db.ErrNotFound), 404, 401},
		{"store failure", errors.New("private database detail"), 500, 500},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, kind := range []string{"profile", "key"} {
				called := false
				next := http.HandlerFunc(func(http.ResponseWriter, *http.Request) { called = true })
				r := httptest.NewRequest("GET", "/", nil)
				store := failingProfileStore{err: tc.err}
				var h http.Handler
				expected := tc.profileStatus
				if kind == "profile" {
					rc := chi.NewRouteContext()
					rc.URLParams.Add("slug", "alpha")
					r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rc))
					h = ProfileScope(store)(next)
				} else {
					r.Header.Set("Authorization", "Token fictional-secret")
					h = IngestionKey(store)(next)
					expected = tc.keyStatus
				}
				w := httptest.NewRecorder()
				h.ServeHTTP(w, r)
				require.Equal(t, expected, w.Code, kind)
				require.False(t, called)
				require.NotContains(t, w.Body.String(), "private database detail")
				require.NotContains(t, w.Body.String(), "fictional-secret")
				require.False(t, strings.Contains(w.Body.String(), "SELECT"))
			}
		})
	}
}
