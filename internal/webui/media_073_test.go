package webui

import (
	"bytes"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/ebrgr/TuneTrail/internal/models"
	"github.com/stretchr/testify/require"
	"regexp"
)

// TestMediaSectionLinks073 follows every section link without relying on scripts.
func TestMediaSectionLinks073(t *testing.T) {
	s := pageStore(t)
	h, err := NewHandler(s, Options{})
	require.NoError(t, err)
	for _, kind := range []string{"artist", "album", "track"} {
		response := pageRequest(h, "/alpha/"+kind+"/1?tz=UTC")
		require.Equal(t, 200, response.Code)
		body := response.Body.String()
		require.NotContains(t, body, "ZgotmplZ")
		require.Contains(t, body, "--media-accent:#253044")
		nav := regexp.MustCompile(`(?s)<nav class="media-tabs"[^>]*>(.*?)</nav>`).FindStringSubmatch(body)
		require.Len(t, nav, 2)
		matches := regexp.MustCompile(`href="([^"]+)"`).FindAllStringSubmatch(nav[1], -1)
		var links []string
		for _, match := range matches {
			links = append(links, match[1])
		}
		require.Len(t, links, 5, kind)
		for _, link := range links {
			parts := strings.SplitN(link, "#", 2)
			destination := body
			if parts[0] != "" {
				w := pageRequest(h, parts[0])
				require.Equal(t, 200, w.Code)
				destination = w.Body.String()
			}
			if len(parts) == 2 {
				require.Contains(t, destination, `id="`+parts[1]+`"`)
			}
		}
	}
}

// TestListenDisclosure073 protects admin-only actions, escaped titles and exact return links.
func TestListenDisclosure073(t *testing.T) {
	tmpl, err := loadTemplates("/web-assets")
	require.NoError(t, err)
	for _, admin := range []bool{false, true} {
		p := LivePage{Admin: admin, Page: Page{Profile: &Profile{Slug: "alpha"}}, Section: "listens", Timezone: "UTC", CurrentURL: "/alpha/listens?tz=UTC"}
		p.access.ManageListening = admin
		p.Listens = []ListenView{{Item: Item{ID: 1, URL: "/alpha/track/1", Title: "<script>long title</script>", Image: models.ImageList{Small: "/image/test/128x128.webp"}, Artists: []Link{{"Artist", "/alpha/artist/1"}}}, Unix: 123, Time: time.Unix(123, 0), Relative: "2 minutes ago"}}
		p.prepareActions()
		var out bytes.Buffer
		require.NoError(t, tmpl.ExecuteTemplate(&out, "partial/listens", p))
		body := out.String()
		require.Contains(t, body, `class="listen-title"`)
		require.Contains(t, body, "2 minutes ago")
		require.NotContains(t, body, "<script>")
		if admin {
			require.Contains(t, body, `<details class="listen-actions">`)
			require.Contains(t, body, `<summary aria-label="Manage this listening record">`)
			require.Contains(t, body, "listen_at=123&amp;tz=UTC&amp;history_return=%2Falpha%2Flistens%3Ftz%3DUTC")
		} else {
			require.NotContains(t, body, "listen-actions")
			require.NotContains(t, body, "/manage/")
		}
	}
	// Real public history still works when requested without a browser or JavaScript.
	s := pageStore(t)
	h, err := NewHandler(s, Options{})
	require.NoError(t, err)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("GET", "/alpha/listens?period=all_time", nil))
	require.Equal(t, 200, w.Code)
}
