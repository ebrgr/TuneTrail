package webui

import (
	"fmt"
	"io/fs"
	"net/http"
	"path"
	"sort"
	"strings"
)

// PublicFiles keeps legacy public URLs available without a frontend directory.
func PublicFiles() http.FileSystem {
	sub, err := fs.Sub(files, "static/public")
	if err != nil {
		panic(err)
	}
	return staticFiles{http.FS(sub)}
}

// Assets exposes only presentation assets, never demo data or fake API routes.
func Assets() http.Handler {
	if err := ensureAssets(); err != nil {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.Error(w, "web assets unavailable", 500) })
	}
	mux := http.NewServeMux()
	sub, _ := fs.Sub(files, "static")
	mux.HandleFunc("GET /themes.json", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		data, _ := files.ReadFile("palettes.json")
		_, _ = w.Write(data)
	})
	mux.HandleFunc("GET /themes.css", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/css; charset=utf-8")
		for _, name := range themeNames {
			fmt.Fprintf(w, "[data-theme=%q]{", name)
			keys := []string{}
			for k := range palettes[name] {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			for _, k := range keys {
				fmt.Fprintf(w, "--color-%s:%s;", cssKey(k), palettes[name][k])
			}
			fmt.Fprintln(w, "}")
		}
	})
	mux.Handle("GET /", StaticHandler(staticFiles{http.FS(sub)}))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		mux.ServeHTTP(w, r)
	})
}

// staticFiles exposes only approved leaf files, never directories, licenses or template source.
type staticFiles struct{ http.FileSystem }

func (s staticFiles) Open(name string) (http.File, error) {
	if strings.Contains(name, "..") || strings.HasSuffix(name, "/") || strings.Contains(name, "\\") {
		return nil, fs.ErrNotExist
	}
	if _, ok := staticType(name); !ok {
		return nil, fs.ErrNotExist
	}
	f, err := s.FileSystem.Open(name)
	if err != nil {
		return nil, err
	}
	st, err := f.Stat()
	if err != nil || st.IsDir() {
		f.Close()
		return nil, fs.ErrNotExist
	}
	return f, nil
}

// StaticHandler keeps explicit MIME types and zero modification times for snapshot responses.
func StaticHandler(source http.FileSystem) http.Handler {
	server := http.FileServer(source)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if contentType, ok := staticType(r.URL.Path); ok {
			w.Header().Set("Content-Type", contentType)
		}
		w.Header().Set("X-Content-Type-Options", "nosniff")
		server.ServeHTTP(w, r)
	})
}

// staticType fixes MIME and extension policy independently of the host MIME database.
func staticType(name string) (string, bool) {
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
	case ".webp":
		return "image/webp", true
	case ".jpg":
		return "image/jpeg", true
	case ".ico":
		return "image/x-icon", true
	case ".webmanifest":
		return "application/manifest+json", true
	}
	return "", false
}
