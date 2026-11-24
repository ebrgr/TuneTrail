package sqlite

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ebrgr/TuneTrail/internal/db"
	"github.com/ebrgr/TuneTrail/internal/enrichment"
	"github.com/ebrgr/TuneTrail/internal/ingest"
	"github.com/ebrgr/TuneTrail/internal/mbz"
	"github.com/ebrgr/TuneTrail/internal/testfixtures"
	"github.com/stretchr/testify/require"
)

type lookup175Transport func(*http.Request) (*http.Response, error)

func (fn lookup175Transport) RoundTrip(r *http.Request) (*http.Response, error) { return fn(r) }

// listenBrainz175Fixture replaces only the transport so tests verify the original destination.
func listenBrainz175Fixture(t *testing.T, handler http.HandlerFunc) *enrichment.ListenBrainz {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	target, err := url.Parse(server.URL)
	require.NoError(t, err)
	return enrichment.NewListenBrainz("TuneTrail/test", lookup175Transport(func(r *http.Request) (*http.Response, error) {
		if r.Method != http.MethodGet || r.URL.Scheme != "https" || r.URL.Host != "api.listenbrainz.org" || r.URL.Path != "/1/metadata/lookup/" {
			return nil, fmt.Errorf("unexpected lookup destination")
		}
		copy := r.Clone(r.Context())
		copy.URL.Scheme, copy.URL.Host = target.Scheme, target.Host
		return server.Client().Transport.RoundTrip(copy)
	}))
}

// approveListenBrainz175 seeds an approved credential without DNS or an external account.
func approveListenBrainz175(t *testing.T, s *Sqlite, profile int32, base, token string) {
	t.Helper()
	require.NoError(t, s.SetProfileRelay(t.Context(), profile, base, token))
	execSQL(t, s, `UPDATE relay_destinations SET awaiting_approval=0 WHERE profile_id=?`, profile)
}

// TestListenBrainzEnrichmentPrecedence verifies event, catalog and Subsonic precedence per field.
func TestListenBrainzEnrichmentPrecedence(t *testing.T) {
	for _, source := range []string{"received", "catalog", "subsonic", "fallback only"} {
		t.Run(source, func(t *testing.T) {
			s := convertedOccurrenceFixture(t)
			require.NoError(t, s.ConfigureEnrichment(t.Context(), source == "subsonic"))
			approveListenBrainz175(t, s, 1, enrichment.ListenBrainzBase, "profile-one")
			snapshot := ingest.Snapshot{Title: "Fallback song", Artist: "A & B", ArtistNames: []string{"A", "B"}, Release: "Fallback album"}
			if source == "received" {
				snapshot.Recording = fixtureRecording
				snapshot.ArtistMBIDs = []string{knownArtist174, ""}
			}
			id := admitOccurrenceFixture(t, s, snapshot, 1700000750, "fallback")
			resolveOccurrenceFixture(t, s, &mbz.MbzErrorCaller{})
			requireReceiptState(t, s, id, "accepted")
			if source == "fallback only" {
				require.NoError(t, s.ConfigureEnrichment(t.Context(), false))
			}
			if source == "catalog" {
				execSQL(t, s, `UPDATE recordings SET musicbrainz_id=? WHERE id=(SELECT recording_id FROM ingest_events WHERE receipt_id=?)`, fixtureRecording, id)
				execSQL(t, s, `UPDATE artists SET musicbrainz_id=? WHERE id=(SELECT artist_id FROM artist_aliases WHERE alias='A')`, knownArtist174)
			}
			var primary enrichment.Provider
			if source == "subsonic" {
				primary = enrichmentFixture{lookup: func(context.Context, enrichment.Request) enrichment.Result {
					at := time.Now()
					return enrichment.Result{Status: enrichment.Found, Declarations: []enrichment.Declaration{
						{Entity: "recording", Field: "musicbrainz_id", Value: fixtureRecording, Source: "navidrome-subsonic", ObservedAt: at},
						{Entity: "artist", Name: "A", Field: "musicbrainz_id", Value: knownArtist174, Source: "navidrome-subsonic", ObservedAt: at},
					}}
				}}
			}
			var calls atomic.Int32
			client := listenBrainz175Fixture(t, func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				if r.Header.Get("Authorization") != "Token profile-one" || r.URL.Query().Get("artist_name") != "A & B" {
					t.Error("credential or artist credit mismatch")
				}
				_, _ = fmt.Fprintf(w, `{"recording_mbid":%q,"recording_name":"Fallback song","artist_credit_name":"A & B","release_name":"Fallback album",
					"artist_mbids":[%q,%q],"metadata":{"artist":{"artists":[{"name":"B","artist_mbid":%q},{"name":"A","artist_mbid":%q}]}}}`,
					fixtureRecording, knownArtist174, enrichedArtist174, enrichedArtist174, knownArtist174)
			})
			found, err := s.ResolveNextEnrichmentWithFallback(t.Context(), primary, client)
			require.NoError(t, err)
			require.True(t, found)
			require.EqualValues(t, 1, calls.Load())
			want := 1
			if source == "fallback only" {
				want = 3
			}
			require.Equal(t, want, scalar(t, s, `SELECT count(*) FROM metadata_values m JOIN ingest_events e
				ON (m.profile_id,m.event_id)=(e.profile_id,e.event_id) WHERE e.receipt_id=? AND m.source=? AND m.observed_at<>''`, id, enrichment.ListenBrainzSource))
			require.Equal(t, 1, scalar(t, s, `SELECT count(*) FROM metadata_values WHERE source=? AND value=? AND artist_id IS NOT NULL`, enrichment.ListenBrainzSource, enrichedArtist174))
			if source == "subsonic" {
				require.Equal(t, 2, scalar(t, s, `SELECT count(*) FROM metadata_values WHERE source='navidrome-subsonic'`))
			}
			if source == "fallback only" {
				checkListenBrainz175Archive(t, s, id)
			}
		})
	}
}

// checkListenBrainz175Archive verifies that export and restore preserve claims and never promote them.
func checkListenBrainz175Archive(t *testing.T, s *Sqlite, receipt string) {
	t.Helper()
	var event string
	var occurrence, artistID int32
	require.NoError(t, s.db.QueryRow(`SELECT event_id,occurrence_id FROM ingest_events WHERE receipt_id=?`, receipt).Scan(&event, &occurrence))
	require.NoError(t, s.db.QueryRow(`SELECT artist_id FROM metadata_values WHERE event_id=? AND artist_id IS NOT NULL AND value=?`, event, enrichedArtist174).Scan(&artistID))
	execSQL(t, s, `INSERT INTO listens(profile_id,track_id,listened_at,origin) VALUES(2,?,1700000760,'legacy')`, occurrence)
	for _, profile := range []int32{1, 2} {
		ctx := db.WithProfile(t.Context(), profile)
		track, err := s.GetTrack(ctx, db.GetTrackOpts{ID: occurrence})
		require.NoError(t, err)
		artist, err := s.GetArtist(ctx, db.GetArtistOpts{ID: artistID})
		require.NoError(t, err)
		if profile == 1 {
			require.NotNil(t, track.MbzID)
			require.Equal(t, fixtureRecording, track.MbzID.String())
			require.NotNil(t, artist.MbzID)
			require.Equal(t, enrichedArtist174, artist.MbzID.String())
		} else {
			require.Nil(t, track.MbzID)
			require.Nil(t, artist.MbzID)
		}
	}
	items, err := s.GetExportPage(t.Context(), db.GetExportPageOpts{ProfileID: 1, Limit: 100})
	require.NoError(t, err)
	var item *db.ExportItem
	for _, row := range items {
		if row.Cursor == event {
			item = row
		}
	}
	require.NotNil(t, item)
	require.Equal(t, enrichment.ListenBrainzSource, item.MBIDSource)
	require.Len(t, item.RecordingDeclarations, 1)
	require.Equal(t, "1", item.RecordingDeclarations[0].Instance)
	require.NotEmpty(t, item.MBIDDeclaredAt)
	for _, artist := range item.Artists {
		require.Len(t, artist.MBIDDeclarations, 1)
		require.Equal(t, enrichment.ListenBrainzSource, artist.MBIDSource)
	}
	target, _, _ := migrationDB(t, currentSchemaVersion)
	testfixtures.Profile(t, target, 2, "restore", "Restore", 1)
	count, err := target.RestoreArchive(t.Context(), 2, []*db.ExportItem{item})
	require.NoError(t, err)
	require.Equal(t, 1, count)
	require.Equal(t, 3, scalar(t, target, `SELECT count(*) FROM metadata_values m JOIN listens l
		ON (m.event_id,m.profile_id)=(l.event_id,l.profile_id) WHERE m.profile_id=2 AND m.source=? AND m.observed_at=?`, enrichment.ListenBrainzSource, item.MBIDDeclaredAt))
	require.Zero(t, scalar(t, target, `SELECT count(*) FROM recordings WHERE musicbrainz_id=?`, fixtureRecording))
	require.Zero(t, scalar(t, target, `SELECT count(*) FROM artists WHERE musicbrainz_id IN (?,?)`, knownArtist174, enrichedArtist174))
}

// TestListenBrainzEnrichmentAuthorization tests the receipt's profile and approval generation.
func TestListenBrainzEnrichmentAuthorization(t *testing.T) {
	for _, scenario := range []string{"no configured providers", "other profile", "custom relay", "draft", "new approval", "removed", "paused", "upstream failure", "wrong release", "empty result"} {
		t.Run(scenario, func(t *testing.T) {
			s := configuredEnrichmentFixture(t)
			switch scenario {
			case "other profile":
				approveListenBrainz175(t, s, 2, enrichment.ListenBrainzBase, "profile-two")
			case "custom relay":
				approveListenBrainz175(t, s, 1, "https://relay.example/1", "custom-token")
			case "draft":
				require.NoError(t, s.SetProfileRelay(t.Context(), 1, enrichment.ListenBrainzBase, "draft-token"))
			case "new approval":
			case "no configured providers":
				require.NoError(t, s.ConfigureEnrichment(t.Context(), false))
			default:
				approveListenBrainz175(t, s, 1, enrichment.ListenBrainzBase, "profile-one")
			}
			id := admitOccurrenceFixture(t, s, ingest.Snapshot{Title: "Title", Artist: "Artist", Release: "Album"}, 1700000751, "authority")
			if scenario == "new approval" {
				approveListenBrainz175(t, s, 1, enrichment.ListenBrainzBase, "new-token")
			} else if scenario == "removed" {
				require.NoError(t, s.SetProfileRelay(t.Context(), 1, "", ""))
			} else if scenario == "paused" {
				execSQL(t, s, `UPDATE relay_destinations SET pause_class='auth' WHERE profile_id=1`)
			}
			resolveOccurrenceFixture(t, s, &mbz.MbzErrorCaller{})
			if scenario == "no configured providers" {
				require.Zero(t, scalar(t, s, `SELECT count(*) FROM enrichment_jobs WHERE receipt_id=?`, id))
				return
			}
			var calls atomic.Int32
			client := listenBrainz175Fixture(t, func(w http.ResponseWriter, _ *http.Request) {
				calls.Add(1)
				switch scenario {
				case "wrong release":
					_, _ = fmt.Fprintf(w, `{"recording_mbid":%q,"recording_name":"Title","artist_credit_name":"Artist","release_name":"Another Album"}`, fixtureRecording)
				case "empty result":
					_, _ = fmt.Fprint(w, "{}")
				default:
					w.WriteHeader(http.StatusServiceUnavailable)
				}
			})
			found, err := s.ResolveNextEnrichmentWithFallback(t.Context(), nil, client)
			require.NoError(t, err)
			require.True(t, found)
			state := "disabled"
			switch scenario {
			case "upstream failure":
				state = "failed"
			case "wrong release":
				state = "ambiguous"
			case "empty result":
				state = "absent"
			}
			if state == "disabled" {
				require.Zero(t, calls.Load())
			} else {
				require.EqualValues(t, 1, calls.Load())
			}
			requireReceiptState(t, s, id, "accepted")
			require.Equal(t, 1, scalar(t, s, `SELECT count(*) FROM enrichment_jobs WHERE receipt_id=? AND state=? AND attempts=1 AND lease IS NULL`, id, state))
			require.Zero(t, scalar(t, s, `SELECT count(*) FROM metadata_values`))
			found, err = s.ResolveNextEnrichmentWithFallback(t.Context(), nil, client)
			require.NoError(t, err)
			require.False(t, found)
		})
	}
}

// TestListenBrainzEnrichmentConcurrentEvidence changes scope while the socket is open.
func TestListenBrainzEnrichmentConcurrentEvidence(t *testing.T) {
	for _, change := range []string{"scoped claim", "relay rotation"} {
		t.Run(change, func(t *testing.T) {
			s := configuredEnrichmentFixture(t)
			approveListenBrainz175(t, s, 1, enrichment.ListenBrainzBase, "profile-one")
			id := admitOccurrenceFixture(t, s, ingest.Snapshot{Title: "Title", Artist: "Artist", Release: "Album"}, 1700000752, "race")
			resolveOccurrenceFixture(t, s, &mbz.MbzErrorCaller{})
			started, release := make(chan struct{}), make(chan struct{})
			client := listenBrainz175Fixture(t, func(w http.ResponseWriter, r *http.Request) {
				close(started)
				select {
				case <-release:
				case <-r.Context().Done():
					return
				}
				_, _ = fmt.Fprintf(w, `{"recording_mbid":%q,"recording_name":"Title","artist_credit_name":"Artist","release_name":"Album"}`, fixtureRecording)
			})
			done := make(chan error, 1)
			go func() {
				_, err := s.ResolveNextEnrichmentWithFallback(t.Context(), nil, client)
				done <- err
			}()
			select {
			case <-started:
			case err := <-done:
				t.Fatalf("lookup stopped before socket opened: %v", err)
			case <-time.After(2 * time.Second):
				t.Fatal("lookup did not start")
			}
			if change == "relay rotation" {
				approveListenBrainz175(t, s, 1, enrichment.ListenBrainzBase, "rotated-token")
			} else {
				execSQL(t, s, `INSERT INTO metadata_values(profile_id,recording_id,occurrence_id,event_id,kind,source,instance,value,observed_at)
					SELECT profile_id,recording_id,occurrence_id,event_id,'mbid','navidrome-subsonic','race-library',?,'2026-01-02T00:00:00Z'
					FROM ingest_events WHERE receipt_id=?`, knownArtist174, id)
			}
			close(release)
			require.NoError(t, <-done)
			require.Zero(t, scalar(t, s, `SELECT count(*) FROM metadata_values WHERE source=?`, enrichment.ListenBrainzSource))
			require.Equal(t, 1, scalar(t, s, `SELECT count(*) FROM enrichment_jobs WHERE receipt_id=? AND state='ambiguous' AND lease IS NULL`, id))
			requireReceiptState(t, s, id, "accepted")
		})
	}
}
