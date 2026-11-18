package imagecache

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// TestArtworkColor073 checks publication, legacy recovery, missing art and corrupt metadata.
func TestArtworkColor073(t *testing.T) {
	cache033(t)
	ctx := context.Background()
	id := uuid.New()
	img := image.NewNRGBA(image.Rect(0, 0, 4, 4))
	for y := 0; y < 4; y++ {
		for x := 0; x < 4; x++ {
			img.SetNRGBA(x, y, color.NRGBA{255, 180, 60, 255})
		}
	}
	var input bytes.Buffer
	require.NoError(t, png.Encode(&input, img))
	require.NoError(t, SaveImageContext(ctx, id, &input))
	raw, err := os.ReadFile(colorPath(id))
	require.NoError(t, err)
	require.Equal(t, "#ffb43c", string(raw))
	url := "/image/" + id.String() + "/640x640.webp"
	accent := HeaderAccent(ctx, url)
	require.NotEqual(t, "#253044", accent)
	// Reading valid metadata must not decode the source again.
	source, err := os.ReadFile(BuildImagePath(id, ImageSizeSource))
	require.NoError(t, err)
	require.NoError(t, os.Remove(BuildImagePath(id, ImageSizeSource)))
	require.Equal(t, accent, HeaderAccent(ctx, url))
	require.NoError(t, os.WriteFile(BuildImagePath(id, ImageSizeSource), source, 0600))
	require.NoError(t, os.Remove(colorPath(id)))
	require.NotEqual(t, "#253044", HeaderAccent(ctx, url))
	require.FileExists(t, colorPath(id))
	require.NoError(t, os.WriteFile(colorPath(id), []byte("red; background:url(evil)"), 0600))
	require.Equal(t, "#253044", HeaderAccent(ctx, url))
	for _, url := range []string{"", "/web-assets/fallback.svg", "/image/nope/640x640.webp", "/image/" + uuid.NewString() + "/640x640.webp"} {
		require.Equal(t, "#253044", HeaderAccent(ctx, url))
	}
	require.NoError(t, DeleteImage(id))
	require.NoFileExists(t, colorPath(id))
}

// TestHeaderContrast073 independently verifies WCAG contrast over the full RGB cube sampled by 17.
func TestHeaderContrast073(t *testing.T) {
	luminance := func(v uint8) float64 {
		s := float64(v) / 255
		if s <= .04045 {
			return s / 12.92
		}
		return math.Pow((s+.055)/1.055, 2.4)
	}
	for r := 0; r <= 255; r += 17 {
		for g := 0; g <= 255; g += 17 {
			for b := 0; b <= 255; b += 17 {
				original := fmt.Sprintf("#%02x%02x%02x", r, g, b)
				result := contrastColor(original, "#253044")
				var red, green, blue uint8
				n, err := fmt.Sscanf(result, "#%02x%02x%02x", &red, &green, &blue)
				require.NoError(t, err)
				require.Equal(t, 3, n)
				ratio := 1.05 / (.2126*luminance(red) + .7152*luminance(green) + .0722*luminance(blue) + .05)
				require.GreaterOrEqual(t, ratio, 7.0, original)
			}
		}
	}
}
