package importer

import (
	"context"
	"fmt"
	"io"
	"os"
	"path"

	"github.com/ebrgr/TuneTrail/internal/cfg"
	"github.com/ebrgr/TuneTrail/internal/db"
	"github.com/ebrgr/TuneTrail/internal/mbz"
	"github.com/google/uuid"
)

// Result distinguishes newly stored listens from duplicates and skipped entries.
type Result struct {
	IgnoredMBIDConflicts                             int  `json:",omitempty"`
	RejectedOversizedItems                           int  `json:",omitempty"`
	TruncatedFields                                  int  `json:",omitempty"`
	IgnoredAliasLimit                                int  `json:",omitempty"`
	IgnoredAliasConflicts                            int  `json:",omitempty"`
	Active                                           bool `json:",omitempty"`
	Processed, Imported, Duplicates, Ignored, Errors int
	Pending                                          int // Active receipts awaiting a decision, or shadow reviews in legacy mode.
}
type resultKey struct{}

// seen advances the absolute file ordinal before validation or filtering.
func seen(ctx context.Context) {
	if shadow, ok := ctx.Value(shadowKey{}).(*fileShadow); ok {
		shadow.ordinal++
	}
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
}
type countingStore struct {
	UploadStore
	result *Result
}

// RestoreArchive forwards the atomic restore and counts idempotent events.
func (s countingStore) RestoreArchive(ctx context.Context, profile int32, items []*db.ExportItem) (int, error) {
	store, ok := s.UploadStore.(archiveRestorer)
	if !ok {
		return 0, fmt.Errorf("archive restore is unavailable")
	}
	count, err := store.RestoreArchive(ctx, profile, items)
	if err == nil {
		s.result.Imported += count
		s.result.Duplicates += len(items) - count
	}
	return count, err
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
