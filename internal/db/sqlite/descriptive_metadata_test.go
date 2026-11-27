package sqlite

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestDescriptiveScopeAndLifecycle covers support, deduplication and stable occurrence bindings.
func TestDescriptiveScopeAndLifecycle(t *testing.T) {
	s := mergeFixture(t)
	execSQL(t, s, `INSERT INTO metadata_sources VALUES(1,'test','fingerprint')`)
	insert := `INSERT INTO descriptive_metadata(profile_id,occurrence_id,source_id,kind,value,observed_at) VALUES(?,?,1,'genre','Rock','2026-01-01')`
	require.ErrorContains(t, s.Exec(insert, 3, 1), "no supporting listen")
	execSQL(t, s, insert, 1, 1)
	require.Error(t, s.Exec(insert, 1, 1))
	execSQL(t, s, insert, 1, 2)
	require.NoError(t, s.MergeTracks(t.Context(), 1, 2))
	require.Equal(t, 2, scalar(t, s, `SELECT count(*) FROM descriptive_metadata`))
	require.NoError(t, s.UndoTrackMerge(t.Context(), 1, 2))
	require.Equal(t, 2, scalar(t, s, `SELECT count(*) FROM descriptive_metadata`))
	execSQL(t, s, `DELETE FROM listens WHERE profile_id=1 AND track_id=1`)
	require.Equal(t, 1, scalar(t, s, `SELECT count(*) FROM descriptive_metadata`))
	assertForeignKeys(t, s)
}

// TestDescriptiveReleaseMerge preserves source values supported by moved listens.
func TestDescriptiveReleaseMerge(t *testing.T) {
	s := mergeFixture(t)
	execSQL(t, s, `INSERT INTO metadata_sources VALUES(1,'test','fingerprint');
 INSERT INTO descriptive_metadata(profile_id,release_id,source_id,kind,value,observed_at) VALUES(1,1,1,'release_date','2005','2026-01-01')`)
	require.NoError(t, s.MergeAlbums(t.Context(), 1, 2, false))
	require.Equal(t, 1, scalar(t, s, `SELECT count(*) FROM descriptive_metadata WHERE release_id=2 AND value='2005'`))
	assertForeignKeys(t, s)
}

// TestDescriptiveDeleteLifecycle keeps values only while their profile has supporting listens.
func TestDescriptiveDeleteLifecycle(t *testing.T) {
	for _, action := range []string{"listen", "track", "album", "artist", "profile"} {
		t.Run(action, func(t *testing.T) {
			s := mergeFixture(t)
			execSQL(t, s, `INSERT INTO metadata_sources VALUES(1,'test','fingerprint');
 INSERT INTO descriptive_metadata(profile_id,occurrence_id,source_id,kind,value,observed_at) VALUES(1,1,1,'genre','Track','2026-01-01');
 INSERT INTO descriptive_metadata(profile_id,release_id,source_id,kind,value,observed_at) VALUES(1,1,1,'genre','Release','2026-01-01');
 INSERT INTO descriptive_metadata(profile_id,artist_id,source_id,kind,value,observed_at) VALUES(1,1,1,'genre','Artist','2026-01-01')`)
			switch action {
			case "listen":
				execSQL(t, s, `DELETE FROM listens WHERE profile_id=1 AND track_id=1`)
			case "track":
				require.NoError(t, s.DeleteTrack(t.Context(), 1))
			case "album":
				require.NoError(t, s.DeleteAlbum(t.Context(), 1))
			case "artist":
				require.NoError(t, s.DeleteArtist(t.Context(), 1))
			case "profile":
				execSQL(t, s, `DELETE FROM listening_profiles WHERE id=1`)
			}
			require.Zero(t, scalar(t, s, `SELECT count(*) FROM descriptive_metadata m WHERE NOT EXISTS (
 SELECT 1 FROM listens l JOIN occurrences o ON o.id=l.track_id WHERE l.profile_id=m.profile_id AND (m.occurrence_id=o.id OR m.release_id=o.release_id OR EXISTS (SELECT 1 FROM artist_tracks a WHERE a.track_id=o.id AND a.artist_id=m.artist_id) OR EXISTS (SELECT 1 FROM occurrence_credits c WHERE c.occurrence_id=o.id AND c.artist_id=m.artist_id)))`))
			assertForeignKeys(t, s)
		})
	}
}

// TestDescriptiveArtistMergePreservesDistinctSources and the supporting profile.
func TestDescriptiveArtistMergePreservesDistinctSources(t *testing.T) {
	s := mergeFixture(t)
	execSQL(t, s, `INSERT INTO metadata_sources VALUES(1,'test','fingerprint');
 INSERT INTO descriptive_metadata(profile_id,artist_id,source_id,kind,value,observed_at) VALUES(1,1,1,'genre','Rock','2026-01-01')`)
	require.NoError(t, s.MergeArtists(t.Context(), 1, 2, false))
	require.Equal(t, 1, scalar(t, s, `SELECT count(*) FROM descriptive_metadata WHERE artist_id=2 AND value='Rock'`))
	assertForeignKeys(t, s)
}

// TestDescriptivePurge removes values when all supporting history is purged.
func TestDescriptivePurge(t *testing.T) {
	s := mergeFixture(t)
	execSQL(t, s, `INSERT INTO metadata_sources VALUES(1,'test','fingerprint');
 INSERT INTO descriptive_metadata(profile_id,occurrence_id,source_id,kind,value,observed_at) VALUES(1,1,1,'genre','Rock','2026-01-01')`)
	require.NoError(t, s.PurgeAllData(t.Context()))
	require.Zero(t, scalar(t, s, `SELECT count(*) FROM descriptive_metadata`))
	assertForeignKeys(t, s)
}
