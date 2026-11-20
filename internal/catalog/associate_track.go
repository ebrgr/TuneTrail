package catalog

import (
	"context"
	"errors"
	"fmt"

	"github.com/ebrgr/TuneTrail/internal/db"
	"github.com/ebrgr/TuneTrail/internal/logger"
	"github.com/ebrgr/TuneTrail/internal/mbz"
	"github.com/ebrgr/TuneTrail/internal/models"
	"github.com/google/uuid"
)

type AssociateTrackOpts struct {
	ArtistIDs  []int32
	AlbumID    int32
	TrackMbzID uuid.UUID
	TrackName  string
	Duration   int32
	Mbzc       mbz.MusicBrainzCaller
}

// AssociateTrack resolves a recording ID or an unambiguous alias in its release context.
func AssociateTrack(ctx context.Context, d db.TrackStore, opts AssociateTrackOpts) (*models.Track, error) {
	l := logger.FromContext(ctx)
	if opts.TrackName == "" {
		return nil, errors.New("AssociateTrack: missing required parameter 'opts.TrackName'")
	}
	if len(opts.ArtistIDs) < 1 {
		return nil, errors.New("AssociateTrack: at least one artist id must be specified")
	}
	if opts.AlbumID == 0 {
		return nil, errors.New("AssociateTrack: release group id must be specified")
	}
	// first, try to match track Mbz ID
	if opts.TrackMbzID != uuid.Nil {
		l.Debug().Msgf("Associating track '%s' by MusicBrainz recording ID", opts.TrackName)
		return matchTrackByMbzID(ctx, d, opts)
	} else {
		l.Debug().Msgf("Associating track '%s' by title and artist", opts.TrackName)
		return matchTrackByTrackInfo(ctx, d, opts)
	}
}

// matchTrackByMbzID fills missing IDs without replacing a concurrent identity.
func matchTrackByMbzID(ctx context.Context, d db.TrackStore, opts AssociateTrackOpts) (*models.Track, error) {
	track, err := d.GetTrack(ctx, db.GetTrackOpts{MusicBrainzID: opts.TrackMbzID})
	if err == nil {
		return track, nil
	}
	if !errors.Is(err, db.ErrNotFound) {
		return nil, fmt.Errorf("matchTrackByMbzID: %w", err)
	}
	track, err = matchTrackByTrackInfo(ctx, d, opts)
	if err != nil {
		return nil, fmt.Errorf("matchTrackByMbzID: %w", err)
	}
	if track.MbzID == nil || *track.MbzID == uuid.Nil {
		if err := d.UpdateTrack(ctx, db.UpdateTrackOpts{
			ID: track.ID, MusicBrainzID: opts.TrackMbzID, OnlyFillMusicBrainzID: true,
		}); err != nil {
			return nil, fmt.Errorf("matchTrackByMbzID: fill recording ID: %w", err)
		}
		// Keep the requested MBID available for legacy duration lookup even if the fill was skipped.
		track.MbzID = &opts.TrackMbzID
	}
	return track, nil
}

// validateTrackIdentity rejects incompatible recording IDs before returning a title match.
func validateTrackIdentity(track *models.Track, incoming uuid.UUID) (*models.Track, error) {
	if incoming != uuid.Nil && track.MbzID != nil && *track.MbzID != uuid.Nil && *track.MbzID != incoming {
		return nil, fmt.Errorf("track %d: %w", track.ID, db.ErrTrackIdentityConflict)
	}
	return track, nil
}

// matchTrackByTrackInfo only creates a track after both title lookups report no match.
func matchTrackByTrackInfo(ctx context.Context, d db.TrackStore, opts AssociateTrackOpts) (*models.Track, error) {
	l := logger.FromContext(ctx)
	// try provided track title
	track, err := d.GetTrack(ctx, db.GetTrackOpts{
		Title:     opts.TrackName,
		ReleaseID: opts.AlbumID,
		ArtistIDs: opts.ArtistIDs,
	})
	if err == nil {
		l.Debug().Msgf("Track '%s' found by title, release and artist match", track.Title)
		return validateTrackIdentity(track, opts.TrackMbzID)
	} else if !errors.Is(err, db.ErrNotFound) {
		return nil, fmt.Errorf("matchTrackByTrackInfo: %w", err)
	} else {
		if opts.TrackMbzID != uuid.Nil && opts.Mbzc != nil {
			mbzTrack, err := opts.Mbzc.GetTrack(ctx, opts.TrackMbzID)
			if err == nil {
				track, err := d.GetTrack(ctx, db.GetTrackOpts{
					Title:     mbzTrack.Title,
					ReleaseID: opts.AlbumID,
					ArtistIDs: opts.ArtistIDs,
				})
				if err == nil {
					return validateTrackIdentity(track, opts.TrackMbzID)
				}
				if !errors.Is(err, db.ErrNotFound) {
					return nil, fmt.Errorf("matchTrackByTrackInfo: MusicBrainz title: %w", err)
				}
			}
		}
		l.Debug().Msgf("Track '%s' could not be found by title and artist match", opts.TrackName)
		t, err := d.SaveTrack(ctx, db.SaveTrackOpts{
			RecordingMbzID: opts.TrackMbzID,
			AlbumID:        opts.AlbumID,
			Title:          opts.TrackName,
			ArtistIDs:      opts.ArtistIDs,
			Duration:       opts.Duration,
		})
		if err != nil {
			return nil, fmt.Errorf("matchTrackByTrackInfo: %w", err)
		}
		if opts.TrackMbzID == uuid.Nil {
			l.Info().Msgf("Created track '%s' with title and artist", opts.TrackName)
		} else {
			l.Info().Msgf("Created track '%s' with MusicBrainz Recording ID", opts.TrackName)
		}
		return t, nil
	}
}
