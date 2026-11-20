// Package ingest defines the data captured before legacy ingestion.
package ingest

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strconv"
	"time"
)

const (
	LegacyDigestVersion = "sha256-json-v1"
	DigestVersion       = "sha256-json-v2"
)

// These source types own temporary shadow integration identities.
const (
	ShadowSourceListenBrainz = "listenbrainz"
	ShadowSourceFile         = "file"
)

// Snapshot only carries fields explicitly selected by an adapter.
type Snapshot struct {
	TrackMBID          string          `json:"track_mbid,omitempty"`
	DiscNumber         int             `json:"disc_number,omitempty"`
	TrackNumber        int             `json:"track_number,omitempty"`
	MBIDSource         string          `json:"mbid_source,omitempty"`
	MBIDDeclaredAt     string          `json:"mbid_declared_at,omitempty"`
	CatalogTrackID     int32           `json:"catalog_track_id,omitempty"`
	ArtistMBIDs        []string        `json:"artist_mbids,omitempty"`
	MappedArtistMBIDs  []string        `json:"mapped_artist_mbids,omitempty"`
	ReleaseGroupMBID   string          `json:"release_group_mbid,omitempty"`
	ArtistNames        []string        `json:"artist_names,omitempty"`
	Client             string          `json:"client,omitempty"`
	ClientVersion      string          `json:"client_version,omitempty"`
	MappedRecording    string          `json:"mapped_recording_mbid,omitempty"`
	ReleaseMBID        string          `json:"release_mbid,omitempty"`
	MappedReleaseMBID  string          `json:"mapped_release_mbid,omitempty"`
	Artist             string          `json:"artist"`
	Title              string          `json:"title"`
	Release            string          `json:"release,omitempty"`
	Recording          string          `json:"recording_mbid,omitempty"`
	DurationMS         int64           `json:"duration_ms,omitempty"`
	TimeSemantics      string          `json:"time_semantics,omitempty"`
	OriginalTime       string          `json:"original_time,omitempty"`
	ArtistMbidMappings []ArtistMapping `json:"artist_mbid_mappings,omitempty"`
	SkipCacheImage     bool            `json:"skip_cache_image,omitempty"`
}

// ArtistMapping preserves a named artist credit and its typed identifier.
type ArtistMapping struct {
	Name string `json:"name"`
	MBID string `json:"mbid"`
}

type Item struct {
	Ordinal                       int
	Snapshot                      Snapshot
	ListenedAt                    time.Time
	ExternalNamespace, ExternalID string
}

type Batch struct {
	ProfileID              int32
	Source, Operation      string
	ReceivedAt             time.Time
	ClientKey              string
	FileDigest, FileFormat string
	Items                  []Item
}

type Result struct {
	ReceiptID string
	Decision  string
	Pending   bool
	Replay    bool
}

// JobMode pins admission and the resolver variant for every page of an import.
type JobMode struct {
	Active  bool
	Variant string
}

// Admission selects active mode atomically or admits pages using a pinned import mode.
type Admission interface {
	AdmitIngest(context.Context, Batch, bool, *JobMode, error) (bool, []Result, error)
	BeginImportIngest(context.Context, int32, bool) (JobMode, error)
}

type Receiver interface {
	ReceiveBatch(context.Context, Batch) ([]Result, error)
}

// ErrUnavailable marks storage failures for which callers must withhold an ACK.
type ErrUnavailable struct{ Cause error }

func (e *ErrUnavailable) Error() string { return "receipt storage unavailable: " + e.Cause.Error() }
func (e *ErrUnavailable) Unwrap() error { return e.Cause }

// Digest hashes a canonical, sanitized representation.
func Digest(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

// Encode preserves the absence of a source timestamp in the snapshot digest.
func (i Item) Encode() ([]byte, string) {
	snapshot := i.Snapshot
	snapshot.OriginalTime = i.OriginalTime()
	data, _ := json.Marshal(snapshot)
	return data, Digest(data)
}

// LegacyDigest reproduces the v1 projection for a keyed retry against a v1 receipt.
func (i Item) LegacyDigest(source, fileFormat string) string {
	// The v1 writer never admitted a native receipt with a fixed track ID.
	i.Snapshot.CatalogTrackID = 0
	i.Snapshot.ArtistMbidMappings = nil
	i.Snapshot.SkipCacheImage = false
	// Non-LBZ file adapters populated these association fields only from v2.
	if source == ShadowSourceFile && fileFormat != "listenbrainz" {
		i.Snapshot.ArtistMBIDs = nil
		i.Snapshot.ReleaseMBID = ""
		i.Snapshot.ReleaseGroupMBID = ""
	}
	_, digest := i.Encode()
	return digest
}

// OriginalTime returns the timestamp as supplied by the adapter, when present.
func (i Item) OriginalTime() string {
	if i.Snapshot.OriginalTime != "" || i.ListenedAt.IsZero() {
		return i.Snapshot.OriginalTime
	}
	return strconv.FormatInt(i.ListenedAt.Unix(), 10)
}

// FileReceipt is a durable ordinal checkpoint that can be read without staging.
type FileReceipt struct {
	Ordinal    int
	ReceiptID  string
	Snapshot   Snapshot
	ListenedAt time.Time
	Precision  string
	Pending    bool
}
