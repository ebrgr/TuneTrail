package catalog_test

import (
	"context"
	"errors"
	"testing"

	"github.com/ebrgr/TuneTrail/internal/catalog"
	"github.com/ebrgr/TuneTrail/internal/db"
	"github.com/ebrgr/TuneTrail/internal/mbz"
	"github.com/ebrgr/TuneTrail/internal/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// TestAssociateTrackAlias reuses a secondary alias without changing the primary title.
func TestAssociateTrackAlias(t *testing.T) {
	s := newTestDB()
	ctx := context.Background()
	t.Cleanup(func() { s.Close(ctx) })
	setupTestDataSansMbzIDs(s, t)
	require.NoError(t, s.SaveTrackAliases(ctx, 1, []string{"Secondary"}, "User"))
	opts := catalog.AssociateTrackOpts{TrackName: "Secondary", AlbumID: 1, ArtistIDs: []int32{1}}
	track, err := catalog.AssociateTrack(ctx, s, opts)
	require.NoError(t, err)
	require.EqualValues(t, 1, track.ID)
	require.Equal(t, "Tokyo Calling", track.Title)
	opts.TrackMbzID = uuid.New()
	track, err = catalog.AssociateTrack(ctx, s, opts)
	require.NoError(t, err)
	persisted, err := s.GetTrack(ctx, db.GetTrackOpts{ID: track.ID})
	require.NoError(t, err)
	require.Equal(t, track.MbzID, persisted.MbzID)
	opts.TrackMbzID = uuid.New()
	track, err = catalog.AssociateTrack(ctx, s, opts)
	require.ErrorIs(t, err, db.ErrTrackIdentityConflict)
	require.Nil(t, track)
}

type trackLookupStub struct {
	db.TrackStore
	lookup func(db.GetTrackOpts) (*models.Track, error)
	saves  int
}

// GetTrack injects lookup errors at either title lookup without touching a database.
func (s *trackLookupStub) GetTrack(_ context.Context, opts db.GetTrackOpts) (*models.Track, error) {
	return s.lookup(opts)
}

// SaveTrack records unintended creation after a failed identity lookup.
func (s *trackLookupStub) SaveTrack(context.Context, db.SaveTrackOpts) (*models.Track, error) {
	s.saves++
	return nil, errors.New("unexpected save")
}

// TestAssociateTrackLookupFailures propagates ambiguity, identity conflicts and database errors on both titles.
func TestAssociateTrackLookupFailures(t *testing.T) {
	incoming := uuid.MustParse("00000000-0000-0000-0000-000000001001")
	for _, secondary := range []bool{false, true} {
		for _, lookupErr := range []error{db.ErrAmbiguousTrack, errors.New("database failed"), db.ErrTrackIdentityConflict} {
			s := &trackLookupStub{}
			s.lookup = func(o db.GetTrackOpts) (*models.Track, error) {
				if o.MusicBrainzID != uuid.Nil || (secondary && o.Title == "Supplied") {
					return nil, db.ErrNotFound
				}
				if errors.Is(lookupErr, db.ErrTrackIdentityConflict) {
					id := uuid.New()
					return &models.Track{ID: 1, MbzID: &id}, nil
				}
				return nil, lookupErr
			}
			track, err := catalog.AssociateTrack(context.Background(), s, catalog.AssociateTrackOpts{
				TrackName: "Supplied", AlbumID: 1, ArtistIDs: []int32{1}, TrackMbzID: incoming,
				Mbzc: &mbz.MbzMockCaller{Tracks: mbzTrackData},
			})
			require.ErrorIs(t, err, lookupErr)
			require.Nil(t, track)
			require.Zero(t, s.saves)
		}
	}
}

// TestAssociateTrackAmbiguousWithoutMBID rejects a title ambiguity even without external metadata.
func TestAssociateTrackAmbiguousWithoutMBID(t *testing.T) {
	s := &trackLookupStub{lookup: func(db.GetTrackOpts) (*models.Track, error) { return nil, db.ErrAmbiguousTrack }}
	track, err := catalog.AssociateTrack(context.Background(), s, catalog.AssociateTrackOpts{TrackName: "Alias", AlbumID: 1, ArtistIDs: []int32{1}})
	require.ErrorIs(t, err, db.ErrAmbiguousTrack)
	require.Nil(t, track)
	require.Zero(t, s.saves)
}
