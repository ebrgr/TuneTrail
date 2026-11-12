package imagepipeline

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	webp "github.com/SeriousBug/webp-go-pure/std"
	"image/color"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestCorpus031 checks that the actual component reproduces the measured candidate
// bytes; the optional work directory is produced by the documented 031 runner.
func TestCorpus031(t *testing.T) {
	work := os.Getenv("KOITO_IMAGE031_WORK")
	if work == "" {
		t.Skip("set KOITO_IMAGE031_WORK to the generated 031 work directory")
	}
	// Evidence moved from docs-dev/comparisons to backlog/evidencias with the Backlog.md adoption (ISS-059).
	report, err := os.ReadFile("../../backlog/evidencias/transicao-imagens/results-031.json")
	if err != nil {
		t.Fatal(err)
	}
	var data struct {
		Rows []struct {
			Mode, Input, Operation string
			Size, Effort           int
			SHA                    string          `json:"output_sha256"`
			Quality                json.RawMessage `json:"quality"`
			Error                  string          `json:"error"`
		} `json:"rows"`
	}
	if err := json.Unmarshal(report, &data); err != nil {
		t.Fatal(err)
	}
	checked := 0
	for i, row := range data.Rows {
		if row.Mode != "pure" || row.Effort != -1 || len(row.Quality) == 0 || row.Error != "" {
			continue
		}
		path := filepath.Join(work, "corpus", row.Input)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			path = filepath.Join(work, row.Input)
		}
		f, err := os.Open(path)
		if err != nil {
			t.Fatalf("row %d input: %v", i, err)
		}
		out, err := Transform(context.Background(), f, row.Size)
		closeErr := f.Close()
		if err != nil || closeErr != nil {
			t.Fatalf("row %d: transform=%v close=%v", i, err, closeErr)
		}
		sum := sha256.Sum256(out)
		stem := strings.TrimSuffix(row.Input, filepath.Ext(row.Input))
		name := fmt.Sprintf("%03d-pure-%s-%s-e-1.webp", i, stem, row.Operation)
		if dir := os.Getenv("KOITO_IMAGE032_OUTPUT"); dir != "" {
			if err := os.WriteFile(filepath.Join(dir, name), out, 0600); err != nil {
				t.Fatal(err)
			}
		}
		if got := hex.EncodeToString(sum[:]); got != row.SHA {
			previous, err := os.ReadFile(filepath.Join(work, name))
			if err != nil {
				t.Fatal(err)
			}
			psnr, err := compareWebP(previous, out)
			if err != nil || psnr < 40 {
				t.Errorf("row %d %s PSNR %.2f dB: %v", i, row.Input, psnr, err)
			} else {
				t.Logf("row %d %s differs: %.2f dB against 031", i, row.Input, psnr)
			}
		}
		checked++
	}
	if checked < 50 {
		t.Fatal(fmt.Sprintf("insufficient corpus coverage: %d", checked))
	}
	t.Logf("verified %d measured candidate outputs", checked)
}

// TestMaxPixels031 exercises the admitted 40 MP image when the corpus is available.
func TestMaxPixels031(t *testing.T) {
	work := os.Getenv("KOITO_IMAGE031_WORK")
	if work == "" {
		t.Skip("set KOITO_IMAGE031_WORK to the generated 031 work directory")
	}
	f, err := os.Open(filepath.Join(work, "corpus", "limit40mp.png"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	out, err := Transform(context.Background(), f, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(out) == 0 {
		t.Fatal("empty WebP")
	}
}

// TestCorpusAdmissions031 exercises byte, pixel and format boundaries from the corpus.
func TestCorpusAdmissions031(t *testing.T) {
	work := os.Getenv("KOITO_IMAGE031_WORK")
	if work == "" {
		t.Skip("set KOITO_IMAGE031_WORK to the generated 031 work directory")
	}
	cases := []struct {
		name string
		want error
	}{
		{"bytes-1.jpg", nil}, {"bytes+0.jpg", nil}, {"bytes+1.jpg", ErrTooLarge},
		{"axis10001.png", ErrDimensions}, {"pixels40008000.png", ErrDimensions},
		{"animated.webp", ErrAnimation}, {"animated.gif", ErrFormat}, {"extra.bmp", ErrFormat}, {"extra.tiff", ErrFormat},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f, err := os.Open(filepath.Join(work, "corpus", tc.name))
			if err != nil {
				t.Fatal(err)
			}
			defer f.Close()
			_, err = Transform(context.Background(), f, 0)
			if tc.want == nil && err != nil || tc.want != nil && !errors.Is(err, tc.want) {
				t.Fatalf("got %v, want %v", err, tc.want)
			}
		})
	}
}

// TestConcurrentResources031 checks two concurrent callers under the global limit.
func TestConcurrentResources031(t *testing.T) {
	work := os.Getenv("KOITO_IMAGE031_WORK")
	if work == "" {
		t.Skip("set KOITO_IMAGE031_WORK to the generated 031 work directory")
	}
	done := make(chan error, 2)
	for range 2 {
		go func() {
			f, err := os.Open(filepath.Join(work, "corpus", "large.png"))
			if err == nil {
				defer f.Close()
				_, err = Transform(context.Background(), f, 0)
			}
			done <- err
		}()
	}
	for range 2 {
		if err := <-done; err != nil {
			t.Fatal(err)
		}
	}
}

// compareWebP checks a cross-toolchain result against the prior decoded image.
func compareWebP(a, b []byte) (float64, error) {
	x, e := webp.DecodeBytes(a)
	if e != nil {
		return 0, e
	}
	y, e := webp.DecodeBytes(b)
	if e != nil {
		return 0, e
	}
	if x.Bounds().Size() != y.Bounds().Size() {
		return 0, fmt.Errorf("dimensions differ")
	}
	size := x.Bounds().Size()
	var sum float64
	for j := 0; j < size.Y; j++ {
		for i := 0; i < size.X; i++ {
			p := color.NRGBAModel.Convert(x.At(i+x.Bounds().Min.X, j+x.Bounds().Min.Y)).(color.NRGBA)
			q := color.NRGBAModel.Convert(y.At(i+y.Bounds().Min.X, j+y.Bounds().Min.Y)).(color.NRGBA)
			if p.A != q.A {
				return 0, fmt.Errorf("alpha differs")
			}
			for _, v := range [][2]uint8{{p.R, q.R}, {p.G, q.G}, {p.B, q.B}} {
				d := float64(int(v[0])-int(v[1])) * float64(p.A) / 255
				sum += d * d
			}
		}
	}
	if sum == 0 {
		return 100, nil
	}
	return 10 * math.Log10(255*255*float64(3*size.X*size.Y)/sum), nil
}
