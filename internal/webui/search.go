package webui

import (
	"context"
	"github.com/ebrgr/TuneTrail/internal/listening"
)

func (h *Pages) search(ctx context.Context, slug, q string) ([]Item, error) {
	result, e := listening.ReadSearch(ctx, h.store, q)
	if e != nil {
		return nil, e
	}
	items := []Item{}
	for _, a := range result.Artists {
		items = append(items, artistItem(slug, a, 0))
	}
	for _, a := range result.Albums {
		items = append(items, albumItem(slug, a, 0))
	}
	for _, a := range result.Tracks {
		items = append(items, trackItem(slug, a, 0))
	}
	return items, nil
}
