// Package webui renders public Go pages and the isolated roadmap-1 preview.
// NewPreview serves only fictional data; production Pages receives a read store.
package webui

import (
	"bytes"
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/ebrgr/TuneTrail/internal/i18n"
	"github.com/ebrgr/TuneTrail/internal/version"
)

//go:embed templates/*.html static palettes.json
var files embed.FS

type Palette map[string]string

var themeNames = []string{"yuu", "varia", "midnight", "catppuccin", "autumn", "black", "wine", "pearl", "rosebud", "urim", "match", "lemon"}
var palettes = func() map[string]Palette {
	data, _ := files.ReadFile("palettes.json")
	var p map[string]Palette
	if err := json.Unmarshal(data, &p); err != nil {
		panic(err)
	}
	return p
}()

type Profile struct {
	ID    int    `json:"id"`
	Slug  string `json:"slug"`
	Name  string `json:"display_name"`
	Icon  string `json:"avatar_icon"`
	Image string `json:"avatar_image_id,omitempty"`
}
type Media struct {
	ID, Count            int
	Title, Artist, Image string
}
type Cell struct {
	Label   string
	Count   int
	Opacity string
	Future  bool
	Week    int
}
type ThemeOption struct {
	Name   string
	Colors Palette
}
type Page struct {
	Locale                        *i18n.View
	ForcedTimezone                bool
	Live                          bool
	SwitchLinks                   map[string]string
	Profile                       *Profile
	Profiles                      []Profile
	Artists, Albums, Tracks       []Media
	Cells                         []Cell
	Themes                        []ThemeOption
	Period, State, Theme, Version string
	Selection                     bool
}

var profiles = []Profile{{1, "demo", "Mara Silva", "headphones", ""}, {2, "luna", "Luna Paz", "", ""}, {3, "iara", "Iara Costa", "", "preview-photo"}}

func findProfile(slug string) *Profile {
	for i := range profiles {
		if profiles[i].Slug == slug {
			return &profiles[i]
		}
	}
	return nil
}

func fixtureMedia(kind string) []Media {
	titles := []string{"Estação Azul", "Maré Baixa", "Órbita", "花と光", "Entre Rios"}
	if kind == "album" {
		titles = []string{"Horizonte", "Depois da Chuva", "Satélite", "夜の色", "Caminhos"}
	}
	if kind == "track" {
		titles = []string{"Primeira Luz", "Mar Aberto", "Órbita", "花と光", "Entre Rios", "Um Lugar", "Ao Longe", "Brisa", "Dança Lenta", "Noite Clara"}
	}
	out := make([]Media, len(titles))
	for i, title := range titles {
		out[i] = Media{i + 1, 28 - i*2, title, []string{"Estação Azul", "Maré Baixa", "Órbita", "花と光", "Entre Rios"}[i%5], fmt.Sprintf("/preview-assets/cover/%d.svg", i%5)}
	}
	return out
}

func makePage(r *http.Request) Page {
	q := r.URL.Query()
	p := Page{Profiles: profiles, Artists: fixtureMedia("artist"), Albums: fixtureMedia("album"), Tracks: fixtureMedia("track"), Period: q.Get("period"), State: q.Get("state"), Theme: q.Get("theme"), Version: version.Current, Selection: r.URL.Path == "/"}
	if _, ok := palettes[p.Theme]; !ok {
		p.Theme = "yuu"
	}
	switch p.Period {
	case "day", "week", "month", "year", "all_time":
	default:
		p.Period = "week"
	}
	if !p.Selection {
		for i := range profiles {
			if r.URL.Path == "/"+profiles[i].Slug {
				p.Profile = &profiles[i]
			}
		}
	}
	if p.Profile != nil && p.Profile.Slug != "demo" {
		p.State = "empty"
	}
	if p.State == "empty" || p.State == "error" {
		p.Artists = nil
		p.Albums = nil
		p.Tracks = nil
	}
	if p.State == "long" {
		for _, list := range [][]Media{p.Artists, p.Albums, p.Tracks} {
			for i := range list {
				list[i].Title = "Uma canção com um título muito longo — São Paulo, 東京 e outras histórias"
			}
		}
	}
	if p.State == "broken" {
		for _, list := range [][]Media{p.Artists, p.Albums, p.Tracks} {
			for i := range list {
				list[i].Image = "/preview-assets/missing.svg"
			}
		}
	}
	for _, name := range themeNames {
		p.Themes = append(p.Themes, ThemeOption{name, palettes[name]})
	}
	loc := time.FixedZone("America/Sao_Paulo", -3*3600)
	now := time.Now().In(loc)
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)
	offset := int(today.Weekday())
	if q.Get("weekstart") == "mon" {
		offset = (offset + 6) % 7
	}
	start := today.AddDate(0, 0, -offset-35*7)
	for i := 0; i < 36*7; i++ {
		date := start.AddDate(0, 0, i)
		count := activityCount(date)
		if p.State == "empty" {
			count = 0
		}
		opacity := 0.1 + float64(count)/100*0.9
		p.Cells = append(p.Cells, Cell{date.Format("02/01/2006") + fmt.Sprintf(" %d plays", count), count, fmt.Sprintf("%.3f", opacity), date.After(today), i / 7})
	}
	return p
}
func activityCount(t time.Time) int {
	n := t.YearDay()
	if n%5 == 0 {
		return 0
	}
	return (n*17)%100 + 1
}
func cssKey(s string) string {
	var b strings.Builder
	for _, c := range s {
		if c >= 'A' && c <= 'Z' {
			b.WriteByte('-')
			c += 32
		}
		b.WriteRune(c)
	}
	return b.String()
}

func NewPreview() (http.Handler, error) {
	tmpl, err := loadTemplates()
	if err != nil {
		return nil, err
	}

	mux := http.NewServeMux()
	bindAssets(mux)
	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" && findProfile(strings.TrimPrefix(r.URL.Path, "/")) == nil {
			http.NotFound(w, r)
			return
		}
		p := makePage(r)
		var b bytes.Buffer
		localized, err := localizedTemplates(tmpl, nil)
		if err == nil {
			err = localized.ExecuteTemplate(&b, "page.html", p)
		}
		if err != nil {
			http.Error(w, "Unable to render preview", 500)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		_, _ = b.WriteTo(w)
	})
	return mux, nil
}
func bindAssets(mux *http.ServeMux) {
	mux.HandleFunc("GET /image/preview-photo/128x128.webp", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/webp")
		data, _ := files.ReadFile("static/avatar.webp")
		_, _ = w.Write(data)
	})
	sub, _ := fs.Sub(files, "static")
	mux.Handle("GET /preview-assets/", http.StripPrefix("/preview-assets/", http.FileServer(http.FS(sub))))
	mux.HandleFunc("GET /preview-assets/themes.json", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		data, _ := files.ReadFile("palettes.json")
		_, _ = w.Write(data)
	})
	mux.HandleFunc("GET /preview-assets/themes.css", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/css; charset=utf-8")
		for _, name := range themeNames {
			fmt.Fprintf(w, "[data-theme=%q]{", name)
			keys := make([]string, 0, 14)
			for key := range palettes[name] {
				keys = append(keys, key)
			}
			sort.Strings(keys)
			for _, key := range keys {
				fmt.Fprintf(w, "--color-%s:%s;", cssKey(key), palettes[name][key])
			}
			fmt.Fprintln(w, "}")
		}
	})
	mux.HandleFunc("GET /preview-assets/cover/{id}", func(w http.ResponseWriter, r *http.Request) {
		n, err := strconv.Atoi(strings.TrimSuffix(r.PathValue("id"), ".svg"))
		if err != nil || n < 0 || n > 4 {
			http.NotFound(w, r)
			return
		}
		pairs := [][2]string{{"#2b5364", "#ecb283"}, {"#652c4e", "#e49b76"}, {"#263c52", "#80c6bd"}, {"#46436c", "#d5b2ca"}, {"#486252", "#b9cb85"}}
		w.Header().Set("Content-Type", "image/svg+xml")
		fmt.Fprintf(w, `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 350 350"><defs><linearGradient id="g" x2="1" y2="1"><stop stop-color="%s"/><stop offset="1" stop-color="%s"/></linearGradient></defs><path fill="url(#g)" d="M0 0h350v350H0z"/><circle cx="250" cy="100" r="100" fill="#fff" opacity=".16"/><path d="M0 240Q90 90 200 240T400 210V350H0Z" fill="#102635" opacity=".4"/><path d="M0 280Q150 130 350 280" fill="none" stroke="#fff" stroke-width="2" opacity=".35"/></svg>`, pairs[n][0], pairs[n][1])
	})
}
