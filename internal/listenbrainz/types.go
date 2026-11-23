// Package listenbrainz contains shared ListenBrainz payload types.
package listenbrainz

import "encoding/json"

type LbzSubmitListenPayload struct {
	OriginalListenedAt string       `json:"-"`
	ListenedAt         UnixSeconds  `json:"listened_at,omitempty"`
	TrackMeta          LbzTrackMeta `json:"track_metadata"`
}

// UnmarshalJSON preserves the original numeric spelling alongside the compatible seconds value.
func (p *LbzSubmitListenPayload) UnmarshalJSON(data []byte) error {
	type payload LbzSubmitListenPayload
	var value struct {
		*payload
		Raw json.RawMessage `json:"listened_at"`
	}
	value.payload = (*payload)(p)
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	if len(value.Raw) == 0 || string(value.Raw) == "null" {
		return nil
	}
	if err := json.Unmarshal(value.Raw, &p.ListenedAt); err != nil {
		return err
	}
	p.OriginalListenedAt = string(value.Raw)
	return nil
}

type LbzTrackMeta struct {
	ArtistName     string            `json:"artist_name"` // required
	TrackName      string            `json:"track_name"`  // required
	ReleaseName    string            `json:"release_name,omitempty"`
	MBIDMapping    LbzMBIDMapping    `json:"mbid_mapping"`
	AdditionalInfo LbzAdditionalInfo `json:"additional_info,omitempty"`
}
type LbzArtist struct {
	ArtistMBID string `json:"artist_mbid"`
	ArtistName string `json:"artist_credit_name"`
}
type LbzMBIDMapping struct {
	ReleaseMBID   string      `json:"release_mbid"`
	RecordingMBID string      `json:"recording_mbid"`
	ArtistMBIDs   []string    `json:"artist_mbids"`
	Artists       []LbzArtist `json:"artists"`
}

type LbzAdditionalInfo struct {
	TrackMBID               string          `json:"track_mbid,omitempty"`
	DiscNumber              json.RawMessage `json:"discnumber,omitempty"`
	TrackNumber             json.RawMessage `json:"tracknumber,omitempty"`
	MBIDSource              string          `json:"mbid_source,omitempty"`
	MBIDDeclaredAt          string          `json:"mbid_declared_at,omitempty"`
	MediaPlayer             string          `json:"media_player,omitempty"`
	SubmissionClient        string          `json:"submission_client,omitempty"`
	SubmissionClientVersion string          `json:"submission_client_version,omitempty"`
	ReleaseMBID             string          `json:"release_mbid,omitempty"`
	ReleaseGroupMBID        string          `json:"release_group_mbid,omitempty"`
	ArtistMBIDs             []string        `json:"artist_mbids,omitempty"`
	ArtistNames             []string        `json:"artist_names,omitempty"`
	RecordingMBID           string          `json:"recording_mbid,omitempty"`
	DurationPlayed          json.RawMessage `json:"duration_played,omitempty"`
	DurationPlayedMs        json.RawMessage `json:"duration_played_ms,omitempty"`
	DurationMs              int32           `json:"duration_ms,omitempty"`
	Duration                int32           `json:"duration,omitempty"`
	Tags                    []string        `json:"tags,omitempty"`
	AlbumArtist             string          `json:"albumartist,omitempty"`
}
