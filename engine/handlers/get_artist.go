package handlers

import (
	"net/http"

	"github.com/ebrgr/TuneTrail/internal/db"
	"github.com/ebrgr/TuneTrail/internal/logger"
	"github.com/ebrgr/TuneTrail/internal/utils"
)

func GetArtistHandler(store db.ArtistStore) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		l := logger.FromContext(ctx)

		l.Debug().Msg("GetArtistHandler: Received request to retrieve artist")

		id, err := utils.ParseIDParam(r, "id")
		if err != nil || id <= 0 {
			l.Debug().AnErr("error", err).Msg("GetArtistHandler: Invalid artist id")
			utils.WriteError(w, "invalid artist id", http.StatusBadRequest)
			return
		}

		l.Debug().Msgf("GetArtistHandler: Retrieving artist with ID %d", id)

		artist, err := store.GetArtist(ctx, db.GetArtistOpts{ID: id})
		if err != nil {
			l.Err(err).Msgf("GetArtistHandler: Failed to retrieve artist with ID %d", id)
			writeMediaReadError(w, err, "artist")
			return
		}

		l.Debug().Msgf("GetArtistHandler: Successfully retrieved artist with ID %d", id)
		utils.WriteJSON(w, http.StatusOK, artist)
	}
}

// GetArtistAliasesHandler reports missing or unrelated artists as not found.
func GetArtistAliasesHandler(store db.ArtistStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		l := logger.FromContext(ctx)

		artistID, err := utils.ParseIDParam(r, "id")
		if err != nil {
			l.Debug().AnErr("error", err).Msg("GetArtistAliasesHandler: Invalid artist id")
			utils.WriteError(w, "invalid artist id", http.StatusBadRequest)
			return
		}

		aliases, err := store.GetAllArtistAliases(ctx, artistID)
		if err != nil {
			l.Err(err).Msg("GetArtistAliasesHandler: Failed to get artist aliases")
			writeMediaReadError(w, err, "artist")
			return
		}

		utils.WriteJSON(w, http.StatusOK, aliases)
	}
}

// GetArtistInterestHandler retrieves interest data for a given artist.
func GetArtistInterestHandler(store db.ListenStore) http.HandlerFunc {
	return interestHandler(store, "artist")
}
