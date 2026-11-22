package engine

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/ebrgr/TuneTrail/internal/ingest"
	"github.com/ebrgr/TuneTrail/internal/mbz"
	"github.com/stretchr/testify/require"
)

// TestIngestHealthIsolation exercises real routes with identical track and time across profiles.
func TestIngestHealthIsolation(t *testing.T) {
	a := newProfileTestApp(t)
	const stamp = 1714564800
	require.NoError(t, a.store.Exec(`UPDATE users SET status='active',password=(SELECT password FROM users WHERE id=1) WHERE id=2;
		UPDATE listening_profiles SET require_login=1 WHERE id=2;
		UPDATE relay_delivery_gate SET policy='durable'`))
	for _, key := range []string{"first-key", "second-key"} {
		w := a.request("POST", "/apis/listenbrainz/1/submit-listens", listenBody("single", "Shared Track", stamp), key, false)
		require.Equal(t, 200, w.Code, w.Body.String())
	}
	for _, profile := range []int{1, 2} {
		require.NoError(t, a.store.SetProfileRelay(t.Context(), int32(profile), "https://fixture.invalid/private-url", "PRIVATE_RELAY_TOKEN"))
		require.NoError(t, a.store.Exec(`INSERT INTO integrations(id,profile_id,source_type,display_name,token_ref,created_at) VALUES
			('health:'||?,?,'listenbrainz','Fixture integration '||?,'PRIVATE_TOKEN_REF',0),
			('relay:'||?,?,'listenbrainz-relay','','PRIVATE_TOKEN_REF',0)`,
			profile, profile, profile, profile, profile))
		require.NoError(t, a.store.Exec(`INSERT INTO ingest_events(receipt_id,profile_id,source_instance_id,schema_version,
			source_type,adapter_version,batch_id,item_ordinal,operation,received_at,time_precision,time_semantics,original_snapshot,
			snapshot_digest,digest_version,mode,state,event_id,duplicate_decision,reason_codes,rule_version,evidence,resolved_at,delivery_policy,identity_status,occurrence_id,recording_id)
			SELECT 'health:'||profile_id,profile_id,'health:'||profile_id,1,'listenbrainz','1','fixture',0,'single',?,
			'estimated','start',json_object('client','Fixture client '||profile_id,'private','PRIVATE_RAW_SNAPSHOT'),
			'fixture','1','active','accepted',event_id,'different',json_array('fixture_reason_'||profile_id),'fixture-rule-'||profile_id,
			json_object('delta_seconds',17,'private','PRIVATE_EVIDENCE'),?,'durable','resolved',l.track_id,o.recording_id
			FROM listens l JOIN occurrences o ON o.id=l.track_id WHERE l.profile_id=?`, stamp+int(profile)*60, stamp+120, profile))
		require.NoError(t, a.store.Exec(`INSERT INTO outbox(id,profile_id,destination_id,event_id,operation,delivery_key,state,created_at,attempt_count,result,format_version)
			SELECT 'health:'||profile_id,profile_id,'relay:'||profile_id,event_id,'listen','PRIVATE_DELIVERY_KEY','failed',?,3,
			'{"reason":"item_rejected","private":"PRIVATE_RELAY_RESULT"}',1 FROM listens WHERE profile_id=?`, stamp, profile))
	}
	owner, err := a.store.SaveSession(t.Context(), 2, time.Now().Add(time.Hour), false)
	require.NoError(t, err)
	ownerApp := *a
	ownerApp.cookie = owner.ID.String()

	const health = "/admin/health"
	firstListen := fmt.Sprintf("/default/manage/track/1/listen?listen_at=%d", stamp)
	secondListen := fmt.Sprintf("/second/manage/track/1/listen?listen_at=%d", stamp)
	for _, tc := range []struct {
		name, path      string
		app             *profileTestApp
		session         bool
		key             string
		status, profile int
		want            []string
	}{
		{"visitor on public profile", health + "?profile=1", a, false, "", 200, 0, []string{"Sign in"}},
		{"visitor on protected profile", health + "?profile=2", a, false, "", 200, 0, []string{"Sign in"}},
		{"ingestion key", health + "?profile=1", a, false, "first-key", 401, 0, nil},
		{"owner default", health, &ownerApp, true, "", 200, 2, []string{"Fixture client 2", "fixture_reason_2", "item_rejected", "estimated", "fixture-rule-2", ">17<"}},
		{"owner explicit", health + "?profile=2", &ownerApp, true, "", 200, 2, []string{"Fixture integration 2"}},
		{"owner cross-profile", health + "?profile=1", &ownerApp, true, "", 403, 0, nil},
		{"root first profile", health + "?profile=1", a, true, "", 200, 1, []string{"Fixture client 1"}},
		{"root second profile", health + "?profile=2", a, true, "", 200, 2, []string{"Fixture client 2"}},
		{"visitor listen", firstListen, a, false, "", 303, 0, nil},
		{"owner listen", secondListen, &ownerApp, true, "", 200, 2, []string{"Why is this listen here?", "<samp>listenbrainz</samp>", "fixture_reason_2", "fixture-rule-2", "failed", "2024-05-01T12:02:00Z"}},
		{"owner cross-profile listen", firstListen, &ownerApp, true, "", 403, 0, nil},
		{"root listen", firstListen, a, true, "", 200, 1, []string{"fixture_reason_1", "fixture-rule-1"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			out := tc.app.request(http.MethodGet, tc.path, "", tc.key, tc.session)
			body := out.Body.String()
			require.Equal(t, tc.status, out.Code, body)
			require.Contains(t, out.Header().Get("Cache-Control"), "no-store")
			for _, want := range tc.want {
				require.Contains(t, body, want)
			}
			for _, secret := range []string{"PRIVATE_", "private-url", "first-key", "second-key"} {
				require.NotContains(t, body, secret)
			}
			for _, profile := range []int{1, 2} {
				if profile != tc.profile {
					require.NotContains(t, body, fmt.Sprintf("Fixture client %d", profile))
					require.NotContains(t, body, fmt.Sprintf("fixture_reason_%d", profile))
				}
			}
		})
	}
	linked := a.request(http.MethodGet, health+"?profile=1&discard_receipt=health%3A1", "", "", true)
	require.Equal(t, http.StatusConflict, linked.Code)
	require.Contains(t, linked.Body.String(), "Use listen deletion for receipts linked to a listen.")

	// A retry counts only for the profile whose destination owns the delivery.
	require.NoError(t, a.store.Exec(`UPDATE outbox SET state='retryable',next_attempt_at=? WHERE id='health:1'`, stamp))
	for profile, count := range map[int]int{1: 1, 2: 0} {
		page := a.request("GET", fmt.Sprintf("/admin/health?profile=%d", profile), "", "", true)
		require.Equal(t, http.StatusOK, page.Code)
		require.Contains(t, page.Body.String(), fmt.Sprintf("<span>Pending Outbox deliveries: %d</span>", count))
	}
}

// TestIngestHealthReprocess uses the same profile authorization for the queue and its action.
func TestIngestHealthReprocess(t *testing.T) {
	a := newProfileTestApp(t)
	require.NoError(t, a.store.Exec(`UPDATE users SET status='active',password=(SELECT password FROM users WHERE id=1) WHERE id=2;
		INSERT INTO profile_ingest_flags(profile_id,active) VALUES(1,1),(2,1)`))
	const payload = "PRIVATE_SNAPSHOT_TITLE"
	const idempotencyKey = "PRIVATE_IDEMPOTENCY_KEY"
	keyDigest := ingest.Digest([]byte(idempotencyKey))
	for _, key := range []string{"first-key", "second-key"} {
		r := httptest.NewRequest(http.MethodPost, "/apis/listenbrainz/1/submit-listens", strings.NewReader(listenBody("single", payload, 1714564800)))
		r.Header.Set("Authorization", "Token "+key)
		r.Header.Set("Content-Type", "application/json")
		r.Header.Set("Idempotency-Key", idempotencyKey)
		w := httptest.NewRecorder()
		a.router.ServeHTTP(w, r)
		require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	}
	// Advance to the fifth claim. The trigger aborts the decision and its listen together.
	require.NoError(t, a.store.Exec(`UPDATE ingest_events SET attempt_count=4,work_version=4;
		CREATE TRIGGER health_fixture_failure BEFORE INSERT ON ingest_revisions WHEN NEW.author='automatic'
		BEGIN SELECT RAISE(ABORT, 'persistent fixture failure'); END`))
	for range 2 {
		found, err := a.store.ResolveNextIngest(t.Context(), &mbz.MbzErrorCaller{})
		require.True(t, found)
		require.ErrorContains(t, err, "persistent fixture failure")
	}
	listens, err := a.store.Count(`SELECT count(*) FROM listens`)
	require.NoError(t, err)
	require.Zero(t, listens)
	var first, second string
	require.NoError(t, a.store.QueryRow(`SELECT receipt_id FROM ingest_events WHERE profile_id=1`).Scan(&first))
	require.NoError(t, a.store.QueryRow(`SELECT receipt_id FROM ingest_events WHERE profile_id=2`).Scan(&second))
	session, err := a.store.SaveSession(t.Context(), 2, time.Now().Add(time.Hour), false)
	require.NoError(t, err)
	owner := *a
	owner.cookie = session.ID.String()
	csrf := regexp.MustCompile(`name="csrf_token" value="([^"]+)"`)
	ownerPage := owner.request("GET", "/admin/health?profile=2", "", "", true)
	require.Equal(t, http.StatusOK, ownerPage.Code)
	require.Contains(t, ownerPage.Body.String(), `href="#health-failures" data-tone="error"`)
	require.Contains(t, ownerPage.Body.String(), "<span>Permanent failures: 1</span>")
	require.Contains(t, ownerPage.Body.String(), second)
	require.NotContains(t, ownerPage.Body.String(), first)
	require.Contains(t, ownerPage.Body.String(), payload)
	require.NotContains(t, ownerPage.Body.String(), idempotencyKey)
	require.NotContains(t, ownerPage.Body.String(), "second-key")
	require.NotContains(t, ownerPage.Body.String(), keyDigest)
	ownerToken := csrf.FindStringSubmatch(ownerPage.Body.String())
	require.Len(t, ownerToken, 2)
	rootPage := a.request("GET", "/admin/health?profile=1", "", "", true)
	require.Equal(t, http.StatusOK, rootPage.Code)
	require.Contains(t, rootPage.Body.String(), first)
	require.NotContains(t, rootPage.Body.String(), second)
	require.NotContains(t, rootPage.Body.String(), idempotencyKey)
	require.NotContains(t, rootPage.Body.String(), "first-key")
	require.NotContains(t, rootPage.Body.String(), keyDigest)
	rootToken := csrf.FindStringSubmatch(rootPage.Body.String())
	require.Len(t, rootToken, 2)

	post := func(app *profileTestApp, profile int, action, field, value, token string) int {
		t.Helper()
		form := url.Values{"csrf_token": {token}, field: {value}}
		r := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/admin/health/%s?profile=%d", action, profile), strings.NewReader(form.Encode()))
		r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		r.AddCookie(&http.Cookie{Name: "tunetrail_session", Value: app.cookie})
		w := httptest.NewRecorder()
		app.router.ServeHTTP(w, r)
		return w.Code
	}
	require.Equal(t, http.StatusForbidden, post(&owner, 1, "reprocess", "receipt", first, ownerToken[1]))
	require.Equal(t, http.StatusForbidden, post(&owner, 2, "reprocess", "receipt", second, ""))
	require.Equal(t, http.StatusForbidden, post(&owner, 2, "reprocess", "receipt", second, rootToken[1]))
	require.Equal(t, http.StatusSeeOther, post(&owner, 2, "reprocess", "receipt", first, ownerToken[1]))
	parked, err := a.store.Count(`SELECT count(*) FROM ingest_events WHERE parked_at IS NOT NULL`)
	require.NoError(t, err)
	require.Equal(t, 2, parked)
	// A stale form or a form submitted against a different profile cannot affect either queue.
	require.Contains(t, ownerPage.Body.String(), "/admin/health/reprocess-all?profile=2")
	require.Equal(t, http.StatusForbidden, post(&owner, 1, "reprocess-all", "profile", "1", ownerToken[1]))
	require.Equal(t, http.StatusForbidden, post(&owner, 2, "reprocess-all", "profile", "2", ""))
	require.Equal(t, http.StatusForbidden, post(&owner, 2, "reprocess-all", "profile", "2", rootToken[1]))
	require.Equal(t, http.StatusNotFound, a.request("GET", "/admin/health/reprocess-all?profile=1", "", "", true).Code)
	require.Equal(t, http.StatusSeeOther, post(a, 1, "reprocess-all", "profile", "2", rootToken[1]))
	require.Equal(t, http.StatusSeeOther, post(&owner, 2, "reprocess-all", "profile", "2", ownerToken[1]))
	require.Equal(t, http.StatusSeeOther, post(a, 2, "reprocess-all", "profile", "2", rootToken[1]))
	parked, err = a.store.Count(`SELECT count(*) FROM ingest_events WHERE profile_id=1 AND parked_at IS NOT NULL
		AND work_version=5 AND reprocessed_by IS NULL`)
	require.NoError(t, err)
	require.Equal(t, 1, parked)
	require.Equal(t, http.StatusSeeOther, post(a, 1, "reprocess-all", "profile", "1", rootToken[1]))

	// Park the first receipt again and verify that a single reprocess updates its audit.
	var reprocessedAt int64
	require.NoError(t, a.store.QueryRow(`SELECT reprocessed_at FROM ingest_events WHERE receipt_id=? AND reprocessed_by=1`, first).Scan(&reprocessedAt))
	require.Positive(t, reprocessedAt)
	require.NoError(t, a.store.Exec(`UPDATE ingest_events SET parked_at=1,park_reason='retry_exhausted',
		reprocessed_at=1 WHERE receipt_id=?`, first))
	require.Equal(t, http.StatusSeeOther, post(a, 1, "reprocess", "receipt", first, rootToken[1]))
	require.Equal(t, http.StatusSeeOther, post(a, 1, "reprocess", "receipt", first, rootToken[1]))
	requeued, err := a.store.Count(`SELECT count(*) FROM ingest_events WHERE parked_at IS NULL
		AND next_attempt_at IS NOT NULL AND attempt_count=5 AND retry_attempt_base=5 AND claim_attempt_base=5
		AND reprocessed_at=next_attempt_at AND reprocessed_at>1
		AND ((profile_id=1 AND work_version=7 AND reprocessed_by=1)
		  OR (profile_id=2 AND work_version=6 AND reprocessed_by=2))`)
	require.NoError(t, err)
	require.Equal(t, 2, requeued)
	require.NoError(t, a.store.Exec(`DROP TRIGGER health_fixture_failure`))
	for range 2 {
		found, err := a.store.ResolveNextIngest(t.Context(), &mbz.MbzErrorCaller{})
		require.NoError(t, err)
		require.True(t, found)
	}
	listens, err = a.store.Count(`SELECT count(*) FROM ingest_events e JOIN listens l USING(profile_id,event_id)
		WHERE e.state='accepted' AND e.attempt_count=6 AND e.client_idempotency_key=?`, keyDigest)
	require.NoError(t, err)
	require.Equal(t, 2, listens)
}

// TestIngestHealthReviewActions checks session authorization and the complete review form flow.
func TestIngestHealthReviewActions(t *testing.T) {
	a := newProfileTestApp(t)
	const stamp = 1714564800
	require.NoError(t, a.store.Exec(`UPDATE users SET status='active',password=(SELECT password FROM users WHERE id=1) WHERE id=2`))
	for _, key := range []string{"first-key", "second-key"} {
		w := a.request("POST", "/apis/listenbrainz/1/submit-listens", listenBody("single", "Review Track", stamp), key, false)
		require.Equal(t, 200, w.Code, w.Body.String())
	}
	require.NoError(t, a.store.Exec(`INSERT INTO profile_ingest_flags(profile_id,active) VALUES(1,1),(2,1)`))
	receipts := map[int]string{1: "f0000000-0000-4000-8000-000000000181", 2: "f0000000-0000-4000-8000-000000000182"}
	for profile, receipt := range receipts {
		require.NoError(t, a.store.Exec(`INSERT INTO integrations(id,profile_id,source_type,token_ref,created_at)
			VALUES('review:'||?,?,'listenbrainz','PRIVATE_TOKEN',0)`, profile, profile))
		require.NoError(t, a.store.Exec(`INSERT INTO ingest_events(receipt_id,profile_id,source_instance_id,schema_version,
			source_type,adapter_version,batch_id,item_ordinal,operation,received_at,listened_at,time_precision,time_semantics,
			original_snapshot,snapshot_digest,digest_version,mode,state,duplicate_decision,reason_codes,evidence,rule_version,
			resolution_version,occurrence_id,recording_id,identity_status)
			SELECT ?,l.profile_id,'review:'||l.profile_id,1,'listenbrainz','1','review',0,'single',?,
			'2024-05-01T12:00:00.000000000Z','second','start','{"private":"PRIVATE_BODY"}','fixture','1','active',
			'pending_duplicate','review','["legacy_identity_collision"]',json_object('candidate_event_id',l.event_id,'private','PRIVATE_EVIDENCE'),
			'active-v1',1,o.id,o.recording_id,'resolved'
			FROM listens l JOIN occurrences o ON o.id=l.track_id WHERE l.profile_id=?`, receipt, stamp, profile))
	}
	session, err := a.store.SaveSession(t.Context(), 2, time.Now().Add(time.Hour), false)
	require.NoError(t, err)
	owner := *a
	owner.cookie = session.ID.String()
	tokenPattern := regexp.MustCompile(`name="csrf_token" value="([^"]+)"`)
	tokens := make(map[int]string, 2)
	digests := make(map[int]string, 2)
	var firstCandidate string
	for profile, app := range map[int]*profileTestApp{1: a, 2: &owner} {
		w := app.request("GET", fmt.Sprintf("/admin/health?profile=%d", profile), "", "", true)
		html := w.Body.String()
		require.Equal(t, http.StatusOK, w.Code, html)
		require.Contains(t, html, `href="#health-review" data-tone="warning"`)
		require.Contains(t, html, "<span>Pending reviews: 1</span>")
		require.Equal(t, 1, strings.Count(html, `href="#health-review"`))
		require.Contains(t, html, `<a href="#health-discard">Discard</a>`)
		for _, want := range []string{"Accept all as duplicates", "Keep all as separate listens", "View evidence", "Accept as duplicate", "Keep as a separate listen", receipts[profile]} {
			require.Contains(t, html, want)
		}
		require.NotContains(t, html, receipts[3-profile])
		require.NotContains(t, html, "PRIVATE_")
		match := tokenPattern.FindStringSubmatch(html)
		require.Len(t, match, 2)
		tokens[profile] = match[1]
		bulk := regexp.MustCompile(`(?s)name="decision" value="different"\s*>\s*<input type="hidden" name="digest" value="([a-f0-9]{64})"`).FindStringSubmatch(html)
		require.Len(t, bulk, 2)
		digests[profile] = bulk[1]
		if profile == 1 {
			option := regexp.MustCompile(`<option value="([a-f0-9-]+)">Review Track`).FindStringSubmatch(html)
			require.Len(t, option, 2)
			firstCandidate = option[1]
		}
	}
	post := func(app *profileTestApp, action string, profile int, form url.Values) int {
		t.Helper()
		path := fmt.Sprintf("/admin/health/%s?profile=%d", action, profile)
		r := httptest.NewRequest(http.MethodPost, path, strings.NewReader(form.Encode()))
		r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		r.AddCookie(&http.Cookie{Name: "tunetrail_session", Value: app.cookie})
		w := httptest.NewRecorder()
		app.router.ServeHTTP(w, r)
		return w.Code
	}
	review := url.Values{"receipt": {receipts[2]}, "version": {"1"}, "decision": {"different"}}
	require.Equal(t, http.StatusForbidden, post(&owner, "review", 2, review))
	review.Set("csrf_token", tokens[1])
	require.Equal(t, http.StatusForbidden, post(&owner, "review", 2, review))
	review.Set("csrf_token", tokens[2])
	require.Equal(t, http.StatusForbidden, post(&owner, "review", 1, review))
	review.Set("receipt", receipts[1])
	require.Equal(t, http.StatusNotFound, post(&owner, "review", 2, review))
	w := owner.request("GET", "/admin/health/review?profile=2&receipt="+receipts[2], "", "", true)
	require.Equal(t, http.StatusNotFound, w.Code)

	flag := url.Values{"csrf_token": {tokens[2]}, "enabled": {"1"}}
	require.Equal(t, http.StatusForbidden, post(&owner, "review-setting", 1, flag))
	require.Equal(t, http.StatusSeeOther, post(&owner, "review-setting", 2, flag))
	w = owner.request("GET", "/admin/health?profile=2", "", "", true)
	require.Contains(t, w.Body.String(), "Turn off duplicate review")

	// A bulk action uses the same session gate and cannot replay onto the next batch.
	bulk := url.Values{"decision": {"different"}, "digest": {digests[2]}}
	require.Equal(t, http.StatusForbidden, post(&owner, "review-batch", 2, bulk))
	bulk.Set("csrf_token", tokens[2])
	require.Equal(t, http.StatusForbidden, post(&owner, "review-batch", 1, bulk))
	bulk.Set("digest", digests[1])
	require.Equal(t, http.StatusConflict, post(&owner, "review-batch", 2, bulk))
	bulk.Set("digest", digests[2])
	require.Equal(t, http.StatusSeeOther, post(&owner, "review-batch", 2, bulk))
	require.Equal(t, http.StatusConflict, post(&owner, "review-batch", 2, bulk))
	review.Set("receipt", receipts[1])
	review.Set("csrf_token", tokens[1])
	review.Set("decision", "same")
	review.Set("candidate", firstCandidate)
	require.Equal(t, http.StatusSeeOther, post(a, "review", 1, review))
	count, err := a.store.Count(`SELECT count(*) FROM ingest_revisions WHERE author='user'
		AND ((profile_id=1 AND decision='same' AND json_extract(evidence,'$.decided_by')=1)
		  OR (profile_id=2 AND decision='different' AND json_extract(evidence,'$.decided_by')=2))`)
	require.NoError(t, err)
	require.Equal(t, 2, count)
	localized := httptest.NewRequest(http.MethodGet, "/admin/health?profile=2", nil)
	localized.Header.Set("Accept-Language", "pt-BR")
	localized.AddCookie(&http.Cookie{Name: "tunetrail_session", Value: owner.cookie})
	out := httptest.NewRecorder()
	owner.router.ServeHTTP(out, localized)
	require.Equal(t, http.StatusOK, out.Code)
	require.Contains(t, out.Body.String(), "Pendências de revisão")
}

func TestIngestHealthInboxKeyFilter(t *testing.T) {
	a := newProfileTestApp(t)
	// The empty profile shows four neutral links and decorative heading icons.
	empty := a.request("GET", "/admin/health?profile=1", "", "", true)
	require.Equal(t, http.StatusOK, empty.Code)
	for anchor, label := range map[string]string{
		"review": "Pending reviews", "failures": "Permanent failures",
		"receipts": "Missing or incomplete metadata", "relay": "Pending Outbox deliveries",
	} {
		require.Regexp(t, `(?s)<a href="#health-`+anchor+`">\s*<svg .*?</svg>\s+<span>`+label+`: 0</span>\s*</a>`, empty.Body.String())
	}
	require.Regexp(t, `<h2><svg [^>]*class="lucide lucide-inbox" aria-hidden="true">.*?</svg>\s+Inbox</h2>`, empty.Body.String())
	require.Regexp(t, `<h2><svg [^>]*class="lucide lucide-send" aria-hidden="true">.*?</svg>\s+Outbox</h2>`, empty.Body.String())

	require.NoError(t, a.store.Exec(`INSERT INTO api_keys(id,key,profile_id,created_at,label)
		VALUES(1841,'INBOX_SECRET_A',1,0,'Inbox player A'),(1842,'INBOX_SECRET_B',1,0,'Inbox player B')`))
	for _, input := range []struct{ key, title string }{
		{"INBOX_SECRET_A", "Track from A"}, {"INBOX_SECRET_B", "Track from B"},
		{"second-key", "Track from a different profile"},
	} {
		// Omit the release so each submitted receipt contributes to the metadata count.
		payload := strings.Replace(listenBody("single", input.title, 1714564800), `"release_name":"Shared Album",`, "", 1)
		result := a.request("POST", "/apis/listenbrainz/1/submit-listens", payload, input.key, false)
		require.Equal(t, http.StatusOK, result.Code)
	}
	count, err := a.store.Count(`SELECT count(*) FROM ingest_events
		WHERE profile_id=1 AND api_key_id IN (1841,1842) AND original_snapshot NOT LIKE '%INBOX_SECRET%'`)
	require.NoError(t, err)
	require.EqualValues(t, 2, count)
	require.NoError(t, a.store.Exec(`DELETE FROM api_keys WHERE id=1842`))
	page := a.request("GET", "/admin/health?profile=1&days=7&key=1841", "", "", true)
	require.Equal(t, http.StatusOK, page.Code)
	body := page.Body.String()
	require.Contains(t, body, `href="#health-receipts" data-tone="warning"`)
	require.Contains(t, body, "<span>Missing or incomplete metadata: 1</span>")
	unfiltered := a.request("GET", "/admin/health?profile=1&days=7", "", "", true)
	require.Equal(t, http.StatusOK, unfiltered.Code)
	require.Contains(t, unfiltered.Body.String(), "<span>Missing or incomplete metadata: 2</span>")
	require.Contains(t, body, "Inbox player A")
	require.Contains(t, body, "Removed key")
	require.Contains(t, body, "Track from A")
	for _, private := range []string{"INBOX_SECRET", "Track from B", "Track from a different profile", "second-key"} {
		require.NotContains(t, body, private)
	}
}

// TestIngestHealthDiscard submits the actual confirmation fields rendered to each account.
func TestIngestHealthDiscard(t *testing.T) {
	a := newProfileTestApp(t)
	require.NoError(t, a.store.Exec(`UPDATE users SET status='active',password=(SELECT password FROM users WHERE id=1) WHERE id=2;
		INSERT INTO profile_ingest_flags(profile_id,active) VALUES(1,1),(2,1)`))
	for _, key := range []string{"first-key", "second-key"} {
		w := a.request("POST", "/apis/listenbrainz/1/submit-listens", listenBody("single", "Discard fixture", 1714564800), key, false)
		require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	}
	var first, second string
	require.NoError(t, a.store.QueryRow(`SELECT receipt_id FROM ingest_events WHERE profile_id=1`).Scan(&first))
	require.NoError(t, a.store.QueryRow(`SELECT receipt_id FROM ingest_events WHERE profile_id=2`).Scan(&second))

	session, err := a.store.SaveSession(t.Context(), 2, time.Now().Add(time.Hour), false)
	require.NoError(t, err)
	owner := *a
	owner.cookie = session.ID.String()

	formFrom := func(app *profileTestApp, path string) url.Values {
		t.Helper()
		w := app.request(http.MethodGet, path, "", "", true)
		body := w.Body.String()
		require.Equal(t, http.StatusOK, w.Code, body)
		require.Contains(t, body, "Confirm discard")
		require.Contains(t, body, "Listens already sent to a relay remain at the destination.")
		formTag := regexp.MustCompile(`(?s)<form [^>]*action="/admin/health/discard\?[^"]*"[^>]*>(.*?)</form>`)
		rendered := formTag.FindStringSubmatch(body)
		require.Len(t, rendered, 2)
		form := url.Values{}
		for _, match := range regexp.MustCompile(`name="([^"]+)" value="([^"]+)"`).FindAllStringSubmatch(rendered[1], -1) {
			form.Set(match[1], match[2])
		}
		require.NotEmpty(t, form.Get("digest"))
		require.NotEmpty(t, form.Get("csrf_token"))
		return form
	}
	post := func(app *profileTestApp, path string, form url.Values) *httptest.ResponseRecorder {
		t.Helper()
		r := httptest.NewRequest(http.MethodPost, path, strings.NewReader(form.Encode()))
		r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		r.AddCookie(&http.Cookie{Name: "tunetrail_session", Value: app.cookie})
		w := httptest.NewRecorder()
		app.router.ServeHTTP(w, r)
		return w
	}

	const action = "/admin/health/discard?profile="
	one := formFrom(&owner, "/admin/health?profile=2&discard_receipt="+second)
	require.Equal(t, second, one.Get("receipt"))
	rootBatch := formFrom(a, "/admin/health?profile=1")
	require.Empty(t, rootBatch.Get("receipt"))
	require.Equal(t, "1", rootBatch.Get("confirm"))
	rootBatch.Del("confirm")
	require.Equal(t, http.StatusBadRequest, post(a, action+"1", rootBatch).Code)
	rootBatch.Set("confirm", "1")
	require.Equal(t, http.StatusNotFound, owner.request(http.MethodGet, action+"2", "", "", true).Code)
	require.Equal(t, http.StatusForbidden, post(&owner, action+"1", one).Code)
	require.Equal(t, http.StatusForbidden, post(a, action+"2", one).Code)
	token := one.Get("csrf_token")
	one.Del("csrf_token")
	require.Equal(t, http.StatusForbidden, post(&owner, action+"2", one).Code)
	one.Set("csrf_token", token)
	one.Del("confirm")
	require.Equal(t, http.StatusBadRequest, post(&owner, action+"2", one).Code)
	one.Set("confirm", "1")
	require.Equal(t, http.StatusSeeOther, post(&owner, action+"2", one).Code)
	require.Equal(t, http.StatusSeeOther, post(&owner, action+"2", one).Code)
	rootBatch.Set("receipt", second)
	require.Equal(t, http.StatusNotFound, post(a, action+"1", rootBatch).Code)
	rootBatch.Del("receipt")
	require.Equal(t, http.StatusSeeOther, post(a, action+"1", rootBatch).Code)
	require.Equal(t, http.StatusConflict, post(a, action+"1", rootBatch).Code)
	count, err := a.store.Count(`SELECT count(*) FROM ingest_discards WHERE
		(profile_id=1 AND actor_id=1 AND receipt_id=?) OR (profile_id=2 AND actor_id=2 AND receipt_id=?)`, first, second)
	require.NoError(t, err)
	require.EqualValues(t, 2, count)
}
