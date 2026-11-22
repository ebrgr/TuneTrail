package adminweb

import (
	"net/url"
	"path"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var slugPattern = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

// SafeReturn permits known read pages only, never a URL supplied as an arbitrary redirect.
func SafeReturn(raw string) string {
	if raw == "" || !strings.HasPrefix(raw, "/") || strings.HasPrefix(raw, "//") || strings.ContainsAny(raw, "\\\r\n\x00") {
		return ""
	}
	u, err := url.Parse(raw)
	if err != nil || u.IsAbs() || u.Host != "" || u.User != nil || u.Fragment != "" || u.RawPath != "" || strings.Contains(u.Path, "%") || path.Clean(u.Path) != u.Path {
		return ""
	}
	q, err := url.ParseQuery(u.RawQuery)
	if err != nil {
		return ""
	}
	// ISSUE-125: the account page takes no query and is a valid login return target.
	if u.Path == "/account" {
		if len(q) != 0 {
			return ""
		}
		return u.String()
	}
	// Permit the exact authenticated configuration page as a login return target.
	if u.Path == "/admin/settings" {
		if len(q) != 0 {
			return ""
		}
		return u.String()
	}
	positive := func(v string, max int64) bool {
		n, e := strconv.ParseInt(v, 10, 32)
		return e == nil && n > 0 && n <= max
	}
	// Health returns may select exactly one canonical profile ID.
	if u.Path == "/admin/health" {
		if len(q) == 0 || len(q) == 1 && len(q["profile"]) == 1 && positive(q.Get("profile"), 2147483647) {
			return u.String()
		}
		return ""
	}
	adminPage := u.Path == "/admin"
	if !adminPage && u.Path != "/" {
		p := strings.Split(strings.TrimPrefix(u.Path, "/"), "/")
		if !slugPattern.MatchString(p[0]) {
			return ""
		}
		switch p[0] {
		case "admin", "account", "apis", "image", "assets", "public", "web-assets", "preview-assets", "theme-helper", "artist", "album", "track", "chart", "listens", "rewind":
			return ""
		}
		valid := len(p) == 1 || len(p) == 2 && (p[1] == "listens" || p[1] == "rewind" || p[1] == "search") || len(p) == 4 && p[1] == "chart" && p[2] == "top" && (p[3] == "artists" || p[3] == "albums" || p[3] == "tracks")
		if len(p) == 3 && (p[1] == "artist" || p[1] == "album" || p[1] == "track") {
			valid = positive(p[2], 2147483647)
		}
		if (len(p) == 4 || len(p) == 5 && MediaTaskAllowed(p[2], p[4])) && p[1] == "manage" && (p[2] == "artist" || p[2] == "album" || p[2] == "track") {
			valid = positive(p[3], 2147483647)
		}
		if !valid {
			return ""
		}
	}
	mediaParts := strings.Split(strings.TrimPrefix(u.Path, "/"), "/")
	mediaManage := (len(mediaParts) == 4 || len(mediaParts) == 5) && mediaParts[1] == "manage"
	trackManage := mediaManage && mediaParts[2] == "track"
	for k, values := range q {
		if len(values) != 1 {
			return ""
		}
		v := values[0]
		// Management returns accept the same context keys as the task document.
		if mediaManage {
			switch k {
			case "merge_query", "artist_query", "listen_at", "event_id", "history_return", "tz":
			default:
				return ""
			}
		}
		if adminPage {
			switch k {
			case "profile":
				if !positive(v, 2147483647) {
					return ""
				}
			case "section":
				switch v {
				case "account", "profiles", "keys", "relay", "import", "export":
				default:
					return ""
				}
			default:
				return ""
			}
			continue
		}
		switch k {
		case "merge_query":
			if (len(mediaParts) != 4 && len(mediaParts) != 5) || mediaParts[1] != "manage" || len(v) > 100 {
				return ""
			}
		case "listen_at":
			if !trackManage {
				return ""
			}
			if _, e := strconv.ParseInt(v, 10, 64); e != nil {
				return ""
			}
		case "event_id":
			// A selected event belongs to a track management URL and its timestamp.
			if !trackManage || !q.Has("listen_at") || v == "" || len(v) > 256 {
				return ""
			}
		case "history_return":
			if !trackManage || len(v) > 2048 {
				return ""
			}
			if SafeHistoryReturn(v, mediaParts[0]) == "" {
				return ""
			}
		case "artist_query":
			if !trackManage || len(v) > 100 {
				return ""
			}
		case "q":
			if len(mediaParts) != 2 || mediaParts[1] != "search" || len([]rune(strings.TrimSpace(v))) > 200 {
				return ""
			}
		case "period":
			switch v {
			case "day", "week", "month", "year", "all_time":
			default:
				return ""
			}
		case "group_by":
			// Keep the chart grouping on return from authentication.
			if len(mediaParts) != 4 || strings.Join(mediaParts[1:], "/") != "chart/top/tracks" ||
				(v != "recording" && v != "release") {
				return ""
			}
		case "year":
			if !positive(v, 9999) {
				return ""
			}
		case "week":
			if !positive(v, 53) {
				return ""
			}
		case "month":
			if !positive(v, 12) {
				return ""
			}
		case "page", "artist_id", "album_id", "track_id":
			if !positive(v, 2147483647) {
				return ""
			}
		case "limit":
			if !positive(v, 500) {
				return ""
			}
		case "show_time":
			if v != "0" && v != "1" {
				return ""
			}
		case "tz":
			if _, e := time.LoadLocation(v); e != nil {
				return ""
			}
		default:
			return ""
		}
	}
	return u.String()
}

// MediaTaskAllowed is shared by routing and return validation so their read surfaces agree.
func MediaTaskAllowed(kind, task string) bool {
	if kind != "artist" && kind != "album" && kind != "track" {
		return false
	}
	switch task {
	case "edit", "image", "merge", "delete":
		return true
	case "add-listen", "listens", "listen":
		return kind == "track"
	}
	return false
}

// SafeHistoryReturn keeps personal history navigation within the route's profile.
func SafeHistoryReturn(raw, slug string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return ""
	}
	parts := strings.Split(strings.TrimPrefix(u.Path, "/"), "/")
	valid := u.Path == "/"+slug || u.Path == "/"+slug+"/listens"
	if len(parts) == 3 && parts[0] == slug && (parts[1] == "artist" || parts[1] == "album" || parts[1] == "track") {
		valid = true
	}
	if !valid {
		return ""
	}
	return SafeReturn(raw)
}
