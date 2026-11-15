// Package webassets loads the external UI distribution into immutable process memory.
package webassets

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

const maxFileBytes = 4 << 20
const maxTotalBytes = 32 << 20

// Snapshot is a read-only generation; callers receive copies of bytes, never its backing map.
type Snapshot struct {
	files  map[string][]byte
	assets map[string]Asset
	Hash   string
	Bytes  int
}

var current struct {
	sync.Once
	snapshot *Snapshot
	err      error
}

// Locate fixes the root independently of cwd; explicit invalid roots never fall back.
func Locate() (string, error) {
	if p := os.Getenv("TUNETRAIL_WEB_DIR"); p != "" {
		return filepath.Abs(p)
	}
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	exe, err = filepath.EvalSymlinks(exe)
	if err != nil {
		return "", err
	}
	return filepath.Join(filepath.Dir(exe), "assets"), nil
}

// Current loads exactly one generation per process. Restart to reload disk content.
func Current() (*Snapshot, error) {
	current.Do(func() {
		dir, err := Locate()
		if err == nil {
			current.snapshot, err = Open(context.Background(), dir)
		}
		current.err = err
	})
	return current.snapshot, current.err
}

// Open confines all disk reads through os.Root, including symlinks, before closing the root.
func Open(ctx context.Context, dir string) (*Snapshot, error) {
	root, err := os.OpenRoot(dir)
	if err != nil {
		return nil, fmt.Errorf("load web assets %q (set TUNETRAIL_WEB_DIR or install assets beside executable): %w", dir, err)
	}
	// The root holds read-only descriptors; closing does not publish or persist data.
	defer root.Close()
	snapshot, err := Load(ctx, confinedFS{root})
	if err != nil {
		return nil, fmt.Errorf("load web assets %q: %w", dir, err)
	}
	return snapshot, nil
}

// Load validates the explicit manifest and bounded regular files before publishing a generation.
func Load(ctx context.Context, source fs.FS) (*Snapshot, error) {
	read := func(name string) ([]byte, error) {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		st, err := fs.Stat(source, name)
		if err != nil {
			return nil, err
		}
		if !st.Mode().IsRegular() {
			return nil, fmt.Errorf("%s is not a regular file", name)
		}
		f, err := source.Open(name)
		if err != nil {
			return nil, err
		}
		defer f.Close()
		info, err := f.Stat()
		if err != nil {
			return nil, err
		}
		if !info.Mode().IsRegular() {
			return nil, fmt.Errorf("%s is not a regular file", name)
		}
		data, err := io.ReadAll(io.LimitReader(f, maxFileBytes+1))
		if err != nil {
			return nil, err
		}
		if len(data) > maxFileBytes {
			return nil, fmt.Errorf("%s exceeds asset size limit", name)
		}
		return data, nil
	}
	data, err := read("manifest.json")
	if err != nil {
		return nil, err
	}
	var raw map[string]json.RawMessage
	if err = json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("manifest: %w", err)
	}
	manifest := make(map[string][]string, len(raw))
	for key, value := range raw {
		if manifest[key], err = manifestSources(key, value); err != nil {
			return nil, err
		}
	}
	for _, key := range []string{"public/templates/live.html", "public/templates/page.html", "public/templates/rewind.html", "admin/templates/admin.html", "admin/templates/nav.html", "public/static/ui.css", "public/static/admin.css", "public/static/theme.js", "public/static/ui.js", "public/static/live.js", "public/static/admin.js", "public/palettes.json"} {
		if _, ok := manifest[key]; !ok {
			return nil, fmt.Errorf("manifest missing %s", key)
		}
	}
	s := &Snapshot{files: map[string][]byte{}}
	keys := make([]string, 0, len(manifest))
	for k := range manifest {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	hash := sha256.New()
	for _, key := range keys {
		// A CSS bundle concatenates its parts in manifest order (ISSUE-068); other keys have one part.
		var b []byte
		for _, name := range manifest[key] {
			if !validEntry(key, name) {
				return nil, fmt.Errorf("invalid manifest path %s", key)
			}
			part, err := read(name)
			if err != nil {
				return nil, fmt.Errorf("%s: %w", name, err)
			}
			if len(b) > 0 && b[len(b)-1] != '\n' {
				b = append(b, '\n')
			}
			b = append(b, part...)
			if len(b) > maxFileBytes {
				return nil, fmt.Errorf("%s exceeds asset size limit", key)
			}
		}
		s.Bytes += len(b)
		if s.Bytes > maxTotalBytes {
			return nil, fmt.Errorf("asset generation exceeds total size limit")
		}
		s.files[key] = b
		hash.Write([]byte(key))
		hash.Write([]byte{0})
		hash.Write(b)
	}
	s.Hash = hex.EncodeToString(hash.Sum(nil))
	if err := s.buildCatalog(); err != nil {
		return nil, err
	}
	return s, nil
}

// FS adapts a private snapshot subtree to fs.FS for template parsing and internal reads.
type FS struct{ Prefix string }

func (f FS) ReadFile(name string) ([]byte, error) {
	if !fs.ValidPath(name) {
		return nil, fs.ErrInvalid
	}
	s, err := Current()
	if err != nil {
		return nil, err
	}
	return s.read(path.Join(f.Prefix, name))
}
func (f FS) Open(name string) (fs.File, error) {
	s, err := Current()
	if err != nil {
		return nil, err
	}
	if !fs.ValidPath(name) {
		return nil, fs.ErrInvalid
	}
	return s.open(path.Join(f.Prefix, name))
}
func (s *Snapshot) read(name string) ([]byte, error) {
	b, ok := s.files[name]
	if !ok {
		return nil, fs.ErrNotExist
	}
	return bytes.Clone(b), nil
}
func (s *Snapshot) open(name string) (fs.File, error) {
	if b, ok := s.files[name]; ok {
		return &file{Reader: bytes.NewReader(b), info: info{name: path.Base(name), size: int64(len(b))}}, nil
	}
	prefix := strings.TrimSuffix(name, "/") + "/"
	children := map[string]fs.DirEntry{}
	for key, b := range s.files {
		if strings.HasPrefix(key, prefix) {
			rel := strings.TrimPrefix(key, prefix)
			child, _, dir := strings.Cut(rel, "/")
			children[child] = fs.FileInfoToDirEntry(info{name: child, size: int64(len(b)), dir: dir})
		}
	}
	if len(children) == 0 {
		return nil, fs.ErrNotExist
	}
	names := make([]string, 0, len(children))
	for k := range children {
		names = append(names, k)
	}
	sort.Strings(names)
	entries := []fs.DirEntry{}
	for _, k := range names {
		entries = append(entries, children[k])
	}
	return &file{Reader: bytes.NewReader(nil), info: info{name: path.Base(name), dir: true}, entries: entries}, nil
}

type info struct {
	name string
	size int64
	dir  bool
}

func (i info) Name() string { return i.name }
func (i info) Size() int64  { return i.size }
func (i info) Mode() fs.FileMode {
	if i.dir {
		return fs.ModeDir | 0555
	}
	return 0444
}
func (i info) ModTime() time.Time { return time.Time{} }
func (i info) IsDir() bool        { return i.dir }
func (i info) Sys() any           { return nil }

type file struct {
	*bytes.Reader
	info    info
	entries []fs.DirEntry
	offset  int
}

func (f *file) Stat() (fs.FileInfo, error) { return f.info, nil }
func (f *file) Close() error               { return nil }
func (f *file) ReadDir(n int) ([]fs.DirEntry, error) {
	if n > 0 && f.offset >= len(f.entries) {
		return nil, io.EOF
	}
	end := len(f.entries)
	if n > 0 && end > f.offset+n {
		end = f.offset + n
	}
	out := f.entries[f.offset:end]
	f.offset = end
	return out, nil
}

// manifestSources accepts one path, or an ordered non-empty list of CSS parts for a public stylesheet bundle (ISSUE-068).
func manifestSources(key string, value json.RawMessage) ([]string, error) {
	var name string
	if err := json.Unmarshal(value, &name); err == nil {
		return []string{name}, nil
	}
	var parts []string
	if err := json.Unmarshal(value, &parts); err != nil {
		return nil, fmt.Errorf("manifest %s: want a path or a list of CSS paths", key)
	}
	if len(parts) == 0 || !strings.HasPrefix(key, "public/static/") || path.Ext(key) != ".css" {
		return nil, fmt.Errorf("manifest %s: only public stylesheets may list parts", key)
	}
	seen := map[string]bool{}
	for _, part := range parts {
		if seen[part] {
			return nil, fmt.Errorf("manifest %s: duplicate part %s", key, part)
		}
		seen[part] = true
	}
	return parts, nil
}

// validEntry separates private templates, runtime files and non-served distribution material.
func validEntry(key, name string) bool {
	if !fs.ValidPath(key) || !fs.ValidPath(name) {
		return false
	}
	if key == "public/palettes.json" {
		return name == "themes/palettes.json"
	}
	for _, kind := range []string{"public", "admin"} {
		if strings.HasPrefix(key, kind+"/templates/") {
			// Shared partials (one head for every page) may be mounted in both template sets.
			own := strings.HasPrefix(name, "templates/"+kind+"/") || strings.HasPrefix(name, "templates/shared/")
			return own && path.Ext(name) == ".html"
		}
	}
	if strings.HasPrefix(key, "public/static/") {
		for _, dir := range []string{"css/", "js/", "icons/", "public/", "preview/", "fonts/"} {
			if strings.HasPrefix(name, dir) {
				if path.Base(name) == "LICENSE" {
					return path.Base(key) == "LICENSE"
				}
				if path.Ext(key) != path.Ext(name) {
					return false
				}
				ext := path.Ext(name)
				return strings.Contains("|.css|.js|.svg|.json|.png|.jpg|.webp|.ico|.webmanifest|.woff2|", "|"+ext+"|")
			}
		}
	}
	return false
}

// confinedFS rejects internal symlinks as well as root escapes, so public/private categories cannot alias.
type confinedFS struct{ root *os.Root }

func (f confinedFS) Open(name string) (fs.File, error) {
	if !fs.ValidPath(name) {
		return nil, fs.ErrInvalid
	}
	prefix := ""
	for _, part := range strings.Split(name, "/") {
		prefix = path.Join(prefix, part)
		st, err := f.root.Lstat(prefix)
		if err != nil {
			return nil, err
		}
		if st.Mode()&fs.ModeSymlink != 0 {
			return nil, fmt.Errorf("asset symlink refused: %s", prefix)
		}
	}
	return f.root.Open(name)
}
