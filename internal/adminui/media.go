package adminui

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/ebrgr/TuneTrail/internal/admin"
	"github.com/ebrgr/TuneTrail/internal/adminweb"
	"github.com/ebrgr/TuneTrail/internal/db"
	"github.com/ebrgr/TuneTrail/internal/i18n"
	"github.com/ebrgr/TuneTrail/internal/models"
	"github.com/ebrgr/TuneTrail/internal/webassets"
	"github.com/ebrgr/TuneTrail/internal/webquery"
	"github.com/google/uuid"
)

var errInvalidListenTimezone = fmt.Errorf("invalid listen timezone: %w", db.ErrInvalidInput)

type mergeCandidate struct {
	ID      int32
	Name    string
	Image   string
	Artists []models.SimpleArtist
}
type mediaView struct {
	Nav                             webassets.Nav
	Locale                          *i18n.View
	Submission                      string
	Profile                         *models.ListeningProfile
	Kind                            string
	ID                              int32
	Name, MBID, CSRF, Error, Notice string
	AlbumName                       string
	SuggestedURL                    string
	MergeQuery                      string
	MergeCandidates                 []mergeCandidate
	FocusListen                     *models.Listen
	HistoryReturn                   string
	ArtistQuery                     string
	Timezone                        string
	Location                        *time.Location
	ReturnTo                        string
	ArtistResults                   []*models.Artist
	Aliases                         []models.Alias
	Various                         bool
	AmbiguousTime                   bool
	Artists                         []*models.Artist
	AlbumID                         int32
	Listens                         []*models.Listen
	Values                          url.Values
	Query                           url.Values
	// Section is set when only one section is rendered as a fragment (ISSUE-124).
	Section   string
	Task      string
	TaskQuery string
}

// actionSection names legacy editor sections; artwork keeps its full-page fallback.
// Dialog requests render the complete task instead.
func actionSection(action string) string {
	switch action {
	case "alias-add", "alias-primary", "alias-delete":
		return "aliases"
	case "metadata":
		return "metadata"
	case "artist-add", "artist-remove", "primary-artist":
		return "artists"
	case "listen-add", "listen-delete":
		return "listens"
	}
	return ""
}

// renderMedia buffers the administrative editor, or one section of it, before writing.
func (h *Handler) renderMedia(w http.ResponseWriter, status int, v mediaView) {
	v.Locale = i18n.ViewOf(w)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	var output strings.Builder
	v.Error = localMessage(v.Locale, v.Error)
	tmpl, err := h.adminTemplate(v.Locale)
	name := "admin/manage"
	if v.Section != "" {
		name = "manage/" + v.Section
	}
	if err == nil {
		err = tmpl.ExecuteTemplate(&output, name, v)
	}
	if err == nil && v.Section != "" {
		// Set only after a successful render: the script checks these before swapping, and
		// the ticket replaces the consumed one in every form of the page (ISSUE-124).
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("X-TuneTrail-Fragment", v.Section)
		w.Header().Set("X-TuneTrail-Submission", v.Submission)
	}
	if err != nil {
		textError(w, "operation failed", 500)
		return
	}
	w.WriteHeader(status)
	_, _ = io.WriteString(w, output.String())
}

// media routes are isolated until the administration dispatcher is activated.
func (h *Handler) media(w http.ResponseWriter, r *http.Request, parts []string) {
	if len(parts) < 4 || len(parts) > 5 || parts[1] != "manage" {
		textError(w, "item not found", http.StatusNotFound)
		return
	}
	slug, kind := parts[0], parts[2]
	id, err := parseID(parts[3])
	if err != nil {
		h.fail(w, err)
		return
	}
	identity, err := adminweb.SessionIdentity(r, h.store)
	if errors.Is(err, admin.ErrUnauthorized) && r.Method == "GET" {
		next := adminweb.SafeReturn(r.URL.RequestURI())
		if next == "" && len(parts) == 5 && parts[4] == "review" {
			next = adminweb.SafeReturn(fmt.Sprintf("/%s/manage/%s/%d", url.PathEscape(slug), kind, id))
		}
		if next == "" {
			h.fail(w, db.ErrInvalidInput)
			return
		}
		http.Redirect(w, r, "/admin?next="+url.QueryEscape(next), http.StatusSeeOther)
		return
	}
	if err != nil {
		h.fail(w, err)
		return
	}
	profile, err := h.store.GetListeningProfile(r.Context(), slug)
	if err != nil {
		h.fail(w, err)
		return
	}
	// Personal tasks require canonical ownership; all other editors require root.
	err = admin.AuthorizeAction(identity.User, mediaCapability(r, parts), profile)
	if err != nil {
		h.failForSession(w, r, identity, err)
		return
	}
	if len(parts) == 5 && parts[4] == "confirm" && r.Method == "POST" {
		h.confirmCatalog(w, r, identity, profile, kind, id)
		return
	}
	if !personalMediaRoute(r, parts) {
		if err = admin.MediaExists(r.Context(), h.store, identity.User, kind, id); err != nil {
			h.fail(w, err)
			return
		}
	}
	// Personal deletion is a direct confirmation, including the old review link.
	if len(parts) == 5 && r.Method == "GET" && (parts[4] == "delete" || parts[4] == "review" && r.URL.Query().Get("action") == "delete") {
		h.reviewProfileDelete(w, r, identity, profile, kind, id)
		return
	}
	if len(parts) == 5 && parts[4] == "review" && r.Method == "GET" {
		h.reviewCatalog(w, r, identity, profile, kind, id)
		return
	}
	if r.Method == "POST" {
		if len(parts) != 5 {
			textError(w, "item not found", http.StatusNotFound)
			return
		}
		h.mediaAction(w, r, identity, profile, kind, id, parts[4])
		return
	}
	if r.Method != "GET" || (len(parts) != 4 && !adminweb.MediaTaskAllowed(kind, parts[4])) {
		textError(w, "item not found", http.StatusNotFound)
		return
	}
	// Artist and merge searches can answer with their section only (ISSUE-124).
	section := ""
	if r.URL.Query().Get("partial") == "1" {
		if !adminweb.FragmentRequested(r) {
			q := r.URL.Query()
			q.Del("partial")
			http.Redirect(w, r, (&url.URL{Path: r.URL.Path, RawQuery: q.Encode()}).RequestURI(), http.StatusSeeOther)
			return
		}
		switch {
		case dialogRequested(r) && r.Header.Get("X-TuneTrail-Search") == "merge" && len(parts) == 5 && parts[4] == "merge" && r.URL.Query().Has("merge_query"):
			section = "merge-results"
		case dialogRequested(r) && r.Header.Get("X-TuneTrail-Search") == "artists" && kind == "track" && len(parts) == 5 && parts[4] == "edit" && r.URL.Query().Has("artist_query"):
			section = "artist-results"
		case dialogRequested(r) && len(parts) == 5 && parts[4] != "listens":
			section = "task"
		case r.URL.Query().Has("artist_query") && kind == "track" && (len(parts) == 4 || parts[4] == "edit"):
			section = "artists"
		case r.URL.Query().Has("merge_query") && (len(parts) == 4 || parts[4] == "merge"):
			section = "merge"
		default:
			h.fail(w, db.ErrInvalidInput)
			return
		}
	}
	h.mediaPage(w, r, identity, profile, kind, id, nil, "", nil, section)

}

// mediaPage loads shared metadata and personal listens, retaining validated input and navigation.
// A non-empty section renders only that section (ISSUE-124).
func (h *Handler) mediaPage(w http.ResponseWriter, r *http.Request, identity admin.Identity, profile *models.ListeningProfile, kind string, id int32, form *adminweb.PreparedForm, action string, actionErr error, section string) {
	var err error
	token, err := h.csrf.SessionToken(identity)
	if err != nil {
		h.fail(w, err)
		return
	}
	ctx := db.WithoutProfile(r.Context())
	effectiveZone := webquery.Timezone(r)
	task := ""
	if r.Method == "GET" {
		parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
		if len(parts) == 5 {
			task = parts[4]
		}
	} else {
		task = actionTask(action)
		if action == "listen-delete" && dialogRequested(r) {
			task = "listen"
		}
		// Legacy section requests keep their existing composition.
		if section != "" && section != "task" {
			task = ""
		}
		if form != nil {
			if u, e := url.Parse(adminweb.SafeReturn(form.SingleValue("return_to"))); e == nil {
				parts := strings.Split(strings.Trim(u.Path, "/"), "/")
				if len(parts) == 5 {
					task = parts[4]
				}
			}
		}
	}
	v := mediaView{Task: task, Profile: profile, Kind: kind, ID: id, CSRF: token, Timezone: effectiveZone.String(), Location: effectiveZone}
	// Notices are written by listen actions; other fragments must not consume them.
	if section == "" || section == "listens" || section == "task" {
		v.Notice = h.takeNotice(identity.SessionID, profile.ID, i18n.ViewOf(w))
	}
	query := r.URL.Query()
	if r.Method == "POST" && form != nil {
		if saved := adminweb.SafeReturn(form.SingleValue("return_to")); saved != "" {
			if parsed, e := url.Parse(saved); e == nil {
				// Public history filters remain in ReturnTo; only management context populates this view.
				query = url.Values{}
				savedQuery := parsed.Query()
				for _, key := range []string{"tz", "artist_query", "merge_query", "listen_at", "history_return"} {
					if values, ok := savedQuery[key]; ok {
						query[key] = values
					}
				}
			}
		}
	}
	// A rejected deletion keeps the selected listen and its original history context.
	if r.Method == "POST" && task == "listen" && form != nil {
		query.Set("listen_at", form.SingleValue("unix"))
		if back := adminweb.SafeHistoryReturn(form.SingleValue("return_to"), profile.Slug); back != "" {
			query.Set("history_return", back)
		}
	}
	if query.Get("tz") != "" {
		if _, err = time.LoadLocation(query.Get("tz")); err != nil {
			h.fail(w, db.ErrInvalidInput)
			return
		}
		timezoneRequest := r.Clone(r.Context())
		copiedURL := *r.URL
		copiedURL.RawQuery = query.Encode()
		timezoneRequest.URL = &copiedURL
		v.Location = webquery.Timezone(timezoneRequest)
		v.Timezone = v.Location.String()
	}
	if r.Method == "POST" {
		v.ArtistQuery = query.Get("artist_query")
	}
	switch kind {
	case "artist":
		var x *models.Artist
		x, err = h.store.GetArtist(ctx, db.GetArtistOpts{ID: id})
		if err == nil {
			v.Name = x.Name
			if x.MbzID != nil {
				v.MBID = x.MbzID.String()
			}
			v.Aliases, err = h.store.GetAllArtistAliases(ctx, id)
		}
	case "album":
		var x *models.Album
		x, err = h.store.GetAlbum(ctx, db.GetAlbumOpts{ID: id})
		if err == nil {
			v.Name = x.Title
			v.Various = x.VariousArtists
			if x.MbzID != nil {
				v.MBID = x.MbzID.String()
				v.SuggestedURL = "https://coverartarchive.org/release/" + v.MBID + "/front"
			}
			v.Aliases, err = h.store.GetAllAlbumAliases(ctx, id)
			if err == nil {
				v.Artists, err = h.store.GetArtistsForAlbum(ctx, id)
			}
		}
	case "track":
		var x *models.Track
		x, err = h.store.GetTrack(ctx, db.GetTrackOpts{ID: id})
		if err == nil {
			v.Name = x.Title
			v.AlbumID = x.AlbumID
			album, e := h.store.GetAlbum(ctx, db.GetAlbumOpts{ID: x.AlbumID})
			if e != nil {
				err = e
				break
			}
			v.AlbumName = album.Title
			if album.MbzID != nil {
				v.SuggestedURL = "https://coverartarchive.org/release/" + album.MbzID.String() + "/front"
			}
			if x.MbzID != nil {
				v.MBID = x.MbzID.String()
			}
			v.Aliases, err = h.store.GetAllTrackAliases(ctx, id)
			if err == nil {
				v.Artists, err = h.store.GetArtistsForTrack(ctx, id)
			}
			if err == nil && (task == "listens" || section == "listens" && task != "add-listen") {
				var page *db.PaginatedResponse[*models.Listen]
				page, err = h.store.GetListensPaginated(db.WithProfile(r.Context(), profile.ID), db.GetItemsOpts{TrackID: int(id), Page: 1, Limit: 25, Timeframe: db.Timeframe{Period: db.PeriodAllTime}})
				if err == nil {
					v.Listens = page.Items
				}
			}
		}
	}
	if err == nil {
		rawQuery := query.Encode()
		if r.Method == "GET" {
			rawQuery = r.URL.RawQuery
		}
		q, qe := url.ParseQuery(rawQuery)
		if qe != nil {
			err = db.ErrInvalidInput
		} else {
			for key, values := range q {
				if (key != "artist_query" && key != "tz" && key != "merge_query" && key != "listen_at" && key != "history_return" && key != "partial") || len(values) != 1 || len(values[0]) > 4096 || (key == "artist_query" && kind != "track") {
					err = db.ErrInvalidInput
					break
				}
			}
			if err == nil && (len(q.Get("merge_query")) > 100 || len(q.Get("artist_query")) > 100) {
				err = db.ErrInvalidInput
			}
			if err == nil {
				v.MergeQuery = strings.TrimSpace(q.Get("merge_query"))
				// Opening merge starts with the current name, as in the reference UI.
				if task == "merge" && !q.Has("merge_query") && len(v.Name) <= 100 {
					v.MergeQuery = v.Name
				}
				if v.MergeQuery != "" {
					switch kind {
					case "artist":
						var found []*models.Artist
						found, err = h.store.SearchArtists(ctx, v.MergeQuery)
						for _, x := range found {
							if x.ID != id {
								v.MergeCandidates = append(v.MergeCandidates, mergeCandidate{ID: x.ID, Name: x.Name, Image: x.Image.Small})
							}
						}
					case "album":
						var found []*models.Album
						found, err = h.store.SearchAlbums(ctx, v.MergeQuery)
						for _, x := range found {
							if x.ID != id {
								v.MergeCandidates = append(v.MergeCandidates, mergeCandidate{ID: x.ID, Name: x.Title, Image: x.Image.Small, Artists: x.Artists})
							}
						}
					case "track":
						var found []*models.Track
						found, err = h.store.SearchTracks(ctx, v.MergeQuery)
						for _, x := range found {
							if x.ID != id {
								v.MergeCandidates = append(v.MergeCandidates, mergeCandidate{ID: x.ID, Name: x.Title, Image: x.Image.Small, Artists: x.Artists})
							}
						}
					}
				}
			}
			if task == "listen" && !q.Has("listen_at") {
				err = db.ErrInvalidInput
			}
			for _, key := range []string{"listen_at", "history_return", "tz"} {
				if q.Has(key) && q.Get(key) == "" {
					err = db.ErrInvalidInput
				}
			}
			if err == nil && q.Get("listen_at") != "" {
				if kind != "track" {
					err = db.ErrInvalidInput
				} else {
					stamp, e := strconv.ParseInt(q.Get("listen_at"), 10, 64)
					if e != nil {
						err = db.ErrInvalidInput
					} else if task == "listen" || task == "" {
						at := time.Unix(stamp, 0)
						var found *db.PaginatedResponse[*models.Listen]
						found, err = h.store.GetListensPaginated(db.WithProfile(r.Context(), profile.ID), db.GetItemsOpts{TrackID: int(id), Page: 1, Limit: 1, Timeframe: db.Timeframe{Resolved: true, From: at, To: at}})
						if err == nil {
							if len(found.Items) != 1 {
								err = db.ErrNotFound
							} else {
								v.FocusListen = found.Items[0]
							}
						}
					}
				}
			}
			if err == nil && q.Get("history_return") != "" {
				v.HistoryReturn = adminweb.SafeHistoryReturn(q.Get("history_return"), profile.Slug)
				if v.HistoryReturn == "" {
					err = db.ErrInvalidInput
				}
			}
			if err == nil {
				v.ArtistQuery = strings.TrimSpace(q.Get("artist_query"))
				if v.ArtistQuery != "" {
					v.ArtistResults, err = h.store.SearchArtists(ctx, v.ArtistQuery)
					if len(v.ArtistResults) > 50 {
						v.ArtistResults = v.ArtistResults[:50]
					}
				}
			}
		}
	}
	if err != nil {
		h.fail(w, err)
		return
	}
	base := fmt.Sprintf("/%s/manage/%s/%d", url.PathEscape(profile.Slug), kind, id)
	if r.Method == "GET" {
		// partial is not a return parameter: drop it so search fragments keep tz and query.
		page := *r.URL
		q := page.Query()
		q.Del("partial")
		page.RawQuery = q.Encode()
		v.ReturnTo = adminweb.SafeReturn(page.RequestURI())
	} else if form != nil {
		v.ReturnTo = adminweb.SafeReturn(form.SingleValue("return_to"))
	}
	if v.ReturnTo == "" {
		v.ReturnTo = base
	}
	// Legacy query addresses select the corresponding task after validation.
	if task == "" && section == "" && r.Method == "GET" {
		switch {
		case query.Has("listen_at"):
			task = "listen"
		case query.Has("artist_query"):
			task = "edit"
		case query.Has("merge_query"):
			task = "merge"
		case query.Has("tz") && kind == "track":
			task = "listens"
		}
		if task != "" {
			http.Redirect(w, r, base+"/"+task+"?"+query.Encode(), http.StatusSeeOther)
			return
		}
	}
	var ticket string
	if section != "" && r.Method == "GET" && (section != "task" || r.Header.Get("X-TuneTrail-Submission") != "") {
		// Search fragments never mint tickets (the store is bounded for all sessions); their
		// forms reuse the page's current ticket, which is validated again on submission.
		if raw, e := uuid.Parse(r.Header.Get("X-TuneTrail-Submission")); e == nil {
			ticket = raw.String()
		}
	} else if task != "" && task != "merge" && task != "delete" || section != "" && section != "task" {
		// Merge/delete use their reviewed plan token, not a generic mutation ticket.
		// In particular, their search fragments must not fill the bounded ticket store.
		ticket, err = h.newSubmission(identity.SessionID)
		if err != nil {
			h.fail(w, err)
			return
		}
	}
	if len(v.MergeCandidates) > 50 {
		v.MergeCandidates = v.MergeCandidates[:50]
	}
	v.Submission = ticket
	// Task links keep all validated navigation context, without the fragment marker.
	linkQuery := url.Values{}
	for key, values := range query {
		linkQuery[key] = append([]string(nil), values...)
	}
	linkQuery.Del("partial")
	if len(linkQuery) > 0 {
		v.TaskQuery = "?" + linkQuery.Encode()
	}
	v.Task = task
	v.Query = linkQuery
	if form != nil {
		v.Values = form.Values
	}
	if actionErr != nil {
		v.Error = adminweb.PublicError(actionErr)
		if errors.Is(actionErr, errInvalidListenTimezone) {
			v.Error = "Invalid time zone. Use an IANA name such as America/Sao_Paulo."
		}
		if action == "listen-add" && errors.Is(actionErr, db.ErrConflict) {
			v.AmbiguousTime = true
			v.Error = "This local time occurs twice. Specify the UTC offset."
		}
		if action == "metadata" && form != nil {
			v.MBID = form.SingleValue("mbid")
			if kind == "album" {
				v.Various = form.SingleValue("various") == "1"
			}
		}
	}
	status := http.StatusOK
	if actionErr != nil {
		status = adminweb.ErrorStatus(actionErr)
	}
	v.Section = section
	// Task fragments already carry their own form tokens and have no sidebar.
	if v.Section == "" {
		v.Nav = h.navigation(r, identity, profile, "")
	}
	h.renderMedia(w, status, v)
}

// mediaAction validates CSRF, the one-use ticket and profile identity before mutation.
func (h *Handler) mediaAction(w http.ResponseWriter, r *http.Request, identity admin.Identity, p *models.ListeningProfile, kind string, id int32, action string) {
	max := int64(8192)
	if action == "artwork" {
		// Acquire capacity before parsing or buffering an artwork form.
		ctx, release, err := admin.AcquireImageAdmission(r.Context())
		if err != nil {
			h.fail(w, err)
			return
		}
		defer release()
		r = r.WithContext(ctx)
		max = admin.MaxArtworkBytes + (1 << 20)
	}
	f, err := h.csrf.ParseSessionForm(w, r, identity, adminweb.FormLimits{MaxBytes: max, TempDir: h.tempDir})
	if err != nil {
		h.fail(w, err)
		return
	}
	defer f.Close()
	if err = h.consumeSubmission(identity.SessionID, f.SingleValue("submission_token")); err != nil {
		h.fail(w, err)
		return
	}
	returnTo := adminweb.SafeReturn(f.SingleValue("return_to"))
	expected := fmt.Sprintf("/%s/manage/%s/%d", url.PathEscape(p.Slug), kind, id)
	if f.SingleValue("return_to") != "" && returnTo == "" {
		h.fail(w, db.ErrInvalidInput)
		return
	}
	if returnTo != "" {
		parsed, e := url.Parse(returnTo)
		if e != nil {
			h.fail(w, db.ErrInvalidInput)
			return
		}
		destination := strings.Split(strings.TrimPrefix(parsed.Path, "/"), "/")
		isTask := parsed.Path == expected || strings.HasPrefix(parsed.Path, expected+"/") && adminweb.MediaTaskAllowed(kind, strings.TrimPrefix(parsed.Path, expected+"/"))
		isHistory := action == "listen-delete" && adminweb.SafeHistoryReturn(returnTo, p.Slug) != ""
		if !isTask && !isHistory {
			h.fail(w, db.ErrInvalidInput)
			return
		}
		// The form's return context can also choose an editor to render after a mutation.
		if isTask {
			read := r.Clone(r.Context())
			read.Method, read.URL = http.MethodGet, parsed
			if err := admin.AuthorizeAction(identity.User, mediaCapability(read, destination), p); err != nil {
				h.failForSession(w, r, identity, err)
				return
			}
		}
	}
	f.Values.Del("return_to")
	if f.SingleValue("profile_id") != strconv.FormatInt(int64(p.ID), 10) || f.SingleValue("profile_slug") != p.Slug {
		h.fail(w, db.ErrConflict)
		return
	}
	switch action {
	case "alias-add", "alias-primary", "alias-delete":
		if !fields(f, "profile_id", "profile_slug", "alias") {
			err = db.ErrInvalidInput
			break
		}
		op := strings.TrimPrefix(action, "alias-")
		err = admin.EditAlias(r.Context(), h.store, identity.User, kind, id, op, f.SingleValue("alias"))
	case "metadata":
		if !fields(f, "profile_id", "profile_slug", "mbid", "various") {
			err = db.ErrInvalidInput
			break
		}
		var various *bool
		if kind == "album" {
			v := f.SingleValue("various")
			if v != "0" && v != "1" {
				err = db.ErrInvalidInput
				break
			}
			b := v == "1"
			various = &b
		}
		err = admin.UpdateMediaMetadata(r.Context(), h.store, identity.User, kind, id, f.SingleValue("mbid"), various)
	case "artist-add", "artist-remove":
		if kind != "track" || !fields(f, "profile_id", "profile_slug", "artist_id") {
			err = db.ErrInvalidInput
			break
		}
		var artist int32
		artist, err = parseID(f.SingleValue("artist_id"))
		if err == nil {
			err = admin.ChangeTrackArtist(r.Context(), h.store, identity.User, id, artist, action == "artist-add")
		}
	case "primary-artist":
		if !fields(f, "profile_id", "profile_slug", "artist_id", "primary") {
			err = db.ErrInvalidInput
			break
		}
		var a int32
		a, err = parseID(f.SingleValue("artist_id"))
		if err != nil {
			break
		}
		v := f.SingleValue("primary")
		if v != "0" && v != "1" {
			err = db.ErrInvalidInput
			break
		}
		err = admin.SetMediaArtistPrimary(r.Context(), h.store, identity.User, kind, id, a, v == "1")
	case "listen-add":
		if kind != "track" || !fields(f, "profile_id", "profile_slug", "local_time", "timezone", "offset") {
			err = db.ErrInvalidInput
			break
		}
		// An omitted timezone uses the same effective page context, including a validated return query.
		zone := f.SingleValue("timezone")
		if zone == "" {
			zoneRequest := r.Clone(r.Context())
			pageURL := *r.URL
			if saved, parseErr := url.Parse(returnTo); parseErr == nil {
				pageURL.RawQuery = saved.RawQuery
				if savedZone := saved.Query().Get("tz"); savedZone != "" {
					if _, zoneErr := time.LoadLocation(savedZone); zoneErr != nil {
						err = errInvalidListenTimezone
						break
					}
				}
			}
			zoneRequest.URL = &pageURL
			zone = webquery.Timezone(zoneRequest).String()
		}
		if _, zoneErr := time.LoadLocation(zone); zoneErr != nil {
			err = errInvalidListenTimezone
			break
		}
		at, e := admin.LocalListenTime(f.SingleValue("local_time"), zone, f.SingleValue("offset"))
		if e != nil {
			err = e
			break
		}
		inserted, e := admin.ManualListen(r.Context(), h.store, identity.User, p.ID, id, at, "")
		if e != nil {
			err = e
			break
		}
		if inserted {
			h.putNotice(identity.SessionID, p.ID, "Listen added.")
		} else {
			h.putNotice(identity.SessionID, p.ID, "This listen already exists.")
		}
	case "listen-delete":
		if kind != "track" || !fields(f, "profile_id", "profile_slug", "unix", "confirm") || f.SingleValue("confirm") != "1" {
			err = db.ErrInvalidInput
			break
		}
		unix, e := strconv.ParseInt(f.SingleValue("unix"), 10, 64)
		if e != nil {
			err = db.ErrInvalidInput
			break
		}
		err = admin.RemovePersonalListen(r.Context(), h.store, identity.User, p.ID, id, time.Unix(unix, 0))
		if err == nil {
			h.putNotice(identity.SessionID, p.ID, "Listen deleted.")
		}
	case "artwork":
		if !fields(f, "profile_id", "profile_slug", "image", "image_url") {
			err = db.ErrInvalidInput
			break
		}
		var reader io.Reader
		file, e := openImage(f)
		if e != nil {
			err = e
			break
		}
		if file != nil {
			defer file.Close()
			reader = file
		}
		_, err = admin.ReplaceArtwork(r.Context(), h.store, identity.User, kind, id, f.SingleValue("image_url"), reader, admin.ArtworkClient{})
	default:
		textError(w, "item not found", http.StatusNotFound)
		return
	}
	// A deleted listen has no detail fragment; acknowledge it without rerendering a missing record.
	if err == nil && action == "listen-delete" && dialogRequested(r) {
		w.Header().Set("X-TuneTrail-Listen-Deleted", f.SingleValue("unix"))
		w.WriteHeader(http.StatusNoContent)
		return
	}
	// Dialog mutations refresh the task, keeping its return context and issuing a fresh ticket.
	section := ""
	if adminweb.FragmentRequested(r) {
		section = actionSection(action)
		if dialogRequested(r) {
			section = "task"
		}
	}
	if err != nil {
		if returnTo != "" {
			f.Values.Set("return_to", returnTo)
		}
		h.mediaPage(w, r, identity, p, kind, id, f, action, err, section)
		return
	}
	if section != "" {
		// Success: fresh section without the submitted values, keeping the return context.
		f.Values = url.Values{}
		if returnTo != "" {
			f.Values.Set("return_to", returnTo)
		}
		h.mediaPage(w, r, identity, p, kind, id, f, action, nil, section)
		return
	}
	target := expected + "/" + actionTask(action)
	if returnTo != "" {
		target = returnTo
		backTask, _ := url.Parse(target)
		if backTask.Path == expected {
			backTask.Path += "/" + actionTask(action)
			target = backTask.String()
		}
		if action == "listen-delete" {
			back, _ := url.Parse(returnTo)
			if back.Path == "/"+p.Slug+"/listens" {
				queryReq := r.Clone(r.Context())
				queryReq.URL = back
				opts, _, parseErr := webquery.HTML(queryReq, false)
				if parseErr == nil {
					remaining, e := h.store.GetListensPaginated(db.WithProfile(r.Context(), p.ID), opts)
					if e == nil {
						last := int((remaining.TotalCount + int64(opts.Limit) - 1) / int64(opts.Limit))
						if last < 1 {
							last = 1
						}
						if opts.Page > last {
							q := back.Query()
							q.Set("page", strconv.Itoa(last))
							back.RawQuery = q.Encode()
							target = back.String()
						}
					}
				}
			}
		}
	}
	// A removed selected listen no longer has a detail page; return to its personal list.
	if action == "listen-delete" {
		u, e := url.Parse(target)
		if e == nil && strings.HasPrefix(u.Path, expected+"/") {
			if u.Path == expected+"/listen" {
				u.Path = expected + "/listens"
			}
			q := u.Query()
			q.Del("listen_at")
			u.RawQuery = q.Encode()
			target = u.String()
		}
	}
	if action == "listen-delete" {
		target = h.historyMediaDestination(r.Context(), p, target)
	}
	// Without JavaScript, come back to the edited section instead of the top (ISSUE-124).
	if anchor := actionSection(action); anchor != "" {
		if u, e := url.Parse(target); e == nil && strings.HasPrefix(u.Path, expected+"/") {
			u.Fragment = anchor
			target = u.String()
		}
	}
	http.Redirect(w, r, target, http.StatusSeeOther)
}

// actionTask preserves old mutation endpoints while selecting their full-page destination.
func actionTask(action string) string {
	switch action {
	case "listen-add":
		return "add-listen"
	case "listen-delete":
		return "listens"
	case "artwork":
		return "image"
	default:
		return "edit"
	}
}
