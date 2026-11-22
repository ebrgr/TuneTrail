package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/ebrgr/TuneTrail/internal/ingest"
	"github.com/ebrgr/TuneTrail/internal/mbz"
	"github.com/google/uuid"
)

const occurrenceRuleVersion = "occurrence-v1"

type occurrenceReleaseCache struct {
	mu       sync.Mutex
	releases map[uuid.UUID]cachedOccurrenceRelease
}

type cachedOccurrenceRelease struct {
	expires time.Time
	release *mbz.MusicBrainzRelease
}

// occurrenceReleaseLookup shares successful release evidence across receipts.
func (c *occurrenceReleaseCache) occurrenceReleaseLookup(ctx context.Context, id uuid.UUID, lookup func(context.Context, uuid.UUID) (*mbz.MusicBrainzRelease, error)) (*mbz.MusicBrainzRelease, error) {
	c.mu.Lock()
	entry, ok := c.releases[id]
	c.mu.Unlock()
	if ok && time.Now().Before(entry.expires) {
		return entry.release, nil
	}
	release, err := lookup(ctx, id)
	if err != nil || release == nil {
		return release, err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.releases == nil || len(c.releases) >= 256 {
		c.releases = make(map[uuid.UUID]cachedOccurrenceRelease)
	}
	c.releases[id] = cachedOccurrenceRelease{expires: time.Now().Add(10 * time.Minute), release: release}
	return release, nil
}

type occurrenceProof struct {
	release, recording, track         string
	conflict, ignoreDeclaredRecording bool
	evidence                          *occurrenceMBEvidence
}

type occurrenceMBEvidence struct {
	ReleaseRequested     string `json:"release_requested"`
	ReleaseReturned      string `json:"release_returned"`
	ReleaseTitle         string `json:"release_title"`
	DeclaredReleaseTitle string `json:"declared_release_title"`
	ReleaseTitleDiffers  bool   `json:"release_title_differs,omitempty"`
	RecordingDeclared    string `json:"recording_declared,omitempty"`
	RecordingReturned    string `json:"recording_returned,omitempty"`
	TrackRecording       string `json:"track_recording,omitempty"`
	TrackDeclared        string `json:"track_declared,omitempty"`
	TrackReturned        string `json:"track_returned,omitempty"`
	TrackTitle           string `json:"track_title,omitempty"`
}

type occurrenceMatch struct {
	id, recording, release   int64
	trackMBID, recordingMBID string
	disc, position           int
	text, validated          bool
}

// verifyOccurrence checks release-track membership before acquiring the SQLite writer.
func verifyOccurrence(ctx context.Context, snapshot ingest.Snapshot, client mbz.MusicBrainzCaller, caches ...*occurrenceReleaseCache) occurrenceProof {
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
	var release *mbz.MusicBrainzRelease
	var err error
	if len(caches) > 0 {
		release, err = caches[0].occurrenceReleaseLookup(ctx, releaseID, getRelease)
	} else {
		release, err = getRelease(ctx, releaseID)
	}
	if err != nil || release == nil {
		return occurrenceProof{}
	}
	canonicalRelease := uuidFromSnapshot(release.ID)
	if canonicalRelease == uuid.Nil {
		return occurrenceProof{}
	}
	evidence := &occurrenceMBEvidence{
		ReleaseRequested: releaseID.String(), ReleaseReturned: canonicalRelease.String(),
		ReleaseTitle: release.Title, DeclaredReleaseTitle: snapshot.Release, TrackDeclared: snapshot.TrackMBID,
	}
	if release.Title != snapshot.Release {
		// Different presented titles leave identity to the catalog's metadata comparison.
		evidence.ReleaseTitleDiffers = true
		return occurrenceProof{evidence: evidence}
	}
	proof := occurrenceProof{release: canonicalRelease.String()}

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
		recording := uuidFromSnapshot(track.Recording.ID)
		if snapshot.MBIDSource == "" && declared != uuid.Nil && recording != uuid.Nil && declared != recording {
			evidence.RecordingDeclared, evidence.TrackRecording = declared.String(), recording.String()
			evidence.TrackReturned, evidence.TrackTitle = track.ID, track.Title
			found, err := client.GetTrack(ctx, declared)
			if err != nil || found == nil || uuidFromSnapshot(found.ID) == uuid.Nil {
				// A failed canonical lookup provides no evidence against the declaration.
				return occurrenceProof{ignoreDeclaredRecording: true}
			}
			resolved := uuidFromSnapshot(found.ID)
			evidence.RecordingReturned = resolved.String()
			if resolved != recording {
				return occurrenceProof{conflict: true, evidence: evidence}
			}
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
		var review bool
		err := tx.QueryRowContext(ctx, `SELECT e.time_precision,coalesce(json_extract(e.reason_codes,'$[0]'),''),f.review
			FROM ingest_events e JOIN listening_profiles p ON p.id=e.profile_id
			JOIN profile_ingest_flags f ON f.profile_id=p.id AND f.occurrence_write=1
			WHERE e.receipt_id=? AND e.profile_id=? AND e.mode='active' AND e.resolver_variant='occurrence'
			  AND e.closed_at IS NULL AND e.state IN ('received','retryable') AND e.resolution_version=?
			  AND e.work_version=? AND e.lease_owner=? AND e.lease_until>?`,
			r.id, r.profile, r.version, r.work, r.lease, s.ingestNow().Unix()).Scan(&precision, &admissionReason, &review)
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		if err != nil {
			return err
		}
		origin := occurrenceOrigin(r)
		if proof.conflict {
			return writeOccurrenceDecision(ctx, tx, r, occurrenceMatch{}, "pending_identity", "contradictory_musicbrainz_identity", precision, nil, "", proof.evidence)
		}
		if origin == "" || strings.TrimSpace(r.snapshot.Title) == "" || len(observedCredits(r.snapshot)) == 0 || r.snapshot.DiscNumber < 0 || r.snapshot.TrackNumber < 0 {
			return writeOccurrenceDecision(ctx, tx, r, occurrenceMatch{}, "pending_identity", "unresolved_identity", precision, nil, "", proof.evidence)
		}
		if admissionReason == "client_batch_conflict" {
			return writeOccurrenceDecision(ctx, tx, r, occurrenceMatch{}, "pending_duplicate", admissionReason, precision, nil, "", proof.evidence)
		}
		// Release-less listens use the usual duplicate decision and relay transaction.
		match, conflicts, err := resolveOccurrence(ctx, tx, r.snapshot, proof)
		if err != nil {
			return err
		}
		if len(conflicts) > 0 {
			return writeOccurrenceDecision(ctx, tx, r, occurrenceMatch{}, "pending_identity", "ambiguous_occurrence", precision, conflicts, "", proof.evidence)
		}
		// An exact repeat stays a single listen even when candidate review is disabled.
		candidates, exact, err := occurrenceListenCandidates(ctx, tx, r, match.id)
		if err != nil {
			return err
		}
		reason := "distinct_occurrence_event"
		if len(candidates) > 0 {
			if review {
				reason = "ambiguous_time_candidate"
				if exact != "" {
					reason = "legacy_identity_collision"
				}
				return writeOccurrenceDecision(ctx, tx, r, match, "pending_duplicate", reason, precision, candidates, "", proof.evidence)
			}
			reason = "review_deferred"
			if exact != "" {
				return writeOccurrenceDecision(ctx, tx, r, match, "duplicate", reason, precision, candidates, exact, proof.evidence)
			}
		}
		if match.id == 0 {
			conflicts, err := occurrenceCreditConflicts(ctx, tx, observedCredits(r.snapshot))
			if err != nil {
				return err
			}
			if len(conflicts) > 0 {
				return writeOccurrenceDecision(ctx, tx, r, occurrenceMatch{}, "pending_identity", "ambiguous_artist_credit", precision, conflicts, "", proof.evidence)
			}
			match, err = createOccurrence(ctx, tx, r.snapshot, proof, match)
			if err != nil {
				return err
			}
		} else if proof.recording != "" {
			// Only independent proof can promote a provisional recording or add a track MBID.
			if _, err := tx.ExecContext(ctx, `UPDATE recordings SET musicbrainz_id=?,mbid_source='validated'
				WHERE id=? AND (musicbrainz_id IS NULL OR musicbrainz_id=?)`, proof.recording, match.recording, proof.recording); err != nil {
				return err
			}
			if proof.track != "" && match.trackMBID == "" {
				if _, err := tx.ExecContext(ctx, `UPDATE occurrences SET track_mbid=? WHERE id=?`, proof.track, match.id); err != nil {
					return err
				}
			}
		}
		event := uuid.NewString()
		if _, err := tx.ExecContext(ctx, `INSERT INTO listens(profile_id,track_id,listened_at,client,event_id,origin) VALUES(?,?,?,?,?,?)`,
			r.profile, match.id, r.unix, r.client, event, origin); err != nil {
			return fmt.Errorf("write occurrence listen: %w", err)
		}
		// Persist the acceptance before creating its relay intent.
		if err := writeOccurrenceDecision(ctx, tx, r, match, "accepted", reason, precision, candidates, event, proof.evidence); err != nil {
			return err
		}
		return insertRelayIntent(ctx, tx, r, event, time.Now().Unix())
	})
}

// writeOccurrenceDecision persists the final state and its revision in the same transaction.
func writeOccurrenceDecision(ctx context.Context, tx *sql.Conn, r claimedReceipt, match occurrenceMatch, state, reason, precision string, candidates []string, event string, mbEvidence *occurrenceMBEvidence) error {
	status, decision := "pending", "review"
	var closed any
	now := time.Now().Unix()
	switch {
	case state == "accepted":
		status, decision, closed = "resolved", "different", now
	case state == "duplicate":
		status, decision, closed = "resolved", "same", now
	case state == "pending_duplicate" && match.id != 0:
		status = "resolved"
	}
	evidence, err := json.Marshal(struct {
		Candidates  []string              `json:"candidates,omitempty"`
		Precision   string                `json:"precision"`
		Truncated   bool                  `json:"truncated,omitempty"`
		MusicBrainz *occurrenceMBEvidence `json:"musicbrainz,omitempty"`
	}{candidates[:min(len(candidates), 32)], precision, len(candidates) > 32, mbEvidence})
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

// occurrenceListenCandidates returns the review window and the first exact match, if any.
// Evidence of distinct deliveries excludes a candidate even at the same second.
func occurrenceListenCandidates(ctx context.Context, tx *sql.Conn, r claimedReceipt, occurrence int64) ([]string, string, error) {
	if occurrence == 0 {
		return nil, "", nil
	}
	rows, err := tx.QueryContext(ctx, `SELECT l.event_id,l.listened_at FROM listens l
		WHERE l.profile_id=? AND l.track_id=? AND l.listened_at BETWEEN ? AND ?
		AND NOT EXISTS (
			SELECT 1 FROM ingest_events prior JOIN ingest_events incoming ON incoming.receipt_id=?
			WHERE (prior.profile_id,prior.event_id)=(l.profile_id,l.event_id) AND prior.state='accepted'
			  AND prior.source_instance_id=incoming.source_instance_id AND (
			    ((prior.external_id_trusted=1 OR json_type(prior.evidence,'$.trusted_replay')='true')
			      AND incoming.external_id_trusted=1 AND prior.external_namespace=incoming.external_namespace
			      AND prior.external_event_id<>incoming.external_event_id)
			    OR (prior.file_digest=incoming.file_digest AND prior.file_format=incoming.file_format
			      AND prior.format_version=incoming.format_version AND prior.item_ordinal<>incoming.item_ordinal)))
		ORDER BY (l.listened_at=?) DESC,l.listened_at,l.event_id LIMIT 33`, r.profile, occurrence, r.unix-2, r.unix+2, r.id, r.unix)
	if err != nil {
		return nil, "", err
	}
	defer rows.Close()
	var candidates []string
	var exact string
	for rows.Next() {
		var id string
		var stamp int64
		if err := rows.Scan(&id, &stamp); err != nil {
			return nil, "", err
		}
		if stamp == r.unix && exact == "" {
			exact = id
		}
		candidates = append(candidates, "event:"+id)
	}
	return candidates, exact, rows.Err()
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
