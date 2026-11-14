package cfg

// Environment variable names: TUNETRAIL_* with KOITO_* as a deprecated fallback
// (ISSUE-111, phase B1). The constants in cfg.go keep their KOITO_* values as
// internal canonical keys, so upstream merges touching cfg.go stay simple; the
// external TUNETRAIL_* name is derived by replacing the prefix.
//
// Precedence for a key K, for each prefix P in {TUNETRAIL, KOITO}: P_K when not
// empty, otherwise P_K_FILE (trimmed) when its name is set. Then:
//
//	TUNETRAIL only          -> TUNETRAIL value
//	KOITO only              -> KOITO value + deprecation notice (rename)
//	both, same value        -> TUNETRAIL value + deprecation notice (remove old)
//	both, different values  -> startup error naming both variables, never values
//	unreadable _FILE        -> startup error, even if the other prefix resolves

import (
	"fmt"
	"os"
	"sort"
	"strings"
)

const (
	legacyPrefix  = "KOITO_"
	currentPrefix = "TUNETRAIL_"
)

// aliasedKeys are the active settings that also accept the TUNETRAIL_* name.
// WEB_UI_ENV and DATABASE_URL_ENV only reject obsolete values, SQLITE_ENABLED only
// warns and ENABLE_FULL_IMAGE_CACHE_ENV is never read: they keep the KOITO_* name.
var aliasedKeys = []string{
	BIND_ADDR_ENV, LISTEN_PORT_ENV, ENABLE_STRUCTURED_LOGGING_ENV, LOG_LEVEL_ENV,
	MUSICBRAINZ_URL_ENV, MUSICBRAINZ_RATE_LIMIT_ENV, ENABLE_LBZ_RELAY_ENV,
	LBZ_RELAY_URL_ENV, LBZ_RELAY_TOKEN_ENV, CONFIG_DIR_ENV, DEFAULT_USERNAME_ENV,
	DEFAULT_PASSWORD_ENV, DEFAULT_THEME_ENV, DISABLE_DEEZER_ENV,
	DISABLE_COVER_ART_ARCHIVE_ENV, DISABLE_MUSICBRAINZ_ENV, SUBSONIC_URL_ENV,
	SUBSONIC_PARAMS_ENV, LASTFM_API_KEY_ENV, SKIP_IMPORT_ENV, ALLOWED_HOSTS_ENV,
	CORS_ORIGINS_ENV, DISABLE_RATE_LIMIT_ENV, THROTTLE_IMPORTS_MS,
	IMPORT_BEFORE_UNIX_ENV, IMPORT_AFTER_UNIX_ENV, FETCH_IMAGES_DURING_IMPORT_ENV,
	ARTIST_SEPARATORS_ENV, LOGIN_GATE_ENV, FORCE_TZ,
}

var aliased = func() map[string]bool {
	m := make(map[string]bool, len(aliasedKeys))
	for _, k := range aliasedKeys {
		m[k] = true
	}
	return m
}()

// CurrentName returns the TUNETRAIL_* name of a canonical KOITO_* key, or the key
// itself when it has no alias.
func CurrentName(key string) string {
	if !aliased[key] {
		return key
	}
	return currentPrefix + strings.TrimPrefix(key, legacyPrefix)
}

// EnvironmentNames lists every variable name the configuration reads, in both
// prefixes and with _FILE variants, so tests can clear them from the environment.
func EnvironmentNames() []string {
	keys := append([]string{WEB_UI_ENV, DATABASE_URL_ENV, SQLITE_ENABLED, ENABLE_FULL_IMAGE_CACHE_ENV}, aliasedKeys...)
	var names []string
	for _, k := range keys {
		for _, name := range []string{k, CurrentName(k)} {
			names = append(names, name, name+"_FILE")
		}
	}
	sort.Strings(names)
	out := names[:0]
	for i, n := range names {
		if i == 0 || n != names[i-1] {
			out = append(out, n)
		}
	}
	return out
}

// prefixedValue is one prefix's view of a key.
type prefixedValue struct {
	value, source string
	origin        Origin
	present       bool
}

// readPrefixed reads NAME, then NAME_FILE, keeping empty file content distinct
// from a missing variable. Errors name the variable, never its content.
func readPrefixed(getenv func(string) string, name string) (prefixedValue, error) {
	if value := getenv(name); value != "" {
		return prefixedValue{value: value, source: name, origin: OriginEnvironment, present: true}, nil
	}
	filename := getenv(name + "_FILE")
	if filename == "" {
		return prefixedValue{origin: OriginDefault}, nil
	}
	data, err := os.ReadFile(filename)
	if err != nil {
		return prefixedValue{source: name + "_FILE", origin: OriginFile, present: true}, fmt.Errorf("cannot read %s_FILE: %w", name, err)
	}
	return prefixedValue{value: strings.TrimSpace(string(data)), source: name + "_FILE", origin: OriginFile, present: true}, nil
}

// resolveEnv applies the precedence table above to one canonical key and
// returns the effective value, the variable name that supplied it and an
// optional deprecation notice.
func resolveEnv(getenv func(string) string, key string) (envValue, string, error) {
	if !aliased[key] {
		v, err := readPrefixed(getenv, key)
		return envValue{value: v.value, origin: v.origin, source: v.source}, "", err
	}
	current, errCurrent := readPrefixed(getenv, CurrentName(key))
	legacy, errLegacy := readPrefixed(getenv, key)
	// Fail closed: an unreadable file is an error even if the other prefix resolves.
	if errCurrent != nil {
		return envValue{origin: current.origin, source: current.source}, "", errCurrent
	}
	if errLegacy != nil {
		return envValue{origin: legacy.origin, source: legacy.source}, "", errLegacy
	}
	switch {
	case current.present && legacy.present && current.value != legacy.value:
		return envValue{}, "", fmt.Errorf("conflicting configuration: %s and %s differ; remove %s", current.source, legacy.source, legacy.source)
	case current.present && legacy.present:
		return envValue{value: current.value, origin: current.origin, source: current.source},
			fmt.Sprintf("%s duplicates %s; remove the deprecated %s", legacy.source, current.source, legacy.source), nil
	case current.present:
		return envValue{value: current.value, origin: current.origin, source: current.source}, "", nil
	case legacy.present:
		return envValue{value: legacy.value, origin: legacy.origin, source: legacy.source},
			fmt.Sprintf("%s is deprecated; rename it to %s", legacy.source, strings.Replace(legacy.source, legacyPrefix, currentPrefix, 1)), nil
	}
	return envValue{origin: OriginDefault}, "", nil
}
