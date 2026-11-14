package importer

import (
	"context"
	"fmt"
	"github.com/ebrgr/TuneTrail/internal/cfg"
	"io"
	"os"
	"path"

	"github.com/ebrgr/TuneTrail/internal/db"
	"github.com/ebrgr/TuneTrail/internal/mbz"
	"github.com/google/uuid"
)

// Result distinguishes newly stored listens from duplicates and skipped entries.
type Result struct{ Processed, Imported, Duplicates, Ignored, Errors int }
type resultKey struct{}

func seen(ctx context.Context) {
	if r, ok := ctx.Value(resultKey{}).(*Result); ok {
		r.Processed++
	}
}
func ignored(ctx context.Context) {
	if r, ok := ctx.Value(resultKey{}).(*Result); ok {
		r.Ignored++
	}
}

type UploadStore interface {
	Store
	SaveListenWithResult(context.Context, db.SaveListenOpts) (bool, error)
}
type countingStore struct {
	UploadStore
	result *Result
}

func (s countingStore) SaveListen(ctx context.Context, o db.SaveListenOpts) error {
	inserted, err := s.UploadStore.SaveListenWithResult(ctx, o)
	if err != nil {
		return err
	}
	if inserted {
		s.result.Imported++
	} else {
		s.result.Duplicates++
	}
	return nil
}

var suffixes = map[string]string{"tunetrail": "tunetrail.json", "koito": "koito.json", "spotify": "Streaming_History_Audio.json", "maloja": "maloja.json", "lastfm": "recenttracks.json", "listenbrainz": "listenbrainz.zip"}

// Upload executes synchronously against the selected profile; staged files are never boot inputs.
func Upload(ctx context.Context, store UploadStore, mbzc mbz.MusicBrainzCaller, profileID int32, format, stagedPath string) (result Result, err error) {
	suffix, ok := suffixes[format]
	if !ok || profileID <= 0 || stagedPath == "" {
		result.Errors++
		return result, db.ErrInvalidInput
	}
	source, e := os.Open(stagedPath)
	if e != nil {
		result.Errors++
		return result, e
	}
	defer source.Close()
	dir := path.Join(cfg.ConfigDir(), "import_staging")
	if e = os.MkdirAll(dir, 0700); e != nil {
		result.Errors++
		return result, e
	}
	info, e := os.Lstat(dir)
	if e != nil || !info.IsDir() {
		result.Errors++
		return result, fmt.Errorf("invalid staging directory")
	}
	stage, e := os.CreateTemp(dir, "upload-*")
	if e != nil {
		result.Errors++
		return result, e
	}
	stagePath := stage.Name()
	defer os.Remove(stagePath)
	_, e = io.Copy(stage, &contextReader{ctx: ctx, reader: io.LimitReader(source, (100<<20)+1)})
	info, statErr := stage.Stat()
	closeErr := stage.Close()
	if e != nil {
		result.Errors++
		return result, e
	}
	if statErr != nil {
		result.Errors++
		return result, statErr
	}
	if closeErr != nil {
		result.Errors++
		return result, closeErr
	}
	if info.Size() > 100<<20 {
		result.Errors++
		return result, db.ErrInvalidInput
	}
	filename := path.Join(fmt.Sprint(profileID), uuid.NewString()+"-"+suffix)
	ctx = db.WithProfile(WithUploadSource(context.WithValue(ctx, resultKey{}, &result), stagePath), profileID)
	tracked := countingStore{store, &result}
	switch format {
	case "tunetrail", "koito": // same JSON schema (ISSUE-111)
		err = ImportKoitoFile(ctx, tracked, filename)
	case "spotify":
		err = ImportSpotifyFile(ctx, tracked, mbzc, filename)
	case "maloja":
		err = ImportMalojaFile(ctx, tracked, mbzc, filename)
	case "lastfm":
		err = ImportLastFMFile(ctx, tracked, mbzc, filename)
	case "listenbrainz":
		err = ImportListenBrainzExport(ctx, tracked, mbzc, filename)
	}
	if err != nil {
		result.Errors++
	}
	return result, err
}
