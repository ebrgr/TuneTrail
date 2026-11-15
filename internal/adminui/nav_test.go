package adminui

import (
	"testing"

	"github.com/ebrgr/TuneTrail/internal/models"
)

// TestAdminNavAdaptsProfile checks the administration's adapter to the shared sidebar.
func TestAdminNavAdaptsProfile(t *testing.T) {
	if nav := adminNav(nil); nav.Kind != "admin" || nav.Profile != nil {
		t.Fatalf("no profile: %+v", nav)
	}
	image, icon := "abc", "radio"
	nav := adminNav(&models.ListeningProfile{DisplayName: "Mara", Slug: "mara", AvatarImageID: &image, AvatarIcon: &icon})
	if p := nav.Profile; p == nil || p.Name != "Mara" || p.Slug != "mara" || p.Image != "abc" || p.Icon != "radio" {
		t.Fatalf("profile: %+v", nav.Profile)
	}
}
