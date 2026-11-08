package summary

import (
	"context"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"io"

	"github.com/disintegration/imaging"
	"github.com/ebrgr/TuneTrail/internal/imagepipeline"
	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
)

// The unfinished GenerateImage prototype remains disabled. A future sharing
// feature must explicitly provide input images and licensed font bytes.

// addImage stretches the decoded image to a square and keeps draw.Draw's source
// point semantics. The caller owns input; this function never closes it.
func addImage(ctx context.Context, baseImage *image.RGBA, input io.Reader, point image.Point, height int) error {
	if baseImage == nil || height <= 0 || height > imagepipeline.SourceMaxSide {
		return imagepipeline.ErrSize
	}
	return imagepipeline.WithDecoded(ctx, input, func(img image.Image) error {
		resized := resize(img, height, height)
		if err := ctx.Err(); err != nil {
			return err
		}
		// point is a source coordinate, not a destination offset.
		draw.Draw(baseImage, baseImage.Bounds(), resized, point, draw.Over)
		return nil
	})
}

// addText uses caller-supplied font bytes and closes every face it creates.
func addText(baseImage *image.RGBA, text, subtext string, point image.Point, fontBytes []byte, fontSize float64) error {
	if baseImage == nil {
		return fmt.Errorf("nil text canvas")
	}
	ttf, err := opentype.Parse(fontBytes)
	if err != nil {
		return fmt.Errorf("parse font: %w", err)
	}
	face, err := opentype.NewFace(ttf, &opentype.FaceOptions{Size: fontSize, DPI: 72, Hinting: font.HintingFull})
	if err != nil {
		return fmt.Errorf("create font face: %w", err)
	}
	defer face.Close()
	drawer := &font.Drawer{Dst: baseImage, Src: image.NewUniform(color.White), Face: face, Dot: fixed.Point26_6{X: fixed.I(point.X), Y: fixed.I(point.Y)}}
	drawer.DrawString(text)
	if subtext != "" {
		secondary, err := opentype.NewFace(ttf, &opentype.FaceOptions{Size: 16, DPI: 72, Hinting: font.HintingFull})
		if err != nil {
			return fmt.Errorf("create secondary font face: %w", err)
		}
		defer secondary.Close()
		drawer.Face = secondary
		drawer.Src = image.NewUniform(color.RGBA{200, 200, 200, 255})
		drawer.DrawString(" - ")
		drawer.DrawString(subtext)
	}
	return nil
}

// resize stretches both dimensions with nearest-neighbor sampling, never Fit or Fill.
// The filter samples pixel centers, unlike the legacy floor-based sampler.
func resize(m image.Image, w, h int) *image.RGBA {
	if w < 0 || h < 0 {
		return nil
	}
	if w == 0 || h == 0 || m.Bounds().Empty() {
		return image.NewRGBA(image.Rect(0, 0, w, h))
	}
	scaled := imaging.Resize(m, w, h, imaging.NearestNeighbor)
	result := image.NewRGBA(scaled.Bounds())
	draw.Draw(result, result.Bounds(), scaled, scaled.Bounds().Min, draw.Src)
	return result
}
