package webui

import (
	"html"
	"regexp"
	"strings"
	"testing"
)

// TestResponsiveNavigationDestinations follows the real links with their selected period and timezone.
func TestResponsiveNavigationDestinations(t *testing.T) {
	h, err := NewHandler(pageStore(t), Options{})
	if err != nil {
		t.Fatal(err)
	}
	body := pageRequest(h, "/alpha?period=month&tz=America%2FSao_Paulo").Body.String()
	for _, route := range []string{"/alpha?period=month&tz=America%2FSao_Paulo", "/alpha/chart/top/artists?period=month&tz=America%2FSao_Paulo", "/alpha/rewind", "/alpha/search?period=month&tz=America%2FSao_Paulo"} {
		if !strings.Contains(html.UnescapeString(body), `href="`+route+`"`) {
			t.Fatalf("missing %s", route)
		}
		if w := pageRequest(h, route); w.Code != 200 {
			t.Fatalf("%s: %d", route, w.Code)
		}
	}
	if !strings.Contains(body, `<details class="nav-more">`) || !strings.Contains(body, `popovertarget="theme-panel"`) {
		t.Fatal("missing native disclosure or theme control")
	}
	// ISSUE-154: the history link left the sidebar and Rewind appears once, outside the More menu.
	if strings.Count(body, `href="/alpha/rewind"`) != 1 {
		t.Fatal("rewind must appear exactly once in the navigation")
	}
	// Check both selected period selectors without depending on translated labels.
	re := regexp.MustCompile(`<a href="([^"]+)" aria-current="true">`)
	matches := re.FindAllStringSubmatch(body, -1)
	if len(matches) == 0 {
		t.Fatal("no selected period")
	}
	for _, m := range matches {
		if !strings.Contains(html.UnescapeString(m[1]), "period=month") {
			t.Fatal("period lost", m[1])
		}
	}
}

// TestNavigationLabelsBelongToThemePanel keeps the no-script default and the sidebar free of a checkbox.
func TestNavigationLabelsBelongToThemePanel(t *testing.T) {
	h, err := NewHandler(pageStore(t), Options{})
	if err != nil {
		t.Fatal(err)
	}
	body := pageRequest(h, "/alpha").Body.String()
	for _, want := range []string{`data-nav-labels-choice`, `name="nav-labels" value="labels"`, `name="nav-labels" value="icons" checked`, `class="nav-periods-menu"`} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing %s", want)
		}
	}
	if strings.Contains(body, `class="nav-collapse"`) || strings.Contains(body, `data-nav-labels="labels"`) {
		t.Fatal("unexpected sidebar checkbox or non-default server labels")
	}
}
