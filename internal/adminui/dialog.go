package adminui

import (
	"net/http"

	"github.com/ebrgr/TuneTrail/internal/adminweb"
)

// dialogRequested scopes progressive enhancement to a media task dialog.
func dialogRequested(r *http.Request) bool {
	return adminweb.FragmentRequested(r) && r.Header.Get("X-TuneTrail-Dialog") == "1"
}
