// Package retry provides scheduling primitives shared by durable workers.
package retry

import "time"

// FullJitter doubles the base for each previous failure and samples up to the cap.
func FullJitter(base, cap time.Duration, failures int64, sample func(int64) int64) time.Duration {
	delay := min(base, cap)
	if delay <= 0 {
		return 0
	}
	for failures > 0 && delay < cap {
		if delay > cap/2 {
			delay = cap
			break
		}
		delay *= 2
		failures--
	}
	return time.Duration(sample(int64(delay)))
}
