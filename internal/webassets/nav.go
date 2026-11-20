package webassets

import (
	"html/template"
	"net/url"
	"strings"
)

// NavProfile is the profile shown by the shared sidebar, whatever page model it came from.
type NavProfile struct {
	ID                      int64 // carried by administration links so pages keep the profile
	Name, Slug, Image, Icon string
}

// Nav is the data for the shared "partial/sidebar" template (ISSUE-122).
type Nav struct {
	NavAccess
	Period, Timezone string // ISSUE-071: carry the selection in navigation URLs, without browser storage.
	PeriodLinks      []NavPeriod
	Section          string // Current listening page, when rendered by the public handler.
	Kind             string // live, preview or admin: selects dialogs or links for the same controls
	Profile          *NavProfile
	Version          string
}

// Icon inlines a bundled, reviewed Lucide SVG; never catalog or user HTML.
func Icon(name string) template.HTML {
	s, err := Current()
	if err != nil {
		return ""
	}
	data, err := s.read("public/static/icons/" + name + ".svg")
	if err != nil {
		return ""
	}
	return template.HTML(data)
}

// AvatarIcon maps a stored avatar icon to its bundled SVG name; unknown values show initials.
func AvatarIcon(name string) string {
	return avatarIcons[name]
}

var avatarIcons = map[string]string{"music": "Music2", "headphones": "Headphones", "disc": "Disc3", "radio": "Radio", "user": "UserRound"}

// Initials renders up to two initials, one per word, for avatars without an image or icon.
func Initials(name string) string {
	var letters []rune
	for _, word := range strings.Fields(name) {
		letters = append(letters, []rune(word)[0])
		if len(letters) == 2 {
			break
		}
	}
	return strings.ToUpper(string(letters))
}

// NavPeriod is a translated server-rendered period destination.
type NavPeriod struct {
	URL, Label, Fallback string
	Selected             bool
}

// Query preserves only the period and timezone when changing navigation destinations.
func (n Nav) Query() string {
	q := url.Values{}
	if n.Period != "" {
		q.Set("period", n.Period)
	}
	if n.Timezone != "" {
		q.Set("tz", n.Timezone)
	}
	if len(q) == 0 {
		return ""
	}
	return "?" + q.Encode()
}

// DefaultPeriods supplies a selector on pages without their own period links.
func (n Nav) DefaultPeriods() []NavPeriod {
	if n.Profile == nil {
		return nil
	}
	var links []NavPeriod
	for _, period := range []string{"day", "week", "month", "year", "all_time"} {
		label := period
		if period == "all_time" {
			label = "All"
		}
		dest := n
		dest.Period = period
		links = append(links, NavPeriod{URL: "/" + n.Profile.Slug + dest.Query(), Label: "period." + period, Fallback: label, Selected: n.Period == period})
	}
	return links
}
