---
id: doc-022
title: 'Design study: listening profiles with a single administrator'
type: specification
created_date: '2026-11-12T23:40:00Z'
updated_date: '2026-11-12T23:40:00Z'
tags:
  - design
---
> Importado de [docs-dev/MULTIUSER_DESIGN.en.md](../../../docs-dev/MULTIUSER_DESIGN.en.md).

# Design study: listening profiles with a single administrator

Historical document version: **Koito 0.3.2-SUM-RC3**. The current TuneTrail version is controlled by `internal/version/VERSION`.

[Portuguese version](../../../docs-dev/MULTIUSER_DESIGN.md)

Status: implementation steps 1–5 have been completed and validated for the local
MVP. The complete workflow has been exercised on a fresh installation, a
consolidated v2 → v3 migration, and a copy of an example database. **The current
schema version is 3.** Historical references to version 4 describe the earlier
migration layout, before the consolidation recorded at the end of this document.

Navigation uses explicit profiles. Management is a tab inside
**Settings → Administration**, with `/admin` available for direct access, and
requires an independent administrator session. Validation limits and remaining
legacy compatibility are recorded under step 5.

## Goals and user experience

Separate listening records (scrobbles), without storing audio or requiring each
listener to register or sign in. Preserve Koito's existing pages, charts, and
themes. Each person has a public profile with a display name, URL identifier,
and photo or icon. Only administration requires a password.

- `/`: profile selection, showing avatars and names. Every explicit visit to the
  root displays this selection, even if a profile was used previously.
- `/default`: the profile that receives the existing history from an installation
  with a single user.
- `/<user>`: the profile's home page.
- `/<user>/listens`, `/<user>/rewind`, and
  `/<user>/chart/top/artists`: the existing pages with that profile's data.
- `/<user>/artist/123`, `/<user>/album/456`, and
  `/<user>/track/789`: shared catalog entries with personal metrics.
- `/admin`: administrator sign-in and profile/key management. The menu entry is
  the **Settings → Administration** tab, whose content appears inside Settings.
  There is no separate administration icon in the main sidebar.

Add a profile control to the existing narrow sidebar. The avatar opens a selector
with photos/icons and names without permanently widening the sidebar. On mobile,
use the same selector in the existing navigation bar. Show the profile name in
the page header so ownership of the displayed data is clear.

Switching profiles updates the URL and all queries. Preserve the current screen
and time period when possible, and close editing dialogs when switching. Use the
URL as the source of truth so two tabs can display different profiles. Do not
use a global “current profile” cookie to determine backend results.

## Identity: administrator, profile, and API key

Keep three concepts separate:

1. Administrator account: authenticates management actions using a password and
   session.
2. Listening profile: a public identity with no password or session of its own.
3. API key: an ingestion credential linked to exactly one profile.

A profile can have several keys, such as “Navidrome” and “Phone”. All contribute
to the same listening history. Create an initial key together with the profile.
Generate, inspect, and revoke credentials only through administration; public
profile responses must never include tokens. Selecting a profile grants no
write permissions.

Submission continues to use `/apis/listenbrainz/1/submit-listens` and
`Authorization: Token ...`, preserving player compatibility. The server resolves
`API key → profile → listening record`. A profile supplied in the payload cannot
override the key's ownership.

Revoking a key blocks future submissions while preserving history. Replacing a
key also preserves listens. In the MVP, keys cannot be transferred between
profiles: generating a new key avoids ambiguity about previous history.

## Proposed data model

Create `listening_profiles` separately from `users`, avoiding fictitious listener
passwords or turning public profiles into administrator accounts.

| Field | Type/rule | Purpose |
| --- | --- | --- |
| `id` | INTEGER PK | Stable internal identity |
| `slug` | TEXT NOT NULL UNIQUE | URL segment: `<user>` |
| `display_name` | TEXT NOT NULL | Visible name, allowing spaces and accents |
| `avatar_image_id` | Optional TEXT | Reference to a locally managed photo |
| `avatar_icon` | Optional TEXT | Icon from an allowlist |
| `created_at` | INTEGER NOT NULL | Creation time in Unix seconds |

Prefer a photo when available, otherwise an icon, otherwise the name's initials.
Slugs use lowercase ASCII letters, numbers, and hyphens, with no spaces. Reserve
technical route names such as `admin`, `apis`, `image`, `public`, `assets`, and
legacy paths. A display name can change without changing the URL. Under the
rule added in step 2, administrators can also explicitly edit a slug, either
with the name or separately. The profile ID and its history/key associations
remain stable. The old URL returns 404; it does not redirect to another profile.

Changes to the rest of the database:

- `api_keys.profile_id`: mandatory FK to `listening_profiles.id` after existing
  rows have been backfilled.
- `listens.profile_id`: mandatory FK to `listening_profiles.id`.
- `listens` PK: `(profile_id, track_id, listened_at)`.
- History index: `(profile_id, listened_at)`; also evaluate
  `(profile_id, track_id, listened_at)` for track queries.
- Sessions remain associated with the administrator account, never the profile
  being viewed.
- Artists, albums, tracks, aliases, and cover images remain shared.

Legacy `user_id` fields require an explicit migration. During the transition,
they can remain as legacy associations, but all new listening queries must use
`profile_id`. The proposed final cleanup removes these associations from
`listens` and `api_keys`; deleting an administrator account must not delete a
profile's history. Do not modify migrations that have already been applied.
The explicitly authorized development consolidation is documented separately
below, including its requirement to restore a v2 database first.

Storing the key on each listen is unnecessary for separating profiles:
`profile_id` is sufficient. A future optional `listens.api_key_id` could provide
source auditing. In that case, key deletion should use `ON DELETE SET NULL`,
while preserving the listen's `profile_id`. Never store a secret token inside
every listening record.

## Reads and permissions

Proposed public API:

```text
GET /apis/web/v1/profiles
GET /apis/web/v1/profiles/{slug}
GET /apis/web/v1/profiles/{slug}/listens
GET /apis/web/v1/profiles/{slug}/stats
GET /apis/web/v1/profiles/{slug}/top/artists
GET /apis/web/v1/profiles/{slug}/now-playing
GET /apis/web/v1/profiles/{slug}/summary
```

Extend this pattern to the remaining read endpoints. Resolve slugs to internal
IDs in the backend. A missing profile must return 404, without falling back to
global data or the default profile. Public search shows catalog entries related
to that profile's listens; charts and metrics use only its listening history.

| Action | Visitor | Ingestion key | Administrator session |
| --- | --- | --- | --- |
| List profiles and read statistics | Yes | Not required | Yes |
| Submit a scrobble / playing now | No | Linked profile only | Does not replace a key |
| Create/edit/delete profiles and manage keys | No | No | Yes |
| Edit catalog entries, aliases, images, and merges | No | No | Yes |
| Delete listens / import or export history files | No | No | Yes, with an explicit profile |
| Global cleanup | No | No | Separate administrative action |

Listening data is public to anyone who can access the instance, including the
profile list and now-playing state. This supports browsing without login;
private profiles are outside the MVP. History downloads require administration
in the first version, even though public queries expose listens.

Authentication must no longer mean that a player's API key authorizes global
editing. Introduce administrator middleware requiring a session and the admin
role. Review routes previously protected by `AuthModeSessionOrAPIKey`. Also
protect cookie-authenticated administrative mutations from cross-origin requests.

Subsonic image credentials can remain global. ListenBrainz relay configuration
must be per profile when used; do not implicitly forward every profile's listens
to a single external account. Explicitly associate the legacy relay with the
original profile during migration.

## Creation, deletion, and existing data

Administrator creation flow: name → suggested/editable slug → photo/icon → create.
Show the initial key and scrobbling endpoint for copying. Allow additional keys
and revocation when editing the profile. Do not ask listeners for passwords.

Deletion requires confirmation of the name and affected listen count. Delete
the profile, keys, listens, owned avatar, and temporary now-playing state. Keep
catalog entities used by other people. Orphan cleanup must consider references
from all profiles. Allow deleting the last profile and display an empty state;
administration remains accessible. `default` is the migration profile, not an
automatic destination for invalid keys or data without an owner.

Migration from an existing installation:

1. Back up the database before migration and backfill associations transactionally.
2. If there is one existing user with data/keys, create `default` and associate
   the existing history and keys without changing tokens.
3. If there are several existing users, create a profile for each, resolve slug
   conflicts, and preserve all associations.
4. Check totals, foreign keys, and the absence of listens/keys without a profile.
5. Rebuild the `listens` primary key and update indexes.
6. Preserve the administrator account and password. Legacy non-administrator
   accounts must not gain privileges; do not introduce listener login.
7. Redirect old page links to the defined legacy profile. With several profiles,
   require a destination choice rather than combining histories.

If several people's data was already stored under the same `user_id`, the old
database does not record which key produced each listen. API keys alone cannot
reconstruct that separation reliably; external sources or an administrator-defined
attribution rule would be required.

## Identified code impacts

- `client/app/routes.ts`: profile layout using `:slug`, root selection, and `/admin`.
- `client/app/components/sidebar/Sidebar.tsx`: profile selector in the existing design.
- `client/app/providers/AppProvider.tsx`: separate `adminSession` from `activeProfile`.
- `client/api/api.ts` and loaders: explicit profile endpoints.
- Existing absolute links (`/track/...`, `/rewind`, etc.): preserve the profile prefix.
- React Query: include the profile ID in every personal query key; cancel old
  requests and prevent temporarily showing another profile's cached data.
- `engine/routes.go` and `engine/middleware`: separate public reads, ingestion,
  and administration.
- `internal/db/opts.go`, `db.go`, and `sqlite`: require profile scope in personal
  queries, including media-page metrics and search.
- `internal/summary`: actually apply the scope; at the start of this study it
  accepted `userId` without applying it, while the handler supplied `1`.
- `engine/handlers/now_playing.go`: remove the hardcoded `"1"` key.
- `internal/catalog`: key now-playing cache entries by profile.
- `internal/importer` and `RunImporter`: explicit destination profile and removal
  of fixed IDs. Prefer administrative imports; automatic directories, if retained,
  must have explicit profile associations.
- Export: use the profile being managed rather than the administrator account.
- Catalog deletion/merge/corrections: preserve different profiles' scrobbles at
  the same track/time and invalidate affected caches.

## Implementation and validation plan

1. Database and migration: profiles, associations, and listen uniqueness.
2. Backend: key identification, scoped queries, permissions, and removal of fixed
   IDs, including import/export. Allow profile name and slug changes without
   changing internal identity or associated keys.
3. Interface: slug layout, sidebar selector, root selection, and profile-scoped cache.
4. Administration: creation, avatar/icon, keys, deletion, and import destination.
   - Issue resolved and validated in step 5: administrator name/password changes
     followed by a new login; profile name/slug are edited separately.
5. Validate the complete workflow on both migrated and fresh installations.

Acceptance criteria:

- Two profiles can record the same track in the same second without losing events.
- Two keys belonging to the same profile contribute to the same history.
- History, rankings, activity, rewind, and now playing respect profile boundaries.
- Direct access, page refreshes, and two tabs retain the correct profile.
- Profile switching preserves the visual design without showing old cached data.
- Visitors and ingestion keys cannot perform administration.
- Revoking a key preserves history; deleting a profile does not delete others' data.
- Migration preserves administrator passwords, existing tokens, and listens.
- Public selection never reveals tokens or session data.

Outside the MVP: listener login, public registration, private profiles, individual
editing permissions, per-person catalog customization, and per-key listen auditing.

## Implementation record — step 1

- Transactional migration `003_listening_profiles.sql`; migrations 001/002 unchanged.
- `listening_profiles` stores public metadata, unique/validated slugs, and IDs
  that are not automatically reused after deletion.
- `default` for one legacy account; `user-<id>` for several accounts, retaining
  display names and avoiding conflicts with reserved or similar names.
- Mandatory `profile_id` in `api_keys` and `listens`, cascading deletion only from
  the profile, and uniqueness on `(profile_id, track_id, listened_at)`.
- `legacy_user_profiles` and a legacy-account profile-creation trigger provide
  temporary compatibility with administrator bootstrap and existing callers.
  The bridge grants no administrative permissions to a profile.
- Optional legacy `user_id` attribution remains in both tables, without a FK
  that deletes data when an account is removed. Writes accept `ProfileID`;
  existing `UserID` callers resolve the bridge without falling back to `default`.
- Before migrating an existing database, `sqlite.New` creates a consistent
  snapshot with `VACUUM INTO`, including WAL, beside the database:
  `koito.db.pre-profiles-*.bak`, with permissions 0600. Backup failure prevents migration.
- No destructive SQL downgrade is provided. To revert, stop all connections and
  restore a complete backup using a SQLite restoration procedure. Restoration
  returns to the pre-migration state and does not include subsequent writes.
- Tests cover fresh installation, single/multiple-user migration, integrity,
  duplicates, credential preservation, backup, rollback, and deletion lifecycle.
- Optional test using a copy of an example database, with the source opened read-only:

  ```sh
  KOITO_MIGRATION_TEST_DB="$PWD/.local/koito/koito.db" go test ./internal/db/sqlite -run TestProfileMigrationExampleCopy -v
  ```

Historical step 1 status: migration runs automatically when the API starts again.
At that point `/default` pages and the sidebar selector were still pending and
were implemented in step 3. Identifying keys without a legacy account and
isolating queries were implemented in step 2.

## Implementation record — step 2

### Identification and authorization

- The ListenBrainz API uses `IngestionKey`: it validates the token directly in
  `api_keys` and resolves `profile_id`, without depending on `users` or creating
  listener passwords. The payload cannot choose the destination. `/validate-token`
  returns the profile's current slug.
- Administration requires a valid session cookie and `role=admin`. An ingestion
  key, even a legacy key associated with the administrator, cannot authorize
  key management, export, import, deletion, or catalog changes.
- Login accepts administrator accounts only. The old middleware combining
  session and API-key authorization was removed. `KOITO_LOGIN_GATE` does not
  require login for profile reads: profiles are public in this model.
- Web mutations validate `Origin` and `Sec-Fetch-Site`; administrator cookies use
  `SameSite=Strict` and `Secure` over HTTPS. The Vite proxy preserves `Host`
  (`changeOrigin=false`) so this validation also works during development.
- Key revocation and label changes verify the selected profile and return 404
  when the key belongs to another profile.

### Available HTTP contract

All paths below use the `/apis/web/v1` prefix:

| Method and route | Access and effect |
| --- | --- |
| `GET /profiles` | Public: lists public metadata without tokens |
| `GET /profiles/{slug}` | Public: resolves the profile or returns 404 |
| `GET /profiles/{slug}/listens`, `/stats`, `/listen-activity`, `/first-activity` | Profile history and metrics |
| `GET /profiles/{slug}/top/tracks`, `/top/albums`, `/top/artists` | Personal rankings |
| `GET /profiles/{slug}/artist/{id}`, `/album/{id}`, `/track/{id}` | Shared metadata with personal metrics; media absent from the profile's history returns 404 |
| Media alias, artist, and interest subroutes | Same profile context; aliases/associations verify a relationship with the profile |
| `GET /profiles/{slug}/search` | Search restricted to the catalog heard by this profile |
| `GET /profiles/{slug}/now-playing`, `/summary` | Profile's current state and rewind |
| `PATCH /profiles/{slug}` | Admin: changes `display_name` and/or `slug` |
| `GET /profiles/{slug}/export` | Admin: exports this profile only |
| `POST /profiles/{slug}/listens`, `DELETE /profiles/{slug}/listens` | Admin: writes/deletes listens within this profile only |
| `GET/POST /profiles/{slug}/user/apikeys` | Admin: lists/generates this profile's keys |
| `PATCH/DELETE /profiles/{slug}/user/apikeys/{id}` | Admin: renames/revokes the associated key |
| `POST /profiles/{slug}/import` | Admin: imports a file placed in the profile's directory; step 4 also adds upload support |
| `PUT/DELETE /profiles/{slug}/relay` | Admin: configures/removes this profile's relay; does not return the token |
| `GET /profiles/{slug}/relay` | Admin: reads saved configuration, including the visible token; `Cache-Control: no-store` |
| `POST /profiles/{slug}/relay/test` | Admin: validates draft credentials without saving or submitting listens |

Legacy read routes without a profile remain available only when exactly one
profile exists. With zero or several profiles, they return 409 requesting an
explicit choice, never global data. Global catalog operations and full cleanup
remain on the original routes, protected by an administrator session. Browser
page routes were adapted in step 3.

Example administrative update:

```http
PATCH /apis/web/v1/profiles/default
Content-Type: application/json
Cookie: koito_session=...

{"display_name":"Everton Berger","slug":"evertonberger"}
```

The frontend must use the returned slug. The previous URL returns 404; tokens,
listens, import directory, and relay stay associated with the same ID. Invalid,
reserved, or duplicate slugs are rejected without partial changes.

### Query isolation and temporary state

- `db.WithProfile` carries the internal ID in the request context. Middleware
  resolves the URL before handlers and queries run. No global selection cookie
  is used, and payload parameters cannot replace this ID.
- SQLite read queries use a statement-local CTE filtering `main.listens` by
  `profile_id`. The same scope applies to count, ranking, first-listen, interest,
  streak, and summary subqueries. Internal catalog jobs can read globally with
  an unscoped context; public statistics routes do not expose that option.
- Search checks participation in listening history. Direct reads of media and
  its aliases/associations verify a relationship with the profile.
- `summary.GenerateSummary` applies its supplied ID. Importers, export, and
  now-playing reads no longer use the fixed ID `1`.
- Now playing uses `now-playing:<profileID>`. A track without a completed scrobble
  can appear as currently playing for that profile, with zero personal metrics.
  If the media has been deleted, obsolete state is removed instead of returning 500.
- The global 30-minute statistics cache was removed: queries return current data
  without mixing profiles or retaining stale counts after deletion. Empty profiles
  return finite numeric values, avoiding division by zero.
- Without an explicit period, the API uses `all_time`. Empty search results are
  empty lists rather than `null`.
- Concurrent-submission coalescing includes profile, type, and timestamp. Scrobble
  association with the shared catalog is serialized within the process to avoid
  duplicate entities when profiles submit the same song simultaneously.
- Track merges remove conflicts only within the same profile, preserving different
  people's listens at the same instant. Merging a track into itself is rejected.

### Import and export

The automatic input layout requires an explicit destination using the stable ID:

```text
<KOITO_CONFIG_DIR>/import/<profileID>/<filename>
<KOITO_CONFIG_DIR>/import_complete/<profileID>/<filename>
```

Example: `import/2/maloja_export.json`. IDs are available from `GET /profiles`.
Renaming the slug does not change the directory. Legacy files directly inside
`import/` have no implicit destination and must be moved into the appropriate
directory. `KOITO_SKIP_IMPORT` still controls automatic imports at startup.

To start an explicit import through the administrative API:

```http
POST /apis/web/v1/profiles/evertonberger/import
Content-Type: application/json
Cookie: koito_session=...

{"filename":"maloja_export.json"}
```

The file must exist in `import/<this profile's ID>/`. The backend recognizes
formats using the filenames already supported by the importers. In step 2, this
endpoint accepted local files only; step 4 added multipart upload to the same
route while retaining the JSON local-file contract. Filenames containing paths
and symlink files/profile directories are rejected. Importers also require an
explicit profile context when called directly; a username inside an imported
file does not select its recipient.

Export uses the selected profile rather than the administrator account. Names
are properly escaped in JSON. A composite timestamp/track pagination cursor
prevents skipping records in exports spanning multiple pages. Rankings and
exports drain and close result sets before querying artists or aliases,
avoiding deadlocks with a single SQLite connection.

### Relay and consolidated migration

Relay support was initially implemented in `004_profile_relays.sql`. After
consolidation, that content is part of `003_listening_profiles.sql`, in the same
transaction as profiles. The single migration adds:

- `profile_relays`: relay URL and token associated with the profile ID.
- `legacy_relay_binding`: records the legacy configuration association decision
  once, preventing reassignment after deletion or revocation.
- A trigger preventing an existing API key from being transferred to another profile.

Legacy global relay configuration is imported only when there is one profile
unambiguously linked to the administrator account. If ambiguous, the backend
records the decision without a recipient and requires explicit administrator
configuration. The relay forwards only events belonging to its associated
profile, after a valid submission has been processed. Renaming a slug does not
change the relay destination.

Explicit configuration, protected by an administrator session:

```http
PUT /apis/web/v1/profiles/evertonberger/relay
Content-Type: application/json
Cookie: koito_session=...

{"url":"https://server.example/apis/listenbrainz/1","token":"EXTERNAL_TOKEN"}
```

The URL is the base to which `/submit-listens` is appended. `DELETE` on the same
route removes the configuration. Persisted per-profile settings are independent
of the legacy bootstrap flag; removing a relay does not recreate it at restart.

### Validation and subsequent work

Implemented tests cover:

- Two profiles without their own accounts submitting the same song/timestamp concurrently.
- Two keys feeding one profile and queries without cross-profile leakage.
- Personal media metrics, search, summaries, empty profiles, and now playing for a new track.
- Rejection of ingestion keys for administration and mutations from external origins.
- Name/slug changes preserving ID, keys, history, and valid exports.
- Attempts to revoke another profile's key, valid revocation, and history preservation.
- Explicit import destinations, ID-based archiving, and relay isolation.
- Key-transfer prohibition, relay behavior without reassignment, and merges with simultaneous listens.

Validation performed: full Go suite, profile HTTP tests with the race detector,
and TypeScript checking. At the end of step 2, the selector and slug navigation
were still pending; step 3 implemented them. Profile creation/deletion and avatar
upload were assigned to step 4. Legacy accounts and columns remain only for
migration/bootstrap compatibility during the transition.

Historical additional check, before consolidation: an example database copy
migrated from version 3 to 4, preserving 72 listens, matching per-profile
statistics, accepting the legacy key, and passing `PRAGMA foreign_key_check`.
The original stayed at version 3 with the same 72 listens. The pagination test
exported all 1,001 expected records, including two at the same second across a
page boundary.

## Implementation record — step 3

### Navigation and presentation

- `/` always presents available profiles, including when only one exists. There
  is no automatic redirect based on the last profile, a cookie, or a session.
  With no profiles, an empty state still provides access to administration.
- `/:slug` uses a profile layout showing its name, slug, and avatar in the header.
  Existing home, history, chart, artist, album, track, and rewind pages are nested
  within it, preserving themes and components.
- The sidebar has a compact avatar/icon selector and name list. On mobile, the
  same control appears in the horizontal navigation bar. The list marks the
  current profile and links back to root selection.
- Avatars use a cached image when provided; missing or failed images fall back
  to an allowed icon or initials. Supported icons are `music`, `headphones`,
  `disc`, `radio`, and `user`. Upload and administrative selection were added in step 4.
- At this stage, `/admin` reused the existing administrator form and account
  controls without requiring a selected profile; creation/deletion was not yet included.
- Sidebar controls use accessible button/link names. The selector dialog has a
  name, keyboard navigation, Escape dismissal, and focus restoration.

Available page routes:

```text
/
/admin
/:slug
/:slug/listens
/:slug/rewind
/:slug/chart/top/artists
/:slug/chart/top/albums
/:slug/chart/top/tracks
/:slug/artist/:id
/:slug/album/:id
/:slug/track/:id
```

### Selection, links, and compatibility

- The URL determines the profile. Direct access, browser refreshes, and two tabs
  displaying different people keep their destinations independent.
- Card, table, related-artist, search-result, and media-page links retain the
  profile prefix. Period navigation, pagination, rewind, and return after merging
  also retain the slug.
- Switching profiles preserves history/chart/rewind screens and their periods
  where applicable. Artist/album/track ID filters and pagination page numbers
  are removed because they may not apply to another person.
- From a specific media page, switching goes to the new profile's home rather
  than automatically opening a track that person may never have heard.
- The home period is also represented in the query string. Local period/activity
  preferences use stable profile IDs and sections, surviving slug changes.
- Old links without slugs resolve automatically only when exactly one profile
  exists. With several, root selection preserves the destination in `next` and
  opens it after a choice. Accepted destinations are limited to known internal routes.
- Missing profiles, renamed old slugs, and media absent from a profile's history
  show an error with a route back to selection, without global-data fallback.

### API, cache, and lifecycle

- `AppProvider` maintains `adminSession` separately from `ProfileProvider`.
  `user` remains a compatibility alias for older administrative controls.
- `createProfileApi(slug)` creates profile-bound API functions. Callbacks capture
  their destination rather than reading `window.location` when deciding where to write.
- Personal queries/actions use `/apis/web/v1/profiles/{slug}/...`. Login, configuration,
  administrator-account operations, and shared-catalog mutations retain the
  appropriate global endpoints, protected by the backend.
- Loaders use `params.slug` and the React Router request's `AbortSignal`, so direct
  access and rapid navigation do not depend on global-context update order.
- Each profile has its own `QueryClient`. All personal query keys, including
  invalidations, start with `["profile", profile.id, ...]`.
- Switching unmounts the previous layout, closes its dialogs, aborts in-flight
  requests, cancels queries, and discards the cache. This lifecycle also works
  under development StrictMode; a new selection does not reuse old responses.
- Search and suggestions discard stale responses. Merge search runs only while
  its dialog is open; navigation cancellation cannot produce delayed results.
- First activity is queried per profile. Empty profiles do not produce a false
  initial date in 1970 or invalid chart year ranges.
- Removed a duplicate empty-history message and added an empty-chart state.
  Recent history can also show now playing without completed listens. Local
  `dev` versions are not compared as valid semantic versions.

### Checks performed

- `make check`: Go suite, type generation/TypeScript checking, and frontend tests.
- `make client.build`: production build and gzip assets.
- `make client.test` / `cd client && corepack yarn test`: Node 24 native-runner
  tests for URLs, mandatory destinations, screen/period switching, captured
  profile callbacks, and cancellation propagation.
- Manual Chrome validation on a disposable copy of the 72-listen example database
  and a second empty profile: root selection, isolated metrics, sidebar switching,
  ranking periods, direct access, and artist/track/album links.
- Two tabs retained different profiles. The selector/header were inspected at
  400 px; Escape closed the dialog and returned focus to its trigger.
- Legacy `/track/19` with several profiles prompted for a choice and then opened
  the prefixed destination. A missing profile showed an error without fallback.
- Deleting all profiles in the disposable copy confirmed the empty root state
  and continued availability of the sign-in form at `/admin`.

The original database was used only to create the validation copy. No test
profiles or records were added to the active database. Test ports were separate
from the project's normal defaults: API 4110 and Vite 5173. Vite accepts
`KOITO_DEV_API_TARGET` to target an explicit validation API without changing the
default development destination.

At the end of step 3, administrative creation/editing/deletion, photo/icon
management, key management, and import upload were pending. Step 4 implemented
those features; step 5 records the final validation.

## Implementation record — step 4

### Administration and creation

- `/admin` provides profile management after login. Visitors see authentication;
  administrative forms and tokens are not loaded without a session. The
  Administration entry, initially in the main sidebar, moved into Settings in step 5.
- The administrator account has a separate collapsible section. “Manage profile”
  selects a person or creates one. In the standalone page, `?profile=<ID>` retains
  the administrative selection by stable ID, including after slug changes.
- Creation accepts a name, a suggested/editable slug, icon, and optional photo.
  Suggestions remove accents and normalize separators. The server rejects invalid,
  reserved, or duplicate slugs, and empty or excessively long names.
- The profile and initial key are created in one transaction. Failure to save the
  key rolls back profile creation. No listener account/password is created.
- After creation, the UI selects the profile and offers its initial key and
  scrobbling endpoint for copying. Photo upload is a subsequent operation; if it
  fails, the UI explains that the profile/key already exist and allows retrying
  only the upload.
- Name/slug changes preserve ID, history, and tokens. The UI uses the returned
  slug; the previous URL returns 404 as designed. Pending operations disable forms
  and the destination selector to prevent accidental switching.
- Fixed the existing administrator-account form: name changes no longer send an
  empty password, password changes no longer send an empty name, confirmation
  must match, and passwords require at least eight characters. Loading state
  ends when the request completes.

### Avatar and images

- Choose initials or one of `music`, `headphones`, `disc`, `radio`, and `user`.
  Photo upload uses multipart with a 5 MiB request limit.
- The server checks content and processes the image through the existing WebP
  cache, assigning its own UUID. The form cannot assign an arbitrary image ID.
- Photos take precedence over icons. Saving without a file replaces the existing
  photo with the chosen icon or initials. “Use initials” removes both photo and icon.
- Replacement/removal and profile deletion remove an old image only if no other
  association uses it. Global image cleanup now checks
  `listening_profiles.avatar_image_id`, preventing avatar deletion at startup.
- Catalog images remain shared and are not changed when editing a person's avatar.

### Key management

- Each profile has an administrative key list: inspect, reveal/hide, copy, create
  with a label, edit labels, and revoke with confirmation.
- The UI shows `/apis/listenbrainz/1/submit-listens` and the base
  `/apis/listenbrainz/1` for players requesting a server URL. It explains
  `Authorization: Token ...`. The origin follows the instance's current address.
- Keys are visually hidden until revealed and remain available only through
  administrative responses. They do not appear in public metadata or URLs.
- Key transfer is unsupported. Revocation stops new submissions without deleting
  history; a new key can be created for the same profile.

### Deletion and data preservation

- “Review deletion” retrieves the current count and displays the name and affected
  listens. The exact display name must be typed to confirm.
- The server requires the name/count and checks both in the deletion statement
  itself. Changes since review return 409 and require another review.
- Deletion removes the profile, its keys/listens, relay/legacy bridge by cascade,
  an avatar without other references, and in-memory now-playing state. It does
  not remove the administrator, artists, albums, tracks, or other profiles' listens.
- Deleting the last profile is supported. The public list becomes empty, `/admin`
  stays available, and a new profile with an initial key can be created.
- Files already archived in `import_complete/<ID>/` remain on disk. They are not
  active listens and are not reassigned to new profiles.

### Explicit import destination

- “Import history” displays the name, slug, and ID chosen in “Manage profile”.
  It requires a file, format, and confirmation that the history belongs to that
  person. Neither the administrator account nor names inside the file choose
  the destination.
- Upload uses the existing administrative import route with `multipart/form-data`,
  `format` and `file` fields, and a 100 MiB request limit. Formats: `koito`,
  `maloja`, `spotify`, and `lastfm` in JSON; `listenbrainz` uses the existing
  importer for ZIP exports containing `listens/*.jsonl` entries.
- The server generates a unique UUID filename with a format-specific suffix.
  The original name determines neither paths nor parser and cannot overwrite
  existing files. The destination stays `import/<profileID>/`; symlink profile
  directories are rejected.
- Success archives the upload in `import_complete/<profileID>/`. Failure removes
  the pending file so it cannot be retried automatically at the next startup.
- Existing importers write progressively: errors/interruption may leave some
  listens saved. The UI explains this limit. Upload is synchronous, without a
  persistent queue or progress bar. Larger files can use the local-file workflow
  documented in step 2.
- Required Koito export content is validated before writes, preventing a panic
  when an uploaded listen has no artists.
- Export explicitly targets the managed profile and requires a session.

### Additional HTTP contract

Prefix: `/apis/web/v1`. Every route below requires an administrator session;
mutations remain protected by origin validation.

| Method and route | Result |
| --- | --- |
| `POST /profiles` | Creates profile and initial key transactionally; returns profile metadata |
| `GET /profiles/{slug}/management` | Metadata and `listen_count` for administrative review |
| `DELETE /profiles/{slug}` | Deletes with required JSON `display_name` and `listen_count` |
| `PUT /profiles/{slug}/avatar` | Replaces avatar using multipart `icon` and optional `image` |
| `DELETE /profiles/{slug}/avatar` | Removes photo/icon and returns the profile using initials |
| `POST /profiles/{slug}/import` | Adds multipart `format` + `file`; retains JSON `filename` |

The step 2 profile-editing and key routes were reused. At the original completion
of step 4, no additional migration was needed and the schema was version 4.
It was subsequently consolidated into version 3.

### Validation and continuation

- `make check`: full Go suite, type generation/checking, and four native frontend
  tests. Includes a regression test ensuring unchanged administrator credentials
  are not submitted as empty fields.
- `make client.build`: production compilation and gzip assets.
- New HTTP tests in `engine/profile_management_test.go`, also run with `-race`:
  authorization, initial-key creation, rollback on key failure, invalid/duplicate
  slugs/icons, outdated deletion counts, preservation of another profile/catalog,
  last-profile deletion, and creation afterward.
- Avatar tests cover actual PNG upload, WebP retrieval, invalid-content rejection,
  preservation during global cleanup, and removal after the last reference.
- Upload tests cover explicit destinations, archiving, path-like filenames,
  invalid formats, visitor/external-origin rejection, invalid Koito files without
  panic, and pending-file cleanup after failure.
- Chrome on a disposable example-database copy: creation with suggested slug/icon,
  available but hidden initial key, name/slug editing with a stable ID, key-label
  editing, and deletion review showing name/count with confirmation initially disabled.
  Existing components and themes were preserved.
- The active database's 72 listens, profiles, administrator credentials, and keys
  were compared with their prior state and preserved. Validation used ports
  14110/15173, stopped afterward; the normal API was left stopped.

At the end of step 4, complete validation was pending. Step 5 records execution
against the real API and compiled frontend, distinguishing protocol validation
from configuring an external player.

## Implementation record — step 5

### Requested navigation adjustment

- Removed the separate administration icon/button from the main sidebar in
  `client/app/components/sidebar/Sidebar.tsx`.
- Administration moved into the Settings menu. It initially linked to `/admin`;
  the later interface adjustment below replaced the link with a tab displaying
  content inside the dialog. Visitors can access the sign-in form, without gaining
  administrative permissions.
- `SettingsModal.tsx` retains styling, themes, and responsive behavior. The initial
  link was separate from the tab group, and the dialog gained the accessible name
  “Settings”. Neither sessions nor current-profile state were duplicated.
- The initial link behavior was checked in desktop Chrome and at 400 px, including
  closing Settings when opening the administrative page. The final embedded-tab
  behavior was checked separately in the later interface adjustment.

### Reproducible validation against the real application

Added `scripts/validate_multiuser.py`, using only the Python 3 standard library,
and `make multiuser.check`. Validation builds API/frontend, starts real processes
on temporary loopback ports, makes HTTP requests, and restarts the API to check
persistence. It does not use the database configured in `.env`: the script strips
inherited `KOITO_*` variables and provides isolated configuration with automatic
imports and external providers disabled.

```sh
# Build and validate fresh installation and a generated legacy v2 database.
make multiuser.check

# Optionally repeat with a copy of an existing database.
# The source is opened read-only; credentials are never printed.
python3 scripts/validate_multiuser.py --binary ./koito \
  --example-db /path/to/koito.db
```

Without `--work-dir`, temporary databases/logs are removed on completion.
`--work-dir /new/path` retains disposable copies for inspection, with a private
directory and logs using permissions 0600. The path must be new. All processes
started by the script are stopped, including when a check fails.

Scenarios executed during this step:

1. **Fresh installation:** database created by the real application bootstrap.
   Initial login/profile and additional profile creation work without listener login.
2. **v2 → v4 migration, before consolidation:** a legacy fixture with administrator,
   password, key, and listen. Initial startup creates a backup and preserves legacy
   data; the old key remains valid and the original password permits login.
   The current consolidated v2 → v3 journey was retested later.
3. **Example database:** a consistent SQLite copy of the local pre-profile backup,
   containing 72 listens. Legacy data hashes, including passwords, tokens, sessions,
   and catalog, are compared before/after migration. Only afterward does the copy
   receive temporary credentials to run the journey. The source stays read-only
   and unchanged.

Across all three scenarios, `PRAGMA integrity_check` returned `ok`,
`PRAGMA foreign_key_check` reported no violations, and no listens or keys lacked
a profile. Restart preserved history and the remaining active key.

### Acceptance criteria and evidence

| Criterion | Result and evidence |
| --- | --- |
| Same track/second for two profiles | Two concurrent HTTP submissions produce two records, one per profile |
| Two keys for one person | Initial and additional keys feed one history; revoking one leaves the other active |
| Payload cannot choose the recipient | Another profile's IDs in the payload cannot override the key's profile |
| History, rankings, and media metrics | Personal totals checked in history, track/artist/album rankings, and corresponding media pages |
| Activity and rewind | Bucket sums and `summary` totals checked separately per person |
| Now playing | Event appears only in the key's profile, does not become a saved scrobble, and does not survive restart |
| Import/export | Two exported listens imported explicitly into the second profile; an existing duplicate does not increase totals |
| Explicit profile | Legacy statistics route with several profiles returns 409; old renamed slugs and deleted profiles return 404 |
| Protected administration | Visitors/keys cannot manage, export, or import; cookie-authenticated mutations from an external origin return 403 |
| Public metadata | Public profile responses contain none of the test keys |
| Revocation and deletion | Revoked key returns 401; history remains; deleting the second profile leaves the first and original profiles unchanged |
| Migration and persistence | Legacy data preserved, backup created, and updated login/keys/history valid after API restart |
| Last profile and catalog | Administrative suite confirms last-profile deletion, administrator/catalog preservation, and subsequent creation |
| Avatar | HTTP suite confirms upload/WebP serving, retention during cleanup, and deletion after the last reference |
| Interface URL/cache | Independent tabs, refresh, and ranking switches with the period preserved, checked in Chrome |

Protocol checks use the application's real ListenBrainz endpoint with
`Authorization: Token ...` and scrobbling payloads, without changing an active
Navidrome or other music server. Relay tests use a local HTTP server covered by
the profile suite.

### Administrator name and password: resolved issue

- Added `TestAdminAccountChangesKeepListeningProfile`: changes only the
  administrator name, logs in with the new name and old password, changes only
  the password, rejects the old password, and logs in with the new one.
- Confirms that these changes leave the public profile's name/slug/ID and ingestion
  key unchanged. It then changes the `default` profile's name/slug and checks that
  administrator login remains independent.
- The real-API journey repeats these changes and verifies login after restarting
  the process in all three database scenarios. The existing frontend test confirms
  unchanged fields are omitted rather than submitting empty names/passwords.
- No further credential implementation changes were needed after step 4: the
  corrected behavior was reproduced and verified. Use “Profile details” to change
  public identity and “Administrator account” to change login/password. With the
  final embedded interface, account controls are in the Account tab.

### Checks performed

- `make check`: full Go suite, type generation/TypeScript checking, and four frontend tests.
- `make multiuser.check`: API/production frontend build with gzip and complete fresh/v2
  journeys, including direct SPA page responses and JavaScript asset availability.
- Additional `--example-db` journey using a pre-profile backup copy containing
  the same 72 example listens.
- `go test -race ./engine ./internal/db/sqlite -run 'Test(Profile|AdminAccount)'`
  with `KOITO_MIGRATION_TEST_DB` targeting the backup: passed, covering migration,
  isolation, authorization, creation/deletion, avatar, import, relay, merge, export
  pagination, and credential updates.
- Compiled frontend in Chrome: root selection, administrator login, independent
  profile tabs, sidebar ranking switches preserving `period=all_time` and year,
  refresh, and Settings → Administration on desktop and at 400 px. Each profile
  displayed its own data without mixing rankings.
- `git diff --check`: no whitespace issues.

### Preservation and limits

At the original step 5 validation, the active database remained at version 4 with
72 listens and one profile. Fingerprints of all data tables were compared
before/after, preserving passwords, keys, sessions, and catalog. Disposable
validation databases/processes were removed/stopped. The normal API on 4110 was
left stopped, and the existing Vite on 5173 was neither restarted nor stopped.

This completes local validation of the MVP acceptance criteria. Existing limits
remain: public profiles, a shared catalog, and synchronous imports with upload
limits and possible partial writes on failure. External player configuration
and remote image/service integrations were neither changed nor tested against
external accounts during this run.

The schema was version 4 at the original completion of step 5; the later
consolidation below changed the target to version 3. Legacy `user_id` columns,
the bridge, and legacy methods remain for technical compatibility. Operational
listening identity is `profile_id`. Physically removing that compatibility, as
proposed in the data model, still requires a separate migration and updates to
legacy consumers; it was not part of this validation step. It does not change
the functional acceptance results above.

## Interface adjustment — administration inside Settings

- Administration is now a `TabsTrigger`/`TabsContent`, like Appearance and Account.
  Its forms appear in the Settings content area. Selecting it keeps the dialog,
  public page, and selected time period in place.
- Extracted `AdministrationPanel.tsx`, shared by the embedded panel and direct
  `/admin` access. Profiles, avatars, keys, and imports use the same existing
  administrative components and endpoints.
- In the embedded panel, the managed profile is selected with local state,
  without adding/removing public URL parameters. Direct `/admin` access retains
  `?profile=<ID>` selection.
- Login/logout update the session through `AppProvider`, without reloading the
  page or closing Settings. The Account tab retains administrator login/password
  controls, separate from public profiles.
- Closing Settings after visiting Administration refreshes the public page's
  queries/loaders to reflect changes. The explicit link to visit a profile closes
  the panel and opens the selected page.
- Validated with TypeScript, the four frontend tests, and production build. Chrome
  on a disposable database confirmed the embedded panel, login without reload,
  selection preserving URL/period, switching to Account, and logout while Settings
  stays open. Presentation was also inspected at 400 px.
- The development version was checked on a separate port: internal tab content,
  dialog dismissal, and focus returning to Settings.
- No database or permission changes were needed for this adjustment.

## Upgrade test on an existing installation — `.sqlexample` v2 database

This is a historical record from before consolidation, when migrations 3 and 4
were separate. The current workflow uses only migration 3, as recorded in the
following section.

The test used `.sqlexample/koito.db`, confirmed as migration version 2 beforehand.
The source was opened read-only and copied with the SQLite backup API, including
WAL. A binary built from the then-current code was started against the copy,
simulating an installation upgrade. The test did not execute migration SQL
manually, reset the password, or generate replacement keys.

**Historical result: automatic v2 → v4 migration passed.**

| Data | Before | After |
| --- | ---: | ---: |
| Users | 1 | 1 |
| API keys | 1 | 1 |
| Sessions | 1 | 1 |
| Listens | 72 | 72 |
| Artists | 18 | 18 |
| Albums | 19 | 19 |
| Catalog tracks | 76 | 76 |

- Automatically applied `003_listening_profiles.sql` and the former
  `004_profile_relays.sql`. Created `/default` with the 72 existing listens and
  67 tracks actually heard. The difference from the 76 catalog tracks already
  existed and does not indicate migration data loss.
- Created an automatic pre-migration version 2 backup with permissions 0600 and
  data hashes matching the source. Legacy tables, including password hash, token,
  and session, were compared without revealing credentials in the report.
- The API accepted the old key for the migrated profile. Public history, statistics,
  rankings, activity, summary, and now-playing reads worked. The ingestion key
  neither authorized administration nor appeared in public profile metadata.
- `PRAGMA integrity_check`: `ok`; `PRAGMA foreign_key_check`: no violations.
  No listens/keys lacked a profile. Verified listen PK
  `(profile_id, track_id, listened_at)`.
- Second startup retained version 4 without another migration, backup, duplicate
  profile, or changed persisted-data hashes.
- Source `.sqlexample/koito.db` and its WAL had identical hashes before/after.
  The active installation was unchanged and the validation API was stopped.
- The password was not used for login: preservation was verified by comparing
  its stored hash. This test neither reset passwords nor wrote scrobbles.

Local artifacts retained for inspection, outside the published source files:

- Migrated database: `.local/upgrade-sqlexample-v2/koito.db`.
- Version 2 backup:
  `.local/upgrade-sqlexample-v2/koito.db.pre-profiles-2790378440.bak`.
- Reports: `.local/upgrade-sqlexample-v2/REPORT.md` and `report.json`.
- Startup/migration/restart log: `.local/upgrade-sqlexample-v2/server.log`.

Added `scripts/validate_upgrade.py` to repeat this type of test without modifying
the source. The work directory must be new. With the current code, it validates
the consolidated v2 → v3 migration rather than the historical v2 → v4 layout:

```sh
CGO_ENABLED=1 go build -o /tmp/koito-upgrade-sqlexample ./cmd/api
python3 scripts/validate_upgrade.py \
  --binary /tmp/koito-upgrade-sqlexample \
  --source .sqlexample/koito.db \
  --work-dir .local/upgrade-sqlexample-v2-retest
```

The source path is an example; supply an existing v2 database when reproducing
the test. To upgrade a real installation, configure the current application with
the old installation's `KOITO_CONFIG_DIR`, retaining its database, image cache,
and other files. Migration runs at API startup. This test verified that behavior
on a copy without replacing the active installation's database.

## Authorized consolidation — single v2 → v3 migration

At the user's request, after restoring the development database to version 2,
all multiuser changes were combined into one migration 3. This replaces the
experimental two-migration layout recorded above for this development cycle.
It does not renumber the history of an already migrated database.

### Changes made

- `003_listening_profiles.sql` now also contains `profile_relays`,
  `legacy_relay_binding`, and `trg_api_key_profile_immutable`, previously in
  `004_profile_relays.sql`. File 004 was removed.
- Profiles, legacy-data association, listen/key table rebuilds, indexes, relay,
  and key-transfer protection run in the same Goose transaction. Failure anywhere
  rolls back the entire v2 → v3 upgrade.
- Migrations 001 and 002 are unchanged. Fresh installations apply 001, 002, and
  003; version 2 databases apply only 003, automatically at API startup.
- The automatic pre-migration backup still preserves the v2 database. Protection
  against destructive downgrade remains in place.
- SQLite tests previously targeting version 4 now target version 3. Real-API checks
  compare the final version against the latest available migration, also detecting
  accidental use of an outdated test binary.
- Added a late-failure test: a conflict creating the relay table must roll back
  profile creation and listen/key rebuilds as well. After removing the conflict,
  retry must finish at version 3 with working relay configuration and protection
  against transferring a key.

Experimental databases already migrated with the earlier layout (the old v3 or
v4) must be restored from a v2 backup before using this consolidation, as was
done in development. Do not manually edit `goose_db_version` or merely relabel
an old v4 database as v3; doing so does not reapply the migration's content.
Old v2 → v4 reports remain as historical records, not evidence of the current version.

### Consolidation validation

- The full Go suite passed, including late-failure rollback followed by a
  successful retry. Migration, profile, relay, and credential tests also passed
  with the race detector (`-race`).
- The complete HTTP journey passed on a fresh installation, a generated v2
  fixture, and a copy of the v2 development database: creation, keys, concurrent
  submissions, isolation, queries, import/export, revocation/deletion, and restart.
  The final version verified in all three scenarios is 3.
- An upgrade test started the current binary on a copy of
  `.local/koito/koito.db`. The log lists only `003_listening_profiles.sql` and
  reports migration completion at version 3. It preserved the user, password hash,
  old key, session, 72 listens, 18 artists, 19 albums, and 76 catalog tracks.
- Confirmed the automatic v2 backup, no FK violations or data without a profile,
  and a second startup without data changes, another migration, or another backup.
- The report and migrated database were retained locally in
  `.local/upgrade-consolidated-v3/`. The actual development database remained at
  v2 with 72 listens, ready to apply migration 3 automatically at the next API startup.
- The local `./koito` executable was rebuilt with the consolidated migration.
  Validation APIs were stopped; the main API was not started.

The old `.sqlexample` directory was no longer available during this run.
Additional tests therefore used the restored v2 development database as their
source, always opened read-only. No database version records were manually
edited to simulate consolidation.


## Profile relay administration — November 1, 2026

Settings → Administration → Manage profile now includes ListenBrainz forwarding:
an enable checkbox, base URL, visible external user token, saved/unsaved state,
Save forwarding and Test connection. The interface uses the existing per-profile
PUT and DELETE operations. Saving disabled removes the stored URL and token;
reactivating later requires entering them again. Existing listens are preserved.

An administrative GET endpoint loads the saved URL/token with `no-store`.
Public profile responses do not expose relay credentials. A new administrative
POST `/relay/test` checks the draft URL's `/validate-token` using the Authorization
header, reports the external username on success, and never saves credentials or
submits a listen. It accepts only HTTP(S) base URLs without embedded credentials,
query or fragment, uses a 10-second timeout and bounded response, and refuses
redirects. External errors are summarized without returning tokens or raw responses.
The protocol follows [ListenBrainz token validation](https://listenbrainz.readthedocs.io/en/latest/users/api/core.html#get--1-validate-token).

No database migration is required. Global legacy environment variables retain
their one-time bootstrap behavior; disabling removes the persistent profile relay.
Persistent delivery queues and changes to relay retry behavior remain future work.

Validation: the full Go suite passed, including authorization, per-profile
read/removal, invalid credentials/responses, redirects and non-persisting tests.
Frontend type checking, all four frontend tests and the production build passed.
Tests used disposable databases and local mock servers without submitting data
to external services or changing the real relay configuration.

## TuneTrail versioning

The current version uses `0.x`, starting with TuneTrail 0.3. `make version.bump` increments the second number, including 0.9 to 0.10; 1.0 is reserved for a future decision. CI checks increments and permits the initial migration from SUM numbering. The original Koito engine base remains 0.3.2, independently of TuneTrail versioning. Existing SUM tags and release notes are historical records.

Historical SUM validation confirmed RC2 in compiled assets and exercised RC2 → RC3 in a temporary copy.


## PR documentation and warnings checklist

Updated the user-facing documentation site for profiles, migration, API keys,
imports and forwarding. Replaced the experimental Node API in frontend tests
with the existing TypeScript compiler, fixed a pre-existing malformed JSON test
tag, and added Go vet to `make check`. See [PR validation](../../../docs-dev/PR_VALIDATION.md) for
commands, results and existing dependency/tooling warnings.

Documentation build passed (9 pages), compared with base `a079fa6` (8 pages):
both emit the same existing Node DEP0190 warning. Application tests/build and
Dockerfile checks have no warnings. Both corresponding PR checklist items are
checked, with these limits recorded in the validation report.
