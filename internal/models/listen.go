package models

import (
	"time"
)

// a Listen is the same thing as a 'scrobble' but i despise the word scrobble so i will not use it
type Listen struct {
	EventID string      `json:"-"`
	Time    time.Time   `json:"time"`
	Track   SimpleTrack `json:"track"`
	// Client identifies the application that submitted this listen.
	Client string `json:"client,omitempty"`
}
