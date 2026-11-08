package engine

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/ebrgr/TuneTrail/internal/db"
	"github.com/ebrgr/TuneTrail/internal/export"
	"github.com/stretchr/testify/require"
)

type pauseExport struct {
	db.ExportStore
	first, resume chan struct{}
	pages         int
}

// GetExportPage pauses between pages after the first page has been read.
func (s *pauseExport) GetExportPage(ctx context.Context, o db.GetExportPageOpts) ([]*db.ExportItem, error) {
	rows, err := s.ExportStore.GetExportPage(ctx, o)
	s.pages++
	if s.pages == 1 && err == nil {
		close(s.first)
		<-s.resume
	}
	return rows, err
}

// TestDynamicExportUnderConcurrentIngestion documents cursor consistency without a snapshot.
func TestDynamicExportUnderConcurrentIngestion(t *testing.T) {
	a := newProfileTestApp(t)
	for _, q := range []string{
		`INSERT INTO artists(id) VALUES(1)`, `INSERT INTO artist_aliases VALUES(1,'Artist','fixture',1)`,
		`INSERT INTO releases(id) VALUES(1)`, `INSERT INTO release_aliases VALUES(1,'Album','fixture',1)`,
		`INSERT INTO artist_releases VALUES(1,1,1)`, `INSERT INTO tracks(id,release_id,duration) VALUES(1,1,180)`,
		`INSERT INTO track_aliases VALUES(1,'Track','fixture',1)`, `INSERT INTO artist_tracks VALUES(1,1,1)`,
		`WITH RECURSIVE n(x) AS (SELECT 1 UNION ALL SELECT x+1 FROM n WHERE x<1002) INSERT INTO listens(profile_id,track_id,listened_at) SELECT 1,1,x FROM n`,
	} {
		require.NoError(t, a.store.Exec(q))
	}
	p, err := a.store.GetListeningProfileByID(context.Background(), 1)
	require.NoError(t, err)
	wrapper := &pauseExport{ExportStore: a.store, first: make(chan struct{}), resume: make(chan struct{})}
	var out bytes.Buffer
	done := make(chan error, 1)
	go func() { done <- export.ExportData(context.Background(), p, wrapper, &out) }()
	select {
	case <-wrapper.first:
	case <-time.After(5 * time.Second):
		t.Fatal("export did not reach first page")
	}
	// One new row is behind the cursor and one is ahead; only the latter is visible.
	require.NoError(t, a.store.SaveListen(context.Background(), db.SaveListenOpts{ProfileID: 1, TrackID: 1, Time: time.Unix(0, 0)}))
	require.NoError(t, a.store.SaveListen(context.Background(), db.SaveListenOpts{ProfileID: 1, TrackID: 1, Time: time.Unix(1003, 0)}))
	close(wrapper.resume)
	select {
	case err := <-done:
		require.NoError(t, err)
	case <-time.After(5 * time.Second):
		t.Fatal("export did not complete")
	}
	var doc export.KoitoExport
	require.NoError(t, json.Unmarshal(out.Bytes(), &doc))
	require.Len(t, doc.Listens, 1003)
	require.Equal(t, int64(1), doc.Listens[0].ListenedAt.Unix())
	require.Equal(t, int64(1003), doc.Listens[len(doc.Listens)-1].ListenedAt.Unix())
	n, err := a.store.ProfileListenCount(context.Background(), 1)
	require.NoError(t, err)
	require.EqualValues(t, 1004, n)
}
