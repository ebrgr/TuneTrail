package sqlite

import (
	"database/sql"
	"fmt"
	"strings"

	migrations "github.com/ebrgr/TuneTrail/db/migrations_sqlite"
)

// validateSchemaLayout checks columns, constraints, indexes and guards against
// the embedded DDL, so a forged version alone can never enable an incomplete v4.
func validateSchemaLayout(db *sql.DB, version int) error {
	reference, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		return fmt.Errorf("open schema reference: %w", err)
	}
	defer reference.Close()
	reference.SetMaxOpenConns(1)
	for _, name := range []string{"001_initial_schema.sql", "002_unique_mbid.sql", "003_listening_profiles.sql", "004_ingest_infrastructure.sql"}[:version] {
		ddl, err := migrations.Files.ReadFile(name)
		if err != nil {
			return fmt.Errorf("read schema reference: %w", err)
		}
		if _, err := reference.Exec(strings.SplitN(string(ddl), "-- +goose Down", 2)[0]); err != nil {
			return fmt.Errorf("build schema reference: %w", err)
		}
	}
	rows, err := reference.Query(`SELECT type,name,sql FROM sqlite_master WHERE sql IS NOT NULL AND name NOT LIKE 'sqlite_%'`)
	if err != nil {
		return fmt.Errorf("inspect schema reference: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var kind, name, want, got string
		if err := rows.Scan(&kind, &name, &want); err != nil {
			return err
		}
		if err := db.QueryRow(`SELECT sql FROM sqlite_master WHERE type=? AND name=?`, kind, name).Scan(&got); err != nil {
			return fmt.Errorf("incompatible v%d schema: required %s %s: %w", version, kind, name, err)
		}
		if strings.Join(strings.Fields(got), " ") != strings.Join(strings.Fields(want), " ") {
			return fmt.Errorf("incompatible v%d schema: altered %s %s", version, kind, name)
		}
	}
	return rows.Err()
}
