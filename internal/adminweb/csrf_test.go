package adminweb

import (
	"bytes"
	"errors"
	"github.com/ebrgr/TuneTrail/internal/admin"
	"github.com/ebrgr/TuneTrail/internal/db"
	"github.com/ebrgr/TuneTrail/internal/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"syscall"
	"testing"
	"time"
)

// testIdentity represents the actor an HTTP adapter resolved immediately before parsing.
func testIdentity() admin.Identity {
	return admin.Identity{User: &models.User{ID: 1, Role: models.UserRoleAdmin}, SessionID: uuid.New()}
}

// encodedForm sends body fields only; no credential can be supplied through the query.
func encodedForm(token string) *http.Request {
	r := httptest.NewRequest("POST", "http://koito.test/admin/action", strings.NewReader(url.Values{"csrf_token": {token}, "name": {"Alice"}}.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	return r
}

// TestSessionCSRFBindings guards identity, purpose, expiry, and server-restart boundaries.
func TestSessionCSRFBindings(t *testing.T) {
	c, e := NewCSRF()
	require.NoError(t, e)
	now := time.Date(2026, 11, 5, 12, 0, 0, 0, time.UTC)
	c.now = func() time.Time { return now }
	id := testIdentity()
	token, e := c.SessionToken(id)
	require.NoError(t, e)
	for i := 0; i < 2; i++ {
		f, e := c.ParseSessionForm(httptest.NewRecorder(), encodedForm(token), id, FormLimits{MaxBytes: 4096})
		require.NoError(t, e)
		require.Equal(t, "Alice", f.Values.Get("name"))
		f.Close()
	}
	_, e = c.ParseSessionForm(httptest.NewRecorder(), encodedForm(token), testIdentity(), FormLimits{MaxBytes: 4096})
	require.ErrorIs(t, e, ErrCSRF)
	_, e = c.ParseSessionForm(httptest.NewRecorder(), encodedForm(token), admin.Identity{}, FormLimits{MaxBytes: 4096})
	require.ErrorIs(t, e, admin.ErrUnauthorized)
	_, e = c.ParseLoginForm(httptest.NewRecorder(), encodedForm(token), FormLimits{MaxBytes: 4096})
	require.ErrorIs(t, e, ErrCSRF)
	other, e := NewCSRF()
	require.NoError(t, e)
	other.now = c.now
	_, e = other.ParseSessionForm(httptest.NewRecorder(), encodedForm(token), id, FormLimits{MaxBytes: 4096})
	require.ErrorIs(t, e, ErrCSRF)
	for _, bad := range []string{"", token + "x", strings.Repeat("x", 513)} {
		_, e = c.ParseSessionForm(httptest.NewRecorder(), encodedForm(bad), id, FormLimits{MaxBytes: 4096})
		require.ErrorIs(t, e, ErrCSRF)
	}
	now = now.Add(30 * time.Minute)
	_, e = c.ParseSessionForm(httptest.NewRecorder(), encodedForm(token), id, FormLimits{MaxBytes: 4096})
	require.ErrorIs(t, e, ErrCSRF)
}

// TestLoginCSRFNonce checks the preauth cookie cannot be replaced by a form field or prolonged by GET.
func TestLoginCSRFNonce(t *testing.T) {
	c, e := NewCSRF()
	require.NoError(t, e)
	now := time.Date(2026, 11, 5, 12, 0, 0, 0, time.UTC)
	c.now = func() time.Time { return now }
	get := httptest.NewRequest("GET", "https://koito.test/admin", nil)
	w := httptest.NewRecorder()
	token, e := c.LoginToken(w, get)
	require.NoError(t, e)
	cookies := w.Result().Cookies()
	require.Len(t, cookies, 1)
	cookie := cookies[0]
	require.True(t, cookie.HttpOnly)
	require.True(t, cookie.Secure)
	require.Equal(t, http.SameSiteStrictMode, cookie.SameSite)
	post := encodedForm(token)
	post.AddCookie(cookie)
	f, e := c.ParseLoginForm(httptest.NewRecorder(), post, FormLimits{MaxBytes: 4096})
	require.NoError(t, e)
	f.Close()
	get.AddCookie(cookie)
	now = now.Add(10 * time.Minute)
	w = httptest.NewRecorder()
	again, e := c.LoginToken(w, get)
	require.NoError(t, e)
	require.Equal(t, token, again)
	require.Empty(t, w.Result().Cookies())
	post = encodedForm(token)
	post.AddCookie(&http.Cookie{Name: preauthCookie, Value: cookie.Value + "x"})
	_, e = c.ParseLoginForm(httptest.NewRecorder(), post, FormLimits{MaxBytes: 4096})
	require.ErrorIs(t, e, ErrCSRF)
	now = now.Add(20 * time.Minute)
	_, e = c.ParseLoginForm(httptest.NewRecorder(), encodedForm(token), FormLimits{MaxBytes: 4096})
	require.ErrorIs(t, e, ErrCSRF)
}

// multipartBody allows adversarial part ordering and late duplicate CSRF fields.
func multipartBody(t *testing.T, token string, firstFile, duplicate bool) (*http.Request, string) {
	t.Helper()
	var b bytes.Buffer
	w := multipart.NewWriter(&b)
	file := func() {
		p, e := w.CreateFormFile("image", "fixture.jpg")
		require.NoError(t, e)
		_, e = p.Write([]byte("fake fixture image"))
		require.NoError(t, e)
	}
	if firstFile {
		file()
	}
	require.NoError(t, w.WriteField("csrf_token", token))
	if !firstFile {
		file()
	}
	require.NoError(t, w.WriteField("name", "Alice"))
	if duplicate {
		require.NoError(t, w.WriteField("csrf_token", token))
	}
	require.NoError(t, w.Close())
	r := httptest.NewRequest("POST", "http://koito.test/admin/upload", bytes.NewReader(b.Bytes()))
	r.Header.Set("Content-Type", w.FormDataContentType())
	return r, b.String()
}

// TestMultipartCSRFStaging rejects invalid envelopes without leaving files or executing a mutation.
func TestMultipartCSRFStaging(t *testing.T) {
	c, e := NewCSRF()
	require.NoError(t, e)
	id := testIdentity()
	token, e := c.SessionToken(id)
	require.NoError(t, e)
	for _, test := range []struct {
		name, token          string
		firstFile, duplicate bool
		max                  int64
		status               int
	}{{"valid", token, false, false, 4096, 200}, {"file first", token, true, false, 4096, 403}, {"wrong token", "wrong", false, false, 4096, 403}, {"long token", strings.Repeat("x", 513), false, false, 4096, 403}, {"late duplicate", token, false, true, 4096, 403}, {"size exceeded", token, false, false, 100, 413}} {
		t.Run(test.name, func(t *testing.T) {
			dir := t.TempDir()
			r, _ := multipartBody(t, test.token, test.firstFile, test.duplicate)
			f, e := c.ParseSessionForm(httptest.NewRecorder(), r, id, FormLimits{MaxBytes: test.max, TempDir: dir})
			if test.status == 200 {
				require.NoError(t, e)
				require.Equal(t, "Alice", f.Values.Get("name"))
				data, e := os.ReadFile(f.Files["image"].Path)
				require.NoError(t, e)
				require.Equal(t, "fake fixture image", string(data))
				f.Close()
			} else {
				require.Error(t, e)
				require.Nil(t, f)
				require.Equal(t, test.status, ErrorStatus(e))
			}
			left, e := os.ReadDir(dir)
			require.NoError(t, e)
			require.Empty(t, left)
		})
	}
	// A malformed suffix after a staged file is still rejected before the form is returned.
	dir := t.TempDir()
	r, body := multipartBody(t, token, false, false)
	r.Body = http.NoBody
	broken := strings.TrimSuffix(body, "--\r\n")
	r.Body = ioBody(broken)
	r.ContentLength = -1
	f, e := c.ParseSessionForm(httptest.NewRecorder(), r, id, FormLimits{MaxBytes: 4096, TempDir: dir})
	require.Error(t, e)
	require.Nil(t, f)
	left, e := os.ReadDir(dir)
	require.NoError(t, e)
	require.Empty(t, left)
}

// ioBody makes unknown-length request cases independent of Content-Length validation.
func ioBody(value string) io.ReadCloser { return io.NopCloser(strings.NewReader(value)) }

// TestFormSourcesLimitsAndMethods blocks duplicate fields and every unsupported token source.
func TestFormSourcesLimitsAndMethods(t *testing.T) {
	c, e := NewCSRF()
	require.NoError(t, e)
	id := testIdentity()
	token, e := c.SessionToken(id)
	require.NoError(t, e)
	for _, mutate := range []func(*http.Request){func(r *http.Request) { r.URL.RawQuery = "csrf_token=" + url.QueryEscape(token) }, func(r *http.Request) { r.Header.Set("X-CSRF-Token", token) }, func(r *http.Request) { r.Method = "GET" }, func(r *http.Request) { r.Body = ioBody("csrf_token=" + token + "&csrf_token=" + token) }} {
		r := encodedForm(token)
		mutate(r)
		f, e := c.ParseSessionForm(httptest.NewRecorder(), r, id, FormLimits{MaxBytes: 4096})
		require.Error(t, e)
		require.Nil(t, f)
	}
	for err, status := range map[error]int{db.ErrInvalidInput: 400, admin.ErrUnprocessable: 422, admin.ErrUnauthorized: 401, admin.ErrForbidden: 403, db.ErrNotFound: 404, db.ErrConflict: 409, errors.New("private SQL detail"): 500} {
		require.Equal(t, status, ErrorStatus(err))
		require.NotContains(t, PublicError(err), "SQL")
	}
}

// TestSafeReturnAllowsOnlyKnownReadPages rejects external, encoded, and action destinations.
func TestSafeReturnAllowsOnlyKnownReadPages(t *testing.T) {
	for _, v := range []string{"/", "/admin", "/admin?profile=1&section=keys", "/alpha", "/alpha/manage/track/1", "/manage/manage/artist/2", "/alpha/listens?page=2&period=all_time", "/alpha/rewind?year=2026&month=1", "/alpha/listens?year=2026&week=45&page=2&limit=25", "/alpha/manage/track/1?listen_at=1730611800&history_return=%2Falpha%2Flistens%3Fweek%3D45%26year%3D2026"} {
		require.Equal(t, v, SafeReturn(v), v)
	}
	for _, v := range []string{"https://evil.test/", "//evil.test/", "/admin/logout", "/admin?next=//evil.test", "/alpha/manage/track/1/delete", "/alpha/track/0", "/alpha/track/2147483648", "/alpha/%2e%2e/admin", "/alpha%2ftrack%2f1", "/alpha\\evil", "/admin?profile=1&profile=2", "/alpha?token=secret", "/alpha#fragment", "/admin?profile=-1", "/alpha/listens?week=54", "/alpha/manage/track/1?history_return=%2Fbeta%2Flistens", "/alpha/manage/track/1?history_return=%2Falpha%2Fmanage%2Ftrack%2F1"} {
		require.Empty(t, SafeReturn(v), v)
	}
}

// TestPreparedValuesAndDiskErrors preserves credential whitespace and infrastructure failures.
func TestPreparedValuesAndDiskErrors(t *testing.T) {
	f := &PreparedForm{Values: url.Values{"password": {" leading and trailing "}}}
	require.Equal(t, " leading and trailing ", f.SingleValue("password"))
	disk := &os.PathError{Op: "write", Path: "private-stage", Err: syscall.ENOSPC}
	require.Equal(t, 500, ErrorStatus(formReadError(disk)))
}
