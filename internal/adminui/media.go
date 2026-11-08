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
	"github.com/ebrgr/TuneTrail/internal/models"
	"github.com/ebrgr/TuneTrail/internal/webquery"
)

type mergeCandidate struct {
	ID   int32
	Name string
}
type mediaView struct {
	Submission                      string
	Profile                         *models.ListeningProfile
	Kind                            string
	ID                              int32
	Name, MBID, CSRF, Error, Notice string
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
	Artists                         []*models.Artist
	AlbumID                         int32
	Listens                         []*models.Listen
	Values                          url.Values
}

// renderMedia buffers the administrative editor before writing the response.
func (h *Handler) renderMedia(w http.ResponseWriter, status int, v mediaView) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	var output strings.Builder
	if err := h.tmpl.ExecuteTemplate(&output, "manage.html", v); err != nil {
		http.Error(w, "operation failed", 500)
		return
	}
	w.WriteHeader(status)
	_, _ = io.WriteString(w, output.String())
}

// media routes are isolated until the administration dispatcher is activated.
func (h *Handler) media(w http.ResponseWriter, r *http.Request, parts []string) {
	if len(parts) < 4 || len(parts) > 5 || parts[1] != "manage" {
		http.NotFound(w, r)
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
	if len(parts) == 5 && parts[4] == "confirm" && r.Method == "POST" {
		h.confirmCatalog(w, r, identity, profile, kind, id)
		return
	}
	if err = admin.MediaExists(r.Context(), h.store, identity.User, kind, id); err != nil {
		h.fail(w, err)
		return
	}
	if len(parts) == 5 && parts[4] == "review" && r.Method == "GET" {
		h.reviewCatalog(w, r, identity, profile, kind, id)
		return
	}
	if r.Method == "POST" {
		if len(parts) != 5 {
			http.NotFound(w, r)
			return
		}
		h.mediaAction(w, r, identity, profile, kind, id, parts[4])
		return
	}
	if r.Method != "GET" || len(parts) != 4 {
		http.NotFound(w, r)
		return
	}
	h.mediaPage(w, r, identity, profile, kind, id, nil, "", nil)

}

// mediaPage loads shared metadata and personal listens, retaining validated input and navigation.
func (h *Handler) mediaPage(w http.ResponseWriter, r *http.Request, identity admin.Identity, profile *models.ListeningProfile, kind string, id int32, form *adminweb.PreparedForm, action string, actionErr error) {
	var err error
	token, err := h.csrf.SessionToken(identity)
	if err != nil {
		h.fail(w, err)
		return
	}
	ticket, err := h.newSubmission(identity.SessionID)
	if err != nil {
		h.fail(w, err)
		return
	}
	ctx := db.WithoutProfile(r.Context())
	effectiveZone := webquery.Timezone(r)
	v := mediaView{Submission: ticket, Profile: profile, Kind: kind, ID: id, CSRF: token, Notice: h.takeNotice(identity.SessionID, profile.ID), Timezone: effectiveZone.String(), Location: effectiveZone}
	query := r.URL.Query()
	if r.Method == "POST" && form != nil {
		if saved := adminweb.SafeReturn(form.SingleValue("return_to")); saved != "" {
			if parsed, e := url.Parse(saved); e == nil {
				query = parsed.Query()
			}
		}
	}
	if kind == "track" && query.Get("tz") != "" {
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
			if err == nil {
				var page *db.PaginatedResponse[*models.Listen]
				page, err = h.store.GetListensPaginated(db.WithProfile(r.Context(), profile.ID), db.GetItemsOpts{TrackID: int(id), Page: 1, Limit: 25, Timeframe: db.Timeframe{Period: db.PeriodAllTime}})
				if err == nil {
					v.Listens = page.Items
				}
			}
		}
	}
	if err == nil && r.Method == "GET" {
		q, qe := url.ParseQuery(r.URL.RawQuery)
		if qe != nil {
			err = db.ErrInvalidInput
		} else {
			for key, values := range q {
				if (key != "artist_query" && key != "tz" && key != "merge_query" && key != "listen_at" && key != "history_return") || len(values) != 1 || len(values[0]) > 4096 || (key == "artist_query" && kind != "track") {
					err = db.ErrInvalidInput
					break
				}
			}
			if err == nil && len(q.Get("merge_query")) > 100 {
				err = db.ErrInvalidInput
			}
			if err == nil {
				v.MergeQuery = strings.TrimSpace(q.Get("merge_query"))
				if v.MergeQuery != "" {
					switch kind {
					case "artist":
						var found []*models.Artist
						found, err = h.store.SearchArtists(ctx, v.MergeQuery)
						for _, x := range found {
							if x.ID != id {
								v.MergeCandidates = append(v.MergeCandidates, mergeCandidate{x.ID, x.Name})
							}
						}
					case "album":
						var found []*models.Album
						found, err = h.store.SearchAlbums(ctx, v.MergeQuery)
						for _, x := range found {
							if x.ID != id {
								v.MergeCandidates = append(v.MergeCandidates, mergeCandidate{x.ID, x.Title})
							}
						}
					case "track":
						var found []*models.Track
						found, err = h.store.SearchTracks(ctx, v.MergeQuery)
						for _, x := range found {
							if x.ID != id {
								v.MergeCandidates = append(v.MergeCandidates, mergeCandidate{x.ID, x.Title})
							}
						}
					}
				}
			}
			if err == nil && q.Get("listen_at") != "" {
				if kind != "track" {
					err = db.ErrInvalidInput
				} else {
					stamp, e := strconv.ParseInt(q.Get("listen_at"), 10, 64)
					if e != nil {
						err = db.ErrInvalidInput
					} else {
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
				v.HistoryReturn = adminweb.SafeReturn(q.Get("history_return"))
				if v.HistoryReturn == "" {
					err = db.ErrInvalidInput
				} else {
					back, _ := url.Parse(v.HistoryReturn)
					if back.Path != "/"+profile.Slug+"/listens" && back.Path != "/"+profile.Slug {
						err = db.ErrInvalidInput
					}
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
		v.ReturnTo = adminweb.SafeReturn(r.URL.RequestURI())
	} else if form != nil {
		v.ReturnTo = adminweb.SafeReturn(form.SingleValue("return_to"))
	}
	if v.ReturnTo == "" {
		v.ReturnTo = base
	}
	if form != nil {
		v.Values = form.Values
	}
	if actionErr != nil {
		v.Error = adminweb.PublicError(actionErr)
		if action == "listen-add" && errors.Is(actionErr, db.ErrConflict) {
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
	if returnTo != "" {
		parsed, e := url.Parse(returnTo)
		if e != nil || (parsed.Path != expected && !(action == "listen-delete" && (parsed.Path == "/"+p.Slug+"/listens" || parsed.Path == "/"+p.Slug))) {
			h.fail(w, db.ErrInvalidInput)
			return
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
		at, e := admin.LocalListenTime(f.SingleValue("local_time"), f.SingleValue("timezone"), f.SingleValue("offset"))
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
		http.NotFound(w, r)
		return
	}
	if err != nil {
		if returnTo != "" {
			f.Values.Set("return_to", returnTo)
		}
		h.mediaPage(w, r, identity, p, kind, id, f, action, err)
		return
	}
	target := expected
	if returnTo != "" {
		target = returnTo
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
	http.Redirect(w, r, target, http.StatusSeeOther)
}
