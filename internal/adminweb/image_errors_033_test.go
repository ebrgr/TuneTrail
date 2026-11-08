package adminweb

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/ebrgr/TuneTrail/internal/imagepipeline"
)

// TestImageErrorContract033 keeps busy and unsupported images distinct for users.
func TestImageErrorContract033(t *testing.T) {
	for _, tc := range []struct {
		err     error
		status  int
		message string
	}{
		{imagepipeline.ErrBusy, http.StatusServiceUnavailable, "image processing is busy; try again"},
		{imagepipeline.ErrFormat, http.StatusBadRequest, "image must be JPEG, PNG or static WebP"},
		{imagepipeline.ErrAnimation, http.StatusBadRequest, "image must be JPEG, PNG or static WebP"},
		{imagepipeline.ErrInvalidImage, http.StatusBadRequest, "invalid request"},
		{imagepipeline.ErrTooLarge, http.StatusRequestEntityTooLarge, "form exceeds its limit"},
	} {
		err := fmt.Errorf("wrapped: %w", tc.err)
		if got := ErrorStatus(err); got != tc.status {
			t.Errorf("%v status %d", err, got)
		}
		if got := PublicError(err); got != tc.message {
			t.Errorf("%v message %q", err, got)
		}
	}
}
