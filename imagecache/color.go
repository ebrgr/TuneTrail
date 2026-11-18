package imagecache

import (
	"context"
	"fmt"
	"image"
	"io"
	"math"
	"os"
	"path/filepath"
	"strings"

	"github.com/ebrgr/TuneTrail/internal/imagepipeline"
	"github.com/google/uuid"
)

// HeaderAccent reads color metadata; older caches are backfilled through bounded decoding.
// A missing cover is decorative and uses the same safe neutral fallback as invalid metadata.
func HeaderAccent(ctx context.Context, imageURL string) string {
	const fallback = "#253044"
	parts := strings.Split(imageURL, "/")
	if len(parts) != 4 || parts[1] != "image" {
		return fallback
	}
	id, err := uuid.Parse(parts[2])
	if err != nil || id == uuid.Nil {
		return fallback
	}
	file := colorPath(id)
	data, err := os.ReadFile(file)
	if os.IsNotExist(err) {
		// Share legacy extraction with derivative work; canceled visitors can stop waiting.
		err = resizeGroup.Do(ctx, id.String()+"/color", func(workCtx context.Context) error {
			if _, e := os.Stat(file); e == nil {
				return nil
			}
			source, e := os.Open(BuildImagePath(id, ImageSizeSource))
			if e != nil {
				return e
			}
			defer source.Close()
			return imagepipeline.WithDecoded(workCtx, source, func(img image.Image) error {
				return saveColor(workCtx, id, imagepipeline.AverageColor(img))
			})
		})
		if err == nil {
			data, err = os.ReadFile(file)
		}
	}
	if err != nil {
		return fallback
	}
	return contrastColor(string(data), fallback)
}

// contrastColor darkens the average until white text has at least 7:1 contrast.
// The gradient only adds black, so every point retains that minimum in every theme.
func contrastColor(value, fallback string) string {
	if len(value) != 7 || value[0] != '#' {
		return fallback
	}
	var r, g, b uint8
	if n, err := fmt.Sscanf(value, "#%02x%02x%02x", &r, &g, &b); err != nil || n != 3 {
		return fallback
	}
	linear := func(v uint8) float64 {
		s := float64(v) / 255
		if s <= .04045 {
			return s / 12.92
		}
		return math.Pow((s+.055)/1.055, 2.4)
	}
	for .2126*linear(r)+.7152*linear(g)+.0722*linear(b) > .1 {
		r = uint8(float64(r) * .95)
		g = uint8(float64(g) * .95)
		b = uint8(float64(b) * .95)
	}
	return fmt.Sprintf("#%02x%02x%02x", r, g, b)
}

// colorPath keeps metadata inside the image directory so orphan collection removes both.
func colorPath(id uuid.UUID) string {
	return filepath.Join(filepath.Dir(BuildImagePath(id, ImageSizeSource)), "average-color")
}

// saveColor atomically publishes metadata without exposing partial color strings.
func saveColor(ctx context.Context, id uuid.UUID, color string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	file, err := os.CreateTemp(filepath.Dir(colorPath(id)), ".color-*")
	if err != nil {
		return fmt.Errorf("create artwork color: %w", err)
	}
	defer os.Remove(file.Name())
	defer file.Close()
	if _, err = io.WriteString(file, color); err != nil {
		return fmt.Errorf("write artwork color: %w", err)
	}
	if err = file.Close(); err != nil {
		return fmt.Errorf("close artwork color: %w", err)
	}
	if err = ctx.Err(); err != nil {
		return err
	}
	if err = os.Rename(file.Name(), colorPath(id)); err != nil {
		return fmt.Errorf("publish artwork color: %w", err)
	}
	return nil
}
