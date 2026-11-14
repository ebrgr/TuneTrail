# Changelog

## [Unreleased]

### Changed

- TuneTrail now identifies itself as TuneTrail, with its own project contact, to MusicBrainz, Deezer, Last.fm and Subsonic servers (ISSUE-111).

## [0.9] - 2026-11-14

### Fixed

- Removing a track artist now removes its derived album participation while preserving independent and legacy album credits, without a database migration (ISSUE-049).

### Added

- Step-by-step guide to recover an interrupted or refused database migration and to roll back, in DEPLOYMENT.md (ISSUE-064).

## [0.8] - 2026-11-13

### Added

- Localize displayed dates, numbers, relative times and plural quantities in English and Brazilian Portuguese.

- Translate administrative pages, notices and accessibility labels using the external language catalogs.

- Translate public navigation, listening pages and rewind into Brazilian Portuguese using external language catalogs.

- Add browser language selection in settings and external English and Brazilian Portuguese catalogs for the profile-selection screen.

- Provide a sample TOML configuration and offline recovery and upgrade instructions.

- Add an authenticated server configuration page with saved, active and override states, plus a non-saving Subsonic connection test.

- Persist `conf.toml` changes atomically with revision tracking and comment preservation.

- Read optional `conf.toml` settings with environment and `_FILE` precedence, validation and bounded Subsonic startup checks.

- Go administration includes login/account, profiles, avatars, keys,
  relay, import/export, catalog editing and reviewed deletion/merging. Forms work
  without JavaScript, preserve profile context and reject duplicate submissions.
- Listening history links to exact-record management with filtered return, and
  the Go editor supports artist/merge search, timezone selection and artwork drop.

- Isolated Go visual preview for the interface migration, with fictional profiles,
  existing themes and cards; available through `make ui.preview` without a database.
- `KOITO_WEB_UI=go` renders profile selection, home, rankings and listening
  history from real scoped data, with search, appearance and existing admin actions.

### Changed

- Improve rewind ranking spacing and keep all items visible at every screen size.

- Refine rewind accent lines and constrain ranking details to the image height.

- Align the rewind card, year navigation and theme-colored accents with the original layout.

- Keep all rewind months visible, move period modes above the summary and apply the listening-time option immediately.

- Use “reproduções” for listening records in the Brazilian Portuguese interface.

- Disable configuration fields controlled by environment variables and show the controlling variable name.

- Show profile relay tokens as ordinary editable text and require both relay fields when saving or testing.

- Migrate legacy `koito.db` databases to `tunetrail.db` with a consistent backup, schema v2-to-v3 upgrade, and conflict protection.

- Name the built application executable `tunetrail`.

- Use the TuneTrail Go module path and update repository and container references.

- Publish the `latest` container tag alongside each versioned image.

- Update project links and the target container image for the TuneTrail repository.

- Adopt TuneTrail 0.x versioning, starting at 0.3, independently of the Koito engine version.

- Rebrand the application as TuneTrail with updated browser icons and project documentation.

- The application builds and runs without CGO or libvips, including in Docker;
  the optional race detector still requires a C compiler.

- Artwork and avatars use a Go image pipeline. JPEG, PNG and static WebP are
  supported; animated images and other formats are rejected. Existing WebP
  cache files remain readable.

- Removed the React application and its build; the Go interface and public assets
  are bundled with the server. Existing `KOITO_WEB_UI=react` must be removed.

### Fixed

- Add breathing room between dropdown arrows and the edge of selection fields.

- Prevent the listening-interest curve from overshooting its values and disappearing at the graph boundaries.

- Automatic artist and album images are protected from orphan cleanup while
  their catalog associations are being saved.

- Administrative image uploads now limit waiting requests before reading their
  bodies, and report oversized files and unsupported formats consistently.
  Concurrent requests for the same image can cancel independently.

- Go administration now uses the same responsive navigation style as public pages.
- The default VS Code debug configuration starts Go directly and opens its
  address, without waiting for the React development server.
- Administrative controls and links in the Pearl theme have more readable
  text contrast and focus outlines.
- Profile selection and personal search remain reachable from Go navigation
  when JavaScript is disabled.
- Successful ListenBrainz submissions and storage errors now send the JSON
  content type in the HTTP response headers.
- ListenBrainz submissions now reject request bodies over 16 MiB before
  buffering or forwarding them; the existing 1,000-listen limit remains.
- A failed artist-to-album association is reported instead of saving a listen
  as if the association had succeeded.
- Concurrent listens with different metadata no longer collide when names
  contain separator characters.
- Automatic image downloads and provider responses now have size and timeout
  limits, image files are published atomically, and failed downloads return a
  fallback to each requester. Provider queues cancel pending work at shutdown.
- Profile and player-key lookups report database failures as server errors
  instead of incorrectly claiming that a profile or key does not exist.
- A stale now-playing lookup or catalog confirmation no longer removes a newer
  playback update from the same profile.
- Upload imports distinguish invalid files from storage failures while warning
  that some records may already have been saved.
- Media alias and artist-relation endpoints return not found for media outside
  the selected profile, instead of reporting a server error.
- Listening activity, streaks and active days now use local calendar dates,
  including fractional offsets and daylight-saving transitions. Month and ISO
  week filters resolve in the selected timezone, and include the last second.
- Activity graphs now fill the correct default, Monday-based weekly and annual
  ranges and retain the process timezone when no timezone is supplied.

- Administrator credentials and request query parameters are no longer written
  to startup or access/search logs. Ambiguous logout cookies are rejected and
  cross-origin rejections prevent caching.
- Catalog deletion and merges no longer sweep unrelated unplayed tracks or
  artists; an unplayed merge destination is retained.
- Profile deletion distinguishes stale confirmation from database failures and
  checks image references before removing a cached avatar.

- Editing track artists uses one transaction and no longer removes unrelated
  unplayed catalog entries; missing relations cannot clear an existing primary.
- Artwork replacement keeps shared images, cleans up failed publications, and
  bounds downloads, redirects, upload size and image dimensions.
- Administrative alias edits reject removal of a primary or missing alias;
  listen deletion rejects overflowing track identifiers.

- Profile import uploads no longer enter the startup import queue, report archive
  failures, respect cancellation and enforce JSON and ZIP resource limits.
- Exports include pre-1970 listens and stop cleanly on stream errors without
  appending an error document to a partially written download.

- Profile relay validation and forwarding share bounded timeouts and responses,
  reject redirects, and retry server errors with the complete payload.
- Avatar publication is protected from concurrent orphan collection; a failed
  association lookup no longer causes that collector to delete images.

- Catalog merges reject invalid, identical or missing source/destination IDs,
  preserve transactional reads with a single SQLite connection, and return
  public errors without database details.
- Rewind and summary API averages now count the full selected calendar period,
  including leap days and daylight-saving changes. All-time averages use the
  profile’s first listen through the current date; rolling ranges use elapsed days.
- Dragging artwork onto a track editor targets the track’s album rather than the
  track ID; image handling follows the active item when switching pages.
- Version checks now read the multiuser design documents from `docs-dev/`.
- Chart filters retain their profile slug; pagination starts at one, date labels
  use the queried interval, and history refreshes after administrative deletion.
- Listening history uses a deterministic tie-break for simultaneous plays and
  propagates count-query failures instead of returning misleading totals.
- Media details propagate count, duration and rank-query failures; their API
  distinguishes missing items (404) from internal failures (500), and rejects
  nonpositive IDs (400).
- Interest intervals now match their listen counts, include the final instant,
  exclude future listens, and reject bucket counts outside 1–512 before allocating.
  Internal interest-query errors no longer expose database details in the API.

### Changed

- Administrator sessions now have fixed 24-hour or 30-day lifetimes matching
  their cookies. Changing the password revokes all sessions and requires login
  again, including logins racing with the password change.
- Go Home and the prepared media pages share activity calendars with accessible
  data tables; media interest uses server-rendered SVG and a non-JavaScript table.
  Media details, rewind and management links are served by Go.
