package admin_test

import (
	"context"
	"testing"

	"github.com/ebrgr/TuneTrail/internal/admin"
	"github.com/ebrgr/TuneTrail/internal/db"
	"github.com/stretchr/testify/require"
)

func TestProfileCreationIsAtomicAndReportsDatabaseErrors(t *testing.T) {
	s, user := accountFixture(t)
	ctx := context.Background()
	require.NoError(t, s.Exec(`CREATE TRIGGER fail_initial_key BEFORE INSERT ON api_keys BEGIN SELECT RAISE(ABORT,'injected database failure'); END`))
	_, err := admin.CreateProfile(ctx, s, user, "Alice", "alice", "")
	require.Error(t, err)
	require.NotErrorIs(t, err, db.ErrInvalidInput)
	require.NotErrorIs(t, err, db.ErrConflict)
	_, err = s.GetListeningProfile(ctx, "alice")
	require.ErrorIs(t, err, db.ErrNotFound)
	require.NoError(t, s.Exec(`DROP TRIGGER fail_initial_key`))
	p, err := admin.CreateProfile(ctx, s, user, "Alice", "alice", "")
	require.NoError(t, err)
	keys, err := s.GetApiKeysByProfileID(ctx, p.ID)
	require.NoError(t, err)
	require.Len(t, keys, 1)
	_, err = admin.CreateProfile(ctx, s, user, "Other", "alice", "")
	require.ErrorIs(t, err, db.ErrConflict)
}
