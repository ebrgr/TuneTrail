// Package imageurl builds the image routes returned with catalog entities. It is
// a leaf package so that the storage adapter can use it without importing the
// catalog (ISSUE-231).
package imageurl

import (
	"context"
	"fmt"

	"github.com/ebrgr/TuneTrail/imagecache"
	"github.com/ebrgr/TuneTrail/internal/db"
	"github.com/ebrgr/TuneTrail/internal/models"
	"github.com/google/uuid"
)

// List adds a stable profile scope to listening-derived image URLs. A nil image
// ID yields the routes of the placeholder image.
func List(ctx context.Context, imageid *uuid.UUID) models.ImageList {
	if imageid == nil {
		imageid = &uuid.Nil
	}
	base := "/image/"
	if id := db.ProfileID(ctx); id > 0 {
		base = fmt.Sprintf("/image/profile/%d/", id)
	}
	base += imageid.String() + "/"
	return models.ImageList{
		XS:     base + imagecache.ImageSizeXS.String() + ".webp",
		Small:  base + imagecache.ImageSizeSmall.String() + ".webp",
		Medium: base + imagecache.ImageSizeMedium.String() + ".webp",
		Large:  base + imagecache.ImageSizeLarge.String() + ".webp",
		XL:     base + imagecache.ImageSizeXL.String() + ".webp",
	}
}
