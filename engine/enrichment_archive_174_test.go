package engine

import (
	"encoding/json"
	"testing"

	"github.com/ebrgr/TuneTrail/internal/export"
	"github.com/stretchr/testify/require"
)

// TestEnrichmentArchiveRoundTrip preserves scoped artist and recording declarations through JSON.
func TestEnrichmentArchiveRoundTrip(t *testing.T) {
	a := newProfileTestApp(t)
	require.NoError(t, a.store.Exec(`INSERT INTO artists(id) VALUES(1);
		INSERT INTO artist_aliases VALUES(1,'Artist','Canonical',1);
		INSERT INTO releases(id) VALUES(1);
		INSERT INTO release_aliases VALUES(1,'Album','Canonical',1);
		INSERT INTO tracks(id,release_id,duration) VALUES(1,1,180);
		INSERT INTO track_aliases VALUES(1,'Song','Canonical',1);
		INSERT INTO artist_tracks VALUES(1,1,1);
		INSERT INTO listens(profile_id,track_id,listened_at,event_id) VALUES(1,1,1700000000,'enriched-event'),(2,1,1700000001,'private-other');
		INSERT INTO metadata_values(profile_id,recording_id,occurrence_id,event_id,instance,value,source,observed_at,kind) SELECT 1,recording_id,id,'enriched-event','server-a','11111111-1111-4111-8111-111111111111','navidrome-subsonic','2026-01-02T00:00:00Z','mbid' FROM occurrences WHERE id=1;
		INSERT INTO metadata_values(profile_id,artist_id,event_id,instance,value,source,observed_at,kind) VALUES(1,1,'enriched-event','server-a','22222222-2222-4222-8222-222222222222','navidrome-subsonic','2026-01-02T00:00:00Z','mbid')`))
	backup := a.request("GET", "/apis/web/v1/profiles/default/export", "", "", true)
	require.Equal(t, 200, backup.Code, backup.Body.String())
	require.NotContains(t, backup.Body.String(), "server-a")
	var doc export.KoitoExport
	require.NoError(t, json.Unmarshal(backup.Body.Bytes(), &doc))
	require.Len(t, doc.Listens, 1)
	require.Nil(t, doc.Listens[0].Track.MBID)
	require.Len(t, doc.Listens[0].Track.MBIDDeclarations, 1)
	require.Equal(t, "navidrome-subsonic", doc.Listens[0].Track.MBIDDeclarations[0].Source)
	require.Nil(t, doc.Listens[0].Artists[0].MBID)
	require.Len(t, doc.Listens[0].Artists[0].MBIDDeclarations, 1)
	require.Equal(t, "1", doc.Listens[0].Artists[0].MBIDDeclarations[0].Instance)
	require.Equal(t, "navidrome-subsonic", doc.Listens[0].Artists[0].MBIDDeclarations[0].Source)
	require.Equal(t, "2026-01-02T00:00:00Z", doc.Listens[0].Artists[0].MBIDDeclarations[0].DeclaredAt)
	require.Equal(t, "22222222-2222-4222-8222-222222222222", doc.Listens[0].Artists[0].MBIDDeclarations[0].MBID.String())
	other := a.request("GET", "/apis/web/v1/profiles/second/export", "", "", true)
	var otherDoc export.KoitoExport
	require.NoError(t, json.Unmarshal(other.Body.Bytes(), &otherDoc))
	require.Nil(t, otherDoc.Listens[0].Artists[0].MBID)
	require.Empty(t, otherDoc.Listens[0].Artists[0].MBIDDeclarations)
	legacy := a.request("GET", "/apis/web/v1/profiles/default/export?version=1", "", "", true)
	var legacyDoc export.KoitoExport
	require.NoError(t, json.Unmarshal(legacy.Body.Bytes(), &legacyDoc))
	require.Nil(t, legacyDoc.Listens[0].Artists[0].MBID)
	require.Empty(t, legacyDoc.Listens[0].Artists[0].MBIDDeclarations)
	require.Nil(t, legacyDoc.Listens[0].Track.MBID)
	require.Equal(t, 204, a.request("DELETE", "/apis/web/v1/profiles/default/listens?event_id=enriched-event", "", "", true).Code)
	upload := a.upload(t, "POST", "/apis/web/v1/profiles/default/import", map[string]string{"format": "tunetrail"}, "file", "history.json", backup.Body.Bytes(), true)
	require.Equal(t, 204, upload.Code, upload.Body.String())
	var count int
	require.NoError(t, a.store.QueryRow(`SELECT count(*) FROM metadata_values WHERE artist_id IS NOT NULL AND profile_id=1 AND value='22222222-2222-4222-8222-222222222222' AND source='navidrome-subsonic' AND observed_at='2026-01-02T00:00:00Z'`).Scan(&count))
	require.Equal(t, 1, count)
	require.NoError(t, a.store.QueryRow(`SELECT count(*) FROM artists WHERE musicbrainz_id='22222222-2222-4222-8222-222222222222'`).Scan(&count))
	require.Zero(t, count)
	require.NoError(t, a.store.QueryRow(`SELECT count(*) FROM recordings WHERE musicbrainz_id='11111111-1111-4111-8111-111111111111'`).Scan(&count))
	require.Zero(t, count)
	require.NoError(t, a.store.QueryRow(`SELECT count(*) FROM metadata_values WHERE recording_id IS NOT NULL AND profile_id=1 AND value='11111111-1111-4111-8111-111111111111'`).Scan(&count))
	require.Equal(t, 1, count)
}

// TestEnrichmentStandaloneArchive retains a linked artist claim without a release.
func TestEnrichmentStandaloneArchive(t *testing.T) {
	a := newProfileTestApp(t)
	require.NoError(t, a.store.Exec(`INSERT INTO artists(id) VALUES(1);
		INSERT INTO artist_aliases VALUES(1,'Standalone artist','Canonical',1);
		INSERT INTO recordings(id) VALUES(1);
		INSERT INTO occurrences(id,recording_id,presented_title) VALUES(1,1,'Standalone song');
		INSERT INTO occurrence_credits VALUES(1,0,'Standalone artist',1);
		INSERT INTO listens(profile_id,track_id,listened_at,event_id) VALUES(1,1,1700000000,'standalone-enriched');
		INSERT INTO metadata_values(profile_id,artist_id,event_id,instance,value,source,observed_at,kind) VALUES
		(1,1,'standalone-enriched','server','22222222-2222-4222-8222-222222222222','navidrome-subsonic','2026-01-02T00:00:00Z','mbid'),
		(1,1,'standalone-enriched','other','33333333-3333-4333-8333-333333333333','navidrome-subsonic','2026-01-03T00:00:00Z','mbid')`))
	backup := a.request("GET", "/apis/web/v1/profiles/default/export", "", "", true)
	require.Equal(t, 200, backup.Code)
	var doc export.KoitoExport
	require.NoError(t, json.Unmarshal(backup.Body.Bytes(), &doc))
	require.Nil(t, doc.Listens[0].Artists[0].MBID)
	require.Len(t, doc.Listens[0].Artists[0].MBIDDeclarations, 2)
	require.Equal(t, 204, a.request("DELETE", "/apis/web/v1/profiles/default/listens?event_id=standalone-enriched", "", "", true).Code)
	w := a.upload(t, "POST", "/apis/web/v1/profiles/default/import", map[string]string{"format": "tunetrail"}, "file", "history.json", backup.Body.Bytes(), true)
	require.Equal(t, 204, w.Code, w.Body.String())
	var count int
	require.NoError(t, a.store.QueryRow(`SELECT count(*) FROM metadata_values WHERE artist_id IS NOT NULL AND profile_id=1 AND value='22222222-2222-4222-8222-222222222222'`).Scan(&count))
	require.Equal(t, 1, count)
	require.NoError(t, a.store.QueryRow(`SELECT count(*) FROM metadata_values WHERE artist_id IS NOT NULL AND profile_id=1`).Scan(&count))
	require.Equal(t, 2, count)
}
