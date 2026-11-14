package adminui

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/ebrgr/TuneTrail/internal/admin"
	"github.com/ebrgr/TuneTrail/internal/db"
	"github.com/ebrgr/TuneTrail/internal/db/sqlite"
	"github.com/ebrgr/TuneTrail/internal/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestExpiredCatalogReviewCannotExecute(t *testing.T) {
	ctx := context.Background()
	s, err := sqlite.NewInMemory()
	require.NoError(t, err)
	defer s.Close(ctx)
	user, err := s.SaveUser(ctx, db.SaveUserOpts{Username: "admin", Password: "test-password", Role: models.UserRoleAdmin})
	require.NoError(t, err)
	session, err := s.SaveSession(ctx, user.ID, time.Now().Add(time.Hour), false)
	require.NoError(t, err)
	h, err := New(s, Options{TempDir: t.TempDir()})
	require.NoError(t, err)
	csrf, err := h.csrf.SessionToken(admin.Identity{User: user, SessionID: session.ID})
	require.NoError(t, err)
	token := uuid.New()
	h.catalogPlans[token] = pendingCatalogPlan{Session: session.ID, ProfileID: 1, ProfileSlug: "default", RouteID: 42, Kind: "track", Plan: db.CatalogPlan{Action: "delete", Kind: "track", From: 42}, Expires: time.Now().Add(-time.Second)}
	v := url.Values{"csrf_token": {csrf}, "profile_id": {"1"}, "profile_slug": {"default"}, "plan": {token.String()}, "confirm": {"1"}}
	r := httptest.NewRequest("POST", "/default/manage/track/42/confirm", strings.NewReader(v.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.AddCookie(&http.Cookie{Name: "tunetrail_session", Value: session.ID.String()})
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	require.Equal(t, http.StatusConflict, w.Code)
	require.Empty(t, h.catalogPlans)
}
