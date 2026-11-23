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
func relayConfigurationVersion(ctx context.Context, q ingestQueryer, profile int32) (int64, error) {
	var version int64
	err := q.QueryRowContext(ctx, "SELECT coalesce((SELECT config_version FROM relay_destinations WHERE profile_id=?),0)", profile).Scan(&version)
	return version, err
}

// probeSavedRelay validates a saved destination without holding the writer reservation.
func (s *Sqlite) probeSavedRelay(ctx context.Context, profile int32) (int64, bool, error) {
	version, err := relayConfigurationVersion(ctx, s.db, profile)
	if err != nil {
		return 0, false, err
	}
	draft, err := s.RelayAwaitingApproval(ctx, profile)
	if err != nil {
		return version, false, err
	}
	if draft {
		return version, false, db.ErrConflict
	}
	url, token, err := s.GetProfileRelay(ctx, profile)
	if err != nil || url == "" || token == "" {
		return version, false, err
	}
	current, err := relayConfigurationVersion(ctx, s.db, profile)
	if err != nil {
		return version, false, err
	}
	if current != version {
		return version, false, db.ErrConflict
	}
	permissions, err := s.RelayNetworkPolicyForProfile(ctx, profile)
	if err != nil {
		return version, false, err
	}
	if _, err := relay.NewDeliverer(permissions, nil).Test(ctx, url, token); err != nil {
		return version, false, err
	}
	return version, true, nil
}

// ResumeRelay checks the saved destination and clears its pause for a live root session.
func (s *Sqlite) ResumeRelay(ctx context.Context, session uuid.UUID, profile int32) error {
	if err := checkIngestRoot(ctx, s.db, session); err != nil {
		return err
	}
	version, present, err := s.probeSavedRelay(ctx, profile)
	if err != nil {
		return err
	}
	if !present {
		return db.ErrNotFound
	}
	err = s.withIngestWriter(ctx, func(tx *sql.Conn) error {
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
	if err == nil {
		base, _, readErr := s.GetProfileRelay(ctx, profile)
		if readErr == nil {
			relay.LogDiagnostic(ctx, profile, relay.Describe(base, "destination_resumed", ""), time.Now())
		}
	}
	return err
}
