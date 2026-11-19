package engine

import (
	"context"
	"fmt"
	"github.com/ebrgr/TuneTrail/internal/admin"
	"github.com/ebrgr/TuneTrail/internal/adminweb"
	"github.com/ebrgr/TuneTrail/internal/db"
	"github.com/ebrgr/TuneTrail/internal/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// owner160 returns a real ordinary session and its canonical profile.
func owner160(t *testing.T, a *profileTestApp) (string, *models.ListeningProfile) {
	t.Helper()
	ctx := context.Background()
	u, err := a.store.SaveUser(ctx, db.SaveUserOpts{Username: "owner160", Password: "test-password"})
	require.NoError(t, err)
	session, err := admin.Login(ctx, a.store, u.Username, "test-password", false)
	require.NoError(t, err)
	profiles, err := a.store.ListListeningProfiles(ctx)
	require.NoError(t, err)
	for _, p := range profiles {
		if p.UserID == u.ID {
			return session.ID.String(), &p
		}
	}
	t.Fatal("owner profile missing")
	return "", nil
}

// json160 deliberately sends only the requested credential, never a hidden admin cookie.
func json160(a *profileTestApp, method, path, body, cookie, key string) *httptest.ResponseRecorder {
	r := httptest.NewRequest(method, "http://koito.test"+path, strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	if cookie != "" {
		r.AddCookie(&http.Cookie{Name: "tunetrail_session", Value: cookie})
	}
	if key != "" {
		r.Header.Set("Authorization", "Token "+key)
	}
	w := httptest.NewRecorder()
	a.router.ServeHTTP(w, r)
	return w
}

// TestJSONManagementMatrix160 covers every existing personal route, plus global
// catalog/server boundaries, API-key separation and unknown-route rejection.
func TestJSONManagementMatrix160(t *testing.T) {
	a := taskApp139(t)
	cookie, p := owner160(t, a)
	own := "/apis/web/v1/profiles/" + p.Slug
	tests := []struct {
		method, path, body string
		rootOnly           bool
	}{
		{"PATCH", own + "/", `{"display_name":"Owner"}`, false},
		{"GET", own + "/management", "", false},
		{"DELETE", own + "/", `{}`, true},
		{"PUT", own + "/avatar", `{}`, false},
		{"DELETE", own + "/avatar", "", false},
		{"GET", own + "/export", "", false},
		{"POST", own + "/import", `{}`, false},
		{"GET", own + "/relay", "", false},
		{"POST", own + "/relay/test", `{}`, false},
		{"PUT", own + "/relay", `{}`, false},
		{"DELETE", own + "/relay", "", false},
		{"POST", own + "/listens", `{"track_id":12,"unix":1600000000}`, false},
		{"DELETE", own + "/listens?track_id=12&unix=1600000000", "", false},
		{"GET", own + "/user/apikeys", "", false},
		{"POST", own + "/user/apikeys", `{"label":"owner key"}`, false},
		{"PATCH", own + "/user/apikeys/999", `{"label":"missing"}`, false},
		{"DELETE", own + "/user/apikeys/999", "", false},
		{"POST", "/apis/web/v1/profiles", `{}`, true},
	}
	for _, tc := range tests {
		t.Run(tc.method+tc.path, func(t *testing.T) {
			require.Equal(t, 401, json160(a, tc.method, tc.path, tc.body, "", "").Code)
			require.Equal(t, 401, json160(a, tc.method, tc.path, tc.body, "", "first-key").Code)
			response := json160(a, tc.method, tc.path, tc.body, cookie, "")
			if tc.rootOnly {
				require.Equal(t, 403, response.Code)
			} else {
				require.NotContains(t, []int{401, 403, 500}, response.Code, response.Body.String())
				other := strings.Replace(tc.path, "/profiles/"+p.Slug, "/profiles/default", 1)
				require.Equal(t, 403, json160(a, tc.method, other, tc.body, cookie, "").Code)
			}
			root := json160(a, tc.method, tc.path, tc.body, a.cookie, "")
			require.NotContains(t, []int{401, 403, 500}, root.Code, root.Body.String())
		})
	}
	// Catalog operations are global regardless of which profile's history contains the item.
	for _, kind := range []string{"artist", "album", "track"} {
		for _, op := range []struct{ method, suffix string }{{"DELETE", ""}, {"PATCH", ""}, {"POST", "/merge"}, {"POST", "/aliases"}, {"DELETE", "/aliases"}, {"PATCH", "/aliases/primary"}} {
			path := "/apis/web/v1/" + kind + "/12" + op.suffix
			require.Equal(t, 403, json160(a, op.method, path, `{}`, cookie, "").Code, path)
			require.Equal(t, 401, json160(a, op.method, path, `{}`, "", "first-key").Code, path)
		}
	}
	require.Equal(t, 403, json160(a, "DELETE", "/apis/web/v1/data", "", cookie, "").Code)
	for _, actor := range []string{"", cookie, a.cookie} {
		require.Equal(t, 404, json160(a, "POST", own+"/unlisted", `{}`, actor, "").Code)
		require.Equal(t, 404, json160(a, "POST", "/admin/profile/1/unlisted", `{}`, actor, "").Code)
	}
}

// TestMassAssignmentAndCurrentRole160 proves role/status/owner cannot be supplied
// through either JSON endpoint, and a demoted cookie loses global access immediately.
func TestMassAssignmentAndCurrentRole160(t *testing.T) {
	a := taskApp139(t)
	cookie, p := owner160(t, a)
	for _, field := range []string{`"role":"admin"`, `"status":"pending"`, `"user_id":1`, `"id":1`} {
		require.Equal(t, 400, json160(a, "PATCH", "/apis/web/v1/user", `{"username":"owner160",`+field+`}`, cookie, "").Code)
		require.Equal(t, 400, json160(a, "PATCH", "/apis/web/v1/profiles/"+p.Slug+"/", `{"display_name":"Owner",`+field+`}`, cookie, "").Code)
	}
	var role, status string
	var owner int32
	require.NoError(t, a.store.QueryRow(`SELECT role,status FROM users WHERE id=?`, p.UserID).Scan(&role, &status))
	require.Equal(t, "user", role)
	require.Equal(t, "active", status)
	require.NoError(t, a.store.QueryRow(`SELECT user_id FROM listening_profiles WHERE id=?`, p.ID).Scan(&owner))
	require.Equal(t, p.UserID, owner)
	// A foreign key ID cannot be renamed or revoked through an authorized own URL.
	var foreignKey int32
	require.NoError(t, a.store.QueryRow(`SELECT id FROM api_keys WHERE key='first-key'`).Scan(&foreignKey))
	keyPath := fmt.Sprintf("/apis/web/v1/profiles/%s/user/apikeys/%d", p.Slug, foreignKey)
	require.Equal(t, 404, json160(a, "PATCH", keyPath, `{"label":"stolen"}`, cookie, "").Code)
	require.Equal(t, 404, json160(a, "DELETE", keyPath, "", cookie, "").Code)
	require.NoError(t, a.store.QueryRow(`SELECT COUNT(*) FROM api_keys WHERE id=? AND key='first-key'`, foreignKey).Scan(&owner))
	require.EqualValues(t, 1, owner)
	// A body-selected destination is ignored: only the server-resolved own profile receives the listen.
	ownListen := "/apis/web/v1/profiles/" + p.Slug + "/listens"
	require.Equal(t, 201, json160(a, "POST", ownListen, `{"track_id":12,"unix":1600000000,"profile_id":1}`, cookie, "").Code)
	var foreignListens int
	require.NoError(t, a.store.QueryRow(`SELECT COUNT(*) FROM listens WHERE profile_id=1 AND listened_at=1600000000`).Scan(&foreignListens))
	require.Zero(t, foreignListens)
	// Keep another active admin before demoting the original actor.
	require.NoError(t, a.store.Exec(`UPDATE users SET role='admin' WHERE id=?`, p.UserID))
	require.NoError(t, a.store.Exec(`UPDATE users SET role='user' WHERE username='administrator'`))
	require.Equal(t, 403, json160(a, "POST", "/apis/web/v1/track/12/aliases", `{"alias":"stale root"}`, a.cookie, "").Code)
	require.Equal(t, 200, json160(a, "GET", "/apis/web/v1/profiles/default/user/apikeys", "", a.cookie, "").Code)
	require.Equal(t, 403, json160(a, "GET", "/apis/web/v1/profiles/second/user/apikeys", "", a.cookie, "").Code)
	require.Equal(t, 200, json160(a, "GET", "/apis/web/v1/user", "", cookie, "").Code)
}

// TestHTMLManagementMatrix160 uses real forms/tickets rather than bypassing CSRF.
func TestHTMLManagementMatrix160(t *testing.T) {
	a := taskApp139(t)
	cookie, p := owner160(t, a)
	pagePath := fmt.Sprintf("/admin?section=keys&profile=%d", p.ID)
	ownPage := taskRequest139(a, "GET", pagePath, cookie, nil, nil)
	require.Equal(t, 200, ownPage.Code, ownPage.Body.String())
	require.NotContains(t, ownPage.Body.String(), "first-key")
	for _, actor := range []struct {
		cookie     string
		own, other int
	}{{"", 401, 401}, {cookie, 200, 403}, {a.cookie, 200, 200}} {
		require.Equal(t, actor.own, taskRequest139(a, "GET", fmt.Sprintf("/admin/profile/%d/export", p.ID), actor.cookie, nil, nil).Code)
		require.Equal(t, actor.other, taskRequest139(a, "GET", "/admin/profile/1/export", actor.cookie, nil, nil).Code)
	}
	for _, section := range []string{"account", "profiles", "keys", "relay", "import", "export"} {
		require.Equal(t, 200, taskRequest139(a, "GET", fmt.Sprintf("/admin?section=%s&profile=%d", section, p.ID), cookie, nil, nil).Code, section)
	}
	for _, op := range []struct {
		suffix string
		values url.Values
	}{
		{"save", url.Values{"display_name": {"Owner"}, "slug": {p.Slug}}},
		{"keys/create", url.Values{"label": {"HTML key"}}},
		{"relay/remove", url.Values{}},
		{"avatar-clear", url.Values{}},
	} {
		page := taskRequest139(a, "GET", pagePath, cookie, nil, nil)
		form := op.values
		form.Set("profile_slug", p.Slug)
		form.Set("csrf_token", taskField139(t, page, "csrf_token"))
		form.Set("submission_token", taskField139(t, page, "submission_token"))
		path := fmt.Sprintf("/admin/profile/%d/%s", p.ID, op.suffix)
		require.Equal(t, 303, taskRequest139(a, "POST", path, cookie, form, nil).Code, op.suffix)
		require.Equal(t, 403, taskRequest139(a, "POST", "/admin/profile/1/"+op.suffix, cookie, form, nil).Code)
	}
	for _, field := range []string{"role", "status", "user_id"} {
		page := taskRequest139(a, "GET", pagePath, cookie, nil, nil)
		form := url.Values{"profile_slug": {p.Slug}, "display_name": {"Owner"}, "slug": {p.Slug}, "csrf_token": {taskField139(t, page, "csrf_token")}, "submission_token": {taskField139(t, page, "submission_token")}, field: {"admin"}}
		require.Equal(t, 400, taskRequest139(a, "POST", fmt.Sprintf("/admin/profile/%d/save", p.ID), cookie, form, nil).Code)
	}
	for _, path := range []string{"/admin/settings", "/admin/profile/1/delete-review", "/second/manage/track/12/edit", "/second/manage/track/12/image", "/second/manage/track/12/merge"} {
		require.Equal(t, 403, taskRequest139(a, "GET", path, cookie, nil, nil).Code, path)
	}
	for _, op := range []string{"metadata", "alias-add", "artwork", "confirm"} {
		require.Equal(t, 403, taskRequest139(a, "POST", "/second/manage/track/12/"+op, cookie, url.Values{}, nil).Code, op)
	}
	// A real own listen can be managed while an identical foreign listen stays untouched.
	require.NoError(t, a.store.Exec(`INSERT INTO listens(profile_id,track_id,listened_at) VALUES(?,12,100)`, p.ID))
	ownTrack := fmt.Sprintf("/%s/manage/track/12", p.Slug)
	require.Equal(t, 200, taskRequest139(a, "GET", ownTrack+"/listens", cookie, nil, nil).Code)
	require.Equal(t, 403, taskRequest139(a, "GET", ownTrack+"/edit", cookie, nil, nil).Code)
	page := taskRequest139(a, "GET", ownTrack+"/listen?listen_at=100", cookie, nil, nil)
	require.Equal(t, 200, page.Code)
	form := url.Values{"profile_id": {fmt.Sprint(p.ID)}, "profile_slug": {p.Slug}, "unix": {"100"}, "confirm": {"1"}, "csrf_token": {taskField139(t, page, "csrf_token")}, "submission_token": {taskField139(t, page, "submission_token")}}
	require.Equal(t, 303, taskRequest139(a, "POST", ownTrack+"/listen-delete", cookie, form, nil).Code)
	var n int
	require.NoError(t, a.store.QueryRow(`SELECT COUNT(*) FROM listens WHERE profile_id=2 AND track_id=12 AND listened_at=100`).Scan(&n))
	require.Equal(t, 1, n)
	// Bulk personal deletion uses a reviewed, session-bound plan and preserves other histories.
	require.NoError(t, a.store.Exec(`INSERT INTO listens(profile_id,track_id,listened_at) VALUES(?,12,200)`, p.ID))
	review := taskRequest139(a, "GET", ownTrack+"/delete", cookie, nil, nil)
	require.Equal(t, 200, review.Code, review.Body.String())
	confirmed := url.Values{"profile_id": {fmt.Sprint(p.ID)}, "profile_slug": {p.Slug}, "confirm": {"1"}}
	for _, field := range []string{"csrf_token", "submission_token", "plan"} {
		confirmed.Set(field, taskField139(t, review, field))
	}
	require.Equal(t, 303, taskRequest139(a, "POST", ownTrack+"/confirm", cookie, confirmed, nil).Code)
	require.NoError(t, a.store.QueryRow(`SELECT COUNT(*) FROM listens WHERE profile_id=? AND track_id=12`, p.ID).Scan(&n))
	require.Zero(t, n)
	require.NoError(t, a.store.QueryRow(`SELECT COUNT(*) FROM listens WHERE profile_id=2 AND track_id=12`).Scan(&n))
	require.Equal(t, 1, n)
	require.NoError(t, a.store.QueryRow(`SELECT COUNT(*) FROM tracks WHERE id=12`).Scan(&n))
	require.Equal(t, 1, n)
	// Unknown tasks fail closed for all roles, including valid root sessions.
	for _, actor := range []string{"", cookie, a.cookie} {
		require.Equal(t, 404, taskRequest139(a, "GET", ownTrack+"/unlisted", actor, nil, nil).Code)
	}
	require.Equal(t, 403, taskRequest139(a, "POST", fmt.Sprintf("/admin/profile/%d/save", p.ID), cookie, url.Values{}, nil).Code)
}

// TestOwnerImportMetadata160 records the explicit exception: new catalog identities
// may receive imported artwork and aliases, while existing identities are not edited.
func TestOwnerImportMetadata160(t *testing.T) {
	a := newProfileTestApp(t)
	cookie, p := owner160(t, a)
	ctx := context.Background()
	req := httptest.NewRequest("GET", "http://koito.test/", nil)
	req.AddCookie(&http.Cookie{Name: "tunetrail_session", Value: cookie})
	identity, err := adminweb.SessionIdentity(req, a.store)
	require.NoError(t, err)
	file := filepath.Join(t.TempDir(), "history.json")
	data := `{"version":"1","user":"someone-else","listens":[{"listened_at":"2023-11-14T22:13:20Z","track":{"aliases":[{"alias":"Import Track","is_primary":true},{"alias":"Track Extra"}]},"album":{"aliases":[{"alias":"Import Album","is_primary":true},{"alias":"Album Extra"}],"image_url":"https://example.invalid/original.jpg"},"artists":[{"aliases":[{"alias":"Import Artist","is_primary":true},{"alias":"Artist Extra"}],"image_url":"https://example.invalid/artist.jpg"}]}]}`
	require.NoError(t, os.WriteFile(file, []byte(data), 0600))
	result, err := admin.ImportHistory(ctx, a.store, identity.User, nil, p.ID, "koito", file)
	require.NoError(t, err)
	require.Equal(t, 1, result.Imported)
	var n int
	require.NoError(t, a.store.QueryRow(`SELECT COUNT(*) FROM listens WHERE profile_id=?`, p.ID).Scan(&n))
	require.Equal(t, 1, n)
	require.NoError(t, a.store.QueryRow(`SELECT COUNT(*) FROM artist_aliases WHERE alias='Artist Extra'`).Scan(&n))
	require.Equal(t, 1, n)
	require.NoError(t, a.store.QueryRow(`SELECT COUNT(*) FROM track_aliases WHERE alias='Track Extra'`).Scan(&n))
	require.Equal(t, 1, n)
	require.NoError(t, a.store.QueryRow(`SELECT COUNT(*) FROM releases WHERE image_source='https://example.invalid/original.jpg'`).Scan(&n))
	require.Equal(t, 1, n)
	// A second export with changed editorial metadata reuses the existing identities.
	changed := strings.ReplaceAll(data, "original.jpg", "changed.jpg")
	changed = strings.ReplaceAll(changed, "Album Extra", "Injected Extra")
	require.NoError(t, os.WriteFile(file, []byte(changed), 0600))
	_, err = admin.ImportHistory(ctx, a.store, identity.User, nil, p.ID, "koito", file)
	require.NoError(t, err)
	require.NoError(t, a.store.QueryRow(`SELECT COUNT(*) FROM release_aliases WHERE alias='Injected Extra'`).Scan(&n))
	require.Zero(t, n)
	require.NoError(t, a.store.QueryRow(`SELECT COUNT(*) FROM releases WHERE image_source='https://example.invalid/changed.jpg'`).Scan(&n))
	require.Zero(t, n)
	_, err = admin.ImportHistory(ctx, a.store, identity.User, nil, 1, "koito", file)
	require.ErrorIs(t, err, admin.ErrForbidden)
}

// sessionUser160 makes direct service fixtures use the same session boundary as HTTP.
func sessionUser160(a *profileTestApp) (*models.User, error) {
	id, err := uuid.Parse(a.cookie)
	if err != nil {
		return nil, err
	}
	identity, err := admin.Authenticate(context.Background(), a.store, id)
	return identity.User, err
}
