package engine

import (
	"bytes"
	"mime/multipart"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/ebrgr/TuneTrail/internal/cfg"
	"github.com/stretchr/testify/require"
)

// TestUploadClassifiesArchiveFailure distinguishes bad input from an I/O failure after import.
func TestUploadClassifiesArchiveFailure(t *testing.T) {
	a := newProfileTestApp(t)
	require.NoError(t, a.store.Exec(`INSERT INTO listening_profiles(id,slug,display_name,created_at) VALUES(99,'archive-fail','Archive failure',123)`))
	dir := filepath.Join(cfg.ConfigDir(), "import_complete")
	require.NoError(t, os.MkdirAll(dir, 0700))
	block := filepath.Join(dir, "99")
	require.NoError(t, os.WriteFile(block, []byte("block"), 0600))
	t.Cleanup(func() { _ = os.Remove(block) })
	for _, tc := range []struct {
		name, contents string
		status         int
	}{
		{"invalid", `[ invalid`, 400},
		{"archive failure", `[{"ts":"2023-11-14T22:13:20Z","master_metadata_track_name":"Archive Track","master_metadata_album_artist_name":"Archive Artist","reason_end":"trackdone"}]`, 500},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var body bytes.Buffer
			mw := multipart.NewWriter(&body)
			require.NoError(t, mw.WriteField("format", "spotify"))
			file, err := mw.CreateFormFile("file", "fictional.json")
			require.NoError(t, err)
			_, err = file.Write([]byte(tc.contents))
			require.NoError(t, err)
			require.NoError(t, mw.Close())
			r := httptest.NewRequest("POST", "http://koito.test/apis/web/v1/profiles/archive-fail/import", &body)
			r.Header.Set("Content-Type", mw.FormDataContentType())
			r.Header.Set("Cookie", "tunetrail_session="+a.cookie)
			w := httptest.NewRecorder()
			a.router.ServeHTTP(w, r)
			require.Equal(t, tc.status, w.Code)
			require.NotContains(t, w.Body.String(), block)
		})
	}
}
