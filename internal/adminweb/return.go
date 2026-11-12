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
	adminPage := u.Path == "/admin"
	if !adminPage && u.Path != "/" {
		p := strings.Split(strings.TrimPrefix(u.Path, "/"), "/")
		if !slugPattern.MatchString(p[0]) {
			return ""
		}
		switch p[0] {
		case "admin", "apis", "image", "assets", "public", "web-assets", "preview-assets", "theme-helper", "artist", "album", "track", "chart", "listens", "rewind":
			return ""
		}
		valid := len(p) == 1 || len(p) == 2 && (p[1] == "listens" || p[1] == "rewind" || p[1] == "search") || len(p) == 4 && p[1] == "chart" && p[2] == "top" && (p[3] == "artists" || p[3] == "albums" || p[3] == "tracks")
		if len(p) == 3 && (p[1] == "artist" || p[1] == "album" || p[1] == "track") {
			valid = positive(p[2], 2147483647)
		}
		if len(p) == 4 && p[1] == "manage" && (p[2] == "artist" || p[2] == "album" || p[2] == "track") {
			valid = positive(p[3], 2147483647)
		}
		if !valid {
			return ""
		}
	}
	mediaParts := strings.Split(strings.TrimPrefix(u.Path, "/"), "/")
	trackManage := len(mediaParts) == 4 && mediaParts[1] == "manage" && mediaParts[2] == "track"
	for k, values := range q {
		if len(values) != 1 {
			return ""
		}
		v := values[0]
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
			if len(mediaParts) != 4 || mediaParts[1] != "manage" || len(v) > 100 {
				return ""
			}
		case "listen_at":
			if !trackManage {
				return ""
			}
			if _, e := strconv.ParseInt(v, 10, 64); e != nil {
				return ""
			}
		case "history_return":
			if !trackManage || len(v) > 2048 {
				return ""
			}
			back, e := url.Parse(v)
			if e != nil || (back.Path != "/"+mediaParts[0] && back.Path != "/"+mediaParts[0]+"/listens") {
				return ""
			}
			if SafeReturn(v) == "" {
				return ""
			}
		case "artist_query":
			if !trackManage || len(v) > 100 {
				return ""
			}
		case "period":
			switch v {
			case "day", "week", "month", "year", "all_time":
			default:
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
