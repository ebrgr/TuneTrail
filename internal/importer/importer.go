package importer

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"time"

	"github.com/ebrgr/TuneTrail/internal/cfg"
	"github.com/ebrgr/TuneTrail/internal/logger"
)

// ErrInvalidFile marks input rejected before or during parsing.
var ErrInvalidFile = errors.New("invalid import file")

func invalidFile(err error) error { return fmt.Errorf("%w: %v", ErrInvalidFile, err) }

// HTTP uploads are staged outside the boot import directory and explicitly archived on success.
type uploadSourceKey struct{}

func WithUploadSource(ctx context.Context, filename string) context.Context {
	return context.WithValue(ctx, uploadSourceKey{}, filename)
}
func importPath(ctx context.Context, filename string) string {
	if source, ok := ctx.Value(uploadSourceKey{}).(string); ok {
		return source
	}
	return path.Join(cfg.ConfigDir(), "import", filename)
}

const MaxJSONBytes int64 = 32 << 20
const MaxImportItems = 250000

func decodeJSON(ctx context.Context, file *os.File, target any) error {
	info, err := file.Stat()
	if err != nil {
		return err
	}
	if info.Size() > MaxJSONBytes {
		return invalidFile(fmt.Errorf("JSON exceeds import limit"))
	}
	// Bound structural complexity before decoding into slices: compact arrays of
	// empty objects can otherwise allocate far more memory than the input size.
	preflight := json.NewDecoder(&contextReader{ctx: ctx, reader: io.LimitReader(file, MaxJSONBytes+1)})
	preflight.UseNumber()
	for tokens := 0; ; tokens++ {
		if tokens > 500000 {
			return invalidFile(fmt.Errorf("JSON exceeds structural limit"))
		}
		_, e := preflight.Token()
		if errors.Is(e, io.EOF) {
			break
		}
		if e != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			var disk *os.PathError
			if errors.As(e, &disk) {
				return e
			}
			return invalidFile(e)
		}
	}
	if _, err = file.Seek(0, io.SeekStart); err != nil {
		return err
	}
	dec := json.NewDecoder(&contextReader{ctx: ctx, reader: io.LimitReader(file, MaxJSONBytes+1)})
	if err = dec.Decode(target); err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		var disk *os.PathError
		if errors.As(err, &disk) {
			return err
		}
		return invalidFile(err)
	}
	var extra any
	if err = dec.Decode(&extra); !errors.Is(err, io.EOF) {
		if err == nil {
			return invalidFile(fmt.Errorf("extra JSON document"))
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return invalidFile(err)
	}
	return ctx.Err()
}

type contextReader struct {
	ctx    context.Context
	reader io.Reader
}

func (r *contextReader) Read(p []byte) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}
	return r.reader.Read(p)
}
func throttle(ctx context.Context) error {
	ms := cfg.ThrottleImportMs()
	if ms <= 0 {
		return ctx.Err()
	}
	timer := time.NewTimer(time.Duration(ms) * time.Millisecond)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

// finishImport archives only completed files and reports failures to the caller.
func finishImport(ctx context.Context, filename string, numImported int) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	destination := path.Join(cfg.ConfigDir(), "import_complete", filename)
	if err := os.MkdirAll(path.Dir(destination), 0700); err != nil {
		return fmt.Errorf("prepare archive: %w", err)
	}
	if err := os.Rename(importPath(ctx, filename), destination); err != nil {
		return fmt.Errorf("archive import: %w", err)
	}
	if numImported > 0 {
		logger.FromContext(ctx).Info().Int("processed", numImported).Msg("Import finished")
	}
	return nil
}

// from https://stackoverflow.com/a/55093788 with modification to use cfg and check for zero values
func inImportTimeWindow(check time.Time) bool {
	end, start := cfg.ImportWindow()
	if start.IsZero() && end.IsZero() {
		return true
	}
	if !start.IsZero() && end.IsZero() {
		return !check.Before(start)
	}
	if start.IsZero() && !end.IsZero() {
		return !check.After(end)
	}
	return !check.Before(start) && !check.After(end)
}
