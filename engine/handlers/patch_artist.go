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

func SetPrimaryArtistAliasHandler(store db.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		l := logger.FromContext(ctx)

		artistID, err := utils.ParseIDParam(r, "id")
		if err != nil {
			l.Debug().AnErr("error", err).Msg("SetPrimaryArtistAliasHandler: Invalid artist id")
			utils.WriteError(w, "invalid artist id", http.StatusBadRequest)
			return
		}

		body, err := utils.DecodeBody[struct {
			Alias string `json:"alias"`
		}](r)
		if err != nil || body.Alias == "" {
			l.Debug().Msg("SetPrimaryArtistAliasHandler: Invalid or missing alias in request body")
			utils.WriteError(w, "alias must be provided", http.StatusBadRequest)
			return
		}

		if err = admin.EditAlias(ctx, store, middleware.GetUserFromContext(ctx), "artist", artistID, "primary", body.Alias); err != nil {
			l.Error().Err(err).Msg("SetPrimaryArtistAliasHandler: Failed to set artist primary alias")
			utils.WriteError(w, adminweb.PublicError(err), adminweb.ErrorStatus(err))
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

// UpdateArtistHandler updates a given artist.
func UpdateArtistHandler(store db.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		l := logger.FromContext(ctx)

		artistID, err := utils.ParseIDParam(r, "id")
		if err != nil {
			l.Debug().AnErr("error", err).Msg("UpdateArtistHandler: Invalid artist id")
			utils.WriteError(w, "invalid artist id", http.StatusBadRequest)
			return
		}

		body, err := utils.DecodeBody[struct {
			MBID *string `json:"mbid"`
		}](r)
		if err != nil {
			l.Debug().Msg("UpdateArtistHandler: Invalid request body")
			utils.WriteError(w, "invalid request body", http.StatusBadRequest)
			return
		}

		if body.MBID == nil {
			l.Debug().Msg("UpdateArtistHandler: Request body contains no updatable fields")
			utils.WriteError(w, "no updatable fields provided", http.StatusBadRequest)
			return
		}

		raw := ""
		if body.MBID != nil {
			raw = *body.MBID
			if raw == "" {
				utils.WriteError(w, "provided musicbrainz id is invalid", 400)
				return
			}
		}
		if err = admin.UpdateMediaMetadata(ctx, store, middleware.GetUserFromContext(ctx), "artist", artistID, raw, nil); err != nil {
			l.Error().Err(err).Msg("UpdateArtistHandler: Failed to update metadata")
			utils.WriteError(w, adminweb.PublicError(err), adminweb.ErrorStatus(err))
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}
