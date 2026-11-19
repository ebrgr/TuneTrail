package dedup

import (
	"testing"
	"time"

	"github.com/ebrgr/TuneTrail/internal/ingest"
	"github.com/stretchr/testify/require"
)

// TestShadowUnicodeAndMappedRecording exercises strong evidence inside the candidate window.
func TestShadowUnicodeAndMappedRecording(t *testing.T) {
	now := time.Unix(1700000000, 0)
	candidate := Receipt{Time: now, Snapshot: ingest.Snapshot{Artist: "Björk", Title: "Jóga Live", Recording: "recording-id"}}
	incoming := Receipt{Time: now.Add(2 * time.Second), Snapshot: ingest.Snapshot{Artist: "  BJO\u0308RK ", Title: " JO\u0301GA\t LIVE   "}}
	result := Compare(Shadow, incoming, &candidate, false)
	require.Equal(t, "review", result.Value)
	require.Equal(t, "shadow-v2", result.RuleVersion)
	incoming.Snapshot = ingest.Snapshot{Artist: "Translated", Title: "Another title", MappedRecording: "RECORDING-ID"}
	result = Compare(Shadow, incoming, &candidate, false)
	require.Equal(t, "review", result.Value)
	require.True(t, result.Evidence.RecordingMatch)
	incoming.Time = now.Add(3 * time.Second)
	require.Equal(t, "different", Compare(Shadow, incoming, &candidate, false).Value)
}
