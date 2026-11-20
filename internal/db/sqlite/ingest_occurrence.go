package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/ebrgr/TuneTrail/internal/ingest"
	"github.com/ebrgr/TuneTrail/internal/mbz"
	"github.com/google/uuid"
)

const occurrenceRuleVersion = "occurrence-v1"

type occurrenceProof struct {
	release, recording, track string
	conflict                  bool
}

type occurrenceMatch struct {
	id, recording, release   int64
	trackMBID, recordingMBID string
	disc, position           int
	text, validated          bool
}

// verifyOccurrence checks release-track membership before acquiring the SQLite writer.
func verifyOccurrence(ctx context.Context, snapshot ingest.Snapshot, client mbz.MusicBrainzCaller) occurrenceProof {
	releaseID := uuidFromSnapshot(snapshot.ReleaseMBID, snapshot.MappedReleaseMBID)
	if client == nil || releaseID == uuid.Nil || snapshot.Release == "" {
		return occurrenceProof{}
	}
	getRelease := client.GetRelease
	if lookup, ok := client.(interface {
		GetReleaseTracks(context.Context, uuid.UUID) (*mbz.MusicBrainzRelease, error)
	}); ok {
		getRelease = lookup.GetReleaseTracks
	}
	release, err := getRelease(ctx, releaseID)
	if err != nil || release == nil {
		return occurrenceProof{}
	}
	if release.ID != releaseID.String() || release.Title != snapshot.Release {
		return occurrenceProof{conflict: true}
	}
	proof := occurrenceProof{release: release.ID}
	var matches []mbz.MusicBrainzReleaseTrack
	for _, medium := range release.Media {
		for _, track := range medium.Tracks {
			if snapshot.DiscNumber > 0 && snapshot.DiscNumber != medium.Position ||
				snapshot.TrackNumber > 0 && snapshot.TrackNumber != track.Position ||
				snapshot.TrackMBID != "" && snapshot.TrackMBID != track.ID {
				continue
			}
			if track.Title == snapshot.Title {
				matches = append(matches, track)
			}
		}
	}
	if len(matches) == 1 {
		track := matches[0]
		declared := uuidFromSnapshot(snapshot.Recording, snapshot.MappedRecording)
		if declared != uuid.Nil && declared.String() != track.Recording.ID {
			return occurrenceProof{conflict: true}
		}
		if uuidFromSnapshot(track.Recording.ID) != uuid.Nil {
			proof.recording = track.Recording.ID
		}
		if uuidFromSnapshot(track.ID) != uuid.Nil {
			proof.track = track.ID
		}
	}
	return proof
}

// occurrenceOrigin maps the persisted adapter, never the supplied client name.
func occurrenceOrigin(r claimedReceipt) string {
	if r.source == ingest.ShadowSourceListenBrainz && r.format == "" {
		return "listenbrainz_api"
	}
	if r.source != ingest.ShadowSourceFile {
		return ""
	}
	switch r.format {
	case "lastfm", "spotify", "maloja", "listenbrainz":
		return r.format + "_import"
	case "koito":
		return "native_import"
	default:
		return ""
	}
}

// finishOccurrence revalidates ownership before making any catalog writes.
func (s *Sqlite) finishOccurrence(ctx context.Context, r claimedReceipt, proof occurrenceProof) error {
	return s.withIngestWriter(ctx, func(tx *sql.Conn) error {
		var precision, admissionReason string
		err := tx.QueryRowContext(ctx, `SELECT e.time_precision,coalesce(json_extract(e.reason_codes,'$[0]'),'')
			FROM ingest_events e JOIN listening_profiles p ON p.id=e.profile_id
			JOIN profile_ingest_flags f ON f.profile_id=p.id AND f.occurrence_write=1
			WHERE e.receipt_id=? AND e.profile_id=? AND e.mode='active' AND e.resolver_variant='occurrence'
			  AND e.closed_at IS NULL AND e.state IN ('received','retryable') AND e.resolution_version=?
			  AND e.work_version=? AND e.lease_owner=? AND e.lease_until>?`,
			r.id, r.profile, r.version, r.work, r.lease, time.Now().Unix()).Scan(&precision, &admissionReason)
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		if err != nil {
			return err
		}
		origin := occurrenceOrigin(r)
		if proof.conflict {
			return writeOccurrenceDecision(ctx, tx, r, occurrenceMatch{}, "pending_identity", "contradictory_musicbrainz_identity", precision, nil, "")
		}
		if r.snapshot.Release == "" && !s.allowStandaloneOccurrences {
			return writeOccurrenceDecision(ctx, tx, r, occurrenceMatch{}, "pending_identity", "release_missing", precision, nil, "")
		}
		if origin == "" || strings.TrimSpace(r.snapshot.Title) == "" || len(observedCredits(r.snapshot)) == 0 || r.snapshot.DiscNumber < 0 || r.snapshot.TrackNumber < 0 {
			return writeOccurrenceDecision(ctx, tx, r, occurrenceMatch{}, "pending_identity", "unresolved_identity", precision, nil, "")
		}
		if admissionReason == "client_batch_conflict" {
			return writeOccurrenceDecision(ctx, tx, r, occurrenceMatch{}, "pending_duplicate", admissionReason, precision, nil, "")
		}
		match, conflicts, err := resolveOccurrence(ctx, tx, r.snapshot, proof)
		if err != nil {
			return err
		}
		if len(conflicts) > 0 {
			return writeOccurrenceDecision(ctx, tx, r, occurrenceMatch{}, "pending_identity", "ambiguous_occurrence", precision, conflicts, "")
		}
		candidates, reason, err := occurrenceListenCandidates(ctx, tx, r, match.id)
		if err != nil {
			return err
		}
		if len(candidates) > 0 {
			return writeOccurrenceDecision(ctx, tx, r, match, "pending_duplicate", reason, precision, candidates, "")
		}
		if match.id == 0 {
			conflicts, err := occurrenceCreditConflicts(ctx, tx, observedCredits(r.snapshot))
			if err != nil {
				return err
			}
			if len(conflicts) > 0 {
				return writeOccurrenceDecision(ctx, tx, r, occurrenceMatch{}, "pending_identity", "ambiguous_artist_credit", precision, conflicts, "")
			}
			match, err = createOccurrence(ctx, tx, r.snapshot, proof, match)
			if err != nil {
				return err
			}
		} else if proof.recording != "" {
			if _, err := tx.ExecContext(ctx, `UPDATE recordings SET mbid_source='validated' WHERE id=? AND musicbrainz_id=?`, match.recording, proof.recording); err != nil {
				return err
			}
		}
		event := uuid.NewString()
		if _, err := tx.ExecContext(ctx, `INSERT INTO listens(profile_id,track_id,listened_at,client,event_id,origin) VALUES(?,?,?,?,?,?)`,
			r.profile, match.id, r.unix, r.client, event, origin); err != nil {
			return fmt.Errorf("write occurrence listen: %w", err)
		}
		// Converted profiles cannot have relay configuration, so no intent is eligible.
		return writeOccurrenceDecision(ctx, tx, r, match, "accepted", "distinct_occurrence_event", precision, nil, event)
	})
}

// writeOccurrenceDecision persists the final state and its revision in the same transaction.
func writeOccurrenceDecision(ctx context.Context, tx *sql.Conn, r claimedReceipt, match occurrenceMatch, state, reason, precision string, candidates []string, event string) error {
	status, decision := "pending", "review"
	var closed any
	now := time.Now().Unix()
	if state == "accepted" {
		status, decision, closed = "resolved", "different", now
	} else if state == "pending_duplicate" && match.id != 0 {
		status = "resolved"
	}
	evidence, err := json.Marshal(struct {
		Candidates []string `json:"candidates,omitempty"`
		Precision  string   `json:"precision"`
		Truncated  bool     `json:"truncated,omitempty"`
	}{candidates[:min(len(candidates), 32)], precision, len(candidates) > 32})
	if err != nil {
		return err
	}
	reasons := `["` + reason + `"]`
	result, err := tx.ExecContext(ctx, `UPDATE ingest_events SET state=?,identity_status=?,duplicate_decision=?,reason_codes=?,evidence=?,
		rule_version=?,resolution_version=resolution_version+1,resolved_at=?,event_id=nullif(?,''),occurrence_id=nullif(?,0),recording_id=nullif(?,0),
		closed_at=?,next_attempt_at=NULL,lease_owner=NULL,lease_until=NULL
		WHERE receipt_id=? AND profile_id=? AND resolver_variant='occurrence' AND resolution_version=? AND work_version=? AND lease_owner=?`,
		state, status, decision, reasons, string(evidence), occurrenceRuleVersion, now, event, match.id, match.recording, closed, r.id, r.profile, r.version, r.work, r.lease)
	if err != nil {
		return err
	}
	changed, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if changed != 1 {
		return errors.New("occurrence claim changed")
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO ingest_revisions VALUES(?,?,?,'automatic',?,?,?,?,?)`,
		r.profile, r.id, r.version+1, decision, reasons, string(evidence), occurrenceRuleVersion, now)
	return err
}

// occurrenceListenCandidates uses the window only for selection and never collapses compatible deliveries.
func occurrenceListenCandidates(ctx context.Context, tx *sql.Conn, r claimedReceipt, occurrence int64) ([]string, string, error) {
	if occurrence == 0 {
		return nil, "", nil
	}
	rows, err := tx.QueryContext(ctx, `SELECT l.event_id,l.listened_at FROM listens l
		WHERE l.profile_id=? AND l.track_id=? AND l.listened_at BETWEEN ? AND ?
		AND (l.listened_at=? OR NOT EXISTS (
			SELECT 1 FROM ingest_events prior JOIN ingest_events incoming ON incoming.receipt_id=?
			WHERE (prior.profile_id,prior.event_id)=(l.profile_id,l.event_id) AND prior.state='accepted'
			  AND prior.source_instance_id=incoming.source_instance_id AND (
			    (prior.external_id_trusted=1 AND incoming.external_id_trusted=1 AND prior.external_namespace=incoming.external_namespace
			      AND prior.external_event_id<>incoming.external_event_id)
			    OR (prior.file_digest=incoming.file_digest AND prior.file_format=incoming.file_format
			      AND prior.format_version=incoming.format_version AND prior.item_ordinal<>incoming.item_ordinal))))
		ORDER BY l.listened_at,l.event_id`, r.profile, occurrence, r.unix-2, r.unix+2, r.unix, r.id)
	if err != nil {
		return nil, "", err
	}
	defer rows.Close()
	var candidates []string
	reason := "ambiguous_time_candidate"
	for rows.Next() {
		var id string
		var stamp int64
		if err := rows.Scan(&id, &stamp); err != nil {
			return nil, "", err
		}
		if stamp == r.unix {
			reason = "legacy_identity_collision"
		}
		if len(candidates) < 33 {
			candidates = append(candidates, id)
		}
	}
	return candidates, reason, rows.Err()
}

// observedCredits preserves the source's ordered list and removes duplicate names.
func observedCredits(s ingest.Snapshot) []string {
	names := s.ArtistNames
	if len(names) == 0 {
		names = []string{s.Artist}
	}
	var credits []string
	for _, name := range names {
		if strings.TrimSpace(name) != "" && !slices.Contains(credits, name) {
			credits = append(credits, name)
		}
	}
	return credits
}
