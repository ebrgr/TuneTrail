package webui

import (
	"regexp"
	"testing"
)

// TestNoScriptNavigationExposesProfileAndScopedSearchLinks checks server-rendered fallbacks independently of attribute indentation.
func TestNoScriptNavigationExposesProfileAndScopedSearchLinks(t *testing.T) {
	s := pageStore(t)
	h, err := NewHandler(s, Options{})
	if err != nil {
		t.Fatal(err)
	}
	body := pageRequest(h, "/alpha?period=all_time").Body.String()
	for _, want := range []string{`<noscript>`, `href="/"\s+aria-label="Choose listening profile"`, `href="/alpha/search"\s+aria-label="Search"`, `Appearance settings require JavaScript.`} {
		if !regexp.MustCompile(want).MatchString(body) {
			t.Fatalf("no-script navigation missing %q", want)
		}
	}
}
