package webassets

import (
	"html/template"
	"strings"
)

// NavProfile is the profile shown by the shared sidebar, whatever page model it came from.
type NavProfile struct {
	Name, Slug, Image, Icon string
}

// Nav is the data for the shared "partial/sidebar" template (ISSUE-122).
type Nav struct {
	Kind    string // live, preview or admin: selects dialogs or links for the same controls
	Profile *NavProfile
	// Second sidebar (ISSUE-123): administration links for a valid session, open on
	// administration pages, with the current item and the version shown in About.
	Admin   bool
	Open    bool
	Current string // account, profiles or settings
	Version string
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
