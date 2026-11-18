package adminui

import (
	"context"
	"github.com/ebrgr/TuneTrail/internal/admin"
	"github.com/ebrgr/TuneTrail/internal/db"
	"github.com/ebrgr/TuneTrail/internal/db/sqlite"
	"github.com/ebrgr/TuneTrail/internal/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
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
	h.submissions[otherID] = submissionTicket{other, time.Now().Add(time.Minute)}
	oldest, err := h.newSubmission(user)
	require.NoError(t, err)
	oldestID := uuid.MustParse(oldest)
	h.submissions[oldestID] = submissionTicket{user, time.Now().Add(2 * time.Minute)}
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
	h.submissions[expiredID] = submissionTicket{session, time.Now().Add(-time.Second)}
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
		h.submissions[uuid.New()] = submissionTicket{uuid.New(), time.Now().Add(30 * time.Minute)}
	}
	_, err = h.newSubmission(session)
	require.NoError(t, err)
	require.Len(t, h.submissions, 4096)
	_, err = h.newSubmission(uuid.New())
	require.ErrorIs(t, err, db.ErrConflict)
	require.Len(t, h.submissions, 4096)
}
