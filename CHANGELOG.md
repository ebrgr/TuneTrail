# Changelog

## [Unreleased]

## [0.21] - 2026-11-15

- Media management now has separate task pages, preserves old addresses, and returns confirmed merges to the surviving item (ISSUE-139).

## [0.20] - 2026-11-15

### Changed

- The stylesheets in `assets/css/` are now split into smaller files that the server joins into the same `ui.css`; the page looks the same, but a custom web directory set with `TUNETRAIL_WEB_DIR` must follow the new layout listed in `assets/manifest.json` (ISSUE-068).

## [0.19] - 2026-11-15

### Changed

- Media management (aliases, metadata, artists, listens, artist and merge searches) updates only the edited section, keeping scroll and focus and refreshing the heading when the primary alias changes. Without JavaScript, actions return to the edited section instead of the top of the page. Artwork uploads still reload the page (ISSUE-124).

## [0.18] - 2026-11-15

### Added

- Show search results while typing in the search dialog, up to five artists, albums and tracks with a link to all results; arrow keys, Enter and Escape work from the keyboard and screen readers hear the result count. Without JavaScript the full search page works as before (ISSUE-107).

### Fixed

- Closing a dialog opened with a keyboard shortcut no longer leaves focus on a hidden field, so the next shortcut works (ISSUE-107).

## [0.17] - 2026-11-15

### Added

- Add a second sidebar opened from the administration icon: Sign in for visitors; Account, Profiles and Server configuration for administrators; and About. It opens beside the first sidebar on wide screens, below it on phones, and stays open in administration (ISSUE-123).
- The theme panel separates colors, the sidebar layout and the language into bordered sections; the new layout option chooses whether the second sidebar pushes the content aside or opens over it, remembered per browser (ISSUE-123).

## [0.16] - 2026-11-15

### Changed

- Replace the About icon with a theme icon that opens a simple panel with the themes, the custom theme and the language, without tabs; the panel also works in administration and opens with the \ key (ISSUE-110).
- The settings icon now links to administration; account, server and About move to a second sidebar in a later version (ISSUE-110, ISSUE-123). The administrator badge stays only on the profile avatar.

## [0.15] - 2026-11-15

### Added

- Show a shield badge on the profile avatar when an administrator session is active, linking to administration; visitors never receive it (ISSUE-109).

### Changed

- Public pages learn the administrator session from the server, so listen deletion controls appear without an extra request (ISSUE-109).

## [0.14] - 2026-11-14

### Fixed

- Accept an IPv6 bind address such as `::` or `[::]` (Koito #74; ISSUE-115).
- Accept a decimal `listened_at` from ListenBrainz clients and exports, keeping whole seconds (Koito PR #373; ISSUE-115).
- Send a semantic API version in the Subsonic connection test, which LMS requires (Koito #211; ISSUE-115).
- Finish imports when the import and archive folders are on different volumes (Koito #185; ISSUE-115).

## [0.13] - 2026-11-14

### Fixed

- Show the Home top artists, albums and tracks side by side from about 1318px wide instead of only from 1536px; narrower windows keep the stacked layout (ISSUE-113).

## [0.12] - 2026-11-14

### Changed

- Cache versioned interface assets by content and restore public history navigation while revalidating administrative controls and sessions (ISSUE-106).

## [0.11] - 2026-11-14

### Changed

- Load the web interface from an external assets directory distributed with the executable; templates and static files are validated at startup and updates take effect after restart (ISSUE-066).

## [0.10] - 2026-11-14

### Added

- Configuration variables can use the TUNETRAIL_ prefix (and TUNETRAIL_*_FILE); KOITO_ names keep working with a startup warning and a DEPRECATED mark on the settings page, and different values under both names stop startup without revealing them (ISSUE-111).

### Changed

- The default data directory is `/etc/tunetrail`; installations that still mount `/etc/koito` keep using it with a log notice until 2.0, and TuneTrail refuses to start if both hold a database (ISSUE-111).
- The supplied compose project and service are renamed to `tunetrail`; stop the old `koito-sum` project before starting the new one (see DEPLOYMENT.md). A second server on the same data directory now refuses to start (ISSUE-111).
- Exports are named `tunetrail_export.json` and the import page offers the TuneTrail format; Koito exports are still accepted (ISSUE-111).
- The container image no longer presets configuration variables and the supplied compose file passes `TUNETRAIL_` names; an existing `deploy/.env` keeps working unchanged (ISSUE-111).
- The administrative session cookie is now `tunetrail_session`; administrators sign in again once after upgrading (ISSUE-111).
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
