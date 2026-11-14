package webui

import (
	"github.com/ebrgr/TuneTrail/internal/version"
	"github.com/ebrgr/TuneTrail/internal/webassets"
	"net/http"
)

// Assets serves versioned representations of the immutable generation.
func Assets() http.Handler { return webassets.Handler("", true) }

// PublicAssets preserves fixed legacy URLs without assigning them an immutable lifetime.
func PublicAssets() http.Handler { return webassets.Handler("public/", false) }

// LegacyAssets serves the fixed favicon alias with mandatory revalidation.
func LegacyAssets() http.Handler { return webassets.Handler("", false) }

// uiGeneration includes code and snapshot identity, without invalidating independent asset URLs.
func uiGeneration() (string, error) {
	s, err := webassets.Current()
	if err != nil {
		return "", err
	}
	return version.Current + "-" + s.Hash, nil
}
