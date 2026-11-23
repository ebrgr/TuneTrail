package sqlite

import (
	"testing"

	"github.com/ebrgr/TuneTrail/internal/ingest"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// TestOccurrenceArtistMBIDFallback170 keeps declared artist IDs first and uses
// mbid_mapping only when no declared ID is valid.
func TestOccurrenceArtistMBIDFallback170(t *testing.T) {
	declared := uuid.NewString()
	mapped := uuid.NewString()
	for _, tc := range []struct {
		name     string
		snapshot ingest.Snapshot
		want     []string
	}{
		{"mapped only", ingest.Snapshot{MappedArtistMBIDs: []string{mapped}}, []string{mapped}},
		{"declared wins", ingest.Snapshot{ArtistMBIDs: []string{declared}, MappedArtistMBIDs: []string{mapped}}, []string{declared}},
		{"invalid declared falls back", ingest.Snapshot{ArtistMBIDs: []string{"not-a-uuid"}, MappedArtistMBIDs: []string{mapped}}, []string{mapped}},
		{"invalid mapped ignored", ingest.Snapshot{MappedArtistMBIDs: []string{"not-a-uuid"}}, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var got []string
			for _, id := range occurrenceArtistDeclarations(tc.snapshot) {
				got = append(got, id.String())
			}
			require.Equal(t, tc.want, got)
		})
	}
}
