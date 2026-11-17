package sqlite

import (
	"context"
	"fmt"
	"time"
)

// PurgeClosedIngestDetails removes only expired, unprotected operational details.
// It is an explicit maintenance operation, not a scheduler or an ingest writer.
// Current snapshots/evidence live in the core; open work and active leases win
// over retention. A short write transaction revalidates eligibility on each run.
func (s *Sqlite) PurgeClosedIngestDetails(ctx context.Context, now time.Time, limit int) (int64, error) {
	if limit < 1 || limit > 500 {
		return 0, fmt.Errorf("purge batch limit must be between 1 and 500")
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("begin ingest purge: %w", err)
	}
	defer tx.Rollback()
	var total int64
	for _, spec := range []struct{ table, parent, key, link string }{
		{"ingest_event_details", "ingest_events", "receipt_id", "source_instance_id"},
		{"ingest_observations", "ingest_events", "receipt_id", "source_instance_id"},
		{"outbox_details", "outbox", "outbox_id", "destination_id"},
	} {
		remaining := int64(limit) - total
		if remaining == 0 {
			break
		}
		parentKey := "receipt_id"
		states := "'invalid'"
		if spec.parent == "outbox" {
			parentKey = "id"
			states = "'sent','discarded'"
		}
		query := fmt.Sprintf(`DELETE FROM %s WHERE id IN (
   SELECT d.id FROM %s d JOIN %s p ON p.profile_id=d.profile_id AND p.%s=d.%s
   JOIN integrations i ON i.profile_id=p.profile_id AND i.id=p.%s
   WHERE d.protected=0 AND d.closed_at IS NOT NULL
    AND d.closed_at <= ? - i.details_retention_days*86400
    AND p.closed_at IS NOT NULL AND p.state IN (%s)
    AND (p.lease_until IS NULL OR p.lease_until<=?)
   ORDER BY d.closed_at,d.id LIMIT ?
  )`, spec.table, spec.table, spec.parent, parentKey, spec.key, spec.link, states)
		result, err := tx.ExecContext(ctx, query, now.Unix(), now.Unix(), remaining)
		if err != nil {
			return 0, fmt.Errorf("purge %s: %w", spec.table, err)
		}
		count, err := result.RowsAffected()
		if err != nil {
			return 0, fmt.Errorf("count purged details: %w", err)
		}
		total += count
	}
	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("commit ingest purge: %w", err)
	}
	return total, nil
}
