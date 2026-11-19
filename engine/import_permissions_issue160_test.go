package engine

import (
	"bytes"
	"fmt"
	"github.com/ebrgr/TuneTrail/internal/cfg"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
)

// TestOwnerHTTPImportProvenance160 covers every human upload path and the later anonymous lazy image fetch.
func TestOwnerHTTPImportProvenance160(t *testing.T) {
	for _, mode := range []string{"json-file", "json-upload", "html-upload"} {
		t.Run(mode, func(t *testing.T) {
			a := newProfileTestApp(t)
			cookie, p := owner160(t, a)
			actor := &profileTestApp{store: a.store, router: a.router, cookie: cookie}
			var hits atomic.Int32
			target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { hits.Add(1); _, _ = w.Write([]byte("internal service")) }))
			defer target.Close()
			artistMBID, albumMBID, trackMBID := uuid.NewString(), uuid.NewString(), uuid.NewString()
			data := fmt.Sprintf(`{"version":"1","listens":[{"listened_at":"2023-11-14T22:13:20Z","track":{"mbid":%q,"aliases":[{"alias":"HTTP Track","is_primary":true}]},"album":{"mbid":%q,"aliases":[{"alias":"HTTP Album","is_primary":true}],"image_url":%q},"artists":[{"mbid":%q,"aliases":[{"alias":"HTTP Artist","is_primary":true}]}]}]}`, trackMBID, albumMBID, target.URL, artistMBID)
			path := "/apis/web/v1/profiles/" + p.Slug + "/import"
			switch mode {
			case "json-file":
				dir := filepath.Join(cfg.ConfigDir(), "import", fmt.Sprint(p.ID))
				require.NoError(t, os.MkdirAll(dir, 0700))
				file := filepath.Join(dir, "tunetrail-160.json")
				require.NoError(t, os.WriteFile(file, []byte(data), 0600))
				t.Cleanup(func() { os.Remove(file) })
				response := json160(a, "POST", path, `{"filename":"tunetrail-160.json"}`, cookie, "")
				require.Equal(t, 204, response.Code, response.Body.String())
			case "json-upload":
				response := actor.upload(t, "POST", path, map[string]string{"format": "koito"}, "file", "history.json", []byte(data), true)
				require.Equal(t, 204, response.Code, response.Body.String())
				// Unknown MBIDs must still reuse the owner's name-scoped identities on retry.
				duplicate := actor.upload(t, "POST", path, map[string]string{"format": "koito"}, "file", "history.json", []byte(data), true)
				require.Equal(t, 204, duplicate.Code, duplicate.Body.String())
				for _, table := range []string{"artists", "releases", "tracks", "listens"} {
					var count int
					require.NoError(t, a.store.QueryRow(`SELECT COUNT(*) FROM `+table).Scan(&count))
					require.Equal(t, 1, count, table)
				}
			case "html-upload":
				page := taskRequest139(a, "GET", fmt.Sprintf("/admin?profile=%d&section=import", p.ID), cookie, nil, nil)
				require.Equal(t, 200, page.Code)
				fields := map[string]string{"format": "koito", "confirm": "1", "profile_name": p.DisplayName, "profile_slug": p.Slug, "csrf_token": taskField139(t, page, "csrf_token"), "submission_token": taskField139(t, page, "submission_token")}
				// Browser multipart forms put CSRF first; keep the existing parser contract intact.
				var body bytes.Buffer
				writer := multipart.NewWriter(&body)
				require.NoError(t, writer.WriteField("csrf_token", fields["csrf_token"]))
				delete(fields, "csrf_token")
				for key, value := range fields {
					require.NoError(t, writer.WriteField(key, value))
				}
				part, err := writer.CreateFormFile("file", "history.json")
				require.NoError(t, err)
				_, err = part.Write([]byte(data))
				require.NoError(t, err)
				require.NoError(t, writer.Close())
				request := httptest.NewRequest("POST", fmt.Sprintf("http://koito.test/admin/profile/%d/import", p.ID), &body)
				request.Header.Set("Content-Type", writer.FormDataContentType())
				request.AddCookie(&http.Cookie{Name: "tunetrail_session", Value: cookie})
				response := httptest.NewRecorder()
				a.router.ServeHTTP(response, request)
				require.Equal(t, 303, response.Code, response.Body.String())
				followed := taskRequest139(a, "GET", response.Header().Get("Location"), cookie, nil, nil)
				require.Equal(t, 200, followed.Code, followed.Body.String())
				require.Contains(t, followed.Body.String(), "Imported: 1")
			}
			var source, imageID string
			require.NoError(t, a.store.QueryRow(`SELECT image_source,image FROM releases WHERE image IS NOT NULL LIMIT 1`).Scan(&source, &imageID))
			require.Equal(t, target.URL+"#tunetrail-public-only", source)
			for _, table := range []string{"artists", "releases", "tracks"} {
				var count int
				require.NoError(t, a.store.QueryRow(`SELECT COUNT(*) FROM `+table+` WHERE musicbrainz_id IS NOT NULL`).Scan(&count))
				require.Zero(t, count)
			}
			var provenance string
			require.NoError(t, a.store.QueryRow(`SELECT source FROM artist_aliases WHERE alias='HTTP Artist'`).Scan(&provenance))
			require.Equal(t, fmt.Sprintf("ProfileImport:%d", p.ID), provenance)
			response := json160(a, "GET", "/image/"+imageID+"/128x128.webp", "", "", "")
			require.Equal(t, 200, response.Code, response.Body.String())
			require.True(t, strings.HasPrefix(response.Header().Get("Content-Type"), "image/svg+xml"))
			require.Zero(t, hits.Load(), "lazy image download reached an internal service")
		})
	}
}
