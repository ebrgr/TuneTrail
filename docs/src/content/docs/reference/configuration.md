---
title: Configuration
description: Configuration options for TuneTrail.
---

TuneTrail supports environment variables and an optional `conf.toml` for the settings listed below. The other options continue to use environment variables.

## Optional `conf.toml`

Place `conf.toml` in the configured data directory, next to `tunetrail.db`. The
directory is selected by `KOITO_CONFIG_DIR` or `KOITO_CONFIG_DIR_FILE` before the
file is read; its default is `/etc/koito`. An absent file preserves the existing
environment-based configuration. Neither the file nor values are created automatically.

The supported schema is version 1. The repository also provides a fictional `deploy/conf.example.toml`; copy it to `conf.toml` in the data directory before adapting it. Use a subset of these settings as needed:

```toml
schema_version = 1

[logging]
level = "info"

[musicbrainz]
url = "https://musicbrainz.org"
rate_limit = 1
disabled = false

[images]
disable_deezer = false
disable_cover_art_archive = false
lastfm_api_key = ""

[subsonic]
url = ""
params = ""

[time]
force_tz = ""
```

A nonempty direct environment variable takes priority over its `_FILE` value,
then the TOML value, then the built-in default. An explicitly selected empty
`_FILE` does not fall back to TOML. Invalid TOML, unknown keys or versions, and
invalid combinations prevent startup; correct the file offline and restart.
The stored values are also validated before environment overrides are applied.

`time.force_tz` is the application-wide forced timezone. Leave it empty to use
the request/browser preference and then the process timezone (`TZ`). Subsonic
URL and params must both be set or both be empty. Subsonic is used to fetch
artwork and metadata, while ListenBrainz relay remains configured per profile
in SQLite. Sign in as administrator and open **Server configuration** at
`/admin/settings` to edit these ten global settings. The page shows saved,
effective and running values. Fields controlled by environment variables are disabled and identify the controlling variable, including the `_FILE` suffix where applicable. Remove the override and restart to edit the field. Restart TuneTrail to
apply effective changes. The Subsonic test uses the entered draft without saving;
relay settings remain on each profile's administration page.

### Deployment, upgrades and offline recovery

With the supplied Docker Compose file, `KOITO_DATA_DIR` is mounted at
`/etc/koito`. Place `conf.toml` inside that host directory; it persists across
container replacement along with `tunetrail.db`. Compose supplies `TZ` through
`KOITO_TIMEZONE`; it does not set `KOITO_FORCE_TZ`. The application setting
`time.force_tz` overrides the browser preference where a forced timezone is used.
An empty setting keeps the browser/request preference and falls back to process `TZ`.

Before upgrading, back up the data directory, including the SQLite database and
configuration, with the service stopped or using a SQLite-consistent backup.
Changing `KOITO_CONFIG_DIR` selects a different data directory; it does not move
existing data. Global changes take effect after a service or container restart.
Variables supplied by the host, Compose or VS Code keep their precedence after
restart. `reset-password` uses the same data directory and configuration, so an
invalid configuration must be corrected before running that command too.

If TuneTrail cannot start after a configuration change, stop the service, edit
or temporarily move `<data-directory>/conf.toml`, and restart it. Moving the file
restores environment/default behavior; it does not reset or delete the database.
Fix invalid environment or `_FILE` values as well. Preserve the original file for
inspection and verify the active values in Administration after recovery.

### Interface language files

TuneTrail loads the English and Brazilian Portuguese catalogs from the external
`langs` directory at startup. Set `TUNETRAIL_LANGS_DIR` to an absolute directory
to choose another location. If unset, TuneTrail looks next to the executable,
then in the Go checkout root. Docker ships the files in `/app/langs`. Keep the
files with the executable when distributing a build; missing or invalid catalogs
prevent startup. Restart after editing them. A missing Portuguese message falls
back to English. Adding another supported language currently also requires a
code change; copying an extra JSON file alone will not enable it.

Choose a language under **Settings → Appearance**; the choice is stored in a
browser cookie. Without JavaScript, open the **Settings** disclosure on the page. “Use browser language” follows `Accept-Language`. The public and administrative interfaces, displayed dates, numbers and relative times support English and Brazilian Portuguese. The preference does not change profiles or time zones.

##### KOITO_WEB_UI

- Default: `go`
- Values: `go`; other values prevent startup.
- Description: The Go interface provides server-rendered profile selection, home,
  rankings, listening history, search, artist/album/track details and rewind.
  Administration, profile management, import/export, media editing and reviewed
  deletion/merging also use Go forms. Use “Manage this item” to sign in and edit;
  history rows include a management link that works without JavaScript. Go mode
  returns 404 for unsupported routes. The former `react` value is no longer
  supported and must be removed from existing configurations. SQLite schema v3
  and its existing data remain unchanged.

The suffix `_FILE` is also supported for every environment variable. For example, `KOITO_SUBSONIC_PARAMS_FILE=/etc/koito/subsonic-params` loads the value of `KOITO_SUBSONIC_PARAMS` from that file.

:::caution
If the environment variable is defined without **and** with the suffix at the same time, the content of the environment variable without the `_FILE` suffix will have the higher priority.
:::

##### KOITO_DEFAULT_USERNAME

- Default: `admin`
- Description: The username for the user that is created on first startup. Only applies when running Koito for the first time.

##### KOITO_DEFAULT_PASSWORD

- Default: `changeme`
- Description: The password for the user that is created on first startup. Only applies when running Koito for the first time.

##### KOITO_DEFAULT_THEME

- Default: `yuu`
- Description: The lowercase name of the default theme to be used by the client. Overridden if a user picks a theme in the theme switcher.

##### KOITO_LOGIN_GATE

- Default: `false`
- Description: When `true`, Koito will not show any statistics unless the user is logged in.

##### KOITO_BIND_ADDR

- Description: The address to bind to. The default blank value is equivalent to `0.0.0.0`.

##### KOITO_LISTEN_PORT

- Default: `4110`
- Description: The port Koito will listen on.

##### KOITO_ENABLE_STRUCTURED_LOGGING

- Default: `false`
- Description: When set to `true`, will log in JSON format.

##### KOITO_LOG_LEVEL

- Default: `info`
- Description: One of `debug | info | warn | error | fatal`

##### KOITO_ARTIST_SEPARATORS_REGEX

- Default: `\s+·\s+`
- Description: The list of regex patterns Koito will use to separate artist strings, separated by two semicolons (`;;`).

##### KOITO_MUSICBRAINZ_URL

- Default: `https://musicbrainz.org`
- Description: The URL Koito will use to contact MusicBrainz. Replace this value if you have your own MusicBrainz mirror.

##### KOITO_MUSICBRAINZ_RATE_LIMIT

- Default: `1`
- Description: The number of requests to send to the MusicBrainz server per second. Unless you are using your own MusicBrainz mirror, **do not touch this value**.

##### KOITO_ENABLE_LBZ_RELAY

- Default: `false`
- Description: Legacy bootstrap only: set to `true` to associate the URL/token below once with a single eligible existing profile. With multiple profiles, configure each destination in Settings → Administration. This flag does not globally enable or disable saved per-profile relays. See [relay setup](/guides/scrobbler/#forward-a-profile-to-another-server).

##### KOITO_LBZ_RELAY_URL

- Required: `true` if relays are enabled.
- Description: Legacy relay base URL, for example `https://api.listenbrainz.org/1`. Used only for the one-time profile association; later edits belong in Administration.

##### KOITO_LBZ_RELAY_TOKEN

- Required: `true` if relays are enabled.
- Description: External user token for the legacy one-time relay association. Saved tokens are visible only to administrators. Removing these environment variables does not remove a saved relay; disable and save it in Administration.

##### KOITO_CONFIG_DIR

- Default: `/etc/koito`
- Description: The location where import folders and image caches are stored.

Automatic image downloads are limited to 10 MiB per image and 10 seconds per
request, with at most three redirects. Image-provider and MusicBrainz JSON
responses are limited to 8 MiB. A failed image download can fall back to the
default artwork; self-hosted HTTP(S) sources are supported. Images must be JPEG,
PNG or static WebP; animated images and other formats are not supported.

##### KOITO_FORCE_TZ

- Description: A canonical IANA database time zone name (https://en.wikipedia.org/wiki/List_of_tz_database_time_zones) that Koito will use to serve all clients. Overrides any timezones requested via a `tz` cookie or `tz` query parameter. Koito will fail to start if this value is invalid.

When this option is unset, a valid `tz` query parameter takes precedence over
a valid `tz` cookie; otherwise Koito uses the process timezone. Activity and
streaks use local calendar days. Weekly activity begins on Monday.

##### KOITO_DISABLE_DEEZER

- Default: `false`
- Description: Disables Deezer as a source for finding artist and album images.

##### KOITO_DISABLE_COVER_ART_ARCHIVE

- Default: `false`
- Description: Disables Cover Art Archive as a source for finding album images.

##### KOITO_DISABLE_MUSICBRAINZ

- Default: `false`

##### KOITO_SUBSONIC_URL

- Required: `true` if KOITO_SUBSONIC_PARAMS is set
- Description: The URL of your subsonic compatible music server. For example, `https://navidrome.mydomain.com`.

##### KOITO_SUBSONIC_PARAMS

- Required: `true` if KOITO_SUBSONIC_URL is set
- Description: The `u`, `t`, and `s` authentication parameters to use for authenticated requests to your subsonic server, in the format `u=XXX&t=XXX&s=XXX`. An easy way to find them is to open the network tab in the developer tools of your browser of choice and copy them from a request.
  :::caution
  If Koito is unable to validate your Subsonic configuration, it will fail to start. If you notice your container isn't running after
  changing these parameters, check the logs!
  :::

##### KOITO_LASTFM_API_KEY

- Required: `false`
- Description: Your LastFM API key, which will be used for fetching images if provided. You can get an API key [here](https://www.last.fm/api/authentication),

##### KOITO_SKIP_IMPORT

- Default: `false`
- Description: Skips running the importer on startup.

##### KOITO_DISABLE_RATE_LIMIT

- Default: `false`
- Description: When enabled, disables the rate limiter that Koito has on the `/apis/web/v1/login` endpoint.

##### KOITO_THROTTLE_IMPORTS_MS

- Default: `0`
- Description: The amount of time to wait, in milliseconds, between listen imports. Can help when running Koito on low-powered machines.

##### KOITO_IMPORT_BEFORE_UNIX

- Description: A unix timestamp. If an imported listen has a timestamp after this, it will be discarded.

##### KOITO_IMPORT_AFTER_UNIX

- Description: A unix timestamp. If an imported listen has a timestamp before this, it will be discarded.

##### KOITO_FETCH_IMAGES_DURING_IMPORT

- Default: `false`
- Description: When true, images will be downloaded and cached during imports.

##### KOITO_CORS_ALLOWED_ORIGINS

- Default: No CORS policy
- Description: A comma separated list of origins to allow CORS requests from. The special value `*` allows CORS requests from all origins.

:::danger
Environment variables below this notice are deprecated, and will not work with current versions of Koito.
:::

##### KOITO_SQLITE_ENABLED

- Deprecated as of: `v0.2.1`
- Deprecated because: SQLite is the only supported database as of `v0.2.1`.
- Default: `false` (will always be enabled in `v0.2.0` onwards)
- Description: Enables SQLite instead of PostgreSQL as the database engine. Automatically migrates data from PostgreSQL if `KOITO_DATABASE_URL` is also set on versions `v0.1.8` to `v0.1.10`.

##### KOITO_DATABASE_URL

- Deprecated as of: `v0.2.1`
- Deprecated because: PostgreSQL is no longer a supported database as of `v0.2.1`.
- Required: `true` if you have not yet migrated to SQLite (PostgreSQL support will be deprecated in `v0.2.0` onwards)
- Description: A Postgres connection URI. See https://www.postgresql.org/docs/current/libpq-connect.html#LIBPQ-CONNSTRING-URIS for more information. Having this set will cause automatic migration to SQLite on versions `v0.2.X`. On `v0.3.X` or higher, having this set will cause the application to fail to start.


### Contributing translations

Edit `langs/en.json` and `langs/pt-BR.json` together and restart TuneTrail. Message keys must match. Preserve every named placeholder such as `{{.Count}}`; do not translate placeholder names. Messages that depend on a quantity use JSON objects with `one` and `other` forms, for example:

```json
"quantity.plays": {
  "one": "{{.Count}} play",
  "other": "{{.Count}} plays"
}
```

Plural forms are selected using CLDR, with zero displayed in the plural. Keep user data, environment variable names, URLs, machine-readable timestamps and form values unchanged. Run `go test ./internal/i18n ./internal/webui` to validate the catalogs and presentation. Adding a third supported language also requires a code change.
