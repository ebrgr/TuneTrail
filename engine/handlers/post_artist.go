package handlers

import (
	"net/http"

	"github.com/ebrgr/TuneTrail/engine/middleware"
	"github.com/ebrgr/TuneTrail/internal/admin"
	"github.com/ebrgr/TuneTrail/internal/adminweb"
	"github.com/ebrgr/TuneTrail/internal/db"
	"github.com/ebrgr/TuneTrail/internal/logger"
	"github.com/ebrgr/TuneTrail/internal/utils"
)

// CreateArtistAliasHandler creates a new alias for a given artist.
func CreateArtistAliasHandler(store db.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		l := logger.FromContext(ctx)

		artistID, err := utils.ParseIDParam(r, "id")
		if err != nil {
			l.Debug().AnErr("error", err).Msg("CreateArtistAliasHandler: Invalid artist id")
			utils.WriteError(w, "invalid artist id", http.StatusBadRequest)
			return
		}

		body, err := utils.DecodeBody[struct {
			Alias string `json:"alias"`
		}](r)
		if err != nil || body.Alias == "" {
			if err != nil {
				l.Debug().AnErr("error", err).Msg("Error decoding request body")
			}
			l.Debug().Msg("CreateArtistAliasHandler: Invalid or missing alias in request body")
			utils.WriteError(w, "alias must be provided", http.StatusBadRequest)
			return
		}

		if err = admin.EditAlias(ctx, store, middleware.GetUserFromContext(ctx), "artist", artistID, "add", body.Alias); err != nil {
			l.Error().Err(err).Msg("CreateArtistAliasHandler: Failed to save artist alias")
			utils.WriteError(w, adminweb.PublicError(err), adminweb.ErrorStatus(err))
			return
		}

		w.WriteHeader(http.StatusCreated)
	}
}
