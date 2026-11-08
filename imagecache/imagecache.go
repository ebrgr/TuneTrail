// Package imagecache handles the downloading and cacheing of images.
package imagecache

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/ebrgr/TuneTrail/internal/cfg"
	"github.com/ebrgr/TuneTrail/internal/imagefetch"
	"github.com/ebrgr/TuneTrail/internal/imagepipeline"
	"github.com/ebrgr/TuneTrail/internal/imagework"
	"github.com/google/uuid"
)

// publicationGate coordinates cache publication with orphan collection.
var publicationGate = make(chan struct{}, 1)

// reservations is accessed only while publicationGate is held.
var reservations = make(map[uuid.UUID]int)
var resizeGroup = imagework.New(5, 30*time.Second)

// LockPublication coordinates cache publication with orphan collection.
func LockPublication() func() {
	unlock, _ := LockPublicationContext(context.Background())
	return unlock
}

// LockPublicationContext allows a canceled publication or collection to leave the queue.
func LockPublicationContext(ctx context.Context) (func(), error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	select {
	case publicationGate <- struct{}{}:
		if err := ctx.Err(); err != nil {
			<-publicationGate
			return nil, err
		}
		return func() { <-publicationGate }, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// ReservePublication protects an in-flight automatic image without holding the
// publication gate during download or database work. Release is idempotent.
func ReservePublication(ctx context.Context, id uuid.UUID) (func(), error) {
	if id == uuid.Nil {
		return nil, errors.New("cannot reserve a nil image ID")
	}
	unlock, err := LockPublicationContext(ctx)
	if err != nil {
		return nil, err
	}
	reservations[id]++
	unlock()
	var once sync.Once
	return func() {
		once.Do(func() {
			unlock := LockPublication()
			defer unlock()
			reservations[id]--
			if reservations[id] == 0 {
				delete(reservations, id)
			}
		})
	}, nil
}

// PublicationReservedLocked must be called while holding LockPublication.
func PublicationReservedLocked(id uuid.UUID) bool { return reservations[id] > 0 }

type ImageSize string

const (
	ImageSizeXS     ImageSize = "64x64"
	ImageSizeSmall  ImageSize = "128x128"
	ImageSizeMedium ImageSize = "300x300"
	ImageSizeLarge  ImageSize = "640x640"
	ImageSizeXL     ImageSize = "1000x1000"
	ImageSizeSource ImageSize = "source"

	ImageCacheDir = "image_cache"
)

func (s ImageSize) Width() int {
	switch s {
	case ImageSizeXS:
		return 64
	case ImageSizeSmall:
		return 128
	case ImageSizeMedium:
		return 300
	case ImageSizeLarge:
		return 640
	case ImageSizeXL:
		return 1000
	case ImageSizeSource:
		return 1200
	default:
		return 0
	}
}

func (s ImageSize) String() string {
	return string(s)
}

// DownloadImage downloads an image from the given URL, then saves it to the cache at source quality.
func DownloadImage(imgid uuid.UUID, url string) error {
	return DownloadImageContext(context.Background(), imgid, url)
}

// DownloadImageContext bounds network I/O and observes request cancellation.
func DownloadImageContext(ctx context.Context, imgid uuid.UUID, url string) error {
	data, err := imagefetch.Download(ctx, url)
	if err != nil {
		return fmt.Errorf("DownloadImage: %w", err)
	}
	// The pipeline validates format and dimensions before decoding.
	if err := compressAndSaveImageContext(ctx, imgid, ImageSizeSource, bytes.NewReader(data)); err != nil {
		return fmt.Errorf("DownloadImage: %w", err)
	}
	return nil
}

// SaveImage saves the image at source quality to the cache.
func SaveImage(imgid uuid.UUID, image io.Reader) error {
	return SaveImageContext(context.Background(), imgid, image)
}

// SaveImageContext observes cancellation while waiting for processing capacity.
func SaveImageContext(ctx context.Context, imgid uuid.UUID, image io.Reader) error {
	return compressAndSaveImageContext(ctx, imgid, ImageSizeSource, image)
}

type ImageInfo struct {
	Path string
	Mime string
}

// GetImage takes the UUID image id and image filename e.g. "128x128.webp" and returns the path and mime type of the
// image in the cache. If the image does not exist in the cache in either the desired or source sizes,
// returns a fs.ErrNotExist error.
func GetImage(imgid uuid.UUID, image string) (*ImageInfo, error) {
	return GetImageContext(context.Background(), imgid, image)
}

// GetImageContext shares regeneration and opens the source only inside singleflight.
func GetImageContext(ctx context.Context, imgid uuid.UUID, image string) (*ImageInfo, error) {
	imageSize, err := ParseImageSize(strings.Split(image, ".")[0])
	if err != nil {
		return nil, errors.New("GetImage: failed to parse image size")
	}
	desired := BuildImagePath(imgid, imageSize)
	if _, err = os.Stat(desired); errors.Is(err, fs.ErrNotExist) {
		key := imgid.String() + "/" + imageSize.String()
		err = resizeGroup.Do(ctx, key, func(workCtx context.Context) error {
			// Another request may have published the derivative while we waited.
			if _, err := os.Stat(desired); err == nil {
				return nil
			} else if !errors.Is(err, fs.ErrNotExist) {
				return err
			}
			source, err := os.Open(BuildImagePath(imgid, ImageSizeSource))
			if err != nil {
				return fmt.Errorf("GetImage: failed to open source: %w", err)
			}
			defer source.Close()
			return compressAndSaveImageContext(workCtx, imgid, imageSize, source)
		})
		if err != nil {
			return nil, fmt.Errorf("GetImage: failed to cache image: %w", err)
		}
	} else if err != nil {
		return nil, fmt.Errorf("GetImage: failed to access image: %w", err)
	}
	return &ImageInfo{Path: desired, Mime: "image/webp"}, nil
}

// Returns the ImageSize from a filename such as "1000x1000.webp", or an error if the size cannot be parsed.
func ParseImageSize(s string) (ImageSize, error) {
	s = strings.Split(s, ".")[0]
	switch ImageSize(s) {
	case ImageSizeXS, ImageSizeSmall, ImageSizeMedium, ImageSizeLarge, ImageSizeXL:
		return ImageSize(s), nil
	default:
		return "", fmt.Errorf("ParseImageSize: invalid image size %q", s)
	}
}

// compressAndSaveImageContext transforms before publishing any cache entry.
func compressAndSaveImageContext(ctx context.Context, imgid uuid.UUID, size ImageSize, input io.Reader) error {
	px := size.Width()
	if px == 0 {
		return imagepipeline.ErrSize
	}
	if size == ImageSizeSource {
		px = 0
	}
	encoded, err := imagepipeline.Transform(ctx, input, px)
	if err != nil {
		return fmt.Errorf("compress image: %w", err)
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := saveImageContext(ctx, imgid, size, bytes.NewReader(encoded)); err != nil {
		return fmt.Errorf("save image: %w", err)
	}
	return nil
}

// saveImage atomically publishes a complete image without exposing partial writes.
func saveImage(imgid uuid.UUID, size ImageSize, data io.Reader) error {
	return saveImageContext(context.Background(), imgid, size, data)
}

// saveImageContext checks cancellation again before the atomic rename.
func saveImageContext(ctx context.Context, imgid uuid.UUID, size ImageSize, data io.Reader) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	imagePath := BuildImagePath(imgid, size)

	// Ensure the cache directory exists
	err := os.MkdirAll(path.Dir(imagePath), 0744)
	if err != nil {
		return fmt.Errorf("saveImage: failed to create full image cache directory: %w", err)
	}

	// Create a file in the cache directory
	file, err := os.CreateTemp(path.Dir(imagePath), ".image-*")
	if err != nil {
		return fmt.Errorf("saveImage: failed to create image file: %w", err)
	}
	temporary := file.Name()
	defer os.Remove(temporary)
	defer file.Close()

	// Save the image to the file
	_, err = io.Copy(file, data)
	if err != nil {
		return fmt.Errorf("saveImage: failed to save image: %w", err)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("saveImage: close image: %w", err)
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := os.Rename(temporary, imagePath); err != nil {
		return fmt.Errorf("saveImage: publish image: %w", err)
	}

	return nil
}

func DeleteImage(filename uuid.UUID) error {

	err := os.RemoveAll(filepath.Dir(BuildImagePath(filename, ImageSizeSource)))
	if err != nil {
		return fmt.Errorf("DeleteImage: %w", err)
	}

	return nil
}

func BuildImagePath(imageid uuid.UUID, size ImageSize) string {
	return filepath.Join(cfg.ConfigDir(), ImageCacheDir, imageid.String()[:2], imageid.String(), string(size)+".webp")
}

func ForEachCachedImageID(foreach func(imageID uuid.UUID)) error {
	cacheDir := filepath.Join(cfg.ConfigDir(), ImageCacheDir)
	prefixDirs, err := os.ReadDir(cacheDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("ForEachCachedImage: reading cache dir: %w", err)
	}

	for _, prefixEntry := range prefixDirs {
		if !prefixEntry.IsDir() {
			continue
		}
		prefixPath := filepath.Join(cacheDir, prefixEntry.Name())

		uuidDirs, err := os.ReadDir(prefixPath)
		if err != nil {
			return fmt.Errorf("ForEachCachedImage: reading prefix dir %s: %w", prefixEntry.Name(), err)
		}

		for _, uuidEntry := range uuidDirs {
			if !uuidEntry.IsDir() {
				continue
			}
			uuidStr := uuidEntry.Name()
			imgid, err := uuid.Parse(uuidStr)
			if err != nil {
				// non-uuid filaname -> ignore
				continue
			}
			foreach(imgid)
		}
	}
	return nil
}
