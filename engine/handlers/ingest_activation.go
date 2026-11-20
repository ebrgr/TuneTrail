package handlers

import (
	"errors"
	"net/http"

	"github.com/ebrgr/TuneTrail/engine/middleware"
	"github.com/ebrgr/TuneTrail/internal/adminweb"
	"github.com/ebrgr/TuneTrail/internal/cfg"
	"github.com/ebrgr/TuneTrail/internal/db"
	"github.com/ebrgr/TuneTrail/internal/utils"
)

// IngestActivationHandler exposes an aggregate preview and a separate root confirmation.
func IngestActivationHandler(store db.IngestActivationStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		actor := middleware.GetUserFromContext(ctx)
		profile := middleware.ProfileFromContext(ctx)
		if actor == nil || profile == nil {
			utils.WriteError(w, "root session and profile required", http.StatusUnauthorized)
			return
		}
		w.Header().Set("Cache-Control", "no-store")
		if r.Method == http.MethodGet {
			preview, err := store.PreviewIngestActivation(ctx, actor.SessionID, profile.ID)
			if err != nil {
				utils.WriteError(w, adminweb.PublicError(err), adminweb.ErrorStatus(err))
				return
			}
			utils.WriteJSON(w, http.StatusOK, struct {
				db.IngestActivationPreview
				Master bool `json:"master_active"`
			}{IngestActivationPreview: preview, Master: cfg.ActiveIngestEnabled()})
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, 4096)
		body, err := adminweb.DecodeManagementJSON[struct {
			Active *bool  `json:"active"`
			Review *bool  `json:"review"`
			Digest string `json:"digest"`
		}](r)
		if err != nil || body.Active == nil || body.Review == nil {
			utils.WriteError(w, "active and review flags are required", http.StatusBadRequest)
			return
		}
		if err := store.SetIngestActivation(ctx, actor.SessionID, profile.ID, *body.Active, *body.Review, body.Digest); err != nil {
			// Explain why this particular flag cannot be enabled.
			if errors.Is(err, db.ErrIngestReviewUnavailable) {
				utils.WriteError(w, "review cannot be enabled until the ingestion review queue is available", http.StatusBadRequest)
				return
			}
			utils.WriteError(w, adminweb.PublicError(err), adminweb.ErrorStatus(err))
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}
