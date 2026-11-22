package sqlite

import (
	"context"
	"database/sql"
	"errors"

	"github.com/ebrgr/TuneTrail/internal/dedup"
	"github.com/ebrgr/TuneTrail/internal/ingest"
	"github.com/google/uuid"
)

const trustedReplayReason = `["trusted_external_replay"]`

// reviewTrustedReplay keeps one review receipt per resolved external key. The
// original receipt retains the unique key and remains linked to its listen.
func reviewTrustedReplay(ctx context.Context, tx *sql.Conn, batch ingest.Batch, item ingest.Item, original, batchID, policy string, fallback ingest.Result) (ingest.Result, error) {
	var enabled bool
	if err := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM profile_ingest_flags
		WHERE profile_id=? AND review=1)`, batch.ProfileID).Scan(&enabled); err != nil || !enabled {
		return fallback, err
	}

	id := uuid.NewSHA1(uuid.NameSpaceOID, []byte("tunetrail:trusted-replay:"+original)).String()
	received := batch.ReceivedAt.Unix()
	// Carry the external identity for later comparisons without taking ownership of its key.
	result, err := tx.ExecContext(ctx, `INSERT INTO ingest_events(
		receipt_id,profile_id,source_instance_id,schema_version,source_type,adapter_version,batch_id,item_ordinal,operation,received_at,
		external_namespace,external_event_id,listened_at,original_time,time_precision,time_semantics,original_offset,
		original_snapshot,snapshot_digest,digest_version,mode,resolver_variant,delivery_policy,state,identity_status,
		duplicate_decision,reason_codes,evidence,rule_version,resolution_version,resolved_at,occurrence_id,recording_id)
		SELECT ?,e.profile_id,e.source_instance_id,e.schema_version,e.source_type,e.adapter_version,?,e.item_ordinal,e.operation,?,
			e.external_namespace,e.external_event_id,e.listened_at,e.original_time,e.time_precision,e.time_semantics,e.original_offset,
			e.original_snapshot,e.snapshot_digest,e.digest_version,'active','occurrence',?,'pending_duplicate','resolved',
			'review',?,json_object('candidate_event_id',l.event_id,'candidate_receipt_id',e.receipt_id,'trusted_replay',json('true')),
			?,1,?,o.id,o.recording_id
		FROM ingest_events e JOIN listens l ON (l.profile_id,l.event_id)=(e.profile_id,e.event_id)
		JOIN occurrences o ON o.id=l.track_id
		WHERE e.profile_id=? AND e.receipt_id=? AND e.external_id_trusted=1 AND e.external_namespace=? AND e.external_event_id=?
		ON CONFLICT(receipt_id) DO NOTHING`, id, batchID, received, policy, trustedReplayReason, occurrenceRuleVersion, received,
		batch.ProfileID, original, item.ExternalNamespace, item.ExternalID)
	if err != nil {
		return fallback, err
	}
	if inserted, err := result.RowsAffected(); err != nil {
		return fallback, err
	} else if inserted == 1 {
		if _, err := tx.ExecContext(ctx, `INSERT INTO ingest_revisions
			SELECT profile_id,receipt_id,resolution_version,'automatic',duplicate_decision,reason_codes,evidence,rule_version,resolved_at
			FROM ingest_events WHERE profile_id=? AND receipt_id=?`, batch.ProfileID, id); err != nil {
			return fallback, err
		}
	}
	replay := dedup.Receipt{ID: id}
	err = tx.QueryRowContext(ctx, `SELECT state,coalesce(duplicate_decision,'')
		FROM ingest_events WHERE profile_id=? AND receipt_id=?`, batch.ProfileID, id).Scan(&replay.State, &replay.Decision)
	if errors.Is(err, sql.ErrNoRows) {
		return fallback, nil
	}
	if err != nil {
		return fallback, err
	}
	return replayIngest(&replay), nil
}
