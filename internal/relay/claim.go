package relay

import "errors"

// The claim contract between the durable outbox and its workers lives here so
// that neither side depends on the other (ISSUE-231).

// ErrConfigInvalid identifies non-retryable configuration or permission data.
var ErrConfigInvalid = errors.New("invalid stored relay configuration")

// ErrSourceInvalid identifies persisted source data that cannot be encoded.
var ErrSourceInvalid = errors.New("invalid relay source")

// Batch fences a destination and its items without carrying credentials.
type Batch struct {
	Profile                            int32
	Generation, ConfigVersion, Version int64
	Owner                              string
	Items                              []Item
	PostBudget                         int
}

// Item identifies a versioned outbox claim.
type Item struct {
	ID      string
	Version int64
}
