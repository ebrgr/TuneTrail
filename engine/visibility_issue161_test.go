package engine

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"image"
	"image/png"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/ebrgr/TuneTrail/imagecache"
	"github.com/ebrgr/TuneTrail/internal/db"
	"github.com/ebrgr/TuneTrail/internal/memkv"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// TestListeningVisibility161 exercises the production route table with real sessions and an artwork cache hit.
func TestListeningVisibility161(t *testing.T) {
	a := taskApp139(t)
	owner, p := owner160(t, a)
	ctx := context.Background()
	id := uuid.New()
	var data bytes.Buffer
	require.NoError(t, png.Encode(&data, image.NewRGBA(image.Rect(0, 0, 16, 16))))
	artwork := data.Bytes()
	require.NoError(t, imagecache.SaveImage(id, bytes.NewReader(artwork)))
	t.Cleanup(func() { _ = imagecache.DeleteImage(id) })
	avatarID := uuid.New()
	require.NoError(t, imagecache.SaveImage(avatarID, bytes.NewReader(artwork)))
	t.Cleanup(func() { _ = imagecache.DeleteImage(avatarID) })
	avatar := avatarID.String()
	require.NoError(t, a.store.SetProfileAvatar(ctx, p.ID, &avatar, nil))
	require.NoError(t, a.store.Exec(fmt.Sprintf("UPDATE releases SET image='%s' WHERE id=11", id)))
	require.NoError(t, a.store.Exec(fmt.Sprintf("INSERT INTO listens(profile_id,track_id,listened_at) VALUES(%d,12,100)", p.ID)))
	require.NoError(t, a.store.Exec("DELETE FROM listens WHERE profile_id IN (2,3)"))
	prefix := "/" + p.Slug
	api := "/apis/web/v1/profiles" + prefix
	legacyImage := fmt.Sprintf("/image/%s/128x128.webp", id)
	scopedImage := fmt.Sprintf("/image/profile/%d/%s/128x128.webp", p.ID, id)
	key, err := a.store.SaveApiKey(ctx, db.SaveApiKeyOpts{Key: "visibility-key", ProfileID: p.ID})
	require.NoError(t, err)
	submit := "/apis/listenbrainz/1/submit-listens"
	t.Cleanup(func() { memkv.Store.Delete(fmt.Sprintf("now-playing:%d", p.ID)) })
	require.Equal(t, 200, a.request("POST", submit, listenBody("playing_now", "Playing 161", 0), key.Key, false).Code)
	require.Contains(t, json160(a, "GET", api+"/now-playing", "", "", "").Body.String(), "Playing 161")

	const publicCache = "public, max-age=2592000"
	imageResponse := taskRequest139(a, "GET", scopedImage, "", nil, nil)
	require.Equal(t, 200, imageResponse.Code)
	require.Equal(t, publicCache, imageResponse.Header().Get("Cache-Control"))
	publicTag := imageResponse.Header().Get("ETag")
	require.NotEmpty(t, publicTag)
	require.Contains(t, taskRequest139(a, "GET", prefix+"/listens", "", nil, nil).Body.String(), scopedImage)
	legacy := taskRequest139(a, "GET", legacyImage, "", nil, nil)
	require.Equal(t, 200, legacy.Code)
	require.Equal(t, publicCache, legacy.Header().Get("Cache-Control"))
	conditional := map[string]string{"If-None-Match": publicTag}
	public304 := taskRequest139(a, "HEAD", scopedImage, "", nil, conditional)
	require.Equal(t, 304, public304.Code)
	require.Equal(t, publicCache, public304.Header().Get("Cache-Control"))

	readPaths := []string{
		prefix, prefix + "/listens", prefix + "/chart/top/artists", prefix + "/track/12", prefix + "/rewind",
		prefix + "/search?q=Source", prefix + "/search?partial=1&q=Source",
		api + "/listens", api + "/stats", api + "/listen-activity", api + "/summary",
		api + "/track/12", api + "/search?q=Source", api + "/now-playing", scopedImage,
	}
	other := ordinarySession161(t, a)
	require.Equal(t, 401, json160(a, "PATCH", api+"/", `{"require_login":true}`, "", "").Code)
	require.Equal(t, 403, json160(a, "PATCH", api+"/", `{"require_login":true}`, other, "").Code)
	for _, protected := range []bool{false, true} {
		if protected {
			pagePath := fmt.Sprintf("/admin?profile=%d", p.ID)
			page := taskRequest139(a, "GET", pagePath, owner, nil, nil)
			require.Equal(t, 200, page.Code)
			form := url.Values{
				"csrf_token": {taskField139(t, page, "csrf_token")}, "submission_token": {taskField139(t, page, "submission_token")},
				"profile_slug": {p.Slug}, "display_name": {p.DisplayName}, "slug": {p.Slug},
				"require_login_present": {"1"}, "require_login": {"1"},
			}
			require.Equal(t, 303, taskRequest139(a, "POST", fmt.Sprintf("/admin/profile/%d/save", p.ID), owner, form, nil).Code)
		}
		directory := taskRequest139(a, "GET", "/", "", nil, nil)
		require.Equal(t, 200, directory.Code)
		for _, slug := range []string{p.Slug, "default", "second", "empty"} {
			require.Contains(t, directory.Body.String(), "/"+slug)
		}
		require.NotContains(t, directory.Body.String(), "Source Track")
		require.NotContains(t, directory.Body.String(), id.String())
		avatarPath := fmt.Sprintf("/image/%s/128x128.webp", avatarID)
		require.Contains(t, directory.Body.String(), avatarPath)
		avatar := json160(a, "GET", avatarPath, "", "", "")
		require.Equal(t, 200, avatar.Code)
		require.Equal(t, publicCache, avatar.Header().Get("Cache-Control"))
		require.Equal(t, protected, bytes.Contains(directory.Body.Bytes(), []byte("🔒")))

		for _, actor := range []struct{ name, cookie string }{
			{"visitor", ""}, {"other", other}, {"owner", owner}, {"root", a.cookie},
		} {
			canRead := actor.name == "owner" || actor.name == "root" || !protected
			for _, path := range readPaths {
				response := taskRequest139(a, "GET", path, actor.cookie, nil, map[string]string{"X-TuneTrail-Partial": "1"})
				if canRead {
					require.Equal(t, 200, response.Code, "%s %s: %s", actor.name, path, response.Body.String())
					if protected {
						require.Equal(t, "private, no-store", response.Header().Get("Cache-Control"), path)
					}
				} else {
					require.Equal(t, 404, response.Code, "%s %s", actor.name, path)
					require.NotContains(t, response.Body.String(), "Source")
					require.Empty(t, response.Header().Get("ETag"))
				}
				if path == prefix {
					require.Equal(t, actor.name == "visitor" && !protected, bytes.Contains(response.Body.Bytes(), []byte("speculationrules")))
				}
			}
			for _, path := range []string{api + "/export", fmt.Sprintf("/admin/profile/%d/export", p.ID)} {
				response := taskRequest139(a, "GET", path, actor.cookie, nil, nil)
				if actor.name == "owner" || actor.name == "root" {
					require.Equal(t, 200, response.Code, path)
					require.Contains(t, response.Body.String(), "Source Track")
				} else {
					if protected {
						require.Contains(t, []int{401, 404}, response.Code, path)
					} else {
						require.Contains(t, []int{401, 403}, response.Code, path)
					}
					require.NotContains(t, response.Body.String(), "Source Track")
				}
			}
		}
	}
	for _, path := range []string{legacyImage, scopedImage} {
		for _, method := range []string{"GET", "HEAD"} {
			response := taskRequest139(a, method, path, "", nil, conditional)
			require.Equal(t, 404, response.Code)
			require.Empty(t, response.Header().Get("ETag"))
		}
	}
	protectedImage := taskRequest139(a, "GET", scopedImage, owner, nil, conditional)
	require.Equal(t, 200, protectedImage.Code)
	require.Equal(t, "private, no-store", protectedImage.Header().Get("Cache-Control"))
	protectedTag := protectedImage.Header().Get("ETag")
	require.NotEmpty(t, protectedTag)
	require.NotEqual(t, publicTag, protectedTag)
	protectedConditional := map[string]string{"If-None-Match": protectedTag}
	protected304 := taskRequest139(a, "GET", scopedImage, owner, nil, protectedConditional)
	require.Equal(t, 304, protected304.Code)
	require.Equal(t, "private, no-store", protected304.Header().Get("Cache-Control"))
	for _, cookie := range []string{"", other} {
		for _, method := range []string{"GET", "HEAD"} {
			denied := taskRequest139(a, method, scopedImage, cookie, nil, protectedConditional)
			require.Equal(t, 404, denied.Code)
			require.Equal(t, "private, no-store", denied.Header().Get("Cache-Control"))
			require.Empty(t, denied.Header().Get("ETag"))
		}
	}

	// A different public profile cannot expose protected-only artists, albums or tracks.
	results := json160(a, "GET", "/apis/web/v1/profiles/default/search?q=Source", "", "", "")
	require.Equal(t, 200, results.Code)
	require.NotContains(t, results.Body.String(), "Source Artist")
	require.NotContains(t, results.Body.String(), "Source Album")
	require.NotContains(t, results.Body.String(), "Source Track")
	require.Equal(t, 404, json160(a, "GET", api+"/now-playing", "", "", key.Key).Code)
	for _, cookie := range []string{"", other} {
		for _, path := range []string{prefix, api + "/stats"} {
			headers := map[string]string{"If-None-Match": "*", "Sec-Purpose": "prefetch"}
			w := taskRequest139(a, "GET", path, cookie, nil, headers)
			missing := taskRequest139(a, "GET", strings.Replace(path, prefix, "/unknown-161", 1), cookie, nil, headers)
			require.Equal(t, missing.Body.String(), w.Body.String())
			require.Equal(t, missing.Header(), w.Header())
			require.Equal(t, 404, w.Code)
		}
	}
	// Public membership can make the bytes public without disclosing a protected association.
	require.NoError(t, a.store.Exec("INSERT INTO listens(profile_id,track_id,listened_at) VALUES(2,12,100)"))
	require.Equal(t, 200, taskRequest139(a, "GET", legacyImage, "", nil, nil).Code)
	require.Equal(t, 404, taskRequest139(a, "GET", scopedImage, "", nil, nil).Code)
	require.Equal(t, 200, json160(a, "PATCH", api+"/", `{"require_login":false}`, a.cookie, "").Code)
	require.Equal(t, 200, taskRequest139(a, "GET", scopedImage, "", nil, map[string]string{"If-None-Match": protectedTag}).Code)
}

func ordinarySession161(t *testing.T, a *profileTestApp) string {
	t.Helper()
	ctx := context.Background()
	u, err := a.store.SaveUser(ctx, db.SaveUserOpts{Username: "other161", Password: "test-password"})
	require.NoError(t, err)
	session, err := a.store.SaveSession(ctx, u.ID, time.Now().Add(time.Hour), false)
	require.NoError(t, err)
	return session.ID.String()
}

// TestProtectedIngestionAndRelay161 verifies the complete machine entry and outgoing HTTP path.
func TestProtectedIngestionAndRelay161(t *testing.T) {
	a := newProfileTestApp(t)
	api := "/apis/web/v1/profiles/default"
	require.Equal(t, 200, json160(a, "PATCH", api+"/", `{"require_login":true}`, a.cookie, "").Code)
	type relayRequest struct{ token, path, title string }
	received := make(chan relayRequest, 1)
	relay := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Payload []struct {
				TrackMetadata struct {
					TrackName string `json:"track_name"`
				} `json:"track_metadata"`
			} `json:"payload"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil || len(body.Payload) != 1 {
			http.Error(w, "invalid listen", 400)
			return
		}
		received <- relayRequest{r.Header.Get("Authorization"), r.URL.Path, body.Payload[0].TrackMetadata.TrackName}
	}))
	defer relay.Close()
	require.Equal(t, 204, json160(a, "PUT", api+"/relay", fmt.Sprintf(`{"url":%q,"token":"relay-key"}`, relay.URL), a.cookie, "").Code)
	submit := "/apis/listenbrainz/1/submit-listens"
	now := time.Now().Add(-time.Hour).Unix()
	require.Equal(t, 200, a.request("POST", submit, listenBody("single", "Protected Relay Track", now), "first-key", false).Code)
	select {
	case got := <-received:
		require.Equal(t, "Token relay-key", got.token)
		require.Equal(t, "/submit-listens", got.path)
		require.Equal(t, "Protected Relay Track", got.title)
	case <-time.After(3 * time.Second):
		t.Fatal("protected listen was not forwarded")
	}
	require.Equal(t, 404, json160(a, "GET", api+"/listens", "", "", "first-key").Code)
	response := json160(a, "GET", api+"/listens", "", a.cookie, "")
	require.Equal(t, 200, response.Code)
	require.Contains(t, response.Body.String(), "Protected Relay Track")

	for slug, name := range map[string]string{"second": "Second", "empty": "Empty"} {
		require.Equal(t, 204, json160(a, "DELETE", "/apis/web/v1/profiles/"+slug+"/", fmt.Sprintf(`{"display_name":%q,"listen_count":0}`, name), a.cookie, "").Code)
	}
	// Transitional routes also enforce the gate when only one profile is available.
	require.Equal(t, 404, json160(a, "GET", "/listens", "", "", "").Code)
	require.Equal(t, 404, json160(a, "GET", "/apis/web/v1/listens", "", "", "").Code)
	require.Equal(t, 302, json160(a, "GET", "/listens", "", a.cookie, "").Code)
	require.Contains(t, json160(a, "GET", "/apis/web/v1/listens", "", a.cookie, "").Body.String(), "Protected Relay Track")
}
