package sqlite

import (
	"context"
	"testing"
	"time"

	"github.com/ebrgr/TuneTrail/internal/db"
	"github.com/stretchr/testify/require"
)

// TestIngestBetweenReviewAndConfirmKeepsBothProfilesAndRejectsStalePlan uses a barrier.
func TestIngestBetweenReviewAndConfirmKeepsBothProfilesAndRejectsStalePlan(t *testing.T) {
	s := mergeFixture(t)
	ctx := context.Background()
	plan, err := s.PreviewCatalogMutation(ctx, db.CatalogPlan{Action: "merge", Kind: "track", From: 1, To: 2})
	require.NoError(t, err)
	started := make(chan struct{})
	done := make(chan error, 1)
	go func() {
		close(started)
		done <- s.SaveListen(ctx, db.SaveListenOpts{ProfileID: 1, TrackID: 1, Time: time.Unix(999999, 0)})
	}()
	<-started
	select {
	case err := <-done:
		require.NoError(t, err)
	case <-time.After(5 * time.Second):
		t.Fatal("ingestion blocked")
	}
	require.ErrorIs(t, s.ApplyCatalogMutation(ctx, plan), db.ErrConflict)
	require.Equal(t, 1, scalar(t, s, `SELECT COUNT(*) FROM listens WHERE profile_id=1 AND track_id=1 AND listened_at=999999`))
	require.Equal(t, 1, scalar(t, s, `SELECT COUNT(*) FROM tracks WHERE id=1`))
	require.Equal(t, 1, scalar(t, s, `SELECT COUNT(*) FROM tracks WHERE id=2`))
	assertForeignKeys(t, s)
}

// TestIngestBetweenDeleteReviewAndConfirmRejectsStalePlan protects the new listen.
func TestIngestBetweenDeleteReviewAndConfirmRejectsStalePlan(t *testing.T) {
	s := mergeFixture(t)
	ctx := context.Background()
	plan, err := s.PreviewCatalogMutation(ctx, db.CatalogPlan{Action: "delete", Kind: "track", From: 1})
	require.NoError(t, err)
	require.NoError(t, s.SaveListen(ctx, db.SaveListenOpts{ProfileID: 1, TrackID: 1, Time: time.Unix(999999, 0)}))
	require.ErrorIs(t, s.ApplyCatalogMutation(ctx, plan), db.ErrConflict)
	require.Equal(t, 1, scalar(t, s, `SELECT COUNT(*) FROM listens WHERE profile_id=1 AND track_id=1 AND listened_at=999999`))
	require.Equal(t, 1, scalar(t, s, `SELECT COUNT(*) FROM tracks WHERE id=1`))
	assertForeignKeys(t, s)
}

// TestDeferredConstraintFailureAtCommitLeavesNoPartialWrite validates SQLite rollback at commit.
func TestDeferredConstraintFailureAtCommitLeavesNoPartialWrite(t *testing.T) {
	s, _, _ := migrationDB(t, 3)
	ctx := context.Background()
	tx, err := s.db.BeginTx(ctx, nil)
	require.NoError(t, err)
	_, err = tx.ExecContext(ctx, `PRAGMA defer_foreign_keys=ON`)
	require.NoError(t, err)
	_, err = tx.ExecContext(ctx, `INSERT INTO listens(profile_id,track_id,listened_at) VALUES(999,999,123)`)
	require.NoError(t, err)
	require.Error(t, tx.Commit())
	require.Zero(t, scalar(t, s, `SELECT COUNT(*) FROM listens`))
	assertForeignKeys(t, s)
}
