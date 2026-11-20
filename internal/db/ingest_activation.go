package db

import (
	"context"

	"github.com/google/uuid"
)

// IngestActivationPreview reports the aggregate export and confirmation digest for one profile.
type IngestActivationPreview struct {
	ProfileID int32               `json:"profile_id"`
	Active    bool                `json:"active"`
	Review    bool                `json:"review"`
	Digest    string              `json:"digest"`
	Receipts  int64               `json:"receipts"`
	Groups    []IngestShadowGroup `json:"groups"`
}

// IngestShadowGroup carries only aggregated measurements and adapter provenance.
type IngestShadowGroup struct {
	Source        string `json:"source"`
	Adapter       string `json:"adapter_version"`
	Rule          string `json:"rule_version"`
	State         string `json:"state"`
	Receipts      int64  `json:"receipts"`
	Observations  int64  `json:"observations"`
	Details       int64  `json:"details"`
	SnapshotBytes int64  `json:"snapshot_bytes"`
}

// IngestActivationStore requires a live root session for preview and mutation.
type IngestActivationStore interface {
	PreviewIngestActivation(context.Context, uuid.UUID, int32) (IngestActivationPreview, error)
	SetIngestActivation(context.Context, uuid.UUID, int32, bool, bool, string) error
}
