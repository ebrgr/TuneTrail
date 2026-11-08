package summary

import (
	"bytes"
	"context"
	"errors"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	webp "github.com/SeriousBug/webp-go-pure/std"
	"github.com/ebrgr/TuneTrail/internal/imagepipeline"
	"golang.org/x/image/font/gofont/goregular"
)

// fictionalPNG encodes a translucent image without external assets.
func fictionalPNG(t *testing.T) []byte {
	t.Helper()
	src := image.NewNRGBA(image.Rect(0, 0, 2, 2))
	src.SetNRGBA(0, 0, color.NRGBA{255, 0, 0, 128})
	src.SetNRGBA(1, 0, color.NRGBA{0, 255, 0, 255})
	src.SetNRGBA(0, 1, color.NRGBA{0, 0, 255, 255})
	src.SetNRGBA(1, 1, color.NRGBA{255, 255, 0, 255})
	var out bytes.Buffer
	if err := png.Encode(&out, src); err != nil {
		t.Fatal(err)
	}
	return out.Bytes()
}

// TestResize035 checks stretch, center-sampled nearest pixels and nonzero bounds.
func TestResize035(t *testing.T) {
	src := image.NewNRGBA(image.Rect(10, 20, 14, 22))
	for y := 20; y < 22; y++ {
		for x := 10; x < 14; x++ {
			src.SetNRGBA(x, y, color.NRGBA{uint8((x - 10) * 40), uint8((y - 20) * 100), 10, 255})
		}
	}
	got := resize(src, 2, 4)
	if got.Bounds() != image.Rect(0, 0, 2, 4) {
		t.Fatal(got.Bounds())
	}
	for y := 0; y < 4; y++ {
		for x := 0; x < 2; x++ {
			want := color.RGBA{uint8((2*x + 1) * 40), uint8((y / 2) * 100), 10, 255}
			if pixel := got.RGBAAt(x, y); pixel != want {
				t.Fatalf("%d,%d got %v want %v", x, y, pixel, want)
			}
		}
	}
	if resize(src, -1, 2) != nil {
		t.Fatal("negative dimension accepted")
	}
	if got := resize(src, 0, 2); got.Bounds() != image.Rect(0, 0, 0, 2) {
		t.Fatal(got.Bounds())
	}
	if got := resize(image.NewRGBA(image.Rect(2, 2, 2, 2)), 2, 3); got.Bounds() != image.Rect(0, 0, 2, 3) {
		t.Fatal(got.Bounds())
	}
}

// TestAddImage035 checks clipping, source coordinates, transparency and nonzero canvas origins.
func TestAddImage035(t *testing.T) {
	for _, point := range []image.Point{{-2, -1}, {1, 1}, {50, 50}} {
		canvas := image.NewRGBA(image.Rect(10, 20, 15, 25))
		draw.Draw(canvas, canvas.Bounds(), image.NewUniform(color.RGBA{20, 40, 80, 255}), image.Point{}, draw.Src)
		if err := addImage(context.Background(), canvas, bytes.NewReader(fictionalPNG(t)), point, 2); err != nil {
			t.Fatal(err)
		}
		for y := 20; y < 25; y++ {
			for x := 10; x < 15; x++ {
				sx, sy := x-10+point.X, y-20+point.Y
				want := color.RGBA{20, 40, 80, 255}
				switch {
				case sx == 0 && sy == 0:
					want = color.RGBA{138, 19, 39, 255}
				case sx == 1 && sy == 0:
					want = color.RGBA{0, 255, 0, 255}
				case sx == 0 && sy == 1:
					want = color.RGBA{0, 0, 255, 255}
				case sx == 1 && sy == 1:
					want = color.RGBA{255, 255, 0, 255}
				}
				if got := canvas.RGBAAt(x, y); got != want {
					t.Fatalf("point %v pixel %d,%d = %v want %v", point, x, y, got, want)
				}
			}
		}
	}
}

// TestAddImageFormats035 checks the shared policy and WebP decoder through the helper.
func TestAddImageFormats035(t *testing.T) {
	canvas := image.NewRGBA(image.Rect(0, 0, 8, 8))
	input := fictionalPNG(t)
	im, err := png.Decode(bytes.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := webp.EncodeBytes(im, &webp.Options{Quality: 85, Effort: -1})
	if err != nil {
		t.Fatal(err)
	}
	if err = addImage(context.Background(), canvas, bytes.NewReader(encoded), image.Point{}, 2); err != nil {
		t.Fatal(err)
	}
	if err = addImage(context.Background(), canvas, bytes.NewReader([]byte("GIF89a")), image.Point{}, 2); !errors.Is(err, imagepipeline.ErrFormat) {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err = addImage(ctx, canvas, bytes.NewReader(input), image.Point{}, 2); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if err = addImage(context.Background(), canvas, bytes.NewReader(input), image.Point{}, 0); !errors.Is(err, imagepipeline.ErrSize) {
		t.Fatal(err)
	}
}

// TestText035 checks explicit font bytes, output, error propagation and cwd independence.
func TestText035(t *testing.T) {
	t.Chdir(t.TempDir())
	canvas := image.NewRGBA(image.Rect(0, 0, 400, 80))
	if err := addText(canvas, "Fictional artist", "12 listens", image.Pt(5, 32), goregular.TTF, 24); err != nil {
		t.Fatal(err)
	}
	changed := false
	for _, v := range canvas.Pix {
		if v != 0 {
			changed = true
			break
		}
	}
	if !changed {
		t.Fatal("text did not render")
	}
	if err := addText(canvas, "x", "", image.Pt(0, 20), []byte("bad"), 16); err == nil {
		t.Fatal("invalid font accepted")
	}
}

// TestComposition035 writes an optional fictional artifact without reading a database or assets.
func TestComposition035(t *testing.T) {
	canvas := image.NewRGBA(image.Rect(0, 0, 680, 380))
	draw.Draw(canvas, canvas.Bounds(), image.NewUniform(color.RGBA{23, 20, 27, 255}), image.Point{}, draw.Src)
	input := fictionalPNG(t)
	if err := addText(canvas, "Retrospective", "fictional preview", image.Pt(25, 48), goregular.TTF, 28); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 3; i++ {
		x := 25 + i*210
		if err := addImage(context.Background(), canvas, bytes.NewReader(input), image.Pt(-x, -80), 180); err != nil {
			t.Fatal(err)
		}
		if err := addText(canvas, "Artist", "sample", image.Pt(x, 298), goregular.TTF, 20); err != nil {
			t.Fatal(err)
		}
	}
	if dir := os.Getenv("KOITO_SUMMARY035_OUTPUT"); dir != "" {
		f, err := os.Create(filepath.Join(dir, "composition-035.png"))
		if err != nil {
			t.Fatal(err)
		}
		if err = png.Encode(f, canvas); err != nil {
			f.Close()
			t.Fatal(err)
		}
		if err = f.Close(); err != nil {
			t.Fatal(err)
		}
	}
}

// TestSamplingComparison035 documents the intentional center-sampling difference.
func TestSamplingComparison035(t *testing.T) {
	src := image.NewNRGBA(image.Rect(0, 0, 3, 1))
	src.SetNRGBA(0, 0, color.NRGBA{255, 70, 70, 255})
	src.SetNRGBA(1, 0, color.NRGBA{50, 220, 100, 255})
	src.SetNRGBA(2, 0, color.NRGBA{70, 110, 255, 255})
	current := resize(src, 2, 1)
	if current.RGBAAt(0, 0) != (color.RGBA{255, 70, 70, 255}) || current.RGBAAt(1, 0) != (color.RGBA{70, 110, 255, 255}) {
		t.Fatalf("unexpected center sampling: %v %v", current.RGBAAt(0, 0), current.RGBAAt(1, 0))
	}
	// The original floor sampler selected source pixels 0 and 1 for a 3→2 reduction.
	previous := image.NewRGBA(image.Rect(0, 0, 2, 1))
	previous.Set(0, 0, src.At(0, 0))
	previous.Set(1, 0, src.At(1, 0))
	if dir := os.Getenv("KOITO_SUMMARY035_OUTPUT"); dir != "" {
		canvas := image.NewRGBA(image.Rect(0, 0, 680, 230))
		draw.Draw(canvas, canvas.Bounds(), image.NewUniform(color.RGBA{23, 20, 27, 255}), image.Point{}, draw.Src)
		if err := addText(canvas, "Previous", "floor", image.Pt(20, 35), goregular.TTF, 23); err != nil {
			t.Fatal(err)
		}
		if err := addText(canvas, "Current", "pixel centers", image.Pt(360, 35), goregular.TTF, 23); err != nil {
			t.Fatal(err)
		}
		a, b := resize(previous, 300, 140), resize(current, 300, 140)
		draw.Draw(canvas, image.Rect(20, 60, 320, 200), a, image.Point{}, draw.Src)
		draw.Draw(canvas, image.Rect(360, 60, 660, 200), b, image.Point{}, draw.Src)
		f, err := os.Create(filepath.Join(dir, "sampling-035.png"))
		if err != nil {
			t.Fatal(err)
		}
		if err = png.Encode(f, canvas); err != nil {
			f.Close()
			t.Fatal(err)
		}
		if err = f.Close(); err != nil {
			t.Fatal(err)
		}
	}
}
