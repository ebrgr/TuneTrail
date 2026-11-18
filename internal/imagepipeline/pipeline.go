// Package imagepipeline transforms bounded JPEG, PNG and still WebP inputs.
// Cache and administrative image uploads share this pipeline.
package imagepipeline

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io"

	codec "github.com/SeriousBug/webp-go-pure"
	webp "github.com/SeriousBug/webp-go-pure/std"
	"github.com/disintegration/imaging"
)

const (
	MaxBytes      = 10 << 20
	MaxSide       = 10000
	MaxPixels     = 40000000
	SourceMaxSide = 1200
)

var (
	ErrBusy         = errors.New("image pipeline is busy")
	ErrTooLarge     = errors.New("image exceeds byte limit")
	ErrDimensions   = errors.New("image dimensions exceed limits")
	ErrFormat       = errors.New("unsupported image format")
	ErrInvalidImage = errors.New("invalid image")
	ErrAnimation    = errors.New("animated images are unsupported")
	ErrSize         = errors.New("invalid output size")
)

// slots bounds active requests and waiters; active ensures one transform globally.
var slots = make(chan struct{}, 5)
var active = make(chan struct{}, 1)

// Transform validates and encodes a square WebP using bounded shared processing.
// Size zero selects source quality and the shortest side capped at 1200 pixels.
func Transform(ctx context.Context, input io.Reader, size int) ([]byte, error) {
	// Keep existing callers on the same transform and color calculation path.
	encoded, _, err := TransformWithColor(ctx, input, size)
	return encoded, err
}

// TransformWithColor computes the average of the oriented, cropped artwork while encoding.
func TransformWithColor(ctx context.Context, input io.Reader, size int) ([]byte, string, error) {
	if size < 0 || size > SourceMaxSide {
		return nil, "", ErrSize
	}
	var encoded []byte
	var average string
	err := WithDecoded(ctx, input, func(img image.Image) error {
		bounds := img.Bounds()
		target, quality := size, 85
		if target == 0 {
			target = min(bounds.Dx(), bounds.Dy(), SourceMaxSide)
			quality = 100
		}
		var result image.Image = img
		// Match bimg: preserve the original rectangle only when both sides are smaller.
		if bounds.Dx() >= target || bounds.Dy() >= target {
			result = imaging.Fill(img, target, target, imaging.Center, imaging.Lanczos)
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		average = AverageColor(result)
		var err error
		encoded, err = webp.EncodeBytes(result, &webp.Options{Quality: quality, Effort: -1})
		if err != nil {
			return fmt.Errorf("encode WebP: %w", err)
		}
		return nil
	})
	if err != nil {
		return nil, "", err
	}
	return encoded, average, nil
}

// WithDecoded validates and orients a bounded image before invoking use while
// holding the processing slot. The caller owns input and must not retain img
// beyond use. A blocked reader, decoder, or callback cannot be interrupted.
func WithDecoded(ctx context.Context, input io.Reader, use func(image.Image) error) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	select {
	case slots <- struct{}{}:
	default:
		return ErrBusy
	}
	defer func() { <-slots }()
	select {
	case active <- struct{}{}:
		defer func() { <-active }()
	case <-ctx.Done():
		return ctx.Err()
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	data, err := io.ReadAll(io.LimitReader(input, MaxBytes+1))
	if err != nil {
		return fmt.Errorf("read image: %w", err)
	}
	if len(data) > MaxBytes {
		return ErrTooLarge
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	width, height, err := inspect(data)
	if err != nil {
		return err
	}
	if err := validateDimensions(width, height); err != nil {
		return err
	}
	img, err := imaging.Decode(bytes.NewReader(data), imaging.AutoOrientation(true))
	if err != nil {
		return fmt.Errorf("%w: decode: %v", ErrInvalidImage, err)
	}
	bounds := img.Bounds()
	if err := validateDimensions(bounds.Dx(), bounds.Dy()); err != nil {
		return err
	}
	// EXIF may swap axes, but must not change the total decoded dimensions.
	if !((bounds.Dx() == width && bounds.Dy() == height) || (bounds.Dx() == height && bounds.Dy() == width)) {
		return ErrDimensions
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := use(img); err != nil {
		return err
	}
	return ctx.Err()
}

// inspect applies the allowlist before decoder selection and rejects animation.
func inspect(data []byte) (int, int, error) {
	if len(data) >= 12 && string(data[:4]) == "RIFF" && string(data[8:12]) == "WEBP" {
		f, err := codec.Features(data)
		if err != nil {
			return 0, 0, fmt.Errorf("%w: inspect WebP: %v", ErrInvalidImage, err)
		}
		if f.HasAnimation {
			return 0, 0, ErrAnimation
		}
		return f.Width, f.Height, nil
	}
	var config image.Config
	var err error
	switch {
	case len(data) >= 3 && data[0] == 0xff && data[1] == 0xd8 && data[2] == 0xff:
		config, err = jpeg.DecodeConfig(bytes.NewReader(data))
	case len(data) >= 8 && bytes.Equal(data[:8], []byte{137, 80, 78, 71, 13, 10, 26, 10}):
		if hasPNGAnimation(data) {
			return 0, 0, ErrAnimation
		}
		config, err = png.DecodeConfig(bytes.NewReader(data))
	default:
		return 0, 0, ErrFormat
	}
	if err != nil {
		return 0, 0, fmt.Errorf("%w: inspect: %v", ErrInvalidImage, err)
	}
	return config.Width, config.Height, nil
}

// validateDimensions uses int64 multiplication to avoid overflow on narrow hosts.
func validateDimensions(width, height int) error {
	if width <= 0 || height <= 0 || width > MaxSide || height > MaxSide || int64(width)*int64(height) > MaxPixels {
		return ErrDimensions
	}
	return nil
}

// hasPNGAnimation scans bounded PNG chunks without allocating their contents.
func hasPNGAnimation(data []byte) bool {
	for offset := 8; offset+12 <= len(data); {
		size := uint64(binary.BigEndian.Uint32(data[offset : offset+4]))
		if string(data[offset+4:offset+8]) == "acTL" {
			return true
		}
		next := uint64(offset) + 12 + size
		if next > uint64(len(data)) {
			return false
		}
		offset = int(next)
	}
	return false
}
