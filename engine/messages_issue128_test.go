package engine

import (
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestSingleMessageComponent128 keeps every administration template on admin/message: no template
// but messages.html may hand-write a role=alert/status message, and the old classes are gone.
func TestSingleMessageComponent128(t *testing.T) {
	files, err := filepath.Glob("../assets/templates/admin/*.html")
	require.NoError(t, err)
	require.NotEmpty(t, files)
	handWritten := regexp.MustCompile(`<p[^>]*role="(alert|status)"`)
	for _, f := range files {
		raw, err := os.ReadFile(f)
		require.NoError(t, err)
		for _, line := range strings.Split(string(raw), "\n") {
			// Screen-reader-only live regions are announcers, not messages.
			if handWritten.MatchString(line) && !strings.Contains(line, "sr-only") && !strings.HasSuffix(f, "messages.html") {
				t.Errorf("%s: hand-written message: %s", filepath.Base(f), strings.TrimSpace(line))
			}
			if strings.Contains(string(raw), "account-message") || strings.Contains(string(raw), "account-error") {
				t.Errorf("%s: obsolete message class", filepath.Base(f))
				break
			}
		}
	}
}

// TestMessageComponentInCards128 renders an error and a notice through the account cards and checks
// the role, the id and the field link.
func TestMessageComponentInCards128(t *testing.T) {
	a := taskApp139(t)
	cookie := users128(t, a, "listener")
	bad := accountPost125(t, a, cookie, "password", url.Values{"password": {"first-password"}, "confirm_password": {"another-one"}}, nil)
	require.Equal(t, 400, bad.Code)
	body := bad.Body.String()
	require.Contains(t, body, `id="account-password-error" class="message message-error" role="alert" data-section-message`)
	require.Contains(t, body, "The password and confirmation do not match.")
	require.Contains(t, body, `aria-describedby="account-password-warning account-password-error"`)
	// The success notice is a status, not an alert, and sits in its own card.
	ok := accountPost125(t, a, cookie, "username", url.Values{"username": {"listener2"}}, nil)
	require.Equal(t, 303, ok.Code)
	page := taskRequest139(a, "GET", "/account", cookie, nil, nil)
	require.Contains(t, page.Body.String(), `class="message message-success" role="status" data-section-message`)
	require.NotContains(t, page.Body.String(), `class="message message-error"`)
}
