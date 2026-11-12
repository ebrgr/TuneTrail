package adminweb

import (
	"net/http"
	"net/url"
	"strings"
)

// SameOrigin keeps the existing origin and Fetch Metadata policy for every web mutation.
func SameOrigin(next http.Handler) http.Handler { return SameOriginWithReject(next, nil) }

// SameOriginWithReject preserves the shared policy while allowing HTML-specific feedback.
func SameOriginWithReject(next http.Handler, reject http.HandlerFunc) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" && r.Method != "HEAD" && r.Method != "OPTIONS" {
			scheme := "http"
			if SecureRequest(r) {
				scheme = "https"
			}
			if origin := r.Header.Get("Origin"); origin != "" {
				parsed, err := url.Parse(origin)
				if err != nil || parsed.Scheme != scheme || !strings.EqualFold(parsed.Host, r.Host) {
					// Authentication middleware has not run yet, so protect the rejection here.
					w.Header().Set("Cache-Control", "no-store")
					if reject != nil {
						reject(w, r)
					} else {
						http.Error(w, `{"error":"cross-origin mutation forbidden"}`, 403)
					}
					return
				}
			}
			if r.Header.Get("Sec-Fetch-Site") == "cross-site" {
				w.Header().Set("Cache-Control", "no-store")
				if reject != nil {
					reject(w, r)
				} else {
					http.Error(w, `{"error":"cross-site mutation forbidden"}`, 403)
				}
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}
