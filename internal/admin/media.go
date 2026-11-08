package admin

import (
	"context"
	"strings"
	"unicode/utf8"

	"github.com/ebrgr/TuneTrail/internal/db"
	"github.com/ebrgr/TuneTrail/internal/models"
	"github.com/google/uuid"
)

// MediaStore contains the shared catalog operations required by administrative editors.
type MediaStore interface {
	db.ArtistStore
	db.AlbumStore
	db.TrackStore
}

// MediaExists resolves an explicitly identified catalog item outside personal read scope.
func MediaExists(ctx context.Context, s MediaStore, user *models.User, kind string, id int32) error {
	if err := Authorize(user); err != nil {
		return err
	}
	if id <= 0 {
		return db.ErrInvalidInput
	}
	ctx = db.WithoutProfile(ctx)
	switch kind {
	case "artist":
		_, err := s.GetArtist(ctx, db.GetArtistOpts{ID: id})
		return err
	case "album":
		_, err := s.GetAlbum(ctx, db.GetAlbumOpts{ID: id})
		return err
	case "track":
		_, err := s.GetTrack(ctx, db.GetTrackOpts{ID: id})
		return err
	default:
		return db.ErrInvalidInput
	}
}

// EditAlias applies the same rules to administrative HTML and JSON adapters.
func EditAlias(ctx context.Context, s MediaStore, user *models.User, kind string, id int32, action, alias string) error {
	if err := MediaExists(ctx, s, user, kind, id); err != nil {
		return err
	}
	if action == "add" {
		alias = strings.TrimSpace(alias)
	}
	if strings.TrimSpace(alias) == "" || len(alias) > 512 || !utf8.ValidString(alias) {
		return db.ErrInvalidInput
	}
	ctx = db.WithoutProfile(ctx)
	switch kind {
	case "artist":
		switch action {
		case "add":
			return s.SaveArtistAliases(ctx, id, []string{alias}, "Manual")
		case "primary":
			return s.SetPrimaryArtistAlias(ctx, id, alias)
		case "delete":
			return s.DeleteArtistAlias(ctx, id, alias)
		}
	case "album":
		switch action {
		case "add":
			return s.SaveAlbumAliases(ctx, id, []string{alias}, "Manual")
		case "primary":
			return s.SetPrimaryAlbumAlias(ctx, id, alias)
		case "delete":
			return s.DeleteAlbumAlias(ctx, id, alias)
		}
	case "track":
		switch action {
		case "add":
			return s.SaveTrackAliases(ctx, id, []string{alias}, "Manual")
		case "primary":
			return s.SetPrimaryTrackAlias(ctx, id, alias)
		case "delete":
			return s.DeleteTrackAlias(ctx, id, alias)
		}
	}
	return db.ErrInvalidInput
}

// UpdateMediaMetadata validates MBID and changes only fields offered by the existing editor.
func UpdateMediaMetadata(ctx context.Context, s MediaStore, user *models.User, kind string, id int32, rawMBID string, various *bool) error {
	if err := MediaExists(ctx, s, user, kind, id); err != nil {
		return err
	}
	var mbid uuid.UUID
	var err error
	if rawMBID != "" {
		mbid, err = uuid.Parse(rawMBID)
		if err != nil || mbid == uuid.Nil {
			return db.ErrInvalidInput
		}
	}
	if mbid == uuid.Nil && various == nil {
		return db.ErrInvalidInput
	}
	ctx = db.WithoutProfile(ctx)
	switch kind {
	case "artist":
		if various != nil {
			return db.ErrInvalidInput
		}
		return s.UpdateArtist(ctx, db.UpdateArtistOpts{ID: id, MusicBrainzID: mbid})
	case "album":
		o := db.UpdateAlbumOpts{ID: id, MusicBrainzID: mbid}
		if various != nil {
			o.VariousArtistsUpdate = true
			o.VariousArtistsValue = *various
		}
		return s.UpdateAlbum(ctx, o)
	case "track":
		if various != nil {
			return db.ErrInvalidInput
		}
		return s.UpdateTrack(ctx, db.UpdateTrackOpts{ID: id, MusicBrainzID: mbid})
	}
	return db.ErrInvalidInput
}

// SetMediaArtistPrimary refuses to create a relationship implicitly.
func SetMediaArtistPrimary(ctx context.Context, s MediaStore, user *models.User, kind string, id, artistID int32, value bool) error {
	if err := MediaExists(ctx, s, user, kind, id); err != nil {
		return err
	}
	if artistID <= 0 {
		return db.ErrInvalidInput
	}
	ctx = db.WithoutProfile(ctx)
	switch kind {
	case "album":
		return s.SetPrimaryAlbumArtist(ctx, id, artistID, value)
	case "track":
		return s.SetPrimaryTrackArtist(ctx, id, artistID, value)
	}
	return db.ErrInvalidInput
}

// ChangeTrackArtist adds an existing artist or removes a non-primary relation only.
func ChangeTrackArtist(ctx context.Context, s MediaStore, user *models.User, trackID, artistID int32, add bool) error {
	if err := MediaExists(ctx, s, user, "track", trackID); err != nil {
		return err
	}
	if artistID <= 0 {
		return db.ErrInvalidInput
	}
	ctx = db.WithoutProfile(ctx)
	opts := db.UpdateTrackOpts{ID: trackID}
	if add {
		opts.AddArtists = []int32{artistID}
	} else {
		opts.RemoveArtists = []int32{artistID}
	}
	return s.UpdateTrack(ctx, opts)
}
