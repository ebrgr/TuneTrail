package models

import (
	"math"
	"strconv"
)

// AvatarColor is one entry of the fixed avatar background palette (ISSUE-202). Key names the
// translation (admin.avatar_color.<key>) and Hex is the only value stored and sent to the page.
type AvatarColor struct{ Hex, Key string }

// AvatarPalette is the complete set of accepted avatar backgrounds, black first. Nothing outside
// this list is stored, so no free-form value can reach the CSS. assets/css/ui/components/profile.css
// carries one rule per entry, kept in step by TestAvatarPaletteMatchesCSS.
var AvatarPalette = []AvatarColor{
	{"#000000", "black"}, {"#374151", "graphite"}, {"#b91c1c", "red"}, {"#c2410c", "orange"},
	{"#a16207", "amber"}, {"#15803d", "green"}, {"#0f766e", "teal"}, {"#1d4ed8", "blue"},
	{"#6d28d9", "violet"}, {"#be185d", "pink"}, {"#d1d5db", "light-gray"}, {"#ffffff", "white"},
}

// ValidAvatarColor reports whether hex is exactly one palette entry.
func ValidAvatarColor(hex string) bool {
	for _, c := range AvatarPalette {
		if c.Hex == hex {
			return true
		}
	}
	return false
}

// Light and dark foregrounds used on top of the palette; the icon and the initials follow the text colour.
const (
	AvatarTextLight = "#ffffff"
	AvatarTextDark  = "#111111"
)

// AvatarTextColor returns the foreground with the higher WCAG contrast against the background, so a
// dark background gets light initials and icon and a light one gets dark ones.
func AvatarTextColor(background string) string {
	if contrastRatio(background, AvatarTextLight) >= contrastRatio(background, AvatarTextDark) {
		return AvatarTextLight
	}
	return AvatarTextDark
}

// AvatarContrast is the WCAG contrast ratio between a palette background and the text colour chosen for it.
func AvatarContrast(background string) float64 {
	return contrastRatio(background, AvatarTextColor(background))
}

func contrastRatio(a, b string) float64 {
	la, lb := relativeLuminance(a), relativeLuminance(b)
	if la < lb {
		la, lb = lb, la
	}
	return (la + 0.05) / (lb + 0.05)
}

// relativeLuminance implements the WCAG 2.x definition for a #rrggbb colour.
func relativeLuminance(hex string) float64 {
	if len(hex) != 7 || hex[0] != '#' {
		return 0
	}
	channel := func(from int) float64 {
		v, err := strconv.ParseUint(hex[from:from+2], 16, 8)
		if err != nil {
			return 0
		}
		c := float64(v) / 255
		if c <= 0.03928 {
			return c / 12.92
		}
		return math.Pow((c+0.055)/1.055, 2.4)
	}
	return 0.2126*channel(1) + 0.7152*channel(3) + 0.0722*channel(5)
}
