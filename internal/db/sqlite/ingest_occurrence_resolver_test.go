package sqlite

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/ebrgr/TuneTrail/internal/db"
	"github.com/ebrgr/TuneTrail/internal/ingest"
	"github.com/ebrgr/TuneTrail/internal/mbz"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

const fixtureRecording = "fa000000-0000-4000-8000-000000000001"

type occurrenceReleaseFixture struct {
	mbz.MbzErrorCaller
	releases map[uuid.UUID]*mbz.MusicBrainzRelease
}

func (m occurrenceReleaseFixture) GetRelease(_ context.Context, id uuid.UUID) (*mbz.MusicBrainzRelease, error) {
	return m.releases[id], nil
}

// convertedOccurrenceFixture installs an opt-in after legacy fixture setup.
func convertedOccurrenceFixture(t *testing.T) *Sqlite {
	t.Helper()
	s, _, _ := migrationDB(t, currentSchemaVersion)
	seedIdentityHistory(t, s)
	execSQL(t, s, `INSERT INTO profile_ingest_flags(profile_id,active,occurrence_write) VALUES(1,1,1)`)
	return s
}

func admitOccurrenceFixture(t *testing.T, s *Sqlite, snapshot ingest.Snapshot, stamp int64, key string) string {
	t.Helper()
	active, receipts, err := s.AdmitIngest(context.Background(), ingest.Batch{
		ProfileID: 1, Source: ingest.ShadowSourceListenBrainz, Operation: "single", ClientKey: key, ReceivedAt: time.Now(),
		Items: []ingest.Item{{ListenedAt: time.Unix(stamp, 0), Snapshot: snapshot}},
	}, true, nil, nil)
	require.NoError(t, err)
	require.True(t, active)
	require.Len(t, receipts, 1)
	return receipts[0].ReceiptID
}

func resolveOccurrenceFixture(t *testing.T, s *Sqlite, client mbz.MusicBrainzCaller) {
	t.Helper()
	found, err := s.ResolveNextIngest(context.Background(), client)
	require.NoError(t, err)
	require.True(t, found)
}

func requireReceiptState(t *testing.T, s *Sqlite, id, state string) {
	t.Helper()
	require.Equal(t, 1, scalar(t, s, `SELECT count(*) FROM ingest_events e JOIN ingest_revisions r USING(profile_id,receipt_id,resolution_version)
		WHERE e.receipt_id=? AND e.state=? AND e.rule_version='occurrence-v1' AND r.rule_version=e.rule_version`, id, state))
}

// TestOccurrenceResolverAB verifies shared recording identity and distinct visible A/B bridges.
func TestOccurrenceResolverAB(t *testing.T) {
	for _, reverse := range []bool{false, true} {
		t.Run(map[bool]string{false: "A-first", true: "B-first"}[reverse], func(t *testing.T) {
			ctx := context.Background()
			s := convertedOccurrenceFixture(t)
			s.db.SetMaxOpenConns(2)
			client := &occurrenceReleaseFixture{releases: map[uuid.UUID]*mbz.MusicBrainzRelease{}}
			var snapshots []ingest.Snapshot
			for i, title := range []string{"Album A", "Album B"} {
				id := uuid.New()
				track := mbz.MusicBrainzReleaseTrack{Title: "Shared", Position: 1, ID: uuid.NewString()}
				track.Recording.ID = fixtureRecording
				client.releases[id] = &mbz.MusicBrainzRelease{ID: id.String(), Title: title, Media: []mbz.MusicBrainzMedium{{Position: 1, Tracks: []mbz.MusicBrainzReleaseTrack{track}}}}
				snapshots = append(snapshots, ingest.Snapshot{Title: "Shared", Artist: "Artist", Release: title, ReleaseMBID: id.String(),
					Recording: fixtureRecording, TrackMBID: track.ID, DiscNumber: 1, TrackNumber: 1, DurationMS: int64(i + 1)})
			}
			if reverse {
				snapshots[0], snapshots[1] = snapshots[1], snapshots[0]
			}
			first := admitOccurrenceFixture(t, s, snapshots[0], 100, "first")
			resolveOccurrenceFixture(t, s, client)
			second := admitOccurrenceFixture(t, s, snapshots[1], 100, "second")
			// Two independent workers contend for the next claim.
			var group sync.WaitGroup
			for range 2 {
				group.Go(func() {
					_, err := s.ResolveNextIngest(ctx, client)
					if err != nil {
						t.Error(err)
					}
				})
			}
			group.Wait()
			requireReceiptState(t, s, first, "accepted")
			requireReceiptState(t, s, second, "accepted")
			require.Equal(t, 1, scalar(t, s, `SELECT count(*) FROM recordings WHERE musicbrainz_id=? AND mbid_source='validated'`, fixtureRecording))
			require.Equal(t, 2, scalar(t, s, `SELECT count(DISTINCT o.release_id) FROM occurrences o JOIN recordings r ON r.id=o.recording_id
				JOIN tracks t ON t.id=o.legacy_track_id WHERE r.musicbrainz_id=?`, fixtureRecording))
			require.Equal(t, 2, scalar(t, s, `SELECT count(*) FROM listens WHERE origin='listenbrainz_api' AND listened_at=100`))
			require.Zero(t, scalar(t, s, `SELECT count(*) FROM recordings r WHERE NOT EXISTS (SELECT 1 FROM occurrences o WHERE o.recording_id=r.id)`))

			execSQL(t, s, `DELETE FROM tracks WHERE musicbrainz_id=?`, fixtureRecording)
			legacy, err := s.SaveTrack(ctx, db.SaveTrackOpts{RecordingMbzID: uuid.MustParse(fixtureRecording), AlbumID: 1, ArtistIDs: []int32{1}, Title: "Shared"})
			require.NoError(t, err)
			require.NoError(t, s.SaveListen(ctx, db.SaveListenOpts{ProfileID: 2, TrackID: legacy.ID, Time: time.Unix(160, 0)}))
			require.NoError(t, s.UpdateTrack(ctx, db.UpdateTrackOpts{ID: legacy.ID, MusicBrainzID: uuid.MustParse(fixtureRecording), OnlyFillMusicBrainzID: true}))
			require.ErrorIs(t, s.UpdateTrack(ctx, db.UpdateTrackOpts{ID: legacy.ID, MusicBrainzID: uuid.New()}), db.ErrConflict)
			again := admitOccurrenceFixture(t, s, snapshots[1], 200, "after-delete")
			resolveOccurrenceFixture(t, s, client)
			requireReceiptState(t, s, again, "accepted")
			require.Equal(t, 1, scalar(t, s, `SELECT count(*) FROM recordings WHERE musicbrainz_id=?`, fixtureRecording))
		})
	}
}

// TestOccurrenceResolverIntro exercises positions, proof, ambiguity and source claims.
func TestOccurrenceResolverIntro(t *testing.T) {
	s := convertedOccurrenceFixture(t)
	releaseID := uuid.New()
	tracks := []mbz.MusicBrainzReleaseTrack{{Title: "Intro", Position: 1, ID: uuid.NewString()}, {Title: "Intro", Position: 1, ID: uuid.NewString()}}
	for i := range tracks {
		tracks[i].Recording.ID = fixtureRecording
	}
	release := &mbz.MusicBrainzRelease{ID: releaseID.String(), Title: "Two discs",
		Media: []mbz.MusicBrainzMedium{{Position: 1, Tracks: tracks[:1]}, {Position: 2, Tracks: tracks[1:]}}}
	client := &occurrenceReleaseFixture{releases: map[uuid.UUID]*mbz.MusicBrainzRelease{releaseID: release}}
	snapshot := ingest.Snapshot{Title: "Intro", Artist: "Artist", Release: release.Title, ReleaseMBID: release.ID, Recording: fixtureRecording}
	for i, track := range tracks {
		item := snapshot
		item.DiscNumber, item.TrackNumber, item.TrackMBID = i+1, 1, track.ID
		id := admitOccurrenceFixture(t, s, item, 300, track.ID)
		resolveOccurrenceFixture(t, s, client)
		requireReceiptState(t, s, id, "accepted")
	}
	require.Equal(t, 2, scalar(t, s, `SELECT count(*) FROM occurrences WHERE track_mbid IS NOT NULL AND track_number=1`))
	for _, item := range []ingest.Snapshot{
		snapshot,
		{Title: "Intro", Artist: "Artist", Release: release.Title, DiscNumber: 2, TrackNumber: 1, TrackMBID: tracks[0].ID},
		{Title: "Intro", Artist: "Artist", Release: "Wrong album", TrackMBID: tracks[0].ID},
		{Title: "No album", Artist: "Artist"},
	} {
		id := admitOccurrenceFixture(t, s, item, 301, uuid.NewString())
		resolveOccurrenceFixture(t, s, nil)
		requireReceiptState(t, s, id, "pending_identity")
	}
	for _, position := range []int{1, 2} {
		item := ingest.Snapshot{Title: "Intro", Artist: "Artist", Release: "Provisional album", DiscNumber: 1, TrackNumber: position,
			Recording: uuid.NewString(), MBIDSource: "external-service", MBIDDeclaredAt: "2026-11-20T00:00:00Z"}
		id := admitOccurrenceFixture(t, s, item, 400, uuid.NewString())
		resolveOccurrenceFixture(t, s, &mbz.MbzErrorCaller{})
		requireReceiptState(t, s, id, "accepted")
	}
	require.Equal(t, 2, scalar(t, s, `SELECT count(*) FROM occurrences o JOIN recordings r ON r.id=o.recording_id
		WHERE o.track_mbid IS NULL AND o.track_number IN (1,2) AND r.musicbrainz_id IS NULL`))
	require.Zero(t, scalar(t, s, `SELECT count(*) FROM outbox`))
}

// TestOccurrenceResolverOverlap times first visibility and preserves ambiguous imports.
func TestOccurrenceResolverOverlap(t *testing.T) {
	s := convertedOccurrenceFixture(t)
	snapshot := ingest.Snapshot{Title: "Track", Artist: "Artist", Release: "Album"}
	var elapsed time.Duration
	for i, stamp := range []int64{500, 501, 500} {
		batch := ingest.Batch{ProfileID: 1, Source: ingest.ShadowSourceFile, FileFormat: "lastfm", Operation: "import",
			FileDigest: ingest.Digest([]byte(uuid.NewString())),
			ReceivedAt: time.Now(), Items: []ingest.Item{{ListenedAt: time.Unix(stamp, 0), Snapshot: snapshot}}}
		active, items, err := s.AdmitIngest(context.Background(), batch, true, nil, nil)
		require.NoError(t, err)
		require.True(t, active)
		began := time.Now()
		resolveOccurrenceFixture(t, s, nil)
		elapsed += time.Since(began)
		state := "pending_duplicate"
		if i == 0 {
			state = "accepted"
			t.Logf("first admitted listen visible after %s", time.Since(began))
		}
		requireReceiptState(t, s, items[0].ReceiptID, state)
	}
	t.Logf("three resolver calls: %s; two pending out of three", elapsed)
	require.Equal(t, 1, scalar(t, s, `SELECT count(*) FROM listens WHERE origin='lastfm_import'`))
	require.Equal(t, 1, scalar(t, s, `SELECT count(*) FROM ingest_events WHERE state='pending_duplicate' AND reason_codes='["legacy_identity_collision"]'`))
	require.Zero(t, scalar(t, s, `SELECT count(*) FROM outbox`))
}

// TestOccurrenceResolverRecovery proves stale claims and failed commits leave no catalog data.
func TestOccurrenceResolverRecovery(t *testing.T) {
	ctx := context.Background()
	s := convertedOccurrenceFixture(t)
	snapshot := ingest.Snapshot{Title: "Never residual", Artist: "New artist", Release: "New release"}
	id := admitOccurrenceFixture(t, s, snapshot, 800, "durable")
	receipt, _, _, err := s.claimNextIngest(ctx)
	require.NoError(t, err)
	receipt.snapshot, receipt.unix = snapshot, 800
	execSQL(t, s, `UPDATE ingest_events SET work_version=work_version+1,lease_owner=NULL,lease_until=NULL WHERE receipt_id=?`, id)
	require.NoError(t, s.finishOccurrence(ctx, receipt, occurrenceProof{}))
	require.Zero(t, scalar(t, s, `SELECT count(*) FROM release_aliases WHERE alias='New release'`))
	execSQL(t, s, `CREATE TABLE occurrence_commit_failure (profile INTEGER REFERENCES listening_profiles(id) DEFERRABLE INITIALLY DEFERRED);
		CREATE TRIGGER occurrence_commit_failure_inject AFTER INSERT ON ingest_revisions WHEN NEW.rule_version='occurrence-v1'
		BEGIN INSERT INTO occurrence_commit_failure VALUES(-1); END`)
	found, err := s.ResolveNextIngest(ctx, nil)
	require.True(t, found)
	var unavailable *ingest.ErrUnavailable
	require.ErrorAs(t, err, &unavailable)
	require.Zero(t, scalar(t, s, `SELECT count(*) FROM release_aliases WHERE alias='New release'`))
	require.Zero(t, scalar(t, s, `SELECT count(*) FROM listens WHERE listened_at=800`))
	require.Equal(t, 1, scalar(t, s, `SELECT count(*) FROM ingest_events WHERE receipt_id=? AND state='retryable'`, id))
	execSQL(t, s, `DROP TRIGGER occurrence_commit_failure_inject;
		UPDATE ingest_events SET next_attempt_at=0 WHERE receipt_id=?`, id)
	resolveOccurrenceFixture(t, s, nil)
	requireReceiptState(t, s, id, "accepted")
}

// TestOccurrenceResolverCut prevents new writes by old admission paths.
func TestOccurrenceResolverCut(t *testing.T) {
	ctx := context.Background()
	s := convertedOccurrenceFixture(t)
	snapshot := ingest.Snapshot{Title: "Track", Artist: "Artist", Release: "B"}
	id := admitOccurrenceFixture(t, s, snapshot, 900, "original")
	resolveOccurrenceFixture(t, s, nil)
	requireReceiptState(t, s, id, "accepted")
	batch := ingest.Batch{ProfileID: 1, Source: ingest.ShadowSourceListenBrainz, Operation: "single", ClientKey: "original", ReceivedAt: time.Now(),
		Items: []ingest.Item{{Snapshot: snapshot, ListenedAt: time.Unix(900, 0)}}}
	active, results, err := s.AdmitIngest(ctx, batch, false, nil, nil)
	var unavailable *ingest.ErrUnavailable
	require.ErrorAs(t, err, &unavailable)
	require.True(t, active)
	require.Empty(t, results)
	_, err = s.BeginImportIngest(ctx, 1, false)
	require.ErrorAs(t, err, &unavailable)
	err = s.SaveListen(ctx, db.SaveListenOpts{ProfileID: 1, TrackID: 1, Time: time.Unix(900, 0)})
	require.ErrorContains(t, err, "Legacy listen conflicts")
	_, err = s.db.Exec(`INSERT INTO profile_relays VALUES(1,'https://fixture.invalid','fixture')`)
	require.ErrorContains(t, err, "Relay conflicts")
	require.Equal(t, 1, scalar(t, s, `SELECT count(*) FROM listens WHERE profile_id=1 AND listened_at=900`))
}

func TestOccurrenceResolverStandalone(t *testing.T) {
	s := convertedOccurrenceFixture(t)
	s.allowStandaloneOccurrences = true
	item := ingest.Snapshot{Title: "Solo", ArtistNames: []string{"A", "B"}}
	id := admitOccurrenceFixture(t, s, item, 901, "standalone")
	resolveOccurrenceFixture(t, s, nil)
	requireReceiptState(t, s, id, "accepted")
	require.Equal(t, 1, scalar(t, s, `SELECT count(*) FROM occurrences WHERE release_id IS NULL AND legacy_track_id IS NULL AND presented_title='Solo'`))
	require.Equal(t, 2, scalar(t, s, `SELECT count(*) FROM occurrence_credits`))
}

// TestOccurrenceProvisionalPromotion verifies that later proof keeps the original bridge and recording.
func TestOccurrenceProvisionalPromotion(t *testing.T) {
	s := convertedOccurrenceFixture(t)
	item := ingest.Snapshot{Title: "Shared", Artist: "Artist", Release: "Provisional", DiscNumber: 1, TrackNumber: 1}
	original := admitOccurrenceFixture(t, s, item, 1000, "provisional")
	resolveOccurrenceFixture(t, s, nil)
	requireReceiptState(t, s, original, "accepted")

	releaseID := uuid.New()
	track := mbz.MusicBrainzReleaseTrack{Title: item.Title, Position: 1, ID: uuid.NewString()}
	track.Recording.ID = fixtureRecording
	client := &occurrenceReleaseFixture{releases: map[uuid.UUID]*mbz.MusicBrainzRelease{
		releaseID: {ID: releaseID.String(), Title: item.Release, Media: []mbz.MusicBrainzMedium{{Position: 1, Tracks: []mbz.MusicBrainzReleaseTrack{track}}}},
	}}
	item.Recording, item.ReleaseMBID, item.TrackMBID = fixtureRecording, releaseID.String(), track.ID
	proven := admitOccurrenceFixture(t, s, item, 1010, "proven")
	resolveOccurrenceFixture(t, s, client)
	requireReceiptState(t, s, proven, "accepted")
	require.Equal(t, 1, scalar(t, s, `SELECT count(DISTINCT occurrence_id) FROM ingest_events WHERE receipt_id IN (?,?)`, original, proven))
	require.Equal(t, 1, scalar(t, s, `SELECT count(*) FROM occurrences o JOIN recordings r ON r.id=o.recording_id
		WHERE o.track_mbid=? AND r.musicbrainz_id=? AND r.mbid_source='validated'`, track.ID, fixtureRecording))
}
