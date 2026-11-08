package handlers

import (
	"github.com/ebrgr/TuneTrail/internal/webquery"
	"net/http"

	"github.com/ebrgr/TuneTrail/internal/db"
	"github.com/ebrgr/TuneTrail/internal/logger"
	"github.com/ebrgr/TuneTrail/internal/utils"
)

func GetTopAlbumsHandler(store db.AlbumStore) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		l := logger.FromContext(ctx)

		l.Debug().Msg("GetTopAlbumsHandler: Received request to retrieve top albums")

		opts := OptsFromRequest(r)
		opts.Timeframe = webquery.ResolveTimeframe(opts.Timeframe)
		l.Debug().Msgf("GetTopAlbumsHandler: Retrieving top albums with options: %+v", opts)

		albums, err := store.GetTopAlbumsPaginated(ctx, opts)
		if err != nil {
			l.Err(err).Msg("GetTopAlbumsHandler: Failed to retrieve top albums")
			utils.WriteError(w, "failed to get albums", http.StatusBadRequest)
			return
		}

		l.Debug().Msg("GetTopAlbumsHandler: Successfully retrieved top albums")
		writePage(w, opts.Timeframe, albums)
	}
}
