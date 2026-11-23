# Changelog

## [Unreleased]

- Server configuration can generate the Navidrome (Subsonic) parameters: a dialog takes the Navidrome username and password, calculates the token in the browser and fills the parameters field, with a connection test in the same dialog. The password is never saved or sent to TuneTrail (ISSUE-127).

## [0.68] - 2026-11-22

- Changing the language on the sign-in card keeps you there with your destination, error text uses a higher-contrast tone in the Pearl theme, and checkbox rows in confirmations are 44 px touch targets (ISSUE-128).
- Catalog reviews, account deletion reviews and the "delete listens" confirmation are now cards: an impact summary, the list of what will change, and the confirmation alone in a Danger zone, in the page and in the dialog; who can open each one is unchanged (ISSUE-128).
- Errors, warnings and success messages share one component across sign-in, redemption, account tabs, server settings and the manage pages: same look in the Yuu and Pearl themes, alerts for errors, status for the rest, and fields point to their error (ISSUE-128).
- Sign-in has one address for every role, `/login`, shown as a centred card; "Sign in" links point to it, `/admin` keeps working, and `login` is now a reserved profile URL name (ISSUE-128).
- Sign in required, access denied and page unavailable use one neutral card with fixed messages and the links of your own session; a hidden profile and a missing one look the same (ISSUE-128).
- After changing your password the sign-in form now says "Password changed. Sign in with your new password.", and a protected page opened without a session says "Sign in to continue." (ISSUE-128).

## [0.67] - 2026-11-22

- No user-visible changes (ISSUE-194).

## [0.66] - 2026-11-22

- No user-visible changes (ISSUE-196).

## [0.65] - 2026-11-22

- Occurrence ingestion now handles listening history for every profile. Duplicate review is enabled by default, and schema 21 removes the old activation flags (ISSUE-172).

- Saving a relay creates a draft. Only listens received after network approval are eligible for delivery (ISSUE-172).

- File imports use occurrence ingestion, report oversized items individually, and preserve declared MusicBrainz IDs without overwriting existing IDs (ISSUE-172).

- Occurrence file imports trim oversized optional metadata at UTF-8 boundaries and report truncated fields (ISSUE-172).

- Koito v1 occurrence imports keep listens with large alias lists and report aliases skipped at the per-item limit (ISSUE-172).

- Occurrence ingestion links known artist MusicBrainz IDs to their existing artist and holds ambiguous IDs for review (ISSUE-172).

- Koito v1 imports through occurrence ingestion preserve explicit secondary aliases and count conflicting aliases they skip (ISSUE-172).

## [0.64] - 2026-11-22

- "My account" is now the single place to manage your account and profile, with five tabs: Account, Profile, API keys, History and Relay. The Profile tab keeps the name, URL name, listen protection (with the relay note) and avatar; the separate "My profile" menu entry is gone, and the own profile has one address, `/account/profile`, so `/admin?profile=<own id>` and its `section=` variants redirect there (ISSUE-126).
- Administrators get an Administration list of accounts and profiles with a Create account and profile card. Opening another account shows the same tabs in a context banner ("Maria (@maria) · you are operating as administrator", with a Back button); its Account tab holds the root tools (recovery token, role, deletion review), which ordinary users never receive (ISSUE-126).
- With JavaScript, saving the profile, avatar, API keys, the relay and the administrator role updates only the affected card; without it every form posts and returns to the same tab. Ingestion health stays in the main menu (ISSUE-126).

## [0.63] - 2026-11-22

- The account page moves to `/account` and opens for every signed-in user, not only administrators. It has Identity, Password and Session cards, warns before a password change that every session ends, and keeps `/admin?section=account` working by redirecting to it (ISSUE-125).
- "My account" in the menu and the post-login destination of ordinary users point to `/account`; `account` is now a reserved profile slug (ISSUE-125).
- Redeeming an account token now lists the password rules before sending, asks to confirm the password, and ends at the sign-in form with a "Password set" message. A stale or foreign form (expired page, another tab) shows "This page expired or was opened in another tab. Try again." with a fresh form instead of a permission error (ISSUE-125).
- Redeeming an account token no longer fails in a real browser: the page asked for `Referrer-Policy: no-referrer`, under which browsers send `Origin: null` on the form POST and the same-origin check refused it with a permission message. The redemption page now uses `same-origin`, and a refused origin on the sign-in or redemption form shows the expired-page text with a fresh form (ISSUE-125).
- Account cards and the redemption form reuse the shared colours, hover, pressed, focus and disabled states of the rest of the app in the Yuu and Pearl themes (ISSUE-125).
- The token redemption page opens with a notice that redemption needs a browser without an open account, with links to Sign out… and All profiles; when a session is open it names the signed-in user and asks to sign out first (ISSUE-125).
- The avatar Icon field when creating or editing a profile is now a dropdown with Music, Headphones, Disc, Radio, User and Initials (no icon); the current icon is preselected (ISSUE-125).

## [0.62] - 2026-11-22

- Ingestion health shows linked counts for open items in the selected profile and adds icons to the Inbox and Outbox headings (ISSUE-188).

## [0.61] - 2026-11-22

- Occurrence ingestion accepts listens without an album and counts them under Unknown release in track rankings grouped by release (ISSUE-186).
- Listens without an album retain the submitted artist credit when an alias matches multiple artists (ISSUE-186).

## [0.60] - 2026-11-22

- Profile owners and root can discard receipts without a listen in Ingestion health, individually or all at once with a single confirmation. Each discard leaves a minimal audit entry with 90-day retention (ISSUE-185).

## [0.59] - 2026-11-22

- Ingestion health separates incoming receipts from relay deliveries and shows receipt counts and missing metadata by API key, client, and time window (ISSUE-184).

## [0.58] - 2026-11-22

- Ingestion Health lets profile owners and root reprocess all parked receipts for a profile and logs each reprocessing with the account ID and time. Resolution attempts have a 100-second deadline, and receipts parked after exhausting their claim budget get five more claims when reprocessed (ISSUE-182).

## [0.57] - 2026-11-21

- Profile owners and root can turn on duplicate review in Ingestion health, inspect evidence, and decide individually or in batches. The revision records the deciding account. With review off, indistinguishable same-second repeats count once. Distinct trusted event IDs and distinct rows in a file remain separate in either mode. With review on, an exact trusted-ID replay of a resolved non-file submission can be reviewed (ISSUE-181).
- Ingestion health keeps rarely used sources in its last-submission list and paginates the source summary (ISSUE-181).

## [0.56] - 2026-11-21

- Track merges join only the recording identity and preserve both tracks, their releases and every listen. Administrators can undo a track merge for 30 days. Later track merge or undo operations prune expired entries (ISSUE-183).
- Download and restore TuneTrail v2 history archives with event identity, release information and provenance, including listens without a release. Each download excludes events added after its first page (ISSUE-183).

## [0.55] - 2026-11-21

- Top tracks are grouped by recording by default and can be split by release, including listens with an unknown release (ISSUE-135).

## [0.54] - 2026-11-21

- Active ingestion retries transient failures up to five attempts. Profile owners and root can reprocess parked receipts from the responsive Ingestion health page (ISSUE-173).

## [0.53] - 2026-11-21

- ListenBrainz submissions whose artist ID appears only in `mbid_mapping` now match the existing artist by that ID instead of creating a duplicate artist, in both the legacy writer and the active resolver. IDs in `additional_info` keep priority (ISSUE-170).

## [0.52] - 2026-11-21

- Profile owners and root can inspect recent ingestion sources, deduplication decisions, and relay failures, and open a listen’s provenance from its management page (ISSUE-083).
- **Health** now appears in the menu above server configuration (with a heart-pulse icon) and the page follows the style of the media edit dialog, with responsive layout and hover states. The account links under the profile cards are spaced out and centered (ISSUE-083).

## [0.51] - 2026-11-21

- The sign-out item in the menu is now labeled “Sign out” without the ellipsis and shows the exit icon like the other items (ISSUE-180).

## [0.50] - 2026-11-21

- Artist, album and track headers use a color derived from the cover again, instead of always showing the same blue-gray fallback (ISSUE-179).

## [0.49] - 2026-11-21

- Active ListenBrainz ingestion queues accepted events for durable relay delivery in batches of up to 100. Destinations pause independently and retry transient failures using backoff and server rate-limit hints. Shutdown preserves pending deliveries and uses an approximately eight-second shutdown budget. Outside the supplied Compose setup, allow a longer container stop timeout. Delivery is at least once. A crash after remote acceptance, a failed local commit, or an ambiguous cancellation can cause re-delivery. A late 2xx from a previous URL becomes `destination_changed`; a manual retry uses the new URL. Active replays of pre-upgrade receipts do not repeat the original raw POST (ISSUE-082).
- The root Server panel saves `allow_http` and `allow_private_network` in `conf.toml`. Both defaults are off and changes apply after restart. Root can use **Allow this relay** to review and confirm the permissions needed by a saved destination, and remove existing exceptions. Advanced CIDR rules remain available through the API. Global permissions apply to every profile. Loopback and link-local require destination exceptions, and forbidden addresses always stay blocked. These controls govern durable delivery and connection tests. Legacy, shadow, and playing-now forwarding keep the previous transport (ISSUE-082).
- Relay warnings report the configured authority and a stable reason, at most once per profile and reason per minute. DNS answers, URL paths, queries, and tokens are omitted. Root receives detailed validation results. Non-root validation uses broad classes. The admin relay GET returns the canonical URL and omits the saved token (ISSUE-082).
- **Upgrade action for existing relays:** The SQLite v11-to-v12 upgrade is one-way and keeps a `.pre-relay-delivery-*.bak` backup. TuneTrail 0.48 and earlier refuse v12. Roll back by restoring the v11 backup. Eligible administrator relays get seeded permissions. Under the default policy, existing HTTP or LAN relays on non-administrator profiles stop delivering until root grants the necessary exceptions. Failed DNS at startup schedules a new attempt after 30 seconds (ISSUE-082).
- The upgrade guides explain that physical SQLite backups preserve credentials and require the same protection as the live database (ISSUE-082).

## [0.48] - 2026-11-20

- No user-visible changes (ISSUE-092).

## [0.47] - 2026-11-20

- The database upgrades from v9 to v11 and creates a `.pre-occurrence-ingest-*.bak` backup. TuneTrail 0.46 and earlier cannot open a v11 database. Stop the service for the upgrade and keep free disk space of at least three times the database size in the data and `TMPDIR` directories; about 21 seconds were needed with 300,000 receipts. Restoring the backup loses writes made after the upgrade (ISSUE-134).
- Occurrence-based ingestion can preserve separate album appearances of the same recording across ListenBrainz submissions and supported imports. It is opt-in per profile and available for testing on disposable databases. Profiles with a configured relay cannot use it until durable relay delivery is available (ISSUE-134, ISSUE-082).

## [0.46] - 2026-11-20

- Signing out now asks for confirmation and returns you to the public directory. “Sign out…” is available on the account and 403 pages (ISSUE-157).

## [0.45] - 2026-11-20

- Navigation follows account and profile permissions. Listeners can open their own account, manage their own listens, and sign out from the menu. Sign-in returns to an authorized page or the account’s default destination. Opening a new page refreshes the sign-out form’s validity (ISSUE-163).

- The profile protection form explains that a configured relay continues sending listens to the external service (ISSUE-163).

- Legacy listening links use the same unavailable page as missing profiles. The directory announces which profiles are protected to screen readers (ISSUE-163).

## [0.44] - 2026-11-20

- Active ingestion of ListenBrainz submissions and imports requires the master switch plus per-profile activation, with a preview and a shadow purge confirmed by the root administrator. The Review option stays blocked until the review queue exists (ISSUE-081).

## [0.43] - 2026-11-19

- To migrate the database from v7 to v8, stop the service in a maintenance window and reserve at least three times the database size in both the database directory and the SQLite temporary directory (TMPDIR). The upgrade creates the automatic backup `.pre-recording-identity-*.bak`. Versions 0.41 and 0.42 refuse a v8 database. Restoring the backup loses writes made after the upgrade, and deliveries already made by the relay remain at the destination (ISSUE-133).

## [0.42] - 2026-11-19

- Administrators can manage accounts and profiles together, grant the administrator role, and issue single-use activation and recovery tokens (ISSUE-162).

## [0.41] - 2026-11-19

- Deleting listens also removes the matching ingest receipts and frees their keys for re-import (ISSUE-171).

- The ListenBrainz API answers HTTP 503 when receipt storage fails, so the batch can be resent (ISSUE-171).

## [0.40] - 2026-11-19

- The public directory lists every profile, and owners and administrators can require login to view a profile's listens and covers (ISSUE-161).

## [0.39] - 2026-11-19

- Users manage their own data, keys and relay, with imports that keep profiles isolated and block image lookups on the internal network; global administration and the last active administrator stay protected (ISSUE-160).

## [0.38] - 2026-11-19

- Navigation gains smooth transitions and prefetching of public pages of the same profile; charts show per-interval details without extra JavaScript (ISSUE-075).

## [0.37] - 2026-11-18

- Regular user accounts now authenticate with individual sessions, administration stays restricted to administrators, and the access-denied page offers a way to sign out (ISSUE-159).

## [0.36] - 2026-11-18

- Profiles now have individual accounts; deleting the last administrator's profile is blocked (ISSUE-158).

## [0.35] - 2026-11-18

- Compact history on mobile, ellipsis actions, and media pages with a cover gradient and section navigation (ISSUE-073).

## [0.34] - 2026-11-18

- No user-visible changes (ISSUE-080).

## [0.33] - 2026-11-17

- Home shows period KPIs with comparison, now playing at the top and a fixed period selector, with historical totals next to the fluid heatmap, cell tooltips and variations beside the numbers ("More", "Less" or "Same") (ISSUE-072).

## [0.32] - 2026-11-17

- Administrators see the empty page of a track, album or artist without listens, also on reload, instead of a 404 error; visitors still get 404 (ISSUE-150).

## [0.31] - 2026-11-17

- Administration actions are now grouped in the (+) menu, with no second sidebar, no push/overlay option and no repeated period selector (ISSUE-156).

## [0.30] - 2026-11-17

- The Home "Last played" list is full width again and the desktop sidebar is icons-only by default, with a label option in the theme panel; the Retrospective replaces History in the bar and the (+) menu closes when clicking outside; the activity chart has the same maximum width as the interest chart and fits the window without scrollbars; names and text of the highlight cards and top tracks use the full column width without breaking words; the sign-in button in the administration bar uses the key icon; the retrospective's "Show playback time?" option is centered in the column (ISSUE-154).
## [0.29] - 2026-11-16

### Changed
- The interface loads local fonts, without Google Fonts; the retrospective aligns weights and casing with the Koito reference (ISSUE-070).

## [0.28] - 2026-11-16

- Navigation with a bottom bar on mobile, an icon sidebar on tablet, and a label choice in the theme menu with a period selector on desktop (ISSUE-071).

## [0.27] - 2026-11-16

- Fixes identification by secondary track aliases and refuses ambiguous associations or ones with conflicting MusicBrainz IDs (ISSUE-131).

- Pages now adapt with fluid typography, shared spacing, three screen breakpoints and cards that respond to their available space (ISSUE-069).

## [0.26] - 2026-11-16

- No user-visible changes (ISSUE-144).

## [0.25] - 2026-11-16

- Administrators can drag a new album or artist cover onto its image, preview it, and confirm or cancel before the existing artwork form is submitted (ISSUE-142).

## [0.24] - 2026-11-16

- Deleting a track, album or artist now removes only that profile's matching listens, after a confirmation that shows the count and cannot be undone; the shared catalog and other profiles are untouched, and tracks shared with other artists are kept when an artist is deleted (ISSUE-143).

## [0.23] - 2026-11-16

- The history × opens listen details and an irreversible-deletion confirmation; the list and counts update without reloading (ISSUE-141).

## [0.22] - 2026-11-16

- Media pages now offer administrator action icons with tooltips and themed task dialogs, live merge search, optional listen timezone, and artwork and heading updates without reloading (ISSUE-140).

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
