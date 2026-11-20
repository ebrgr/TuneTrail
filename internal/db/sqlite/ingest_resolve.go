package sqlite

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"strconv"
	"time"

	"github.com/ebrgr/TuneTrail/internal/catalog"
	"github.com/ebrgr/TuneTrail/internal/cfg"
	"github.com/ebrgr/TuneTrail/internal/ingest"
	"github.com/ebrgr/TuneTrail/internal/mbz"
	"github.com/google/uuid"
)

const activeRuleVersion = "active-v1"

var errIngestTrackMissing = errors.New("ingest track no longer exists")

type claimedReceipt struct {
	id, lease, client, source, variant, format string
	profile, track                             int32
	version, work, unix                        int64
	snapshot                                   ingest.Snapshot
}

// ResolveNextIngest claims one active receipt independently of the admission flags.
func (s *Sqlite) ResolveNextIngest(ctx context.Context, mbzc mbz.MusicBrainzCaller) (bool, error) {
	receipt, snapshot, instant, err := s.claimNextIngest(ctx)
	if err != nil || receipt.id == "" {
		return false, err
	}
	at, err := time.Parse(receiptTime, instant)
	if err != nil {
		return true, s.retryIngest(ctx, receipt, err)
	}
	receipt.unix = at.Unix()
	if err := json.Unmarshal([]byte(snapshot), &receipt.snapshot); err != nil {
		return true, s.retryIngest(ctx, receipt, err)
	}
	receipt.client = receipt.snapshot.Client
	if receipt.variant == "occurrence" {
		proof := verifyOccurrence(ctx, receipt.snapshot, mbzc)
		if err := s.finishOccurrence(ctx, receipt, proof); err != nil {
			return true, s.retryIngest(ctx, receipt, err)
		}
		return true, nil
	}
	// Only the native file adapter can pin a locally associated track.
	if receipt.source == ingest.ShadowSourceFile {
		receipt.track = receipt.snapshot.CatalogTrackID
	}
	if receipt.track != 0 {
		err = s.finishIngest(ctx, receipt)
		if !errors.Is(err, errIngestTrackMissing) {
			if err != nil {
				return true, s.retryIngest(ctx, receipt, err)
			}
			return true, nil
		}
	}
	// A native track can disappear after admission. Reassociate from the durable snapshot.
	receipt.track, err = catalog.ResolveListenTrack(ctx, s, ingestCatalogOptions(receipt.snapshot, receipt.source, mbzc))
	if err == nil {
		err = s.finishIngest(ctx, receipt)
	}
	if err != nil {
		return true, s.retryIngest(ctx, receipt, err)
	}
	return true, nil
}

// claimNextIngest reads a due receipt and atomically reserves its lease.
func (s *Sqlite) claimNextIngest(ctx context.Context) (claimedReceipt, string, string, error) {
	now := time.Now().Unix()
	receipt := claimedReceipt{lease: uuid.NewString()}
	var snapshot, instant string
	const nextReceipt = `SELECT receipt_id FROM ingest_events WHERE mode='active' AND closed_at IS NULL
		AND state IN ('received','retryable') AND coalesce(next_attempt_at,0)<=? AND coalesce(lease_until,0)<=?
		ORDER BY received_at,receipt_id LIMIT 1`
	// An idle worker does not take the writer lock. The claim still rechecks the lease.
	var next string
	err := s.db.QueryRowContext(ctx, nextReceipt, now, now).Scan(&next)
	if errors.Is(err, sql.ErrNoRows) {
		return claimedReceipt{}, "", "", nil
	}
	if err != nil {
		return claimedReceipt{}, "", "", err
	}
	err = s.db.QueryRowContext(ctx, `UPDATE ingest_events SET lease_owner=?,lease_until=?,work_version=work_version+1,attempt_count=attempt_count+1
		WHERE receipt_id=? AND mode='active' AND closed_at IS NULL AND state IN ('received','retryable')
		AND coalesce(next_attempt_at,0)<=? AND coalesce(lease_until,0)<=?
		RETURNING receipt_id,profile_id,resolution_version,work_version,listened_at,original_snapshot,source_type,resolver_variant,coalesce(file_format,'')`,
		receipt.lease, now+120, next, now, now).Scan(&receipt.id, &receipt.profile, &receipt.version, &receipt.work, &instant, &snapshot, &receipt.source, &receipt.variant, &receipt.format)
	if errors.Is(err, sql.ErrNoRows) {
		return claimedReceipt{}, "", "", nil
	}
	if err != nil {
		return claimedReceipt{}, "", "", err
	}
	return receipt, snapshot, instant, nil
}

// ingestCatalogOptions maps only the durable allowlisted snapshot to catalog inputs.
func ingestCatalogOptions(s ingest.Snapshot, source string, mbzc mbz.MusicBrainzCaller) catalog.SubmitListenOpts {
	opts := catalog.SubmitListenOpts{
		Artist: s.Artist, ArtistNames: s.ArtistNames, TrackTitle: s.Title, ReleaseTitle: s.Release,
		Duration: int32(s.DurationMS / 1000), MbzCaller: mbzc,
		RecordingMbzID:    uuidFromSnapshot(s.Recording, s.MappedRecording),
		ReleaseMbzID:      uuidFromSnapshot(s.ReleaseMBID, s.MappedReleaseMBID),
		ReleaseGroupMbzID: uuidFromSnapshot(s.ReleaseGroupMBID),
		SkipCacheImage:    source == ingest.ShadowSourceFile && !cfg.FetchImagesDuringImport(),
	}
	for _, raw := range s.ArtistMBIDs {
		if id, err := uuid.Parse(raw); err == nil {
			opts.ArtistMbzIDs = append(opts.ArtistMbzIDs, id)
		}
	}
	for _, credit := range s.ArtistMbidMappings {
		if id, err := uuid.Parse(credit.MBID); err == nil {
			opts.ArtistMbidMappings = append(opts.ArtistMbidMappings, catalog.ArtistMbidMap{Artist: credit.Name, Mbid: id})
		}
	}
	return opts
}

// uuidFromSnapshot uses the first valid identifier, with declared values first.
func uuidFromSnapshot(values ...string) uuid.UUID {
	for _, raw := range values {
		if id, err := uuid.Parse(raw); err == nil {
			return id
		}
	}
	return uuid.Nil
}

// retryIngest releases this claim while preserving any newer claim or decision.
func (s *Sqlite) retryIngest(ctx context.Context, r claimedReceipt, cause error) error {
	_, err := s.db.ExecContext(ctx, `UPDATE ingest_events SET state='retryable',next_attempt_at=?,lease_owner=NULL,lease_until=NULL
		WHERE receipt_id=? AND mode='active' AND resolution_version=? AND work_version=? AND lease_owner=?`,
		time.Now().Add(time.Minute).Unix(), r.id, r.version, r.work, r.lease)
	return errors.Join(cause, err)
}

// finishIngest rechecks the claim and legacy identity under the same writer reservation as the decision.
func (s *Sqlite) finishIngest(ctx context.Context, r claimedReceipt) error {
	return s.withIngestWriter(ctx, func(tx *sql.Conn) error {
		var recording sql.NullInt64
		var review bool
		var precision, admissionReason string
		err := tx.QueryRowContext(ctx, `SELECT o.recording_id,coalesce(f.review,0),e.time_precision,coalesce(json_extract(e.reason_codes,'$[0]'),'')
			FROM ingest_events e LEFT JOIN occurrences o ON o.id=? AND o.legacy_track_id=o.id
			LEFT JOIN profile_ingest_flags f ON f.profile_id=e.profile_id
			WHERE e.receipt_id=? AND e.profile_id=? AND e.mode='active' AND e.closed_at IS NULL
			AND e.resolution_version=? AND e.work_version=? AND e.lease_owner=? AND e.lease_until>?`,
			r.track, r.id, r.profile, r.version, r.work, r.lease, time.Now().Unix()).Scan(&recording, &review, &precision, &admissionReason)
		if err != nil {
			return err
		}
		if !recording.Valid {
			return errIngestTrackMissing
		}
		var existing, candidate string
		err = tx.QueryRowContext(ctx, `SELECT event_id FROM listens WHERE profile_id=? AND track_id=? AND listened_at=?`,
			r.profile, r.track, r.unix).Scan(&existing)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return err
		}
		candidate = existing
		if candidate == "" {
			err = tx.QueryRowContext(ctx, `SELECT l.event_id FROM listens l JOIN occurrences o ON o.id=l.track_id
				WHERE l.profile_id=? AND l.listened_at BETWEEN ? AND ?
				AND (l.track_id=? OR o.recording_id=?)
				AND NOT EXISTS (
					SELECT 1 FROM ingest_events prior JOIN ingest_events incoming ON incoming.receipt_id=?
					WHERE prior.profile_id=l.profile_id AND prior.event_id=l.event_id AND prior.mode='active'
					AND prior.state='accepted' AND prior.source_instance_id=incoming.source_instance_id
					AND (
						(prior.external_id_trusted=1 AND incoming.external_id_trusted=1
						AND prior.external_namespace=incoming.external_namespace AND prior.external_event_id<>incoming.external_event_id)
						OR (prior.file_digest=incoming.file_digest AND prior.file_format=incoming.file_format
						AND prior.format_version=incoming.format_version AND prior.item_ordinal<>incoming.item_ordinal)
					)
				) ORDER BY l.listened_at,l.event_id LIMIT 1`,
				r.profile, r.unix-2, r.unix+2, r.track, recording.Int64, r.id).Scan(&candidate)
			if err != nil && !errors.Is(err, sql.ErrNoRows) {
				return err
			}
		}
		return finalizeActiveReceipt(ctx, tx, r, recording.Int64, precision, review, existing, candidate, admissionReason)
	})
}

// finalizeActiveReceipt writes the decision, revision, listen and eligible relay intent atomically.
func finalizeActiveReceipt(ctx context.Context, tx *sql.Conn, r claimedReceipt, recording int64, precision string, review bool, existing, candidate, admissionReason string) error {
	now := time.Now().Unix()
	state, decision, reason := "accepted", "different", "no_candidate"
	var event, closed any
	switch {
	case candidate != "" && review:
		state, decision, reason = "pending_duplicate", "review", "ambiguous_time_candidate"
		if existing != "" {
			reason = "legacy_identity_collision"
		}
	case existing != "":
		state, decision, reason = "duplicate", "same", "review_deferred"
		event, closed = existing, now
	default:
		if candidate != "" || admissionReason == "review_deferred" {
			reason = "review_deferred"
		}
		event, closed = uuid.NewString(), now
		if _, err := tx.ExecContext(ctx, `INSERT INTO listens(profile_id,track_id,listened_at,client,event_id) VALUES(?,?,?,?,?)`,
			r.profile, r.track, r.unix, r.client, event); err != nil {
			return err
		}
	}
	reasons, _ := json.Marshal([]string{reason})
	evidence, _ := json.Marshal(map[string]string{"candidate_event_id": candidate, "precision": precision})
	result, err := tx.ExecContext(ctx, `UPDATE ingest_events SET state=?,identity_status='resolved',duplicate_decision=?,reason_codes=?,evidence=?,
		rule_version=?,resolution_version=resolution_version+1,resolved_at=?,event_id=?,occurrence_id=?,recording_id=?,
		closed_at=?,next_attempt_at=NULL,lease_owner=NULL,lease_until=NULL
		WHERE receipt_id=? AND profile_id=? AND mode='active' AND resolution_version=? AND work_version=? AND lease_owner=?`,
		state, decision, string(reasons), string(evidence), activeRuleVersion, now, event, r.track, recording, closed,
		r.id, r.profile, r.version, r.work, r.lease)
	if err != nil {
		return err
	}
	changed, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if changed != 1 {
		return errors.New("ingest claim changed")
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO ingest_revisions VALUES(?,?,?,'automatic',?,?,?,?,?)`,
		r.profile, r.id, r.version+1, decision, string(reasons), string(evidence), activeRuleVersion, now); err != nil {
		return err
	}
	if state != "accepted" || r.source != ingest.ShadowSourceListenBrainz {
		return nil
	}
	destination := "relay:" + strconv.FormatInt(int64(r.profile), 10)
	if _, err := tx.ExecContext(ctx, `INSERT INTO integrations(id,profile_id,source_type,created_at)
		SELECT ?,profile_id,'listenbrainz-relay',? FROM profile_relays WHERE profile_id=? AND url<>'' AND token<>''
		ON CONFLICT(id) DO NOTHING`, destination, now, r.profile); err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO outbox(id,profile_id,destination_id,event_id,operation,delivery_key,state,created_at,closed_at,result)
		SELECT ?,profile_id,?,?,'listen',?,'discarded',?,?,'{"reason":"pre_082_best_effort"}'
		FROM profile_relays WHERE profile_id=? AND url<>'' AND token<>''`,
		uuid.NewString(), destination, event, event, now, now, r.profile)
	return err
}

// withIngestWriter takes the SQLite writer lock before any transactional reads.
func (s *Sqlite) withIngestWriter(ctx context.Context, write func(*sql.Conn) error) error {
	tx, err := s.db.Conn(ctx)
	if err != nil {
		return err
	}
	defer tx.Close()
	if _, err := tx.ExecContext(ctx, "BEGIN IMMEDIATE"); err != nil {
		return err
	}
	committed := false
	defer func() {
		if !committed {
			if _, err := tx.ExecContext(context.WithoutCancel(ctx), "ROLLBACK"); err != nil && !isAlreadyRolledBack(err) {
				_ = tx.Raw(func(any) error { return driver.ErrBadConn })
			}
		}
	}()
	if err := write(tx); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, "COMMIT"); err != nil {
		return &ingest.ErrUnavailable{Cause: err}
	}
	committed = true
	return nil
}
