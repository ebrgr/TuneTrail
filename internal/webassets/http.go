package webassets

import (
	"net/http"
	"strings"
	"time"
)

// Handler serves a logical subtree from memory. Legacy aliases only accept fixed URLs.
func Handler(subtree string, versioned bool) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Cache-Control", "no-store")
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			w.Header().Set("Allow", "GET, HEAD")
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		s, err := Current()
		if err != nil {
			http.Error(w, "web assets unavailable", http.StatusServiceUnavailable)
			return
		}
		name := subtree + strings.TrimPrefix(r.URL.Path, "/")
		a, ok := s.Asset(name)
		if !ok || strings.ContainsAny(name, "\\%") {
			http.NotFound(w, r)
			return
		}
		policy := "no-cache"
		if r.URL.RawQuery != "" || r.URL.ForceQuery {
			if !versioned || r.URL.RawQuery != "v="+a.Digest {
				http.NotFound(w, r)
				return
			}
			policy = "public, max-age=31536000, immutable"
		}
		w.Header().Set("Cache-Control", policy)
		w.Header().Set("ETag", `"`+a.Digest+`"`)
		w.Header().Set("Content-Type", a.ContentType)
		http.ServeContent(assetResponse{w}, r, "", time.Time{}, a.Reader())
	})
}

// assetResponse prevents conditional/range errors from retaining success cache headers.
type assetResponse struct{ http.ResponseWriter }

// WriteHeader also covers 412 (which ServeContent writes directly) and 416 (which clears headers).
func (w assetResponse) WriteHeader(status int) {
	if status >= 400 {
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Del("ETag")
		w.Header().Del("Last-Modified")
	}
	w.ResponseWriter.WriteHeader(status)
}

// Unwrap preserves access to the original writer for standard response controllers.
func (w assetResponse) Unwrap() http.ResponseWriter { return w.ResponseWriter }
