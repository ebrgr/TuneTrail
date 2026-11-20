package sqlite

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/ebrgr/TuneTrail/internal/db"
	"github.com/google/uuid"
)

type ingestActivationReader interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}

const shadowPurgeScope = `profile_id=? AND mode='shadow' AND event_id IS NULL`

// PreviewIngestActivation exports only aggregate measurements from one consistent snapshot.
func (s *Sqlite) PreviewIngestActivation(ctx context.Context, session uuid.UUID, profile int32) (db.IngestActivationPreview, error) {
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return db.IngestActivationPreview{}, err
	}
	defer tx.Rollback()
	if err := checkIngestRoot(ctx, tx, session); err != nil {
		return db.IngestActivationPreview{}, err
	}
	preview, err := previewIngestActivation(ctx, tx, profile)
	if err != nil {
		return preview, err
	}
	return preview, tx.Commit()
}

// SetIngestActivation confirms the exact unlinked shadow set before enabling new admissions.
func (s *Sqlite) SetIngestActivation(ctx context.Context, session uuid.UUID, profile int32, active, review bool, digest string) error {
	return s.withIngestWriter(ctx, func(tx *sql.Conn) error {
		if err := checkIngestRoot(ctx, tx, session); err != nil {
			return err
		}
		// Keep review disabled until the operator queue is available.
		if review {
			return db.ErrIngestReviewUnavailable
		}
		current, err := ingestActivationFlags(ctx, tx, profile)
		if err != nil {
			return err
		}
		// The cut is irreversible even through the existing activation API.
		if current.OccurrenceWrite && !active {
			return db.ErrConflict
		}
		if active && (!current.Active || digest != "") {
			if len(digest) != sha256.Size*2 {
				return db.ErrInvalidInput
			}
			preview, err := previewIngestActivation(ctx, tx, profile)
			if err != nil {
				return err
			}
			if preview.Digest != digest {
				return db.ErrConflict
			}
			if current.Active {
				return db.ErrConflict
			}
			result, err := tx.ExecContext(ctx, `DELETE FROM ingest_events WHERE `+shadowPurgeScope, profile)
			if err != nil {
				return err
			}
			n, err := result.RowsAffected()
			if err != nil {
				return err
			}
			if n != preview.Receipts {
				return db.ErrConflict
			}
		}
		_, err = tx.ExecContext(ctx, `INSERT INTO profile_ingest_flags(profile_id,active,review) VALUES(?,?,?)
			ON CONFLICT(profile_id) DO UPDATE SET active=excluded.active,review=excluded.review`, profile, active, review)
		return err
	})
}

// checkIngestRoot validates a live root session inside the caller's SQLite transaction.
func checkIngestRoot(ctx context.Context, tx ingestActivationReader, session uuid.UUID) error {
	var root bool
	err := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM sessions s JOIN users u ON u.id=s.user_id
		WHERE s.id=? AND s.expires_at>unixepoch() AND u.role='admin' AND u.status='active')`, session.String()).Scan(&root)
	if err == nil && !root {
		return db.ErrForbidden
	}
	return err
}

// ingestActivationFlags checks that the target profile still exists and reads its current switches.
func ingestActivationFlags(ctx context.Context, tx ingestActivationReader, profile int32) (p db.IngestActivationPreview, err error) {
	p.ProfileID = profile
	p.Groups = []db.IngestShadowGroup{}
	err = tx.QueryRowContext(ctx, `SELECT coalesce(f.active,0),coalesce(f.review,0),coalesce(f.occurrence_write,0)
		FROM listening_profiles p LEFT JOIN profile_ingest_flags f ON f.profile_id=p.id WHERE p.id=?`, profile).Scan(&p.Active, &p.Review, &p.OccurrenceWrite)
	if errors.Is(err, sql.ErrNoRows) {
		err = db.ErrNotFound
	}
	return p, err
}

// previewIngestActivation binds the digest to the profile, flags, receipts and their children.
func previewIngestActivation(ctx context.Context, tx ingestActivationReader, profile int32) (db.IngestActivationPreview, error) {
	p, err := ingestActivationFlags(ctx, tx, profile)
	if err != nil {
		return p, err
	}
	hash := sha256.New()
	fmt.Fprintf(hash, "ingest-activation-v1/%d/%t/%t\n", profile, p.Active, p.Review)
	if p.OccurrenceWrite {
		fmt.Fprintln(hash, "occurrence-write=1")
	}
	rows, err := tx.QueryContext(ctx, `WITH targets AS MATERIALIZED (
		SELECT * FROM ingest_events WHERE `+shadowPurgeScope+`
	)
	SELECT receipt_id AS receipt, 0 AS kind, '' AS child, source_type, adapter_version, coalesce(rule_version,''), state,
		json_array(batch_id,item_ordinal,operation,source_instance_id,received_at,schema_version,snapshot_digest,digest_version,
			external_namespace,external_event_id,external_id_trusted,client_idempotency_key,batch_digest,file_digest,file_format,format_version,
			listened_at,original_time,time_precision,time_semantics,original_offset,uncertainty_ms,client_name,client_version,
			original_snapshot,normalized_snapshot,normalizer_version,reason_codes,duplicate_decision,evidence,identity_status,
			resolution_version,resolved_at,occurrence_id,recording_id,attempt_count,next_attempt_at,lease_owner,lease_until,work_version,closed_at),
		length(CAST(original_snapshot AS BLOB))
	FROM targets
	UNION ALL
	SELECT t.receipt_id, 1, o.id, '', '', '', '',
		json_array(o.schema_version,o.snapshot_digest,o.digest_version,o.original_snapshot,o.received_at,o.closed_at,o.protected),
		length(CAST(o.original_snapshot AS BLOB))
	FROM targets t JOIN ingest_observations o ON o.profile_id=t.profile_id AND o.receipt_id=t.receipt_id
	UNION ALL
	SELECT t.receipt_id, 2, CAST(d.id AS TEXT), '', '', '', '',
		json_array(d.resolution_version,d.kind,d.payload,d.closed_at,d.protected), 0
	FROM targets t JOIN ingest_event_details d ON d.profile_id=t.profile_id AND d.receipt_id=t.receipt_id
	ORDER BY receipt,kind,child`, profile)
	if err != nil {
		return p, err
	}
	defer rows.Close()

	type groupKey struct{ source, adapter, rule, state string }
	index := make(map[groupKey]int)
	var pos int
	for rows.Next() {
		var id, child, row string
		var k groupKey
		var kind int
		var size int64
		if err := rows.Scan(&id, &kind, &child, &k.source, &k.adapter, &k.rule, &k.state, &row, &size); err != nil {
			return p, err
		}
		fmt.Fprintf(hash, "%q/%d/%q/%q/%q/%q/%q/%q\n", id, kind, child, k.source, k.adapter, k.rule, k.state, row)
		switch kind {
		case 0:
			var found bool
			pos, found = index[k]
			if !found {
				pos = len(p.Groups)
				index[k] = pos
				p.Groups = append(p.Groups, db.IngestShadowGroup{Source: k.source, Adapter: k.adapter, Rule: k.rule, State: k.state})
			}
			p.Receipts++
			p.Groups[pos].Receipts++
		case 1:
			p.Groups[pos].Observations++
		case 2:
			p.Groups[pos].Details++
		}
		p.Groups[pos].SnapshotBytes += size
	}
	if err := rows.Err(); err != nil {
		return p, err
	}
	p.Digest = hex.EncodeToString(hash.Sum(nil))
	return p, nil
}
