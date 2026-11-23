package sqlite

import (
	"encoding/json"
	"path/filepath"
	"testing"
	"time"

	migrations "github.com/ebrgr/TuneTrail/db/migrations_sqlite"
	"github.com/ebrgr/TuneTrail/internal/db"
	"github.com/ebrgr/TuneTrail/internal/ingest"
	"github.com/ebrgr/TuneTrail/internal/listenbrainz"
	"github.com/ebrgr/TuneTrail/internal/summary"
	"github.com/ebrgr/TuneTrail/internal/testfixtures"
	"github.com/pressly/goose/v3"
	"github.com/stretchr/testify/require"
)

func TestDurationPlayedMigration(t *testing.T) {
	s, _, source := migrationDB(t, 22)
	testfixtures.Profile(t, s, 1, "one", "One", 0)
	seedMusic(t, s)
	execSQL(t, s, `INSERT INTO listens(profile_id,track_id,listened_at) VALUES(1,1,42)`)
	dir := t.TempDir()
	target := filepath.Join(dir, "tunetrail.db")
	require.NoError(t, copyDatabase(source, target))
	_, err := bootstrapAtVersion(dir, false, 23)
	require.NoError(t, err)

	backups, err := filepath.Glob(target + ".pre-ingest-*.bak")
	require.NoError(t, err)
	require.Len(t, backups, 1)
	backup, err := openExisting(backups[0])
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, backup.Close()) })
	version, err := schemaVersion(backup)
	require.NoError(t, err)
	require.Equal(t, 22, version)
	require.NoError(t, checkDatabase(backup))
	_, err = backup.Exec(`UPDATE listens SET duration_played=0`)
	require.ErrorContains(t, err, "no such column")

	conn, err := openExisting(target)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, conn.Close()) })
	version, err = schemaVersion(conn)
	require.NoError(t, err)
	require.Equal(t, 23, version)
	store := &Sqlite{db: conn}
	provider, err := goose.NewProvider(goose.DialectSQLite3, conn, migrations.Files)
	require.NoError(t, err)
	_, err = provider.Down(t.Context())
	require.ErrorContains(t, err, "restore_verified_backup")
	require.Equal(t, scalar(t, s, `SELECT COUNT(*) FROM listens`),
		scalar(t, store, `SELECT COUNT(*) FROM listens WHERE duration_played IS NULL`))
	_, err = conn.Exec(`UPDATE listens SET duration_played=-1`)
	require.ErrorContains(t, err, "CHECK constraint failed")
	_, err = conn.Exec(`UPDATE listens SET duration_played=1.5`)
	require.ErrorContains(t, err, "CHECK constraint failed")
	_, err = conn.Exec(`UPDATE listens SET duration_played=0`)
	require.NoError(t, err)

	restored := filepath.Join(t.TempDir(), "tunetrail.db")
	require.NoError(t, copyDatabase(backups[0], restored))
	_, err = bootstrapAtVersion(filepath.Dir(restored), false, 23)
	require.NoError(t, err)
	roundTrip, err := openExisting(restored)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, roundTrip.Close()) })
	require.Equal(t, scalar(t, s, `SELECT COUNT(*) FROM listens`),
		scalar(t, &Sqlite{db: roundTrip}, `SELECT COUNT(*) FROM listens WHERE duration_played IS NULL`))
}

func TestDurationPlayedIngestionAndTotals(t *testing.T) {
	s := convertedOccurrenceFixture(t)
	execSQL(t, s, `DELETE FROM listens`)
	ctx := db.WithProfile(t.Context(), 1)
	for i, info := range []string{
		`{"duration":200,"duration_played":120,"duration_played_ms":50000}`,
		`{"duration_played_ms":90500}`, `{}`, `{"duration_played":0}`, `{"duration_played":999}`,
	} {
		var meta listenbrainz.LbzTrackMeta
		require.NoError(t, json.Unmarshal([]byte(`{"artist_name":"Some Artist","track_name":"Played","release_name":"Fixture Album","additional_info":`+info+`}`), &meta))
		id := admitOccurrenceFixture(t, s, ingest.ListenBrainzSnapshot(meta), 2000+int64(i)*100, "")
		resolveOccurrenceFixture(t, s, nil)
		requireReceiptState(t, s, id, "accepted")
	}
	// Absent and zero are both unknown: stored as NULL, so the whole track is counted (zero is not a measurement).
	require.Equal(t, 2, scalar(t, s, `SELECT COUNT(*) FROM listens WHERE listened_at IN (2200,2300) AND duration_played IS NULL`))
	for at, expected := range map[int]int{2000: 120, 2100: 90, 2400: 200} {
		var seconds *int
		require.NoError(t, s.db.QueryRow(`SELECT duration_played FROM listens WHERE listened_at=?`, at).Scan(&seconds))
		require.NotNil(t, seconds)
		require.Equal(t, expected, *seconds)
	}
	var track, album, artist int32
	require.NoError(t, s.db.QueryRow(`SELECT l.track_id,t.release_id,a.artist_id FROM listens l
		JOIN tracks t ON t.id=l.track_id JOIN artist_tracks a ON a.track_id=t.id WHERE l.listened_at=2000`).Scan(&track, &album, &artist))
	window := db.Timeframe{From: time.Unix(2000, 0), To: time.Unix(2200, 0)}
	total, err := s.CountTimeListened(ctx, window)
	require.NoError(t, err)
	require.EqualValues(t, 410, total)
	for _, opts := range []db.TimeListenedOpts{
		{TrackID: track, Timeframe: window}, {AlbumID: album, Timeframe: window}, {ArtistID: artist, Timeframe: window},
	} {
		total, err := s.CountTimeListenedToItem(ctx, opts)
		require.NoError(t, err)
		require.EqualValues(t, 410, total)
	}
	rewind, err := summary.GenerateSummary(ctx, s, 1, window, "Rewind")
	require.NoError(t, err)
	require.Equal(t, 6, rewind.MinutesListened)
	require.EqualValues(t, 410, rewind.TopTracks[0].Item.TimeListened)
	require.EqualValues(t, 410, rewind.TopAlbums[0].Item.TimeListened)
	require.EqualValues(t, 410, rewind.TopArtists[0].Item.TimeListened)
	// The SQL write rule alone (used by archive restores): zero is unknown, others are capped at the track length of 200.
	for arg, want := range map[int64]*int64{0: nil, 50: ingest.PlayedSeconds(50), 500: ingest.PlayedSeconds(200)} {
		var stored *int64
		require.NoError(t, s.db.QueryRow(`SELECT `+boundedPlayedSeconds, arg, track).Scan(&stored))
		require.Equal(t, want, stored, "argument %d", arg)
	}
	gotTrack, err := s.GetTrack(ctx, db.GetTrackOpts{ID: track})
	require.NoError(t, err)
	require.EqualValues(t, 810, gotTrack.TimeListened) // 120+90+200 (absent)+200 (zero)+200 (999 capped)
	gotAlbum, err := s.GetAlbum(ctx, db.GetAlbumOpts{ID: album})
	require.NoError(t, err)
	require.EqualValues(t, 810, gotAlbum.TimeListened)
	gotArtist, err := s.GetArtist(ctx, db.GetArtistOpts{ID: artist})
	require.NoError(t, err)
	require.EqualValues(t, 810, gotArtist.TimeListened)
}

func TestDurationPlayedRelay(t *testing.T) {
	for _, reported := range []*int64{nil, ingest.PlayedSeconds(120), ingest.PlayedSeconds(90.5)} {
		for _, version := range []int{1, 2} {
			body, err := encodeRelayItem(version, ingest.Snapshot{Title: "Song", Artist: "Artist", DurationPlayed: reported}, 123)
			require.NoError(t, err)
			var payload listenbrainz.LbzSubmitListenPayload
			require.NoError(t, json.Unmarshal(body, &payload))
			require.Equal(t, reported, ingest.ListenBrainzSnapshot(payload.TrackMeta).DurationPlayed)
		}
	}
}
