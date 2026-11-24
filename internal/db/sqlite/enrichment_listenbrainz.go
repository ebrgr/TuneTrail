package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"reflect"
	"time"

	"github.com/ebrgr/TuneTrail/internal/enrichment"
	"github.com/ebrgr/TuneTrail/internal/ingest"
	"github.com/google/uuid"
)

type enrichmentReader interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

type listenBrainzRun struct {
	request enrichment.ListenBrainzRequest
	version int64
	result  enrichment.Result
}

type listenBrainzTarget struct {
	request enrichment.ListenBrainzRequest
	version int64
	token   string
}

const listenBrainzRelay = `
	FROM ingest_events e JOIN relay_destinations d ON d.profile_id=e.profile_id AND d.generation=e.relay_admission_generation
	JOIN profile_relays p ON p.profile_id=d.profile_id
	WHERE d.awaiting_approval=0 AND p.url=? AND p.token<>''
	AND (d.pause_class IS NULL OR d.paused_until<=unixepoch())`

// enqueueListenBrainzEnrichment queues only receipts with an eligible profile relay.
// The schema 24 trigger already covers the same receipt when Subsonic is active.
func (s *Sqlite) enqueueListenBrainzEnrichment(ctx context.Context, tx *sql.Conn, receipt string) error {
	if !s.enrichmentReady.Load() {
		return nil
	}
	_, err := tx.ExecContext(ctx, `INSERT OR IGNORE INTO enrichment_jobs(receipt_id) SELECT e.receipt_id
		`+listenBrainzRelay+` AND e.receipt_id=? AND e.mode='active' AND e.source_type='listenbrainz'
		AND e.file_format IS NULL AND NOT (SELECT enabled FROM enrichment_configuration WHERE singleton=1)`,
		enrichment.ListenBrainzBase, receipt)
	return err
}

// matches fences credential changes and all evidence used to select the fallback fields.
func (r listenBrainzRun) matches(target listenBrainzTarget) bool {
	return target.token != "" && r.version == target.version && reflect.DeepEqual(r.request, target.request)
}

// lookupListenBrainz reads credentials immediately before the bounded network call.
func (s *Sqlite) lookupListenBrainz(ctx context.Context, c enrichmentClaim, primary enrichment.Result, client *enrichment.ListenBrainz) (listenBrainzRun, error) {
	target, err := prepareListenBrainz(ctx, s.db, c, primary)
	if err != nil {
		return listenBrainzRun{}, err
	}
	attempt, cancel := context.WithTimeout(ctx, 7*time.Second)
	defer cancel()
	return listenBrainzRun{request: target.request, version: target.version, result: client.Lookup(attempt, target.token, target.request)}, nil
}

// prepareListenBrainz uses the approved relay generation captured by this receipt.
// The exact base URL binds the credential to the official HTTPS API.
func prepareListenBrainz(ctx context.Context, q enrichmentReader, c enrichmentClaim, primary enrichment.Result) (listenBrainzTarget, error) {
	var target listenBrainzTarget
	err := q.QueryRowContext(ctx, `SELECT d.generation,d.config_version,p.token `+listenBrainzRelay+
		` AND e.receipt_id=? AND e.profile_id=?`, enrichment.ListenBrainzBase, c.receipt, c.profile).
		Scan(&target.request.Generation, &target.version, &target.token)
	if errors.Is(err, sql.ErrNoRows) {
		return target, nil
	}
	if err != nil {
		return target, err
	}
	req := &target.request
	req.ProfileID = c.profile
	if err := json.Unmarshal([]byte(c.snapshot), &req.Snapshot); err != nil {
		return target, err
	}
	s := &req.Snapshot
	// A prior declaration of any provenance occupies the field, even when its values disagree.
	var recording, declared sql.NullString
	var count int
	var received bool
	err = q.QueryRowContext(ctx, `SELECT r.musicbrainz_id,
		(SELECT min(value) FROM metadata_values WHERE profile_id=?1 AND recording_id=r.id AND kind='mbid'),
		(SELECT count(DISTINCT value) FROM metadata_values WHERE profile_id=?1 AND recording_id=r.id AND kind='mbid'),
		EXISTS(SELECT 1 FROM ingest_events e WHERE e.profile_id=?1 AND e.recording_id=r.id AND e.state IN ('accepted','duplicate')
			AND (coalesce(json_extract(e.original_snapshot,'$.recording_mbid'),'')<>'' OR coalesce(json_extract(e.original_snapshot,'$.mapped_recording_mbid'),'')<>''))
		FROM recordings r WHERE r.id=?2`, c.profile, c.recording).Scan(&recording, &declared, &count, &received)
	if errors.Is(err, sql.ErrNoRows) {
		return listenBrainzTarget{}, nil
	}
	if err != nil {
		return target, err
	}
	if received && uuidFromSnapshot(s.Recording, s.MappedRecording) == uuid.Nil && !recording.Valid {
		// There is source evidence on another receipt without a selected recording ID.
		return listenBrainzTarget{}, nil
	}
	if recording.Valid {
		for _, id := range []string{s.Recording, s.MappedRecording} {
			if known := uuidFromSnapshot(id); known != uuid.Nil && known != uuidFromSnapshot(recording.String) {
				return listenBrainzTarget{}, nil
			}
		}
		s.Recording = recording.String
	} else if count > 0 && uuidFromSnapshot(s.Recording, s.MappedRecording) == uuid.Nil {
		if count != 1 {
			return listenBrainzTarget{}, nil
		}
		s.Recording = declared.String
	}
	req.Recording = !received && !recording.Valid && count == 0 && uuidFromSnapshot(s.Recording, s.MappedRecording) == uuid.Nil
	for _, observation := range primary.Declarations {
		id := uuidFromSnapshot(observation.Value)
		if observation.Source != "navidrome-subsonic" || observation.Entity != "recording" || id == uuid.Nil {
			continue
		}
		if known := uuidFromSnapshot(s.Recording, s.MappedRecording); known != uuid.Nil && known != id {
			return listenBrainzTarget{}, nil
		}
		s.Recording = observation.Value
		req.Recording = false
	}

	for _, name := range enrichment.MissingArtistNames(*s) {
		id := c.artists[name]
		if id == 0 {
			continue
		}
		var known sql.NullString
		var occupied bool
		err := q.QueryRowContext(ctx, scopedEnrichmentArtist, c.profile, id).Scan(&known, &occupied)
		if errors.Is(err, sql.ErrNoRows) {
			continue
		}
		if err != nil {
			return target, err
		}
		primaryID := subsonicArtistID(primary, name)
		if occupied {
			if known.Valid {
				if primaryID != "" && uuidFromSnapshot(primaryID) != uuidFromSnapshot(known.String) {
					return listenBrainzTarget{}, nil
				}
				s.ArtistMbidMappings = append(s.ArtistMbidMappings, ingest.ArtistMapping{Name: name, MBID: known.String})
			}
			continue
		}
		if primaryID != "" {
			s.ArtistMbidMappings = append(s.ArtistMbidMappings, ingest.ArtistMapping{Name: name, MBID: primaryID})
		} else {
			req.Artists = append(req.Artists, name)
		}
	}
	return target, nil
}

// scopedEnrichmentArtist distinguishes an empty field from conflicting prior evidence.
const scopedEnrichmentArtist = `
	WITH scope(profile_id,artist_id) AS (VALUES (?1,?2)), evidence AS (
		SELECT a.musicbrainz_id AS value FROM artists a WHERE a.id=?2 AND a.musicbrainz_id IS NOT NULL
		UNION SELECT value FROM metadata_values WHERE profile_id=?1 AND artist_id=?2 AND kind='mbid'
	)
	SELECT CASE WHEN count(*)=1 THEN min(value) END, count(*)>0 OR EXISTS (
		SELECT 1 FROM scope s CROSS JOIN artist_tracks a CROSS JOIN artist_aliases n CROSS JOIN enrichment_received_artist_credits r
		WHERE a.artist_id=s.artist_id AND n.artist_id=a.artist_id
		  AND r.profile_id=s.profile_id AND r.occurrence_id=a.track_id AND r.name=n.alias
		UNION ALL
		SELECT 1 FROM scope s CROSS JOIN occurrence_credits a CROSS JOIN enrichment_received_artist_credits r
		WHERE a.artist_id=s.artist_id AND r.profile_id=s.profile_id AND r.occurrence_id=a.occurrence_id AND r.name=a.name
	) FROM evidence`

// subsonicArtistID reads only the declaration for this exact artist credit.
func subsonicArtistID(primary enrichment.Result, name string) string {
	for _, observation := range primary.Declarations {
		if observation.Source == "navidrome-subsonic" && observation.Entity == "artist" && observation.Name == name &&
			uuidFromSnapshot(observation.Value) != uuid.Nil {
			return observation.Value
		}
	}
	return ""
}

// combinedEnrichmentStatus folds the providers' terminal states into the shared job.
func combinedEnrichmentStatus(first, second enrichment.Status) enrichment.Status {
	for _, status := range []enrichment.Status{enrichment.Found, enrichment.Failed, enrichment.Ambiguous, enrichment.Absent} {
		if first == status || second == status {
			return status
		}
	}
	return enrichment.Disabled
}
