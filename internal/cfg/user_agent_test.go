package cfg

import (
	"strings"
	"testing"
)

// TestUserAgentIdentifiesTuneTrail guards ISSUE-111: external services must see
// TuneTrail with its own contact, never the upstream Koito contact address.
func TestUserAgentIdentifiesTuneTrail(t *testing.T) {
	c, err := loadConfig(func(string) string { return "" }, "0.9")
	if err != nil {
		t.Fatal(err)
	}
	if want := "TuneTrail/0.9 ( https://github.com/ebrgr/TuneTrail )"; c.userAgent != want {
		t.Fatalf("User-Agent %q, want %q", c.userAgent, want)
	}
	if strings.Contains(strings.ToLower(c.userAgent), "koito") {
		t.Fatalf("User-Agent still mentions Koito: %q", c.userAgent)
	}
}
