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

// SetPrimaryAlbumAliasHandler sets the primary alias for a given album.
func SetPrimaryAlbumAliasHandler(store db.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		l := logger.FromContext(ctx)

		albumID, err := utils.ParseIDParam(r, "id")
		if err != nil {
			l.Debug().AnErr("error", err).Msg("SetPrimaryAlbumAliasHandler: Invalid album id")
			utils.WriteError(w, "invalid album id", http.StatusBadRequest)
			return
		}

		body, err := utils.DecodeBody[struct {
			Alias string `json:"alias"`
		}](r)
		if err != nil || body.Alias == "" {
			l.Debug().Msg("SetPrimaryAlbumAliasHandler: Invalid or missing alias in request body")
			utils.WriteError(w, "alias must be provided", http.StatusBadRequest)
			return
		}

		if err = admin.EditAlias(ctx, store, middleware.GetUserFromContext(ctx), "album", albumID, "primary", body.Alias); err != nil {
			l.Error().Err(err).Msg("SetPrimaryAlbumAliasHandler: Failed to set album primary alias")
			utils.WriteError(w, adminweb.PublicError(err), adminweb.ErrorStatus(err))
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

// SetPrimaryAlbumArtistHandler sets whether a given artist is the primary artist for an album.
func SetPrimaryAlbumArtistHandler(store db.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		l := logger.FromContext(ctx)

		albumID, err := utils.ParseIDParam(r, "id")
		if err != nil {
			l.Debug().AnErr("error", err).Msg("SetPrimaryAlbumArtistHandler: Invalid album id")
			utils.WriteError(w, "invalid album id", http.StatusBadRequest)
			return
		}

		artistID, err := utils.ParseIDParam(r, "artist_id")
		if err != nil {
			l.Debug().AnErr("error", err).Msg("SetPrimaryAlbumArtistHandler: Invalid artist id")
			utils.WriteError(w, "invalid artist id", http.StatusBadRequest)
			return
		}

		body, err := utils.DecodeBody[struct {
			IsPrimary *bool `json:"is_primary"`
		}](r)
		if err != nil || body.IsPrimary == nil {
			l.Debug().Msg("SetPrimaryAlbumArtistHandler: Invalid or missing is_primary in request body")
			utils.WriteError(w, "is_primary must be provided", http.StatusBadRequest)
			return
		}

		if err = admin.SetMediaArtistPrimary(ctx, store, middleware.GetUserFromContext(ctx), "album", albumID, artistID, *body.IsPrimary); err != nil {
			l.Error().Err(err).Msg("SetPrimaryAlbumArtistHandler: Failed to set primary album artist")
			utils.WriteError(w, adminweb.PublicError(err), adminweb.ErrorStatus(err))
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

// UpdateAlbumHandler updates a given album.
func UpdateAlbumHandler(store db.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		l := logger.FromContext(ctx)

		albumID, err := utils.ParseIDParam(r, "id")
		if err != nil {
			l.Debug().AnErr("error", err).Msg("UpdateAlbumHandler: Invalid album id")
			utils.WriteError(w, "invalid album id", http.StatusBadRequest)
			return
		}

		body, err := utils.DecodeBody[struct {
			MBID             *string `json:"mbid"`
			IsVariousArtists *bool   `json:"is_various_artists"`
		}](r)
		if err != nil {
			l.Debug().Msg("UpdateAlbumHandler: Invalid request body")
			utils.WriteError(w, "invalid request body", http.StatusBadRequest)
			return
		}

		if body.MBID == nil && body.IsVariousArtists == nil {
			l.Debug().Msg("UpdateAlbumHandler: Request body contains no updatable fields")
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
		if err = admin.UpdateMediaMetadata(ctx, store, middleware.GetUserFromContext(ctx), "album", albumID, raw, body.IsVariousArtists); err != nil {
			l.Error().Err(err).Msg("UpdateAlbumHandler: Failed to update metadata")
			utils.WriteError(w, adminweb.PublicError(err), adminweb.ErrorStatus(err))
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}
