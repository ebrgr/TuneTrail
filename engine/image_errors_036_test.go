package engine

import (
	"bytes"
	"context"
	"image"
	"image/png"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ebrgr/TuneTrail/internal/admin"
	"github.com/ebrgr/TuneTrail/internal/imagepipeline"
	"github.com/stretchr/testify/require"
)

// TestArtworkHTTPFailures036 checks the real upload endpoint and format/size responses.
func TestArtworkHTTPFailures036(t *testing.T) {
	a := newProfileTestApp(t)
	require.NoError(t, a.store.Exec(`INSERT INTO artists(id) VALUES(10)`))
	require.NoError(t, a.store.Exec(`INSERT INTO artist_aliases VALUES(10,'Fake Artist','User',1)`))
	path := "/apis/web/v1/artist/10/image"
	w := a.upload(t, "PATCH", path, nil, "image", "unsupported.svg", []byte("<svg xmlns=\"http://www.w3.org/2000/svg\"></svg>"), true)
	require.Equal(t, 400, w.Code, w.Body.String())
	require.Contains(t, w.Body.String(), "JPEG, PNG or static WebP")
	w = a.upload(t, "PATCH", path, nil, "image", "large.png", bytes.Repeat([]byte{'x'}, imagepipeline.MaxBytes+1), true)
	require.Equal(t, 413, w.Code, w.Body.String())
	w = a.upload(t, "PATCH", path, nil, "image", "huge.png", bytes.Repeat([]byte{'x'}, 12<<20), true)
	require.Equal(t, 413, w.Code, w.Body.String())
	w = a.upload(t, "PUT", "/apis/web/v1/profiles/second/avatar", nil, "image", "huge.png", bytes.Repeat([]byte{'x'}, 5<<20), true)
	require.Equal(t, 413, w.Code, w.Body.String())
	var data bytes.Buffer
	require.NoError(t, png.Encode(&data, image.NewRGBA(image.Rect(0, 0, 8, 8))))
	w = a.upload(t, "PATCH", path, nil, "image", "valid.png", data.Bytes(), true)
	require.Equal(t, 200, w.Code, w.Body.String())
}

// TestArtworkRemoteLimit036 confirms oversized remote responses map to 413.
func TestArtworkRemoteLimit036(t *testing.T) {
	a := newProfileTestApp(t)
	ctx := context.Background()
	user, err := a.store.GetAdminUser(ctx)
	require.NoError(t, err)
	require.NoError(t, a.store.Exec(`INSERT INTO artists(id) VALUES(10)`))
	require.NoError(t, a.store.Exec(`INSERT INTO artist_aliases VALUES(10,'Fake Artist','User',1)`))
	transport := artworkTransport(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(io.LimitReader(strings.NewReader(strings.Repeat("x", imagepipeline.MaxBytes+1)), imagepipeline.MaxBytes+1))}, nil
	})
	_, err = admin.ReplaceArtwork(ctx, a.store, user, "artist", 10, "http://localhost/test", nil, admin.ArtworkClient{Transport: transport})
	require.ErrorIs(t, err, imagepipeline.ErrTooLarge)
}

type watchedBody036 struct{ read atomic.Bool }

// Read records whether multipart parsing began before admission.
func (r *watchedBody036) Read([]byte) (int, error) { r.read.Store(true); return 0, io.EOF }

// TestAvatarAdmissionBeforeBody036 cancels a queued HTTP request without reading its body.
func TestAvatarAdmissionBeforeBody036(t *testing.T) {
	a := newProfileTestApp(t)
	_, release, err := admin.AcquireImageAdmission(context.Background())
	require.NoError(t, err)
	defer release()
	body := &watchedBody036{}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	req := httptest.NewRequest("PUT", "http://koito.test/apis/web/v1/profiles/second/avatar", body).WithContext(ctx)
	req.Header.Set("Content-Type", "multipart/form-data; boundary=fictional")
	req.AddCookie(&http.Cookie{Name: "tunetrail_session", Value: a.cookie})
	done := make(chan struct{})
	go func() { a.router.ServeHTTP(httptest.NewRecorder(), req); close(done) }()
	select {
	case <-done:
		t.Fatal("request passed the held admission gate")
	case <-time.After(50 * time.Millisecond):
	}
	require.False(t, body.read.Load())
	cancel()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("canceled request remained blocked")
	}
	require.False(t, body.read.Load())
}
