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

type ReplaceImageResponse struct {
	Success bool   `json:"success"`
	Image   string `json:"image"`
	Message string `json:"message,omitempty"`
}

// ReplaceArtistImageHandler delegates artwork safety to the shared administrative service.
func ReplaceArtistImageHandler(store db.DB) http.HandlerFunc {
	return replaceArtworkHandler(store, "artist")
}

// ReplaceAlbumImageHandler delegates artwork safety to the shared administrative service.
func ReplaceAlbumImageHandler(store db.DB) http.HandlerFunc {
	return replaceArtworkHandler(store, "album")
}
func replaceArtworkHandler(store db.DB, kind string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := utils.ParseIDParam(r, "id")
		if err != nil || id <= 0 {
			utils.WriteError(w, "invalid id", 400)
			return
		}
		if r.URL.RawQuery != "" {
			utils.WriteError(w, "invalid image request", 400)
			return
		}
		// Acquire image capacity before parsing multipart or URL-encoded data.
		ctx, release, err := admin.AcquireImageAdmission(r.Context())
		if err != nil {
			utils.WriteError(w, adminweb.PublicError(err), adminweb.ErrorStatus(err))
			return
		}
		defer release()
		r = r.WithContext(ctx)
		r.Body = http.MaxBytesReader(w, r.Body, admin.MaxArtworkBytes+(1<<20))
		if err = r.ParseMultipartForm(1 << 20); err != nil && err != http.ErrNotMultipart {
			// Distinguish request limits from malformed multipart data.
			var tooLarge *http.MaxBytesError
			if errors.As(err, &tooLarge) {
				utils.WriteError(w, adminweb.PublicError(adminweb.ErrTooLarge), http.StatusRequestEntityTooLarge)
			} else {
				utils.WriteError(w, "invalid image upload", http.StatusBadRequest)
			}
			return
		}
		if r.MultipartForm != nil {
			defer r.MultipartForm.RemoveAll()
		}
		for key, values := range r.PostForm {
			if (key != "image_url" && key != "artist_id" && key != "album_id") || len(values) != 1 {
				utils.WriteError(w, "invalid image request", 400)
				return
			}
		}
		if r.MultipartForm != nil {
			for key, files := range r.MultipartForm.File {
				if key != "image" || len(files) != 1 {
					utils.WriteError(w, "invalid image upload", 400)
					return
				}
			}
		}
		raw := r.PostForm.Get("image_url")
		var file io.ReadCloser
		if r.MultipartForm != nil && len(r.MultipartForm.File["image"]) > 0 {
			if len(r.MultipartForm.File["image"]) != 1 {
				utils.WriteError(w, "invalid image upload", 400)
				return
			}
			file, _, err = r.FormFile("image")
			if err != nil {
				utils.WriteError(w, "invalid image upload", 400)
				return
			}
			defer file.Close()
		}
		var reader io.Reader
		if file != nil {
			reader = file
		}
		image, err := admin.ReplaceArtwork(r.Context(), store, middleware.GetUserFromContext(r.Context()), kind, id, raw, reader, admin.ArtworkClient{})
		if err != nil {
			utils.WriteError(w, adminweb.PublicError(err), adminweb.ErrorStatus(err))
			return
		}
		utils.WriteJSON(w, http.StatusOK, ReplaceImageResponse{Success: true, Image: image.String()})
	}
}
