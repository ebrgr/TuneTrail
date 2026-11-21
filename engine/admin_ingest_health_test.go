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
		CREATE TRIGGER health_fixture_failure BEFORE INSERT ON ingest_revisions
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
	require.Contains(t, ownerPage.Body.String(), second)
	require.NotContains(t, ownerPage.Body.String(), first)
	require.NotContains(t, ownerPage.Body.String(), "PRIVATE_")
	require.NotContains(t, ownerPage.Body.String(), keyDigest)
	ownerToken := csrf.FindStringSubmatch(ownerPage.Body.String())
	require.Len(t, ownerToken, 2)
	rootPage := a.request("GET", "/admin/health?profile=1", "", "", true)
	require.Equal(t, http.StatusOK, rootPage.Code)
	require.Contains(t, rootPage.Body.String(), first)
	require.NotContains(t, rootPage.Body.String(), second)
	require.NotContains(t, rootPage.Body.String(), "PRIVATE_")
	require.NotContains(t, rootPage.Body.String(), keyDigest)
	rootToken := csrf.FindStringSubmatch(rootPage.Body.String())
	require.Len(t, rootToken, 2)

	post := func(app *profileTestApp, profile int, receipt, token string) int {
		t.Helper()
		form := url.Values{"csrf_token": {token}, "receipt": {receipt}}
		r := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/admin/health/reprocess?profile=%d", profile), strings.NewReader(form.Encode()))
		r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		r.AddCookie(&http.Cookie{Name: "tunetrail_session", Value: app.cookie})
		w := httptest.NewRecorder()
		app.router.ServeHTTP(w, r)
		return w.Code
	}
	require.Equal(t, http.StatusForbidden, post(&owner, 1, first, ownerToken[1]))
	require.Equal(t, http.StatusForbidden, post(&owner, 2, second, ""))
	require.Equal(t, http.StatusForbidden, post(&owner, 2, second, rootToken[1]))
	require.Equal(t, http.StatusSeeOther, post(&owner, 2, first, ownerToken[1]))
	parked, err := a.store.Count(`SELECT count(*) FROM ingest_events WHERE parked_at IS NOT NULL`)
	require.NoError(t, err)
	require.Equal(t, 2, parked)
	require.Equal(t, http.StatusSeeOther, post(&owner, 2, second, ownerToken[1]))
	require.Equal(t, http.StatusSeeOther, post(&owner, 2, second, ownerToken[1]))
	require.Equal(t, http.StatusSeeOther, post(a, 1, first, rootToken[1]))
	requeued, err := a.store.Count(`SELECT count(*) FROM ingest_events WHERE parked_at IS NULL
		AND next_attempt_at IS NOT NULL AND attempt_count=5 AND retry_attempt_base=5 AND work_version=6`)
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
