package adminui

import (
	"regexp"
	"strings"
)

var routeSlug = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

// IsPage identifies administrative documents; authorization remains in Handler.
func IsPage(path string) bool {
	if !strings.HasPrefix(path, "/") || strings.HasPrefix(path, "//") {
		return false
	}
	if path == "/admin" || path == "/admin/settings" {
		return true
	}
	p := strings.Split(strings.Trim(path, "/"), "/")
	if len(p) == 4 && p[0] == "admin" && p[1] == "profile" && (p[3] == "export" || p[3] == "delete-review") {
		return true
	}
	if (len(p) == 4 || len(p) == 5 && p[4] == "review") && routeSlug.MatchString(p[0]) && p[1] == "manage" && (p[2] == "artist" || p[2] == "album" || p[2] == "track") {
		switch p[0] {
		case "apis", "image", "assets", "public", "web-assets", "preview-assets", "admin", "theme-helper", "artist", "album", "track", "chart", "listens", "rewind":
			return false
		}
		return true
	}
	return false
}

// IsAction identifies only known POST endpoints, before body parsing or staging.
func IsAction(path string) bool {
	if !strings.HasPrefix(path, "/") || strings.HasPrefix(path, "//") {
		return false
	}
	p := strings.Split(strings.Trim(path, "/"), "/")
	if path == "/admin/login" || path == "/admin/logout" || path == "/admin/account/username" || path == "/admin/account/password" || path == "/admin/profile/create" || path == "/admin/settings/save" || path == "/admin/settings/subsonic/test" {
		return true
	}
	if len(p) >= 4 && p[0] == "admin" && p[1] == "profile" {
		if len(p) == 4 {
			switch p[3] {
			case "save", "avatar", "avatar-clear", "delete", "import":
				return true
			}
		}
		if len(p) == 5 {
			switch p[3] + "/" + p[4] {
			case "keys/create", "relay/save", "relay/test", "relay/remove":
				return true
			}
		}
		return len(p) == 6 && p[3] == "keys" && (p[5] == "label" || p[5] == "revoke")
	}
	if len(p) == 5 && IsPage("/"+strings.Join(p[:4], "/")) {
		switch p[4] {
		case "alias-add", "alias-primary", "alias-delete", "metadata", "artist-add", "artist-remove", "primary-artist", "listen-add", "listen-delete", "artwork", "confirm":
			return true
		}
	}
	return false
}

// IsRoute is the shared document/action boundary, independent of authentication.
func IsRoute(method, path string) bool {
	if method == "GET" || method == "HEAD" {
		return IsPage(path)
	}
	return method == "POST" && IsAction(path)
}
