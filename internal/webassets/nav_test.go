package webassets

import "testing"

// TestNavHelpers guards the shared sidebar helpers used by both template sets (ISSUE-122).
func TestNavHelpers(t *testing.T) {
	for in, want := range map[string]string{"Mara Silva": "MS", "luna": "L", "Ana Maria Costa": "AM", "": ""} {
		if got := Initials(in); got != want {
			t.Errorf("Initials(%q) = %q, want %q", in, got, want)
		}
	}
	if AvatarIcon("headphones") != "Headphones" || AvatarIcon("unknown") != "" || AvatarIcon("") != "" {
		t.Error("AvatarIcon must map only known avatar icons")
	}
}
