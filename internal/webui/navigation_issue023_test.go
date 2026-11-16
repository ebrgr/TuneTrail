package webui

import (
	"regexp"
	"testing"
)

// TestNoScriptNavigationExposesProfileAndScopedSearchLinks checks real fallback links with the selected period (ISSUE-071).
func TestNoScriptNavigationExposesProfileAndScopedSearchLinks(t *testing.T) {
	s := pageStore(t)
	h, err := NewHandler(s, Options{})
	if err != nil {
		t.Fatal(err)
	}
	body := pageRequest(h, "/alpha?period=all_time").Body.String()
	for _, want := range []string{`<noscript>`, `href="/"\s+aria-label="Choose listening profile"`, `href="/alpha/search\?period=all_time(?:&amp;tz=[^"]+)?"\s+aria-label="Search"`, `Changing the theme requires JavaScript.`} {
		if !regexp.MustCompile(want).MatchString(body) {
			t.Fatalf("no-script navigation missing %q", want)
		}
	}
}
