package sqlite

import (
	"database/sql"
	"fmt"
	"strings"

	migrations "github.com/ebrgr/TuneTrail/db/migrations_sqlite"
)

// validateSchemaLayout checks columns, constraints, indexes and guards against
// the embedded DDL, so a forged version alone can never enable an incomplete supported schema.
func validateSchemaLayout(db *sql.DB, version int) error {
	reference, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		return fmt.Errorf("open schema reference: %w", err)
	}
	defer reference.Close()
	reference.SetMaxOpenConns(1)
	// Build the reference through the requested version, including the ingestion source summary.
	for _, name := range []string{"001_initial_schema.sql", "002_unique_mbid.sql", "003_listening_profiles.sql", "004_ingest_infrastructure.sql", "005_accounts_profiles.sql", "006_active_admin_guards.sql", "007_account_tokens.sql", "008_recording_identity.sql", "009_ingest_activation.sql", "010_occurrence_ingest.sql", "011_occurrence_writer_guards.sql", relayMigration, "013_ingest_retry.sql", "014_occurrence_archive.sql", "015_archive_export_boundary.sql", "016_ingest_review.sql", "017_ingest_source_summary.sql", "018_ingest_reprocess.sql", "019_ingest_inbox.sql", "020_ingest_discards.sql", "021_occurrence_only.sql", "022_avatar_color.sql", "023_duration_played.sql", "024_enrichment.sql", "025_enrichment_registry.sql", "026_descriptive_metadata.sql"}[:version] {
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
	expectedObjects := make(map[[2]string]struct{})
	for rows.Next() {
		var kind, name, want, got string
		if err := rows.Scan(&kind, &name, &want); err != nil {
			return err
		}
		expectedObjects[[2]string{kind, name}] = struct{}{}
		if err := db.QueryRow(`SELECT sql FROM sqlite_master WHERE type=? AND name=?`, kind, name).Scan(&got); err != nil {
			return fmt.Errorf("incompatible v%d schema: required %s %s: %w", version, kind, name, err)
		}
		if strings.Join(strings.Fields(got), " ") != strings.Join(strings.Fields(want), " ") {
			return fmt.Errorf("incompatible v%d schema: altered %s %s", version, kind, name)
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}
	// Check the upgrade source too, before a rebuild can discard an unknown object.
	return validateIdentityObjects(db, expectedObjects)
}

// validateIdentityObjects rejects obsolete or extra objects on tables touched by the identity and activation migrations.
func validateIdentityObjects(db *sql.DB, expected map[[2]string]struct{}) error {
	rows, err := db.Query(`SELECT type,name FROM sqlite_master WHERE sql IS NOT NULL
		AND tbl_name IN ('tracks','recordings','occurrences','event_origins','listens','ingest_events',
		                 'ingest_event_details','ingest_observations','ingest_revisions','outbox','outbox_details',
		                 'profile_ingest_flags','profile_ingest_review','occurrence_credits','profile_relays','relay_destinations','relay_allowlist','relay_delivery_gate','relay_sent_archive',
                         'archive_identity','archive_catalog','archive_events','recording_moves','listen_export_order','ingest_source_summary')`)
	if err != nil {
		return fmt.Errorf("inspect identity schema: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var kind, name string
		if err := rows.Scan(&kind, &name); err != nil {
			return err
		}
		if _, ok := expected[[2]string{kind, name}]; !ok {
			return fmt.Errorf("incompatible identity schema: unexpected %s %s", kind, name)
		}
	}
	return rows.Err()
}
