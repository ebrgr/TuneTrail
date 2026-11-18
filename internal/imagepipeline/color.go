package imagepipeline

import (
	"fmt"
	"image"
)

// AverageColor averages premultiplied RGB over black, including transparent pixels.
// The bounded transform owns the decoded image; no second decode is needed.
func AverageColor(img image.Image) string {
	var red, green, blue, count uint64
	bounds := img.Bounds()
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, _ := img.At(x, y).RGBA()
			red += uint64(r)
			green += uint64(g)
			blue += uint64(b)
			count++
		}
	}
	if count == 0 {
		return "#000000"
	}
	return fmt.Sprintf("#%02x%02x%02x", red/count/257, green/count/257, blue/count/257)
}
