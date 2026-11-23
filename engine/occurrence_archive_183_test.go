package engine

import (
	"encoding/json"
	"fmt"
	"sort"
	"testing"

	"github.com/ebrgr/TuneTrail/internal/export"
	"github.com/stretchr/testify/require"
)

// TestOccurrenceHTTPCompatibility183 versions the ISSUE-133 AC3 comparison as a repeatable HTTP test.
func TestOccurrenceHTTPCompatibility183(t *testing.T) {
	a := newProfileTestApp(t)
	for _, statement := range []string{
		`INSERT INTO artists(id) VALUES(1);
		  INSERT INTO artist_aliases VALUES(1,'Artist','Canonical',1);
		  INSERT INTO releases(id) VALUES(1),(2);
		  INSERT INTO release_aliases VALUES(1,'Album','Canonical',1),(2,'Single','Canonical',1);
		  INSERT INTO artist_releases VALUES(1,1,1),(1,2,1);
		  INSERT INTO tracks(id,release_id,duration) VALUES(1,1,180),(2,2,180);
		  INSERT INTO artist_tracks VALUES(1,1,1),(1,2,1);
		  INSERT INTO track_aliases VALUES(1,'Intro','Canonical',1),(2,'Intro','Canonical',1)`,
		`INSERT INTO listens(profile_id,track_id,listened_at,event_id,client,duration_played)
		  VALUES(1,1,1700000000,'first','A',120),(2,1,1700000000,'other','B',0),(1,2,1700000000,'single','A',NULL)`,
	} {
		require.NoError(t, a.store.Exec(statement))
	}

	const prefix = "/apis/web/v1/profiles/"
	read := func(slug string) map[string]any {
		t.Helper()
		result := map[string]any{}
		for _, path := range []string{"stats?period=all_time", "listens?page=1", "listens?page=2",
			"top/artists", "top/albums", "top/tracks", "top/tracks?group_by=release", "search?q=Intro",
			"track/1", "track/1/aliases", "album/1", "export?version=1", "export"} {
			w := a.request("GET", prefix+slug+"/"+path, "", "", true)
			require.Equal(t, 200, w.Code, path+": "+w.Body.String())
			var payload any
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &payload))
			if object, ok := payload.(map[string]any); ok {
				delete(object, "exported_at")
				if period, ok := object["range"].(map[string]any); ok {
					delete(period, "to")
				}
				// The archive's event order is a cursor detail, while v1 entries have no ID.
				if path == "export?version=1" {
					entries := object["listens"].([]any)
					sort.Slice(entries, func(i, j int) bool { return fmt.Sprint(entries[i]) < fmt.Sprint(entries[j]) })
				}
			}
			result[path] = payload
		}
		return result
	}

	before := read("default")
	other := read("second")

	require.Equal(t, before, read("default"))
	require.Equal(t, other, read("second"))

	backup := a.request("GET", prefix+"default/export", "", "", true)
	require.Equal(t, 200, backup.Code)
	var doc export.KoitoExport
	require.NoError(t, json.Unmarshal(backup.Body.Bytes(), &doc))
	require.Equal(t, export.ArchiveVersion, doc.Version)
	require.Len(t, doc.Listens, 2)
	require.NotNil(t, doc.Listens[0].DurationPlayed)
	require.EqualValues(t, 120, *doc.Listens[0].DurationPlayed)
	require.Nil(t, doc.Listens[1].DurationPlayed)
	legacy := a.request("GET", prefix+"default/export?version=1", "", "", true)
	var old export.KoitoExport
	require.NoError(t, json.Unmarshal(legacy.Body.Bytes(), &old))
	require.Equal(t, "1", old.Version)
	require.Len(t, old.Listens, 2)
	require.Nil(t, old.Listens[0].Identity)

	merge := "/apis/web/v1/track/2/merge"
	const pair = `{"merge_from_id":1}`
	require.Equal(t, 204, a.request("POST", merge, pair, "", true).Code)
	require.Equal(t, 204, a.request("POST", merge+"/undo", pair, "", true).Code)
	require.Equal(t, before, read("default"))
	require.Equal(t, other, read("second"))

	deleted := a.request("DELETE", prefix+"default/listens?event_id=first", "", "", true)
	require.Equal(t, 204, deleted.Code, deleted.Body.String())
	require.Equal(t, other, read("second"))
	for range 2 {
		upload := a.upload(t, "POST", prefix+"default/import", map[string]string{"format": "tunetrail"}, "file", "history.json", backup.Body.Bytes(), true)
		require.Equal(t, 204, upload.Code, upload.Body.String())
	}
	require.Equal(t, before, read("default"))
	require.Equal(t, other, read("second"))
}

// TestOccurrenceArchiveLocalNamespaceScope checks the owner upload and the trusted root path.
func TestOccurrenceArchiveLocalNamespaceScope(t *testing.T) {
	a := newProfileTestApp(t)
	cookie, owner := owner160(t, a)
	const otherTrack = 2
	require.NoError(t, a.store.Exec(fmt.Sprintf(`
		INSERT INTO artists(id) VALUES(1);
		INSERT INTO artist_aliases VALUES(1,'Artist','Canonical',1);
		INSERT INTO releases(id) VALUES(1);
		INSERT INTO release_aliases VALUES(1,'Album','Canonical',1);
		INSERT INTO tracks(id,release_id,duration) VALUES(1,1,120),(2,1,130);
		INSERT INTO track_aliases VALUES(1,'Owner track','Canonical',1),(2,'Other track','Canonical',1);
		INSERT INTO artist_tracks VALUES(1,1,1),(1,2,1);
		INSERT INTO listens(profile_id,track_id,listened_at,event_id) VALUES(%d,1,1700000000,'owner-original'),
			(2,2,1700000000,'other-original')`, owner.ID)))

	actor := &profileTestApp{store: a.store, router: a.router, cookie: cookie}
	prefix := "/apis/web/v1/profiles/" + owner.Slug
	otherPage := "/" + owner.Slug + "/track/2"
	require.Equal(t, 404, actor.request("GET", otherPage, "", "", true).Code)
	out := actor.request("GET", prefix+"/export", "", "", true)
	require.Equal(t, 200, out.Code, out.Body.String())
	var doc export.KoitoExport
	require.NoError(t, json.Unmarshal(out.Body.Bytes(), &doc))
	require.Len(t, doc.Listens, 1)
	original := doc.Listens[0].Identity
	require.NotEmpty(t, original.CatalogNamespace)
	importArchive := func(who *profileTestApp, target string, id int32, event string) {
		t.Helper()
		original.OccurrenceID, original.EventID = id, event
		data, err := json.Marshal(doc)
		require.NoError(t, err)
		w := who.upload(t, "POST", target+"/import", map[string]string{"format": "tunetrail"}, "file", "history.json", data, true)
		require.Equal(t, 204, w.Code, w.Body.String())
	}
	importedTrack := func(profile int32, event string) int32 {
		t.Helper()
		var track int32
		require.NoError(t, a.store.QueryRow(`SELECT l.track_id FROM listens l JOIN archive_events e USING(profile_id,event_id)
			WHERE l.profile_id=? AND e.source_event_id=?`, profile, event).Scan(&track))
		return track
	}

	importArchive(actor, prefix, 1, "owner-known")
	require.EqualValues(t, 1, importedTrack(owner.ID, "owner-known"))
	importArchive(actor, prefix, otherTrack, "owner-edited")
	created := importedTrack(owner.ID, "owner-edited")
	require.Greater(t, created, int32(otherTrack))
	require.Equal(t, 404, actor.request("GET", otherPage, "", "", true).Code)
	require.Equal(t, 404, actor.request("GET", prefix+"/track/2", "", "", true).Code)

	// The existing profile mapping remains valid even if its first listen is removed.
	require.Equal(t, 204, actor.request("DELETE", prefix+"/listens?event_id=owner-edited", "", "", true).Code)
	importArchive(actor, prefix, otherTrack, "owner-mapped")
	require.Equal(t, created, importedTrack(owner.ID, "owner-mapped"))
	require.Equal(t, 404, actor.request("GET", otherPage, "", "", true).Code)

	importArchive(a, "/apis/web/v1/profiles/empty", otherTrack, "root-edited")
	require.EqualValues(t, otherTrack, importedTrack(3, "root-edited"))
}
