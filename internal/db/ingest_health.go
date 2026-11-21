package db

import (
	"context"
	"time"
)

// IngestHealthStore exposes only the projections used in profile management.
type IngestHealthStore interface {
	IngestHealth(context.Context, int32) (IngestHealth, error)
	ListenProvenance(context.Context, int32, int32, int64) ([]ListenProvenance, error)
}

type IngestHealth struct {
	Sources      []IngestSource
	Decisions    []IngestDecision
	Relay        []IngestDelivery
	Open         int
	Pause        string
	RelayTracked bool
}

type IngestSource struct {
	Integration, Source, Client, Precision, Delivery string
	Received                                         time.Time
}

type IngestDecision struct {
	Mode, Value, Reason, Rule, Precision, CandidateCount string
	DeltaSeconds, RecordingMatch, ReleaseDiffers         string
	At                                                   time.Time
	TrustedExternalID                                    bool
}

type IngestDelivery struct {
	State, Reason string
	Attempts      int64
}

type ListenProvenance struct {
	Origin, Precision, Delivery, Reason, Rule string
	Received                                  *time.Time
}
