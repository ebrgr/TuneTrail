package adminui

import (
	"github.com/ebrgr/TuneTrail/internal/db"
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
