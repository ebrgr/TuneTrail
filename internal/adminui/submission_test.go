package adminui

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/ebrgr/TuneTrail/internal/admin"
	"github.com/ebrgr/TuneTrail/internal/adminweb"
	"github.com/ebrgr/TuneTrail/internal/db"
	"github.com/ebrgr/TuneTrail/internal/db/sqlite"
	"github.com/ebrgr/TuneTrail/internal/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// TestOneUseSubmission protects duplicate actions while permitting separate tabs.
func TestOneUseSubmission(t *testing.T) {
	h := &Handler{requireSubmission: true, submissions: map[uuid.UUID]submissionTicket{}}
	session := uuid.New()
	one, err := h.newSubmission(session)
	require.NoError(t, err)
	two, err := h.newSubmission(session)
	require.NoError(t, err)
	require.ErrorIs(t, h.consumeSubmission(uuid.New(), one), db.ErrConflict)
	require.NoError(t, h.consumeSubmission(session, one))
	require.ErrorIs(t, h.consumeSubmission(session, one), db.ErrConflict)
	require.NoError(t, h.consumeSubmission(session, two))
	require.ErrorIs(t, h.consumeSubmission(session, ""), db.ErrConflict)
}

// TestUserSubmissionIsolation159 verifies tickets work for ordinary sessions without crossing accounts.
func TestUserSubmissionIsolation159(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.NewInMemory()
	require.NoError(t, err)
	t.Cleanup(func() { store.Close(ctx) })
	h := &Handler{requireSubmission: true, submissions: map[uuid.UUID]submissionTicket{}}
	var sessions []uuid.UUID
	for _, name := range []string{"listener-one", "listener-two"} {
		_, err = store.SaveUser(ctx, db.SaveUserOpts{Username: name, Password: "test-password", Role: models.UserRoleUser})
		require.NoError(t, err)
		session, err := admin.Login(ctx, store, name, "test-password", false)
		require.NoError(t, err)
		sessions = append(sessions, session.ID)
	}
	token, err := h.newSubmission(sessions[0])
	require.NoError(t, err)
	require.ErrorIs(t, h.consumeSubmission(sessions[1], token), db.ErrConflict)
	require.NoError(t, h.consumeSubmission(sessions[0], token))
	require.ErrorIs(t, h.consumeSubmission(sessions[0], token), db.ErrConflict)
}

// TestSubmissionQuotaEvictsOnlyOwnOldest159 protects other sessions and preserves one-use semantics.
func TestSubmissionQuotaEvictsOnlyOwnOldest159(t *testing.T) {
	h := &Handler{requireSubmission: true, submissions: map[uuid.UUID]submissionTicket{}}
	user, other := uuid.New(), uuid.New()
	otherToken, err := h.newSubmission(other)
	require.NoError(t, err)
	otherID := uuid.MustParse(otherToken)
	// The globally oldest ticket belongs to another session and must never be evicted.
	h.submissions[otherID] = submissionTicket{other, time.Now().Add(time.Minute), false}
	oldest, err := h.newSubmission(user)
	require.NoError(t, err)
	oldestID := uuid.MustParse(oldest)
	h.submissions[oldestID] = submissionTicket{user, time.Now().Add(2 * time.Minute), false}
	for i := 1; i < maxSessionSubmissions; i++ {
		_, err = h.newSubmission(user)
		require.NoError(t, err)
	}
	_, err = h.newSubmission(user)
	require.NoError(t, err)
	require.NotContains(t, h.submissions, oldestID)
	require.ErrorIs(t, h.consumeSubmission(user, oldest), db.ErrConflict)
	var latest string
	for i := 0; i < 500; i++ {
		latest, err = h.newSubmission(user)
		require.NoError(t, err)
	}
	require.Len(t, h.submissions, maxSessionSubmissions+1)
	require.ErrorIs(t, h.consumeSubmission(other, latest), db.ErrConflict)
	require.NoError(t, h.consumeSubmission(user, latest))
	require.ErrorIs(t, h.consumeSubmission(user, latest), db.ErrConflict)
	require.NoError(t, h.consumeSubmission(other, otherToken))
}

// TestSubmissionExpiryAndGlobalCeiling159 preserves TTL pruning and the global safety ceiling.
func TestSubmissionExpiryAndGlobalCeiling159(t *testing.T) {
	h := &Handler{requireSubmission: true, submissions: map[uuid.UUID]submissionTicket{}}
	session := uuid.New()
	expired, err := h.newSubmission(session)
	require.NoError(t, err)
	expiredID := uuid.MustParse(expired)
	require.WithinDuration(t, time.Now().Add(30*time.Minute), h.submissions[expiredID].Expires, time.Second)
	h.submissions[expiredID] = submissionTicket{session, time.Now().Add(-time.Second), false}
	require.ErrorIs(t, h.consumeSubmission(session, expired), db.ErrConflict)
	_, err = h.newSubmission(session)
	require.NoError(t, err)
	require.NotContains(t, h.submissions, expiredID)
	for i := 1; i < maxSessionSubmissions; i++ {
		_, err = h.newSubmission(session)
		require.NoError(t, err)
	}
	// Fill the pool with independent sessions so a new session cannot displace any of them.
	for len(h.submissions) < 4096 {
		h.submissions[uuid.New()] = submissionTicket{uuid.New(), time.Now().Add(30 * time.Minute), false}
	}
	_, err = h.newSubmission(session)
	require.NoError(t, err)
	require.Len(t, h.submissions, 4096)
	_, err = h.newSubmission(uuid.New())
	require.ErrorIs(t, err, db.ErrConflict)
	require.Len(t, h.submissions, 4096)
	_, err = h.issueSubmission(uuid.New(), true)
	require.ErrorIs(t, err, db.ErrConflict)
	require.Len(t, h.submissions, 4096)
}

// TestLogoutSubmissionRenewsExpiry verifies renewal through the real logout form and POST.
func TestLogoutSubmissionRenewsExpiry(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.NewInMemory()
	require.NoError(t, err)
	defer store.Close(ctx)
	_, err = store.SaveUser(ctx, db.SaveUserOpts{Username: "listener", Password: "test-password", Role: models.UserRoleUser})
	require.NoError(t, err)
	session, err := admin.Login(ctx, store, "listener", "test-password", false)
	require.NoError(t, err)
	h, err := New(store, Options{RequireSubmissionToken: true})
	require.NoError(t, err)

	get := func() (string, string) {
		r := httptest.NewRequest(http.MethodGet, "/admin/sign-out", nil)
		r.AddCookie(&http.Cookie{Name: adminweb.SessionCookie, Value: session.ID.String()})
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		require.Equal(t, http.StatusOK, w.Code)
		field := func(name string) string {
			match := regexp.MustCompile(`name="` + name + `" value="([^"]+)"`).FindStringSubmatch(w.Body.String())
			require.Len(t, match, 2)
			return match[1]
		}
		return field("csrf_token"), field("submission_token")
	}
	_, first := get()
	id := uuid.MustParse(first)
	now := time.Now()
	h.submissions[id] = submissionTicket{session.ID, now.Add(time.Minute), true}
	expired := uuid.New()
	h.submissions[expired] = submissionTicket{uuid.New(), now.Add(-time.Minute), false}

	csrf, renewed := get()
	require.Equal(t, first, renewed)
	require.False(t, h.submissions[id].Expires.Before(now.Add(30*time.Minute)))
	require.False(t, h.submissions[id].Expires.After(time.Now().Add(30*time.Minute)))
	require.NotContains(t, h.submissions, expired)

	form := url.Values{"csrf_token": {csrf}, "submission_token": {renewed}}
	post := func(path string) *httptest.ResponseRecorder {
		r := httptest.NewRequest(http.MethodPost, path, strings.NewReader(form.Encode()))
		r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		r.AddCookie(&http.Cookie{Name: adminweb.SessionCookie, Value: session.ID.String()})
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		return w
	}
	require.Equal(t, http.StatusConflict, post("/admin/account/username").Code)
	require.Equal(t, http.StatusSeeOther, post("/admin/logout").Code)
	// A consumed ticket stays invalid independently of the revoked session.
	require.ErrorIs(t, h.consumeTicket(session.ID, renewed, true), db.ErrConflict)
	_, err = admin.Authenticate(ctx, store, session.ID)
	require.ErrorIs(t, err, admin.ErrUnauthorized)
}
