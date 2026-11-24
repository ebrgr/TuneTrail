package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"slices"
	"time"

	"github.com/ebrgr/TuneTrail/internal/enrichment"
	"github.com/ebrgr/TuneTrail/internal/ingest"
	"github.com/google/uuid"
)

type enrichmentClaim struct {
	receipt, lease, snapshot       string
	catalog                        string
	profile                        int32
	occurrence, recording, version int64
	artists                        map[string]int64
}

// ConfigureEnrichment gates Subsonic work and preserves any eligible ListenBrainz jobs.
func (s *Sqlite) ConfigureEnrichment(ctx context.Context, enabled bool) error {
	err := s.withIngestWriter(ctx, func(tx *sql.Conn) error {
		if _, err := tx.ExecContext(ctx, `UPDATE enrichment_configuration SET enabled=? WHERE singleton=1`, enabled); err != nil {
			return err
		}
		if !enabled {
			_, err := tx.ExecContext(ctx, `UPDATE enrichment_jobs SET state='disabled',lease=NULL,lease_until=NULL
				WHERE state IN ('pending','working') AND NOT EXISTS (
					SELECT 1 `+listenBrainzRelay+` AND e.receipt_id=enrichment_jobs.receipt_id)`, enrichment.ListenBrainzBase)
			return err
		}
		return nil
	})
	// Admission only uses the enrichment tables after engine startup has configured them.
	if err == nil {
		s.enrichmentReady.Store(true)
	}
	return err
}

// enrichmentCatalogFingerprint captures the catalog fields used to bind a lookup.
const enrichmentCatalogFingerprint = `SELECT json_object(
	'title',o.presented_title,'release',o.release_id,'disc',o.disc_number,'track',o.track_number,
	'duration',t.duration,'recording',r.musicbrainz_id,
	'titles',(SELECT json_group_array(json_array(alias,source,is_primary)) FROM (SELECT alias,source,is_primary FROM track_aliases WHERE track_id=o.id ORDER BY alias,source)),
	'releases',(SELECT json_group_array(json_array(alias,source,is_primary)) FROM (SELECT alias,source,is_primary FROM release_aliases WHERE release_id=o.release_id ORDER BY alias,source)),
	'credits',(SELECT json_group_array(json_array(position,name,artist_id)) FROM (SELECT position,name,artist_id FROM occurrence_credits WHERE occurrence_id=o.id ORDER BY position)),
	'artists',(SELECT json_group_array(json_array(id,musicbrainz_id,aliases)) FROM (
	 SELECT a.id,a.musicbrainz_id,(SELECT json_group_array(json_array(alias,source,is_primary)) FROM (SELECT alias,source,is_primary FROM artist_aliases WHERE artist_id=a.id ORDER BY alias,source)) AS aliases
	 FROM artists a WHERE a.id IN (SELECT artist_id FROM artist_tracks WHERE track_id=o.id UNION SELECT artist_id FROM occurrence_credits WHERE occurrence_id=o.id) ORDER BY a.id)))
	FROM occurrences o JOIN recordings r ON r.id=o.recording_id LEFT JOIN tracks t ON t.id=o.legacy_track_id WHERE o.id=?`

// ResolveNextEnrichment makes one independent, bounded attempt outside the writer lock.
func (s *Sqlite) ResolveNextEnrichment(ctx context.Context, provider enrichment.Provider) (bool, error) {
	return s.ResolveNextEnrichmentWithFallback(ctx, provider, nil)
}

// ResolveNextEnrichmentWithFallback shares the receipt lease and writer transaction across sources.
func (s *Sqlite) ResolveNextEnrichmentWithFallback(ctx context.Context, provider enrichment.Provider, fallback *enrichment.ListenBrainz) (bool, error) {
	claim, err := s.claimEnrichment(ctx)
	if err != nil || claim.receipt == "" {
		return false, err
	}
	if claim.lease == "" {
		return true, nil
	}
	var snapshot ingest.Snapshot
	result := enrichment.Result{Status: enrichment.Failed}
	var lookup listenBrainzRun
	var instance string
	if err := json.Unmarshal([]byte(claim.snapshot), &snapshot); err == nil {
		result.Status = enrichment.Disabled
		if provider != nil {
			instance = provider.Instance()
		}
		if provider != nil || fallback != nil {
			// Skip external work for identifiers already known in this profile's catalog.
			if err := s.enrichmentKnownIDs(ctx, &claim, &snapshot, instance); err != nil {
				return true, err
			}
		}
		if provider != nil {
			attempt, cancel := context.WithTimeout(ctx, 20*time.Second)
			result = provider.Lookup(attempt, enrichment.Request{ProfileID: claim.profile, Snapshot: snapshot})
			cancel()
		}
		if fallback != nil {
			lookup, err = s.lookupListenBrainz(ctx, claim, result, fallback)
			if err != nil {
				return true, err
			}
		}
	}
	// A canceled process leaves a finite lease for recovery rather than writing with a canceled context.
	if ctx.Err() != nil {
		return true, ctx.Err()
	}
	return true, s.finishEnrichment(ctx, claim, instance, result, lookup)
}

// enrichmentKnownIDs augments only the transient lookup request, never the receipt.
func (s *Sqlite) enrichmentKnownIDs(ctx context.Context, c *enrichmentClaim, snapshot *ingest.Snapshot, instance string) error {
	var recording sql.NullString
	err := s.db.QueryRowContext(ctx, `SELECT coalesce(r.musicbrainz_id,(SELECT min(value) FROM metadata_values
		WHERE profile_id=? AND recording_id=? AND kind='mbid' AND instance=? HAVING count(DISTINCT value)=1))
		FROM recordings r WHERE r.id=?`, c.profile, c.recording, instance, c.recording).Scan(&recording)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	if uuidFromSnapshot(snapshot.Recording, snapshot.MappedRecording) == uuid.Nil && recording.Valid {
		snapshot.Recording = recording.String
	}
	c.artists = make(map[string]int64)
	for _, name := range enrichment.MissingArtistNames(*snapshot) {
		var mbid sql.NullString
		var artistID, count int64
		err := s.db.QueryRowContext(ctx, `SELECT count(DISTINCT a.id),coalesce(min(a.id),0),CASE WHEN count(DISTINCT a.id)=1 THEN min(coalesce(a.musicbrainz_id,
			(SELECT min(value) FROM metadata_values WHERE profile_id=? AND artist_id=a.id AND kind='mbid' AND instance=? HAVING count(DISTINCT value)=1))) END
			FROM artists a WHERE a.id IN (SELECT artist_id FROM occurrence_credits WHERE occurrence_id=? AND name=? AND artist_id IS NOT NULL
			UNION SELECT at.artist_id FROM artist_tracks at JOIN artist_aliases aa ON aa.artist_id=at.artist_id WHERE at.track_id=? AND aa.alias=? COLLATE BINARY)`,
			c.profile, instance, c.occurrence, name, c.occurrence, name).Scan(&count, &artistID, &mbid)
		if err != nil {
			return err
		}
		if count == 1 {
			c.artists[name] = artistID
		}
		if mbid.Valid {
			snapshot.ArtistMbidMappings = append(snapshot.ArtistMbidMappings, ingest.ArtistMapping{Name: name, MBID: mbid.String})
		}
	}
	return nil
}

// claimEnrichment limits recovery after crashes to two attempts per receipt.
func (s *Sqlite) claimEnrichment(ctx context.Context) (enrichmentClaim, error) {
	var c enrichmentClaim
	now := time.Now().Unix()
	const due = `(state='pending' OR (state='working' AND lease_until<=?))`
	// An idle poll must not compete with application writers for the SQLite lock.
	var next string
	err := s.db.QueryRowContext(ctx, `SELECT receipt_id FROM enrichment_jobs WHERE `+due+` ORDER BY rowid LIMIT 1`, now).Scan(&next)
	if errors.Is(err, sql.ErrNoRows) {
		return c, nil
	}
	if err != nil {
		return c, err
	}
	lease := uuid.NewString()
	err = s.db.QueryRowContext(ctx, `UPDATE enrichment_jobs SET
		state=CASE WHEN attempts>=2 THEN 'failed' ELSE 'working' END,
		lease=CASE WHEN attempts>=2 THEN NULL ELSE ? END,
		lease_until=CASE WHEN attempts>=2 THEN NULL ELSE ? END,
		attempts=attempts+(attempts<2)
		WHERE receipt_id=? AND `+due+`
		RETURNING receipt_id,coalesce(lease,'')`, lease, now+60, next, now).Scan(&c.receipt, &c.lease)
	if errors.Is(err, sql.ErrNoRows) {
		return c, nil
	}
	if err != nil || c.lease == "" {
		return c, err
	}
	err = s.db.QueryRowContext(ctx, `SELECT e.profile_id,e.original_snapshot,e.resolution_version,e.occurrence_id,e.recording_id
		FROM ingest_events e JOIN listens l ON (l.profile_id,l.event_id)=(e.profile_id,e.event_id)
		JOIN occurrences o ON o.id=l.track_id AND o.id=e.occurrence_id AND o.recording_id=e.recording_id
		WHERE e.receipt_id=? AND e.state IN ('accepted','duplicate')`, c.receipt).
		Scan(&c.profile, &c.snapshot, &c.version, &c.occurrence, &c.recording)
	if errors.Is(err, sql.ErrNoRows) {
		_, err = s.db.ExecContext(ctx, `UPDATE enrichment_jobs SET state='failed',lease=NULL,lease_until=NULL WHERE receipt_id=? AND lease=?`, c.receipt, c.lease)
		c.lease = ""
	}
	if err == nil && c.lease != "" {
		err = s.db.QueryRowContext(ctx, enrichmentCatalogFingerprint, c.occurrence).Scan(&c.catalog)
	}
	return c, err
}

// finishEnrichment revalidates ownership, links and existing identifiers under the writer lock.
func (s *Sqlite) finishEnrichment(ctx context.Context, c enrichmentClaim, instance string, result enrichment.Result, lookup listenBrainzRun) error {
	var original ingest.Snapshot
	if err := json.Unmarshal([]byte(c.snapshot), &original); err != nil {
		return err
	}
	missingArtists := enrichment.MissingArtistNames(original)
	return s.withIngestWriter(ctx, func(tx *sql.Conn) error {
		var event string
		err := tx.QueryRowContext(ctx, `SELECT l.event_id FROM enrichment_jobs j JOIN ingest_events e USING(receipt_id)
			JOIN listens l ON (l.profile_id,l.event_id)=(e.profile_id,e.event_id)
			JOIN occurrences o ON o.id=l.track_id AND o.id=e.occurrence_id AND o.recording_id=e.recording_id
			WHERE j.receipt_id=? AND j.state='working' AND j.lease=? AND j.lease_until>? AND e.profile_id=?
			AND e.resolution_version=? AND e.occurrence_id=? AND e.recording_id=? AND e.original_snapshot=?
			AND e.state IN ('accepted','duplicate')`, c.receipt, c.lease, time.Now().Unix(), c.profile, c.version, c.occurrence, c.recording, c.snapshot).Scan(&event)
		if errors.Is(err, sql.ErrNoRows) {
			_, err := tx.ExecContext(ctx, `UPDATE enrichment_jobs SET state='failed',lease=NULL,lease_until=NULL WHERE receipt_id=? AND lease=?`, c.receipt, c.lease)
			return err
		}
		if err != nil {
			return err
		}
		var catalog string
		if err := tx.QueryRowContext(ctx, enrichmentCatalogFingerprint, c.occurrence).Scan(&catalog); err != nil {
			return err
		}
		if catalog != c.catalog {
			_, err := tx.ExecContext(ctx, `UPDATE enrichment_jobs SET state='failed',lease=NULL,lease_until=NULL WHERE receipt_id=? AND lease=?`, c.receipt, c.lease)
			return err
		}
		declarations := result.Declarations
		status := result.Status
		if lookup.result.Status != "" {
			status = combinedEnrichmentStatus(status, lookup.result.Status)
		}
		if len(lookup.result.Declarations) > 0 {
			// A changed relay or catalog observation invalidates in-flight lookup evidence.
			current, err := prepareListenBrainz(ctx, tx, c, result)
			if err != nil {
				return err
			}
			if lookup.matches(current) {
				declarations = append(slices.Clone(declarations), lookup.result.Declarations...)
			} else {
				status = combinedEnrichmentStatus(result.Status, enrichment.Ambiguous)
			}
		}
		jobInstance := instance
		for _, declaration := range declarations {
			declarationInstance := instance
			switch declaration.Source {
			case enrichment.ListenBrainzSource:
				if !slices.Contains(lookup.result.Declarations, declaration) {
					continue
				}
				declarationInstance = enrichment.ListenBrainzBase
				if jobInstance == "" {
					jobInstance = declarationInstance
				}
			case "navidrome-subsonic":
			default:
				continue
			}
			id, err := uuid.Parse(declaration.Value)
			if err != nil || id == uuid.Nil || declaration.Field != "musicbrainz_id" || declaration.ObservedAt.IsZero() || declarationInstance == "" {
				continue
			}
			at := declaration.ObservedAt.UTC().Format(time.RFC3339Nano)
			switch declaration.Entity {
			case "recording":
				if uuidFromSnapshot(original.Recording, original.MappedRecording) != uuid.Nil {
					continue
				}
				_, err = tx.ExecContext(ctx, `INSERT OR IGNORE INTO metadata_values(profile_id,recording_id,occurrence_id,event_id,kind,instance,value,source,observed_at)
					SELECT ?,r.id,o.id,?,'mbid',?,?,?,? FROM occurrences o JOIN recordings r ON r.id=o.recording_id
					WHERE o.id=? AND r.id=? AND r.musicbrainz_id IS NULL`, c.profile, event, declarationInstance, id.String(), declaration.Source, at, c.occurrence, c.recording)
			case "artist":
				if !slices.Contains(missingArtists, declaration.Name) {
					continue
				}
				expected := c.artists[declaration.Name]
				if expected == 0 {
					continue
				}
				// Match only one linked artist with the exact observed credit, never by list position.
				_, err = tx.ExecContext(ctx, `INSERT OR IGNORE INTO metadata_values(profile_id,artist_id,event_id,kind,instance,value,source,observed_at)
					SELECT ?,a.id,?,'mbid',?,?,?,? FROM artists a WHERE a.musicbrainz_id IS NULL AND a.id=? AND a.id IN (
					 SELECT artist_id FROM occurrence_credits WHERE occurrence_id=? AND name=? AND artist_id IS NOT NULL
					 UNION SELECT at.artist_id FROM artist_tracks at JOIN artist_aliases aa ON aa.artist_id=at.artist_id
					 WHERE at.track_id=? AND aa.alias=? COLLATE BINARY)
					AND 1=(SELECT count(DISTINCT artist_id) FROM (
					 SELECT artist_id FROM occurrence_credits WHERE occurrence_id=? AND name=? AND artist_id IS NOT NULL
					 UNION SELECT at.artist_id FROM artist_tracks at JOIN artist_aliases aa ON aa.artist_id=at.artist_id
					WHERE at.track_id=? AND aa.alias=? COLLATE BINARY))`, c.profile, event, declarationInstance, id.String(), declaration.Source, at, expected,
					c.occurrence, declaration.Name, c.occurrence, declaration.Name, c.occurrence, declaration.Name, c.occurrence, declaration.Name)
			default:
				continue
			}
			if err != nil {
				return err
			}
		}
		switch status {
		case enrichment.Found, enrichment.Absent, enrichment.Ambiguous, enrichment.Disabled, enrichment.Failed:
		default:
			status = enrichment.Failed
		}
		_, err = tx.ExecContext(ctx, `UPDATE enrichment_jobs SET state=?,instance=nullif(?,''),lease=NULL,lease_until=NULL WHERE receipt_id=? AND lease=?`, string(status), jobInstance, c.receipt, c.lease)
		return err
	})
}
