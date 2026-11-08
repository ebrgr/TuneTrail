package importer

import "github.com/ebrgr/TuneTrail/internal/db"

// importStore is the minimal interface required by all importer functions.
type Store interface {
	db.ArtistStore
	db.AlbumStore
	db.TrackStore
	db.ListenStore
}

// importStore preserves the existing parser signatures.
type importStore = Store
