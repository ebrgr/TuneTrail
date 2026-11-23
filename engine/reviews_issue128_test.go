package engine

import (
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// section128 returns the card that starts at the element carrying marker, up to its closing tag.
func section128(t *testing.T, body, marker string) string {
	t.Helper()
	start := strings.Index(body, marker)
	require.GreaterOrEqual(t, start, 0, "missing "+marker)
	end := strings.Index(body[start:], "</section>")
	require.Greater(t, end, 0)
	return body[start : start+end]
}

// TestCatalogReviewCards128 checks the three cards of a catalog review: summary and the list of
// changes without any form, and the confirmation alone in the danger zone, in order.
func TestCatalogReviewCards128(t *testing.T) {
	a := taskApp139(t)
	page := taskRequest139(a, "GET", "/second/manage/track/12/review?action=merge&to=13", a.cookie, nil, nil)
	require.Equal(t, 200, page.Code, page.Body.String())
	body := page.Body.String()
	summary := section128(t, body, `aria-labelledby="review-summary-title"`)
	changes := section128(t, body, `aria-labelledby="review-changes-title"`)
	danger := section128(t, body, `aria-labelledby="review-danger-title"`)
	for _, want := range []string{"Impact summary", `class="message message-warning" role="status"`, "<dt>Source</dt>", "<dd>Source Track (ID 12)</dd>", "<dd>Target Track (ID 13)</dd>", "Source-linked listens"} {
		require.Contains(t, summary, want)
	}
	require.Contains(t, changes, "What will change")
	require.Contains(t, changes, "Affected profiles")
	for _, card := range []string{summary, changes} {
		require.NotContains(t, card, "<form")
		require.NotContains(t, card, `name="plan"`)
	}
	for _, want := range []string{"Danger zone", "<form", `name="plan"`, `name="confirm" value="1" required`, `name="csrf_token"`, "Confirm merge", "Cancel"} {
		require.Contains(t, danger, want)
	}
	require.Less(t, strings.Index(body, "review-summary-title"), strings.Index(body, "review-changes-title"))
	require.Less(t, strings.Index(body, "review-changes-title"), strings.Index(body, "review-danger-title"))
	// The dialog fragment keeps the cards one heading level lower, under its own title.
	frag := taskRequest139(a, "GET", "/second/manage/track/12/review?action=merge&to=13&partial=1", a.cookie, nil,
		map[string]string{"X-TuneTrail-Partial": "1", "X-TuneTrail-Dialog": "1"})
	require.Equal(t, 200, frag.Code, frag.Body.String())
	require.Equal(t, "review", frag.Header().Get("X-TuneTrail-Fragment"))
	require.Contains(t, frag.Body.String(), `<h3 id="review-danger-title">Danger zone</h3>`)
	require.NotContains(t, frag.Body.String(), "<h1")
}

// TestReviewScope128 pins the boundary of the three review documents: the catalog and the account
// deletion belong to root, the listens of one profile to its owner or root, and nobody else.
func TestReviewScope128(t *testing.T) {
	a := taskApp139(t)
	ordinary := users128(t, a, "listener")
	ownerCookie, owner := owner160(t, a)
	mine := "/" + owner.Slug + "/manage/track/12"
	catalog := "/second/manage/track/12/review?action=merge&to=13"
	profileReview := "/admin/profile/2/delete-review"
	personal := "/second/manage/track/12/delete"

	for name, tc := range map[string]struct {
		cookie, path string
		want         int
	}{
		"visitor catalog":           {"", catalog, 303},
		"ordinary catalog":          {ordinary, catalog, 403},
		"owner own-profile catalog": {ownerCookie, mine + "/review?action=merge&to=13", 403},
		"root catalog":              {a.cookie, catalog, 200},
		"ordinary account review":   {ordinary, profileReview, 403},
		"owner account review":      {ownerCookie, profileReview, 403},
		"root account review":       {a.cookie, profileReview, 200},
		"ordinary other profile":    {ordinary, personal, 403},
		"owner other profile":       {ownerCookie, personal, 403},
		"owner own profile":         {ownerCookie, mine + "/delete", 200},
		"root any profile":          {a.cookie, personal, 200},
	} {
		out := taskRequest139(a, "GET", tc.path, tc.cookie, nil, nil)
		require.Equal(t, tc.want, out.Code, name)
		if tc.want == 403 {
			// The refusal is the neutral card: no part of the review reaches the body.
			for _, secret := range []string{"review-danger-title", `name="plan"`, "Source (ID"} {
				require.NotContains(t, out.Body.String(), secret, name)
			}
			require.Contains(t, out.Body.String(), "Access denied", name)
		}
	}
	// The owner's own deletion review is a card pair with the danger zone, and the count hook stays.
	own := taskRequest139(a, "GET", mine+"/delete", ownerCookie, nil, nil)
	for _, want := range []string{"delete-summary-title", "delete-danger-title", `data-delete-count="0"`, "data-task-cancel", `class="message message-warning" role="status"`} {
		require.Contains(t, own.Body.String(), want)
	}
	require.NotContains(t, section128(t, own.Body.String(), `aria-labelledby="delete-summary-title"`), "<form")
}

// TestProfileReviewCards128 covers the account deletion review and its obsolete-plan guard.
func TestProfileReviewCards128(t *testing.T) {
	a := taskApp139(t)
	page := taskRequest139(a, "GET", "/admin/profile/2/delete-review", a.cookie, nil, nil)
	require.Equal(t, 200, page.Code)
	body := page.Body.String()
	summary := section128(t, body, `aria-labelledby="profile-review-summary-title"`)
	changes := section128(t, body, `aria-labelledby="profile-review-changes-title"`)
	danger := section128(t, body, `aria-labelledby="profile-review-danger-title"`)
	for _, want := range []string{"permanently deletes the account, its sessions and tokens", `data-listen-count="1"`, "<dd>second</dd>"} {
		require.Contains(t, summary, want)
	}
	for _, want := range []string{"The account, with its sessions and tokens", "Its API keys", "Its relay configuration", "Its listening history", "The shared catalog stays intact"} {
		require.Contains(t, changes, want)
	}
	for _, card := range []string{summary, changes} {
		require.NotContains(t, card, "<form")
	}
	for _, want := range []string{"Danger zone", `name="display_name" required`, `name="confirm" value="1" required`, `name="listen_count" value="1"`, "Delete account and profile"} {
		require.Contains(t, danger, want)
	}
	// An obsolete plan (the listen count moved after the review) is refused, and nothing is deleted.
	require.NoError(t, a.store.Exec(`INSERT INTO listens(profile_id,track_id,listened_at) VALUES(2,13,200)`))
	post := url.Values{"csrf_token": {taskField139(t, page, "csrf_token")}, "submission_token": {taskField139(t, page, "submission_token")},
		"profile_slug": {"second"}, "display_name": {"Second"}, "listen_count": {"1"}, "confirm": {"1"}}
	out := taskRequest139(a, "POST", "/admin/profile/2/delete", a.cookie, post, nil)
	require.Equal(t, 409, out.Code, out.Body.String())
	require.Contains(t, out.Body.String(), "Request status")
}
