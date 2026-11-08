// Package adminweb provides HTTP contracts for administrative adapters without routing pages.
package adminweb

import (
	"errors"
	"github.com/ebrgr/TuneTrail/internal/admin"
	"github.com/ebrgr/TuneTrail/internal/db"
	"github.com/ebrgr/TuneTrail/internal/imagepipeline"
	"net/http"
)

var ErrCSRF = errors.New("invalid or expired form token")
var ErrMethod = errors.New("POST required")
var ErrTooLarge = errors.New("form exceeds its limit")

// ErrorStatus never derives public output from a database error's text.
func ErrorStatus(err error) int {
	switch {
	case errors.Is(err, admin.ErrUnprocessable):
		return 422
	case errors.Is(err, db.ErrInvalidInput), errors.Is(err, imagepipeline.ErrFormat), errors.Is(err, imagepipeline.ErrAnimation), errors.Is(err, imagepipeline.ErrInvalidImage), errors.Is(err, imagepipeline.ErrDimensions):
		return 400
	case errors.Is(err, admin.ErrUnauthorized):
		return 401
	case errors.Is(err, admin.ErrForbidden), errors.Is(err, ErrCSRF):
		return 403
	case errors.Is(err, db.ErrNotFound):
		return 404
	case errors.Is(err, ErrMethod):
		return 405
	case errors.Is(err, db.ErrConflict):
		return 409
	case errors.Is(err, ErrTooLarge), errors.Is(err, imagepipeline.ErrTooLarge):
		return 413
	case errors.Is(err, imagepipeline.ErrBusy):
		return 503
	default:
		return 500
	}
}

// PublicError is shared by JSON errors and future HTML status pages.
func PublicError(err error) string {
	if errors.Is(err, imagepipeline.ErrFormat) || errors.Is(err, imagepipeline.ErrAnimation) {
		return "image must be JPEG, PNG or static WebP"
	}
	switch ErrorStatus(err) {
	case 422:
		return "check form fields"
	case 400:
		return "invalid request"
	case 401:
		return "administrator session required"
	case 403:
		return "request forbidden"
	case 404:
		return "item not found"
	case 405:
		return "POST required"
	case 409:
		return "data changed; refresh and try again"
	case 413:
		return "form exceeds its limit"
	case 503:
		return "image processing is busy; try again"
	default:
		return "operation failed"
	}
}

// NoStore protects administrative responses, including errors and downloads.
func NoStore(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		next.ServeHTTP(w, r)
	})
}
