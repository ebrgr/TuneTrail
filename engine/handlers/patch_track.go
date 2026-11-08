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

// SetPrimaryTrackAliasHandler sets the primary alias for a given track.
func SetPrimaryTrackAliasHandler(store db.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		l := logger.FromContext(ctx)

		trackID, err := utils.ParseIDParam(r, "id")
		if err != nil {
			l.Debug().AnErr("error", err).Msg("SetPrimaryTrackAliasHandler: Invalid track id")
			utils.WriteError(w, "invalid track id", http.StatusBadRequest)
			return
		}

		body, err := utils.DecodeBody[struct {
			Alias string `json:"alias"`
		}](r)
		if err != nil || body.Alias == "" {
			l.Debug().Msg("SetPrimaryTrackAliasHandler: Invalid or missing alias in request body")
			utils.WriteError(w, "alias must be provided", http.StatusBadRequest)
			return
		}

		if err = admin.EditAlias(ctx, store, middleware.GetUserFromContext(ctx), "track", trackID, "primary", body.Alias); err != nil {
			l.Error().Err(err).Msg("SetPrimaryTrackAliasHandler: Failed to set track primary alias")
			utils.WriteError(w, adminweb.PublicError(err), adminweb.ErrorStatus(err))
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

// SetPrimaryTrackArtistHandler sets whether a given artist is the primary artist for a track.
func SetPrimaryTrackArtistHandler(store db.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		l := logger.FromContext(ctx)

		trackID, err := utils.ParseIDParam(r, "id")
		if err != nil {
			l.Debug().AnErr("error", err).Msg("SetPrimaryTrackArtistHandler: Invalid track id")
			utils.WriteError(w, "invalid track id", http.StatusBadRequest)
			return
		}

		artistID, err := utils.ParseIDParam(r, "artist_id")
		if err != nil {
			l.Debug().AnErr("error", err).Msg("SetPrimaryTrackArtistHandler: Invalid artist id")
			utils.WriteError(w, "invalid artist id", http.StatusBadRequest)
			return
		}

		body, err := utils.DecodeBody[struct {
			IsPrimary *bool `json:"is_primary"`
		}](r)
		if err != nil || body.IsPrimary == nil {
			l.Debug().Msg("SetPrimaryTrackArtistHandler: Invalid or missing is_primary in request body")
			utils.WriteError(w, "is_primary must be provided", http.StatusBadRequest)
			return
		}

		if err = admin.SetMediaArtistPrimary(ctx, store, middleware.GetUserFromContext(ctx), "track", trackID, artistID, *body.IsPrimary); err != nil {
			l.Error().Err(err).Msg("SetPrimaryTrackArtistHandler: Failed to set primary track artist")
			utils.WriteError(w, adminweb.PublicError(err), adminweb.ErrorStatus(err))
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

// UpdateTrackHandler updates a given track.
func UpdateTrackHandler(store db.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		l := logger.FromContext(ctx)

		trackID, err := utils.ParseIDParam(r, "id")
		if err != nil {
			l.Debug().AnErr("error", err).Msg("UpdateTrackHandler: Invalid track id")
			utils.WriteError(w, "invalid track id", http.StatusBadRequest)
			return
		}

		body, err := utils.DecodeBody[struct {
			MBID *string `json:"mbid"`
		}](r)
		if err != nil {
			l.Debug().Msg("UpdateTrackHandler: Invalid request body")
			utils.WriteError(w, "invalid request body", http.StatusBadRequest)
			return
		}

		if body.MBID == nil {
			l.Debug().Msg("UpdateTrackHandler: Request body contains no updatable fields")
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
		if err = admin.UpdateMediaMetadata(ctx, store, middleware.GetUserFromContext(ctx), "track", trackID, raw, nil); err != nil {
			l.Error().Err(err).Msg("UpdateTrackHandler: Failed to update metadata")
			utils.WriteError(w, adminweb.PublicError(err), adminweb.ErrorStatus(err))
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}
