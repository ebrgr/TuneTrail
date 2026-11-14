package webassets

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/fs"
	"net/url"
	"path"
	"strings"
)

// Asset is an immutable representation; Reader gives each request its own cursor.
type Asset struct {
	data        []byte
	Digest      string
	ContentType string
}

// Reader never exposes the backing slice for mutation.
func (a Asset) Reader() *bytes.Reader { return bytes.NewReader(a.data) }

// Asset looks up a public logical key, never a disk path.
func (s *Snapshot) Asset(name string) (Asset, bool) {
	a, ok := s.assets[name]
	return a, ok
}

// AssetURL only versions catalog entries under a server-selected mount prefix.
func (s *Snapshot) AssetURL(prefix, name string) (string, error) {
	a, ok := s.Asset(name)
	if !ok {
		return "", fmt.Errorf("unknown public asset %q", name)
	}
	return strings.TrimSuffix(prefix, "/") + "/" + name + "?v=" + a.Digest, nil
}

// AssetFunc binds the immutable process generation before templates are parsed.
func AssetFunc(prefix string) (func(string) (string, error), error) {
	s, err := Current()
	if err != nil {
		return nil, err
	}
	return func(name string) (string, error) { return s.AssetURL(prefix, name) }, nil
}

// digest hashes the exact representation, rather than its filename or generation.
func digest(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

// buildCatalog prepares all derived bytes before any request can observe this generation.
func (s *Snapshot) buildCatalog() error {
	s.assets = make(map[string]Asset)
	for key, data := range s.files {
		if !strings.HasPrefix(key, "public/static/") {
			continue
		}
		name := strings.TrimPrefix(key, "public/static/")
		mime, ok := ContentType(name)
		if !ok {
			continue
		} // License files stay in the private distribution snapshot.
		if !fs.ValidPath(name) || strings.ContainsAny(name, "\\?#%") || name == "themes.css" || name == "themes.json" {
			return fmt.Errorf("invalid public asset key %q", name)
		}
		s.assets[name] = Asset{data: data, Digest: digest(data), ContentType: mime}
	}
	themes, err := renderThemes(s.files["public/palettes.json"])
	if err != nil {
		return err
	}
	s.assets["themes.json"] = Asset{data: s.files["public/palettes.json"], Digest: digest(s.files["public/palettes.json"]), ContentType: "application/json"}
	s.assets["themes.css"] = Asset{data: themes, Digest: digest(themes), ContentType: "text/css; charset=utf-8"}
	extra := 0
	for name, a := range s.assets {
		if path.Ext(name) != ".css" {
			continue
		}
		data, err := s.rewriteCSS(name, a.data)
		if err != nil {
			return fmt.Errorf("asset %s: %w", name, err)
		}
		if len(data) > maxFileBytes {
			return fmt.Errorf("derived asset %s exceeds size limit", name)
		}
		extra += len(data)
		if s.Bytes+extra > maxTotalBytes {
			return fmt.Errorf("derived asset generation exceeds total size limit")
		}
		a.data, a.Digest = data, digest(data)
		s.assets[name] = a
	}
	s.Bytes += extra
	return nil
}

// ContentType is the public allowlist; source templates, licenses and arbitrary files are excluded.
func ContentType(name string) (string, bool) {
	switch path.Ext(name) {
	case ".css":
		return "text/css; charset=utf-8", true
	case ".js":
		return "text/javascript; charset=utf-8", true
	case ".svg":
		return "image/svg+xml", true
	case ".woff2":
		return "font/woff2", true
	case ".json":
		return "application/json", true
	case ".png":
		return "image/png", true
	case ".jpg":
		return "image/jpeg", true
	case ".webp":
		return "image/webp", true
	case ".ico":
		return "image/x-icon", true
	case ".webmanifest":
		return "application/manifest+json", true
	}
	return "", false
}

// cssURL resolves against the published URL, keeping identical CSS under both mounts.
func (s *Snapshot) cssURL(cssName, raw string) (string, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return "", fmt.Errorf("invalid CSS URL: %w", err)
	}
	if u.IsAbs() || u.Host != "" || strings.HasPrefix(raw, "#") {
		return raw, nil
	}
	if u.RawQuery != "" || u.ForceQuery {
		return "", fmt.Errorf("CSS asset query must be generated: %q", raw)
	}
	name := u.Path
	if strings.HasPrefix(name, "/web-assets/") {
		name = strings.TrimPrefix(name, "/web-assets/")
	} else if strings.HasPrefix(name, "/preview-assets/") {
		name = strings.TrimPrefix(name, "/preview-assets/")
	} else if strings.HasPrefix(name, "/") {
		return "", fmt.Errorf("CSS URL outside public assets: %q", raw)
	} else {
		name = path.Join(path.Dir(cssName), name)
	}
	if path.Ext(name) == ".css" {
		return "", fmt.Errorf("CSS dependencies require concatenation, not imports: %q", raw)
	}
	a, ok := s.Asset(name)
	if !ok {
		return "", fmt.Errorf("missing CSS asset %q", name)
	}
	// Logical asset paths use slash separators on every platform.
	base := strings.Split(strings.Trim(path.Dir(cssName), "/"), "/")
	target := strings.Split(name, "/")
	if len(base) == 1 && base[0] == "." {
		base = nil
	}
	for len(base) > 0 && len(target) > 0 && base[0] == target[0] {
		base = base[1:]
		target = target[1:]
	}
	relative := strings.Repeat("../", len(base)) + strings.Join(target, "/")
	return (&url.URL{Path: relative, RawQuery: "v=" + a.Digest, Fragment: u.Fragment}).String(), nil
}
