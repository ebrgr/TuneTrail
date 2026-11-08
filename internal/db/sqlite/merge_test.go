package sqlite

import (
	"context"
	"github.com/ebrgr/TuneTrail/internal/db"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

// mergeFixture gives both people the same two tracks at an identical timestamp.
func mergeFixture(t *testing.T) *Sqlite {
	t.Helper()
	s, err := NewInMemory()
	require.NoError(t, err)
	t.Cleanup(func() { s.Close(context.Background()) })
	for _, q := range []string{
		`INSERT INTO listening_profiles(id,slug,display_name,created_at) VALUES(1,'alpha','Alpha',1),(2,'beta','Beta',1)`,
		`INSERT INTO artists(id) VALUES(1),(2)`,
		`INSERT INTO artist_aliases VALUES(1,'One','User',1),(2,'Two','User',1)`,
		`INSERT INTO releases(id,various_artists) VALUES(1,0),(2,0)`,
		`INSERT INTO release_aliases VALUES(1,'One','User',1),(2,'Two','User',1)`,
		`INSERT INTO artist_releases VALUES(1,1,1),(2,2,1)`,
		`INSERT INTO tracks(id,release_id,duration) VALUES(1,1,180),(2,2,120)`,
		`INSERT INTO track_aliases VALUES(1,'One','User',1),(2,'Two','User',1)`,
		`INSERT INTO artist_tracks VALUES(1,1,1),(2,2,1)`,
		`INSERT INTO listens(profile_id,track_id,listened_at) VALUES(1,1,100),(2,1,100),(1,2,100),(2,2,100)`,
	} {
		require.NoError(t, s.Exec(q))
	}
	return s
}

// mergeSnapshot checks catalog tables as well as personal histories after rejected writes.
func mergeSnapshot(t *testing.T, s *Sqlite) map[string][32]byte {
	t.Helper()
	ret := map[string][32]byte{}
	for _, table := range []string{"artists", "artist_aliases", "releases", "release_aliases", "tracks", "track_aliases", "artist_tracks", "artist_releases", "listens", "listening_profiles"} {
		ret[table] = queryDigest(t, s.db, "SELECT * FROM "+table+" ORDER BY 1,2")
	}
	return ret
}

// TestMergeIdentitiesAndRollback guards self merge, absence, and failures after partial writes.
func TestMergeIdentitiesAndRollback(t *testing.T) {
	for _, kind := range []string{"artist", "album", "track"} {
		t.Run(kind, func(t *testing.T) {
			s := mergeFixture(t)
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			merge := func(from, to int32) error {
				switch kind {
				case "artist":
					return s.MergeArtists(ctx, from, to, true)
				case "album":
					return s.MergeAlbums(ctx, from, to, true)
				default:
					return s.MergeTracks(ctx, from, to)
				}
			}
			before := mergeSnapshot(t, s)
			for _, ids := range [][2]int32{{1, 1}, {0, 2}, {1, -1}} {
				require.ErrorIs(t, merge(ids[0], ids[1]), db.ErrInvalidInput)
				require.Equal(t, before, mergeSnapshot(t, s))
			}
			for _, ids := range [][2]int32{{999, 2}, {1, 999}} {
				require.ErrorIs(t, merge(ids[0], ids[1]), db.ErrNotFound)
				require.Equal(t, before, mergeSnapshot(t, s))
			}
			// Trigger failures occur after a preceding mutation and must roll everything back.
			trigger := map[string]string{"artist": `CREATE TRIGGER fail_merge BEFORE UPDATE ON artist_releases BEGIN SELECT RAISE(ABORT,'injected'); END`, "album": `CREATE TRIGGER fail_merge BEFORE INSERT ON artist_releases BEGIN SELECT RAISE(ABORT,'injected'); END`, "track": `CREATE TRIGGER fail_merge BEFORE INSERT ON artist_releases BEGIN SELECT RAISE(ABORT,'injected'); END`}[kind]
			require.NoError(t, s.Exec(trigger))
			require.Error(t, merge(1, 2))
			require.Equal(t, before, mergeSnapshot(t, s))
			require.NoError(t, s.Exec(`DROP TRIGGER fail_merge`))
			require.NoError(t, merge(1, 2))
			want := 4
			if kind == "track" {
				want = 2
			}
			require.Equal(t, want, scalar(t, s, `SELECT COUNT(*) FROM listens`))
			require.Equal(t, want/2, scalar(t, s, `SELECT COUNT(*) FROM listens WHERE profile_id=1`))
			require.Equal(t, want/2, scalar(t, s, `SELECT COUNT(*) FROM listens WHERE profile_id=2`))
			assertForeignKeys(t, s)
		})
	}
}

func TestMergesPreserveUnrelatedAndUnplayedSurvivors(t *testing.T) {
	for _, kind := range []string{"artist", "album", "track"} {
		t.Run(kind, func(t *testing.T) {
			s := mergeFixture(t)
			ctx := context.Background()
			require.NoError(t, s.Exec(`INSERT INTO artists(id) VALUES(50)`))
			require.NoError(t, s.Exec(`INSERT INTO artist_aliases VALUES(50,'Unplayed artist','User',1)`))
			require.NoError(t, s.Exec(`INSERT INTO releases(id,various_artists) VALUES(50,0)`))
			require.NoError(t, s.Exec(`INSERT INTO release_aliases VALUES(50,'Unplayed album','User',1)`))
			require.NoError(t, s.Exec(`INSERT INTO artist_releases VALUES(50,50,1)`))
			require.NoError(t, s.Exec(`INSERT INTO tracks(id,release_id,duration) VALUES(50,50,0)`))
			require.NoError(t, s.Exec(`INSERT INTO track_aliases VALUES(50,'Unplayed track','User',1)`))
			require.NoError(t, s.Exec(`INSERT INTO artist_tracks VALUES(50,50,1)`))
			require.NoError(t, s.Exec(`DELETE FROM listens WHERE track_id=2`))
			var err error
			switch kind {
			case "artist":
				err = s.MergeArtists(ctx, 1, 2, false)
			case "album":
				err = s.MergeAlbums(ctx, 1, 2, false)
			case "track":
				err = s.MergeTracks(ctx, 1, 2)
			}
			require.NoError(t, err)
			for _, q := range []string{`SELECT COUNT(*) FROM artists WHERE id=50`, `SELECT COUNT(*) FROM releases WHERE id=50`, `SELECT COUNT(*) FROM tracks WHERE id=50`} {
				require.Equal(t, 1, scalar(t, s, q))
			}
			table := map[string]string{"artist": "artists", "album": "releases", "track": "tracks"}[kind]
			require.Equal(t, 1, scalar(t, s, `SELECT COUNT(*) FROM `+table+` WHERE id=2`))
			require.Equal(t, 0, scalar(t, s, `SELECT COUNT(*) FROM `+table+` WHERE id=1`))
		})
	}
}
func TestDeleteTrackPreservesUnrelatedUnplayedMedia(t *testing.T) {
	s := mergeFixture(t)
	require.NoError(t, s.Exec(`DELETE FROM listens WHERE track_id=2`))
	require.NoError(t, s.DeleteTrack(context.Background(), 1))
	require.Equal(t, 1, scalar(t, s, `SELECT COUNT(*) FROM tracks WHERE id=2`))
}
