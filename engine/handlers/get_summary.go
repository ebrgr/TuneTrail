package handlers

import (
	"net/http"

	"github.com/ebrgr/TuneTrail/internal/db"
	"github.com/ebrgr/TuneTrail/internal/logger"
	"github.com/ebrgr/TuneTrail/internal/summary"
	"github.com/ebrgr/TuneTrail/internal/utils"
)

type summaryHandlerStore interface {
	db.ArtistStore
	db.AlbumStore
	db.TrackStore
	db.ListenStore
}

func SummaryHandler(store summaryHandlerStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		l := logger.FromContext(ctx)
		l.Debug().Msg("SummaryHandler: Received request to retrieve summary")
		timeframe := TimeframeFromRequest(r)

		summary, err := summary.GenerateSummary(ctx, store, db.ProfileID(ctx), timeframe, "")
		if err != nil {
			l.Err(err).Int32("profileid", db.ProfileID(ctx)).Any("timeframe", timeframe).Msgf("SummaryHandler: Failed to generate summary")
			utils.WriteError(w, "failed to generate summary", http.StatusInternalServerError)
			return
		}

		utils.WriteJSON(w, http.StatusOK, summary)
	}
}
