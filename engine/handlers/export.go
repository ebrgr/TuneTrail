package handlers

import (
	"net/http"

	"github.com/ebrgr/TuneTrail/engine/middleware"
	"github.com/ebrgr/TuneTrail/internal/admin"
	"github.com/ebrgr/TuneTrail/internal/adminweb"
	"github.com/ebrgr/TuneTrail/internal/db"
	"github.com/ebrgr/TuneTrail/internal/export"
	"github.com/ebrgr/TuneTrail/internal/logger"
	"github.com/ebrgr/TuneTrail/internal/utils"
)

// exportResponse tracks whether the streaming response already started.
type exportResponse struct {
	http.ResponseWriter
	started bool
}

func (w *exportResponse) Write(data []byte) (int, error) {
	w.started = true
	return w.ResponseWriter.Write(data)
}

// ExportHandler requires a human capability even for public profiles.
func ExportHandler(store interface {
	db.ExportStore
	db.ProfileStore
}) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Content-Disposition", `attachment; filename="tunetrail_export.json"`)
		ctx := r.Context()
		l := logger.FromContext(ctx)
		l.Debug().Msg("ExportHandler: Received request for export file")
		u := middleware.ProfileFromContext(ctx)
		if u == nil {
			l.Debug().Msg("ExportHandler: Unauthorized access")
			utils.WriteError(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		if err := admin.AuthorizeAction(middleware.GetUserFromContext(ctx), admin.ManageProfile, u); err != nil {
			utils.WriteError(w, adminweb.PublicError(err), adminweb.ErrorStatus(err))
			return
		}
		w.Header().Set("Cache-Control", "no-store")
		version := r.URL.Query().Get("version")
		if version == "" {
			version = export.ArchiveVersion
		}
		if version != "1" && version != export.ArchiveVersion {
			utils.WriteError(w, "unsupported export version", http.StatusBadRequest)
			return
		}
		stream := &exportResponse{ResponseWriter: w}
		err := admin.ExportHistory(ctx, store, middleware.GetUserFromContext(ctx), u.ID, stream, version)
		if err != nil {
			l.Err(err).Msg("ExportHandler: Failed to create export file")
			if !stream.started {
				utils.WriteError(w, "failed to create export file", http.StatusInternalServerError)
			}
			return
		}
	}
}
