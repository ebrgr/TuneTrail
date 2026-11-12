package adminui_test

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"testing"

	"github.com/ebrgr/TuneTrail/internal/adminui"
	"github.com/ebrgr/TuneTrail/internal/db"
	"github.com/ebrgr/TuneTrail/internal/db/sqlite"
	"github.com/ebrgr/TuneTrail/internal/models"
	"github.com/ebrgr/TuneTrail/internal/relay"
	"github.com/stretchr/testify/require"
)

type roundTrip func(*http.Request) (*http.Response, error)

func (f roundTrip) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func TestAdministrationJourney(t *testing.T) {
	ctx := context.Background()
	s, err := sqlite.NewInMemory()
	require.NoError(t, err)
	defer s.Close(ctx)
	_, err = s.SaveUser(ctx, db.SaveUserOpts{Username: "admin", Password: "original-password", Role: models.UserRoleAdmin})
	require.NoError(t, err)
	calls := 0
	h, err := adminui.New(s, adminui.Options{DisableRateLimit: true, RelayClient: relay.Client{Transport: roundTrip(func(r *http.Request) (*http.Response, error) {
		calls++
		require.Equal(t, "GET", r.Method)
		require.Equal(t, "/validate-token", r.URL.Path)
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"valid":true}`))}, nil
	})}})
	require.NoError(t, err)
	call := func(method, path string, v url.Values, cookie *http.Cookie) *httptest.ResponseRecorder {
		t.Helper()
		var body io.Reader
		if v != nil {
			body = strings.NewReader(v.Encode())
		}
		req := httptest.NewRequest(method, path, body)
		if v != nil {
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		}
		if cookie != nil {
			req.AddCookie(cookie)
		}
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		require.Contains(t, w.Header().Get("Cache-Control"), "no-store")
		return w
	}
	csrf := func(w *httptest.ResponseRecorder) string {
		t.Helper()
		match := regexp.MustCompile(`name="csrf_token" value="([^"]+)"`).FindStringSubmatch(w.Body.String())
		require.Len(t, match, 2)
		return match[1]
	}
	login := call("GET", "/admin?profile=123", nil, nil)
	require.Equal(t, 200, login.Code)
	require.Contains(t, login.Body.String(), `name="next" value="/admin?profile=123"`)
	require.NotEmpty(t, login.Result().Cookies())
	pre := login.Result().Cookies()[0]
	bad := call("POST", "/admin/login", url.Values{"username": {"admin"}, "password": {"original-password"}, "next": {"/admin?profile=123"}}, pre)
	require.Equal(t, 403, bad.Code)
	logged := call("POST", "/admin/login", url.Values{"csrf_token": {csrf(login)}, "username": {"admin"}, "password": {"original-password"}, "next": {"/admin?profile=123"}}, pre)
	require.Equal(t, 303, logged.Code)
	require.Equal(t, "/admin?profile=123", logged.Header().Get("Location"))
	var session *http.Cookie
	for _, c := range logged.Result().Cookies() {
		if c.MaxAge >= 0 && c.Value != "" {
			session = c
		}
	}
	require.NotNil(t, session)
	page := call("GET", "/admin", nil, session)
	require.Equal(t, 200, page.Code)
	token := csrf(page)
	makeProfile := func(name, slug string) int32 {
		w := call("POST", "/admin/profile/create", url.Values{"csrf_token": {token}, "display_name": {name}, "slug": {slug}}, session)
		require.Equal(t, 303, w.Code, w.Body.String())
		p, e := s.GetListeningProfile(ctx, slug)
		require.NoError(t, e)
		keys, e := s.GetApiKeysByProfileID(ctx, p.ID)
		require.NoError(t, e)
		require.Len(t, keys, 1)
		return p.ID
	}
	// Multipart creation commits the profile and its initial key before processing the optional photo.
	var upload bytes.Buffer
	mw := multipart.NewWriter(&upload)
	require.NoError(t, mw.WriteField("csrf_token", token))
	require.NoError(t, mw.WriteField("display_name", "Partial"))
	require.NoError(t, mw.WriteField("slug", "partial"))
	part, e := mw.CreateFormFile("image", "bad.png")
	require.NoError(t, e)
	_, e = part.Write([]byte("not an image"))
	require.NoError(t, e)
	require.NoError(t, mw.Close())
	req := httptest.NewRequest("POST", "/admin/profile/create", &upload)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	req.AddCookie(session)
	failure := httptest.NewRecorder()
	h.ServeHTTP(failure, req)
	require.Equal(t, 200, failure.Code)
	require.Contains(t, failure.Body.String(), "Profile created, but the photo could not be saved")
	partial, e := s.GetListeningProfile(ctx, "partial")
	require.NoError(t, e)
	partialKeys, e := s.GetApiKeysByProfileID(ctx, partial.ID)
	require.NoError(t, e)
	require.Len(t, partialKeys, 1)

	a, b := makeProfile("Alice", "alice"), makeProfile("Bob", "bob")
	ap := fmt.Sprintf("/admin/profile/%d/", a)
	bp := fmt.Sprintf("/admin/profile/%d/", b)
	w := call("POST", ap+"save", url.Values{"csrf_token": {token}, "profile_slug": {"alice"}, "slug": {"alice-new"}, "display_name": {"Alice renamed"}}, session)
	require.Equal(t, 303, w.Code, w.Body.String())
	_, err = s.GetListeningProfile(ctx, "alice")
	require.ErrorIs(t, err, db.ErrNotFound)
	p, err := s.GetListeningProfile(ctx, "alice-new")
	require.NoError(t, err)
	require.Equal(t, a, p.ID)
	w = call("POST", ap+"keys/create", url.Values{"csrf_token": {token}, "profile_slug": {"alice"}, "label": {"stale"}}, session)
	require.Equal(t, 409, w.Code)
	keys, err := s.GetApiKeysByProfileID(ctx, a)
	require.NoError(t, err)
	key := keys[0]
	keypath := fmt.Sprintf("keys/%d/revoke", key.ID)
	w = call("POST", bp+keypath, url.Values{"csrf_token": {token}, "profile_slug": {"bob"}, "confirm": {"1"}}, session)
	require.Equal(t, 404, w.Code)
	w = call("POST", ap+keypath, url.Values{"csrf_token": {token}, "profile_slug": {"alice-new"}}, session)
	require.Equal(t, 400, w.Code)
	w = call("POST", ap+keypath, url.Values{"csrf_token": {token}, "profile_slug": {"alice-new"}, "confirm": {"1"}}, session)
	require.Equal(t, 303, w.Code)
	_, err = s.GetProfileByAPIKey(ctx, key.Key)
	require.ErrorIs(t, err, db.ErrNotFound)
	relayValues := url.Values{"csrf_token": {token}, "profile_slug": {"bob"}, "url": {"http://localhost:1234"}, "token": {"fake-token"}}
	w = call("POST", bp+"relay/test", relayValues, session)
	require.Equal(t, 303, w.Code, w.Body.String())
	require.Equal(t, 1, calls)
	raw, secret, err := s.GetProfileRelay(ctx, b)
	require.NoError(t, err)
	require.Empty(t, raw)
	require.Empty(t, secret)
	w = call("POST", bp+"relay/save", relayValues, session)
	require.Equal(t, 303, w.Code)
	raw, secret, err = s.GetProfileRelay(ctx, b)
	require.NoError(t, err)
	require.Equal(t, "http://localhost:1234", raw)
	require.Equal(t, "fake-token", secret)
	relayPage := call("GET", fmt.Sprintf("/admin?profile=%d", b), nil, session)
	require.Equal(t, 200, relayPage.Code)
	require.Contains(t, relayPage.Body.String(), `type="text" name="token" value="fake-token"`)
	blankRelay := url.Values{"csrf_token": {token}, "profile_slug": {"bob"}, "url": {"http://localhost:1234"}, "token": {""}}
	require.Equal(t, 400, call("POST", bp+"relay/save", blankRelay, session).Code)
	_, unchangedRelay, err := s.GetProfileRelay(ctx, b)
	require.NoError(t, err)
	require.Equal(t, "fake-token", unchangedRelay)
	w = call("POST", bp+"relay/remove", url.Values{"csrf_token": {token}, "profile_slug": {"bob"}}, session)
	require.Equal(t, 303, w.Code)
	raw, secret, err = s.GetProfileRelay(ctx, b)
	require.NoError(t, err)
	require.Empty(t, raw)
	require.Empty(t, secret)
	w = call("POST", "/admin/logout", url.Values{"csrf_token": {token}}, session)
	require.Equal(t, 303, w.Code)
	w = call("POST", bp+"keys/create", url.Values{"csrf_token": {token}, "profile_slug": {"bob"}}, session)
	require.Equal(t, 401, w.Code)
}
