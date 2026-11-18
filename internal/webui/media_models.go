package webui

import "time"

// MediaPage contains public catalog metadata and statistics of the resolved profile.
type MediaPage struct {
	Item
	Kind, Type, MusicBrainzURL    string
	Accent                        string
	TimeListened, MinutesListened int64
	FirstListen                   *time.Time
	Album                         *Link
	TopTracks                     Card
	Albums                        []Item
	AlbumsURL, HistoryURL         string
	AlbumsMore                    bool
	Interest                      *InterestGraph
}
