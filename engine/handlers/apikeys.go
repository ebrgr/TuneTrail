package handlers

import (
	"errors"
	"net/http"

	"github.com/ebrgr/TuneTrail/engine/middleware"
	"github.com/ebrgr/TuneTrail/internal/admin"
	"github.com/ebrgr/TuneTrail/internal/adminweb"
	"github.com/ebrgr/TuneTrail/internal/db"
	"github.com/ebrgr/TuneTrail/internal/logger"
	"github.com/ebrgr/TuneTrail/internal/utils"
)

// GenerateApiKeyHandler uses the shared profile-scoped key service.
func GenerateApiKeyHandler(store interface {
	db.UserStore
	db.ProfileStore
}) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		l := logger.FromContext(ctx)

		user := middleware.GetUserFromContext(ctx)
		if user == nil {
			l.Debug().Msg("GenerateApiKeyHandler: Invalid user context")
			utils.WriteError(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		body, err := utils.DecodeBody[struct {
			Label string `json:"label"`
		}](r)
		if err != nil || body.Label == "" {
			l.Debug().Msg("GenerateApiKeyHandler: Invalid or missing label in request body")
			utils.WriteError(w, "label is required", http.StatusBadRequest)
			return
		}

		p := middleware.ProfileFromContext(ctx)
		key, err := admin.CreateKey(ctx, store, user, p.ID, p.Slug, body.Label)
		if err != nil {
			utils.WriteError(w, adminweb.PublicError(err), adminweb.ErrorStatus(err))
			return
		}

		l.Debug().Msgf("GenerateApiKeyHandler: Successfully generated API key ID %d", key.ID)
		utils.WriteJSON(w, http.StatusCreated, key)
	}
}

// DeleteApiKeyHandler keeps key revocation scoped to the resolved profile.
func DeleteApiKeyHandler(store interface {
	db.UserStore
	db.ProfileStore
}) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		l := logger.FromContext(ctx)

		user := middleware.GetUserFromContext(ctx)
		if user == nil {
			l.Debug().Msg("DeleteApiKeyHandler: Invalid user context")
			utils.WriteError(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		apiKeyID, err := utils.ParseIDParam(r, "id")
		if err != nil {
			l.Debug().AnErr("error", err).Msg("DeleteApiKeyHandler: Invalid API key ID")
			utils.WriteError(w, "invalid id", http.StatusBadRequest)
			return
		}

		if err := admin.RevokeKey(ctx, store, user, db.ProfileID(ctx), middleware.ProfileFromContext(ctx).Slug, apiKeyID); err != nil {
			if errors.Is(err, db.ErrNotFound) {
				utils.WriteError(w, "key not found in profile", 404)
				return
			}
			l.Error().Err(err).Msg("DeleteApiKeyHandler: Failed to delete API key")
			utils.WriteError(w, "failed to delete api key", http.StatusInternalServerError)
			return
		}

		l.Debug().Msgf("DeleteApiKeyHandler: Successfully deleted API key ID %d", apiKeyID)
		w.WriteHeader(http.StatusNoContent)
	}
}

// GetApiKeysHandler reads keys through the shared authorized service.
func GetApiKeysHandler(store interface {
	db.UserStore
	db.ProfileStore
}) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		l := logger.FromContext(ctx)

		user := middleware.GetUserFromContext(ctx)
		if user == nil {
			l.Debug().Msg("GetApiKeysHandler: Invalid user context")
			utils.WriteError(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		apiKeys, err := admin.Keys(ctx, store, user, db.ProfileID(ctx), middleware.ProfileFromContext(ctx).Slug)
		if err != nil {
			l.Error().Err(err).Msg("GetApiKeysHandler: Failed to retrieve API keys")
			utils.WriteError(w, "failed to retrieve api keys", http.StatusInternalServerError)
			return
		}

		l.Debug().Msgf("GetApiKeysHandler: Retrieved %d API keys", len(apiKeys))
		utils.WriteJSON(w, http.StatusOK, apiKeys)
	}
}

// UpdateApiKeyLabelHandler keeps key updates scoped to the resolved profile.
func UpdateApiKeyLabelHandler(store interface {
	db.UserStore
	db.ProfileStore
}) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		l := logger.FromContext(ctx)

		user := middleware.GetUserFromContext(ctx)
		if user == nil {
			l.Debug().Msg("UpdateApiKeyLabelHandler: Invalid user context")
			utils.WriteError(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		apiKeyID, err := utils.ParseIDParam(r, "id")
		if err != nil {
			l.Debug().AnErr("error", err).Msg("UpdateApiKeyLabelHandler: Invalid API key ID")
			utils.WriteError(w, "invalid id", http.StatusBadRequest)
			return
		}

		body, err := utils.DecodeBody[struct {
			Label string `json:"label"`
		}](r)
		if err != nil || body.Label == "" {
			l.Debug().Msg("UpdateApiKeyLabelHandler: Invalid or missing label in request body")
			utils.WriteError(w, "label is required", http.StatusBadRequest)
			return
		}

		if err := admin.RenameKey(ctx, store, user, db.ProfileID(ctx), middleware.ProfileFromContext(ctx).Slug, apiKeyID, body.Label); err != nil {
			if errors.Is(err, db.ErrNotFound) {
				utils.WriteError(w, "key not found in profile", 404)
				return
			}
			if errors.Is(err, db.ErrInvalidInput) {
				utils.WriteError(w, "invalid label", http.StatusBadRequest)
				return
			}
			l.Error().Err(err).Msg("UpdateApiKeyLabelHandler: Failed to update API key label")
			utils.WriteError(w, "failed to update api key label", http.StatusInternalServerError)
			return
		}

		l.Debug().Msgf("UpdateApiKeyLabelHandler: Successfully updated label for API key ID %d", apiKeyID)
		w.WriteHeader(http.StatusNoContent)
	}
}
