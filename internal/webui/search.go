package webui

import (
	"bytes"
	"context"
	"html/template"
	"log"
	"net/http"
	"net/url"
	"strings"
	"unicode/utf8"

	"github.com/ebrgr/TuneTrail/internal/listening"
)

func (h *Pages) search(ctx context.Context, slug, q string) ([]Item, error) {
	result, e := listening.ReadSearch(ctx, h.store, q)
	if e != nil {
		return nil, e
	}
	items := []Item{}
	for _, a := range result.Artists {
		items = append(items, artistItem(slug, a, 0))
	}
	for _, a := range result.Albums {
		items = append(items, albumItem(slug, a, 0))
	}
	for _, a := range result.Tracks {
		items = append(items, trackItem(slug, a, 0))
	}
	return items, nil
}

// Instant search (ISSUE-107): the search dialog asks for a small results fragment while
// the user types. The full search page stays the no-JavaScript path and "see all" target.
const (
	searchPreviewLimit = 5 // items per kind in the fragment
	searchMinRunes     = 2
)

// SearchGroup is one kind of result (artists, albums or tracks) in the fragment.
type SearchGroup struct {
	Kind  string
	Items []Item
}

// SearchFragment is the data of the "partial/search-results" template.
type SearchFragment struct {
	Query, AllURL string
	Groups        []SearchGroup
	Total         int
	TooShort      bool
}

// searchFragment groups the search results by kind and keeps the first few of each.
func (h *Pages) searchFragment(ctx context.Context, slug, q string) (SearchFragment, error) {
	f := SearchFragment{Query: q, AllURL: "/" + slug + "/search?" + url.Values{"q": {q}}.Encode()}
	if utf8.RuneCountInString(q) < searchMinRunes {
		f.TooShort = true
		return f, nil
	}
	result, err := listening.ReadSearch(ctx, h.store, q)
	if err != nil {
		return f, err
	}
	add := func(kind string, items []Item) {
		f.Total += len(items)
		if len(items) > searchPreviewLimit {
			items = items[:searchPreviewLimit]
		}
		if len(items) > 0 {
			f.Groups = append(f.Groups, SearchGroup{Kind: kind, Items: items})
		}
	}
	var artists, albums, tracks []Item
	for _, a := range result.Artists {
		artists = append(artists, artistItem(slug, a, 0))
	}
	for _, a := range result.Albums {
		albums = append(albums, albumItem(slug, a, 0))
	}
	for _, a := range result.Tracks {
		tracks = append(tracks, trackItem(slug, a, 0))
	}
	add("artists", artists)
	add("albums", albums)
	add("tracks", tracks)
	return f, nil
}

// fragmentRequested reports whether a fetch from the search dialog asked for the fragment;
// a navigation to a URL with partial=1 (typed, bookmarked or cross-site) does not qualify.
func fragmentRequested(r *http.Request) bool {
	return r.Header.Get("X-TuneTrail-Partial") == "1" &&
		r.Header.Get("Sec-Fetch-Mode") != "navigate" && r.Header.Get("Sec-Fetch-Dest") != "document"
}

// serveSearchFragment renders only the results list, after the profile was resolved.
func (h *Pages) serveSearchFragment(w http.ResponseWriter, r *http.Request, p LivePage) {
	w.Header().Set("Cache-Control", "no-store")
	if !fragmentRequested(r) {
		q := r.URL.Query()
		q.Del("partial")
		// Built from the resolved profile, never from the request path.
		http.Redirect(w, r, (&url.URL{Path: "/" + p.Profile.Slug + "/search", RawQuery: q.Encode()}).RequestURI(), http.StatusSeeOther)
		return
	}
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	if utf8.RuneCountInString(q) > 200 {
		http.Error(w, "Search query too long", http.StatusBadRequest)
		return
	}
	f, err := h.searchFragment(r.Context(), p.Profile.Slug, q)
	var b bytes.Buffer
	if err == nil {
		var tmpl *template.Template
		if tmpl, err = localizedTemplates(h.templates, p.Locale); err == nil {
			err = tmpl.ExecuteTemplate(&b, "partial/search-results", f)
		}
	}
	if err != nil {
		log.Printf("HTML search fragment failed: %v", err)
		http.Error(w, "Unable to search", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "private, no-cache")
	w.Header().Add("Vary", "Cookie")
	w.Header().Add("Vary", "X-TuneTrail-Partial")
	w.Header().Set("X-Robots-Tag", "noindex")
	_, _ = b.WriteTo(w)
}
