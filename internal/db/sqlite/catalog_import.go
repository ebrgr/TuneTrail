package sqlite

import (
	"context"
	"database/sql"
	"github.com/ebrgr/TuneTrail/internal/db"
)

// importedMBIDAllowed prevents automatic ingestion from publishing an owner's
// imported identity into the global MBID namespace; explicit root edits may validate it.
func importedMBIDAllowed(ctx context.Context, tx *sql.Tx, table, key string, id int32) (bool, error) {
	if db.CatalogEditorial(ctx) {
		return true, nil
	}
	var restricted bool
	// Table and key are internal constants at the three catalog store call sites.
	err := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM `+table+` WHERE `+key+`=? AND is_primary=1 AND source LIKE 'ProfileImport:%')`, id).Scan(&restricted)
	return !restricted, err
}
