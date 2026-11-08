package webui

import (
	"github.com/ebrgr/TuneTrail/internal/adminui"
	"net/http"
	"net/url"
	"regexp"
	"strings"
)

var slugPattern = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)
var reserved = map[string]bool{"admin": true, "apis": true, "image": true, "public": true, "assets": true, "web-assets": true, "preview-assets": true, "theme-helper": true, "artist": true, "album": true, "track": true, "chart": true, "listens": true, "rewind": true}

// IsPage defines the Go document dispatcher boundary.
func IsPage(path string) bool {
	if adminui.IsPage(path) {
		return true
	}
	if !strings.HasPrefix(path, "/") || strings.HasPrefix(path, "//") {
		return false
	}
	if path == "/" {
		return true
	}
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) == 0 || !slugPattern.MatchString(parts[0]) || reserved[parts[0]] {
		return false
	}
	if len(parts) == 1 {
		return true
	}
	suffix := strings.Join(parts[1:], "/")
	switch suffix {
	case "listens", "search", "rewind", "chart/top/artists", "chart/top/albums", "chart/top/tracks":
		return true
	}
	// Route known media shapes before validating the ID in the handler.
	return len(parts) == 3 && (parts[1] == "artist" || parts[1] == "album" || parts[1] == "track")
}

// RedirectLegacy resolves only the known old unscoped page routes.
func (h *Pages) RedirectLegacy(w http.ResponseWriter, r *http.Request) bool {
	destination := safeNext(r.URL.RequestURI())
	if destination == "" {
		return false
	}
	profiles, e := h.store.ListListeningProfiles(r.Context())
	if e != nil {
		http.Error(w, "Unable to resolve profile", 500)
		return true
	}
	target := "/?next=" + url.QueryEscape(destination)
	if len(profiles) == 1 {
		target = "/" + url.PathEscape(profiles[0].Slug) + destination
	}
	http.Redirect(w, r, target, http.StatusFound)
	return true
}
