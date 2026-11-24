package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/ebrgr/TuneTrail/internal/db"
	"github.com/ebrgr/TuneTrail/internal/models"
)

const exportPageQuery = `
	SELECT l.event_id, COALESCE(a.source_event_id,l.event_id),
	       COALESCE(a.namespace,n.namespace), n.namespace, COALESCE(a.origin,l.origin),
	       COALESCE(a.provenance, (
	           SELECT json_group_array(json_object('receipt_id',e.receipt_id,'source',e.source_type,
	               'received_at',e.received_at,'original_time',e.original_time,
		               'time_precision',e.time_precision,'rule_version',e.rule_version,
	               'external_namespace',e.external_namespace,'external_event_id',e.external_event_id,
	               'resolution_version',e.resolution_version,'decision',e.duplicate_decision,
	               'reason_codes',json(e.reason_codes)))
	           FROM ingest_events e WHERE (e.profile_id,e.event_id)=(l.profile_id,l.event_id)
	       )),
	       l.listened_at, l.client, o.id, o.recording_id, o.presented_title, o.legacy_track_id,
	       COALESCE(o.track_mbid,''), COALESCE(o.disc_number,0), COALESCE(o.track_number,0),
	       rec.musicbrainz_id, rec.mbid_source, COALESCE(t.duration,0), l.duration_played,
	       COALESCE(o.release_id,0), r.musicbrainz_id, r.image_source, COALESCE(r.various_artists,0)
	FROM listens l
	JOIN listen_export_order x ON x.event_id=l.event_id
	JOIN occurrences o ON l.track_id=o.id
	JOIN recordings rec ON rec.id=o.recording_id
	JOIN archive_identity n ON n.singleton=1
	LEFT JOIN tracks t ON t.id=o.legacy_track_id
	LEFT JOIN releases r ON r.id=o.release_id
	LEFT JOIN archive_events a ON (a.profile_id,a.event_id)=(l.profile_id,l.event_id)
	WHERE l.profile_id=? AND l.event_id>? AND x.sequence<=?
	ORDER BY l.event_id
	LIMIT ?`

// GetExportPage seeks inside one profile's event index, including standalone occurrences.
func (s *Sqlite) GetExportPage(ctx context.Context, opts db.GetExportPageOpts) ([]*db.ExportItem, error) {
	if opts.ProfileID <= 0 || opts.Limit <= 0 {
		return nil, db.ErrInvalidInput
	}
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	// Freeze membership against inserts in the same read snapshot as the first page.
	ceiling := opts.ThroughSequence
	if ceiling == 0 {
		if err := tx.QueryRowContext(ctx, `SELECT coalesce(max(sequence),0) FROM listen_export_order`).Scan(&ceiling); err != nil {
			return nil, fmt.Errorf("GetExportPage: read ceiling: %w", err)
		}
	}
	rows, err := tx.QueryContext(ctx, exportPageQuery, opts.ProfileID, opts.AfterEventID, ceiling, opts.Limit)
	if err != nil {
		return nil, fmt.Errorf("GetExportPage: %w", err)
	}
	type presentation struct {
		legacy sql.NullInt32
		title  sql.NullString
	}
	var items []*db.ExportItem
	var presentations []presentation
	for rows.Next() {
		item := db.ExportItem{ThroughSequence: ceiling}
		var p presentation
		var listenedAt int64
		var client, provenance string
		var trackMBID, releaseMBID, releaseImageSrc sql.NullString
		// Preserve NULL separately from an explicit zero in the export.
		if err := rows.Scan(&item.Cursor, &item.EventID, &item.EventNamespace, &item.CatalogNamespace, &item.Origin, &provenance,
			&listenedAt, &client, &item.TrackID, &item.RecordingID, &p.title, &p.legacy,
			&item.TrackReference, &item.DiscNumber, &item.TrackNumber,
			&trackMBID, &item.MBIDSource, &item.TrackDuration, &item.DurationPlayed,
			&item.ReleaseID, &releaseMBID, &releaseImageSrc, &item.VariousArtists); err != nil {
			rows.Close()
			return nil, fmt.Errorf("GetExportPage: scan: %w", err)
		}
		item.ListenedAt = time.Unix(listenedAt, 0).UTC()
		item.Client = &client
		item.Provenance = []byte(provenance)
		item.TrackMbid = parseNullableUUID(trackMBID)
		item.ReleaseMbid = parseNullableUUID(releaseMBID)
		item.ReleaseImageSource = releaseImageSrc.String
		items = append(items, &item)
		presentations = append(presentations, p)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, err
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	// A page commonly contains repeated plays of the same occurrence and release.
	occurrences := make(map[int32]*db.ExportItem)
	releases := make(map[int32]*db.ExportItem)
	for i, item := range items {
		p := presentations[i]
		// Profile claims are used only when all recorded provider instances agree.
		if item.TrackMbid == nil {
			var mbid, source, observed sql.NullString
			err := tx.QueryRowContext(ctx, `SELECT min(c.mbid),min(c.source),min(c.observed_at) FROM enrichment_recording_claims c
				WHERE c.profile_id=? AND c.occurrence_id=? AND NOT EXISTS (
				 SELECT 1 FROM ingest_events e WHERE e.profile_id=c.profile_id AND e.event_id=?
				 AND (coalesce(json_extract(e.original_snapshot,'$.recording_mbid'),'')<>''
				 OR coalesce(json_extract(e.original_snapshot,'$.mapped_recording_mbid'),'')<>''))
				HAVING count(DISTINCT c.mbid)=1`, opts.ProfileID, item.TrackID, item.Cursor).Scan(&mbid, &source, &observed)
			if err != nil && err != sql.ErrNoRows {
				return nil, err
			}
			if err == nil {
				item.TrackMbid = parseNullableUUID(mbid)
				item.MBIDSource = source.String
				item.MBIDDeclaredAt = observed.String
			}
		}
		if existing := occurrences[item.TrackID]; existing != nil {
			item.TrackAliases = existing.TrackAliases
			item.ReleaseAliases = existing.ReleaseAliases
			item.ReleaseArtists = existing.ReleaseArtists
			item.Artists = existing.Artists
			continue
		}
		occurrences[item.TrackID] = item
		if p.legacy.Valid {
			item.TrackAliases, err = queryEntityAliases(ctx, tx, "track_aliases", "track_id", p.legacy.Int32)
			if err != nil {
				return nil, err
			}
			item.Artists, err = exportArtists(ctx, tx, "artist_tracks", "track_id", p.legacy.Int32)
			if err != nil {
				return nil, err
			}
		}
		if p.title.Valid {
			for j := range item.TrackAliases {
				item.TrackAliases[j].Primary = false
			}
			item.TrackAliases = append([]models.Alias{{Alias: p.title.String, Primary: true}}, item.TrackAliases...)
		}
		if item.ReleaseID != 0 {
			if existing := releases[item.ReleaseID]; existing != nil {
				item.ReleaseAliases = existing.ReleaseAliases
				item.ReleaseArtists = existing.ReleaseArtists
			} else {
				item.ReleaseAliases, err = queryEntityAliases(ctx, tx, "release_aliases", "release_id", item.ReleaseID)
				if err != nil {
					return nil, err
				}
				item.ReleaseArtists, err = exportArtists(ctx, tx, "artist_releases", "release_id", item.ReleaseID)
				if err != nil {
					return nil, err
				}
				releases[item.ReleaseID] = item
			}
		}
		if !p.legacy.Valid {
			credits, err := tx.QueryContext(ctx, `SELECT name FROM occurrence_credits WHERE occurrence_id=? ORDER BY position`, item.TrackID)
			if err != nil {
				return nil, err
			}
			for credits.Next() {
				var name string
				if err := credits.Scan(&name); err != nil {
					credits.Close()
					return nil, err
				}
				item.Artists = append(item.Artists, models.ArtistWithFullAliases{Name: name, IsPrimary: len(item.Artists) == 0, Aliases: []models.Alias{{Alias: name, Primary: true}}})
			}
			if err := credits.Err(); err != nil {
				credits.Close()
				return nil, err
			}
			credits.Close()
		}
	}
	return items, tx.Commit()
}
