package imagepipeline

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/png"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestAverageColor073 verifies averaging, alpha compositing, bounds and transform output.
func TestAverageColor073(t *testing.T) {
	img := image.NewNRGBA(image.Rect(10, 20, 12, 21))
	img.SetNRGBA(10, 20, color.NRGBA{255, 0, 0, 255})
	img.SetNRGBA(11, 20, color.NRGBA{0, 0, 255, 255})
	require.Equal(t, "#7f007f", AverageColor(img))
	img.SetNRGBA(11, 20, color.NRGBA{0, 255, 0, 0})
	require.Equal(t, "#7f0000", AverageColor(img))
	require.Equal(t, "#000000", AverageColor(image.NewRGBA(image.Rectangle{})))
	var data bytes.Buffer
	require.NoError(t, png.Encode(&data, img))
	encoded, average, err := TransformWithColor(context.Background(), &data, 0)
	require.NoError(t, err)
	require.NotEmpty(t, encoded)
	require.Len(t, average, 7)
}
