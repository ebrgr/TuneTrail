package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/ebrgr/TuneTrail/internal/dedup"
	"github.com/ebrgr/TuneTrail/internal/ingest"
	"github.com/google/uuid"
	sqlitedriver "modernc.org/sqlite"
	sqlitecodes "modernc.org/sqlite/lib"
)

const receiptTime = "2006-01-02T15:04:05.000000000Z"

type preparedReceipt struct {
	ingest.Item
	snapshot, digest, legacyDigest string
}

// ReceiveBatch retains the legacy shadow writer for callers without active admission.
func (s *Sqlite) ReceiveBatch(ctx context.Context, batch ingest.Batch) ([]ingest.Result, error) {
	_, results, err := s.AdmitIngest(ctx, batch, false, nil, nil)
	return results, err
}

// BeginImportIngest selects the mode once for the entire import job.
func (s *Sqlite) BeginImportIngest(ctx context.Context, profile int32, master bool) (active bool, err error) {
	err = s.withIngestWriter(ctx, func(tx *sql.Conn) error {
		version, err := ingestSchemaVersion(ctx, tx)
		// Active import admission requires the occurrence-ready schema.
		if err != nil || version < 10 {
			return err
		}
		if version >= 11 {
			converted, err := occurrenceWriteFlag(ctx, tx, profile)
			if err != nil {
				return err
			}
			// File jobs still use the v9 adapter until the occurrence adapter lands.
			if converted {
				return &ingest.ErrUnavailable{Cause: errOccurrenceAdmissionPaused}
			}
		}
		if master {
			active, err = activeIngestMode(ctx, tx, profile)
		}
		return err
	})
	return active, ingestStorageError(err, true)
}

// ingestSchemaVersion checks whether the mode column exists before choosing receipt SQL.
func ingestSchemaVersion(ctx context.Context, tx *sql.Conn) (int, error) {
	var version int
	err := tx.QueryRowContext(ctx, `SELECT max(version_id) FROM goose_db_version WHERE is_applied=1`).Scan(&version)
	return version, err
}

// activeIngestMode runs inside the admission writer transaction on schema v9 or later.
func activeIngestMode(ctx context.Context, tx *sql.Conn, profile int32) (bool, error) {
	var active bool
	err := tx.QueryRowContext(ctx, `SELECT coalesce((SELECT active FROM profile_ingest_flags WHERE profile_id=?),0)`, profile).Scan(&active)
	return active, err
}

var errOccurrenceAdmissionPaused = errors.New("occurrence admission is paused")

// occurrenceWriteFlag reads the irreversible cut under the writer lock.
func occurrenceWriteFlag(ctx context.Context, tx *sql.Conn, profile int32) (bool, error) {
	var converted bool
	err := tx.QueryRowContext(ctx, `SELECT coalesce((SELECT occurrence_write FROM profile_ingest_flags WHERE profile_id=?),0)`, profile).Scan(&converted)
	return converted, err
}

// AdmitIngest fixes the mode under the writer lock, honoring the import job's pinned mode.
func (s *Sqlite) AdmitIngest(ctx context.Context, batch ingest.Batch, master bool, pinned *bool, invalid error) (active bool, results []ingest.Result, err error) {
	if len(batch.Items) == 0 && !master && (pinned == nil || !*pinned) {
		return false, nil, nil
	}
	source := receiptSourcePrefix(batch.Source) + fmt.Sprint(batch.ProfileID)
	batchID := uuid.NewString()
	prepared := make([]preparedReceipt, len(batch.Items))
	digests := make([]string, len(batch.Items))
	legacyDigests := make([]string, len(batch.Items))
	for n, item := range batch.Items {
		snapshot, digest := item.Encode()
		prepared[n] = preparedReceipt{Item: item, snapshot: string(snapshot), digest: digest, legacyDigest: item.LegacyDigest(batch.Source, batch.FileFormat)}
		digests[n] = digest
		legacyDigests[n] = prepared[n].legacyDigest
	}
	var batchDigest, legacyBatchDigest string
	if batch.ClientKey != "" {
		data, _ := json.Marshal(digests)
		batchDigest = ingest.Digest(data)
		legacy, _ := json.Marshal(legacyDigests)
		legacyBatchDigest = ingest.Digest(legacy)
	}
	modeKnown := !master || pinned != nil
	err = s.withIngestWriter(ctx, func(tx *sql.Conn) error {
		version, err := ingestSchemaVersion(ctx, tx)
		if err != nil {
			return err
		}
		variant := "v9"
		if version >= 11 {
			converted, err := occurrenceWriteFlag(ctx, tx, batch.ProfileID)
			if err != nil {
				return err
			}
			if converted {
				active = true
				if !master || pinned != nil {
					return &ingest.ErrUnavailable{Cause: errOccurrenceAdmissionPaused}
				}
				variant = "occurrence"
			}
		}
		hasMode := version >= 9
		// Older layouts can still receive historical shadow receipts.
		if pinned != nil {
			active = *pinned && version >= 10
		} else if master && version >= 10 {
			var err error
			active, err = activeIngestMode(ctx, tx, batch.ProfileID)
			if err != nil {
				return err
			}
		}
		modeKnown = true
		if active && invalid != nil {
			return invalid
		}
		if len(prepared) == 0 {
			return nil
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO integrations (id, profile_id, source_type, created_at)
			VALUES (?, ?, ?, ?) ON CONFLICT(id) DO NOTHING`, source, batch.ProfileID, batch.Source, batch.ReceivedAt.Unix()); err != nil {
			return err
		}
		var previousBatchDigest, previousBatchVersion string
		if batch.ClientKey != "" {
			err := tx.QueryRowContext(ctx, `SELECT batch_digest,digest_version FROM ingest_events
				WHERE profile_id=? AND source_instance_id=? AND operation=? AND client_idempotency_key=?
				ORDER BY item_ordinal LIMIT 1`, batch.ProfileID, source, batch.Operation, batch.ClientKey).Scan(&previousBatchDigest, &previousBatchVersion)
			if err != nil && !errors.Is(err, sql.ErrNoRows) {
				return err
			}
		}
		comparableBatchDigest := batchDigest
		if previousBatchVersion == ingest.LegacyDigestVersion {
			comparableBatchDigest = legacyBatchDigest
		}
		batchConflict := previousBatchDigest != "" && previousBatchDigest != comparableBatchDigest
		for _, item := range prepared {
			var result ingest.Result
			var err error
			if active {
				result, err = receiveActiveItem(ctx, tx, batch, item, source, batchID, batchDigest, legacyBatchDigest, batchConflict, variant)
			} else {
				result, err = receiveItem(ctx, tx, batch, item, source, batchID, batchDigest, legacyBatchDigest, batchConflict, hasMode)
			}
			if err != nil {
				return err
			}
			results = append(results, result)
		}
		return nil
	})
	if err != nil {
		return active || !modeKnown, nil, ingestStorageError(err, active || !modeKnown || (pinned != nil && *pinned))
	}
	return active, results, nil
}

// ingestStorageError preserves shadow fail-open on lock contention.
func ingestStorageError(err error, active bool) error {
	var storage *sqlitedriver.Error
	if errors.As(err, &storage) {
		switch storage.Code() & 255 {
		case sqlitecodes.SQLITE_FULL, sqlitecodes.SQLITE_IOERR, sqlitecodes.SQLITE_READONLY, sqlitecodes.SQLITE_CANTOPEN, sqlitecodes.SQLITE_CORRUPT, sqlitecodes.SQLITE_NOTADB:
			return &ingest.ErrUnavailable{Cause: err}
		case sqlitecodes.SQLITE_BUSY, sqlitecodes.SQLITE_LOCKED:
			if active {
				return &ingest.ErrUnavailable{Cause: err}
			}
		}
	}
	return err
}

// isOccurrenceGuard recognizes controlled SQLite trigger errors without matching other SQL failures.
func isOccurrenceGuard(err error, message string) bool {
	return matchesSQLiteError(err, sqlitecodes.SQLITE_CONSTRAINT_TRIGGER, message)
}

// matchesSQLiteError recognizes the named constraint without classifying unrelated failures.
func matchesSQLiteError(err error, code int, message string) bool {
	var sqliteErr *sqlitedriver.Error
	return errors.As(err, &sqliteErr) && sqliteErr.Code() == code && strings.Contains(sqliteErr.Error(), message)
}

// isAlreadyRolledBack recognizes automatic SQLite rollback after a storage error.
func isAlreadyRolledBack(err error) bool {
	var sqliteErr *sqlitedriver.Error
	return errors.As(err, &sqliteErr) && sqliteErr.Code() == sqlitecodes.SQLITE_ERROR &&
		strings.Contains(sqliteErr.Error(), "no transaction is active")
}

const receiptColumns = `receipt_id, source_instance_id, coalesce(external_namespace,''), coalesce(external_event_id,''),
	snapshot_digest, coalesce(batch_digest,''), time_precision, listened_at, original_snapshot, state, digest_version`

// scanReceipt reads the narrow projection used by historical shadow receipts.
func scanReceipt(row interface{ Scan(...any) error }) (dedup.Receipt, error) {
	return scanReceiptMode(row, false)
}

// scanReceiptMode adds key ownership and the stored verdict for schema v9.
func scanReceiptMode(row interface{ Scan(...any) error }, hasMode bool) (r dedup.Receipt, err error) {
	var instant sql.NullString
	var snapshot string
	r.Mode = "shadow"
	fields := []any{&r.ID, &r.Source, &r.Namespace, &r.ExternalID, &r.Digest, &r.BatchDigest, &r.Precision, &instant, &snapshot, &r.State, &r.DigestVersion}
	if hasMode {
		fields = append(fields, &r.Mode, &r.Decision)
	}
	if err = row.Scan(fields...); err != nil {
		return r, err
	}
	if instant.Valid {
		r.Time, err = time.Parse(receiptTime, instant.String)
	}
	if err == nil {
		err = json.Unmarshal([]byte(snapshot), &r.Snapshot)
	}
	return r, err
}

var errReceiptKeyConflict = errors.New("receipt keys resolve to different items")

// findKey checks each applicable unique index under the same writer transaction.
func findKey(ctx context.Context, tx *sql.Conn, b ingest.Batch, i ingest.Item, source string, hasMode bool) (*dedup.Receipt, error) {
	columns := receiptColumns
	if hasMode {
		columns += ",mode,coalesce(duplicate_decision,'')"
	}
	base := `SELECT ` + columns + ` FROM ingest_events WHERE profile_id=? AND source_instance_id=? AND `
	var found *dedup.Receipt
	find := func(predicate string, args ...any) error {
		r, err := scanReceiptMode(tx.QueryRowContext(ctx, base+predicate, append([]any{b.ProfileID, source}, args...)...), hasMode)
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		if err != nil {
			return err
		}
		if found != nil && found.ID != r.ID {
			// Prefer active ownership so a shadow retry never mutates another receipt.
			if r.Mode != found.Mode {
				if r.Mode == "active" {
					found = &r
				}
				return nil
			}
			return errReceiptKeyConflict
		}
		found = &r
		return nil
	}
	if i.ExternalID != "" {
		if err := find("external_id_trusted=1 AND external_namespace=? AND external_event_id=?", i.ExternalNamespace, i.ExternalID); err != nil {
			return nil, err
		}
	}
	if b.ClientKey != "" {
		if err := find("operation=? AND client_idempotency_key=? AND item_ordinal=?", b.Operation, b.ClientKey, i.Ordinal); err != nil {
			return nil, err
		}
	}
	if b.FileDigest != "" {
		if err := find("file_digest=? AND file_format=? AND format_version='1' AND item_ordinal=?", b.FileDigest, b.FileFormat, i.Ordinal); err != nil {
			return nil, err
		}
	}
	return found, nil
}

// receiveItem records the decision without setting any of the future identity columns.
func receiveItem(ctx context.Context, tx *sql.Conn, b ingest.Batch, i preparedReceipt, source, batchID, batchDigest, legacyBatchDigest string, batchConflict, hasMode bool) (ingest.Result, error) {
	r := dedup.Receipt{Source: source, Namespace: i.ExternalNamespace, ExternalID: i.ExternalID, Digest: i.digest,
		BatchDigest: batchDigest, Snapshot: i.Snapshot, Time: i.ListenedAt, Precision: "second"}
	if r.Time.IsZero() {
		r.Time, r.Precision = b.ReceivedAt, "estimated"
	}
	previous, err := findKey(ctx, tx, b, i.Item, source, hasMode)
	if err != nil {
		return ingest.Result{}, err
	}
	if previous != nil && previous.Mode == "active" {
		return replayIngest(previous), nil
	}
	if previous != nil && previous.DigestVersion == ingest.LegacyDigestVersion {
		r.Digest, r.BatchDigest = i.legacyDigest, legacyBatchDigest
	}
	decision := dedup.Compare(dedup.Shadow, r, previous, previous != nil)
	if previous == nil && batchConflict {
		// An extra ordinal under a reused batch key is still part of the conflict.
		decision.Value, decision.Reason = "review", "client_batch_conflict"
	} else if previous == nil {
		decision, err = compareNearby(ctx, tx, b.ProfileID, r, decision, hasMode)
		if err != nil {
			return ingest.Result{}, err
		}
	}
	reasons, _ := json.Marshal([]string{decision.Reason})
	evidence, _ := json.Marshal(decision.Evidence)
	state := "received"
	if decision.Value == "review" {
		state = "pending_duplicate"
	}
	id := uuid.NewString()
	if previous != nil {
		id = previous.ID
		if decision.Value == "same" {
			// A keyed retry returns the existing receipt without rewriting its verdict.
			return ingest.Result{ReceiptID: id, Decision: decision.Value, Pending: previous.State == "pending_duplicate"}, nil
		}
		observation, err := tx.ExecContext(ctx, `INSERT INTO ingest_observations(id,profile_id,receipt_id,schema_version,snapshot_digest,digest_version,original_snapshot,received_at)
				VALUES(?,?,?,1,?,?,?,?) ON CONFLICT(profile_id,receipt_id,digest_version,snapshot_digest) DO NOTHING`,
			uuid.NewString(), b.ProfileID, id, i.digest, ingest.DigestVersion, i.snapshot, b.ReceivedAt.Unix())
		if err != nil {
			return ingest.Result{}, err
		}
		added, err := observation.RowsAffected()
		if err != nil {
			return ingest.Result{}, err
		}
		// Only a new conflicting observation can update the stored review evidence.
		if added > 0 {
			_, err = tx.ExecContext(ctx, `UPDATE ingest_events SET duplicate_decision=?,reason_codes=?,evidence=?,rule_version=?,state=?
				WHERE profile_id=? AND receipt_id=?`+shadowModeClause(hasMode), decision.Value, string(reasons), string(evidence), decision.RuleVersion, state, b.ProfileID, id)
		}
		return ingest.Result{ReceiptID: id, Decision: decision.Value, Pending: true}, err
	} else {
		_, err = tx.ExecContext(ctx, `INSERT INTO ingest_events
			(receipt_id,profile_id,source_instance_id,schema_version,source_type,adapter_version,batch_id,item_ordinal,operation,received_at,
			external_namespace,external_event_id,external_id_trusted,client_idempotency_key,batch_digest,file_digest,file_format,format_version,
			listened_at,original_time,time_precision,time_semantics,original_snapshot,snapshot_digest,digest_version,state,duplicate_decision,reason_codes,evidence,rule_version)
			VALUES(?,?,?,1,?,'1',?,?,?,?,nullif(?,''),nullif(?,''),?,nullif(?,''),nullif(?,''),nullif(?,''),nullif(?,''),CASE WHEN ?!='' THEN '1' END,
			?,nullif(?,''),?,'unknown',?,?,?,?,?,?,?,?)`,
			id, b.ProfileID, source, b.Source, batchID, i.Ordinal, b.Operation, b.ReceivedAt.Unix(), i.ExternalNamespace, i.ExternalID, i.ExternalID != "",
			b.ClientKey, batchDigest, b.FileDigest, b.FileFormat, b.FileDigest, r.Time.UTC().Format(receiptTime), i.OriginalTime(), r.Precision, i.snapshot, i.digest, ingest.DigestVersion, state, decision.Value, string(reasons), string(evidence), decision.RuleVersion)
	}
	return ingest.Result{ReceiptID: id, Decision: decision.Value, Pending: state == "pending_duplicate"}, err
}

// replayIngest reports the stored outcome without updating a receipt or claim.
func replayIngest(r *dedup.Receipt) ingest.Result {
	return ingest.Result{ReceiptID: r.ID, Decision: r.Decision, Replay: true,
		Pending: r.State != "accepted" && r.State != "duplicate" && r.State != "invalid"}
}

// receiveActiveItem replaces unlinked shadow receipts and replays linked or active owners.
func receiveActiveItem(ctx context.Context, tx *sql.Conn, b ingest.Batch, i preparedReceipt, source, batchID, batchDigest, legacyBatchDigest string, batchConflict bool, variant string) (ingest.Result, error) {
	previous, err := findKey(ctx, tx, b, i.Item, source, true)
	if errors.Is(err, errReceiptKeyConflict) {
		return ingest.Result{Decision: "review", Pending: true}, nil
	}
	if err != nil {
		return ingest.Result{}, err
	}
	var shadowEstimate time.Time
	if previous != nil && previous.Mode == "shadow" {
		// Release the shared key under the admission writer lock. Foreign keys
		// remove shadow observations and details as part of the same transaction.
		result, err := tx.ExecContext(ctx, `DELETE FROM ingest_events WHERE profile_id=? AND receipt_id=?
			AND mode='shadow' AND event_id IS NULL`, b.ProfileID, previous.ID)
		if err != nil {
			return ingest.Result{}, err
		}
		deleted, err := result.RowsAffected()
		if err != nil {
			return ingest.Result{}, err
		}
		if deleted == 0 {
			return replayIngest(previous), nil
		}
		if previous.Precision == "estimated" {
			shadowEstimate = previous.Time
		}
		previous = nil
	}
	if previous != nil {
		result := replayIngest(previous)
		digest, comparableBatchDigest := i.digest, batchDigest
		if previous.DigestVersion == ingest.LegacyDigestVersion {
			digest, comparableBatchDigest = i.legacyDigest, legacyBatchDigest
		}
		if previous.Digest != digest || previous.BatchDigest != comparableBatchDigest {
			_, err := tx.ExecContext(ctx, `INSERT INTO ingest_observations(id,profile_id,receipt_id,schema_version,snapshot_digest,digest_version,original_snapshot,received_at)
				VALUES(?,?,?,1,?,?,?,?) ON CONFLICT(profile_id,receipt_id,digest_version,snapshot_digest) DO NOTHING`,
				uuid.NewString(), b.ProfileID, previous.ID, i.digest, ingest.DigestVersion, i.snapshot, b.ReceivedAt.Unix())
			return result, err
		}
		return result, nil
	}
	id := uuid.NewString()
	at, precision := i.ListenedAt, "second"
	if at.IsZero() {
		at, precision = b.ReceivedAt, "estimated"
		// Reuse the original estimate for a keyed retry without a timestamp.
		if !shadowEstimate.IsZero() {
			at = shadowEstimate
		}
	}
	state, reasons := "received", "[]"
	if batchConflict && variant == "occurrence" {
		reasons = `["client_batch_conflict"]`
	} else if batchConflict {
		var review bool
		if err := tx.QueryRowContext(ctx, `SELECT coalesce((SELECT review FROM profile_ingest_flags WHERE profile_id=?),0)`, b.ProfileID).Scan(&review); err != nil {
			return ingest.Result{}, err
		}
		reasons = `["review_deferred"]`
		if review {
			state, reasons = "pending_duplicate", `["client_batch_conflict"]`
		}
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO ingest_events
		(receipt_id,profile_id,source_instance_id,schema_version,source_type,adapter_version,batch_id,item_ordinal,operation,received_at,
		external_namespace,external_event_id,external_id_trusted,client_idempotency_key,batch_digest,file_digest,file_format,format_version,
		listened_at,original_time,time_precision,time_semantics,original_snapshot,snapshot_digest,digest_version,mode,state,reason_codes,rule_version,resolver_variant)
		VALUES(?,?,?,1,?,'1',?,?,?,?,nullif(?,''),nullif(?,''),?,nullif(?,''),nullif(?,''),nullif(?,''),nullif(?,''),CASE WHEN ?!='' THEN '1' END,
		?,nullif(?,''),?,'unknown',?,?,?,'active',?,?,?,?)`,
		id, b.ProfileID, source, b.Source, batchID, i.Ordinal, b.Operation, b.ReceivedAt.Unix(), i.ExternalNamespace, i.ExternalID, i.ExternalID != "",
		b.ClientKey, batchDigest, b.FileDigest, b.FileFormat, b.FileDigest, at.UTC().Format(receiptTime), i.OriginalTime(), precision,
		i.snapshot, i.digest, ingest.DigestVersion, state, reasons, activeRuleVersion, variant)
	return ingest.Result{ReceiptID: id, Pending: true}, err
}

// shadowModeClause selects only shadow rows once mode is present in the schema.
func shadowModeClause(hasMode bool) string {
	if hasMode {
		return " AND mode='shadow'"
	}
	return ""
}

// compareNearby keeps the shadow baseline limited to shadow receipts; active
// evidence comes from live events and must not change the shadow verdicts.
func compareNearby(ctx context.Context, tx *sql.Conn, profile int32, r dedup.Receipt, decision dedup.Decision, hasMode bool) (dedup.Decision, error) {
	rows, err := tx.QueryContext(ctx, `SELECT `+receiptColumns+` FROM ingest_events
		WHERE profile_id=? AND listened_at BETWEEN ? AND ? AND state!='invalid'`+shadowModeClause(hasMode)+`
		ORDER BY listened_at,receipt_id`, profile, r.Time.Add(-dedup.Shadow.Window).UTC().Format(receiptTime), r.Time.Add(dedup.Shadow.Window).UTC().Format(receiptTime))
	if err != nil {
		return decision, err
	}
	defer rows.Close()
	for rows.Next() {
		candidate, err := scanReceipt(rows)
		if err != nil {
			return decision, err
		}
		result := dedup.Compare(dedup.Shadow, r, &candidate, false)
		if result.Value == "review" {
			return result, nil
		}
	}
	return decision, rows.Err()
}

// ReadFileReceipts returns checkpoints in absolute ordinal order, independent of staging.
// Pass -1 to begin and the last returned ordinal to continue.
func (s *Sqlite) ReadFileReceipts(ctx context.Context, profile int32, format, digest string, after int) ([]ingest.FileReceipt, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT item_ordinal,receipt_id,original_snapshot,listened_at,time_precision,state
		FROM ingest_events WHERE profile_id=? AND source_instance_id=? AND file_format=? AND format_version='1'
		  AND file_digest=? AND item_ordinal>? ORDER BY item_ordinal LIMIT 500`,
		profile, receiptSourcePrefix(ingest.ShadowSourceFile)+fmt.Sprint(profile), format, digest, after)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []ingest.FileReceipt
	for rows.Next() {
		var record ingest.FileReceipt
		var snapshot, instant, state string
		if err := rows.Scan(&record.Ordinal, &record.ReceiptID, &snapshot, &instant, &record.Precision, &state); err != nil {
			return nil, err
		}
		if err := json.Unmarshal([]byte(snapshot), &record.Snapshot); err != nil {
			return nil, err
		}
		record.ListenedAt, err = time.Parse(receiptTime, instant)
		if err != nil {
			return nil, err
		}
		record.Pending = state == "pending_duplicate"
		result = append(result, record)
	}
	return result, rows.Err()
}
