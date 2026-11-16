package db

import "errors"

// ErrNotFound is returned by Store methods when a queried row does not exist.
var ErrNotFound = errors.New("not found")

// ErrInvalidInput marks identities or values rejected before mutation.
var ErrInvalidInput = errors.New("invalid input")

// ErrConflict marks a stale credential or confirmation snapshot.
var ErrConflict = errors.New("data changed")

// ErrAmbiguousTrack marks multiple tracks matching the supplied alias and credits.
var ErrAmbiguousTrack = errors.New("ambiguous track identity")

// ErrTrackIdentityConflict marks incompatible recording IDs for a matched track.
var ErrTrackIdentityConflict = errors.New("conflicting track identity")
