package sqlite

import (
	"context"
	"database/sql"
	"errors"
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
	releases   map[uuid.UUID]*mbz.MusicBrainzRelease
	recordings map[uuid.UUID]*mbz.MusicBrainzTrack
}

func (m occurrenceReleaseFixture) GetRelease(_ context.Context, id uuid.UUID) (*mbz.MusicBrainzRelease, error) {
	return m.releases[id], nil
}

func (m occurrenceReleaseFixture) GetTrack(_ context.Context, id uuid.UUID) (*mbz.MusicBrainzTrack, error) {
	return m.recordings[id], nil
}

// convertedOccurrenceFixture installs an opt-in after legacy fixture setup.
func convertedOccurrenceFixture(t *testing.T) *Sqlite {
	t.Helper()
	s, _, _ := migrationDB(t, currentSchemaVersion)
	seedIdentityHistory(t, s)
	execSQL(t, s, `DELETE FROM ingest_events WHERE profile_id=1 AND mode='shadow' AND event_id IS NULL;
		INSERT INTO profile_ingest_flags(profile_id,active,occurrence_write) VALUES(1,1,1)`)
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
	require.Equal(t, 1, scalar(t, s, `SELECT count(*) FROM ingest_events WHERE state='pending_duplicate' AND reason_codes='["legacy_identity_collision"]' AND json_extract(evidence,'$.candidates[0]') LIKE 'event:%'`))
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

// TestOccurrenceAliasCredits keeps the first artist primary when its aliases repeat.
func TestOccurrenceAliasCredits(t *testing.T) {
	s := convertedOccurrenceFixture(t)
	execSQL(t, s, `INSERT INTO artist_aliases VALUES(1,'Artist alias','Fixture',0);
		INSERT INTO artists(id) VALUES(2);
		INSERT INTO artist_aliases VALUES(2,'Guest','Fixture',1)`)
	item := ingest.Snapshot{Title: "One", Release: "Alias album", ArtistNames: []string{"Artist", "Guest", "Artist alias"}}
	for i, title := range []string{"One", "Two"} {
		item.Title = title
		id := admitOccurrenceFixture(t, s, item, int64(1200+i), "aliases-"+title)
		resolveOccurrenceFixture(t, s, nil)
		requireReceiptState(t, s, id, "accepted")
	}
	require.Equal(t, 1, scalar(t, s, `SELECT count(*) FROM release_aliases WHERE alias='Alias album'`))
	require.Equal(t, 2, scalar(t, s, `SELECT count(*) FROM artist_tracks at JOIN track_aliases t ON t.track_id=at.track_id
		WHERE t.alias IN ('One','Two') AND at.artist_id=1 AND at.is_primary=1`))
	require.Equal(t, 2, scalar(t, s, `SELECT count(*) FROM artist_releases a JOIN release_aliases r USING(release_id) WHERE r.alias='Alias album'`))
}

// TestOccurrenceOriginGuard permits duplicates from other writers but rejects false origins on acceptance.
func TestOccurrenceOriginGuard(t *testing.T) {
	s := convertedOccurrenceFixture(t)
	execSQL(t, s, `INSERT INTO listens(profile_id,track_id,listened_at,event_id,origin) VALUES(1,1,1300,'file-event','lastfm_import')`)
	item := ingest.Snapshot{Title: "Intro", Artist: "Artist", Release: "Album"}
	for _, query := range []string{`SELECT event_id FROM listens WHERE rowid=10`, `SELECT event_id FROM listens WHERE event_id='file-event'`} {
		id := admitOccurrenceFixture(t, s, item, 1301, uuid.NewString())
		update := `UPDATE ingest_events SET state=?,identity_status='resolved',resolution_version=1,event_id=(` + query + `),
			occurrence_id=1,recording_id=(SELECT recording_id FROM occurrences WHERE id=1),closed_at=1 WHERE receipt_id=?`
		_, err := s.db.Exec(update, "accepted", id)
		require.ErrorContains(t, err, "link origin conflicts")
		execSQL(t, s, update, "duplicate", id)
	}
	require.Equal(t, 2, scalar(t, s, `SELECT count(*) FROM ingest_events WHERE resolver_variant='occurrence' AND state='duplicate'`))
}

// TestOccurrenceCutRequiresClearedShadow checks both conversion paths against shadow replays.
func TestOccurrenceCutRequiresClearedShadow(t *testing.T) {
	s, _, _ := migrationDB(t, currentSchemaVersion)
	seedIdentityHistory(t, s)
	const enable = `INSERT INTO profile_ingest_flags(profile_id,active,occurrence_write) VALUES(1,1,1)`
	_, err := s.db.Exec(enable)
	require.ErrorContains(t, err, "cleared shadow")
	execSQL(t, s, `INSERT INTO profile_ingest_flags(profile_id,active) VALUES(1,1)`)
	const convert = `UPDATE profile_ingest_flags SET occurrence_write=1 WHERE profile_id=1`
	_, err = s.db.Exec(convert)
	require.ErrorContains(t, err, "cleared shadow")
	execSQL(t, s, `DELETE FROM ingest_events WHERE profile_id=1 AND mode='shadow' AND event_id IS NULL`)
	execSQL(t, s, convert)
	require.Equal(t, 1, scalar(t, s, `SELECT occurrence_write FROM profile_ingest_flags WHERE profile_id=1`))
}

// TestOccurrenceProofRedirects distinguishes canonical IDs, missing proof and conflicting recordings.
func TestOccurrenceProofRedirects(t *testing.T) {
	request, canonical, recording := uuid.New(), uuid.New(), uuid.New()
	track := mbz.MusicBrainzReleaseTrack{Title: "Song", ID: uuid.NewString(), Position: 1}
	track.Recording.ID = fixtureRecording
	release := &mbz.MusicBrainzRelease{ID: canonical.String(), Title: "Album", Media: []mbz.MusicBrainzMedium{{Position: 1, Tracks: []mbz.MusicBrainzReleaseTrack{track}}}}
	client := &occurrenceReleaseFixture{releases: map[uuid.UUID]*mbz.MusicBrainzRelease{request: release},
		recordings: map[uuid.UUID]*mbz.MusicBrainzTrack{recording: {ID: fixtureRecording}}}
	item := ingest.Snapshot{Title: track.Title, Artist: "Artist", Release: release.Title, ReleaseMBID: request.String(), Recording: recording.String(), TrackMBID: track.ID}
	proof := verifyOccurrence(context.Background(), item, client)
	require.False(t, proof.conflict)
	require.Equal(t, canonical.String(), proof.release)
	require.Equal(t, fixtureRecording, proof.recording)
	delete(client.recordings, recording)
	proof = verifyOccurrence(context.Background(), item, client)
	require.False(t, proof.conflict)
	require.True(t, proof.ignoreDeclaredRecording)
	require.Empty(t, proof.recording)
	client.recordings[recording] = &mbz.MusicBrainzTrack{ID: recording.String()}
	proof = verifyOccurrence(context.Background(), item, client)
	require.True(t, proof.conflict)
	require.Equal(t, recording.String(), proof.evidence.RecordingReturned)
	conflicted := convertedOccurrenceFixture(t)
	conflictID := admitOccurrenceFixture(t, conflicted, item, 1300, "recording-conflict")
	resolveOccurrenceFixture(t, conflicted, client)
	requireReceiptState(t, conflicted, conflictID, "pending_identity")
	require.Equal(t, 1, scalar(t, conflicted, `SELECT count(*) FROM ingest_events WHERE receipt_id=?
		AND json_extract(evidence,'$.musicbrainz.release_requested')=? AND json_extract(evidence,'$.musicbrainz.release_returned')=?
		AND json_extract(evidence,'$.musicbrainz.release_title')=? AND json_extract(evidence,'$.musicbrainz.recording_declared')=?
		AND json_extract(evidence,'$.musicbrainz.recording_returned')=? AND json_extract(evidence,'$.musicbrainz.track_recording')=?
		AND json_extract(evidence,'$.musicbrainz.track_declared')=? AND json_extract(evidence,'$.musicbrainz.track_returned')=?`,
		conflictID, request.String(), canonical.String(), release.Title, recording.String(), recording.String(), fixtureRecording, track.ID, track.ID))
	item.Release = "Presented album"
	s := convertedOccurrenceFixture(t)
	// The raw track ID already exists under a different album and must not steer metadata fallback.
	execSQL(t, s, `UPDATE occurrences SET track_mbid=? WHERE id=1`, track.ID)
	id := admitOccurrenceFixture(t, s, item, 1300, "title-disagreement")
	resolveOccurrenceFixture(t, s, client)
	requireReceiptState(t, s, id, "accepted")
	require.Equal(t, 1, scalar(t, s, `SELECT count(*) FROM ingest_events WHERE receipt_id=?
		AND json_extract(evidence,'$.musicbrainz.release_title_differs')=1
		AND json_extract(evidence,'$.musicbrainz.release_returned')=?`, id, canonical.String()))
	require.Equal(t, 1, scalar(t, s, `SELECT count(*) FROM release_aliases a JOIN releases r ON r.id=a.release_id
		WHERE a.alias='Presented album' AND r.musicbrainz_id IS NULL`))
}

// TestOccurrenceExternalRecordingClaim preserves the validated recording on a conflicting service hint.
func TestOccurrenceExternalRecordingClaim(t *testing.T) {
	s := convertedOccurrenceFixture(t)
	execSQL(t, s, `INSERT INTO release_aliases VALUES(1,'Album','Fixture',1);
		UPDATE recordings SET musicbrainz_id=?,mbid_source='validated' WHERE id=(SELECT recording_id FROM occurrences WHERE id=1);
		UPDATE track_aliases SET alias='Single song' WHERE track_id=1 AND is_primary=1`, fixtureRecording)
	item := ingest.Snapshot{Title: "Single song", Artist: "Artist", Release: "Album", Recording: uuid.NewString(),
		MBIDSource: "external-service", MBIDDeclaredAt: "2026-11-20T00:00:00Z"}
	id := admitOccurrenceFixture(t, s, item, 1400, "external-hint")
	resolveOccurrenceFixture(t, s, nil)
	requireReceiptState(t, s, id, "accepted")
	require.Equal(t, 1, scalar(t, s, `SELECT count(*) FROM ingest_events e JOIN recordings r ON r.id=e.recording_id
		WHERE e.receipt_id=? AND e.occurrence_id=1 AND r.musicbrainz_id=?
		AND json_extract(e.original_snapshot,'$.mbid_source')='external-service'`, id, fixtureRecording))
	item.MBIDSource = ""
	id = admitOccurrenceFixture(t, s, item, 1410, "source-contradiction")
	resolveOccurrenceFixture(t, s, nil)
	requireReceiptState(t, s, id, "pending_identity")
	require.Equal(t, 1, scalar(t, s, `SELECT count(*) FROM ingest_events WHERE receipt_id=? AND json_extract(evidence,'$.candidates[0]')='occurrence:1'`, id))
}

// TestOccurrenceProvenTrackDisambiguatesRelease uses the proof before the raw track claim.
func TestOccurrenceProvenTrackDisambiguatesRelease(t *testing.T) {
	ctx := context.Background()
	s := convertedOccurrenceFixture(t)
	proven := uuid.NewString()
	execSQL(t, s, `INSERT INTO releases(id) VALUES(2);
		INSERT INTO release_aliases VALUES(1,'Same album','Fixture',1),(2,'Same album','Fixture',1);
		INSERT INTO artist_releases VALUES(1,2,1);
		UPDATE occurrences SET track_mbid=? WHERE id=1`, proven)
	require.NoError(t, s.withIngestWriter(ctx, func(tx *sql.Conn) error {
		match, candidates, err := resolveOccurrence(ctx, tx, ingest.Snapshot{Title: "Intro", Artist: "Artist", Release: "Same album", TrackMBID: uuid.NewString()},
			occurrenceProof{track: proven})
		require.NoError(t, err)
		require.Empty(t, candidates)
		require.EqualValues(t, 1, match.id)
		return nil
	}))
}

// TestOccurrenceWriteConflicts exposes the three SQL boundaries as typed administrative conflicts.
func TestOccurrenceWriteConflicts(t *testing.T) {
	ctx := context.Background()
	s := convertedOccurrenceFixture(t)
	require.ErrorIs(t, s.SetProfileRelay(ctx, 1, "https://fixture.invalid", "token"), db.ErrConflict)
	require.ErrorIs(t, s.SaveListen(ctx, db.SaveListenOpts{ProfileID: 1, TrackID: 1, Time: time.Unix(1500, 0)}), db.ErrConflict)
	execSQL(t, s, `UPDATE recordings SET musicbrainz_id=?,mbid_source='validated' WHERE id=(SELECT recording_id FROM occurrences WHERE id=1)`, fixtureRecording)
	err := s.UpdateTrack(ctx, db.UpdateTrackOpts{ID: 70, MusicBrainzID: uuid.MustParse(fixtureRecording)})
	require.ErrorIs(t, err, db.ErrConflict)
	require.ErrorContains(t, err, "recordings.musicbrainz_id")
}

// TestOccurrenceLinkedReleaseGuard rejects merge reassignment and permits release deletion.
func TestOccurrenceLinkedReleaseGuard(t *testing.T) {
	s := convertedOccurrenceFixture(t)
	item := ingest.Snapshot{Title: "Track", Artist: "Artist", Release: "Bound release"}
	id := admitOccurrenceFixture(t, s, item, 1600, "release-guard")
	resolveOccurrenceFixture(t, s, nil)
	requireReceiptState(t, s, id, "accepted")
	release := scalar(t, s, `SELECT release_id FROM release_aliases WHERE alias='Bound release'`)
	err := s.MergeAlbums(context.Background(), int32(release), 1, false)
	require.ErrorIs(t, err, db.ErrConflict)
	require.ErrorContains(t, err, "Cannot move release")
	requireReceiptState(t, s, id, "accepted")
	execSQL(t, s, `DELETE FROM releases WHERE id=?`, release)
	require.Zero(t, scalar(t, s, `SELECT count(*) FROM ingest_events WHERE receipt_id=?`, id))
	require.NoError(t, checkIntegrity(s.db))
}

// TestOccurrenceImportMasterOffReadsUnderContention keeps the legacy import gate read-only.
func TestOccurrenceImportMasterOffReadsUnderContention(t *testing.T) {
	ctx := context.Background()
	s, _, path := migrationDB(t, currentSchemaVersion)
	seedIdentityHistory(t, s)
	execSQL(t, s, `DELETE FROM ingest_events WHERE profile_id=1 AND mode='shadow';
		INSERT INTO profile_ingest_flags(profile_id,active,occurrence_write) VALUES(1,1,1)`)
	other, err := sql.Open("sqlite", sqliteURI(path)+"?mode=rw")
	require.NoError(t, err)
	defer other.Close()
	_, err = other.ExecContext(ctx, "BEGIN IMMEDIATE")
	require.NoError(t, err)
	defer func() { _, err := other.ExecContext(ctx, "ROLLBACK"); require.NoError(t, err) }()
	active, err := s.BeginImportIngest(ctx, 2, false)
	require.NoError(t, err)
	require.False(t, active.Active)
	active, err = s.BeginImportIngest(ctx, 1, false)
	var unavailable *ingest.ErrUnavailable
	require.ErrorAs(t, err, &unavailable)
	require.False(t, active.Active)
}

// TestOccurrenceLegacyUnpositionedCharacterization records how textual matching reuses a legacy bridge.
func TestOccurrenceLegacyUnpositionedCharacterization(t *testing.T) {
	s := convertedOccurrenceFixture(t)
	execSQL(t, s, `DELETE FROM tracks WHERE id=70;
		INSERT INTO release_aliases VALUES(1,'Legacy album','Fixture',1)`)
	item := ingest.Snapshot{Title: "Intro", Artist: "Artist", Release: "Legacy album", TrackNumber: 1}
	// The legacy bridge has no position, so either asserted disc matches its text.
	for _, disc := range []int{1, 2} {
		item.DiscNumber = disc
		id := admitOccurrenceFixture(t, s, item, int64(1700+10*disc), uuid.NewString())
		resolveOccurrenceFixture(t, s, nil)
		requireReceiptState(t, s, id, "accepted")
		require.Equal(t, 1, scalar(t, s, `SELECT count(*) FROM ingest_events WHERE receipt_id=? AND occurrence_id=1`, id))
	}
	require.Equal(t, 1, scalar(t, s, `SELECT count(*) FROM occurrences WHERE id=1 AND disc_number IS NULL AND track_number IS NULL`))
}

// TestOccurrenceReimportAfterDeletion recreates only missing ordinals in keyed API and file deliveries.
func TestOccurrenceReimportAfterDeletion(t *testing.T) {
	for _, source := range []string{ingest.ShadowSourceListenBrainz, ingest.ShadowSourceFile} {
		t.Run(source, func(t *testing.T) {
			ctx := context.Background()
			s := convertedOccurrenceFixture(t)
			batch := ingest.Batch{ProfileID: 1, Source: source, Operation: "import", ReceivedAt: time.Now(),
				Items: []ingest.Item{
					{Ordinal: 0, ListenedAt: time.Unix(1800, 0), Snapshot: ingest.Snapshot{Title: "A", Artist: "Artist", Release: "Album"}},
					{Ordinal: 1, ListenedAt: time.Unix(1820, 0), Snapshot: ingest.Snapshot{Title: "B", Artist: "Artist", Release: "Album"}},
				}}
			if source == ingest.ShadowSourceFile {
				batch.FileFormat, batch.FileDigest = "lastfm", ingest.Digest([]byte("file"))
			} else {
				batch.ClientKey = "batch"
			}
			_, first, err := s.AdmitIngest(ctx, batch, true, nil, nil)
			require.NoError(t, err)
			resolveOccurrenceFixture(t, s, nil)
			resolveOccurrenceFixture(t, s, nil)
			var track int32
			var oldEvent string
			require.NoError(t, s.db.QueryRow(`SELECT occurrence_id,event_id FROM ingest_events WHERE receipt_id=?`, first[0].ReceiptID).Scan(&track, &oldEvent))
			require.NoError(t, s.DeleteListen(db.WithProfile(ctx, 1), track, time.Unix(1800, 0)))
			require.Zero(t, scalar(t, s, `SELECT count(*) FROM ingest_revisions WHERE receipt_id=?`, first[0].ReceiptID))
			_, partial, err := s.AdmitIngest(ctx, batch, true, nil, nil)
			require.NoError(t, err)
			require.NotEqual(t, first[0].ReceiptID, partial[0].ReceiptID)
			require.Equal(t, first[1].ReceiptID, partial[1].ReceiptID)
			resolveOccurrenceFixture(t, s, nil)
			require.Zero(t, scalar(t, s, `SELECT count(*) FROM listens WHERE event_id=?`, oldEvent))
			require.Equal(t, 2, scalar(t, s, `SELECT count(*) FROM listens WHERE profile_id=1 AND listened_at IN (1800,1820)`))
			execSQL(t, s, `DELETE FROM listens WHERE profile_id=1 AND listened_at IN (1800,1820)`)
			_, full, err := s.AdmitIngest(ctx, batch, true, nil, nil)
			require.NoError(t, err)
			require.NotEqual(t, partial[0].ReceiptID, full[0].ReceiptID)
			require.NotEqual(t, partial[1].ReceiptID, full[1].ReceiptID)
		})
	}
}

// TestOccurrenceDiscardPending revokes an in-flight lease and releases the key.
func TestOccurrenceDiscardPending(t *testing.T) {
	ctx := context.Background()
	s := convertedOccurrenceFixture(t)
	snapshot := ingest.Snapshot{Title: "Discarded", Artist: "Artist", Release: "Album"}
	id := admitOccurrenceFixture(t, s, snapshot, 1900, "discard")
	lease, _, _, err := s.claimNextIngest(ctx)
	require.NoError(t, err)
	require.Equal(t, id, lease.id)
	require.ErrorIs(t, s.DiscardIngestPending(ctx, 2, id), db.ErrNotFound)
	require.NoError(t, s.DiscardIngestPending(ctx, 1, id))
	lease.snapshot = snapshot
	lease.unix = 1900
	require.NoError(t, s.finishOccurrence(ctx, lease, occurrenceProof{}))
	require.Zero(t, scalar(t, s, `SELECT count(*) FROM listens WHERE listened_at=1900`))
	require.NotEqual(t, id, admitOccurrenceFixture(t, s, snapshot, 1900, "discard"))
}

// TestExpandedV9SnapshotReplay upgrades a pre-v10 key and records the enriched payload as an observation.
func TestExpandedV9SnapshotReplay(t *testing.T) {
	ctx := context.Background()
	s, provider, _ := migrationDB(t, 9)
	seedIdentityHistory(t, s)
	batch := ingest.Batch{ProfileID: 1, Source: ingest.ShadowSourceListenBrainz, Operation: "single", ClientKey: "stable", ReceivedAt: time.Now(),
		Items: []ingest.Item{{ListenedAt: time.Unix(1780000000, 0), Snapshot: ingest.Snapshot{Title: "Song", Artist: "Artist", Release: "Album"}}}}
	snapshot, digest := batch.Items[0].Encode()
	execSQL(t, s, `INSERT INTO integrations(id,profile_id,source_type,created_at) VALUES('listenbrainz:1',1,'listenbrainz',0);
		INSERT INTO profile_ingest_flags(profile_id,active) VALUES(1,1);
		INSERT INTO ingest_events(receipt_id,profile_id,source_instance_id,schema_version,source_type,adapter_version,batch_id,
			item_ordinal,operation,received_at,time_precision,time_semantics,original_snapshot,snapshot_digest,digest_version,mode,
			client_idempotency_key,batch_digest)
		VALUES('old',1,'listenbrainz:1',1,'listenbrainz','1','seed',0,'single',0,'second','unknown',?,?,?,'active','stable','old')`,
		string(snapshot), digest, ingest.DigestVersion)
	_, err := provider.Up(ctx)
	require.NoError(t, err)
	batch.Items[0].Snapshot.TrackNumber = 2
	for range 2 {
		active, results, err := s.AdmitIngest(ctx, batch, true, nil, nil)
		require.NoError(t, err)
		require.True(t, active)
		require.Len(t, results, 1)
		require.True(t, results[0].Replay)
		require.Equal(t, "old", results[0].ReceiptID)
	}
	require.Equal(t, 1, scalar(t, s, `SELECT count(*) FROM ingest_events WHERE mode='active' AND resolver_variant='v9'`))
	require.Equal(t, 1, scalar(t, s, `SELECT count(*) FROM ingest_observations WHERE receipt_id='old' AND json_extract(original_snapshot,'$.track_number')=2`))
}

// TestOccurrenceReleaseProofCache reuses successful evidence and retries unavailable evidence.
func TestOccurrenceReleaseProofCache(t *testing.T) {
	var cache occurrenceReleaseCache
	id := uuid.New()
	calls := 0
	lookup := func(context.Context, uuid.UUID) (*mbz.MusicBrainzRelease, error) {
		calls++
		if calls == 1 {
			return nil, errors.New("fixture unavailable")
		}
		return &mbz.MusicBrainzRelease{ID: id.String()}, nil
	}
	_, err := cache.occurrenceReleaseLookup(context.Background(), id, lookup)
	require.Error(t, err)
	for range 2 {
		release, err := cache.occurrenceReleaseLookup(context.Background(), id, lookup)
		require.NoError(t, err)
		require.Equal(t, id.String(), release.ID)
	}
	require.Equal(t, 2, calls)
}
