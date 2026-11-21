package sqlite

import (
	"context"
	"testing"
	"time"

	"github.com/ebrgr/TuneTrail/internal/admin"
	"github.com/ebrgr/TuneTrail/internal/db"
	"github.com/ebrgr/TuneTrail/internal/ingest"
	"github.com/ebrgr/TuneTrail/internal/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// TestIngestActivationConfirmation checks root authorization, scope and stale previews.
func TestIngestActivationConfirmation(t *testing.T) {
	ctx := context.Background()
	s, err := NewInMemory()
	require.NoError(t, err)
	t.Cleanup(func() { s.Close(ctx) })
	root, err := s.SaveUser(ctx, db.SaveUserOpts{Username: "root", Password: "root-password", Role: models.UserRoleAdmin})
	require.NoError(t, err)
	session, err := admin.Login(ctx, s, root.Username, "root-password", false)
	require.NoError(t, err)
	target, err := s.CreateListeningProfile(ctx, "Target", "target", "", "target-key")
	require.NoError(t, err)
	other, err := s.CreateListeningProfile(ctx, "Other", "other", "", "other-key")
	require.NoError(t, err)

	batch := ingest.Batch{ProfileID: target.ID, Source: ingest.ShadowSourceFile, Operation: "import", ReceivedAt: time.Now(),
		FileFormat: "test", FileDigest: ingest.Digest([]byte("file")),
		Items: []ingest.Item{{Snapshot: ingest.Snapshot{Artist: "A", Title: "T"}}}}
	first, err := s.ReceiveBatch(ctx, batch)
	require.NoError(t, err)
	otherBatch := batch
	otherBatch.ProfileID = other.ID
	_, err = s.ReceiveBatch(ctx, otherBatch)
	require.NoError(t, err)

	_, err = s.PreviewIngestActivation(ctx, uuid.Nil, target.ID)
	require.ErrorIs(t, err, db.ErrForbidden)
	preview, err := s.PreviewIngestActivation(ctx, session.ID, target.ID)
	require.NoError(t, err)
	require.EqualValues(t, 1, preview.Receipts)
	require.Len(t, preview.Groups, 1)
	require.Positive(t, preview.Groups[0].SnapshotBytes)
	require.False(t, preview.Active)
	require.False(t, preview.Review)
	err = s.SetIngestActivation(ctx, session.ID, target.ID, false, true, "")
	require.ErrorIs(t, err, db.ErrInvalidInput)

	batch.Items[0].Snapshot.Title = "Conflicting variant"
	_, err = s.ReceiveBatch(ctx, batch)
	require.NoError(t, err)
	require.ErrorIs(t, s.SetIngestActivation(ctx, session.ID, target.ID, true, false, preview.Digest), db.ErrConflict)
	preview, err = s.PreviewIngestActivation(ctx, session.ID, target.ID)
	require.NoError(t, err)
	require.EqualValues(t, 1, preview.Groups[0].Observations)
	require.NoError(t, s.SetIngestActivation(ctx, session.ID, target.ID, true, true, preview.Digest))
	enabled, err := s.PreviewIngestActivation(ctx, session.ID, target.ID)
	require.NoError(t, err)
	require.True(t, enabled.Review)
	require.Zero(t, scalar(t, s, `SELECT count(*) FROM ingest_events WHERE receipt_id=?`, first[0].ReceiptID))
	require.Zero(t, scalar(t, s, `SELECT count(*) FROM ingest_observations`))
	require.Equal(t, 1, scalar(t, s, `SELECT count(*) FROM ingest_events WHERE profile_id=? AND mode='shadow'`, other.ID))
	require.Equal(t, 1, scalar(t, s, `SELECT count(*) FROM integrations WHERE profile_id=?`, target.ID))

	active, results, err := s.AdmitIngest(ctx, batch, true, nil, nil)
	require.NoError(t, err)
	require.True(t, active)
	require.False(t, results[0].Replay)
	require.NoError(t, s.SetIngestActivation(ctx, session.ID, target.ID, false, false, ""))
	batch.FileDigest = ingest.Digest([]byte("second file"))
	_, err = s.ReceiveBatch(ctx, batch)
	require.NoError(t, err)
	preview, err = s.PreviewIngestActivation(ctx, session.ID, target.ID)
	require.NoError(t, err)
	require.EqualValues(t, 1, preview.Receipts)
	require.NoError(t, s.SetIngestActivation(ctx, session.ID, target.ID, true, false, preview.Digest))
	require.Equal(t, 1, scalar(t, s, `SELECT count(*) FROM ingest_events WHERE profile_id=? AND mode='active' AND receipt_id=?`, target.ID, results[0].ReceiptID))
	require.Zero(t, scalar(t, s, `SELECT count(*) FROM ingest_events WHERE profile_id=? AND mode='shadow'`, target.ID))
}
