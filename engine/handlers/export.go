package handlers

import (
	"net/http"

	"github.com/ebrgr/TuneTrail/engine/middleware"
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

func ExportHandler(store db.ExportStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Content-Disposition", `attachment; filename="koito_export.json"`)
		ctx := r.Context()
		l := logger.FromContext(ctx)
		l.Debug().Msg("ExportHandler: Received request for export file")
		u := middleware.ProfileFromContext(ctx)
		if u == nil {
			l.Debug().Msg("ExportHandler: Unauthorized access")
			utils.WriteError(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		w.Header().Set("Cache-Control", "no-store")
		stream := &exportResponse{ResponseWriter: w}
		err := export.ExportData(ctx, u, store, stream)
		if err != nil {
			l.Err(err).Msg("ExportHandler: Failed to create export file")
			if !stream.started {
				utils.WriteError(w, "failed to create export file", http.StatusInternalServerError)
			}
			return
		}
	}
}
