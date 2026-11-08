package handlers

import (
	"errors"
	"github.com/ebrgr/TuneTrail/engine/middleware"
	"github.com/ebrgr/TuneTrail/internal/admin"
	"github.com/ebrgr/TuneTrail/internal/adminweb"
	"github.com/ebrgr/TuneTrail/internal/db"
	"github.com/ebrgr/TuneTrail/internal/utils"
	"net/http"
)

func ListProfilesHandler(store db.ProfileStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		profiles, err := store.ListListeningProfiles(r.Context())
		if err != nil {
			utils.WriteError(w, "failed to list profiles", 500)
			return
		}
		utils.WriteJSON(w, 200, profiles)
	}
}
func GetProfileHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		utils.WriteJSON(w, 200, middleware.ProfileFromContext(r.Context()))
	}
}

// UpdateProfileHandler preserves profile identity while using the shared update service.
func UpdateProfileHandler(store db.ProfileStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		p := middleware.ProfileFromContext(r.Context())
		body, err := utils.DecodeBody[struct {
			DisplayName *string `json:"display_name"`
			Slug        *string `json:"slug"`
		}](r)
		if err != nil || (body.DisplayName == nil && body.Slug == nil) {
			utils.WriteError(w, "display_name or slug required", 400)
			return
		}
		name, slug := p.DisplayName, p.Slug
		if body.DisplayName != nil {
			name = *body.DisplayName
		}
		if body.Slug != nil {
			slug = *body.Slug
		}
		updated, err := admin.UpdateProfile(r.Context(), store, middleware.GetUserFromContext(r.Context()), p.ID, p.Slug, name, slug)
		if err != nil {
			status := adminweb.ErrorStatus(err)
			if errors.Is(err, db.ErrConflict) {
				status = 400
			}
			utils.WriteError(w, adminweb.PublicError(err), status)
			return
		}
		utils.WriteJSON(w, 200, updated)
	}
}
