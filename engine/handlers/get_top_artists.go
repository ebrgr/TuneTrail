package handlers

import (
	"github.com/ebrgr/TuneTrail/internal/webquery"
	"net/http"

	"github.com/ebrgr/TuneTrail/internal/db"
	"github.com/ebrgr/TuneTrail/internal/logger"
	"github.com/ebrgr/TuneTrail/internal/utils"
)

func GetTopArtistsHandler(store db.ArtistStore) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		l := logger.FromContext(ctx)

		l.Debug().Msg("GetTopArtistsHandler: Received request to retrieve top artists")

		opts := OptsFromRequest(r)
		opts.Timeframe = webquery.ResolveTimeframe(opts.Timeframe)
		l.Debug().Msgf("GetTopArtistsHandler: Retrieving top artists with options: %+v", opts)

		artists, err := store.GetTopArtistsPaginated(ctx, opts)
		if err != nil {
			l.Err(err).Msg("GetTopArtistsHandler: Failed to retrieve top artists")
			utils.WriteError(w, "failed to get artists", http.StatusBadRequest)
			return
		}

		l.Debug().Msg("GetTopArtistsHandler: Successfully retrieved top artists")
		writePage(w, opts.Timeframe, artists)
	}
}
