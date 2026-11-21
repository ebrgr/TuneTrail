package sqlite

import (
	"context"
	"database/sql"
	"time"

	"github.com/ebrgr/TuneTrail/internal/db"
	"github.com/ebrgr/TuneTrail/internal/relay"
	"github.com/google/uuid"
)

// relayConfigurationVersion fences permission snapshots and network validation.
func relayConfigurationVersion(ctx context.Context, q ingestActivationReader, profile int32) (int64, error) {
	var version int64
	err := q.QueryRowContext(ctx, "SELECT coalesce((SELECT config_version FROM relay_destinations WHERE profile_id=?),0)", profile).Scan(&version)
	return version, err
}

// probeSavedRelay validates a saved destination without holding the writer reservation.
func (s *Sqlite) probeSavedRelay(ctx context.Context, profile int32, requireHealthy bool) (int64, bool, error) {
	version, err := relayConfigurationVersion(ctx, s.db, profile)
	if err != nil {
		return 0, false, err
	}
	url, token, err := s.GetProfileRelay(ctx, profile)
	if err != nil || url == "" || token == "" {
		return version, false, err
	}
	if requireHealthy {
		_, until, reason, err := s.RelayQueueStatus(ctx, profile)
		if err != nil {
			return version, false, err
		}
		if reason != "" && (!until.Valid || time.Now().Before(until.Time)) {
			return version, false, db.ErrConflict
		}
	}
	permissions, err := s.RelayPermissions(ctx, profile)
	if err != nil {
		return version, false, err
	}
	if _, err := relay.NewDeliverer(permissions, nil).Test(ctx, url, token); err != nil {
		return version, false, db.ErrConflict
	}
	return version, true, nil
}

// ResumeRelay checks the saved destination and clears its pause for a live root session.
func (s *Sqlite) ResumeRelay(ctx context.Context, session uuid.UUID, profile int32) error {
	if err := checkIngestRoot(ctx, s.db, session); err != nil {
		return err
	}
	version, present, err := s.probeSavedRelay(ctx, profile, false)
	if err != nil {
		return err
	}
	if !present {
		return db.ErrNotFound
	}
	return s.withIngestWriter(ctx, func(tx *sql.Conn) error {
		if err := checkIngestRoot(ctx, tx, session); err != nil {
			return err
		}
		result, err := tx.ExecContext(ctx, "UPDATE relay_destinations SET config_version=config_version+1,pause_class=NULL,paused_until=NULL,backoff_count=0 WHERE profile_id=? AND config_version=?", profile, version)
		if err != nil {
			return err
		}
		n, err := result.RowsAffected()
		if err == nil && n != 1 {
			return db.ErrConflict
		}
		return err
	})
}

// ConvertOccurrence allows the one-way cut for active profiles with a verified or absent relay.
func (s *Sqlite) ConvertOccurrence(ctx context.Context, session uuid.UUID, profile int32) error {
	if err := checkIngestRoot(ctx, s.db, session); err != nil {
		return err
	}
	version, _, err := s.probeSavedRelay(ctx, profile, true)
	if err != nil {
		return err
	}
	return s.withIngestWriter(ctx, func(tx *sql.Conn) error {
		if err := checkIngestRoot(ctx, tx, session); err != nil {
			return err
		}
		current, err := relayConfigurationVersion(ctx, tx, profile)
		if err != nil {
			return err
		}
		if current != version {
			return db.ErrConflict
		}
		result, err := tx.ExecContext(ctx, "UPDATE profile_ingest_flags SET occurrence_write=1 WHERE profile_id=? AND active=1 AND (SELECT policy FROM relay_delivery_gate WHERE id=1)='durable' AND NOT EXISTS(SELECT 1 FROM relay_destinations WHERE profile_id=? AND pause_class IS NOT NULL AND (paused_until IS NULL OR paused_until>unixepoch()) AND EXISTS(SELECT 1 FROM profile_relays WHERE profile_id=?))", profile, profile, profile)
		if isOccurrenceGuard(err, "Occurrence writing requires") {
			return db.ErrConflict
		}
		if err != nil {
			return err
		}
		n, err := result.RowsAffected()
		if err == nil && n == 0 {
			return db.ErrConflict
		}
		return err
	})
}

// validateOccurrenceRelay checks a replacement endpoint before opening its write transaction.
func (s *Sqlite) validateOccurrenceRelay(ctx context.Context, profile int32, base, token string) (bool, int64, error) {
	converted, err := occurrenceWriteFlag(ctx, s.db, profile)
	if err != nil || !converted {
		return false, 0, err
	}
	version, err := relayConfigurationVersion(ctx, s.db, profile)
	if err != nil {
		return false, 0, err
	}
	permissions, err := s.RelayPermissions(ctx, profile)
	if err != nil {
		return false, 0, err
	}
	if _, err := relay.NewDeliverer(permissions, nil).Test(ctx, base, token); err != nil {
		return false, 0, db.ErrConflict
	}
	return true, version, nil
}

// requireOccurrenceRelayValidation rechecks the conversion and configuration under the writer lock.
func requireOccurrenceRelayValidation(ctx context.Context, tx *sql.Conn, profile int32, validated bool, version int64) error {
	converted, err := occurrenceWriteFlag(ctx, tx, profile)
	if err != nil || !converted {
		return err
	}
	if !validated {
		return db.ErrConflict
	}
	current, err := relayConfigurationVersion(ctx, tx, profile)
	if err != nil {
		return err
	}
	if current != version {
		return db.ErrConflict
	}
	return nil
}
