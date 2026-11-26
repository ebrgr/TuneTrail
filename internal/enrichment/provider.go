// Package enrichment defines scoped external metadata declarations.
package enrichment

import (
	"context"
	"time"

	"github.com/ebrgr/TuneTrail/internal/ingest"
)

// Status describes a completed lookup independently of the ingest state.
type Status string

const (
	Found     Status = "found"
	Absent    Status = "absent"
	Ambiguous Status = "ambiguous"
	Disabled  Status = "disabled"
	Failed    Status = "failed"
)

// Declaration records a provider's claim, not independently verified identity.
type Declaration struct {
	Entity     string
	Name       string
	Field      string
	Value      string
	Source     string
	ObservedAt time.Time
}

// Request is scoped to the owner and provider instance for caching and storage.
// The fields after Snapshot are filled by the chain runner (ISSUE-214); they are excluded from
// JSON so providers that hash the request for their cache keep the same keys.
type Request struct {
	ProfileID int32
	Snapshot  ingest.Snapshot
	// Chain names the question being asked.
	Chain Chain `json:"-"`
	// Credential and Generation bind a token-bound provider to the approved credential of the
	// profile that owns the receipt. The credential is never logged or stored.
	Credential string `json:"-"`
	Generation int64  `json:"-"`
	// Recording and Artists list the fields a token-bound provider must still resolve, computed
	// from local evidence before the call.
	Recording bool     `json:"-"`
	Artists   []string `json:"-"`
	// Prior holds the MusicBrainz IDs that other sources already stored for this listen's
	// recording and artists (Entity, Name, Field, Value and Source only). A provider reads it to
	// leave alone the fields that are already resolved: the chain stops per field, not per receipt.
	// A provider returns Absent without network use when nothing it provides is still open.
	Prior []Declaration `json:"-"`
}

// Result carries field-level claims; this provider returns only missing identifiers.
type Result struct {
	Status       Status
	Declarations []Declaration
}

// Provider isolates source-specific lookup from persistence and reconciliation.
type Provider interface {
	Lookup(context.Context, Request) Result
	Instance() string
}
