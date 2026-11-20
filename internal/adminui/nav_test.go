package adminui

import (
	"github.com/ebrgr/TuneTrail/internal/admin"
	"net/http/httptest"
	"testing"

	"github.com/ebrgr/TuneTrail/internal/models"
)

// TestAdminNavAdaptsProfile checks the administration's adapter to the shared sidebar.
func TestAdminNavAdaptsProfile(t *testing.T) {
	h := &Handler{}
	r := httptest.NewRequest("GET", "/admin", nil)
	if nav := h.navigation(r, admin.Identity{}, nil, ""); nav.Kind != "admin" || nav.Profile != nil || nav.AdminBadge {
		t.Fatalf("no profile: %+v", nav)
	}
	image, icon := "abc", "radio"
	nav := h.navigation(r, admin.Identity{}, &models.ListeningProfile{DisplayName: "Mara", Slug: "mara", AvatarImageID: &image, AvatarIcon: &icon}, "settings")
	if nav.AdminBadge || nav.Version == "" {
		t.Fatalf("admin nav: %+v", nav)
	}
	if p := nav.Profile; p == nil || p.Name != "Mara" || p.Slug != "mara" || p.Image != "abc" || p.Icon != "radio" {
		t.Fatalf("profile: %+v", nav.Profile)
	}
}
