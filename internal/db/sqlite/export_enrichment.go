package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
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

// exportClaims batches one page of scoped catalog claims and rejects conflicting IDs.
func exportClaims(ctx context.Context, tx *sql.Tx, profile int32, column string, ids []int64) (map[int64]exportDeclaration, map[int64][]models.MBIDDeclaration, error) {
	encoded, err := json.Marshal(ids)
	if err != nil {
		return nil, nil, err
	}
	query := fmt.Sprintf(`SELECT DISTINCT %s,value,source,observed_at,instance FROM metadata_values
		WHERE profile_id=? AND kind='mbid' AND %s IN (SELECT value FROM json_each(?)) ORDER BY %s,observed_at,source,instance,value`, column, column, column)
	rows, err := tx.QueryContext(ctx, query, profile, string(encoded))
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	result := make(map[int64]exportDeclaration)
	all := make(map[int64][]models.MBIDDeclaration)
	conflicts := make(map[int64]bool)
	for rows.Next() {
		var key int64
		var raw, source, at, instance string
		if err := rows.Scan(&key, &raw, &source, &at, &instance); err != nil {
			return nil, nil, err
		}
		id, err := uuid.Parse(raw)
		if err != nil || id == uuid.Nil {
			continue
		}
		// Keep distinct source instances without exposing credential-derived hashes.
		declaration := models.MBIDDeclaration{MBID: id, Source: source, DeclaredAt: at, Instance: strconv.Itoa(len(all[key]) + 1)}
		all[key] = append(all[key], declaration)
		if previous, ok := result[key]; ok {
			if previous.id != id {
				conflicts[key] = true
			}
		} else {
			result[key] = exportDeclaration{id: id, source: source, at: at}
		}
	}
	if err := rows.Err(); err != nil {
		return nil, nil, err
	}
	for key := range conflicts {
		delete(result, key)
	}
	return result, all, nil
}

// applyExportDeclarations reads each page in batches and overlays claims per event.
func applyExportDeclarations(ctx context.Context, tx *sql.Tx, profile int32, items []*db.ExportItem) error {
	if len(items) == 0 {
		return nil
	}
	var recordingsIDs, artists []int64
	var events []string
	for _, item := range items {
		recordingsIDs = append(recordingsIDs, item.RecordingID)
		events = append(events, item.Cursor)
		for _, artist := range item.Artists {
			if artist.ID > 0 {
				artists = append(artists, int64(artist.ID))
			}
		}
	}
	recordings, recordingDeclarations, err := exportClaims(ctx, tx, profile, "recording_id", recordingsIDs)
	if err != nil {
		return err
	}
	artistClaims, artistDeclarations, err := exportClaims(ctx, tx, profile, "artist_id", artists)
	if err != nil {
		return err
	}
	encoded, err := json.Marshal(events)
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
		item.RecordingDeclarations = append([]models.MBIDDeclaration(nil), recordingDeclarations[item.RecordingID]...)
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
			} else if claim, ok := recordings[item.RecordingID]; ok && len(recordingIDs) == 0 {
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
			artist.MBIDDeclarations = append([]models.MBIDDeclaration(nil), artistDeclarations[int64(artist.ID)]...)
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
			if claim, ok := artistClaims[int64(artist.ID)]; ok {
				id := claim.id
				artist.MbzID = &id
				artist.MBIDSource = claim.source
				artist.MBIDDeclaredAt = claim.at
			}
		}
	}
	return nil
}
