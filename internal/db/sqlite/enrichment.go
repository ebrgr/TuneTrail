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

// enrichmentLease is how long one step may hold the receipt. Every finished step renews it, so a
// chain of slow sources never loses the lease between providers.
const enrichmentLease = 60

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

// ResolveNextEnrichment keeps the ISSUE-174 entry point: one provider, no fallback.
func (s *Sqlite) ResolveNextEnrichment(ctx context.Context, provider enrichment.Provider) (bool, error) {
	return s.ResolveNextEnrichmentWithFallback(ctx, provider, nil)
}

// ResolveNextEnrichmentWithFallback keeps the ISSUE-175 entry point: Navidrome first, then
// ListenBrainz. It builds the two-provider registry those callers implied and runs the chain.
func (s *Sqlite) ResolveNextEnrichmentWithFallback(ctx context.Context, provider enrichment.Provider, fallback *enrichment.ListenBrainz) (bool, error) {
	registry := enrichment.NewRegistry()
	if err := registry.Register(enrichment.SubsonicDescriptor, provider); err != nil {
		return false, err
	}
	var listenBrainz enrichment.Provider
	if fallback != nil {
		listenBrainz = enrichment.NewListenBrainzProvider(fallback)
	}
	if err := registry.Register(enrichment.ListenBrainzDescriptor, listenBrainz); err != nil {
		return false, err
	}
	return s.ResolveNextEnrichmentChain(ctx, registry)
}

// ResolveNextEnrichmentChain reserves one receipt and walks the identity chain and then the
// metadata chain, in the stored order, with one fenced write per provider. It makes independent,
// bounded attempts outside the writer lock; every source decides for itself which fields are still
// missing, so a later source is only asked for what earlier ones did not resolve.
func (s *Sqlite) ResolveNextEnrichmentChain(ctx context.Context, registry *enrichment.Registry) (bool, error) {
	claim, err := s.claimEnrichment(ctx)
	if err != nil || claim.receipt == "" {
		return false, err
	}
	if claim.lease == "" {
		return true, nil
	}
	var original ingest.Snapshot
	if err := json.Unmarshal([]byte(claim.snapshot), &original); err != nil {
		return true, err
	}
	for _, chain := range enrichment.Chains {
		order, err := s.enrichmentOrder(ctx, registry, chain)
		if err != nil {
			return true, err
		}
		for _, descriptor := range order {
			provider, _ := registry.Provider(descriptor.ID)
			closed, err := s.runEnrichmentStep(ctx, &claim, descriptor, provider, chain)
			if err != nil || closed {
				return true, err
			}
		}
	}
	// A canceled process leaves a finite lease for recovery rather than writing with a canceled context.
	if ctx.Err() != nil {
		return true, ctx.Err()
	}
	return true, s.closeEnrichment(ctx, claim)
}

// enrichmentBinding prepares one provider's request from local evidence and rechecks that evidence
// under the writer lock. Token-bound sources need more than the snapshot, so they bind specially.
type enrichmentBinding interface {
	prepare(ctx context.Context, c *enrichmentClaim, d enrichment.Descriptor, p enrichment.Provider, chain enrichment.Chain) (enrichmentTarget, error)
	// current reports whether the evidence behind the call still holds when the result is written.
	current(ctx context.Context, q enrichmentReader, c enrichmentClaim, t enrichmentTarget) (bool, error)
}

type enrichmentTarget struct {
	request enrichment.Request
	timeout time.Duration
	// listenBrainz holds the evidence that fences a token-bound lookup.
	listenBrainz listenBrainzTarget
}

func (s *Sqlite) bindingFor(d enrichment.Descriptor) enrichmentBinding {
	if d.Eligibility == enrichment.EligibleProfileRelay {
		return listenBrainzBinding{s}
	}
	return snapshotBinding{s}
}

// snapshotBinding sends the receipt snapshot with the identifiers already known to the catalog.
type snapshotBinding struct{ s *Sqlite }

func (b snapshotBinding) prepare(ctx context.Context, c *enrichmentClaim, d enrichment.Descriptor, p enrichment.Provider, chain enrichment.Chain) (enrichmentTarget, error) {
	var snapshot ingest.Snapshot
	if err := json.Unmarshal([]byte(c.snapshot), &snapshot); err != nil {
		return enrichmentTarget{}, err
	}
	// Skip external work for identifiers already known in this profile's catalog.
	if err := b.s.enrichmentKnownIDs(ctx, c, &snapshot, p.Instance()); err != nil {
		return enrichmentTarget{}, err
	}
	prior, err := b.s.enrichmentPrior(ctx, *c, d.Source)
	if err != nil {
		return enrichmentTarget{}, err
	}
	return enrichmentTarget{request: enrichment.Request{ProfileID: c.profile, Snapshot: snapshot, Chain: chain, Prior: prior}, timeout: 20 * time.Second}, nil
}

func (snapshotBinding) current(context.Context, enrichmentReader, enrichmentClaim, enrichmentTarget) (bool, error) {
	return true, nil
}

// runEnrichmentStep asks one provider and stores its answer. It reports closed when the receipt
// can no longer be worked, so the caller stops the chain.
func (s *Sqlite) runEnrichmentStep(ctx context.Context, c *enrichmentClaim, d enrichment.Descriptor, p enrichment.Provider, chain enrichment.Chain) (bool, error) {
	var state string
	err := s.db.QueryRowContext(ctx, `SELECT state FROM enrichment_provider_runs WHERE receipt_id=? AND provider=? AND chain=?`, c.receipt, d.ID, string(chain)).Scan(&state)
	if err == nil && state != string(enrichment.Failed) {
		// A recovered job keeps the answers that were already final instead of repeating their calls.
		return false, nil
	}
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return false, err
	}
	binding := s.bindingFor(d)
	target, err := binding.prepare(ctx, c, d, p, chain)
	if err != nil {
		return false, err
	}
	attempt, cancel := context.WithTimeout(ctx, target.timeout)
	result := p.Lookup(attempt, target.request)
	cancel()
	if ctx.Err() != nil {
		return false, ctx.Err()
	}
	return s.finishEnrichmentStep(ctx, *c, d, p, binding, target, chain, result)
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

// enrichmentPrior lists the IDs that other sources already stored for this receipt's recording and
// linked artists, so a later source leaves those fields alone. Names come from the linked credits.
func (s *Sqlite) enrichmentPrior(ctx context.Context, c enrichmentClaim, exceptSource string) ([]enrichment.Declaration, error) {
	var prior []enrichment.Declaration
	collect := func(entity, name, query string, args ...any) error {
		rows, err := s.db.QueryContext(ctx, query, args...)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var value, source string
			if err := rows.Scan(&value, &source); err != nil {
				return err
			}
			prior = append(prior, enrichment.Declaration{Entity: entity, Name: name, Field: "musicbrainz_id", Value: value, Source: source})
		}
		return rows.Err()
	}
	if err := collect("recording", "", `SELECT value,source FROM metadata_values WHERE profile_id=? AND recording_id=? AND kind='mbid'
		AND source<>? ORDER BY value,source`, c.profile, c.recording, exceptSource); err != nil {
		return nil, err
	}
	names := make([]string, 0, len(c.artists))
	for name := range c.artists {
		names = append(names, name)
	}
	slices.Sort(names)
	for _, name := range names {
		if err := collect("artist", name, `SELECT value,source FROM metadata_values WHERE profile_id=? AND artist_id=? AND kind='mbid'
			AND source<>? ORDER BY value,source`, c.profile, c.artists[name], exceptSource); err != nil {
			return nil, err
		}
	}
	return prior, nil
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
		RETURNING receipt_id,coalesce(lease,'')`, lease, now+enrichmentLease, next, now).Scan(&c.receipt, &c.lease)
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

// finishEnrichmentStep revalidates ownership, links and existing identifiers under the writer lock,
// stores one provider's answer and renews the lease for the next step. It reports closed when the
// receipt changed underneath the lookup and the job was failed instead.
func (s *Sqlite) finishEnrichmentStep(ctx context.Context, c enrichmentClaim, d enrichment.Descriptor, p enrichment.Provider, binding enrichmentBinding,
	target enrichmentTarget, chain enrichment.Chain, result enrichment.Result) (bool, error) {
	var original ingest.Snapshot
	if err := json.Unmarshal([]byte(c.snapshot), &original); err != nil {
		return false, err
	}
	missingArtists := enrichment.MissingArtistNames(original)
	closed := false
	failJob := func(tx *sql.Conn) error {
		closed = true
		_, err := tx.ExecContext(ctx, `UPDATE enrichment_jobs SET state='failed',lease=NULL,lease_until=NULL WHERE receipt_id=? AND lease=?`, c.receipt, c.lease)
		return err
	}
	err := s.withIngestWriter(ctx, func(tx *sql.Conn) error {
		var event string
		err := tx.QueryRowContext(ctx, `SELECT l.event_id FROM enrichment_jobs j JOIN ingest_events e USING(receipt_id)
			JOIN listens l ON (l.profile_id,l.event_id)=(e.profile_id,e.event_id)
			JOIN occurrences o ON o.id=l.track_id AND o.id=e.occurrence_id AND o.recording_id=e.recording_id
			WHERE j.receipt_id=? AND j.state='working' AND j.lease=? AND j.lease_until>? AND e.profile_id=?
			AND e.resolution_version=? AND e.occurrence_id=? AND e.recording_id=? AND e.original_snapshot=?
			AND e.state IN ('accepted','duplicate')`, c.receipt, c.lease, time.Now().Unix(), c.profile, c.version, c.occurrence, c.recording, c.snapshot).Scan(&event)
		if errors.Is(err, sql.ErrNoRows) {
			return failJob(tx)
		}
		if err != nil {
			return err
		}
		var catalog string
		if err := tx.QueryRowContext(ctx, enrichmentCatalogFingerprint, c.occurrence).Scan(&catalog); err != nil {
			return err
		}
		if catalog != c.catalog {
			return failJob(tx)
		}
		status := result.Status
		declarations := result.Declarations
		if len(declarations) > 0 {
			// A changed relay or catalog observation invalidates in-flight lookup evidence.
			holds, err := binding.current(ctx, tx, c, target)
			if err != nil {
				return err
			}
			if !holds {
				declarations, status = nil, enrichment.Ambiguous
			}
		}
		wrote := false
		for _, declaration := range declarations {
			// A provider can only write under its own source, never under another provider's.
			if declaration.Source != d.Source {
				continue
			}
			wrote = true
			if err := writeMBIDDeclaration(ctx, tx, c, event, original, missingArtists, p.Instance(), declaration); err != nil {
				return err
			}
		}
		switch status {
		case enrichment.Found, enrichment.Absent, enrichment.Ambiguous, enrichment.Disabled, enrichment.Failed:
		default:
			status = enrichment.Failed
		}
		// A source that needs a profile credential records its instance only when it contributed.
		instance := ""
		if d.Eligibility == enrichment.EligibleConfigured || wrote {
			instance = p.Instance()
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO enrichment_provider_runs(receipt_id,provider,chain,state,instance,finished_at)
			VALUES(?,?,?,?,nullif(?,''),?) ON CONFLICT(receipt_id,provider,chain)
			DO UPDATE SET state=excluded.state,instance=excluded.instance,finished_at=excluded.finished_at`,
			c.receipt, d.ID, string(chain), string(status), instance, time.Now().Unix()); err != nil {
			return err
		}
		res, err := tx.ExecContext(ctx, `UPDATE enrichment_jobs SET lease_until=? WHERE receipt_id=? AND lease=? AND state='working'`,
			time.Now().Unix()+enrichmentLease, c.receipt, c.lease)
		if err != nil {
			return err
		}
		if n, err := res.RowsAffected(); err != nil || n == 0 {
			closed = true
		}
		return err
	})
	return closed, err
}

// writeMBIDDeclaration stores one declared MusicBrainz ID for the recording or for exactly one
// linked artist. Other fields (genres, tags, dates) arrive with their providers in ISSUE-217 and 218.
func writeMBIDDeclaration(ctx context.Context, tx *sql.Conn, c enrichmentClaim, event string, original ingest.Snapshot,
	missingArtists []string, instance string, declaration enrichment.Declaration) error {
	id, err := uuid.Parse(declaration.Value)
	if err != nil || id == uuid.Nil || declaration.Field != "musicbrainz_id" || declaration.ObservedAt.IsZero() || instance == "" {
		return nil
	}
	at := declaration.ObservedAt.UTC().Format(time.RFC3339Nano)
	switch declaration.Entity {
	case "recording":
		if uuidFromSnapshot(original.Recording, original.MappedRecording) != uuid.Nil {
			return nil
		}
		_, err = tx.ExecContext(ctx, `INSERT OR IGNORE INTO metadata_values(profile_id,recording_id,occurrence_id,event_id,kind,instance,value,source,observed_at)
			SELECT ?,r.id,o.id,?,'mbid',?,?,?,? FROM occurrences o JOIN recordings r ON r.id=o.recording_id
			WHERE o.id=? AND r.id=? AND r.musicbrainz_id IS NULL`, c.profile, event, instance, id.String(), declaration.Source, at, c.occurrence, c.recording)
	case "artist":
		if !slices.Contains(missingArtists, declaration.Name) {
			return nil
		}
		expected := c.artists[declaration.Name]
		if expected == 0 {
			return nil
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
			WHERE at.track_id=? AND aa.alias=? COLLATE BINARY))`, c.profile, event, instance, id.String(), declaration.Source, at, expected,
			c.occurrence, declaration.Name, c.occurrence, declaration.Name, c.occurrence, declaration.Name, c.occurrence, declaration.Name)
	}
	return err
}

// closeEnrichment folds the answers of every provider into the receipt job.
func (s *Sqlite) closeEnrichment(ctx context.Context, c enrichmentClaim) error {
	return s.withIngestWriter(ctx, func(tx *sql.Conn) error {
		rows, err := tx.QueryContext(ctx, `SELECT r.state,coalesce(r.instance,'') FROM enrichment_provider_runs r
			JOIN enrichment_providers p ON p.id=r.provider WHERE r.receipt_id=?
			ORDER BY CASE r.chain WHEN 'identity' THEN 0 ELSE 1 END,p.priority,p.id`, c.receipt)
		if err != nil {
			return err
		}
		defer rows.Close()
		status, instance := enrichment.Disabled, ""
		for rows.Next() {
			var state, runInstance string
			if err := rows.Scan(&state, &runInstance); err != nil {
				return err
			}
			status = combinedEnrichmentStatus(status, enrichment.Status(state))
			if instance == "" {
				instance = runInstance
			}
		}
		if err := rows.Err(); err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, `UPDATE enrichment_jobs SET state=?,instance=nullif(?,''),lease=NULL,lease_until=NULL WHERE receipt_id=? AND lease=?`,
			string(status), instance, c.receipt, c.lease)
		return err
	})
}
