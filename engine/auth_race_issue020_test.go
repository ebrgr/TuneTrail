package engine

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ebrgr/TuneTrail/engine/middleware"
	"github.com/ebrgr/TuneTrail/internal/db"
	"github.com/stretchr/testify/require"
)

// TestIngestionAuthorizationBoundaryAfterRevocation defines the fate of an already authorized request.
func TestIngestionAuthorizationBoundaryAfterRevocation(t *testing.T) {
	a := newProfileTestApp(t)
	for _, q := range []string{`INSERT INTO artists(id) VALUES(1)`, `INSERT INTO releases(id) VALUES(1)`, `INSERT INTO tracks(id,release_id) VALUES(1,1)`} {
		require.NoError(t, a.store.Exec(q))
	}
	keys, err := a.store.GetApiKeysByProfileID(context.Background(), 1)
	require.NoError(t, err)
	var keyID int32
	for _, key := range keys {
		if key.Key == "first-key" {
			keyID = key.ID
		}
	}
	require.NotZero(t, keyID)
	entered := make(chan struct{})
	release := make(chan struct{})
	done := make(chan error, 1)
	h := middleware.IngestionKey(a.store)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(entered)
		<-release
		p := middleware.ProfileFromContext(r.Context())
		done <- a.store.SaveListen(r.Context(), db.SaveListenOpts{ProfileID: p.ID, TrackID: 1, Time: time.Unix(1700000000, 0)})
		w.WriteHeader(204)
	}))
	req := httptest.NewRequest("POST", "/", nil)
	req.Header.Set("Authorization", "Token first-key")
	rec := httptest.NewRecorder()
	finished := make(chan struct{})
	go func() { h.ServeHTTP(rec, req); close(finished) }()
	select {
	case <-entered:
	case <-time.After(5 * time.Second):
		t.Fatal("request did not reach barrier")
	}
	require.NoError(t, a.store.DeleteProfileApiKey(context.Background(), keyID, 1))
	close(release)
	select {
	case err := <-done:
		require.NoError(t, err)
	case <-time.After(5 * time.Second):
		t.Fatal("authorized request did not complete")
	}
	<-finished
	require.Equal(t, 204, rec.Code)
	n, err := a.store.ProfileListenCount(context.Background(), 1)
	require.NoError(t, err)
	require.EqualValues(t, 1, n)
	next := httptest.NewRecorder()
	h.ServeHTTP(next, req)
	require.Equal(t, 401, next.Code)
}

// TestDeletedProfileRejectsPreviouslyAuthorizedListen prevents reassignment to another profile.
func TestDeletedProfileRejectsPreviouslyAuthorizedListen(t *testing.T) {
	a := newProfileTestApp(t)
	ctx := context.Background()
	for _, q := range []string{`INSERT INTO releases(id) VALUES(1)`, `INSERT INTO tracks(id,release_id) VALUES(1,1)`} {
		require.NoError(t, a.store.Exec(q))
	}
	p, err := a.store.GetListeningProfileByID(ctx, 1)
	require.NoError(t, err)
	entered, release := make(chan struct{}), make(chan struct{})
	done := make(chan error, 1)
	h := middleware.IngestionKey(a.store)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(entered)
		<-release
		profile := middleware.ProfileFromContext(r.Context())
		done <- a.store.SaveListen(r.Context(), db.SaveListenOpts{ProfileID: profile.ID, TrackID: 1, Time: time.Unix(1700000000, 0)})
	}))
	req := httptest.NewRequest("POST", "/", nil)
	req.Header.Set("Authorization", "Token first-key")
	go h.ServeHTTP(httptest.NewRecorder(), req)
	select {
	case <-entered:
	case <-time.After(5 * time.Second):
		t.Fatal("request did not reach barrier")
	}
	err = a.store.DeleteListeningProfile(ctx, p.ID, p.DisplayName, 0)
	close(release)
	require.NoError(t, err)
	select {
	case err := <-done:
		require.Error(t, err)
	case <-time.After(5 * time.Second):
		t.Fatal("authorized request did not complete")
	}
	n, err := a.store.ProfileListenCount(ctx, 2)
	require.NoError(t, err)
	require.Zero(t, n)
	next := httptest.NewRecorder()
	h.ServeHTTP(next, req)
	require.Equal(t, http.StatusUnauthorized, next.Code)
}
