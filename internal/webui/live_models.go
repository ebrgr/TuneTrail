package webui

import (
	"github.com/ebrgr/TuneTrail/internal/db"
	"github.com/ebrgr/TuneTrail/internal/listening"
	"github.com/ebrgr/TuneTrail/internal/models"
	"net/url"
	"time"
)

type Link struct{ Label, URL string }
type Item struct {
	ID          int32
	Rank, Count int64
	Title, URL  string
	Image       models.ImageList
	Artists     []Link
	Various     bool
}
type ListenView struct {
	Item     Item
	Time     time.Time
	Unix     int64
	Relative string
}
type Card struct {
	Title, URL string
	Items      []Item
	Error      bool
}
type Pager struct {
	Current        int
	Count          int64
	Previous, Next string
}
type LivePage struct {
	CurrentURL string
	Page
	Title, Section, Error, DateRange, Timezone string
	Status                                     int
	Stats                                      listening.Stats
	Cards                                      []Card
	Items                                      []Item
	Listens                                    []ListenView
	Playing                                    *Item
	Activity                                   *ActivityGrid
	Errors                                     map[string]bool
	Pager                                      Pager
	Options                                    db.GetItemsOpts
	Params                                     url.Values
	Periods                                    []Link
	Years                                      []int
	Next                                       string
	HistoryMore                                bool
	WeekStart                                  int
	Search                                     string
	Media                                      *MediaPage
	Rewind                                     *RewindPage
}

func publicProfile(p models.ListeningProfile) Profile {
	v := Profile{ID: int(p.ID), Slug: p.Slug, Name: p.DisplayName}
	if p.AvatarIcon != nil {
		v.Icon = *p.AvatarIcon
	}
	if p.AvatarImageID != nil {
		v.Image = *p.AvatarImageID
	}
	return v
}
func itemLinks(slug string, artists []models.SimpleArtist) []Link {
	links := make([]Link, 0, len(artists))
	for _, a := range artists {
		links = append(links, Link{a.Name, mediaURL(slug, "artist", a.ID)})
	}
	return links
}
func artistItem(slug string, a *models.Artist, rank int64) Item {
	return Item{ID: a.ID, Rank: rank, Count: a.ListenCount, Title: a.Name, URL: mediaURL(slug, "artist", a.ID), Image: a.Image}
}
func albumItem(slug string, a *models.Album, rank int64) Item {
	return Item{ID: a.ID, Rank: rank, Count: a.ListenCount, Title: a.Title, URL: mediaURL(slug, "album", a.ID), Image: a.Image, Artists: itemLinks(slug, a.Artists), Various: a.VariousArtists}
}
func trackItem(slug string, a *models.Track, rank int64) Item {
	return Item{ID: a.ID, Rank: rank, Count: a.ListenCount, Title: a.Title, URL: mediaURL(slug, "track", a.ID), Image: a.Image, Artists: itemLinks(slug, a.Artists)}
}
