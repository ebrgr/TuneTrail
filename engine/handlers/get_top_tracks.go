package handlers

import (
	"github.com/ebrgr/TuneTrail/internal/webquery"
	"net/http"

	"github.com/ebrgr/TuneTrail/internal/db"
	"github.com/ebrgr/TuneTrail/internal/logger"
	"github.com/ebrgr/TuneTrail/internal/utils"
)

func GetTopTracksHandler(store db.TrackStore) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		l := logger.FromContext(ctx)

		l.Debug().Msg("GetTopTracksHandler: Received request to retrieve top tracks")

		opts := OptsFromRequest(r)
		opts.Timeframe = webquery.ResolveTimeframe(opts.Timeframe)
		l.Debug().Msgf("GetTopTracksHandler: Retrieving top tracks with options: %+v", opts)

		tracks, err := store.GetTopTracksPaginated(ctx, opts)
		if err != nil {
			l.Err(err).Msg("GetTopTracksHandler: Failed to retrieve top tracks")
			utils.WriteError(w, "failed to get tracks", http.StatusBadRequest)
			return
		}

		l.Debug().Msg("GetTopTracksHandler: Successfully retrieved top tracks")
		writePage(w, opts.Timeframe, tracks)
	}
}
