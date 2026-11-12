package webui

import "github.com/ebrgr/TuneTrail/internal/summary"

type RewindChoice struct {
	Label, URL        string
	Current, Disabled bool
}
type RewindItem struct {
	Item
	Minutes int64
}
type RewindGroup struct {
	Title string
	Items []RewindItem
}
type RewindPage struct {
	Title, RangeLabel, Timezone string
	Year, Month                 int
	ShowTime, InProgress, Empty bool
	Stats                       *summary.Summary
	Groups                      []RewindGroup
	Monthly, Yearly             RewindChoice
	PreviousYear, NextYear      RewindChoice
	Years, Months               []RewindChoice
}
