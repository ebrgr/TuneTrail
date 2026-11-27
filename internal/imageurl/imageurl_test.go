package imageurl

import (
	"context"
	"testing"

	"github.com/ebrgr/TuneTrail/internal/db"
	"github.com/ebrgr/TuneTrail/internal/models"
	"github.com/google/uuid"
)

// The expected values were recorded from catalog.BuildImageList before the
// function moved here (ISSUE-231); they are the public image routes.
func TestList(t *testing.T) {
	id := uuid.MustParse("0b6e1c52-9f4a-4d8e-a1c3-7e5b2a90fd14")
	list := func(base string) models.ImageList {
		return models.ImageList{
			XS:     base + "64x64.webp",
			Small:  base + "128x128.webp",
			Medium: base + "300x300.webp",
			Large:  base + "640x640.webp",
			XL:     base + "1000x1000.webp",
		}
	}
	for _, tc := range []struct {
		name    string
		profile int32
		id      *uuid.UUID
		want    models.ImageList
	}{
		{"no profile", 0, &id, list("/image/0b6e1c52-9f4a-4d8e-a1c3-7e5b2a90fd14/")},
		{"profile", 7, &id, list("/image/profile/7/0b6e1c52-9f4a-4d8e-a1c3-7e5b2a90fd14/")},
		{"missing image", 0, nil, list("/image/00000000-0000-0000-0000-000000000000/")},
		{"missing image in a profile", 7, nil, list("/image/profile/7/00000000-0000-0000-0000-000000000000/")},
		{"negative profile", -1, &id, list("/image/0b6e1c52-9f4a-4d8e-a1c3-7e5b2a90fd14/")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			if tc.profile != 0 {
				ctx = db.WithProfile(ctx, tc.profile)
			}
			if got := List(ctx, tc.id); got != tc.want {
				t.Errorf("List = %+v, want %+v", got, tc.want)
			}
		})
	}
}
