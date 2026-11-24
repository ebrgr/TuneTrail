package sqlite

import (
	"database/sql/driver"
	"encoding/json"
	"slices"

	"github.com/ebrgr/TuneTrail/internal/ingest"
	sqlite3 "modernc.org/sqlite"
)

func init() {
	// Reuse the ingest credit rules during migration and receipt writes.
	sqlite3.MustRegisterDeterministicScalarFunction("tt_received_artist_names", 1,
		func(_ *sqlite3.FunctionContext, args []driver.Value) (driver.Value, error) {
			raw, ok := args[0].(string)
			var snapshot ingest.Snapshot
			if !ok || json.Unmarshal([]byte(raw), &snapshot) != nil {
				return "[]", nil
			}
			names := make([]string, 0, len(snapshot.ArtistNames))
			for name := range fileArtistDeclarations(snapshot) {
				names = append(names, name)
			}
			slices.Sort(names)
			data, err := json.Marshal(names)
			return string(data), err
		})
}

// scopedArtistClaim checks received credit names via the artist's current
// relationships. The lookup visits catalog occurrences and indexed observations.
const scopedArtistClaim = `
	WITH scope(profile_id,artist_id) AS (VALUES (?,?))
	SELECT min(c.value)
	FROM scope s CROSS JOIN metadata_values c
	WHERE c.profile_id=s.profile_id AND c.artist_id=s.artist_id AND c.kind='mbid'
	HAVING count(DISTINCT c.value)=1 AND NOT EXISTS (
		SELECT 1 FROM artist_tracks a
		CROSS JOIN artist_aliases n
		CROSS JOIN enrichment_received_artist_credits r
		WHERE a.artist_id=s.artist_id AND n.artist_id=a.artist_id
		  AND r.profile_id=s.profile_id AND r.occurrence_id=a.track_id AND r.name=n.alias
		UNION ALL
		SELECT 1 FROM occurrence_credits a
		CROSS JOIN enrichment_received_artist_credits r
		WHERE a.artist_id=s.artist_id
		  AND r.profile_id=s.profile_id AND r.occurrence_id=a.occurrence_id AND r.name=a.name
	)`
