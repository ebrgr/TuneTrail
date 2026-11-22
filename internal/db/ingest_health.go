package db

import (
	"context"
	"errors"
	"time"
)

// IngestHealthStore provides profile diagnostics and manual recovery.
type IngestHealthStore interface {
	IngestHealth(context.Context, int32, ...int) (IngestHealth, error)
	IngestInbox(context.Context, int32, *int64, time.Time, time.Time) ([]IngestInboxGroup, error)
	ReprocessIngest(context.Context, int32, int32, string) error
	ReprocessAllIngest(context.Context, int32, int32) error
	SetIngestReview(context.Context, int32, bool) error
	ResolveIngestReview(context.Context, int32, int32, IngestReviewDecision) error
	ResolveIngestReviewBatch(context.Context, int32, int32, string, string) error
	IngestDiscardPreview(context.Context, int32, string, int) (IngestDiscardPreview, error)
	DiscardIngestPending(context.Context, int32, int32, string, string) error
	ListenProvenance(context.Context, int32, int32, int64, ...string) ([]ListenProvenance, error)
}

type IngestHealth struct {
	Sources                                 []IngestSource
	Decisions                               []IngestDecision
	Relay                                   []IngestDelivery
	Open                                    int
	Pause                                   string
	RelayTracked                            bool
	Failures                                []IngestFailure
	FailureCount                            int
	Reviews                                 []IngestReview
	ReviewCount, ReviewPrevious, ReviewNext int
	ReviewAvailable, ReviewEnabled          bool
	ReviewSame, ReviewDifferent             IngestReviewBatch
	SourcePrevious, SourceNext              int
}

// IngestFailure is the credential-free projection of a parked receipt.
type IngestFailure struct {
	Receipt, Source, Reason string
	Attempts                int64
	Parked                  time.Time
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

// IngestReview contains only displayable evidence scoped to the authorized profile.
type IngestReview struct {
	Receipt, Source, Title, Reason, Precision, Rule string
	Received                                        time.Time
	ListenedAt                                      *time.Time
	Version                                         int64
	CanKeep                                         bool
	Candidates                                      []IngestReviewCandidate
}

type IngestReviewCandidate struct {
	Event, Title string
	At           time.Time
}

type IngestReviewDecision struct {
	Receipt, Candidate, Value string
	Version                   int64
}

// IngestReviewBatch describes the next bounded batch and its receipt-version digest.
type IngestReviewBatch struct {
	Digest string
	Count  int
}

// IngestInboxGroup describes receipts for one key and client in the selected window.
type IngestInboxGroup struct {
	Key                                          int64
	Client, Label                                string
	KeyPresent                                   bool
	Total, Accepted, Duplicate, Invalid, Pending int64
	Missing, Incomplete                          int64
	Examples                                     [2]*IngestInboxExample
}

// IngestInboxExample is the safe display projection of a receipt snapshot.
type IngestInboxExample struct {
	Title, Artist, Client string
	At                    time.Time
}

// ErrIngestDiscardListen directs linked receipts to the normal listen deletion flow.
var ErrIngestDiscardListen = errors.New("use listen deletion for this receipt")

type IngestDiscardPreview struct {
	Receipts   []IngestDiscardReceipt
	Batch      IngestReviewBatch
	Page, Next int
	More       bool
}

type IngestDiscardReceipt struct {
	Receipt, State, Source, Reason, ParkReason, Digest string
	Received                                           time.Time
}
