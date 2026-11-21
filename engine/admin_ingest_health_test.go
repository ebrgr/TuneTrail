package engine

import (
	"fmt"
	"net/http"
	"testing"
	"time"

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
