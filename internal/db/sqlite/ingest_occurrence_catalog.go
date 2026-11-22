package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strconv"

	"github.com/ebrgr/TuneTrail/internal/ingest"
	"github.com/google/uuid"
)

// resolveOccurrence ranks complete candidate sets under the same lock used to write.
func resolveOccurrence(ctx context.Context, tx *sql.Conn, snapshot ingest.Snapshot, proof occurrenceProof) (occurrenceMatch, []string, error) {
	// Release-less presentations are matched without track position or release proof.
	if snapshot.Release == "" {
		return resolveStandaloneOccurrence(ctx, tx, snapshot)
	}
	var match occurrenceMatch
	if proof.evidence != nil && proof.evidence.ReleaseTitleDiffers {
		// Match the presented metadata without reusing IDs from the mismatched release.
		snapshot.TrackMBID, snapshot.Recording, snapshot.MappedRecording = "", "", ""
	}
	credits, err := json.Marshal(observedCredits(snapshot))
	if err != nil {
		return match, nil, err
	}
	track := snapshot.TrackMBID
	if proof.track != "" {
		track = proof.track
	}
	releases, err := occurrenceReleaseCandidates(ctx, tx, snapshot.Release, proof.release, string(credits))
	if err != nil {
		return match, nil, err
	}
	if len(releases) > 1 {
		// A stored release-track identity can disambiguate identical release labels.
		var trackRelease string
		err := tx.QueryRowContext(ctx, `SELECT coalesce(release_id,0) FROM occurrences WHERE track_mbid=?`, track).Scan(&trackRelease)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return match, nil, err
		}
		if !slices.Contains(releases, trackRelease) {
			for i := range releases {
				releases[i] = "release:" + releases[i]
			}
			return match, releases, nil
		}
		releases = []string{trackRelease}
	}
	if len(releases) == 1 {
		match.release, err = strconv.ParseInt(releases[0], 10, 64)
		if err != nil {
			return match, nil, err
		}
	}
	declaredID := uuidFromSnapshot(snapshot.Recording, snapshot.MappedRecording)
	if proof.ignoreDeclaredRecording {
		declaredID = uuid.Nil
	}
	declared := declaredID.String()
	if proof.recording != "" {
		declared = proof.recording
	}
	if proof.recording != "" || (declaredID != uuid.Nil && snapshot.MBIDSource == "") {
		err = tx.QueryRowContext(ctx, `SELECT r.id FROM recordings r WHERE r.musicbrainz_id=? AND
			(?<>'' OR (r.mbid_source='validated' AND EXISTS (
				SELECT 1 FROM occurrences o JOIN track_aliases t ON t.track_id=o.id
				WHERE o.recording_id=r.id AND t.alias=?
				  AND NOT EXISTS (SELECT 1 FROM json_each(?) c WHERE NOT EXISTS (
				    SELECT 1 FROM artist_tracks at JOIN artist_aliases a ON a.artist_id=at.artist_id
				    WHERE at.track_id=o.id AND a.alias=c.value)))))
			`, declared, proof.recording, snapshot.Title, string(credits)).Scan(&match.recording)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return match, nil, err
		}
	}

	rows, err := tx.QueryContext(ctx, `SELECT o.id,o.recording_id,coalesce(o.release_id,0),coalesce(o.track_mbid,''),
		coalesce(r.musicbrainz_id,''),coalesce(o.disc_number,0),coalesce(o.track_number,0),r.mbid_source='validated',
		(EXISTS (SELECT 1 FROM track_aliases t WHERE t.track_id=o.id AND t.alias=?)
		 OR (o.legacy_track_id IS NULL AND o.presented_title=?))
		AND NOT EXISTS (
		  SELECT 1 FROM json_each(?) c WHERE NOT EXISTS (
		    SELECT 1 FROM artist_tracks t JOIN artist_aliases a ON a.artist_id=t.artist_id
		    WHERE t.track_id=o.id AND a.alias=c.value
		    UNION ALL SELECT 1 FROM occurrence_credits oc WHERE oc.occurrence_id=o.id AND oc.name=c.value))
		FROM occurrences o JOIN recordings r ON r.id=o.recording_id
		WHERE o.release_id=? OR o.track_mbid=?
		ORDER BY o.id`, snapshot.Title, snapshot.Title, string(credits), match.release, track)
	if err != nil {
		return match, nil, err
	}
	defer rows.Close()
	var tiers [4][]occurrenceMatch
	var textConflicts []string
	for rows.Next() {
		var c occurrenceMatch
		if err := rows.Scan(&c.id, &c.recording, &c.release, &c.trackMBID, &c.recordingMBID, &c.disc, &c.position, &c.validated, &c.text); err != nil {
			return match, nil, err
		}
		differentPosition := snapshot.DiscNumber > 0 && c.disc > 0 && snapshot.DiscNumber != c.disc ||
			snapshot.TrackNumber > 0 && c.position > 0 && snapshot.TrackNumber != c.position
		differentRecording := match.recording != 0 && match.recording != c.recording ||
			c.validated && snapshot.MBIDSource == "" && declaredID != uuid.Nil && c.recordingMBID != declared ||
			proof.recording != "" && c.recordingMBID != "" && c.recordingMBID != proof.recording
		byTrack := track != "" && track == c.trackMBID
		byPosition := snapshot.DiscNumber > 0 && snapshot.TrackNumber > 0 && c.disc == snapshot.DiscNumber && c.position == snapshot.TrackNumber
		switch {
		case byTrack:
			if c.release != match.release || differentPosition || differentRecording || !c.text {
				return match, []string{"occurrence:" + strconv.FormatInt(c.id, 10)}, nil
			}
			tiers[0] = append(tiers[0], c)
		case byPosition:
			if differentRecording || (proof.track != "" && c.trackMBID != "" && c.trackMBID != proof.track) || !c.text {
				return match, []string{"occurrence:" + strconv.FormatInt(c.id, 10)}, nil
			}
			tiers[1] = append(tiers[1], c)
		case differentPosition || (proof.track != "" && c.trackMBID != "" && c.trackMBID != proof.track):
			continue
		case match.recording != 0 && match.recording == c.recording:
			tiers[2] = append(tiers[2], c)
		case c.text:
			if differentRecording || (snapshot.TrackMBID != "" && c.trackMBID != "" && snapshot.TrackMBID != c.trackMBID) {
				if len(textConflicts) < 33 {
					textConflicts = append(textConflicts, "occurrence:"+strconv.FormatInt(c.id, 10))
				}
			} else {
				tiers[3] = append(tiers[3], c)
			}
		}
	}
	if err := rows.Err(); err != nil {
		return match, nil, err
	}
	for i, tier := range tiers {
		if i == 3 && len(textConflicts) > 0 {
			return match, textConflicts, nil
		}
		switch len(tier) {
		case 0:
			continue
		case 1:
			return tier[0], nil, nil
		default:
			ids := make([]string, 0, min(len(tier), 33))
			for _, candidate := range tier[:min(len(tier), 33)] {
				ids = append(ids, "occurrence:"+strconv.FormatInt(candidate.id, 10))
			}
			return match, ids, nil
		}
	}
	return match, nil, nil
}

// resolveStandaloneOccurrence matches only release-less presentations with exact,
// ordered credits. A recording MBID narrows identity without supplying an album.
func resolveStandaloneOccurrence(ctx context.Context, tx *sql.Conn, snapshot ingest.Snapshot) (occurrenceMatch, []string, error) {
	var match occurrenceMatch
	credits, err := json.Marshal(observedCredits(snapshot))
	if err != nil {
		return match, nil, err
	}
	mbid := standaloneRecordingMBID(snapshot)
	if mbid != "" {
		err := tx.QueryRowContext(ctx, `SELECT id FROM recordings WHERE musicbrainz_id=?`, mbid).Scan(&match.recording)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return match, nil, err
		}
	}
	rows, err := tx.QueryContext(ctx, `
		WITH credits AS (SELECT key AS position,value AS name FROM json_each(?))
		SELECT o.id,o.recording_id
		FROM occurrences o
		WHERE o.release_id IS NULL AND (?='' OR o.recording_id=?)
		  AND (o.presented_title=? OR EXISTS (
		    SELECT 1 FROM track_aliases t WHERE t.track_id=o.id AND t.alias=?))
		  AND (SELECT count(*) FROM occurrence_credits oc WHERE oc.occurrence_id=o.id)=(SELECT count(*) FROM credits)
		  AND NOT EXISTS (
		    SELECT 1 FROM credits c WHERE NOT EXISTS (
		      SELECT 1 FROM occurrence_credits oc WHERE oc.occurrence_id=o.id AND oc.position=c.position
		      AND (oc.name=c.name OR EXISTS (
		        SELECT 1 FROM artist_aliases a WHERE a.artist_id=oc.artist_id AND a.alias=c.name))))
		ORDER BY o.id LIMIT 33`, string(credits), mbid, match.recording, snapshot.Title, snapshot.Title)
	if err != nil {
		return match, nil, err
	}
	defer rows.Close()
	var candidate occurrenceMatch
	var candidates []string
	for rows.Next() {
		if err := rows.Scan(&candidate.id, &candidate.recording); err != nil {
			return match, nil, err
		}
		candidates = append(candidates, "occurrence:"+strconv.FormatInt(candidate.id, 10))
	}
	if err := rows.Err(); err != nil {
		return match, nil, err
	}
	if len(candidates) == 1 {
		return candidate, nil, nil
	}
	return match, candidates, nil
}

// A mapped MBID remains a hint. A source declaration can identify a recording
// without release proof, but retains the schema's unvalidated provenance.
func standaloneRecordingMBID(snapshot ingest.Snapshot) string {
	id := uuidFromSnapshot(snapshot.Recording)
	if id == uuid.Nil {
		return ""
	}
	return id.String()
}

// occurrenceReleaseCandidates keeps release identity separate from recording identity.
func occurrenceReleaseCandidates(ctx context.Context, tx *sql.Conn, title, mbid, credits string) ([]string, error) {
	if title == "" {
		return nil, nil
	}
	if mbid != "" {
		var id string
		err := tx.QueryRowContext(ctx, `SELECT id FROM releases WHERE musicbrainz_id=?`, mbid).Scan(&id)
		if err == nil {
			return []string{id}, nil
		}
		if !errors.Is(err, sql.ErrNoRows) {
			return nil, err
		}
	}
	rows, err := tx.QueryContext(ctx, `SELECT DISTINCT r.id FROM release_aliases ra JOIN releases r ON r.id=ra.release_id
		WHERE ra.alias=? AND (?='' OR r.musicbrainz_id IS NULL)
			AND EXISTS (SELECT 1 FROM json_each(?) c JOIN artist_aliases a ON a.alias=c.value WHERE
				EXISTS (SELECT 1 FROM artist_releases ar WHERE ar.release_id=r.id AND ar.artist_id=a.artist_id)
				OR EXISTS (SELECT 1 FROM tracks t JOIN artist_tracks at ON at.track_id=t.id WHERE t.release_id=r.id AND at.artist_id=a.artist_id))
		ORDER BY r.id`, title, mbid, credits)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var releases []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		releases = append(releases, id)
	}
	return releases, rows.Err()
}

// createOccurrence installs presentation and the compatibility bridge before linking a listen.
func createOccurrence(ctx context.Context, tx *sql.Conn, snapshot ingest.Snapshot, proof occurrenceProof, match occurrenceMatch) (occurrenceMatch, error) {
	if snapshot.Release == "" {
		// The legacy writer falls back to the track title as the album name.
		// Keep the same listen count with a NULL release, which readers expose as
		// unknown and the archive preserves. Inbox completeness uses the snapshot.
		if match.recording == 0 {
			err := tx.QueryRowContext(ctx, `INSERT INTO recordings(musicbrainz_id) VALUES(nullif(?,'')) RETURNING id`, standaloneRecordingMBID(snapshot)).Scan(&match.recording)
			if err != nil {
				return match, err
			}
		}
		err := tx.QueryRowContext(ctx, `INSERT INTO occurrences(recording_id,presented_title,disc_number,track_number)
			VALUES(?,?,nullif(?,0),nullif(?,0)) RETURNING id`, match.recording, snapshot.Title, snapshot.DiscNumber, snapshot.TrackNumber).Scan(&match.id)
		if err != nil {
			return match, err
		}
		// Keep the observed name even when its exact alias cannot identify one artist.
		for position, name := range observedCredits(snapshot) {
			if _, err := tx.ExecContext(ctx, `INSERT INTO occurrence_credits(occurrence_id,position,name,artist_id)
				VALUES(?,?,?,(SELECT min(artist_id) FROM artist_aliases WHERE alias=? COLLATE BINARY HAVING count(DISTINCT artist_id)=1))`, match.id, position, name, name); err != nil {
				return match, err
			}
		}
		return match, nil
	}
	artists, err := ensureOccurrenceArtists(ctx, tx, observedCredits(snapshot))
	if err != nil {
		return match, err
	}
	if match.release == 0 {
		err = tx.QueryRowContext(ctx, `INSERT INTO releases(musicbrainz_id) VALUES(nullif(?,'')) RETURNING id`, proof.release).Scan(&match.release)
		if err != nil {
			return match, err
		}
		if _, err = tx.ExecContext(ctx, `INSERT INTO release_aliases VALUES(?,?,'Canonical',1)`, match.release, snapshot.Release); err != nil {
			return match, err
		}
		for i, id := range artists {
			if _, err = tx.ExecContext(ctx, `INSERT INTO artist_releases VALUES(?,?,?)`, id, match.release, i == 0); err != nil {
				return match, err
			}
		}
	}
	var primaryMBID any
	if match.recording == 0 && proof.recording != "" {
		primaryMBID = proof.recording
	}
	if err = tx.QueryRowContext(ctx, insertBridgedTrackSQL, primaryMBID, match.release, snapshot.DurationMS/1000).Scan(&match.id); err != nil {
		return match, fmt.Errorf("insert occurrence bridge: %w", err)
	}
	if match.recording != 0 {
		if err = attachBridgedRecording(ctx, tx, match.id, match.recording); err != nil {
			return match, err
		}
	} else if err = tx.QueryRowContext(ctx, `SELECT recording_id FROM occurrences WHERE id=?`, match.id).Scan(&match.recording); err != nil {
		return match, err
	}
	if proof.recording != "" {
		if _, err = tx.ExecContext(ctx, `UPDATE recordings SET mbid_source='validated' WHERE id=? AND musicbrainz_id=?`, match.recording, proof.recording); err != nil {
			return match, err
		}
	}
	if _, err = tx.ExecContext(ctx, `UPDATE occurrences SET track_mbid=nullif(?,''),disc_number=nullif(?,0),track_number=nullif(?,0) WHERE id=?`,
		proof.track, snapshot.DiscNumber, snapshot.TrackNumber, match.id); err != nil {
		return match, err
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO track_aliases VALUES(?,?,'Canonical',1)`, match.id, snapshot.Title); err != nil {
		return match, err
	}
	for i, id := range artists {
		if _, err = tx.ExecContext(ctx, `INSERT INTO artist_tracks VALUES(?,?,?)`, id, match.id, i == 0); err != nil {
			return match, err
		}
	}
	return match, nil
}

// occurrenceCreditConflicts keeps new associations from picking a homonymous artist by row order.
func occurrenceCreditConflicts(ctx context.Context, tx *sql.Conn, names []string) ([]string, error) {
	credits, err := json.Marshal(names)
	if err != nil {
		return nil, err
	}
	rows, err := tx.QueryContext(ctx, `WITH ambiguous AS (
		SELECT a.alias FROM artist_aliases a JOIN json_each(?) c ON a.alias=c.value COLLATE BINARY
		GROUP BY a.alias HAVING count(DISTINCT a.artist_id)>1
	) SELECT DISTINCT a.artist_id FROM artist_aliases a JOIN ambiguous b USING(alias)
	ORDER BY a.artist_id`, string(credits))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, "artist:"+strconv.FormatInt(id, 10))
		if len(ids) == 33 {
			break
		}
	}
	return ids, rows.Err()
}

// ensureOccurrenceArtists reuses exact known names and keeps the observed order.
func ensureOccurrenceArtists(ctx context.Context, tx *sql.Conn, names []string) ([]int64, error) {
	var result []int64
	for _, name := range names {
		var id int64
		var matches int
		err := tx.QueryRowContext(ctx, `SELECT coalesce(min(artist_id),0),count(DISTINCT artist_id)
			FROM artist_aliases WHERE alias=? COLLATE BINARY`, name).Scan(&id, &matches)
		if err != nil {
			return nil, err
		}
		// The receipt resolver routes homonyms to pending_identity before
		// creation under the same writer lock. Reject any ambiguous write here too.
		if matches > 1 {
			return nil, fmt.Errorf("ambiguous occurrence artist alias %q", name)
		}
		if matches == 0 {
			if err = tx.QueryRowContext(ctx, `INSERT INTO artists DEFAULT VALUES RETURNING id`).Scan(&id); err != nil {
				return nil, err
			}
			if _, err = tx.ExecContext(ctx, `INSERT INTO artist_aliases VALUES(?,?,'Canonical',1)`, id, name); err != nil {
				return nil, err
			}
		}
		// Distinct aliases can resolve to the same artist; the first credit stays primary.
		if !slices.Contains(result, id) {
			result = append(result, id)
		}
	}
	return result, nil
}
