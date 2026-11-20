package adminui

import (
	"context"
	"github.com/ebrgr/TuneTrail/internal/admin"
	"testing"

	"github.com/ebrgr/TuneTrail/internal/models"
)

// TestAdminNavAdaptsProfile checks the administration's adapter to the shared sidebar.
func TestAdminNavAdaptsProfile(t *testing.T) {
	h := &Handler{}
	if nav := h.navigation(context.Background(), admin.Identity{}, nil, ""); nav.Kind != "admin" || nav.Profile != nil || nav.AdminBadge {
		t.Fatalf("no profile: %+v", nav)
	}
	image, icon := "abc", "radio"
	nav := h.navigation(context.Background(), admin.Identity{}, &models.ListeningProfile{DisplayName: "Mara", Slug: "mara", AvatarImageID: &image, AvatarIcon: &icon}, "settings")
	if nav.AdminBadge || nav.Version == "" {
		t.Fatalf("admin nav: %+v", nav)
	}
	if p := nav.Profile; p == nil || p.Name != "Mara" || p.Slug != "mara" || p.Image != "abc" || p.Icon != "radio" {
		t.Fatalf("profile: %+v", nav.Profile)
	}
}
