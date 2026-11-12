package cfg

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// settingField maps the supported environment names to their TOML location and type.
type settingField struct{ section, key, kind string }

var settingFields = map[string]settingField{
	LOG_LEVEL_ENV:                 {"logging", "level", "string"},
	MUSICBRAINZ_URL_ENV:           {"musicbrainz", "url", "string"},
	MUSICBRAINZ_RATE_LIMIT_ENV:    {"musicbrainz", "rate_limit", "int"},
	DISABLE_MUSICBRAINZ_ENV:       {"musicbrainz", "disabled", "bool"},
	DISABLE_DEEZER_ENV:            {"images", "disable_deezer", "bool"},
	DISABLE_COVER_ART_ARCHIVE_ENV: {"images", "disable_cover_art_archive", "bool"},
	LASTFM_API_KEY_ENV:            {"images", "lastfm_api_key", "string"},
	SUBSONIC_URL_ENV:              {"subsonic", "url", "string"},
	SUBSONIC_PARAMS_ENV:           {"subsonic", "params", "string"},
	FORCE_TZ:                      {"time", "force_tz", "string"},
}
var settingSections = []string{"logging", "musicbrainz", "images", "subsonic", "time"}

type byteEdit struct {
	start, end int
	value      []byte
}

// encodeSetting produces TOML-compatible values without altering the supplied text.
func encodeSetting(field settingField, value string) (string, error) {
	switch field.kind {
	case "string":
		encoded, err := json.Marshal(value)
		return string(encoded), err
	case "bool":
		if value != "true" && value != "false" {
			return "", fmt.Errorf("%w: %s must be true or false", ErrInvalidSettings, field.key)
		}
		return value, nil
	case "int":
		n, err := strconv.Atoi(value)
		if err != nil {
			return "", fmt.Errorf("%w: %s must be an integer", ErrInvalidSettings, field.key)
		}
		return strconv.Itoa(n), nil
	default:
		return "", ErrInvalidSettings
	}
}

// simpleName rejects quoted and dotted keys so unsupported TOML cannot be rewritten accidentally.
func simpleName(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '_' || r == '-') {
			return false
		}
	}
	return true
}

// valueBounds recognizes only complete single-line scalar values and retains inline comments.
func valueBounds(raw []byte, start, end int) (int, int, error) {
	for start < end && (raw[start] == ' ' || raw[start] == '\t') {
		start++
	}
	if start >= end {
		return 0, 0, ErrUnsupportedSettingsFormat
	}
	i := start
	if raw[i] == '"' || raw[i] == '\'' {
		quote := raw[i]
		if i+2 < end && raw[i+1] == quote && raw[i+2] == quote {
			return 0, 0, ErrUnsupportedSettingsFormat
		}
		i++
		closed := false
		for i < end {
			if quote == '"' && raw[i] == '\\' {
				i += 2
				continue
			}
			if raw[i] == quote {
				i++
				closed = true
				break
			}
			i++
		}
		if !closed {
			return 0, 0, ErrUnsupportedSettingsFormat
		}
	} else {
		for i < end && raw[i] != '#' && raw[i] != ' ' && raw[i] != '\t' && raw[i] != '\r' {
			i++
		}
	}
	valueEnd := i
	for i < end && (raw[i] == ' ' || raw[i] == '\t' || raw[i] == '\r') {
		i++
	}
	if i < end && raw[i] != '#' {
		return 0, 0, ErrUnsupportedSettingsFormat
	}
	return start, valueEnd, nil
}

// editSettingsDocument modifies only scalar value spans; unsupported valid TOML is refused.
func editSettingsDocument(raw []byte, updates map[string]string) ([]byte, error) {
	if len(raw) == 0 {
		raw = []byte("schema_version = 1\n")
	}
	newline := "\n"
	if bytes.Contains(raw, []byte("\r\n")) {
		newline = "\r\n"
	}
	edits := []byteEdit{}
	found := map[string]bool{}
	sections := map[string]int{}
	section := ""
	offset := 0
	for offset < len(raw) {
		end := offset
		for end < len(raw) && raw[end] != '\n' {
			end++
		}
		line := string(raw[offset:end])
		trimmed := strings.TrimSpace(line)
		if trimmed != "" && !strings.HasPrefix(trimmed, "#") {
			if strings.HasPrefix(trimmed, "[") {
				close := strings.IndexByte(trimmed, ']')
				if close < 0 {
					return nil, ErrUnsupportedSettingsFormat
				}
				name := strings.TrimSpace(trimmed[1:close])
				tail := strings.TrimSpace(trimmed[close+1:])
				if !simpleName(name) || (tail != "" && !strings.HasPrefix(tail, "#")) {
					return nil, ErrUnsupportedSettingsFormat
				}
				valid := false
				for _, candidate := range settingSections {
					if name == candidate {
						valid = true
						break
					}
				}
				if !valid {
					return nil, ErrUnsupportedSettingsFormat
				}
				if section != "" {
					sections[section] = offset
				}
				section = name
				if _, ok := sections[name]; ok {
					return nil, ErrUnsupportedSettingsFormat
				}
				sections[name] = -1
			} else {
				eq := strings.IndexByte(line, '=')
				if eq < 0 {
					return nil, ErrUnsupportedSettingsFormat
				}
				key := strings.TrimSpace(line[:eq])
				if !simpleName(key) {
					return nil, ErrUnsupportedSettingsFormat
				}
				if section == "" && key != "schema_version" {
					return nil, ErrUnsupportedSettingsFormat
				}
				fieldName := ""
				for name, field := range settingFields {
					if field.section == section && field.key == key {
						fieldName = name
						break
					}
				}
				if section != "" && fieldName == "" {
					return nil, ErrUnsupportedSettingsFormat
				}
				a, b, err := valueBounds(raw, offset+eq+1, end)
				if err != nil {
					return nil, err
				}
				if fieldName != "" {
					found[fieldName] = true
					if value, ok := updates[fieldName]; ok {
						encoded, err := encodeSetting(settingFields[fieldName], value)
						if err != nil {
							return nil, err
						}
						edits = append(edits, byteEdit{a, b, []byte(encoded)})
					}
				}
			}
		}
		offset = end
		if offset < len(raw) {
			offset++
		}
	}
	if section != "" {
		sections[section] = len(raw)
	}
	insertions := map[int]string{}
	for _, existingPass := range []bool{true, false} {
		for _, name := range settingSections {
			keys := []string{}
			for field, value := range updates {
				if settingFields[field].section == name && !found[field] {
					encoded, err := encodeSetting(settingFields[field], value)
					if err != nil {
						return nil, err
					}
					keys = append(keys, settingFields[field].key+" = "+encoded)
				}
			}
			if len(keys) == 0 {
				continue
			}
			sort.Strings(keys)
			pos, exists := sections[name]
			// Existing sections at EOF must precede any newly appended table headers.
			if exists != existingPass {
				continue
			}
			if !exists {
				pos = len(raw)
			}
			prefix := ""
			if pos > 0 && raw[pos-1] != '\n' {
				prefix = newline
			}
			if !exists {
				prefix += "[" + name + "]" + newline
			}
			// Multiple absent sections share the same insertion point; avoid duplicate newlines.
			if _, ok := insertions[pos]; ok && pos > 0 && raw[pos-1] != '\n' {
				prefix = strings.TrimPrefix(prefix, newline)
			}
			insertions[pos] += prefix + strings.Join(keys, newline) + newline
		}
	}
	for pos, value := range insertions {
		edits = append(edits, byteEdit{pos, pos, []byte(value)})
	}
	sort.SliceStable(edits, func(i, j int) bool { return edits[i].start > edits[j].start })
	out := append([]byte(nil), raw...)
	for _, edit := range edits {
		out = append(append(append([]byte{}, out[:edit.start]...), edit.value...), out[edit.end:]...)
	}
	return out, nil
}
