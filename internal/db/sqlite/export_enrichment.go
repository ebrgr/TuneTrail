package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"sort"
	"strconv"
	"time"

	"github.com/ebrgr/TuneTrail/internal/db"
	"github.com/ebrgr/TuneTrail/internal/ingest"
	"github.com/ebrgr/TuneTrail/internal/models"
	"github.com/google/uuid"
)

type exportDeclaration struct {
	id         uuid.UUID
	source, at string
}

type exportObservation struct {
	snapshot   ingest.Snapshot
	receivedAt int64
}

// rememberObservation uses a deterministic timestamp when an event has duplicate receipts.
func rememberObservation(known map[uuid.UUID]int64, id uuid.UUID, at int64) {
	if old, ok := known[id]; !ok || at < old {
		known[id] = at
	}
}

// uniqueCreditName prevents identical names from assigning an ID to multiple artists.
func uniqueCreditName(credits []models.ArtistWithFullAliases, index int, name string) bool {
	for i, credit := range credits {
		if i == index {
			continue
		}
		if credit.Name == name {
			return false
		}
		for _, alias := range credit.Aliases {
			if alias.Alias == name {
				return false
			}
		}
	}
	return true
}

type exportClaimKey struct {
	event  string
	artist int64 // Zero denotes the event's recording.
}

// exportClaims reads only declarations supported by an event on this page.
func exportClaims(ctx context.Context, tx *sql.Tx, profile int32, events string) (map[exportClaimKey][]models.MBIDDeclaration, error) {
	rows, err := tx.QueryContext(ctx, `SELECT event_id,coalesce(artist_id,0),value,source,observed_at FROM metadata_values
		WHERE profile_id=? AND kind='mbid' AND release_id IS NULL AND event_id IN (SELECT value FROM json_each(?))
		ORDER BY event_id,artist_id,observed_at,source,instance,value`, profile, events)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make(map[exportClaimKey][]models.MBIDDeclaration)
	for rows.Next() {
		var key exportClaimKey
		var raw, source, at string
		if err := rows.Scan(&key.event, &key.artist, &raw, &source, &at); err != nil {
			return nil, err
		}
		id, err := uuid.Parse(raw)
		if err != nil || id == uuid.Nil {
			continue
		}
		// Keep source instances distinct without exposing their credential-derived keys.
		result[key] = append(result[key], models.MBIDDeclaration{
			MBID: id, Source: source, DeclaredAt: at, Instance: strconv.Itoa(len(result[key]) + 1),
		})
	}
	return result, rows.Err()
}

// singleExportClaim returns the first observation only if all observations agree.
func singleExportClaim(values []models.MBIDDeclaration) (exportDeclaration, bool) {
	if len(values) == 0 {
		return exportDeclaration{}, false
	}
	first := values[0]
	for _, value := range values[1:] {
		if value.MBID != first.MBID {
			return exportDeclaration{}, false
		}
	}
	return exportDeclaration{id: first.MBID, source: first.Source, at: first.DeclaredAt}, true
}

// applyExportDeclarations reads each page in batches and overlays claims per event.
func applyExportDeclarations(ctx context.Context, tx *sql.Tx, profile int32, items []*db.ExportItem) error {
	if len(items) == 0 {
		return nil
	}
	events := make([]string, 0, len(items))
	for _, item := range items {
		events = append(events, item.Cursor)
	}
	encoded, err := json.Marshal(events)
	if err != nil {
		return err
	}
	claims, err := exportClaims(ctx, tx, profile, string(encoded))
	if err != nil {
		return err
	}
	rows, err := tx.QueryContext(ctx, `SELECT event_id,original_snapshot,received_at FROM ingest_events WHERE profile_id=? AND event_id IN (SELECT value FROM json_each(?))`, profile, string(encoded))
	if err != nil {
		return err
	}
	observed := make(map[string][]exportObservation)
	for rows.Next() {
		var event, raw string
		var at int64
		if err := rows.Scan(&event, &raw, &at); err != nil {
			rows.Close()
			return err
		}
		var snapshot ingest.Snapshot
		if err := json.Unmarshal([]byte(raw), &snapshot); err != nil {
			rows.Close()
			return err
		}
		observed[event] = append(observed[event], exportObservation{snapshot: snapshot, receivedAt: at})
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return err
	}
	if err := rows.Close(); err != nil {
		return err
	}
	for _, item := range items {
		snapshots := observed[item.Cursor]
		item.RecordingDeclarations = append([]models.MBIDDeclaration(nil), claims[exportClaimKey{event: item.Cursor}]...)
		received := make(map[uuid.UUID]int64)
		for _, observation := range snapshots {
			for _, raw := range []string{observation.snapshot.Recording, observation.snapshot.MappedRecording} {
				if id := uuidFromSnapshot(raw); id != uuid.Nil {
					rememberObservation(received, id, observation.receivedAt)
				}
			}
		}
		var recordingIDs []string
		for id := range received {
			recordingIDs = append(recordingIDs, id.String())
		}
		sort.Strings(recordingIDs)
		for _, raw := range recordingIDs {
			id := uuid.MustParse(raw)
			if item.TrackMbid != nil && *item.TrackMbid == id {
				continue
			}
			item.RecordingDeclarations = append(item.RecordingDeclarations, models.MBIDDeclaration{MBID: id, Source: "source-event", DeclaredAt: time.Unix(received[id], 0).UTC().Format(time.RFC3339Nano), Instance: strconv.Itoa(len(item.RecordingDeclarations) + 1)})
		}
		if item.TrackMbid == nil {
			if len(recordingIDs) == 1 {
				id := uuid.MustParse(recordingIDs[0])
				item.TrackMbid = &id
				item.MBIDSource = "source-event"
				item.MBIDDeclaredAt = time.Unix(received[id], 0).UTC().Format(time.RFC3339Nano)
			} else if claim, ok := singleExportClaim(claims[exportClaimKey{event: item.Cursor}]); ok && len(recordingIDs) == 0 {
				id := claim.id
				item.TrackMbid = &id
				item.MBIDSource = claim.source
				item.MBIDDeclaredAt = claim.at
			}
		}
		// A shared occurrence slice must not carry another event's observations.
		item.Artists = append([]models.ArtistWithFullAliases(nil), item.Artists...)
		for i := range item.Artists {
			artist := &item.Artists[i]
			artist.MBIDDeclarations = append([]models.MBIDDeclaration(nil), claims[exportClaimKey{event: item.Cursor, artist: int64(artist.ID)}]...)
			if artist.ID == 0 {
				continue
			}
			known := make(map[uuid.UUID]int64)
			ambiguous := false
			for _, observation := range snapshots {
				declared := fileArtistDeclarations(observation.snapshot)
				if id := declared[artist.Name]; id != uuid.Nil {
					if uniqueCreditName(item.Artists, i, artist.Name) {
						rememberObservation(known, id, observation.receivedAt)
					} else {
						ambiguous = true
					}
				}
				for _, alias := range artist.Aliases {
					if id := declared[alias.Alias]; id != uuid.Nil {
						if uniqueCreditName(item.Artists, i, alias.Alias) {
							rememberObservation(known, id, observation.receivedAt)
						} else {
							ambiguous = true
						}
					}
				}
			}
			if ambiguous {
				continue
			}
			// Preserve received values even when the shared catalog has another ID.
			var receivedIDs []string
			for id := range known {
				receivedIDs = append(receivedIDs, id.String())
			}
			sort.Strings(receivedIDs)
			for _, raw := range receivedIDs {
				id := uuid.MustParse(raw)
				if artist.MbzID != nil && *artist.MbzID == id {
					continue
				}
				artist.MBIDDeclarations = append(artist.MBIDDeclarations, models.MBIDDeclaration{MBID: id, Source: "source-event", DeclaredAt: time.Unix(known[id], 0).UTC().Format(time.RFC3339Nano), Instance: strconv.Itoa(len(artist.MBIDDeclarations) + 1)})
			}
			if artist.MbzID != nil {
				continue
			}
			if len(known) == 1 {
				for id, at := range known {
					value := id
					artist.MbzID = &value
					artist.MBIDSource = "source-event"
					artist.MBIDDeclaredAt = time.Unix(at, 0).UTC().Format(time.RFC3339Nano)
				}
				continue
			}
			if len(known) > 1 {
				continue
			}
			if claim, ok := singleExportClaim(claims[exportClaimKey{event: item.Cursor, artist: int64(artist.ID)}]); ok {
				id := claim.id
				artist.MbzID = &id
				artist.MBIDSource = claim.source
				artist.MBIDDeclaredAt = claim.at
			}
		}
	}
	return nil
}
