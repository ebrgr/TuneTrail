package handlers

import (
	"github.com/ebrgr/TuneTrail/internal/webquery"
	"net/http"

	"github.com/ebrgr/TuneTrail/internal/db"
	"github.com/ebrgr/TuneTrail/internal/logger"
	"github.com/ebrgr/TuneTrail/internal/utils"
)

func GetListensHandler(store db.ListenStore) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		l := logger.FromContext(ctx)

		l.Debug().Msg("GetListensHandler: Received request to retrieve listens")

		opts := OptsFromRequest(r)
		opts.Timeframe = webquery.ResolveTimeframe(opts.Timeframe)
		l.Debug().Msgf("GetListensHandler: Retrieving listens with options: %+v", opts)

		listens, err := store.GetListensPaginated(ctx, opts)
		if err != nil {
			l.Err(err).Msg("GetListensHandler: Failed to retrieve listens")
			utils.WriteError(w, "failed to get listens: "+err.Error(), http.StatusBadRequest)
			return
		}

		l.Debug().Msg("GetListensHandler: Successfully retrieved listens")
		writePage(w, opts.Timeframe, listens)
	}
}
