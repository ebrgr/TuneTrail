package models

import (
	"os"
	"regexp"
	"strings"
	"testing"
)

// TestAvatarPalette checks the palette format, uniqueness, order and the contrast of every entry (ISSUE-202).
func TestAvatarPalette(t *testing.T) {
	format := regexp.MustCompile(`^#[0-9a-f]{6}$`)
	seen := map[string]bool{}
	for _, c := range AvatarPalette {
		if !format.MatchString(c.Hex) {
			t.Errorf("%s is not a lower-case #rrggbb colour", c.Hex)
		}
		if seen[c.Hex] || seen[c.Key] {
			t.Errorf("duplicate palette entry %s/%s", c.Hex, c.Key)
		}
		seen[c.Hex], seen[c.Key] = true, true
		if r := AvatarContrast(c.Hex); r < 4.5 {
			t.Errorf("%s has contrast %.2f with %s, below 4.5", c.Hex, r, AvatarTextColor(c.Hex))
		}
	}
	if AvatarPalette[0].Hex != "#000000" {
		t.Error("black must be the first entry")
	}
}

// TestAvatarColorValidation accepts only exact palette entries, never a free-form value.
func TestAvatarColorValidation(t *testing.T) {
	for _, ok := range []string{"#000000", "#ffffff", "#1d4ed8"} {
		if !ValidAvatarColor(ok) {
			t.Errorf("%s should be accepted", ok)
		}
	}
	for _, bad := range []string{"", "#000", "#00000", "#0000000", "#FFFFFF", "000000", "red", "#123456", "url(x)", "#000000;x", "#000000\" onload=\"x"} {
		if ValidAvatarColor(bad) {
			t.Errorf("%q must be refused", bad)
		}
	}
}

// TestAvatarTextColorFollowsBackground gives light text on dark backgrounds and dark text on light ones.
func TestAvatarTextColorFollowsBackground(t *testing.T) {
	if AvatarTextColor("#000000") != AvatarTextLight || AvatarTextColor("#1d4ed8") != AvatarTextLight {
		t.Error("dark backgrounds need the light text")
	}
	if AvatarTextColor("#ffffff") != AvatarTextDark || AvatarTextColor("#d1d5db") != AvatarTextDark {
		t.Error("light backgrounds need the dark text")
	}
}

// TestAvatarPaletteMatchesCSS keeps the stylesheet in step with the palette: one rule per entry with the same text colour.
func TestAvatarPaletteMatchesCSS(t *testing.T) {
	raw, err := os.ReadFile("../../assets/css/ui/components/profile.css")
	if err != nil {
		t.Fatal(err)
	}
	css := string(raw)
	for _, c := range AvatarPalette {
		rule := `.avatar[data-avatar-color="` + c.Hex + `"] { background: ` + c.Hex + `; color: ` + AvatarTextColor(c.Hex) + ` }`
		if !strings.Contains(css, rule) {
			t.Errorf("profile.css lacks the rule %q", rule)
		}
	}
	if got := strings.Count(css, `.avatar[data-avatar-color=`); got != len(AvatarPalette) {
		t.Errorf("profile.css has %d colour rules, the palette has %d", got, len(AvatarPalette))
	}
}
