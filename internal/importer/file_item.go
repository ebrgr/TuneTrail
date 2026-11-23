package importer

import (
	"errors"
	"fmt"
	"time"
	"unicode/utf8"

	"github.com/ebrgr/TuneTrail/internal/export"
	"github.com/ebrgr/TuneTrail/internal/ingest"
)

// ErrRequiredTextTooLarge can be shown without exposing the imported text.
var ErrRequiredTextTooLarge = errors.New("An artist or title exceeds 4 KiB.")

var errFileItemEnvelopeTooLarge = errors.New("file item exceeds 10 KiB")

// prepareFileItem applies common file limits before allocating space to native aliases.
func prepareFileItem(ordinal int, snapshot ingest.Snapshot, at time.Time, native []export.KoitoListen) (ingest.Item, error) {
	if snapshot.Artist == "" || snapshot.Title == "" {
		return ingest.Item{}, invalidFile(fmt.Errorf("listen at ordinal %d requires an artist and title", ordinal))
	}
	if len(snapshot.Artist) > 4<<10 || len(snapshot.Title) > 4<<10 {
		return ingest.Item{}, fmt.Errorf("%w: listen at ordinal %d: %w", ErrInvalidFile, ordinal, ErrRequiredTextTooLarge)
	}
	shorten := func(value *string, limit int) {
		if len(*value) <= limit {
			return
		}
		for !utf8.RuneStart((*value)[limit]) {
			limit--
		}
		*value = (*value)[:limit]
		snapshot.TruncatedFields++
	}
	shorten(&snapshot.Client, 255)
	for _, value := range []*string{
		&snapshot.ClientVersion, &snapshot.Release, &snapshot.TrackMBID, &snapshot.MBIDSource,
		&snapshot.MBIDDeclaredAt, &snapshot.ReleaseGroupMBID, &snapshot.MappedRecording,
		&snapshot.ReleaseMBID, &snapshot.MappedReleaseMBID, &snapshot.Recording,
	} {
		shorten(value, 1<<10)
	}
	for _, values := range [][]string{snapshot.ArtistNames, snapshot.ArtistMBIDs, snapshot.MappedArtistMBIDs} {
		for i := range values {
			shorten(&values[i], 1<<10)
		}
	}
	for i := range snapshot.ArtistMbidMappings {
		shorten(&snapshot.ArtistMbidMappings[i].Name, 1<<10)
		shorten(&snapshot.ArtistMbidMappings[i].MBID, 1<<10)
	}
	// Omit oversized image sources rather than changing their transport policy marker.
	if len(snapshot.NativeReleaseImage) > 1<<10 {
		snapshot.NativeReleaseImage = ""
		snapshot.TruncatedFields++
	}
	for name, value := range snapshot.NativeArtistImages {
		if len(value) > 1<<10 {
			delete(snapshot.NativeArtistImages, name)
			snapshot.TruncatedFields++
		}
	}
	// Many bounded optional values can still exceed the total envelope budget.
	probe := ingest.Item{Snapshot: snapshot, ListenedAt: at}
	if encoded, _ := probe.Encode(); len(encoded) > ingest.MaxItemBytes {
		for _, field := range []*[]string{&snapshot.ArtistNames, &snapshot.ArtistMBIDs, &snapshot.MappedArtistMBIDs} {
			if len(*field) != 0 {
				*field = nil
				snapshot.TruncatedFields++
			}
		}
		if len(snapshot.NativeArtistImages) != 0 {
			snapshot.TruncatedFields += len(snapshot.NativeArtistImages)
			snapshot.NativeArtistImages = nil
		}
		if len(snapshot.ArtistMbidMappings) != 0 {
			snapshot.ArtistMbidMappings = nil
			snapshot.TruncatedFields++
		}
		for _, field := range []*string{&snapshot.ClientVersion, &snapshot.NativeReleaseImage, &snapshot.Release, &snapshot.Client} {
			probe.Snapshot = snapshot
			if encoded, _ := probe.Encode(); len(encoded) <= ingest.MaxItemBytes {
				break
			}
			if *field != "" {
				*field = ""
				snapshot.TruncatedFields++
			}
		}
	}
	if len(native) != 0 {
		snapshot = budgetNativeAliases(snapshot, native[0])
	}
	item := ingest.Item{Ordinal: ordinal, Snapshot: snapshot, ListenedAt: at}
	encoded, _ := item.Encode()
	if len(encoded) > ingest.MaxItemBytes {
		return ingest.Item{}, fmt.Errorf("listen at ordinal %d: %w", ordinal, errFileItemEnvelopeTooLarge)
	}
	return item, nil
}
