package webassets

import (
	"bytes"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"
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
