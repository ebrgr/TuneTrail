package listenbrainz

import (
	"bytes"
	"fmt"
	"math"
	"strconv"
)

// UnixSeconds is a listened_at timestamp. ListenBrainz documents an integer, but some
// clients and exports send a decimal (Koito PR #373); the fraction is truncated.
type UnixSeconds int64

func (u *UnixSeconds) UnmarshalJSON(b []byte) error {
	b = bytes.TrimSpace(b)
	if string(b) == "null" {
		return nil
	}
	if n, err := strconv.ParseInt(string(b), 10, 64); err == nil {
		*u = UnixSeconds(n)
		return nil
	}
	f, err := strconv.ParseFloat(string(b), 64)
	if err != nil || math.IsNaN(f) || math.IsInf(f, 0) || f < math.MinInt64 || f >= math.MaxInt64 {
		return fmt.Errorf("listened_at: invalid timestamp %s", b)
	}
	*u = UnixSeconds(math.Floor(f))
	return nil
}
