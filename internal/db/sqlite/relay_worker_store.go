package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/ebrgr/TuneTrail/internal/relay"
)

// ErrRelayConfigInvalid identifies non-retryable configuration or permission data.
var ErrRelayConfigInvalid = errors.New("invalid stored relay configuration")

// RelayTarget checks that the credential and permission snapshot still belongs to the claim.
func (s *Sqlite) RelayTarget(ctx context.Context, batch RelayBatch) (string, string, relay.Permissions, error) {
	var endpoint, token string
	err := s.db.QueryRowContext(ctx, `SELECT p.url,p.token FROM profile_relays p JOIN relay_destinations d USING(profile_id)
		WHERE p.profile_id=? AND d.generation=? AND d.config_version=? AND d.work_version=? AND d.lease_owner=?`,
		batch.Profile, batch.Generation, batch.ConfigVersion, batch.Version, batch.Owner).Scan(&endpoint, &token)
	if err != nil {
		return "", "", nil, err
	}
	if _, _, err := relay.Normalize(endpoint, token); err != nil {
		return "", "", nil, ErrRelayConfigInvalid
	}
	permissions, err := s.RelayPermissions(ctx, batch.Profile)
	// Renewal rechecks config_version after these reads, fencing concurrent configuration changes.
	return endpoint, token, permissions, err
}

// RelayProfiles lists all destination records, including paused and removed configurations.
func (s *Sqlite) RelayProfiles(ctx context.Context) ([]int32, error) {
	rows, err := s.db.QueryContext(ctx, "SELECT profile_id FROM relay_destinations ORDER BY profile_id")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var profiles []int32
	for rows.Next() {
		var id int32
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		profiles = append(profiles, id)
	}
	return profiles, rows.Err()
}

// FailRelaySource retires a retained item whose accepted receipt cannot produce a payload.
func (s *Sqlite) FailRelaySource(ctx context.Context, batch RelayBatch, item RelayItem, cause error) error {
	reason := "source_missing"
	if errors.Is(cause, ErrRelaySourceInvalid) {
		reason = "source_invalid"
	}
	return s.withIngestWriter(ctx, func(tx *sql.Conn) error {
		_, err := tx.ExecContext(ctx, `UPDATE outbox SET state='failed',result=json_object('reason',?),next_attempt_at=NULL,
			lease_owner=NULL,lease_until=NULL WHERE profile_id=? AND id=? AND work_version=? AND lease_owner=? AND relay_generation=?
			AND `+relayDue+` AND EXISTS(SELECT 1 FROM relay_destinations WHERE profile_id=? AND generation=? AND work_version=? AND lease_owner=?)`,
			reason, batch.Profile, item.ID, item.Version, batch.Owner, batch.Generation, batch.Profile, batch.Generation, batch.Version, batch.Owner)
		return err
	})
}

// RelayDeliveryStatus projects safe health data without loading credentials or payloads.
type RelayDeliveryStatus struct {
	ID, EventID, State                    string
	Attempts                              int64
	LastAttempt, NextAttempt, DeliveredAt sql.NullInt64
	Reason                                string
}

// RelayDeliveries reads a stable page of delivery state for the requested profile.
// Only current-generation work can appear pending. Old open work is projected
// as destination_changed even before cleanup materializes the failure.
func (s *Sqlite) RelayDeliveries(ctx context.Context, profile int32, after string) ([]RelayDeliveryStatus, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT o.id,o.event_id,
		CASE WHEN o.relay_generation<>d.generation AND (o.state IN ('pending','retryable') OR json_extract(o.result,'$.reason')='queue_capacity') THEN 'failed' ELSE o.state END,
		o.attempt_count,o.last_attempt_at,CASE WHEN o.state='sent' THEN o.closed_at END,
		CASE WHEN o.relay_generation=d.generation THEN o.next_attempt_at END,
		CASE WHEN o.relay_generation<>d.generation AND (o.state IN ('pending','retryable') OR json_extract(o.result,'$.reason')='queue_capacity') THEN 'destination_changed'
		ELSE coalesce(json_extract(o.result,'$.reason'),json_extract(o.result,'$.class'),'') END
		FROM outbox o JOIN relay_destinations d USING(profile_id)
		WHERE o.profile_id=? AND o.destination_id='relay:'||? AND o.format_version>0 AND o.id>?
		ORDER BY o.id LIMIT 100`, profile, profile, after)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []RelayDeliveryStatus
	for rows.Next() {
		var item RelayDeliveryStatus
		if err := rows.Scan(&item.ID, &item.EventID, &item.State, &item.Attempts, &item.LastAttempt, &item.DeliveredAt, &item.NextAttempt, &item.Reason); err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	return result, rows.Err()
}

// RelayQueueStatus reports current-generation capacity and the destination pause.
// A profile without a destination returns sql.ErrNoRows.
func (s *Sqlite) RelayQueueStatus(ctx context.Context, profile int32) (int, sql.NullTime, string, error) {
	var count int
	var seconds sql.NullInt64
	var class string
	err := s.db.QueryRowContext(ctx, `SELECT open_count,paused_until,coalesce(pause_class,'') FROM relay_destinations WHERE profile_id=?`, profile).
		Scan(&count, &seconds, &class)
	return count, sql.NullTime{Time: time.Unix(seconds.Int64, 0), Valid: seconds.Valid}, class, err
}
