// Package catalog manages the internal metadata of the catalog of music the user has submitted listens for.
// This includes artists, releases (album, single, ep, etc), and tracks, as well as ingesting
// listens submitted both via the API(s) and other methods.
package catalog

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/ebrgr/TuneTrail/internal/db"
	"github.com/ebrgr/TuneTrail/internal/logger"
	"github.com/ebrgr/TuneTrail/internal/mbz"
	"github.com/ebrgr/TuneTrail/internal/memkv"
	"github.com/ebrgr/TuneTrail/internal/models"
	"github.com/google/uuid"
)

type ArtistMbidMap struct {
	Artist string
	Mbid   uuid.UUID
}

type SubmitListenOpts struct {
	// When true, skips caching the images and only stores the image url in the db
	SkipCacheImage bool

	MbzCaller          mbz.MusicBrainzCaller
	ArtistNames        []string
	Artist             string
	ArtistMbzIDs       []uuid.UUID
	ArtistMbidMappings []ArtistMbidMap
	TrackTitle         string
	RecordingMbzID     uuid.UUID
	Duration           int32 // in seconds
	ReleaseTitle       string
	ReleaseMbzID       uuid.UUID
	ReleaseGroupMbzID  uuid.UUID
	Time               time.Time

	UserID    int32
	ProfileID int32
	Client    string
}

const (
	ImageSourceUserUpload = "User Upload"
)

type listenCatalogStore interface {
	db.ArtistStore
	db.AlbumStore
	db.TrackStore
}

type listenAssociation struct {
	track      *models.Track
	artists    []*models.Artist
	albumTitle string
}

var submitListenGate = make(chan struct{}, 1)

// LockAssociations serializes read-then-create catalog operations with cancellation.
func LockAssociations(ctx context.Context) (func(), error) {
	select {
	case submitListenGate <- struct{}{}:
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	if err := ctx.Err(); err != nil {
		<-submitListenGate
		return nil, err
	}
	return func() { <-submitListenGate }, nil
}

// PublishNowPlaying resolves the displayed track separately from historical admission.
func PublishNowPlaying(ctx context.Context, store listenCatalogStore, opts SubmitListenOpts) error {
	unlock, err := LockAssociations(ctx)
	if err != nil {
		return err
	}
	defer unlock()
	association, err := resolveListenAssociation(db.WithoutProfile(ctx), store, opts)
	if err != nil {
		return err
	}
	setNowPlaying(opts.ProfileID, association.track)
	return nil
}

// setNowPlaying keeps the existing cache lifetime for the chosen occurrence.
func setNowPlaying(profile int32, track *models.Track) {
	key := fmt.Sprintf("now-playing:%d", profile)
	if track.Duration == 0 {
		memkv.Store.Set(key, track.ID)
	} else {
		memkv.Store.Set(key, track.ID, time.Duration(track.Duration)*time.Second)
	}
}

// resolveListenAssociation resolves shared catalog identity under the caller's
// association lock. The caller owns validation and any listen or now-playing write.
func resolveListenAssociation(ctx context.Context, store listenCatalogStore, opts SubmitListenOpts) (listenAssociation, error) {
	l := logger.FromContext(ctx)

	artists, err := AssociateArtists(
		ctx,
		store,
		AssociateArtistsOpts{
			ArtistMbzIDs:   opts.ArtistMbzIDs,
			ArtistNames:    opts.ArtistNames,
			ArtistName:     opts.Artist,
			ArtistMbidMap:  opts.ArtistMbidMappings,
			Mbzc:           opts.MbzCaller,
			TrackTitle:     opts.TrackTitle,
			SkipCacheImage: opts.SkipCacheImage,
		})
	if err != nil {
		l.Err(err).Msg("Failed to associate artists to listen")
		return listenAssociation{}, fmt.Errorf("SubmitListen: %w", err)
	} else if len(artists) < 1 {
		l.Debug().Msg("Failed to associate any artists to release")
	}

	artistIDs := make([]int32, len(artists))

	for i, artist := range artists {
		artistIDs[i] = artist.ID
		l.Debug().Any("artist", artist).Msg("Matched listen to artist")
	}
	rg, err := AssociateAlbum(ctx, store, AssociateAlbumOpts{
		ReleaseMbzID:      opts.ReleaseMbzID,
		ReleaseGroupMbzID: opts.ReleaseGroupMbzID,
		ReleaseName:       opts.ReleaseTitle,
		TrackName:         opts.TrackTitle,
		Mbzc:              opts.MbzCaller,
		Artists:           artists,
		SkipCacheImage:    opts.SkipCacheImage,
	})
	if err != nil {
		l.Error().Err(err).Msg("Failed to associate release group to listen")
		return listenAssociation{}, fmt.Errorf("SubmitListen: %w", err)
	}
	l.Debug().Any("album", rg).Msg("Matched listen to release")

	// ensure artists are associated with release group
	if err := store.AddArtistsToAlbum(ctx, db.AddArtistsToAlbumOpts{
		Derived:   true, // Track metadata must not promote an album credit to independent.
		ArtistIDs: artistIDs,
		AlbumID:   rg.ID,
	}); err != nil {
		return listenAssociation{}, fmt.Errorf("SubmitListen: associate album artists: %w", err)
	}

	track, err := AssociateTrack(ctx, store, AssociateTrackOpts{
		ArtistIDs:  artistIDs,
		AlbumID:    rg.ID,
		TrackMbzID: opts.RecordingMbzID,
		TrackName:  opts.TrackTitle,
		Duration:   opts.Duration,
		Mbzc:       opts.MbzCaller,
	})
	if err != nil {
		l.Error().Err(err).Msg("Failed to associate track to listen")
		return listenAssociation{}, fmt.Errorf("SubmitListen: %w", err)
	}
	l.Debug().Any("track", track).Msg("Matched listen to track")

	if track.Duration == 0 {
		if opts.Duration != 0 {
			l.Debug().Msg("Updating duration using request information")
			err := store.UpdateTrack(ctx, db.UpdateTrackOpts{
				ID:       track.ID,
				Duration: opts.Duration,
			})
			if err != nil {
				l.Err(err).Msgf("Failed to update duration for track %s", track.Title)
			} else {
				l.Info().Msgf("Duration updated to %d for track '%s'", opts.Duration, track.Title)
			}
		} else if track.MbzID != nil && *track.MbzID != uuid.Nil {
			l.Debug().Msg("Attempting to update duration using MusicBrainz ID")
			mbztrack, err := opts.MbzCaller.GetTrack(ctx, *track.MbzID)
			if err != nil {
				l.Err(err).Msg("Failed to make request to MusicBrainz")
			} else {
				err = store.UpdateTrack(ctx, db.UpdateTrackOpts{
					ID:       track.ID,
					Duration: int32(mbztrack.LengthMs / 1000),
				})
				if err != nil {
					l.Err(err).Msgf("Failed to update duration for track %s", track.Title)
				} else {
					l.Info().Msgf("Duration updated to %d for track '%s'", mbztrack.LengthMs/1000, track.Title)
				}
			}
		}
	}

	return listenAssociation{track: track, artists: artists, albumTitle: rg.Title}, nil
}

var (
	// Bracketed feat patterns
	bracketFeatPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)\([fF]eat\. ([^)]*)\)`),
		regexp.MustCompile(`(?i)\[[fF]eat\. ([^\]]*)\]`),
	}
	// Inline feat (not in brackets)
	inlineFeatPattern = regexp.MustCompile(`(?i)[fF]eat\. ([^()\[\]]+)$`)

	// Delimiters only used inside feat. sections
	featSplitDelimiters = regexp.MustCompile(`(?i)\s*(?:,|&|and|·)\s*`)
)

// ParseArtists extracts all contributing artist names from the artist and title strings
func ParseArtists(artist string, title string, addlSeparators []*regexp.Regexp) []string {
	seen := make(map[string]struct{})
	var out []string

	add := func(name string) {
		name = strings.TrimSpace(name)
		if name == "" {
			return
		}
		if _, exists := seen[name]; !exists {
			seen[name] = struct{}{}
			out = append(out, name)
		}
	}

	// Extract bracketed features from artist
	for _, re := range bracketFeatPatterns {
		if matches := re.FindStringSubmatch(artist); matches != nil {
			artist = strings.Replace(artist, matches[0], "", 1)
			for _, name := range featSplitDelimiters.Split(matches[1], -1) {
				add(name)
			}
		}
	}
	// Extract inline feat. from artist
	if matches := inlineFeatPattern.FindStringSubmatch(artist); matches != nil {
		artist = strings.Replace(artist, matches[0], "", 1)
		for _, name := range featSplitDelimiters.Split(matches[1], -1) {
			add(name)
		}
	}

	// Add base artist(s)
	l1 := len(out)
	for _, re := range addlSeparators {
		for _, name := range re.Split(artist, -1) {
			if name == artist {
				continue
			}
			add(name)
		}
	}
	// Only add the full artist string if no splitters were matched
	if l1 == len(out) {
		add(artist)
	}

	// Extract features from title
	for _, re := range bracketFeatPatterns {
		if matches := re.FindStringSubmatch(title); matches != nil {
			for _, name := range featSplitDelimiters.Split(matches[1], -1) {
				add(name)
			}
		}
	}
	if matches := inlineFeatPattern.FindStringSubmatch(title); matches != nil {
		for _, name := range featSplitDelimiters.Split(matches[1], -1) {
			add(name)
		}
	}

	return out
}
