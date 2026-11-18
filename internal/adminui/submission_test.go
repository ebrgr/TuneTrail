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
