package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"slices"
	"strings"
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
	descriptor := enrichment.SubsonicDescriptor
	descriptor.Chains = []enrichment.Chain{enrichment.ChainIdentity} // The legacy entry point retains its original contract.
	if err := registry.Register(descriptor, provider); err != nil {
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
	var answers []enrichmentAnswer
	metadataCollected := make(map[string]bool)
	for _, chain := range enrichment.Chains {
		order, err := s.enrichmentOrder(ctx, registry, chain)
		if err != nil {
			return true, err
		}
		for _, descriptor := range order {
			provider, _ := registry.Provider(descriptor.ID)
			// Subsonic obtains the same metadata while resolving identity, so avoid a second
			// network lookup in this pass. Recovered legacy runs must still get a metadata pass.
			_, subsonic := provider.(*enrichment.Subsonic)
			if chain == enrichment.ChainMetadata && subsonic && metadataCollected[descriptor.ID] {
				continue
			}
			answer, closed, err := s.runEnrichmentStep(ctx, &claim, descriptor, provider, chain)
			if err != nil || closed {
				return true, err
			}
			answers = append(answers, answer)
			if chain == enrichment.ChainIdentity && subsonic && !answer.recovered && answer.status != enrichment.Failed {
				metadataCollected[descriptor.ID] = true
			}
		}
	}
	// A canceled process leaves a finite lease for recovery rather than writing with a canceled context.
	if ctx.Err() != nil {
		return true, ctx.Err()
	}
	return true, s.closeEnrichment(ctx, claim, answers)
}

// enrichmentAnswer is what one provider said about a receipt in this pass.
type enrichmentAnswer struct {
	status    enrichment.Status
	instance  string
	recovered bool
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

// bindingFor picks how a provider's request is prepared. Today the only source that needs a profile
// relay credential is ListenBrainz, so that eligibility selects its binding; a second relay-bound
// source needs its own binding before it can reuse the eligibility.
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
	var metadataPresent bool
	if err := b.s.db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM descriptive_metadata m
 JOIN metadata_sources s ON s.id=m.source_id WHERE m.profile_id=? AND m.occurrence_id=? AND s.source=? AND s.instance=?
 AND NOT EXISTS (SELECT 1 FROM metadata_retries r WHERE r.profile_id=m.profile_id AND r.occurrence_id=m.occurrence_id AND r.source_id=m.source_id))`, c.profile, c.occurrence, d.Source, p.Instance()).Scan(&metadataPresent); err != nil {
		return enrichmentTarget{}, err
	}
	return enrichmentTarget{request: enrichment.Request{ProfileID: c.profile, Snapshot: snapshot, Chain: chain, Prior: prior, MetadataPresent: metadataPresent}, timeout: 20 * time.Second}, nil
}

func (snapshotBinding) current(context.Context, enrichmentReader, enrichmentClaim, enrichmentTarget) (bool, error) {
	return true, nil
}

// runEnrichmentStep asks one provider and stores its answer. It reports closed when the receipt
// can no longer be worked, so the caller stops the chain.
func (s *Sqlite) runEnrichmentStep(ctx context.Context, c *enrichmentClaim, d enrichment.Descriptor, p enrichment.Provider, chain enrichment.Chain) (enrichmentAnswer, bool, error) {
	var state, instance string
	err := s.db.QueryRowContext(ctx, `SELECT state,coalesce(instance,'') FROM enrichment_provider_runs WHERE receipt_id=? AND provider=? AND chain=?`,
		c.receipt, d.ID, string(chain)).Scan(&state, &instance)
	if err == nil && state != string(enrichment.Failed) {
		// A recovered job keeps the answers that were already final instead of repeating their calls.
		return enrichmentAnswer{status: enrichment.Status(state), instance: instance, recovered: true}, false, nil
	}
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return enrichmentAnswer{}, false, err
	}
	binding := s.bindingFor(d)
	target, err := binding.prepare(ctx, c, d, p, chain)
	if err != nil {
		return enrichmentAnswer{}, false, err
	}
	attempt, cancel := context.WithTimeout(ctx, target.timeout)
	result := p.Lookup(attempt, target.request)
	cancel()
	if ctx.Err() != nil {
		return enrichmentAnswer{}, false, ctx.Err()
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
	missing := enrichment.MissingArtistNames(*snapshot)
	for _, name := range snapshot.ArtistCreditNames() {
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
		if mbid.Valid && slices.Contains(missing, name) {
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
	rows, err := s.db.QueryContext(ctx, `SELECT m.value,s.source FROM descriptive_metadata m
 JOIN metadata_sources s ON s.id=m.source_id WHERE m.profile_id=? AND m.occurrence_id=? AND m.kind='isrc' AND s.source<>? ORDER BY m.value`, c.profile, c.occurrence, exceptSource)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var value, source string
		if err := rows.Scan(&value, &source); err != nil {
			return nil, err
		}
		prior = append(prior, enrichment.Declaration{Entity: "recording", Field: "isrc", Value: value, Source: source})
	}
	if err := rows.Err(); err != nil {
		return nil, err
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
	err = s.withIngestWriter(ctx, func(tx *sql.Conn) error {
		err := tx.QueryRowContext(ctx, `UPDATE enrichment_jobs SET
        state=CASE WHEN attempts>=2 THEN 'failed' ELSE 'working' END,
        lease=CASE WHEN attempts>=2 THEN NULL ELSE ? END,
        lease_until=CASE WHEN attempts>=2 THEN NULL ELSE ? END,
        attempts=attempts+(attempts<2)
        WHERE receipt_id=? AND `+due+`
        RETURNING receipt_id,coalesce(lease,'')`, lease, now+enrichmentLease, next, now).Scan(&c.receipt, &c.lease)
		if err != nil {
			return err
		}
		if c.lease == "" {
			return deleteSuccessfulRuns(ctx, tx, c.receipt)
		}
		return nil
	})
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
		err = s.withIngestWriter(ctx, func(tx *sql.Conn) error {
			return finishClaimedEnrichment(ctx, tx, c, "failed", "")
		})
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
//
// Only an answer that changed something is stored (found, ambiguous or failed). "Nothing to add" is
// by far the most common answer for a listen that already carries its IDs, and keeping a row for it
// would cost roughly 260 bytes on every listen; a recovered job simply asks such a source again.
func (s *Sqlite) finishEnrichmentStep(ctx context.Context, c enrichmentClaim, d enrichment.Descriptor, p enrichment.Provider, binding enrichmentBinding,
	target enrichmentTarget, chain enrichment.Chain, result enrichment.Result) (enrichmentAnswer, bool, error) {
	var original ingest.Snapshot
	if err := json.Unmarshal([]byte(c.snapshot), &original); err != nil {
		return enrichmentAnswer{}, false, err
	}
	missingArtists := enrichment.MissingArtistNames(original)
	closed := false
	answer := enrichmentAnswer{status: enrichment.Failed}
	failJob := func(tx *sql.Conn) error {
		closed = true
		return finishClaimedEnrichment(ctx, tx, c, "failed", "")
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
			if err := writeDescriptiveDeclaration(ctx, tx, c, p.Instance(), declaration); err != nil {
				return err
			}
			if err := writeMBIDDeclaration(ctx, tx, c, event, original, missingArtists, p.Instance(), declaration); err != nil {
				return err
			}
		}
		// A partial lookup must be retried even if its useful track claims were saved.
		if result.Retryable && p.Instance() != "" {
			if _, err := tx.ExecContext(ctx, `INSERT OR IGNORE INTO metadata_sources(source,instance) VALUES(?,?)`, d.Source, p.Instance()); err != nil {
				return err
			}
			if _, err := tx.ExecContext(ctx, `INSERT OR IGNORE INTO metadata_retries(profile_id,occurrence_id,source_id) SELECT ?,?,id FROM metadata_sources WHERE source=? AND instance=?`, c.profile, c.occurrence, d.Source, p.Instance()); err != nil {
				return err
			}
		} else if status == enrichment.Found || status == enrichment.Absent {
			if _, err := tx.ExecContext(ctx, `DELETE FROM metadata_retries WHERE profile_id=? AND occurrence_id=? AND source_id IN (SELECT id FROM metadata_sources WHERE source=? AND instance=?)`, c.profile, c.occurrence, d.Source, p.Instance()); err != nil {
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
		answer = enrichmentAnswer{status: status, instance: instance}
		if status == enrichment.Absent || status == enrichment.Disabled {
			// Drop a failure stored by an earlier pass, so the answer on file is always the latest.
			if _, err := tx.ExecContext(ctx, `DELETE FROM enrichment_provider_runs WHERE receipt_id=? AND provider=? AND chain=?`,
				c.receipt, d.ID, string(chain)); err != nil {
				return err
			}
		} else if _, err := tx.ExecContext(ctx, `INSERT INTO enrichment_provider_runs(receipt_id,provider,chain,state,instance,finished_at)
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
	return answer, closed, err
}

// writeDescriptiveDeclaration retains one observation per profile, entity, source and value.
// observed_at is the time the value was first observed; repeated observations do not overwrite it.
func writeDescriptiveDeclaration(ctx context.Context, tx *sql.Conn, c enrichmentClaim, instance string, d enrichment.Declaration) error {
	if instance == "" || d.Source == "" || strings.TrimSpace(d.Value) == "" || d.ObservedAt.IsZero() {
		return nil
	}
	allowed := false
	switch d.Entity {
	case "recording":
		allowed = slices.Contains([]string{"genre", "tag", "recording_year", "isrc", "bpm", "first_recording_date"}, d.Field)
	case "release":
		allowed = slices.Contains([]string{"genre", "tag", "release_date", "first_release_date"}, d.Field)
	case "artist":
		allowed = slices.Contains([]string{"genre", "tag", "artist_origin_country"}, d.Field) && c.artists[d.Name] != 0
	}
	if !allowed {
		return nil
	}
	if _, err := tx.ExecContext(ctx, `INSERT OR IGNORE INTO metadata_sources(source,instance) VALUES(?,?)`, d.Source, instance); err != nil {
		return err
	}
	var artist, occurrence, release any
	switch d.Entity {
	case "recording":
		occurrence = c.occurrence
	case "release":
		var id sql.NullInt64
		if err := tx.QueryRowContext(ctx, `SELECT release_id FROM occurrences WHERE id=?`, c.occurrence).Scan(&id); err != nil {
			return err
		}
		if !id.Valid {
			return nil
		}
		release = id.Int64
	case "artist":
		artist = c.artists[d.Name]
	}
	_, err := tx.ExecContext(ctx, `INSERT OR IGNORE INTO descriptive_metadata(profile_id,artist_id,occurrence_id,release_id,source_id,kind,value,observed_at)
 SELECT ?,?,?,?,id,?,?,? FROM metadata_sources WHERE source=? AND instance=?`, c.profile, artist, occurrence, release,
		d.Field, strings.TrimSpace(d.Value), d.ObservedAt.UTC().Format(time.RFC3339Nano), d.Source, instance)
	return err
}

// writeMBIDDeclaration stores one declared MusicBrainz ID for the recording or for exactly one
// linked artist. Descriptive fields are stored separately by writeDescriptiveDeclaration.
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

// closeEnrichment folds the answers of this pass into the receipt job. The first recorded instance
// wins, as it did when a single provider owned the job.
func (s *Sqlite) closeEnrichment(ctx context.Context, c enrichmentClaim, answers []enrichmentAnswer) error {
	status, instance := enrichment.Disabled, ""
	for _, answer := range answers {
		status = combinedEnrichmentStatus(status, answer.status)
		if instance == "" {
			instance = answer.instance
		}
	}
	return s.withIngestWriter(ctx, func(tx *sql.Conn) error {
		return finishClaimedEnrichment(ctx, tx, c, string(status), instance)
	})
}

// deleteSuccessfulRuns keeps completed jobs compact; found declarations are already durable and
// these rows are needed only until the job finishes or is recovered.
func deleteSuccessfulRuns(ctx context.Context, tx *sql.Conn, receipt string) error {
	_, err := tx.ExecContext(ctx, `DELETE FROM enrichment_provider_runs WHERE receipt_id=? AND state='found'`, receipt)
	return err
}

// finishClaimedEnrichment removes successful runs only for the lease that actually closed the job.
func finishClaimedEnrichment(ctx context.Context, tx *sql.Conn, c enrichmentClaim, state, instance string) error {
	result, err := tx.ExecContext(ctx, `UPDATE enrichment_jobs SET state=?,instance=nullif(?,''),lease=NULL,lease_until=NULL WHERE receipt_id=? AND state='working' AND lease=? AND lease_until>?`, state, instance, c.receipt, c.lease, time.Now().Unix())
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err != nil || n == 0 {
		return err
	}
	return deleteSuccessfulRuns(ctx, tx, c.receipt)
}
