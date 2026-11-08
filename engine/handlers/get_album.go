package handlers

import (
	"net/http"

	"github.com/ebrgr/TuneTrail/internal/db"
	"github.com/ebrgr/TuneTrail/internal/logger"
	"github.com/ebrgr/TuneTrail/internal/utils"
)

func GetAlbumHandler(store db.AlbumStore) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		l := logger.FromContext(ctx)

		l.Debug().Msg("GetAlbumHandler: Received request to retrieve album")

		id, err := utils.ParseIDParam(r, "id")
		if err != nil || id <= 0 {
			l.Debug().AnErr("error", err).Msg("GetAlbumHandler: Invalid album id")
			utils.WriteError(w, "invalid album id", http.StatusBadRequest)
			return
		}

		l.Debug().Msgf("GetAlbumHandler: Retrieving album with ID %d", id)

		album, err := store.GetAlbum(ctx, db.GetAlbumOpts{ID: id})
		if err != nil {
			l.Err(err).Msgf("GetAlbumHandler: Failed to retrieve album with ID %d", id)
			writeMediaReadError(w, err, "album")
			return
		}

		l.Debug().Msgf("GetAlbumHandler: Successfully retrieved album with ID %d", id)
		utils.WriteJSON(w, http.StatusOK, album)
	}
}

// GetAlbumAliasesHandler reports missing or unrelated albums as not found.
func GetAlbumAliasesHandler(store db.AlbumStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		l := logger.FromContext(ctx)

		albumID, err := utils.ParseIDParam(r, "id")
		if err != nil {
			l.Debug().AnErr("error", err).Msg("GetAlbumAliasesHandler: Invalid album id")
			utils.WriteError(w, "invalid album id", http.StatusBadRequest)
			return
		}

		aliases, err := store.GetAllAlbumAliases(ctx, albumID)
		if err != nil {
			l.Err(err).Msg("GetAlbumAliasesHandler: Failed to get album aliases")
			writeMediaReadError(w, err, "album")
			return
		}

		utils.WriteJSON(w, http.StatusOK, aliases)
	}
}

// GetArtistsForAlbumHandler reports missing or unrelated albums as not found.
func GetArtistsForAlbumHandler(store db.ArtistStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		l := logger.FromContext(ctx)

		albumID, err := utils.ParseIDParam(r, "id")
		if err != nil {
			l.Debug().AnErr("error", err).Msg("GetArtistsForAlbumHandler: Invalid album id")
			utils.WriteError(w, "invalid album id", http.StatusBadRequest)
			return
		}

		artists, err := store.GetArtistsForAlbum(ctx, albumID)
		if err != nil {
			l.Err(err).Msg("GetArtistsForAlbumHandler: Failed to retrieve artists")
			writeMediaReadError(w, err, "album")
			return
		}

		utils.WriteJSON(w, http.StatusOK, artists)
	}
}

// GetAlbumInterestHandler retrieves interest data for a given album.
func GetAlbumInterestHandler(store db.ListenStore) http.HandlerFunc {
	return interestHandler(store, "album")
}
