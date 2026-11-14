package adminui

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/ebrgr/TuneTrail/internal/admin"
	"github.com/ebrgr/TuneTrail/internal/adminweb"
	"github.com/ebrgr/TuneTrail/internal/cfg"
	"github.com/ebrgr/TuneTrail/internal/db"
	"github.com/ebrgr/TuneTrail/internal/i18n"
	"github.com/ebrgr/TuneTrail/internal/models"
)

type settingSpec struct{ Name, Label, Description, Kind string }
type settingOption struct {
	Value, Label string
	Selected     bool
}
type settingField struct {
	Name, Label, Description, Kind, Value, Saved, Effective, Active, Source, SavedSource string
	OverrideVariable                                                                     string
	Options                                                                              []settingOption
	Overridden                                                                           bool
}
type settingsView struct {
	Locale                                                   *i18n.View
	User                                                     *models.User
	Profile                                                  *models.ListeningProfile
	CSRF, Submission, Revision, Notice, Error, LocalTimezone string
	Fields                                                   []settingField
	RestartRequired, StateAvailable                          bool
}

var settingsFields = []settingSpec{
	{cfg.LOG_LEVEL_ENV, "Log level", "Controls the detail written to the application log.", "level"},
	{cfg.MUSICBRAINZ_URL_ENV, "MusicBrainz URL", "Base URL of the MusicBrainz server.", "text"},
	{cfg.MUSICBRAINZ_RATE_LIMIT_ENV, "MusicBrainz rate limit", "Requests per second; the official server requires 1.", "number"},
	{cfg.DISABLE_MUSICBRAINZ_ENV, "Disable MusicBrainz", "Disables MusicBrainz metadata lookups.", "bool"},
	{cfg.DISABLE_DEEZER_ENV, "Disable Deezer images", "Disables the Deezer artwork provider.", "bool"},
	{cfg.DISABLE_COVER_ART_ARCHIVE_ENV, "Disable Cover Art Archive", "Disables the Cover Art Archive provider.", "bool"},
	{cfg.LASTFM_API_KEY_ENV, "Last.fm API key", "Enables Last.fm artwork lookup when set.", "text"},
	{cfg.SUBSONIC_URL_ENV, "Subsonic URL", "Server used to find artists, albums and artwork.", "text"},
	{cfg.SUBSONIC_PARAMS_ENV, "Subsonic parameters", "Query parameters used to access the server, such as u, t and s.", "text"},
	{cfg.FORCE_TZ, "Application time zone", "IANA time zone, for example America/Sao_Paulo. Leave empty for automatic selection.", "text"},
}

// originLabel presents where the effective setting was obtained.
func originLabel(origin cfg.Origin) string {
	switch origin {
	case cfg.OriginEnvironment:
		return "Environment"
	case cfg.OriginFile:
		return "Environment file"
	case cfg.OriginTOML:
		return "Configuration file"
	default:
		return "Default"
	}
}

// originKey selects presentation labels without exposing override values.
func originKey(origin cfg.Origin) string {
	switch origin {
	case cfg.OriginEnvironment:
		return "environment"
	case cfg.OriginFile:
		return "environment_file"
	case cfg.OriginTOML:
		return "configuration_file"
	default:
		return "default"
	}
}
func displaySetting(value string) string {
	if value == "" {
		return "Not set"
	}
	return value
}

// renderSettings keeps submitted values and issues a fresh one-use ticket after each response.
func (h *Handler) renderSettings(w http.ResponseWriter, id admin.Identity, snap cfg.SettingsSnapshot, draft map[string]string, revision, notice, message string, status int) {
	token, err := h.csrf.SessionToken(id)
	if err != nil {
		h.fail(w, err)
		return
	}
	ticket, err := h.newSubmission(id.SessionID)
	if err != nil {
		h.fail(w, err)
		return
	}
	// Show the current process zone and offset without changing TZ or time.Local.
	now := time.Now()
	zone, _ := now.Zone()
	local := fmt.Sprintf("%s (%s, UTC%s)", now.Location(), zone, now.Format("-07:00"))
	v := settingsView{Locale: i18n.ViewOf(w), User: id.User, CSRF: token, Submission: ticket, Revision: revision, Notice: notice, Error: message, LocalTimezone: local, RestartRequired: snap.RestartRequired, StateAvailable: len(snap.Revision) == 64}
	for _, spec := range settingsFields {
		value := snap.Saved[spec.Name]
		if draft != nil {
			value = draft[spec.Name]
		}
		// Translation keys are prefix-neutral (admin.setting.LOG_LEVEL.label), ISSUE-111.
		key := "admin.setting." + strings.TrimPrefix(spec.Name, "KOITO_")
		field := settingField{Name: spec.Name, Label: adminText(v.Locale, key+".label", spec.Label), Description: adminText(v.Locale, key+".description", spec.Description), Kind: spec.Kind, Value: value, Saved: displaySetting(snap.Saved[spec.Name]), Effective: displaySetting(snap.Effective[spec.Name]), Active: displaySetting(snap.Active[spec.Name]), Source: adminText(v.Locale, "admin.text."+originKey(snap.Origins[spec.Name]), originLabel(snap.Origins[spec.Name])), SavedSource: adminText(v.Locale, "admin.text."+originKey(snap.SavedOrigins[spec.Name]), originLabel(snap.SavedOrigins[spec.Name]))}
		if snap.Saved[spec.Name] == "" {
			field.Saved = adminText(v.Locale, "admin.text.not_set", "Not set")
		}
		if snap.Effective[spec.Name] == "" {
			field.Effective = adminText(v.Locale, "admin.text.not_set", "Not set")
		}
		if snap.Active[spec.Name] == "" {
			field.Active = adminText(v.Locale, "admin.text.not_set", "Not set")
		}
		field.Overridden = snap.Origins[spec.Name] == cfg.OriginEnvironment || snap.Origins[spec.Name] == cfg.OriginFile
		if field.Overridden {
			field.Value = snap.Effective[spec.Name]
			value = field.Value
			// Show the variable actually used: TUNETRAIL_X, KOITO_X or their _FILE forms (ISSUE-111).
			field.OverrideVariable = snap.Sources[spec.Name]
			if field.OverrideVariable == "" {
				field.OverrideVariable = cfg.CurrentName(spec.Name)
				if snap.Origins[spec.Name] == cfg.OriginFile {
					field.OverrideVariable += "_FILE"
				}
			}
		}
		if spec.Kind == "bool" {
			field.Options = []settingOption{{"false", adminText(v.Locale, "admin.text.no", "No"), value == "false"}, {"true", adminText(v.Locale, "admin.text.yes", "Yes"), value == "true"}}
		}
		if spec.Kind == "level" {
			for _, level := range []string{"debug", "info", "warn", "error", "fatal"} {
				field.Options = append(field.Options, settingOption{level, level, value == level})
			}
		}
		// Preserve unexpected submitted select values instead of silently selecting the first option.
		if len(field.Options) > 0 {
			selected := false
			for _, option := range field.Options {
				if option.Selected {
					selected = true
					break
				}
			}
			if !selected {
				field.Options = append([]settingOption{{value, adminText(v.Locale, "admin.invalid_option", "Invalid: {{.Value}}", "Value", value), true}}, field.Options...)
			}
		}
		v.Fields = append(v.Fields, field)
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	var out strings.Builder
	v.Notice = localMessage(v.Locale, v.Notice)
	v.Error = localMessage(v.Locale, v.Error)
	tmpl, err := h.adminTemplate(v.Locale)
	if err == nil {
		err = tmpl.ExecuteTemplate(&out, "settings.html", v)
	}
	if err != nil {
		textError(w, "operation failed", 500)
		return
	}
	w.WriteHeader(status)
	_, _ = io.WriteString(w, out.String())
}

// settingsPage is available only after the dispatcher has authenticated an administrator.
func (h *Handler) settingsPage(w http.ResponseWriter, r *http.Request, id admin.Identity) {
	if r.URL.RawQuery != "" {
		h.fail(w, db.ErrInvalidInput)
		return
	}
	if h.settings == nil {
		textError(w, "Configuration is unavailable.", http.StatusServiceUnavailable)
		return
	}
	snap, err := h.settings.Read(r.Context())
	if err != nil {
		textError(w, "Configuration could not be read. Check the configuration file and try again.", http.StatusServiceUnavailable)
		return
	}
	h.renderSettings(w, id, snap, nil, snap.Revision, h.takeNotice(id.SessionID, 0, i18n.ViewOf(w)), "", http.StatusOK)
}

// settingsValidationMessage gives actionable feedback without including submitted values.
func settingsValidationMessage(err error) string {
	text := err.Error()
	switch {
	case strings.Contains(text, "time.force_tz"):
		return "Enter a valid IANA application time zone, or leave it empty for automatic selection."
	case strings.Contains(text, "official MusicBrainz"):
		return "The official MusicBrainz server requires a rate limit of 1."
	case strings.Contains(text, "rate_limit"), strings.Contains(text, "RATE_LIMIT"):
		return "Enter a positive whole-number MusicBrainz rate limit."
	case strings.Contains(text, "musicbrainz.url"):
		return "Enter a valid MusicBrainz base URL without a query string."
	case strings.Contains(text, "subsonic.url and subsonic.params"):
		return "Enter both the Subsonic URL and parameters, or leave both empty."
	case strings.Contains(text, "subsonic.url"):
		return "Enter a valid Subsonic base URL without a query string."
	case strings.Contains(text, "subsonic.params"):
		return "Check the Subsonic parameters: provide u and either p or t and s."
	case strings.Contains(text, "logging.level"):
		return "Select a supported log level."
	default:
		return "Some settings are invalid. Check the values and their combinations."
	}
}

// settingsAction validates a complete form before saving or explicitly testing a draft.
func (h *Handler) settingsAction(w http.ResponseWriter, r *http.Request, id admin.Identity, f *adminweb.PreparedForm, path string) {
	if h.settings == nil {
		textError(w, "Configuration is unavailable.", http.StatusServiceUnavailable)
		return
	}
	names := []string{"revision"}
	for _, spec := range settingsFields {
		names = append(names, spec.Name)
	}
	if !fields(f, names...) {
		h.fail(w, db.ErrInvalidInput)
		return
	}
	revisions, ok := f.Values["revision"]
	if !ok || len(revisions) != 1 || len(revisions[0]) != 64 {
		h.fail(w, db.ErrInvalidInput)
		return
	}
	revision := revisions[0]
	draft := make(map[string]string, len(settingsFields))
	for _, spec := range settingsFields {
		if values := f.Values[spec.Name]; len(values) == 1 {
			draft[spec.Name] = values[0]
		}
	}
	current, readErr := h.settings.Read(r.Context())
	if readErr != nil {
		h.renderSettings(w, id, cfg.SettingsSnapshot{}, draft, revision, "", "Configuration state is temporarily unavailable. Your draft is shown below.", http.StatusServiceUnavailable)
		return
	}
	for _, spec := range settingsFields {
		values, present := f.Values[spec.Name]
		overridden := current.Origins[spec.Name] == cfg.OriginEnvironment || current.Origins[spec.Name] == cfg.OriginFile
		if overridden {
			if present {
				h.fail(w, db.ErrInvalidInput)
				return
			}
			draft[spec.Name] = current.Effective[spec.Name]
		} else if !present || len(values) != 1 {
			h.fail(w, db.ErrInvalidInput)
			return
		}
	}
	if path == "settings/subsonic/test" {
		base, err := cfg.ValidateSubsonicDraft(draft[cfg.SUBSONIC_URL_ENV], draft[cfg.SUBSONIC_PARAMS_ENV])
		notice, message, status := "", "", http.StatusOK
		if err != nil {
			message = "Enter a valid Subsonic URL and parameters before testing."
			status = http.StatusBadRequest
		} else if err = h.subsonicTest(r.Context(), base, draft[cfg.SUBSONIC_PARAMS_ENV]); err != nil {
			message = "Subsonic connection could not be validated. Check the server and parameters."
			status = http.StatusBadGateway
		} else {
			notice = "Subsonic connection succeeded. The draft has not been saved."
		}
		snap, readErr := h.settings.Read(r.Context())
		if readErr != nil {
			message = "Configuration could not be read. Your draft is shown below."
			status = http.StatusServiceUnavailable
		}
		h.renderSettings(w, id, snap, draft, revision, notice, message, status)
		return
	}
	// Avoid rewriting the TOML representation of fields the user did not change.
	updates := make(map[string]string)
	for name, value := range draft {
		if current.Origins[name] != cfg.OriginEnvironment && current.Origins[name] != cfg.OriginFile && current.Saved[name] != value {
			updates[name] = value
		}
	}
	snap, err := h.settings.Save(r.Context(), revision, updates)
	if err == nil {
		notice := "Configuration saved. The running server already uses the effective values."
		if len(updates) == 0 {
			notice = "No configuration changes to save."
		}
		if len(updates) > 0 && snap.RestartRequired {
			notice = "Configuration saved. Restart TuneTrail to apply effective changes."
		}
		h.putNotice(id.SessionID, 0, notice)
		http.Redirect(w, r, "/admin/settings", http.StatusSeeOther)
		return
	}
	if errors.Is(err, cfg.ErrSettingsDurability) {
		h.renderSettings(w, id, snap, nil, snap.Revision, "", "The configuration was saved, but its durability could not be confirmed. Check the file before continuing.", http.StatusInternalServerError)
		return
	}
	message, status := "Configuration could not be saved. Your draft is shown below.", http.StatusInternalServerError
	switch {
	case errors.Is(err, cfg.ErrSettingsOverridden):
		message = "A setting is now controlled by an environment variable. Your draft is preserved; reload before saving."
		status = http.StatusConflict
	case errors.Is(err, cfg.ErrSettingsConflict):
		message = "Configuration changed in another session. Your draft is preserved; reload and reconcile it before saving."
		status = http.StatusConflict
	case errors.Is(err, cfg.ErrInvalidSettings):
		message = settingsValidationMessage(err)
		status = http.StatusBadRequest
	case errors.Is(err, cfg.ErrUnsupportedSettingsFormat):
		message = "This configuration format cannot be edited here without losing formatting. Edit the file manually."
		status = http.StatusUnprocessableEntity
	case errors.Is(err, cfg.ErrSettingsReadOnly), errors.Is(err, os.ErrPermission):
		message = "The configuration file cannot be written. Check file and directory permissions."
		status = http.StatusServiceUnavailable
	case errors.Is(err, cfg.ErrSettingsPath):
		message = "The configuration path is not a writable regular file."
		status = http.StatusServiceUnavailable
	}
	current, readErr = h.settings.Read(r.Context())
	if readErr != nil {
		current = cfg.SettingsSnapshot{}
		message = localMessage(i18n.ViewOf(w), message) + localMessage(i18n.ViewOf(w), " Saved and running values are temporarily unavailable.")
	}
	h.renderSettings(w, id, current, draft, revision, "", message, status)
}
