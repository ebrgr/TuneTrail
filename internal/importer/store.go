package importer

import (
	"context"
	"github.com/ebrgr/TuneTrail/internal/db"
	"github.com/ebrgr/TuneTrail/internal/ingest"
)

// importStore is the minimal interface required by all importer functions.
type Store interface {
	AdmitOccurrence(context.Context, ingest.Batch) ([]ingest.Result, error)
	db.ArtistStore
	db.AlbumStore
	db.TrackStore
	db.ListenStore
}

// importStore preserves the existing parser signatures.
type importStore = Store
