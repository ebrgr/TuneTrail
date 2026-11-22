package adminui_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/ebrgr/TuneTrail/engine/middleware"
	"github.com/ebrgr/TuneTrail/internal/admin"
	"github.com/ebrgr/TuneTrail/internal/adminui"
	"github.com/ebrgr/TuneTrail/internal/adminweb"
	"github.com/ebrgr/TuneTrail/internal/db"
	"github.com/ebrgr/TuneTrail/internal/db/sqlite"
	"github.com/ebrgr/TuneTrail/internal/models"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
)

// TestAccountTokenWeb162 exercises the browser forms and credential-free responses.
func TestAccountTokenWeb162(t *testing.T) {
	ctx := context.Background()
	s, err := sqlite.NewInMemory()
	require.NoError(t, err)
	defer s.Close(ctx)
	root, err := s.SaveUser(ctx, db.SaveUserOpts{Username: "root", Password: "original-root-password", Role: models.UserRoleAdmin})
	require.NoError(t, err)
	rootSession, err := admin.Login(ctx, s, "root", "original-root-password", false)
	require.NoError(t, err)
	_, err = s.SaveUser(ctx, db.SaveUserOpts{Username: "ordinary", Password: "original-user-password"})
	require.NoError(t, err)
	userSession, err := admin.Login(ctx, s, "ordinary", "original-user-password", false)
	require.NoError(t, err)
	rootCookie := &http.Cookie{Name: adminweb.SessionCookie, Value: rootSession.ID.String()}
	userCookie := &http.Cookie{Name: adminweb.SessionCookie, Value: userSession.ID.String()}
	h, err := adminui.New(s, adminui.Options{DisableRateLimit: true, RequireSubmissionToken: true})
	require.NoError(t, err)
	var logs bytes.Buffer
	l := zerolog.New(&logs)
	handler := middleware.Logger(&l)(h)
	request := func(method, path string, form url.Values, cookie *http.Cookie) *httptest.ResponseRecorder {
		t.Helper()
		req := httptest.NewRequest(method, path, strings.NewReader(form.Encode()))
		if form != nil {
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		}
		if cookie != nil {
			req.AddCookie(cookie)
		}
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		require.Equal(t, "no-store", rec.Header().Get("Cache-Control"))
		return rec
	}
	field := func(rec *httptest.ResponseRecorder, name string) string {
		t.Helper()
		match := regexp.MustCompile(`name="` + name + `" value="([^"]+)"`).FindStringSubmatch(rec.Body.String())
		require.Len(t, match, 2)
		return match[1]
	}
	post := func(path, page string, form url.Values, cookie *http.Cookie) *httptest.ResponseRecorder {
		t.Helper()
		rec := request("GET", page, nil, cookie)
		form.Set("csrf_token", field(rec, "csrf_token"))
		form.Set("submission_token", field(rec, "submission_token"))
		return request("POST", path, form, cookie)
	}
	rec := post("/admin/profile/create", "/admin", url.Values{"display_name": {"Pending"}, "slug": {"pending"}}, rootCookie)
	require.Equal(t, 303, rec.Code)
	pending, err := s.GetListeningProfile(ctx, "pending")
	require.NoError(t, err)
	rootProfile, err := s.GetListeningProfile(ctx, "default")
	require.NoError(t, err)
	profilePage := fmt.Sprintf("/admin?profile=%d", pending.ID)
	route := fmt.Sprintf("/admin/profile/%d/", pending.ID)
	for _, op := range []string{route + "account-token", route + "account-role", route + "delete", "/admin/profile/create"} {
		require.Equal(t, 401, request("POST", op, nil, nil).Code)
		require.Equal(t, 403, request("POST", op, nil, userCookie).Code)
	}
	selfRoute := fmt.Sprintf("/admin/profile/%d/account-token", rootProfile.ID)
	require.Equal(t, 403, request("POST", selfRoute, nil, rootCookie).Code)
	require.ErrorIs(t, admin.AuthorizeAction(root, admin.IssueAccountToken, rootProfile), db.ErrOwnAccountToken)
	require.ErrorIs(t, admin.AuthorizeAction(root, admin.Action("accounts.unknown"), rootProfile), admin.ErrForbidden)
	require.NotContains(t, request("GET", "/admin", nil, userCookie).Body.String(), "Issue activation token")
	newPassword := "fictional-password-162"
	_, err = admin.Login(ctx, s, "pending", newPassword, false)
	require.ErrorIs(t, err, admin.ErrUnauthorized)
	issue := func() string {
		t.Helper()
		rec := post(route+"account-token", profilePage, url.Values{"profile_slug": {"pending"}}, rootCookie)
		require.Equal(t, 200, rec.Code, rec.Body.String())
		require.Empty(t, rec.Header().Get("Location"))
		require.Equal(t, "no-referrer", rec.Header().Get("Referrer-Policy"))
		match := regexp.MustCompile(`<code>([a-zA-Z0-9_-]{43})</code>`).FindStringSubmatch(rec.Body.String())
		require.Len(t, match, 2)
		return match[1]
	}
	first := issue()
	activation := issue()
	redeemPage := request("GET", "/admin/redeem", nil, nil)
	require.Equal(t, "same-origin", redeemPage.Header().Get("Referrer-Policy"))
	preauth := redeemPage.Result().Cookies()[0]
	query := "/admin/redeem?" + url.Values{"account_token": {activation}}.Encode()
	csrf := field(redeemPage, "csrf_token")
	form := url.Values{"username": {"pending"}, "password": {newPassword}, "csrf_token": {csrf}}
	for _, method := range []string{"GET", "HEAD", "POST"} {
		rejected := request(method, query, form, preauth)
		require.Equal(t, 400, rejected.Code)
		require.NotContains(t, rejected.Body.String(), activation)
	}
	head := request("HEAD", "/admin/redeem", nil, preauth)
	require.Equal(t, 200, head.Code)
	require.Empty(t, head.Body.String())
	require.Equal(t, "same-origin", head.Header().Get("Referrer-Policy"))
	// The form keeps the typed username (ISSUE-125); every other byte must match across failures.
	neutral := func(rec *httptest.ResponseRecorder, name string) string {
		return strings.ReplaceAll(rec.Body.String(), `value="`+name+`"`, `value=""`)
	}
	redeem := func(username, token, formCSRF string) *httptest.ResponseRecorder {
		t.Helper()
		rec := request("POST", "/admin/redeem", url.Values{"username": {username}, "account_token": {token}, "password": {newPassword}, "confirm_password": {newPassword}, "csrf_token": {formCSRF}}, preauth)
		require.Equal(t, "same-origin", rec.Header().Get("Referrer-Policy"))
		require.NotContains(t, rec.Body.String(), token)
		require.NotContains(t, rec.Body.String(), newPassword)
		return rec
	}
	expired := redeem("pending", activation, "bad-csrf")
	require.Equal(t, 403, expired.Code)
	// ISSUE-125: a stale form is an expired page, not a permission error, and keeps only the username.
	require.Contains(t, expired.Body.String(), "This page expired or was opened in another tab. Try again.")
	require.NotContains(t, expired.Body.String(), "does not have permission")
	require.Contains(t, expired.Body.String(), `name="username" value="pending"`)
	require.Contains(t, expired.Body.String(), `name="confirm_password"`)
	short := request("POST", "/admin/redeem", url.Values{"username": {"pending"}, "account_token": {activation}, "password": {"short"}, "confirm_password": {"short"}, "csrf_token": {csrf}}, preauth)
	require.Equal(t, 400, short.Code)
	require.Contains(t, short.Body.String(), "at most 72 bytes")
	require.Contains(t, short.Body.String(), `aria-invalid="true"`)
	require.NotContains(t, short.Body.String(), activation)
	mismatch := request("POST", "/admin/redeem", url.Values{"username": {"pending"}, "account_token": {activation}, "password": {newPassword}, "confirm_password": {newPassword + "x"}, "csrf_token": {csrf}}, preauth)
	require.Equal(t, 400, mismatch.Code)
	require.Contains(t, mismatch.Body.String(), "The password and confirmation do not match.")
	require.NotContains(t, mismatch.Body.String(), newPassword)
	long := request("POST", "/admin/redeem", url.Values{"username": {"pending"}, "account_token": {activation}, "password": {strings.Repeat("a", 73)}, "confirm_password": {strings.Repeat("a", 73)}, "csrf_token": {csrf}}, preauth)
	require.Equal(t, 400, long.Code)
	wrong := redeem("pending", first, csrf)
	require.Equal(t, 400, wrong.Code)
	require.Equal(t, neutral(wrong, "pending"), neutral(redeem("missing", activation, csrf), "missing"))
	ordinaryProfile, err := s.GetListeningProfile(ctx, fmt.Sprintf("user-%d", userSession.UserID))
	require.NoError(t, err)
	_, err = admin.IssueProfileAccountToken(ctx, s, rootSession.ID, ordinaryProfile.ID)
	require.NoError(t, err)
	require.Equal(t, neutral(wrong, "pending"), neutral(redeem("ordinary", activation, csrf), "ordinary"))
	rec = redeem("pending", activation, csrf)
	require.Equal(t, 303, rec.Code)
	// ISSUE-125: success lands on the sign-in form, which explains the next step.
	require.Equal(t, "/admin?redeemed=1", rec.Header().Get("Location"))
	require.Equal(t, neutral(wrong, "pending"), neutral(redeem("pending", activation, csrf), "pending"))
	one, err := admin.Login(ctx, s, "pending", newPassword, false)
	require.NoError(t, err)
	two, err := admin.Login(ctx, s, "pending", newPassword, true)
	require.NoError(t, err)
	beforePromotion := issue()
	rec = post(route+"account-role", profilePage, url.Values{"profile_slug": {"pending"}, "role": {"admin"}}, rootCookie)
	require.Equal(t, 303, rec.Code, rec.Body.String())
	account, err := s.ProfileAccount(ctx, pending.ID)
	require.NoError(t, err)
	require.Equal(t, models.UserRoleAdmin, account.Role)
	require.Equal(t, neutral(wrong, "pending"), neutral(redeem("pending", beforePromotion, csrf), "pending"))
	reset := issue()
	require.Equal(t, 303, redeem("pending", reset, csrf).Code)
	for _, session := range []*models.Session{one, two} {
		_, err = admin.Authenticate(ctx, s, session.ID)
		require.ErrorIs(t, err, admin.ErrUnauthorized)
	}
	_, err = admin.Authenticate(ctx, s, userSession.ID)
	require.NoError(t, err)
	unconfirmed := post(fmt.Sprintf("/admin/profile/%d/delete", rootProfile.ID), "/account/profile",
		url.Values{"profile_slug": {rootProfile.Slug}, "display_name": {rootProfile.DisplayName}, "listen_count": {"0"}}, rootCookie)
	require.Equal(t, 400, unconfirmed.Code)
	_, err = admin.Authenticate(ctx, s, rootSession.ID)
	require.NoError(t, err)
	rec = post(fmt.Sprintf("/admin/profile/%d/delete", rootProfile.ID), "/account/profile",
		url.Values{"profile_slug": {rootProfile.Slug}, "display_name": {rootProfile.DisplayName}, "listen_count": {"0"}, "confirm": {"1"}}, rootCookie)
	require.Equal(t, 303, rec.Code)
	require.Contains(t, rec.Header().Get("Set-Cookie"), "Max-Age=0")
	_, err = admin.Authenticate(ctx, s, rootSession.ID)
	require.ErrorIs(t, err, admin.ErrUnauthorized)
	adminSession, err := admin.Login(ctx, s, "pending", newPassword, false)
	require.NoError(t, err)
	lastCookie := &http.Cookie{Name: adminweb.SessionCookie, Value: adminSession.ID.String()}
	last := post(route+"account-role", "/account/profile", url.Values{"profile_slug": {"pending"}, "role": {"user"}}, lastCookie)
	require.Equal(t, 409, last.Code)
	listing := request("GET", "/account/profile", nil, lastCookie)
	for _, token := range []string{first, activation, beforePromotion, reset} {
		raw, err := base64.RawURLEncoding.DecodeString(token)
		require.NoError(t, err)
		digest := sha256.Sum256(raw)
		for _, secret := range []string{token, hex.EncodeToString(digest[:]), base64.StdEncoding.EncodeToString(digest[:]), newPassword} {
			require.NotContains(t, listing.Body.String(), secret)
			require.NotContains(t, logs.String(), secret)
		}
	}
	current, err := s.GetUserByUsername(ctx, "pending")
	require.NoError(t, err)
	require.NotContains(t, listing.Body.String(), string(current.Password))
	require.NotContains(t, logs.String(), string(current.Password))
	require.True(t, adminSession.ExpiresAt.After(time.Now()))
}
