package handlers

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/ebrgr/TuneTrail/engine/middleware"
	"github.com/ebrgr/TuneTrail/internal/admin"
	"github.com/ebrgr/TuneTrail/internal/adminweb"
	"github.com/ebrgr/TuneTrail/internal/db"
	"github.com/ebrgr/TuneTrail/internal/logger"
	"github.com/ebrgr/TuneTrail/internal/utils"
)

// DeleteListenHandler checks personal ownership in the shared service.
func DeleteListenHandler(store interface {
	db.ListenStore
	db.ProfileStore
}) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		l := logger.FromContext(ctx)

		l.Debug().Msg("DeleteListenHandler: Received request to delete listen record")

		trackIDStr := r.URL.Query().Get("track_id")
		if trackIDStr == "" {
			l.Debug().Msg("DeleteListenHandler: Missing track ID in request")
			utils.WriteError(w, "track_id must be provided", http.StatusBadRequest)
			return
		}

		trackID, err := strconv.ParseInt(trackIDStr, 10, 32)
		if err != nil || trackID <= 0 {
			l.Debug().AnErr("error", err).Msg("DeleteListenHandler: Invalid track ID")
			utils.WriteError(w, "invalid id", http.StatusBadRequest)
			return
		}

		unixStr := r.URL.Query().Get("unix")
		if unixStr == "" {
			l.Debug().Msg("DeleteListenHandler: Missing timestamp in request")
			utils.WriteError(w, "unix timestamp must be provided", http.StatusBadRequest)
			return
		}

		unix, err := strconv.ParseInt(unixStr, 10, 64)
		if err != nil {
			l.Debug().AnErr("error", err).Msg("DeleteListenHandler: Invalid timestamp")
			utils.WriteError(w, "invalid unix timestamp", http.StatusBadRequest)
			return
		}

		l.Debug().Msgf("DeleteListenHandler: Deleting listen record for track ID %d at timestamp %d", trackID, unix)

		err = admin.RemovePersonalListen(ctx, store, middleware.GetUserFromContext(ctx), db.ProfileID(ctx), int32(trackID), time.Unix(unix, 0))
		if err != nil {
			if errors.Is(err, db.ErrNotFound) {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			l.Err(err).Msg("DeleteListenHandler: Failed to delete listen record")
			utils.WriteError(w, adminweb.PublicError(err), adminweb.ErrorStatus(err))
			return
		}

		l.Debug().Msgf("DeleteListenHandler: Successfully deleted listen record for track ID %d at timestamp %d", trackID, unix)
		w.WriteHeader(http.StatusNoContent)
	}
}

func PurgeAllDataHandler(store db.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		l := logger.FromContext(ctx)

		l.Debug().Msg("PurgeAllDataHandler: Received request to purge all data")

		// Global deletion must remain root-only even when called without route middleware.
		if err := admin.AuthorizeAction(middleware.GetUserFromContext(ctx), admin.ManageServer, nil); err != nil {
			utils.WriteError(w, adminweb.PublicError(err), adminweb.ErrorStatus(err))
			return
		}
		if err := admin.PurgeData(ctx, store, middleware.GetUserFromContext(ctx)); err != nil {
			l.Err(err).Msg("PurgeAllDataHandler: Failed to purge all data")
			utils.WriteError(w, "failed to purge data", http.StatusInternalServerError)
			return
		}

		l.Debug().Msg("PurgeAllDataHandler: Successfully purged all data")
		w.WriteHeader(http.StatusNoContent)
	}
}
