package handlers

import (
	"net/http"
	"strconv"

	"github.com/ebrgr/TuneTrail/internal/db"
	"github.com/ebrgr/TuneTrail/internal/logger"
	"github.com/ebrgr/TuneTrail/internal/utils"
)

func interestHandler(store db.ListenStore, kind string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := utils.ParseIDParam(r, "id")
		if err != nil || id <= 0 {
			utils.WriteError(w, "invalid "+kind+" id", http.StatusBadRequest)
			return
		}
		count, err := strconv.Atoi(r.URL.Query().Get("buckets"))
		if err != nil || db.ValidateInterestBuckets(count) != nil {
			utils.WriteError(w, "parameter 'buckets' must be an integer between 1 and "+strconv.Itoa(db.MaxInterestBuckets), http.StatusBadRequest)
			return
		}
		opts := db.GetInterestOpts{Buckets: count}
		switch kind {
		case "artist":
			opts.ArtistID = id
		case "album":
			opts.AlbumID = id
		case "track":
			opts.TrackID = id
		}
		items, err := store.GetInterest(r.Context(), opts)
		if err != nil {
			logger.FromContext(r.Context()).Err(err).Msg("Failed to retrieve interest")
			utils.WriteError(w, "failed to retrieve interest", http.StatusInternalServerError)
			return
		}
		utils.WriteJSON(w, http.StatusOK, items)
	}
}
