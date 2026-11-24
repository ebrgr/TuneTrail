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
type Request struct {
	ProfileID int32
	Snapshot  ingest.Snapshot
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
