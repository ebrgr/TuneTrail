package handlers

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/ebrgr/TuneTrail/internal/admin"
	"github.com/ebrgr/TuneTrail/internal/adminweb"
	"github.com/ebrgr/TuneTrail/internal/db"
	"github.com/ebrgr/TuneTrail/internal/memkv"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// ImageAccess denies before the handler can inspect disk, redownload, or handle validators.
func ImageAccess(store db.DB) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Cache-Control", "private, no-store")
			w.Header().Add("Vary", "Cookie")
			images, ok := store.(db.ListeningImageStore)
			image, err := uuid.Parse(chi.URLParam(r, "image_id"))
			if !ok || err != nil {
				http.NotFound(w, r)
				return
			}
			ctx := r.Context()
			var profileID int32
			visibility := "legacy"
			if raw := chi.URLParam(r, "profile_id"); raw != "" {
				id, err := strconv.ParseInt(raw, 10, 32)
				if err != nil || id <= 0 {
					http.NotFound(w, r)
					return
				}
				profileID = int32(id)
				p, err := store.GetListeningProfileByID(ctx, profileID)
				if err != nil || !adminweb.CanReadListening(r, store, p) {
					http.NotFound(w, r)
					return
				}
				visibility = strconv.FormatBool(p.RequireLogin)
			}
			if image != uuid.Nil {
				// A playing-now event can expose its own artwork before the first scrobble.
				var playingTrackID int32
				if profileID > 0 {
					value, _ := memkv.Store.Get(fmt.Sprintf("now-playing:%d", profileID))
					playingTrackID, _ = value.(int32)
				}
				visible, err := images.ListeningImageExists(ctx, image, profileID, playingTrackID)
				if err == nil && !visible && profileID == 0 {
					visible, err = images.DirectoryAvatarExists(ctx, image)
				}
				// Root's existing editorial pages also use legacy URLs for unplayed catalog.
				if err == nil && !visible && profileID == 0 {
					actor, sessionErr := adminweb.SessionIdentity(r, store)
					if sessionErr == nil && admin.AuthorizeAction(actor.User, admin.ManageCatalog, nil) == nil {
						visible, err = store.ImageHasAssociation(ctx, image)
						visibility = "true"
					}
				}
				if err != nil || !visible {
					http.NotFound(w, r)
					return
				}
			}
			// Fresh public copies may outlive a visibility change. Origin requests still pass the gate.
			if visibility != "true" {
				w.Header().Set("Cache-Control", "public, max-age=2592000")
			}
			ctx = withImageVisibility(ctx, visibility)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
