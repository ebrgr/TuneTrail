package handlers

import (
	"net/http"

	"github.com/ebrgr/TuneTrail/internal/db"
	"github.com/ebrgr/TuneTrail/internal/logger"
	"github.com/ebrgr/TuneTrail/internal/utils"
)

func GetTrackHandler(store db.TrackStore) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		l := logger.FromContext(ctx)

		l.Debug().Msg("GetTrackHandler: Received request to retrieve track")

		id, err := utils.ParseIDParam(r, "id")
		if err != nil || id <= 0 {
			l.Debug().AnErr("error", err).Msg("GetTrackHandler: Invalid track id")
			utils.WriteError(w, "invalid track id", http.StatusBadRequest)
			return
		}

		l.Debug().Msgf("GetTrackHandler: Retrieving track with ID %d", id)

		track, err := store.GetTrack(ctx, db.GetTrackOpts{ID: id})
		if err != nil {
			l.Err(err).Msgf("GetTrackHandler: Failed to retrieve track with ID %d", id)
			writeMediaReadError(w, err, "track")
			return
		}

		l.Debug().Msgf("GetTrackHandler: Successfully retrieved track with ID %d", id)
		utils.WriteJSON(w, http.StatusOK, track)
	}
}

// GetTrackAliasesHandler reports missing or unrelated tracks as not found.
func GetTrackAliasesHandler(store db.TrackStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		l := logger.FromContext(ctx)

		trackID, err := utils.ParseIDParam(r, "id")
		if err != nil {
			l.Debug().AnErr("error", err).Msg("GetTrackAliasesHandler: Invalid track id")
			utils.WriteError(w, "invalid track id", http.StatusBadRequest)
			return
		}

		aliases, err := store.GetAllTrackAliases(ctx, trackID)
		if err != nil {
			l.Err(err).Msg("GetTrackAliasesHandler: Failed to get track aliases")
			writeMediaReadError(w, err, "track")
			return
		}

		utils.WriteJSON(w, http.StatusOK, aliases)
	}
}

// GetArtistsForTrackHandler reports missing or unrelated tracks as not found.
func GetArtistsForTrackHandler(store db.ArtistStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		l := logger.FromContext(ctx)

		trackID, err := utils.ParseIDParam(r, "id")
		if err != nil {
			l.Debug().AnErr("error", err).Msg("GetArtistsForTrackHandler: Invalid track id")
			utils.WriteError(w, "invalid track id", http.StatusBadRequest)
			return
		}

		artists, err := store.GetArtistsForTrack(ctx, trackID)
		if err != nil {
			l.Err(err).Msg("GetArtistsForTrackHandler: Failed to retrieve artists")
			writeMediaReadError(w, err, "track")
			return
		}

		utils.WriteJSON(w, http.StatusOK, artists)
	}
}

// GetTrackInterestHandler retrieves interest data for a given track.
func GetTrackInterestHandler(store db.ListenStore) http.HandlerFunc {
	return interestHandler(store, "track")
}
