package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
)

// backupBeforeProfiles takes a consistent SQLite snapshot, including committed WAL
// pages. Unlike copying koito.db, VACUUM INTO also works with an active WAL.
// A backup failure prevents migration. Fresh or already-migrated DBs are skipped.
func backupBeforeProfiles(ctx context.Context, conn *sql.DB, databasePath string) (string, error) {
	var existing bool
	if err := conn.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM sqlite_master WHERE type = 'table' AND name = 'users')`).Scan(&existing); err != nil {
		return "", err
	}
	if !existing {
		return "", nil
	}
	var applied bool
	err := conn.QueryRowContext(ctx, `SELECT is_applied FROM goose_db_version WHERE version_id = 3 ORDER BY id DESC LIMIT 1`).Scan(&applied)
	if err != nil && err != sql.ErrNoRows {
		return "", err
	}
	if applied {
		return "", nil
	}
	f, err := os.CreateTemp(filepath.Dir(databasePath), filepath.Base(databasePath)+".pre-profiles-*.bak")
	if err != nil {
		return "", err
	}
	filename := f.Name()
	if err := f.Close(); err != nil {
		os.Remove(filename)
		return "", err
	}
	if _, err := conn.ExecContext(ctx, `VACUUM INTO ?`, filename); err != nil {
		os.Remove(filename)
		return "", fmt.Errorf("snapshot: %w", err)
	}
	return filename, nil
}

// backupBeforeUpgrade snapshots each supported source with its committed WAL.
func backupBeforeUpgrade(ctx context.Context, conn *sql.DB, databasePath string, version int) (string, error) {
	if version < 3 {
		return backupBeforeProfiles(ctx, conn, databasePath)
	}
	pattern := ".pre-ingest-*.bak"
	if version == 4 {
		pattern = ".pre-accounts-*.bak"
	}
	if version == 5 {
		pattern = ".pre-authorization-*.bak"
	}
	// Snapshot the pre-token schema before installing the credential table.
	if version == 6 {
		pattern = ".pre-account-tokens-*.bak"
	}
	// Keep the v7 snapshot separate from backups of previous upgrades.
	if version == 7 {
		pattern = ".pre-recording-identity-*.bak"
	}
	// Preserve the v8 snapshot for rollback across ingestion activation.
	if version == 8 {
		pattern = ".pre-ingest-activation-*.bak"
	}
	// Keep a dedicated snapshot for rollback to the v9 writer.
	if version == 9 {
		pattern = ".pre-occurrence-ingest-*.bak"
	}
	// Preserve the pre-cut schema when installing writer guards.
	if version == 10 {
		pattern = ".pre-occurrence-writer-*.bak"
	}
	if version == 11 {
		pattern = ".pre-relay-delivery-*.bak"
	}
	// Preserve a rollback copy before adding occurrence archive mappings.
	if version == 13 {
		pattern = ".pre-occurrence-archive-*.bak"
	}
	// Preserve v14 before backfilling stable export sequence numbers.
	if version == 14 {
		pattern = ".pre-archive-export-boundary-*.bak"
	}
	filename, err := tempDatabase(filepath.Dir(databasePath), filepath.Base(databasePath)+pattern)
	if err != nil {
		return "", err
	}
	if _, err := conn.ExecContext(ctx, `VACUUM INTO ?`, filename); err != nil {
		os.Remove(filename)
		return "", fmt.Errorf("pre-upgrade snapshot: %w", err)
	}
	return filename, nil
}
