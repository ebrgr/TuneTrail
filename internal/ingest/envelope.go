// Package ingest defines the data captured before legacy ingestion.
package ingest

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strconv"
	"strings"
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

// MaxItemBytes limits the serialized receipt snapshot.
const MaxItemBytes = 10 << 10

// Snapshot only carries fields explicitly selected by an adapter.

type Snapshot struct {
	TruncatedFields   int            `json:"truncated_fields,omitempty"`
	IgnoredAliasLimit int            `json:"ignored_alias_limit,omitempty"`
	NativeAliases     *NativeAliases `json:"native_aliases,omitempty"`

	TrackMBID            string            `json:"track_mbid,omitempty"`
	DiscNumber           int               `json:"disc_number,omitempty"`
	TrackNumber          int               `json:"track_number,omitempty"`
	MBIDSource           string            `json:"mbid_source,omitempty"`
	MBIDDeclaredAt       string            `json:"mbid_declared_at,omitempty"`
	CatalogTrackID       int32             `json:"catalog_track_id,omitempty"`
	ArtistMBIDs          []string          `json:"artist_mbids,omitempty"`
	MappedArtistMBIDs    []string          `json:"mapped_artist_mbids,omitempty"`
	ReleaseGroupMBID     string            `json:"release_group_mbid,omitempty"`
	ArtistNames          []string          `json:"artist_names,omitempty"`
	Client               string            `json:"client,omitempty"`
	ClientVersion        string            `json:"client_version,omitempty"`
	MappedRecording      string            `json:"mapped_recording_mbid,omitempty"`
	ReleaseMBID          string            `json:"release_mbid,omitempty"`
	MappedReleaseMBID    string            `json:"mapped_release_mbid,omitempty"`
	Artist               string            `json:"artist"`
	Title                string            `json:"title"`
	Release              string            `json:"release,omitempty"`
	Recording            string            `json:"recording_mbid,omitempty"`
	DurationPlayed       *int64            `json:"duration_played,omitempty"`
	DurationMS           int64             `json:"duration_ms,omitempty"`
	TimeSemantics        string            `json:"time_semantics,omitempty"`
	OriginalTime         string            `json:"original_time,omitempty"`
	ArtistMbidMappings   []ArtistMapping   `json:"artist_mbid_mappings,omitempty"`
	NativeReleaseImage   string            `json:"native_release_image,omitempty"`
	NativeArtistImages   map[string]string `json:"native_artist_images,omitempty"`
	NativeVariousArtists bool              `json:"native_various_artists,omitempty"`
	OwnerImport          bool              `json:"owner_import,omitempty"`
	SkipCacheImage       bool              `json:"skip_cache_image,omitempty"`
}

// NativeAliases carries explicit secondary names from a Koito v1 file.
type NativeAliases struct {
	Artists map[string][]string `json:"artists,omitempty"`
	Release []string            `json:"release,omitempty"`
	Track   []string            `json:"track,omitempty"`
}

// DeclaredMBIDProvenance recognizes reserved lookup and source-event declarations.
func DeclaredMBIDProvenance(source, at string) (string, string) {
	switch source {
	case "navidrome-subsonic", "listenbrainz-lookup", "source-event":
		if _, err := time.Parse(time.RFC3339Nano, at); err != nil {
			at = ""
		}
		return source, at
	default:
		return "", ""
	}
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
	APIKeyID               int64 // Internal credential ID, zero for unattributed receipts.
	ProfileID              int32
	Source, Operation      string
	ReceivedAt             time.Time
	ClientKey              string
	FileDigest, FileFormat string
	Items                  []Item
}

type Result struct {
	IgnoredMBIDConflicts  int
	IgnoredAliasConflicts int
	DeliveryPolicy        string
	ReceiptID             string
	Decision              string
	Pending               bool
	Replay                bool
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
	// Keep the import report counter in memory if the required fields filled the envelope.
	if len(data) > MaxItemBytes && snapshot.NativeAliases == nil && snapshot.IgnoredAliasLimit != 0 {
		snapshot.IgnoredAliasLimit = 0
		data, _ = json.Marshal(snapshot)
	}
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

// OccurrenceTime keeps source precision and offset while normalizing the recorded instant.
func (i Item) OccurrenceTime() (at time.Time, precision, offset, semantics string) {
	at, precision, semantics = i.ListenedAt, "second", "unknown"
	switch i.Snapshot.TimeSemantics {
	case "start", "end":
		semantics = i.Snapshot.TimeSemantics
	}
	raw := i.Snapshot.OriginalTime
	if raw == "" || at.IsZero() {
		return
	}
	clock := raw
	if parsed, err := time.Parse(time.RFC3339Nano, raw); err == nil {
		at = parsed
		clock = raw[11:]
		if strings.HasSuffix(raw, "Z") {
			offset = "Z"
			clock = strings.TrimSuffix(clock, "Z")
		} else if len(raw) >= 6 {
			offset = raw[len(raw)-6:]
			clock = strings.TrimSuffix(clock, offset)
		}
	}
	whole, fraction, ok := strings.Cut(clock, ".")
	if !ok {
		return
	}
	if len(fraction) == 0 {
		return
	}
	for _, digit := range fraction {
		if digit < '0' || digit > '9' {
			return
		}
	}
	switch {
	case len(fraction) <= 3:
		precision = "millisecond"
	case len(fraction) <= 6:
		precision = "microsecond"
	default:
		precision = "nanosecond"
	}
	if seconds, err := strconv.ParseInt(whole, 10, 64); err == nil {
		digits := fraction[:min(len(fraction), 9)]
		nanos, _ := strconv.ParseInt(digits+strings.Repeat("0", 9-len(digits)), 10, 64)
		if strings.HasPrefix(whole, "-") {
			nanos = -nanos
		}
		at = time.Unix(seconds, nanos)
	}
	return
}

// FileReceipt is a durable ordinal checkpoint that can be read without staging.
type FileReceipt struct {
	IgnoredMBIDConflicts  int
	IgnoredAliasConflicts int
	Ordinal               int
	ReceiptID             string
	Snapshot              Snapshot
	ListenedAt            time.Time
	Precision             string
	Pending               bool
}
