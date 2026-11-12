package engine

import (
	"fmt"
	"github.com/ebrgr/TuneTrail/internal/adminui"
	"github.com/ebrgr/TuneTrail/internal/adminweb"
	"github.com/ebrgr/TuneTrail/internal/i18n"
	"github.com/ebrgr/TuneTrail/internal/webui"
	"net/http"
	"strings"
	"sync/atomic"
	"time"

	"github.com/ebrgr/TuneTrail/engine/handlers"
	"github.com/ebrgr/TuneTrail/engine/middleware"
	"github.com/ebrgr/TuneTrail/internal/cfg"
	dbstore "github.com/ebrgr/TuneTrail/internal/db"
	mbz "github.com/ebrgr/TuneTrail/internal/mbz"
	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/go-chi/httprate"
)

// bindRoutes composes Go documents, actions and JSON routes.
func bindRoutes(
	r *chi.Mux,
	ready *atomic.Bool,
	db dbstore.DB,
	mbz mbz.MusicBrainzCaller,
) {
	if !(len(cfg.AllowedOrigins()) == 0) && !(cfg.AllowedOrigins()[0] == "") {
		r.Use(cors.Handler(cors.Options{
			AllowedOrigins: cfg.AllowedOrigins(),
			AllowedMethods: []string{"GET", "OPTIONS", "HEAD"},
		}))
	}
	r.Use(chimiddleware.GetHead)

	r.With(chimiddleware.RequestSize(5<<20)).
		Get("/image/{image_id}/{filename}", handlers.ImageHandler(db))

	// The same global limiter instance counts HTML and JSON login attempts together.
	loginLimit := func(h http.Handler) http.Handler { return h }
	if !cfg.RateLimitDisabled() {
		loginLimit = httprate.Limit(10, time.Minute, httprate.WithLimitHandler(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Cache-Control", "no-store")
			http.Error(w, `{"error":"too many requests"}`, http.StatusTooManyRequests)
		}))
	}
	r.Route("/apis/web/v1", func(r chi.Router) {
		r.Use(middleware.SameOrigin)
		r.Get("/config", handlers.GetCfgHandler())

		r.Get("/profiles", handlers.ListProfilesHandler(db))
		r.With(middleware.RequireAdmin(db)).Post("/profiles", handlers.CreateProfileHandler(db))
		r.Route("/profiles/{slug}", func(r chi.Router) {
			r.Use(middleware.ProfileScope(db))
			r.Get("/", handlers.GetProfileHandler())
			bindProfileReads(r, db)
			r.Group(func(r chi.Router) {
				r.Use(middleware.RequireAdmin(db))
				r.Patch("/", handlers.UpdateProfileHandler(db))
				r.Get("/management", handlers.ProfileManagementHandler(db))
				r.Delete("/", handlers.DeleteProfileHandler(db))
				r.Put("/avatar", handlers.ProfileAvatarHandler(db))
				r.Delete("/avatar", handlers.ProfileAvatarHandler(db))
				r.Get("/export", handlers.ExportHandler(db))
				r.Post("/import", profileImportHandler(db, mbz))
				r.Get("/relay", handlers.GetProfileRelayHandler(db))
				r.Post("/relay/test", handlers.TestProfileRelayHandler(db))
				r.Put("/relay", handlers.SetProfileRelayHandler(db))
				r.Delete("/relay", handlers.SetProfileRelayHandler(db))
				r.Post("/listens", handlers.SubmitListenWithIDHandler(db))
				r.Delete("/listens", handlers.DeleteListenHandler(db))
				r.Get("/user/apikeys", handlers.GetApiKeysHandler(db))
				r.Post("/user/apikeys", handlers.GenerateApiKeyHandler(db))
				r.Patch("/user/apikeys/{id}", handlers.UpdateApiKeyLabelHandler(db))
				r.Delete("/user/apikeys/{id}", handlers.DeleteApiKeyHandler(db))
			})
		})
		// Transitional endpoints only resolve an unambiguous single profile.
		r.Group(func(r chi.Router) {
			r.Use(middleware.ProfileScope(db))
			bindProfileReads(r, db)
		})

		r.Post("/logout", handlers.LogoutHandler(db))
		r.With(loginLimit).Post("/login", handlers.LoginHandler(db))

		r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
			if !ready.Load() {
				http.Error(w, "not ready", http.StatusServiceUnavailable)
				return
			}
			w.WriteHeader(http.StatusOK)
		})

		r.Group(func(r chi.Router) {
			r.Use(middleware.RequireAdmin(db))

			r.Delete("/artist/{id}", handlers.DeleteArtistHandler(db))
			r.Delete("/artist/{id}/aliases", handlers.DeleteArtistAliasHandler(db))
			r.Post("/artist/{id}/merge", handlers.MergeArtistsHandler(db))
			r.Post("/artist/{id}/aliases", handlers.CreateArtistAliasHandler(db))
			r.Patch("/artist/{id}", handlers.UpdateArtistHandler(db))
			r.Patch("/artist/{id}/image", handlers.ReplaceArtistImageHandler(db))
			r.Patch("/artist/{id}/aliases/primary", handlers.SetPrimaryArtistAliasHandler(db))

			r.Delete("/album/{id}", handlers.DeleteAlbumHandler(db))
			r.Delete("/album/{id}/aliases", handlers.DeleteAlbumAliasHandler(db))
			r.Post("/album/{id}/merge", handlers.MergeAlbumsHandler(db))
			r.Post("/album/{id}/aliases", handlers.CreateAlbumAliasHandler(db))
			r.Patch("/album/{id}", handlers.UpdateAlbumHandler(db))
			r.Patch("/album/{id}/image", handlers.ReplaceAlbumImageHandler(db))
			r.Patch("/album/{id}/aliases/primary", handlers.SetPrimaryAlbumAliasHandler(db))
			r.Patch("/album/{id}/artists/{artist_id}", handlers.SetPrimaryAlbumArtistHandler(db))

			r.Delete("/track/{id}", handlers.DeleteTrackHandler(db))
			r.Delete("/track/{id}/aliases", handlers.DeleteTrackAliasHandler(db))
			r.Delete("/track/{id}/artists/{artist_id}", handlers.DeleteTrackArtistHandler(db))
			r.Post("/track/{id}/merge", handlers.MergeTracksHandler(db))
			r.Post("/track/{id}/aliases", handlers.CreateTrackAliasHandler(db))
			r.Post("/track/{id}/artists", handlers.AddTrackArtistsHandler(db))
			r.Patch("/track/{id}", handlers.UpdateTrackHandler(db))
			r.Patch("/track/{id}/aliases/primary", handlers.SetPrimaryTrackAliasHandler(db))
			r.Patch("/track/{id}/artists/{artist_id}", handlers.SetPrimaryTrackArtistHandler(db))

			r.With(middleware.ProfileScope(db)).Post("/listens", handlers.SubmitListenWithIDHandler(db))
			r.With(middleware.ProfileScope(db)).Delete("/listens", handlers.DeleteListenHandler(db))

			r.With(middleware.ProfileScope(db)).Get("/user/apikeys", handlers.GetApiKeysHandler(db))
			r.With(middleware.ProfileScope(db)).Post("/user/apikeys", handlers.GenerateApiKeyHandler(db))
			r.With(middleware.ProfileScope(db)).Patch("/user/apikeys/{id}", handlers.UpdateApiKeyLabelHandler(db))
			r.With(middleware.ProfileScope(db)).Delete("/user/apikeys/{id}", handlers.DeleteApiKeyHandler(db))

			r.Get("/user", handlers.MeHandler())
			r.Patch("/user", handlers.UpdateUserHandler(db))

			r.With(middleware.ProfileScope(db)).Get("/export", handlers.ExportHandler(db))
			r.Delete("/data", handlers.PurgeAllDataHandler(db))
		})
	})

	r.Route("/apis/listenbrainz/1", func(r chi.Router) {
		r.Use(cors.Handler(cors.Options{
			AllowedOrigins: []string{"*"},
			AllowedHeaders: []string{"Content-Type", "Authorization"},
		}))

		r.With(middleware.IngestionKey(db)).
			Post("/submit-listens", handlers.LbzSubmitListenHandler(db, mbz))
		r.With(middleware.IngestionKey(db)).
			Get("/validate-token", handlers.LbzValidateTokenHandler())
	})

	// Serve Go documents and bundled assets independently of the working directory.
	var pages *webui.Pages
	var administration *adminui.Handler
	{
		var err error
		langDir, err := i18n.Directory()
		if err != nil {
			panic(fmt.Errorf("locate language catalogs: %w", err))
		}
		catalog, err := i18n.Load(langDir)
		if err != nil {
			panic(fmt.Errorf("load language catalogs: %w", err))
		}
		r.With(adminweb.SameOrigin).Post("/language", catalog.SetLanguage)
		pages, err = webui.NewHandler(db, webui.Options{Languages: catalog, DefaultTheme: cfg.DefaultTheme(), ForcedTimezone: cfg.ForceTZ() != nil})
		if err != nil {
			panic(fmt.Errorf("initialize Go pages: %w", err))
		}
		adminStore, ok := db.(adminui.Store)
		if !ok {
			panic("Go administration requires result-aware SQLite store")
		}
		settingsStore, err := cfg.NewSettingsStore(nil)
		if err != nil {
			panic(fmt.Errorf("initialize settings: %w", err))
		}
		administration, err = adminui.New(adminStore, adminui.Options{Languages: catalog, Settings: settingsStore, RequireSubmissionToken: true, MusicBrainz: mbz, LoginMiddleware: loginLimit, DisableRateLimit: cfg.RateLimitDisabled(), DefaultTheme: cfg.DefaultTheme()})
		if err != nil {
			panic(fmt.Errorf("initialize Go administration: %w", err))
		}
		r.Post("/admin/*", administration.ServeHTTP)
		r.Post("/{slug}/manage/{kind}/{id}/{action}", administration.ServeHTTP)
		r.Handle("/web-assets/*", http.StripPrefix("/web-assets", webui.Assets()))
		r.Handle("/favicon.svg", webui.Assets())
		goPageServer(r, pages, administration)
	}

	publicServer(r, "/public", webui.PublicFiles())
}

// goPageServer routes Go documents and returns 404 for paths outside their scope.
func goPageServer(r chi.Router, pages *webui.Pages, administration *adminui.Handler) {
	r.Get("/*", func(w http.ResponseWriter, req *http.Request) {
		if adminui.IsPage(req.URL.Path) || adminui.IsAction(req.URL.Path) {
			administration.ServeHTTP(w, req)
			return
		}
		if pages.RedirectLegacy(w, req) {
			return
		}
		if webui.IsPage(req.URL.Path) {
			pages.ServeHTTP(w, req)
			return
		}
		http.NotFound(w, req)
	})
}

func publicServer(r chi.Router, path string, root http.FileSystem) {
	if strings.ContainsAny(path, "{}*") {
		panic("FileServer does not permit any URL parameters.")
	}
	fs := http.FileServer(root)
	r.Get(path+"*", func(w http.ResponseWriter, r *http.Request) {
		r.URL.Path = strings.TrimPrefix(r.URL.Path, path)
		fs.ServeHTTP(w, r)
	})
}

func bindProfileReads(r chi.Router, db dbstore.DB) {

	r.Get("/artist/{id}", handlers.GetArtistHandler(db))                  // done
	r.Get("/artist/{id}/aliases", handlers.GetArtistAliasesHandler(db))   // done
	r.Get("/artist/{id}/interest", handlers.GetArtistInterestHandler(db)) // done

	r.Get("/album/{id}", handlers.GetAlbumHandler(db))                   // done
	r.Get("/album/{id}/artists", handlers.GetArtistsForAlbumHandler(db)) // done
	r.Get("/album/{id}/aliases", handlers.GetAlbumAliasesHandler(db))    // done
	r.Get("/album/{id}/interest", handlers.GetAlbumInterestHandler(db))  // done

	r.Get("/track/{id}", handlers.GetTrackHandler(db))                   // done
	r.Get("/track/{id}/artists", handlers.GetArtistsForTrackHandler(db)) // done
	r.Get("/track/{id}/aliases", handlers.GetTrackAliasesHandler(db))    // done
	r.Get("/track/{id}/interest", handlers.GetTrackInterestHandler(db))  // done

	r.Get("/top/tracks", handlers.GetTopTracksHandler(db))
	r.Get("/top/albums", handlers.GetTopAlbumsHandler(db))
	r.Get("/top/artists", handlers.GetTopArtistsHandler(db))

	r.Get("/listens", handlers.GetListensHandler(db))
	r.Get("/listen-activity", handlers.GetListenActivityHandler(db))
	r.Get("/first-activity", handlers.FirstActivityHandler(db))
	r.Get("/now-playing", handlers.NowPlayingHandler(db))
	r.Get("/stats", handlers.StatsHandler(db))
	r.Get("/search", handlers.SearchHandler(db))
	r.Get("/summary", handlers.SummaryHandler(db))
}
