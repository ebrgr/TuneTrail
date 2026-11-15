package adminweb

import "net/http"

// FragmentRequested reports whether a script asked for an HTML fragment (ISSUE-107,
// ISSUE-124); a navigation to a URL carrying partial=1 (typed, bookmarked or cross-site)
// does not qualify and gets the full page.
func FragmentRequested(r *http.Request) bool {
	return r.Header.Get("X-TuneTrail-Partial") == "1" &&
		r.Header.Get("Sec-Fetch-Mode") != "navigate" && r.Header.Get("Sec-Fetch-Dest") != "document"
}
