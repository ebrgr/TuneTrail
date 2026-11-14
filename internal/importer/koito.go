package importer

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/ebrgr/TuneTrail/internal/catalog"
	"github.com/ebrgr/TuneTrail/internal/db"
	"github.com/ebrgr/TuneTrail/internal/export"
	"github.com/ebrgr/TuneTrail/internal/logger"
	"github.com/ebrgr/TuneTrail/internal/models"
	"github.com/ebrgr/TuneTrail/internal/utils"
	"github.com/google/uuid"
	"math"
)

func ImportKoitoFile(ctx context.Context, store importStore, filename string) error {
	if db.ProfileID(ctx) <= 0 {
		return fmt.Errorf("explicit import profile required")
	}
	profileID := db.ProfileID(ctx)
	ctx = db.WithoutProfile(ctx)
	l := logger.FromContext(ctx)
	l.Info().Msgf("Beginning TuneTrail/Koito import on file: %s", filename) // same JSON schema (ISSUE-111)
	data := new(export.KoitoExport)
	f, err := os.Open(importPath(ctx, filename))
	if err != nil {
		return fmt.Errorf("ImportKoitoFile: os.Open: %w", err)
	}
	defer f.Close()
	err = decodeJSON(ctx, f, data)
	if err != nil {
		return fmt.Errorf("ImportKoitoFile: Decode: %w", err)
	}

	if data.Version != "1" {
		return invalidFile(fmt.Errorf("unsupported TuneTrail/Koito export version"))
	}

	if len(data.Listens) > MaxImportItems {
		return invalidFile(fmt.Errorf("too many import items"))
	}
	// Validate all required identities before writing anything. Uploaded exports
	// are untrusted and an empty artist list must not panic at artistIds[0].
	for _, item := range data.Listens {
		if err := ctx.Err(); err != nil {
			return err
		}
		if item.Track.Duration < 0 || item.Track.Duration > math.MaxInt32 {
			return invalidFile(fmt.Errorf("invalid track duration"))
		}
		if len(item.Artists) == 0 || getPrimaryAliasFromAliasSlice(item.Track.Aliases) == "" || getPrimaryAliasFromAliasSlice(item.Album.Aliases) == "" {
			return invalidFile(fmt.Errorf("invalid TuneTrail/Koito export: artists, album and track required"))
		}
		for _, artist := range item.Artists {
			if getPrimaryAliasFromAliasSlice(artist.Aliases) == "" {
				return invalidFile(fmt.Errorf("invalid TuneTrail/Koito export: artist name required"))
			}
		}
	}
	l.Info().Msg("Beginning TuneTrail/Koito data import")

	count := 0

	for i := range data.Listens {
		seen(ctx)
		if err := ctx.Err(); err != nil {
			return err
		}
		if !inImportTimeWindow(data.Listens[i].ListenedAt) {
			l.Debug().Msgf("Skipping import due to import time rules")
			ignored(ctx)
			continue
		}
		err := func() error {
			unlock, err := catalog.LockAssociations(ctx)
			if err != nil {
				return err
			}
			defer unlock()
			// use this for save/get mbid for all artist/album/track
			var mbid uuid.UUID

			artistIds := make([]int32, 0)
			for _, ia := range data.Listens[i].Artists {
				mbid = uuid.Nil
				if ia.MBID != nil {
					mbid = *ia.MBID
				}
				artist, err := store.GetArtist(ctx, db.GetArtistOpts{
					MusicBrainzID: mbid,
					Name:          getPrimaryAliasFromAliasSlice(ia.Aliases),
				})
				if errors.Is(err, db.ErrNotFound) {
					var imgid = uuid.Nil
					// not a perfect way to check if the image url is an actual source vs manual upload but
					// im like 99% sure it will work perfectly
					if strings.HasPrefix(ia.ImageUrl, "http") {
						imgid = uuid.New()
					}
					// save artist
					artist, err := store.SaveArtist(ctx, db.SaveArtistOpts{
						Name:          getPrimaryAliasFromAliasSlice(ia.Aliases),
						Image:         imgid,
						ImageSrc:      ia.ImageUrl,
						MusicBrainzID: mbid,
						Aliases:       utils.FlattenAliases(ia.Aliases),
					})
					if err != nil {
						return fmt.Errorf("ImportKoitoFile: %w", err)
					}
					artistIds = append(artistIds, artist.ID)
				} else if err != nil {
					return fmt.Errorf("ImportKoitoFile: %w", err)
				} else {
					artistIds = append(artistIds, artist.ID)
				}
			}
			// call associate album
			albumId := int32(0)
			mbid = uuid.Nil
			if data.Listens[i].Album.MBID != nil {
				mbid = *data.Listens[i].Album.MBID
			}
			album, err := store.GetAlbum(ctx, db.GetAlbumOpts{
				MusicBrainzID: mbid,
				Title:         getPrimaryAliasFromAliasSlice(data.Listens[i].Album.Aliases),
				ArtistID:      artistIds[0],
			})
			if errors.Is(err, db.ErrNotFound) {
				var imgid = uuid.Nil
				// not a perfect way to check if the image url is an actual source vs manual upload but
				// im like 99% sure it will work perfectly
				if strings.HasPrefix(data.Listens[i].Album.ImageUrl, "http") {
					imgid = uuid.New()
				}
				// save album
				album, err = store.SaveAlbum(ctx, db.SaveAlbumOpts{
					Title:          getPrimaryAliasFromAliasSlice(data.Listens[i].Album.Aliases),
					Image:          imgid,
					ImageSrc:       data.Listens[i].Album.ImageUrl,
					MusicBrainzID:  mbid,
					Aliases:        utils.FlattenAliases(data.Listens[i].Album.Aliases),
					ArtistIDs:      artistIds,
					VariousArtists: data.Listens[i].Album.VariousArtists,
				})
				if err != nil {
					return fmt.Errorf("ImportKoitoFile: %w", err)
				}
				albumId = album.ID
			} else if err != nil {
				return fmt.Errorf("ImportKoitoFile: %w", err)
			} else {
				albumId = album.ID
			}

			// call associate track
			mbid = uuid.Nil
			if data.Listens[i].Track.MBID != nil {
				mbid = *data.Listens[i].Track.MBID
			}
			track, err := store.GetTrack(ctx, db.GetTrackOpts{
				MusicBrainzID: mbid,
				Title:         getPrimaryAliasFromAliasSlice(data.Listens[i].Track.Aliases),
				ReleaseID:     albumId,
				ArtistIDs:     artistIds,
			})
			if errors.Is(err, db.ErrNotFound) {
				// save track
				track, err = store.SaveTrack(ctx, db.SaveTrackOpts{
					Title:          getPrimaryAliasFromAliasSlice(data.Listens[i].Track.Aliases),
					RecordingMbzID: mbid,
					Duration:       int32(data.Listens[i].Track.Duration),
					ArtistIDs:      artistIds,
					AlbumID:        albumId,
				})
				if err != nil {
					return fmt.Errorf("ImportKoitoFile: %w", err)
				}
				// save track aliases
				err = store.SaveTrackAliases(ctx, track.ID, utils.FlattenAliases(data.Listens[i].Track.Aliases), "Import")
				if err != nil {
					return fmt.Errorf("ImportKoitoFile: %w", err)
				}
			} else if err != nil {
				return fmt.Errorf("ImportKoitoFile: %w", err)
			}

			// save listen
			err = store.SaveListen(ctx, db.SaveListenOpts{
				TrackID:   track.ID,
				Time:      data.Listens[i].ListenedAt,
				Client:    data.Listens[i].Client,
				ProfileID: profileID,
			})
			if err != nil {
				return fmt.Errorf("ImportKoitoFile: %w", err)
			}

			l.Info().Msgf("ImportKoitoFile: Imported listen for track %s", track.Title)
			count++
			return nil
		}()
		if err != nil {
			return err
		}
	}

	return finishImport(ctx, filename, count)
}
func getPrimaryAliasFromAliasSlice(aliases []models.Alias) string {
	for _, a := range aliases {
		if a.Primary {
			return a.Alias
		}
	}
	return ""
}
