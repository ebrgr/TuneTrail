package engine

import (
	"bytes"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"

	"github.com/ebrgr/TuneTrail/internal/adminui"
	"github.com/ebrgr/TuneTrail/internal/export"
	"github.com/ebrgr/TuneTrail/internal/mbz"
	"github.com/stretchr/testify/require"
)

func TestAdminUIImportExport(t *testing.T) {
	a := newProfileTestApp(t)
	h, err := adminui.New(a.store, adminui.Options{MusicBrainz: &mbz.MbzErrorCaller{}, TempDir: t.TempDir()})
	require.NoError(t, err)
	req := httptest.NewRequest("GET", "/admin?profile=2", nil)
	req.AddCookie(&http.Cookie{Name: "tunetrail_session", Value: a.cookie})
	page := httptest.NewRecorder()
	h.ServeHTTP(page, req)
	require.Equal(t, 200, page.Code)
	m := regexp.MustCompile(`name="csrf_token" value="([^"]+)"`).FindStringSubmatch(page.Body.String())
	require.Len(t, m, 2)
	token := m[1]
	upload := func(csrf string) *httptest.ResponseRecorder {
		var b bytes.Buffer
		w := multipart.NewWriter(&b)
		require.NoError(t, w.WriteField("csrf_token", csrf))
		require.NoError(t, w.WriteField("profile_slug", "second"))
		require.NoError(t, w.WriteField("profile_name", "Second"))
		require.NoError(t, w.WriteField("confirm", "1"))
		require.NoError(t, w.WriteField("format", "spotify"))
		f, e := w.CreateFormFile("file", "fictional.json")
		require.NoError(t, e)
		_, e = f.Write([]byte(`[{"ts":"2023-11-14T22:13:20Z","master_metadata_track_name":"Synthetic Track","master_metadata_album_artist_name":"Synthetic Artist","master_metadata_album_album_name":"Synthetic Album","reason_end":"trackdone"}]`))
		require.NoError(t, e)
		require.NoError(t, w.Close())
		r := httptest.NewRequest("POST", "/admin/profile/2/import", &b)
		r.Header.Set("Content-Type", w.FormDataContentType())
		r.AddCookie(&http.Cookie{Name: "tunetrail_session", Value: a.cookie})
		out := httptest.NewRecorder()
		h.ServeHTTP(out, r)
		return out
	}
	require.Equal(t, 403, upload("invalid").Code)
	out := upload(token)
	require.Equal(t, 303, out.Code, out.Body.String())
	require.Equal(t, "/admin?profile=2", out.Header().Get("Location"))
	req = httptest.NewRequest("GET", "/admin?profile=2", nil)
	req.AddCookie(&http.Cookie{Name: "tunetrail_session", Value: a.cookie})
	page = httptest.NewRecorder()
	h.ServeHTTP(page, req)
	require.Contains(t, page.Body.String(), "Imported: 1")
	req = httptest.NewRequest("GET", "/admin/profile/2/export", nil)
	req.AddCookie(&http.Cookie{Name: "tunetrail_session", Value: a.cookie})
	out = httptest.NewRecorder()
	h.ServeHTTP(out, req)
	require.Equal(t, 200, out.Code)
	require.Contains(t, out.Header().Get("Cache-Control"), "no-store")
	require.Contains(t, out.Header().Get("Content-Disposition"), "attachment")
	var doc export.KoitoExport
	require.NoError(t, json.Unmarshal(out.Body.Bytes(), &doc))
	require.Len(t, doc.Listens, 1)
	req = httptest.NewRequest("GET", "/admin/profile/2/export", nil)
	out = httptest.NewRecorder()
	h.ServeHTTP(out, req)
	require.Equal(t, 401, out.Code, fmt.Sprint(out.Body))
}
