package handlers

import (
	"net/http"
	"time"

	"github.com/ebrgr/TuneTrail/engine/middleware"
	"github.com/ebrgr/TuneTrail/internal/admin"
	"github.com/ebrgr/TuneTrail/internal/adminweb"
	"github.com/ebrgr/TuneTrail/internal/db"
	"github.com/ebrgr/TuneTrail/internal/logger"
	"github.com/ebrgr/TuneTrail/internal/utils"
)

func SubmitListenWithIDHandler(store db.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		l := logger.FromContext(ctx)
		l.Debug().Msg("SubmitListenWithIDHandler: Got request")

		u := middleware.GetUserFromContext(ctx)
		if u == nil {
			l.Debug().Msg("SubmitListenWithIDHandler: Unauthorized request (user context is nil)")
			utils.WriteError(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		body, err := utils.DecodeBody[struct {
			TrackID int32  `json:"track_id"`
			Unix    int64  `json:"unix"`
			Client  string `json:"client"`
		}](r)
		if err != nil || body.TrackID == 0 || body.Unix == 0 {
			l.Debug().Msg("SubmitListenWithIDHandler: Invalid or missing required fields in request body")
			utils.WriteError(w, "track_id and unix (timestamp) must be provided", http.StatusBadRequest)
			return
		}

		if time.Now().Unix() < body.Unix {
			l.Debug().Msg("SubmitListenWithIDHandler: Timestamp is in the future")
			utils.WriteError(w, "invalid timestamp", http.StatusBadRequest)
			return
		}

		_, err = admin.ManualListen(ctx, store, u, db.ProfileID(ctx), body.TrackID, time.Unix(body.Unix, 0), body.Client)
		if err != nil {
			l.Err(err).Msg("SubmitListenWithIDHandler: Failed to submit listen")
			utils.WriteError(w, adminweb.PublicError(err), adminweb.ErrorStatus(err))
			return
		}

		w.WriteHeader(http.StatusCreated)
	}
}
