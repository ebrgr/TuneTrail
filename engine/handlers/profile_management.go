package handlers

import (
	"errors"
	"io"
	"net/http"

	"github.com/ebrgr/TuneTrail/engine/middleware"
	"github.com/ebrgr/TuneTrail/internal/admin"
	"github.com/ebrgr/TuneTrail/internal/adminweb"
	"github.com/ebrgr/TuneTrail/internal/db"
	"github.com/ebrgr/TuneTrail/internal/utils"
)

// CreateProfileHandler adapts the shared atomic profile creation service.
func CreateProfileHandler(store db.ProfileStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, 4096)
		body, err := utils.DecodeBody[struct {
			DisplayName string `json:"display_name"`
			Slug        string `json:"slug"`
			AvatarIcon  string `json:"avatar_icon"`
		}](r)
		if err != nil {
			utils.WriteError(w, "invalid profile or icon", 400)
			return
		}
		profile, err := admin.CreateProfile(r.Context(), store, middleware.GetUserFromContext(r.Context()), body.DisplayName, body.Slug, body.AvatarIcon)
		if err != nil {
			status := adminweb.ErrorStatus(err)
			if errors.Is(err, db.ErrConflict) {
				status = 400
			}
			utils.WriteError(w, adminweb.PublicError(err), status)
			return
		}
		utils.WriteJSON(w, 201, profile)
	}
}
func ProfileManagementHandler(store db.ProfileStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		count, err := store.ProfileListenCount(r.Context(), db.ProfileID(r.Context()))
		if err != nil {
			utils.WriteError(w, "failed to count listens", 500)
			return
		}
		utils.WriteJSON(w, 200, map[string]any{"profile": middleware.ProfileFromContext(r.Context()), "listen_count": count})
	}
}
func DeleteProfileHandler(store interface {
	db.ProfileStore
	db.ImageStore
}) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		p := middleware.ProfileFromContext(r.Context())
		body, err := utils.DecodeBody[struct {
			DisplayName string `json:"display_name"`
			ListenCount *int64 `json:"listen_count"`
		}](r)
		if err != nil || body.DisplayName != p.DisplayName || body.ListenCount == nil || *body.ListenCount < 0 {
			utils.WriteError(w, "confirm profile name and listen count", 400)
			return
		}
		if err = admin.DeleteProfile(r.Context(), store, middleware.GetUserFromContext(r.Context()), p.ID, p.Slug, body.DisplayName, *body.ListenCount); err != nil {
			utils.WriteError(w, adminweb.PublicError(err), adminweb.ErrorStatus(err))
			return
		}
		w.WriteHeader(204)
	}
}

// ProfileAvatarHandler shares validation and file cleanup policy with Go forms.
func ProfileAvatarHandler(store interface {
	db.ProfileStore
	db.ImageStore
}) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		p := middleware.ProfileFromContext(r.Context())
		// Bound avatar requests before multipart data is parsed.
		ctx, release, err := admin.AcquireImageAdmission(r.Context())
		if err != nil {
			utils.WriteError(w, adminweb.PublicError(err), adminweb.ErrorStatus(err))
			return
		}
		defer release()
		r = r.WithContext(ctx)
		var image io.ReadSeeker
		icon := ""
		if r.Method != "DELETE" {
			r.Body = http.MaxBytesReader(w, r.Body, 5<<20)
			if err := r.ParseMultipartForm(1 << 20); err != nil {
				// The multipart envelope shares the avatar's five MiB limit.
				var tooLarge *http.MaxBytesError
				if errors.As(err, &tooLarge) {
					utils.WriteError(w, "avatar must be a multipart upload up to 5 MiB", http.StatusRequestEntityTooLarge)
				} else {
					utils.WriteError(w, "avatar must be a multipart upload up to 5 MiB", http.StatusBadRequest)
				}
				return
			}
			defer r.MultipartForm.RemoveAll()
			icon = r.FormValue("icon")
			f, _, err := r.FormFile("image")
			if err == nil {
				defer f.Close()
				image = f
			} else if err != http.ErrMissingFile {
				utils.WriteError(w, "invalid image upload", 400)
				return
			}
		}
		updated, err := admin.SetAvatar(r.Context(), store, middleware.GetUserFromContext(r.Context()), p.ID, p.Slug, icon, image)
		if err != nil {
			utils.WriteError(w, adminweb.PublicError(err), adminweb.ErrorStatus(err))
			return
		}
		utils.WriteJSON(w, 200, updated)
	}
}
