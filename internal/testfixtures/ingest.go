package testfixtures

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/ebrgr/TuneTrail/internal/db"
	"github.com/ebrgr/TuneTrail/internal/ingest"
	"github.com/ebrgr/TuneTrail/internal/mbz"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// OccurrenceStore exposes the production admission and resolver used by history fixtures.
type OccurrenceStore interface {
	AdmitOccurrence(context.Context, ingest.Batch) ([]ingest.Result, error)
	db.TrackStore
	db.AlbumStore
	Exec(string, ...any) error
	Count(string, ...any) (int, error)
	ResolveNextIngest(context.Context, mbz.MusicBrainzCaller) (bool, error)
}

// OccurrenceListen seeds one accepted event using the production occurrence resolver.
func OccurrenceListen(t testing.TB, store OccurrenceStore, opts db.SaveListenOpts) {
	t.Helper()
	ctx := t.Context()
	profile, at := opts.ProfileID, opts.Time
	require.Positive(t, profile)
	require.False(t, at.IsZero())
	track, err := store.GetTrack(ctx, db.GetTrackOpts{ID: opts.TrackID, AllowUnplayed: true})
	require.NoError(t, err)
	snapshot := ingest.Snapshot{Title: track.Title, Client: opts.Client, DurationMS: int64(track.Duration) * 1000}
	for _, artist := range track.Artists {
		snapshot.ArtistNames = append(snapshot.ArtistNames, artist.Name)
	}
	require.NotEmpty(t, snapshot.ArtistNames)
	snapshot.Artist = snapshot.ArtistNames[0]
	if track.MbzID != nil {
		snapshot.Recording = track.MbzID.String()
	}
	if track.AlbumID != 0 {
		album, err := store.GetAlbum(ctx, db.GetAlbumOpts{ID: track.AlbumID})
		require.NoError(t, err)
		snapshot.Release = album.Title
	}
	batch := ingest.Batch{ProfileID: profile, Source: ingest.ShadowSourceListenBrainz, Operation: "single", ReceivedAt: time.Now(),
		Items: []ingest.Item{{ListenedAt: at, Snapshot: snapshot, ExternalNamespace: "fixture", ExternalID: uuid.NewString()}}}
	results, err := store.AdmitOccurrence(ctx, batch)
	require.NoError(t, err)
	require.Len(t, results, 1)
	resolved, err := store.ResolveNextIngest(ctx, nil)
	require.NoError(t, err)
	require.True(t, resolved)
	count, err := store.Count(`SELECT count(*) FROM ingest_events e JOIN listens l USING(profile_id,event_id)
		WHERE e.receipt_id=? AND e.state='accepted' AND l.track_id=? AND l.listened_at=?`, results[0].ReceiptID, opts.TrackID, at.Unix())
	require.NoError(t, err)
	require.Equal(t, 1, count, fmt.Sprintf("accepted fixture event must retain occurrence %d", opts.TrackID))
}
