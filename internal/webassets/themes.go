package webassets

import (
	"bytes"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"sync"
)

// Palette is the fixed set of appearance colors bundled with the distribution.
type Palette map[string]string

var themeOrder = []string{"yuu", "varia", "midnight", "catppuccin", "autumn", "black", "wine", "pearl", "rosebud", "urim", "match", "lemon"}
var colorKeys = []string{"bg", "bgSecondary", "bgTertiary", "fg", "fgSecondary", "fgTertiary", "primary", "primaryDim", "accent", "accentDim", "error", "warning", "info", "success"}
var colorValue = regexp.MustCompile(`^(#[0-9a-fA-F]{3}|#[0-9a-fA-F]{6}|#[0-9a-fA-F]{8})$`)

// ThemeNames returns a private copy of the canonical presentation order.
func ThemeNames() []string { return append([]string(nil), themeOrder...) }

// renderThemes validates and generates deterministic CSS once, including code-dependent formatting.
func renderThemes(data []byte) ([]byte, error) {
	var palettes map[string]Palette
	if err := json.Unmarshal(data, &palettes); err != nil {
		return nil, fmt.Errorf("palettes: %w", err)
	}
	if len(palettes) != len(themeOrder) {
		return nil, fmt.Errorf("invalid palette count")
	}
	keys := append([]string(nil), colorKeys...)
	sort.Strings(keys)
	var out bytes.Buffer
	for _, name := range themeOrder {
		colors := palettes[name]
		if len(colors) != len(colorKeys) {
			return nil, fmt.Errorf("invalid palette %s", name)
		}
		fmt.Fprintf(&out, "[data-theme=%q]{", name)
		for _, key := range keys {
			if !colorValue.MatchString(colors[key]) {
				return nil, fmt.Errorf("invalid palette color %s/%s", name, key)
			}
			var css strings.Builder
			for _, c := range key {
				if c >= 'A' && c <= 'Z' {
					css.WriteByte('-')
					c += 32
				}
				css.WriteRune(c)
			}
			fmt.Fprintf(&out, "--color-%s:%s;", css.String(), colors[key])
		}
		fmt.Fprintln(&out, "}")
	}
	return out.Bytes(), nil
}

// ThemeOption is one entry of the theme grid shared by public pages and administration.
type ThemeOption struct {
	Name   string
	Colors Palette
}

// themeCache keeps the parsed theme grid of the current generation; a new snapshot
// (different Hash) rebuilds it once instead of on every rendered page.
var themeCache struct {
	sync.Mutex
	hash    string
	options []ThemeOption
}

// ThemeOptions lists the bundled themes in display order for the shared theme panel (ISSUE-110).
func ThemeOptions() ([]ThemeOption, error) {
	s, err := Current()
	if err != nil {
		return nil, err
	}
	themeCache.Lock()
	defer themeCache.Unlock()
	if themeCache.hash == s.Hash {
		return themeCache.options, nil
	}
	var palettes map[string]Palette
	if err := json.Unmarshal(s.files["public/palettes.json"], &palettes); err != nil {
		return nil, fmt.Errorf("palettes: %w", err)
	}
	options := make([]ThemeOption, 0, len(themeOrder))
	for _, name := range themeOrder {
		if len(palettes[name]) != len(colorKeys) {
			return nil, fmt.Errorf("invalid or missing palette %s", name)
		}
		options = append(options, ThemeOption{Name: name, Colors: palettes[name]})
	}
	themeCache.hash, themeCache.options = s.Hash, options
	return options, nil
}
