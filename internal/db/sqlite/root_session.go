package sqlite

import (
	"context"
	"database/sql"

	"github.com/ebrgr/TuneTrail/internal/db"
	"github.com/google/uuid"
)

type ingestQueryer interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}

// checkIngestRoot validates a live root session inside the caller's SQLite transaction.
func checkIngestRoot(ctx context.Context, tx ingestQueryer, session uuid.UUID) error {
	var root bool
	err := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM sessions s JOIN users u ON u.id=s.user_id
		WHERE s.id=? AND s.expires_at>unixepoch() AND u.role='admin' AND u.status='active')`, session.String()).Scan(&root)
	if err == nil && !root {
		return db.ErrForbidden
	}
	return err
}
