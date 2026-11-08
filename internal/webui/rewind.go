package webui

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/ebrgr/TuneTrail/internal/summary"
	"github.com/ebrgr/TuneTrail/internal/webquery"
)

func (h *Pages) serveRewind(w http.ResponseWriter, r *http.Request, p LivePage) {
	now := h.options.Now()
	selection, err := webquery.Rewind(r, now)
	if err != nil {
		h.fail(w, p, http.StatusBadRequest)
		return
	}
	loc := selection.Timeframe.Timezone
	now = now.In(loc)
	title := fmt.Sprintf("Your %d Rewind", selection.Year)
	if selection.Month > 0 {
		title = fmt.Sprintf("Your %s %d Rewind", time.Month(selection.Month), selection.Year)
	}
	stats, err := summary.GenerateSummary(r.Context(), h.store, int32(p.Profile.ID), selection.Timeframe, title)
	if err != nil {
		log.Printf("HTML rewind summary failed: %v", err)
		h.fail(w, p, http.StatusInternalServerError)
		return
	}
	firstUnix, firstErr := h.store.GetFirstListenUnix(r.Context())
	if firstErr != nil && !errors.Is(firstErr, sql.ErrNoRows) {
		log.Printf("HTML rewind first activity failed: %v", firstErr)
		h.fail(w, p, http.StatusInternalServerError)
		return
	}
	var first *time.Time
	if firstErr == nil {
		t := time.Unix(firstUnix, 0).In(loc)
		if !t.After(now) {
			first = &t
		}
	}
	page := &RewindPage{
		Title: title, Year: selection.Year, Month: selection.Month, ShowTime: selection.ShowTime,
		InProgress: selection.InProgress, Timezone: loc.String(), Stats: stats,
		RangeLabel: stats.IntervalFrom.In(loc).Format("02 Jan 2006 15:04") + " – " + stats.IntervalTo.In(loc).Format("02 Jan 2006 15:04"),
		Empty:      len(stats.TopArtists) == 0 || len(stats.TopAlbums) == 0 || len(stats.TopTracks) == 0,
	}
	artistGroup, albumGroup, trackGroup := RewindGroup{Title: "Top Artist"}, RewindGroup{Title: "Top Album"}, RewindGroup{Title: "Top Track"}
	for _, entry := range stats.TopArtists {
		artistGroup.Items = append(artistGroup.Items, RewindItem{artistItem(p.Profile.Slug, entry.Item, entry.Rank), entry.Item.TimeListened / 60})
	}
	for _, entry := range stats.TopAlbums {
		albumGroup.Items = append(albumGroup.Items, RewindItem{albumItem(p.Profile.Slug, entry.Item, entry.Rank), entry.Item.TimeListened / 60})
	}
	for _, entry := range stats.TopTracks {
		trackGroup.Items = append(trackGroup.Items, RewindItem{trackItem(p.Profile.Slug, entry.Item, entry.Rank), entry.Item.TimeListened / 60})
	}
	page.Groups = []RewindGroup{artistGroup, albumGroup, trackGroup}
	buildRewindChoices(page, selection.Query, "/"+p.Profile.Slug+"/rewind", now, first)
	p.Rewind, p.Title, p.Timezone, p.Params = page, title, loc.String(), selection.Query
	h.render(w, p)
}

// Each destination is fully represented in a link, so all controls work without JS.
func buildRewindChoices(p *RewindPage, base url.Values, path string, now time.Time, first *time.Time) {
	startYear := now.Year() - 10
	if first != nil {
		startYear = first.Year()
	}
	if startYear < 1 {
		startYear = 1
	}
	if startYear > now.Year() {
		startYear = now.Year()
	}
	periodURL := func(year, month int) string {
		q := cloneQuery(base)
		q.Set("year", strconv.Itoa(year))
		q.Del("month")
		if month > 0 {
			q.Set("month", strconv.Itoa(month))
		}
		return queryURL(path, q)
	}
	validMonths := func(year int) (int, int) {
		firstMonth, lastMonth := 1, 12
		if first != nil && year == first.Year() {
			firstMonth = int(first.Month())
		}
		if year == now.Year() {
			lastMonth = int(now.Month()) - 1
		}
		return firstMonth, lastMonth
	}
	p.Monthly = RewindChoice{Label: "Monthly", Current: p.Month > 0, Disabled: true}
	p.Yearly = RewindChoice{Label: "Yearly", Current: p.Month == 0, Disabled: true}
	for year := now.Year(); year >= startYear; year-- {
		minMonth, maxMonth := validMonths(year)
		if p.Monthly.Disabled && minMonth <= maxMonth {
			m := maxMonth
			if p.Year == year && p.Month > 0 {
				m = max(minMonth, min(p.Month, maxMonth))
			}
			p.Monthly.URL, p.Monthly.Disabled = periodURL(year, m), false
		}
		if p.Yearly.Disabled && year < now.Year() {
			p.Yearly.URL, p.Yearly.Disabled = periodURL(year, 0), false
		}
		if p.Month == 0 {
			if year < now.Year() {
				p.Years = append(p.Years, RewindChoice{Label: strconv.Itoa(year), URL: periodURL(year, 0), Current: p.Year == year})
			}
		} else if minMonth <= maxMonth {
			m := maxMonth
			if p.Month > 0 {
				m = max(minMonth, min(p.Month, maxMonth))
			}
			p.Years = append(p.Years, RewindChoice{Label: strconv.Itoa(year), URL: periodURL(year, m), Current: p.Year == year})
		}
	}
	if p.Year >= startYear && p.Year <= now.Year() {
		minMonth, maxMonth := validMonths(p.Year)
		if minMonth <= maxMonth {
			m := maxMonth
			if p.Month > 0 {
				m = max(minMonth, min(p.Month, maxMonth))
			}
			p.Monthly.URL, p.Monthly.Disabled = periodURL(p.Year, m), false
		}
		if p.Year < now.Year() {
			p.Yearly.URL, p.Yearly.Disabled = periodURL(p.Year, 0), false
		}
	}
	if p.Month > 0 {
		minMonth, maxMonth := validMonths(p.Year)
		for month := 1; month <= 12; month++ {
			enabled := p.Year >= startYear && p.Year <= now.Year() && month >= minMonth && month <= maxMonth
			choice := RewindChoice{Label: time.Month(month).String()[:3], Current: p.Month == month, Disabled: !enabled}
			if enabled {
				choice.URL = periodURL(p.Year, month)
			}
			p.Months = append(p.Months, choice)
		}
	}
}
