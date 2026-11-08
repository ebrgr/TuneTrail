package handlers

import (
	"github.com/ebrgr/TuneTrail/internal/db"
	"github.com/ebrgr/TuneTrail/internal/listening"
	"github.com/ebrgr/TuneTrail/internal/logger"
	"github.com/ebrgr/TuneTrail/internal/utils"
	"net/http"
)

type NowPlayingResponse = listening.NowPlaying

func NowPlayingHandler(store db.TrackStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		v, err := listening.ReadNowPlaying(r.Context(), store)
		if err != nil {
			logger.FromContext(r.Context()).Err(err).Msg("now-playing read failed")
			utils.WriteError(w, "failed to fetch currently playing track", 500)
			return
		}
		utils.WriteJSON(w, 200, v)
	}
}
