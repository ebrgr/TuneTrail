package sqlite

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/ebrgr/TuneTrail/internal/ingest"
	"github.com/google/uuid"
)

// RelayPayload reconstructs a claimed item from its immutable accepted receipt.
func (s *Sqlite) RelayPayload(ctx context.Context, batch RelayBatch, item RelayItem) ([]byte, error) {
	var version int
	var snapshot, listenedAt string
	err := s.db.QueryRowContext(ctx, `SELECT o.format_version,e.original_snapshot,e.listened_at
		FROM outbox o JOIN ingest_events e ON e.profile_id=o.profile_id AND e.event_id=o.event_id
		JOIN relay_destinations d ON d.profile_id=o.profile_id
		WHERE o.profile_id=? AND o.id=? AND o.work_version=? AND o.lease_owner=?
		  AND d.generation=? AND d.work_version=? AND d.lease_owner=?
		  AND e.mode='active' AND e.state='accepted' AND e.delivery_policy='durable'`,
		batch.Profile, item.ID, item.Version, batch.Owner, batch.Generation, batch.Version, batch.Owner).
		Scan(&version, &snapshot, &listenedAt)
	if err != nil {
		return nil, err
	}
	var original ingest.Snapshot
	if err := json.Unmarshal([]byte(snapshot), &original); err != nil {
		return nil, errors.New("invalid relay snapshot")
	}
	at, err := time.Parse(receiptTime, listenedAt)
	if err != nil {
		return nil, errors.New("invalid relay receipt time")
	}
	return encodeRelayItem(version, original, at.Unix())
}

// encodeRelayItem keeps both wire formats available for already queued items.
func encodeRelayItem(version int, s ingest.Snapshot, at int64) ([]byte, error) {
	metadata := map[string]any{"artist_name": s.Artist, "track_name": s.Title}
	if s.Release != "" {
		metadata["release_name"] = s.Release
	}
	info := map[string]any{}
	if len(s.ArtistNames) > 0 {
		info["artist_names"] = s.ArtistNames
	}
	if s.DurationMS > 0 {
		info["duration_ms"] = s.DurationMS
	}
	if s.Client != "" {
		info["submission_client"] = s.Client
	}
	if s.DiscNumber > 0 {
		info["discnumber"] = s.DiscNumber
	}
	if s.TrackNumber > 0 {
		info["tracknumber"] = s.TrackNumber
	}
	if ids := relayUUIDs(s.ArtistMBIDs); len(ids) > 0 {
		info["artist_mbids"] = ids
	}
	for key, id := range map[string]string{
		"track_mbid": s.TrackMBID, "recording_mbid": s.Recording,
		"release_mbid": s.ReleaseMBID, "release_group_mbid": s.ReleaseGroupMBID,
	} {
		if ids := relayUUIDs([]string{id}); len(ids) > 0 {
			info[key] = ids[0]
		}
	}
	if len(info) > 0 {
		metadata["additional_info"] = info
	}
	mapping := map[string]any{}
	if ids := relayUUIDs(s.MappedArtistMBIDs); len(ids) > 0 {
		mapping["artist_mbids"] = ids
	}
	if ids := relayUUIDs([]string{s.MappedRecording}); len(ids) > 0 {
		mapping["recording_mbid"] = ids[0]
	}
	if ids := relayUUIDs([]string{s.MappedReleaseMBID}); len(ids) > 0 {
		mapping["release_mbid"] = ids[0]
	}
	var credits []map[string]string
	for _, credit := range s.ArtistMbidMappings {
		if ids := relayUUIDs([]string{credit.MBID}); len(ids) > 0 && credit.Name != "" {
			credits = append(credits, map[string]string{"artist_name": credit.Name, "artist_mbid": ids[0]})
		}
	}
	if len(credits) > 0 {
		mapping["artists"] = credits
	}
	if len(mapping) > 0 {
		metadata["mbid_mapping"] = mapping
	}
	item := map[string]any{"listened_at": at, "track_metadata": metadata}
	switch version {
	case 1:
		return json.Marshal(item)
	case 2:
		// V2 permits literal HTML characters in JSON strings. Admission still writes V1.
		var buffer bytes.Buffer
		encoder := json.NewEncoder(&buffer)
		encoder.SetEscapeHTML(false)
		if err := encoder.Encode(item); err != nil {
			return nil, err
		}
		return bytes.TrimSuffix(buffer.Bytes(), []byte("\n")), nil
	default:
		return nil, errors.New("unsupported relay payload format")
	}
}

// relayUUIDs preserves the order of canonical, nonzero identifiers.
func relayUUIDs(values []string) []string {
	var ids []string
	for _, value := range values {
		if id, err := uuid.Parse(value); err == nil && id != uuid.Nil {
			ids = append(ids, id.String())
		}
	}
	return ids
}
