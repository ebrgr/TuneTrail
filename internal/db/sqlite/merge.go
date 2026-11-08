package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/ebrgr/TuneTrail/internal/db"
)

// requireMergeRows checks both catalog identities in the transaction that will write them.
func requireMergeRows(ctx context.Context, tx *sql.Tx, table string, from, to int32) error {
	if from <= 0 || to <= 0 || from == to {
		return db.ErrInvalidInput
	}
	// Only fixed table names supplied by the store methods are allowed.
	switch table {
	case "artists", "releases", "tracks":
	default:
		return db.ErrInvalidInput
	}
	var n int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM `+table+` WHERE id IN (?,?)`, from, to).Scan(&n); err != nil {
		return fmt.Errorf("merge identities: %w", err)
	}
	if n != 2 {
		return db.ErrNotFound
	}
	return nil
}
