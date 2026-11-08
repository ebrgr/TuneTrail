package engine

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/ebrgr/TuneTrail/engine/middleware"
	"github.com/ebrgr/TuneTrail/internal/catalog"
	"github.com/ebrgr/TuneTrail/internal/cfg"
	"github.com/ebrgr/TuneTrail/internal/db"
	"github.com/ebrgr/TuneTrail/internal/db/sqlite"
	"github.com/ebrgr/TuneTrail/internal/images"
	"github.com/ebrgr/TuneTrail/internal/logger"
	mbz "github.com/ebrgr/TuneTrail/internal/mbz"
	"github.com/ebrgr/TuneTrail/internal/models"
	"github.com/ebrgr/TuneTrail/internal/utils"
	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/rs/zerolog"
)

func initLogger(getenv func(string) string, version string, w io.Writer) (*zerolog.Logger, context.Context, error) {
	if err := cfg.Load(getenv, version); err != nil {
		return nil, nil, fmt.Errorf("failed to load configuration: %w", err)
	}

	l := logger.Get()

	if cfg.StructuredLogging() {
		l.Debug().Msg("Engine: Enabling structured logging")
		*l = l.Output(w)
	} else {
		l.Debug().Msg("Engine: Enabling console logging")
		*l = l.Output(zerolog.ConsoleWriter{
			Out:        w,
			TimeFormat: time.RFC3339,
			FormatMessage: func(i any) string {
				return fmt.Sprintf("\u001b[30;1m>\u001b[0m %s |", i)
			},
		})
	}

	ctx := logger.NewContext(l)
	l.Info().Msgf("Koito %s", version)
	return l, ctx, nil
}

// connectDB reports database failures without creating an empty replacement.
func connectDB(l *zerolog.Logger) (db.DB, error) {
	l.Debug().Msg("Engine: Initializing database connection")
	var store *sqlite.Sqlite
	var err error
	// Retry only transient SQLite busy or locked errors; fail closed otherwise.
	for attempt := 0; attempt < 3; attempt++ {
		store, err = sqlite.New()
		if err == nil {
			break
		}
		var sqliteErr interface{ Code() int }
		if !errors.As(err, &sqliteErr) || (sqliteErr.Code()&255 != 5 && sqliteErr.Code()&255 != 6) {
			break
		}
		if attempt < 2 {
			time.Sleep(time.Second)
		}
	}
	if err != nil {
		return nil, fmt.Errorf("database initialization: %w", err)
	}
	l.Info().Msg("Engine: Database connection established")
	return store, nil
}

func Run(
	getenv func(string) string,
	w io.Writer,
	version string,
) error {
	l, ctx, err := initLogger(getenv, version, w)
	if err != nil {
		log.Fatalf("Engine: %v", err)
	}

	l.Debug().Msg("Engine: Starting application initialization")

	l.Debug().Msgf("Engine: Checking config directory: %s", cfg.ConfigDir())
	_, err = os.Stat(cfg.ConfigDir())
	if err != nil {
		l.Info().Msgf("Engine: Creating config directory: %s", cfg.ConfigDir())
		err = os.MkdirAll(cfg.ConfigDir(), 0744)
		if err != nil {
			l.Fatal().Err(err).Msg("Engine: Failed to create config directory")
			return err
		}
	}
	f, err := os.CreateTemp(cfg.ConfigDir(), ".koito_perm_check_*")
	if err != nil {
		l.Fatal().Err(err).Msg("Engine: Config directory is not writable")
		return err
	}
	f.Close()
	os.Remove(f.Name())
	l.Info().Msgf("Engine: Using config directory: %s", cfg.ConfigDir())

	l.Debug().Msgf("Engine: Checking import directory: %s", path.Join(cfg.ConfigDir(), "import"))
	_, err = os.Stat(path.Join(cfg.ConfigDir(), "import"))
	if err != nil {
		l.Info().Msgf("Engine: Creating import directory: %s", path.Join(cfg.ConfigDir(), "import"))
		err = os.Mkdir(path.Join(cfg.ConfigDir(), "import"), 0744)
		if err != nil {
			l.Fatal().Err(err).Msg("Engine: Failed to create import directory")
			return err
		}
	}

	if cfg.SqliteEnabled() {
		l.Info().Msg("Engine: The environment variable " + cfg.SQLITE_ENABLED + " is no longer needed and can be removed.")
	}

	store, err := connectDB(l)
	if err != nil {
		return err
	}
	defer store.Close(ctx)

	l.Debug().Msg("Engine: Checking for default user")
	userCount, _ := store.CountUsers(ctx)
	if userCount < 1 {
		l.Info().Msg("Engine: Creating default user")
		user, err := store.SaveUser(ctx, db.SaveUserOpts{
			Username: cfg.DefaultUsername(),
			Password: cfg.DefaultPassword(),
			Role:     models.UserRoleAdmin,
		})
		if err != nil {
			l.Fatal().Err(err).Msg("Engine: Failed to save default user in database")
		}
		apikey, err := utils.GenerateRandomString(48)
		if err != nil {
			l.Fatal().Err(err).Msg("Engine: Failed to generate default API key")
		}
		label := "Default"
		_, err = store.SaveApiKey(ctx, db.SaveApiKeyOpts{
			Key:    apikey,
			UserID: user.ID,
			Label:  label,
		})
		if err != nil {
			l.Fatal().Err(err).Msg("Engine: Failed to save default API key in database")
		}
		// The configured password must not be disclosed in startup logs.
		l.Info().Msg("Engine: Default administrator created")
	}

	if cfg.LbzRelayEnabled() && cfg.LbzRelayUrl() != "" && cfg.LbzRelayToken() != "" {
		if err := store.InitializeLegacyRelay(ctx, cfg.LbzRelayUrl(), cfg.LbzRelayToken()); err != nil {
			l.Warn().Err(err).Msg("Relay disabled for profiles without an explicit association")
		}
	}

	if cfg.ForceTZ() != nil {
		l.Debug().Msgf("Engine: Forcing the use of timezone '%s'", cfg.ForceTZ().String())
	}

	l.Debug().Msg("Engine: Initializing MusicBrainz client")
	var mbzC mbz.MusicBrainzCaller
	if !cfg.MusicBrainzDisabled() {
		mbzC = mbz.NewMusicBrainzClient()
		l.Info().Msg("Engine: MusicBrainz client initialized")
	} else {
		mbzC = &mbz.MbzErrorCaller{}
		l.Warn().Msg("Engine: MusicBrainz client disabled")
	}

	if cfg.SubsonicEnabled() {
		l.Debug().Msg("Engine: Checking Subsonic configuration")
		pingURL := cfg.SubsonicUrl() + "/rest/ping.view?" + cfg.SubsonicParams() + "&f=json&v=1&c=koito"

		resp, err := http.Get(pingURL)
		if err != nil {
			l.Fatal().Err(err).Msg("Engine: Failed to contact Subsonic server! Ensure the provided URL is correct")
		} else {
			defer resp.Body.Close()

			var result struct {
				Response struct {
					Status string `json:"status"`
				} `json:"subsonic-response"`
			}

			if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
				l.Fatal().Err(err).Msg("Engine: Failed to parse Subsonic response")
			} else if result.Response.Status != "ok" {
				l.Fatal().Msg("Engine: Provided Subsonic credentials are invalid")
			} else {
				l.Info().Msg("Engine: Subsonic credentials validated successfully")
			}
		}
	}

	l.Debug().Msg("Engine: Initializing image sources")
	images.Initialize(images.ImageSourceOpts{
		UserAgent:      cfg.UserAgent(),
		EnableCAA:      !cfg.CoverArtArchiveDisabled(),
		EnableDeezer:   !cfg.DeezerDisabled(),
		EnableSubsonic: cfg.SubsonicEnabled(),
		EnableLastFM:   cfg.LastFMApiKey() != "",
	})
	l.Info().Msg("Engine: Image sources initialized")

	if len(cfg.AllowedOrigins()) == 0 || cfg.AllowedOrigins()[0] == "" {
		l.Info().Msgf("Engine: Using default CORS policy")
	} else {
		l.Info().Msgf("Engine: CORS policy: Allowing origins: %v", cfg.AllowedOrigins())
	}

	if cfg.LbzRelayEnabled() && (cfg.LbzRelayUrl() == "" || cfg.LbzRelayToken() == "") {
		l.Warn().Msg("You have enabled ListenBrainz relay, but either the URL or token is missing. Double check your configuration to make sure it is correct!")
	}

	l.Debug().Msg("Engine: Setting up HTTP server")

	if len(cfg.AllowedHosts()) != 1 || cfg.AllowedHosts()[0] != "" {
		l.Info().Msg("Engine: The environment variable " + cfg.ALLOWED_HOSTS_ENV + " is no longer used and can be removed.")
	}

	var ready atomic.Bool
	mux := chi.NewRouter()
	mux.Use(middleware.WithRequestID)
	mux.Use(middleware.Logger(l))
	mux.Use(chimiddleware.Recoverer)
	mux.Use(chimiddleware.RealIP)
	bindRoutes(mux, &ready, store, mbzC)

	httpServer := &http.Server{
		Addr:    cfg.ListenAddr(),
		Handler: mux,
	}

	go func() {
		ready.Store(true)
		l.Info().Msgf("Engine: Listening on %s", cfg.ListenAddr())
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			l.Fatal().Err(err).Msg("Engine: Error when running ListenAndServe")
		}
	}()

	l.Info().Msg("Engine: Beginning startup tasks...")

	l.Debug().Msg("Engine: Checking import configuration")
	if !cfg.SkipImport() {
		go func() {
			RunImporter(l, store, mbzC)
		}()
	}

	l.Info().Msg("Engine: Pruning orphaned images")
	go catalog.PruneOrphanedImages(logger.NewContext(l), store)
	l.Info().Msg("Engine: Checking image cache migration status")
	go catalog.MigrateImageCache(logger.NewContext(l), store)
	l.Info().Msg("Engine: Running duration backfill task")
	go catalog.BackfillTrackDurationsFromMusicBrainz(ctx, store, mbzC)
	l.Info().Msg("Engine: Attempting to fetch missing artist images")
	go catalog.FetchMissingArtistImages(ctx, store)
	l.Info().Msg("Engine: Attempting to fetch missing album images")
	go catalog.FetchMissingAlbumImages(ctx, store)

	l.Info().Msg("Engine: Initialization finished")
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	l.Info().Msg("Engine: Received server shutdown notice")

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	l.Info().Msg("Engine: Waiting for all processes to finish")
	mbzC.Shutdown()
	if err := httpServer.Shutdown(ctx); err != nil {
		l.Fatal().Err(err).Msg("Engine: Error during server shutdown")
		return err
	}
	// Stop provider queues after active HTTP requests have drained.
	images.Shutdown()
	l.Info().Msg("Engine: Shutdown successful")
	return nil
}

func RunImporter(l *zerolog.Logger, store db.DB, mbzc mbz.MusicBrainzCaller) {
	profiles, err := store.ListListeningProfiles(logger.NewContext(l))
	if err != nil {
		l.Error().Err(err).Msg("Importer: cannot list profiles")
		return
	}
	for _, profile := range profiles {
		folder := path.Join(cfg.ConfigDir(), "import", fmt.Sprint(profile.ID))
		info, err := os.Lstat(folder)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil || !info.IsDir() {
			l.Warn().Msg("Importer: profile directory missing or symlinked")
			continue
		}
		files, err := os.ReadDir(folder)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			l.Error().Err(err).Msg("Importer: cannot read profile import directory")
			continue
		}
		ctx := db.WithProfile(logger.NewContext(l), profile.ID)
		for _, file := range files {
			if file.IsDir() {
				continue
			}
			filename := path.Join(fmt.Sprint(profile.ID), file.Name())
			if err := importProfileFile(ctx, store, mbzc, filename); err != nil {
				l.Error().Err(err).Msg("Importer: profile import failed")
			}
		}
	}
}
