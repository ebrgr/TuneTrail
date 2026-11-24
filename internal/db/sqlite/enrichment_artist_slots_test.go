package sqlite

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"
	"time"

	"github.com/ebrgr/TuneTrail/internal/db"
	"github.com/ebrgr/TuneTrail/internal/enrichment"
	"github.com/ebrgr/TuneTrail/internal/ingest"
	"github.com/ebrgr/TuneTrail/internal/listenbrainz"
	"github.com/ebrgr/TuneTrail/internal/mbz"
	"github.com/ebrgr/TuneTrail/internal/testfixtures"
	"github.com/stretchr/testify/require"
)

// TestListenBrainzSparseArtistSlots covers real JSON ingestion, both providers, and archive restoration.
func TestListenBrainzSparseArtistSlots(t *testing.T) {
	names := []string{"A", "B", "C"}
	ids := []string{knownArtist174, enrichedArtist174, "33333333-3333-4333-8333-333333333333"}
	for _, scenario := range []struct {
		name, source string
		hole         int
	}{
		{"first via ListenBrainz", enrichment.ListenBrainzSource, 0},
		{"middle via Subsonic", "navidrome-subsonic", 1},
		{"last via ListenBrainz after Subsonic", enrichment.ListenBrainzSource, 2},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			s := convertedOccurrenceFixture(t)
			require.NoError(t, s.ConfigureEnrichment(t.Context(), scenario.hole != 0))
			approveListenBrainz175(t, s, 1, enrichment.ListenBrainzBase, "profile-one")
			slots := slices.Clone(ids)
			slots[scenario.hole] = ""
			rawSlots, err := json.Marshal(slots)
			require.NoError(t, err)
			var payload listenbrainz.LbzSubmitListenPayload
			require.NoError(t, json.Unmarshal(fmt.Appendf(nil, `{"listened_at":1700000750,"track_metadata":{
				"track_name":"Sparse song","artist_name":"A & B & C","release_name":"Sparse album","additional_info":{
				"artist_names":["A","B","C"],"artist_mbids":%s,"recording_mbid":%q}}}`, rawSlots, fixtureRecording), &payload))
			snapshot := ingest.ListenBrainzSnapshot(payload.TrackMeta)
			require.Equal(t, slots, snapshot.ArtistMBIDs)
			receipt := admitOccurrenceFixture(t, s, snapshot, int64(payload.ListenedAt), "sparse-credits")
			resolveOccurrenceFixture(t, s, &mbz.MbzErrorCaller{})
			requireReceiptState(t, s, receipt, "accepted")
			var claim enrichmentClaim
			var event string
			require.NoError(t, s.db.QueryRow(`SELECT profile_id,original_snapshot,occurrence_id,recording_id,event_id
				FROM ingest_events WHERE receipt_id=?`, receipt).Scan(&claim.profile, &claim.snapshot, &claim.occurrence, &claim.recording, &event))
			claim.receipt = receipt
			require.NoError(t, s.enrichmentKnownIDs(t.Context(), &claim, &snapshot, ""))
			target, err := prepareListenBrainz(t.Context(), s.db, claim, enrichment.Result{})
			require.NoError(t, err)
			require.Equal(t, []string{names[scenario.hole]}, target.request.Artists)
			require.False(t, target.request.Recording)

			// For relayed payloads the preserved slots stay aligned on another TuneTrail receiver.
			packet, err := encodeRelayItem(1, target.request.Snapshot, int64(payload.ListenedAt))
			require.NoError(t, err)
			var relayed listenbrainz.LbzSubmitListenPayload
			require.NoError(t, json.Unmarshal(packet, &relayed))
			require.Equal(t, slots, ingest.ListenBrainzSnapshot(relayed.TrackMeta).ArtistMBIDs)

			var primary enrichment.Provider
			if scenario.hole != 0 {
				subIDs := slices.Clone(slots)
				if scenario.source == "navidrome-subsonic" {
					subIDs[scenario.hole] = ids[scenario.hole]
				}
				sub := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
					_, _ = fmt.Fprintf(w, `{"subsonic-response":{"status":"ok","searchResult3":{"song":[{
						"title":"Sparse song","artist":"A & B & C","album":"Sparse album","mediaType":"song",
						"musicBrainzId":%q,"artists":[{"name":"C","musicBrainzId":%q},{"name":"A","musicBrainzId":%q},{"name":"B","musicBrainzId":%q}]}]}}}`,
						fixtureRecording, subIDs[2], subIDs[0], subIDs[1])
				}))
				defer sub.Close()
				primary, err = enrichment.NewSubsonic(sub.URL, "", "TuneTrail/test", time.Second, time.Millisecond)
				require.NoError(t, err)
			}
			fallback := listenBrainz175Fixture(t, func(w http.ResponseWriter, _ *http.Request) {
				if scenario.source != enrichment.ListenBrainzSource {
					t.Error("fallback requested after Subsonic supplied the missing ID")
				}
				_, _ = fmt.Fprintf(w, `{"recording_mbid":%q,"recording_name":"Sparse song","release_name":"Sparse album",
					"artist_credit_name":"A & B & C","artist_mbids":[%q,%q,%q],"metadata":{"artist":{"artists":[
					{"name":"C","artist_mbid":%q},{"name":"A","artist_mbid":%q},{"name":"B","artist_mbid":%q}]}}}`,
					fixtureRecording, ids[0], ids[1], ids[2], ids[2], ids[0], ids[1])
			})
			found, err := s.ResolveNextEnrichmentWithFallback(t.Context(), primary, fallback)
			require.NoError(t, err)
			require.True(t, found)
			require.Equal(t, 1, scalar(t, s, `SELECT count(*) FROM metadata_values`))
			require.Equal(t, 1, scalar(t, s, `SELECT count(*) FROM metadata_values m JOIN artist_aliases a USING(artist_id)
				JOIN ingest_events e ON (e.event_id,e.profile_id)=(m.event_id,m.profile_id)
				WHERE e.receipt_id=? AND a.alias=? AND m.value=? AND m.source=?`, receipt, names[scenario.hole], ids[scenario.hole], scenario.source))

			page, err := s.GetExportPage(t.Context(), db.GetExportPageOpts{ProfileID: 1, Limit: 20})
			require.NoError(t, err)
			var item *db.ExportItem
			for _, row := range page {
				if row.Cursor == event {
					item = row
				}
			}
			require.NotNil(t, item)
			require.Len(t, item.Artists, len(names))
			wantSources := []string{"source-event", "source-event", "source-event"}
			wantSources[scenario.hole] = scenario.source
			for _, credit := range item.Artists {
				index := slices.Index(names, credit.Name)
				require.NotEqual(t, -1, index)
				require.NotNil(t, credit.MbzID)
				require.Equal(t, ids[index], credit.MbzID.String())
				require.Equal(t, wantSources[index], credit.MBIDSource)
			}
			restored, _, _ := migrationDB(t, currentSchemaVersion)
			testfixtures.Profile(t, restored, 2, "restore", "Restore", 1)
			count, err := restored.RestoreArchive(t.Context(), 2, []*db.ExportItem{item})
			require.NoError(t, err)
			require.Equal(t, 1, count)
			for i, name := range names {
				require.Equal(t, 1, scalar(t, restored, `SELECT count(*) FROM metadata_values m JOIN artist_aliases a USING(artist_id)
					WHERE m.profile_id=2 AND a.alias=? AND m.value=? AND m.source=?`, name, ids[i], wantSources[i]))
			}
		})
	}
}

// TestListenBrainzSparseArtistDigest keeps keyed retries compatible with compact snapshots.
func TestListenBrainzSparseArtistDigest(t *testing.T) {
	s := convertedOccurrenceFixture(t)
	ctx := t.Context()
	var payload listenbrainz.LbzSubmitListenPayload
	require.NoError(t, json.Unmarshal(fmt.Appendf(nil, `{"track_metadata":{
		"track_name":"Sparse song","artist_name":"A & B","additional_info":{"artist_names":["A","B"],"artist_mbids":[%q,""]}}}`,
		knownArtist174), &payload))
	snapshot := ingest.ListenBrainzSnapshot(payload.TrackMeta)
	legacy := fmt.Sprintf(`{"artist_mbids":[%q],"artist_names":["A","B"],"artist":"A \u0026 B","title":"Sparse song"}`, knownArtist174)
	oldItemDigest := ingest.Digest([]byte(legacy))
	oldBatchDigest := ingest.Digest(fmt.Appendf(nil, `[%q]`, oldItemDigest))
	for _, version := range []string{ingest.LegacyDigestVersion, ingest.DigestVersionV2} {
		t.Run(version, func(t *testing.T) {
			execSQL(t, s, `INSERT OR IGNORE INTO integrations(id,profile_id,source_type,created_at) VALUES('listenbrainz:1',1,'listenbrainz',0)`)
			execSQL(t, s, `INSERT INTO ingest_events(receipt_id,profile_id,source_instance_id,schema_version,source_type,adapter_version,batch_id,item_ordinal,
				operation,received_at,time_precision,time_semantics,original_snapshot,snapshot_digest,digest_version,client_idempotency_key,batch_digest,mode,resolver_variant,state,duplicate_decision,identity_status,closed_at,event_id,occurrence_id,recording_id,resolution_version)
				VALUES(?,1,'listenbrainz:1',1,'listenbrainz','1',?,0,'single',0,'estimated','unknown',?,?,?,?,?,'active','occurrence','duplicate','same','resolved',1,
				(SELECT event_id FROM listens WHERE rowid=10),1,(SELECT recording_id FROM occurrences WHERE id=1),1)`,
				version, version, legacy, oldItemDigest, version, version, oldBatchDigest)
			batch := ingest.Batch{ProfileID: 1, Source: ingest.ShadowSourceListenBrainz, Operation: "single", ClientKey: version, ReceivedAt: time.Now(),
				Items: []ingest.Item{{Snapshot: snapshot}}}
			got, err := s.AdmitOccurrence(ctx, batch)
			require.NoError(t, err)
			require.Len(t, got, 1)
			require.True(t, got[0].Replay)
			require.Equal(t, version, got[0].ReceiptID)
			require.Zero(t, scalar(t, s, `SELECT count(*) FROM ingest_observations WHERE receipt_id=?`, version))
			// Legacy snapshots lack the missing slot and must never pick a credit.
			var old ingest.Snapshot
			require.NoError(t, json.Unmarshal([]byte(legacy), &old))
			require.Empty(t, enrichment.MissingArtistNames(old))
			require.Empty(t, fileArtistDeclarations(old))
		})
	}
	batch := ingest.Batch{ProfileID: 1, Source: ingest.ShadowSourceListenBrainz, Operation: "single", ClientKey: "positional", ReceivedAt: time.Now(),
		Items: []ingest.Item{{Snapshot: snapshot}}}
	first, err := s.AdmitOccurrence(ctx, batch)
	require.NoError(t, err)
	require.Len(t, first, 1)
	batch.Items[0].Snapshot.ArtistMBIDs = []string{"", knownArtist174}
	replay, err := s.AdmitOccurrence(ctx, batch)
	require.NoError(t, err)
	require.Len(t, replay, 1)
	require.True(t, replay[0].Replay)
	require.Equal(t, first[0].ReceiptID, replay[0].ReceiptID)
	require.Equal(t, 1, scalar(t, s, `SELECT count(*) FROM ingest_observations WHERE receipt_id=? AND digest_version=?`, first[0].ReceiptID, ingest.DigestVersion))
}
