package webassets

import "fmt"

// Head is the data for the shared "partial/head" template.
type Head struct {
	Kind  string // live, preview or admin: selects page stylesheets and scripts
	Title string
}

// HeadFunc builds the shared head data; an unknown kind fails rendering instead of
// silently dropping a page's scripts.
func HeadFunc(kind, title string) (Head, error) {
	switch kind {
	case "live", "preview", "admin":
		return Head{Kind: kind, Title: title}, nil
	}
	return Head{}, fmt.Errorf("unknown head kind %q", kind)
}
