package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/ebrgr/TuneTrail/internal/db"
)

const healthDeliveryJoin = ` LEFT JOIN outbox o ON o.profile_id=e.profile_id AND o.event_id=e.event_id
	AND o.destination_id='relay:'||e.profile_id AND o.operation='listen' AND o.format_version>0
	LEFT JOIN relay_destinations d ON d.profile_id=o.profile_id `

// healthDeliverySQL only references durable-relay columns on schemas that provide them.
func healthDeliverySQL(version int) (state, join string) {
	if version < 12 {
		return "'untracked'", ""
	}
	return `coalesce(` + relayProjectedState + `,
		CASE WHEN e.delivery_policy='durable' THEN 'not_queued' ELSE 'untracked' END)`, healthDeliveryJoin
}

// IngestHealth reads only typed diagnostics for a single, previously authorized profile.
func (s *Sqlite) IngestHealth(ctx context.Context, profile int32) (db.IngestHealth, error) {
	var health db.IngestHealth
	version, err := ingestSchemaVersion(ctx, s.db)
	if err != nil {
		return health, err
	}
	health.Sources, err = s.healthSources(ctx, profile, version)
	if err != nil {
		return health, err
	}
	health.Decisions, err = s.healthDecisions(ctx, profile)
	if err != nil || version < 12 {
		return health, err
	}
	health.RelayTracked = true
	health.Relay, err = s.healthDeliveries(ctx, profile)
	if err != nil {
		return health, err
	}
	count, until, pause, err := s.RelayQueueStatus(ctx, profile)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return health, err
	}
	health.Open = count
	if !until.Valid || until.Time.After(time.Now()) {
		health.Pause = pause
	}
	if version >= 13 {
		health.Failures, health.FailureCount, err = s.healthFailures(ctx, profile)
	}
	return health, err
}

// healthFailures reads the queue count and the most recent receipts from one snapshot.
func (s *Sqlite) healthFailures(ctx context.Context, profile int32) ([]db.IngestFailure, int, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT receipt_id,coalesce(file_format,source_type),park_reason,attempt_count,parked_at,
		(SELECT count(*) FROM ingest_events WHERE profile_id=? AND parked_at IS NOT NULL)
		FROM ingest_events WHERE profile_id=? AND parked_at IS NOT NULL
		ORDER BY parked_at DESC,receipt_id DESC LIMIT 25`, profile, profile)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var failures []db.IngestFailure
	var count int
	for rows.Next() {
		var item db.IngestFailure
		var parked int64
		if err := rows.Scan(&item.Receipt, &item.Source, &item.Reason, &item.Attempts, &parked, &count); err != nil {
			return nil, 0, err
		}
		item.Parked = time.Unix(parked, 0).UTC()
		failures = append(failures, item)
	}
	return failures, count, rows.Err()
}

// healthSources selects the latest receipt per integration, adapter format and declared client.
// The original snapshot stays in SQLite; only its scalar client field can be used as a fallback.
func (s *Sqlite) healthSources(ctx context.Context, profile int32, version int) ([]db.IngestSource, error) {
	delivery, join := healthDeliverySQL(version)
	policy := "'legacy' AS delivery_policy"
	if version >= 12 {
		policy = "e.delivery_policy"
	}
	rows, err := s.db.QueryContext(ctx, `WITH sources AS (
		SELECT e.profile_id,e.receipt_id,e.source_instance_id,e.received_at,e.time_precision,e.event_id,`+policy+`,
			substr(coalesce(nullif(e.client_name,''),
			CASE WHEN json_valid(e.original_snapshot) THEN
				CASE WHEN json_type(e.original_snapshot,'$.client')='text' THEN json_extract(e.original_snapshot,'$.client') END
			END,''),1,120) AS client,
			coalesce(e.file_format,e.source_type) AS source
		FROM ingest_events e WHERE e.profile_id=?
	), latest AS (
		SELECT *, row_number() OVER (PARTITION BY source_instance_id, source, client ORDER BY received_at DESC, receipt_id DESC) AS rank
		FROM sources
	)
	SELECT coalesce(nullif(i.display_name,''),e.source_instance_id), e.source, e.client, e.received_at, e.time_precision,
		`+delivery+`
	FROM latest e JOIN integrations i ON i.profile_id=e.profile_id AND i.id=e.source_instance_id
	`+join+` WHERE e.rank=1 ORDER BY e.received_at DESC,e.receipt_id DESC LIMIT 100`, profile)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var sources []db.IngestSource
	for rows.Next() {
		var source db.IngestSource
		var received int64
		if err := rows.Scan(&source.Integration, &source.Source, &source.Client, &received, &source.Precision, &source.Delivery); err != nil {
			return nil, err
		}
		source.Received = time.Unix(received, 0).UTC()
		sources = append(sources, source)
	}
	return sources, rows.Err()
}

// healthDecisions deliberately projects scalar evidence rather than exposing the evidence document.
func (s *Sqlite) healthDecisions(ctx context.Context, profile int32) ([]db.IngestDecision, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT e.mode,e.duplicate_decision,
		coalesce(json_extract(e.reason_codes,'$[0]'),''),coalesce(e.rule_version,''),e.time_precision,
		CASE WHEN json_type(e.evidence,'$.candidates')='array' THEN json_array_length(e.evidence,'$.candidates')
			WHEN json_type(e.evidence,'$.candidate_event_id')='text' THEN (json_extract(e.evidence,'$.candidate_event_id')<>'')
			WHEN json_type(e.evidence,'$.candidate_receipt_id')='text' THEN 1 ELSE '' END,
		CASE WHEN json_type(e.evidence,'$.delta_seconds')='integer' THEN json_extract(e.evidence,'$.delta_seconds') ELSE '' END,
		CASE json_type(e.evidence,'$.recording_mbid_match') WHEN 'true' THEN '1' WHEN 'false' THEN '0' ELSE '' END,
		CASE json_type(e.evidence,'$.musicbrainz.release_title_differs') WHEN 'true' THEN '1' WHEN 'false' THEN '0' ELSE '' END,
		e.external_id_trusted,coalesce(e.resolved_at,e.received_at)
		FROM ingest_events e WHERE e.profile_id=? AND e.duplicate_decision IS NOT NULL
		ORDER BY coalesce(e.resolved_at,e.received_at) DESC,e.receipt_id DESC LIMIT 25`, profile)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var decisions []db.IngestDecision
	for rows.Next() {
		var item db.IngestDecision
		var stamp int64
		if err := rows.Scan(&item.Mode, &item.Value, &item.Reason, &item.Rule, &item.Precision, &item.CandidateCount, &item.DeltaSeconds, &item.RecordingMatch, &item.ReleaseDiffers, &item.TrustedExternalID, &stamp); err != nil {
			return nil, err
		}
		item.At = time.Unix(stamp, 0).UTC()
		decisions = append(decisions, item)
	}
	return decisions, rows.Err()
}

// healthDeliveries prioritizes recent attempts, including exhausted and fenced work.
func (s *Sqlite) healthDeliveries(ctx context.Context, profile int32) ([]db.IngestDelivery, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT `+relayProjectedState+`,o.attempt_count,
		`+relayProjectedReason+`
		FROM outbox o JOIN relay_destinations d USING(profile_id)
		WHERE o.profile_id=? AND o.destination_id='relay:'||? AND o.format_version>0 AND o.state<>'sent'
		ORDER BY coalesce(o.last_attempt_at,o.closed_at,o.created_at) DESC,o.id DESC LIMIT 25`, profile, profile)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var deliveries []db.IngestDelivery
	for rows.Next() {
		var item db.IngestDelivery
		if err := rows.Scan(&item.State, &item.Attempts, &item.Reason); err != nil {
			return nil, err
		}
		deliveries = append(deliveries, item)
	}
	return deliveries, rows.Err()
}

// ListenProvenance uses the canonical accepted receipt and exact event link.
// Distinct events for the same track and second can each have their own explanation.
func (s *Sqlite) ListenProvenance(ctx context.Context, profile, track int32, at int64, eventIDs ...string) ([]db.ListenProvenance, error) {
	version, err := ingestSchemaVersion(ctx, s.db)
	if err != nil {
		return nil, err
	}
	delivery, join := healthDeliverySQL(version)
	var event string
	if len(eventIDs) > 0 {
		event = eventIDs[0]
	}
	archiveJoin := `LEFT JOIN archive_events a ON (a.profile_id,a.event_id)=(l.profile_id,l.event_id)`
	if version < 14 {
		archiveJoin = `LEFT JOIN (SELECT NULL AS origin,NULL AS provenance) a ON 0`
	}
	// Archived receipts are historical evidence and cannot enter the live processing queue.
	// The v9 writer keeps the legacy origin enum even when a receipt identifies the source.
	rows, err := s.db.QueryContext(ctx, `SELECT
		CASE WHEN l.origin='legacy' AND e.receipt_id IS NOT NULL THEN coalesce(e.file_format,e.source_type) ELSE coalesce(a.origin,l.origin) END,
		coalesce(e.received_at,CAST(json_extract(a.provenance,'$[0].received_at') AS INTEGER)),coalesce(e.time_precision,json_extract(a.provenance,'$[0].time_precision'),''),
		`+delivery+`,coalesce(json_extract(e.reason_codes,'$[0]'),json_extract(a.provenance,'$[0].reason_codes[0]'),''),coalesce(e.rule_version,json_extract(a.provenance,'$[0].rule_version'),'')
		FROM listens l LEFT JOIN ingest_events e ON e.profile_id=l.profile_id AND e.event_id=l.event_id AND e.state='accepted'
		`+join+`
		`+archiveJoin+`
		WHERE l.profile_id=? AND l.track_id=? AND l.listened_at=? AND (?='' OR l.event_id=?) ORDER BY l.event_id`, profile, track, at, event, event)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var provenance []db.ListenProvenance
	for rows.Next() {
		var item db.ListenProvenance
		var received sql.NullInt64
		if err := rows.Scan(&item.Origin, &received, &item.Precision, &item.Delivery, &item.Reason, &item.Rule); err != nil {
			return nil, err
		}
		if received.Valid {
			at := time.Unix(received.Int64, 0).UTC()
			item.Received = &at
		}
		provenance = append(provenance, item)
	}
	return provenance, rows.Err()
}
