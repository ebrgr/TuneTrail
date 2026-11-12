package cfg

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
)

// Origin describes the source of an effective configuration value.
type Origin string

const (
	OriginDefault     Origin = "default"
	OriginTOML        Origin = "toml"
	OriginFile        Origin = "file"
	OriginEnvironment Origin = "environment"
)

// Document retains the original bytes so future editing can preserve comments.
type Document struct {
	Path    string
	Raw     []byte
	Present bool
	Origins map[string]Origin
}

type tomlDocument struct {
	Version *int `toml:"schema_version"`
	Logging struct {
		Level *string `toml:"level"`
	} `toml:"logging"`
	MusicBrainz struct {
		URL       *string `toml:"url"`
		RateLimit *int    `toml:"rate_limit"`
		Disabled  *bool   `toml:"disabled"`
	} `toml:"musicbrainz"`
	Images struct {
		DisableDeezer *bool   `toml:"disable_deezer"`
		DisableCAA    *bool   `toml:"disable_cover_art_archive"`
		LastFMKey     *string `toml:"lastfm_api_key"`
	} `toml:"images"`
	Subsonic struct {
		URL    *string `toml:"url"`
		Params *string `toml:"params"`
	} `toml:"subsonic"`
	Time struct {
		ForceTZ *string `toml:"force_tz"`
	} `toml:"time"`
}

var managedDefaults = map[string]string{
	LOG_LEVEL_ENV: "info", MUSICBRAINZ_URL_ENV: defaultMusicBrainzUrl,
	MUSICBRAINZ_RATE_LIMIT_ENV: "1", DISABLE_MUSICBRAINZ_ENV: "false",
	DISABLE_DEEZER_ENV: "false", DISABLE_COVER_ART_ARCHIVE_ENV: "false",
	LASTFM_API_KEY_ENV: "", SUBSONIC_URL_ENV: "", SUBSONIC_PARAMS_ENV: "", FORCE_TZ: "",
}

// envReader keeps empty _FILE content distinct from a missing option.
type envValue struct {
	value  string
	origin Origin
}
type envReader struct {
	getenv func(string) string
	err    error
	cache  map[string]envValue
}

// get snapshots each source once to keep discovery and loading consistent.
func (r *envReader) get(key string) (string, Origin) {
	if value, ok := r.cache[key]; ok {
		return value.value, value.origin
	}
	if r.cache == nil {
		r.cache = make(map[string]envValue)
	}
	result := envValue{origin: OriginDefault}
	if value := r.getenv(key); value != "" {
		result = envValue{value: value, origin: OriginEnvironment}
	} else if filename := r.getenv(key + "_FILE"); filename != "" {
		data, err := os.ReadFile(filename)
		result.origin = OriginFile
		if err != nil {
			if r.err == nil {
				r.err = fmt.Errorf("cannot read %s_FILE: %w", key, err)
			}
		} else {
			result.value = strings.TrimSpace(string(data))
		}
	}
	r.cache[key] = result
	return result.value, result.origin
}

func readTOML(path string) (*tomlDocument, []byte, bool, error) {
	// Share bounded regular-file reading with the persistence service.
	raw, present, err := readRawSettings(path)
	if err != nil {
		return nil, nil, false, fmt.Errorf("read conf.toml: %w", err)
	}
	if !present {
		return nil, nil, false, nil
	}
	parsed, err := decodeTOML(raw)
	if err != nil {
		return nil, nil, false, err
	}
	return parsed, raw, true, nil
}

// decodeTOML validates a document without altering its original bytes.
func decodeTOML(raw []byte) (*tomlDocument, error) {
	if len(raw) > 64<<10 {
		return nil, fmt.Errorf("conf.toml exceeds 64 KiB")
	}
	var doc tomlDocument
	md, err := toml.Decode(string(raw), &doc)
	if err != nil {
		return nil, fmt.Errorf("invalid conf.toml: %w", err)
	}
	if doc.Version == nil || *doc.Version != 1 {
		return nil, fmt.Errorf("conf.toml: unsupported or missing schema_version")
	}
	if unknown := md.Undecoded(); len(unknown) != 0 {
		return nil, fmt.Errorf("conf.toml: unknown option %s", unknown[0])
	}
	return &doc, nil
}

// documentValues extracts only explicitly provided TOML fields.
func documentValues(d *tomlDocument) map[string]string {
	out := make(map[string]string)
	putString := func(k string, v *string) {
		if v != nil {
			out[k] = *v
		}
	}
	putBool := func(k string, v *bool) {
		if v != nil {
			out[k] = strconv.FormatBool(*v)
		}
	}
	putString(LOG_LEVEL_ENV, d.Logging.Level)
	putString(MUSICBRAINZ_URL_ENV, d.MusicBrainz.URL)
	if d.MusicBrainz.RateLimit != nil {
		out[MUSICBRAINZ_RATE_LIMIT_ENV] = strconv.Itoa(*d.MusicBrainz.RateLimit)
	}
	putBool(DISABLE_MUSICBRAINZ_ENV, d.MusicBrainz.Disabled)
	putBool(DISABLE_DEEZER_ENV, d.Images.DisableDeezer)
	putBool(DISABLE_COVER_ART_ARCHIVE_ENV, d.Images.DisableCAA)
	putString(LASTFM_API_KEY_ENV, d.Images.LastFMKey)
	putString(SUBSONIC_URL_ENV, d.Subsonic.URL)
	putString(SUBSONIC_PARAMS_ENV, d.Subsonic.Params)
	putString(FORCE_TZ, d.Time.ForceTZ)
	return out
}

func baseURL(value, name string) (string, error) {
	u, err := url.Parse(value)
	if err != nil || u == nil || (u.Scheme != "http" && u.Scheme != "https") || u.Hostname() == "" || u.User != nil || u.RawQuery != "" || u.ForceQuery || u.Fragment != "" || u.Opaque != "" {
		return "", fmt.Errorf("%s must be an HTTP(S) base URL without userinfo, query or fragment", name)
	}
	return strings.TrimRight(value, "/"), nil
}

// validateManaged validates and normalizes a complete managed configuration.
func validateManaged(v map[string]string) error {
	switch v[LOG_LEVEL_ENV] {
	case "debug", "info", "warn", "error", "fatal":
	default:
		return fmt.Errorf("invalid logging.level")
	}
	var err error
	v[MUSICBRAINZ_URL_ENV], err = baseURL(v[MUSICBRAINZ_URL_ENV], "musicbrainz.url")
	if err != nil {
		return err
	}
	rate, err := strconv.Atoi(v[MUSICBRAINZ_RATE_LIMIT_ENV])
	if err != nil || rate < 1 {
		return fmt.Errorf("musicbrainz.rate_limit must be a positive integer")
	}
	u, _ := url.Parse(v[MUSICBRAINZ_URL_ENV])
	port := u.Port()
	if strings.EqualFold(u.Hostname(), "musicbrainz.org") && (port == "" || (u.Scheme == "https" && port == "443") || (u.Scheme == "http" && port == "80")) && rate != 1 {
		return fmt.Errorf("official MusicBrainz requires rate_limit 1")
	}
	for _, key := range []string{DISABLE_MUSICBRAINZ_ENV, DISABLE_DEEZER_ENV, DISABLE_COVER_ART_ARCHIVE_ENV} {
		if v[key] != "true" && v[key] != "false" {
			return fmt.Errorf("%s must be true or false", key)
		}
	}
	surl, params := v[SUBSONIC_URL_ENV], v[SUBSONIC_PARAMS_ENV]
	if (surl == "") != (params == "") {
		return fmt.Errorf("subsonic.url and subsonic.params must both be set or empty")
	}
	if surl != "" {
		v[SUBSONIC_URL_ENV], err = baseURL(surl, "subsonic.url")
		if err != nil {
			return err
		}
		q, parseErr := url.ParseQuery(params)
		if parseErr != nil {
			return fmt.Errorf("invalid subsonic.params")
		}
		for k, values := range q {
			// Preserve non-reserved provider extensions while rejecting ambiguous query fields.
			switch strings.ToLower(k) {
			case "f", "v", "c", "query", "id", "artistcount", "songcount", "albumcount":
				return fmt.Errorf("invalid subsonic.params")
			}
			if k == "" || len(values) != 1 || values[0] == "" {
				return fmt.Errorf("invalid subsonic.params")
			}
		}
		if q.Get("u") == "" || !((q.Get("p") != "" && q.Get("t") == "" && q.Get("s") == "") || (q.Get("p") == "" && q.Get("t") != "" && q.Get("s") != "")) {
			return fmt.Errorf("subsonic.params requires u and either p or t+s")
		}
	}
	if tz := v[FORCE_TZ]; tz != "" {
		if _, err := time.LoadLocation(tz); err != nil {
			return fmt.Errorf("invalid time.force_tz")
		}
	}
	return nil
}

// prepareTOML loads both the stored and effective configurations without writing files.
func prepareTOML(r *envReader) (*Document, map[string]string, error) {
	dir, _ := r.get(CONFIG_DIR_ENV)
	if r.err != nil {
		return nil, nil, r.err
	}
	if dir == "" {
		dir = "/etc/koito"
	}
	path := filepath.Join(dir, "conf.toml")
	parsed, raw, present, err := readTOML(path)
	if err != nil {
		return nil, nil, err
	}
	doc := &Document{Path: path, Raw: raw, Present: present, Origins: make(map[string]Origin)}
	if !present {
		for k := range managedDefaults {
			_, source := r.get(k)
			if r.err != nil {
				return nil, nil, r.err
			}
			doc.Origins[k] = source
		}
		return doc, nil, nil
	}
	stored := make(map[string]string, len(managedDefaults))
	for k, v := range managedDefaults {
		stored[k] = v
		doc.Origins[k] = OriginDefault
	}
	for k, v := range documentValues(parsed) {
		stored[k] = v
		doc.Origins[k] = OriginTOML
	}
	if err := validateManaged(stored); err != nil {
		return nil, nil, fmt.Errorf("conf.toml: %w", err)
	}
	effective := make(map[string]string, len(stored))
	for k, v := range stored {
		effective[k] = v
	}
	for k := range managedDefaults {
		value, source := r.get(k)
		if r.err != nil {
			return nil, nil, r.err
		}
		if source != OriginDefault {
			effective[k] = value
			doc.Origins[k] = source
		}
	}
	if err := validateManaged(effective); err != nil {
		return nil, nil, fmt.Errorf("effective configuration: %w", err)
	}
	return doc, effective, nil
}

// ConfigurationDocument returns a copy of the startup document and its origins.
func ConfigurationDocument() Document {
	lock.RLock()
	defer lock.RUnlock()
	d := startupDocument
	d.Raw = append([]byte(nil), d.Raw...)
	d.Origins = make(map[string]Origin, len(startupDocument.Origins))
	for k, v := range startupDocument.Origins {
		d.Origins[k] = v
	}
	return d
}

// ValidateSubsonicDraft validates only the supplied Subsonic fields without saving them.
func ValidateSubsonicDraft(raw, params string) (string, error) {
	if raw == "" || params == "" {
		return "", fmt.Errorf("Subsonic URL and parameters are required")
	}
	values := make(map[string]string, len(managedDefaults))
	for k, v := range managedDefaults {
		values[k] = v
	}
	values[SUBSONIC_URL_ENV] = raw
	values[SUBSONIC_PARAMS_ENV] = params
	if err := validateManaged(values); err != nil {
		return "", err
	}
	return values[SUBSONIC_URL_ENV], nil
}
