# Database alterations from schema v2 to v20

TuneTrail started from Koito's SQLite database. A Koito 0.3 database sits at schema version 2. The current release (0.62) is at version 20. This file describes every change in between: what each migration adds, how the tables connect, and what the fields mean. The migrations live in `db/migrations_sqlite/` and run with goose. A production database that is still at version 2 is upgraded in one pass, in order, and each migration runs in its own transaction.

Going back is not supported by the migrations. The `Down` section of every structural migration aborts on purpose, so the only way back is to restore the backup the upgrade makes first. Several migrations also copy the data they rebuild into temporary tables and assert that the counts and the contents match before they commit. If an assertion fails, the whole migration rolls back.

A section at the end describes schema 21, which exists only on the ISSUE-172 branch and has not been released.

## Where schema v2 starts

Version 2 is Koito's own layout. It has no concept of people, profiles or receipts.

- `artists`, `releases` and `tracks` hold the catalog. Each has a unique `musicbrainz_id` (version 2 added that constraint to all three) and one alias table: `artist_aliases`, `release_aliases`, `track_aliases`. An alias row has the name, a `source` and an `is_primary` flag. The primary alias is the display name, exposed through the views `artists_with_name`, `releases_with_title` and `tracks_with_title`.
- `artist_releases` and `artist_tracks` link artists to releases and tracks, each with `is_primary`.
- A track belongs to one release (`tracks.release_id`) and has a `duration` in seconds. Orphaned releases are deleted by a trigger.
- `users` has `username`, `role` (`admin` or `user`) and a password hash. `sessions` stores login sessions with an expiry. `api_keys` stores the keys that submit listens.
- `listens` has `track_id`, `listened_at` (Unix seconds), `user_id` and `client`. Its primary key is `(track_id, listened_at)`, so a listen is identified by the track and the second it happened.

Koito databases created before upstream pull request 257 (April 2026) reached version 2 without two indexes, `idx_artist_aliases_artist_id` and `idx_listens_track_id_listened_at`. Migration 004 creates them if they are missing, so those databases also upgrade cleanly.

## What each migration does

| Version | Name | Change |
| --- | --- | --- |
| 3 | listening profiles | Adds `listening_profiles`. Every existing user gets a profile, and history, API keys and the old relay move under it. |
| 4 | ingest infrastructure | Adds the tables for receipts, receipt details, revisions and the delivery queue. Nothing writes to them yet. |
| 5 | accounts and profiles | Ties each profile to one owner account. Adds account status and the last-administrator guards. |
| 6 | active admin guards | The last-administrator guard counts only administrators who can still sign in. |
| 7 | account tokens | Adds `account_tokens` for activation and password-reset tokens. |
| 8 | recording identity | Adds `recordings`, `occurrences` and `event_origins`. Rebuilds `listens` with an event ID and an origin. |
| 9 | ingest activation | Adds `ingest_events.mode` (shadow or active) and the per-profile switch table `profile_ingest_flags`. |
| 10 | occurrence ingest | Adds the occurrence resolver, `occurrence_credits` and the one-way conversion of a profile. |
| 11 | occurrence writer guards | Triggers that stop the old writer once a profile has been converted. |
| 12 | relay delivery | Adds the durable relay: destinations, allowlist, sent archive and queue counters. |
| 13 | ingest retry | Adds parking of receipts that keep failing. |
| 14 | occurrence archive | Adds tables for restoring exported history and for undoing recording merges. |
| 15 | archive export boundary | Adds a never-reused export sequence for listens. Undo records get an expiry. |
| 16 | ingest review | Adds `listens.review_distinct` and the indexes behind the review queue. |
| 17 | ingest source summary | Adds the latest-receipt summary per source and client, with two generated columns. |
| 18 | ingest reprocess | Adds the claim budget and the record of who reprocessed a receipt. |
| 19 | ingest inbox | Rebuilds `api_keys` with IDs that are never reused. Adds attribution of receipts to API keys and daily counts. |
| 20 | ingest discards | Adds `ingest_discards`, an audit trail for receipts removed by hand. |

## Profiles and accounts (versions 3 to 7)

### listening_profiles

A profile is the owner of a listening history. It has an `id`, a unique `slug` used in URLs, a `display_name`, an `avatar_image_id` or an `avatar_icon`, `created_at`, `user_id` and `require_login`.

- `slug` is lowercase letters, digits and hyphens, 1 to 64 characters, with no hyphen at either end. A check constraint refuses reserved route names such as `admin`, `apis`, `image` and `chart`.
- `user_id` (version 5) points to the owning account and is unique, so accounts and profiles are one to one. A trigger refuses a profile without an owner, and another refuses changing the owner afterwards.
- `require_login` (version 5) hides the listens from visitors who are not signed in. Only the owner and administrators can read them.
- `avatar_icon` holds one of five names (`music`, `headphones`, `disc`, `radio`, `user`). When it is empty and there is no image, the interface shows initials.

Version 3 created one profile per existing user. With a single user the slug is `default`, otherwise `user-<id>`. A temporary table, `legacy_user_profiles`, mapped users to profiles during the transition. Version 5 dropped it.

### Moving history and keys under profiles

Version 3 rebuilt two tables:

- `api_keys` gained `profile_id`. A key now submits listens to a profile, and a trigger forbids moving a key to a different profile. `user_id` stays as attribution only. `label` is a name the owner gives the key.
- `listens` gained `profile_id`, and its primary key became `(profile_id, track_id, listened_at)`. Two profiles can now record the same track in the same second.

The single relay of Koito moved into `profile_relays` (`url`, `token`, one row per profile). `legacy_relay_binding` is a one-row table that remembers which profile inherited the old relay.

### users, sessions and tokens

Version 5 added `users.status`, which is `active` or `pending`. A pending account has an empty password and cannot sign in until the owner redeems a token. Triggers refuse deleting or demoting the last administrator. Version 6 tightened that rule so a pending or disabled administrator does not count.

`account_tokens` (version 7) holds at most one token per user. It stores a SHA-256 hash of the secret (never the secret), a `purpose` (`activation` or `reset`) and `expires_at`. An activation token lasts 24 hours and a recovery token 30 minutes. Redeeming one sets the password and, for a new account, makes it active. A password change from the account page ends every session of that user.

## Identity: recordings, occurrences and events (version 8)

Version 2 treated a track as a song on one release. The same song on a single, an album and a compilation became three unrelated tracks. Version 8 separates three ideas:

- A recording is the song itself. `recordings` has an `id`, a unique `musicbrainz_id` and `mbid_source` (`legacy_declared` for an identifier a client sent, `validated` after MusicBrainz confirmed it).
- An occurrence is that song as it appears on a particular release, or on none. `occurrences` has `recording_id`, an optional `release_id`, `disc_number`, `track_number`, a unique `track_mbid` and `presented_title` (added in version 10, the title as the source printed it). `legacy_track_id` links an occurrence to the old `tracks` row it came from.
- An event is one listen. Version 8 rebuilt `listens` so each row has its own `event_id`, a random 128-bit hex string that is now the primary key.

For existing data the migration kept the numbers stable: every `tracks.id` became the `occurrences.id` of the same value, and `listens.track_id` now points to `occurrences(id)`. Recording IDs are allocated separately. After the rebuild the migration compares counts and per-row contents against the saved copy and aborts if anything differs.

`listens.origin` records which writer created the row, and it points to `event_origins`. The registered origins are `legacy`, `listenbrainz_api`, `listenbrainz_import`, `lastfm_import`, `spotify_import`, `maloja_import`, `native_import` and `archive_restore`. The origin is fixed when the row is inserted, and a trigger refuses later changes.

`occurrence_credits` (version 10) stores the artist credit as the source gave it: `occurrence_id`, `position`, the `name` as sent and an optional `artist_id` when the name was matched to exactly one artist. A listen with an ambiguous alias keeps the name and leaves `artist_id` empty.

`recording_moves` (version 14) remembers merges of recordings so they can be undone for 30 days. Its key is `(source_track, destination_track, occurrence_id)`. It stores the original and merged recording IDs, a `stage` and `expires_at` (added in version 15).

## Receipts: how a submission becomes a listen (versions 4, 9 to 20)

Every submission from the API or an import file first becomes a receipt in `ingest_events`. A receipt keeps the original payload and the decisions made about it, so an administrator can see why a listen exists or why it does not.

### The columns of ingest_events

The table has many columns. Grouped by purpose:

| Group | Columns | Meaning |
| --- | --- | --- |
| Identity of the receipt | `receipt_id`, `profile_id`, `source_instance_id`, `batch_id`, `item_ordinal`, `operation`, `received_at` | Which profile, which source (a row in `integrations`), which batch and position, and when it arrived. |
| Deduplication keys | `external_namespace`, `external_event_id`, `external_id_trusted`, `client_idempotency_key`, `batch_digest`, `file_digest`, `file_format`, `format_version` | Three unique partial indexes use these. A trusted external ID, a client idempotency key plus position, or a file digest plus position can each appear only once per profile and source. |
| Time | `listened_at`, `original_time`, `original_offset`, `time_precision`, `time_semantics`, `uncertainty_ms` | The time as the source stated it, how precise it is (second, millisecond, estimated and so on) and whether it marks the start or the end of the play. |
| Source description | `client_name`, `client_version`, `source_type`, `adapter_version`, `schema_version` | Who sent it and which adapter parsed it. |
| Payload | `original_snapshot`, `normalized_snapshot`, `normalizer_version`, `snapshot_digest`, `digest_version` | The submitted fields as JSON and the cleaned version. The original is immutable: a trigger rejects updates. |
| Decision | `state`, `identity_status`, `duplicate_decision`, `reason_codes`, `evidence`, `rule_version`, `resolution_version`, `resolved_at` | The outcome of the resolver and why. |
| Result | `event_id`, `occurrence_id`, `recording_id` | The listen, occurrence and recording this receipt resolved to. |
| Work queue | `attempt_count`, `next_attempt_at`, `lease_owner`, `lease_until`, `work_version`, `closed_at` | Used by the background resolver. A worker takes a lease, and an expired lease can be taken over. |
| Retry and parking | `parked_at`, `park_reason`, `retry_attempt_base`, `claim_attempt_base`, `reprocessed_at`, `reprocessed_by` | Versions 13 and 18. `attempt_count` is the lifetime count and the two base columns mark where the current budget started. A receipt that exhausts its budget is parked. |
| Mode | `mode`, `resolver_variant`, `delivery_policy` | `mode` is `shadow` or `active`, `resolver_variant` is `v9` or `occurrence`, and `delivery_policy` is `legacy` or `durable`. All three are fixed at admission. |
| Health views | `health_source`, `health_client`, `api_key_id`, `inbox_key`, `inbox_metadata` | Versions 17 and 19. See below. |

`state` takes one of seven values:

- `received`: waiting for the resolver.
- `pending_identity`: the resolver needs a decision about which artist, release or recording this is.
- `pending_duplicate`: the resolver found a likely duplicate and the review queue holds it for a decision.
- `retryable`: the resolver failed and will try again.
- `invalid`: rejected. A rule version and at least one reason code are required.
- `accepted`: a listen exists and `event_id` points to it.
- `duplicate`: resolved as a duplicate of an existing listen, and `event_id` points to that listen.

A check constraint ties the last two together: `event_id` is set exactly when the state is `accepted` or `duplicate`. The foreign key from `(profile_id, event_id)` to `listens` cascades on delete.

### Tables around a receipt

- `integrations` has one row per source of a profile: an `id`, `source_type`, `display_name`, a `token_ref`, `revoked_at`, `details_retention_days` (7 to 3650, default 90) and `diagnostic_enabled`. Receipts and queue items reference `(profile_id, source_instance_id)` here.
- `ingest_event_details` holds large per-receipt diagnostic payloads. They follow the retention period of the integration (`details_retention_days`).
- `ingest_observations` stores further observations attached to a receipt. Each has its own original snapshot and digest, and a trigger makes it immutable.
- `ingest_revisions` is the history of decisions: one row per `(profile_id, receipt_id, resolution_version)` with the author, the decision, the reason codes, the evidence and the rule version. It records, for example, a catalog merge that changed an accepted receipt.
- `ingest_source_summary` (version 17) keeps the latest receipt for each `(profile, source, client)`, so the health page does not scan all receipts. Triggers keep it current.
- `ingest_inbox_daily` (version 19) holds daily counts per `(profile, day, key, client, state, metadata)`. Complete UTC days are read from here, and individual receipts are read only at the edges of a window.
- `ingest_discards` (version 20) is the audit for receipts removed by hand: `profile_id`, `receipt_id`, `actor_id`, `discarded_at`, the state it was in, the park reason and the reason codes. It outlives the receipt, which cascades with its payload.

### The generated columns

`health_source` is the file format or, for API submissions, the source type. `health_client` is the client name, truncated to 120 characters, read from `client_name` or from the snapshot. Both are virtual, so they cost no storage.

`api_key_id` (version 19) records which API key made the submission. It has no foreign key on purpose, so deleting a key does not erase the attribution. `inbox_key` is generated from it: `-1` for file imports, the key ID for API submissions, and `0` when the key is unknown.

`inbox_metadata` classifies how complete the submitted data was:

- `0` means complete.
- `1` means missing: the title or the artist is absent or empty.
- `2` means incomplete: the title and artist exist, but the release, a MusicBrainz ID for the recording, release or artist, or the duration is missing.

An index over these columns lets the Inbox page count receipts by key, client and completeness without reading payloads.

## Listens (versions 8, 14, 15, 16)

The final shape of `listens`:

| Field | Meaning |
| --- | --- |
| `event_id` | Primary key, a random hex string. |
| `profile_id` | The profile that owns the listen. Cascade delete. |
| `track_id` | The occurrence. The name is historical. |
| `listened_at` | Unix seconds. |
| `user_id` | Attribution kept from before profiles. Not a foreign key. |
| `client` | The client name as submitted, for example `Navidrome`. |
| `origin` | Which writer created the row. Fixed at insert. |
| `review_distinct` | Version 16. Set to 1 when someone reviewing a suspected duplicate chose to keep both listens. |

`idx_listens_legacy_identity` (profile, track, time) is a unique index with a condition: it applies only to rows whose origin is `legacy` and whose `review_distinct` is 0. Listens written by the older writer cannot collide in the same second, and an explicitly kept duplicate is exempt. Listens from the newer writers are identified by `event_id` and by their receipts instead.

`listen_export_order` (version 15) is a sequence table with one row per listen. Exports page through it in order. Sequence numbers are never reused, even if the newest listen is deleted, so an interrupted export can resume without skipping or repeating rows.

`archive_identity`, `archive_catalog` and `archive_events` (version 14) support restoring an exported TuneTrail history. `archive_identity` is a single row with a random namespace for this database. `archive_catalog` maps artists, releases and tracks of a file to local rows (`trusted` marks a mapping safe to reuse), and `archive_events` maps each restored listen to its source event ID, so restoring the same file twice does not add listens twice. Triggers remove stale mappings when an artist or release row is deleted, because SQLite can reuse its ID.

## Relay: sending listens elsewhere (versions 3 and 12)

The relay forwards accepted listens to another ListenBrainz-compatible server.

- `profile_relays` holds the destination: `url` and `token`, one row per profile.
- `relay_destinations` holds the state of that destination. `generation` and `config_version` change when the URL or token changes, so queued work for an old configuration is not sent to the new one. `open_count` counts queued items (a check constraint caps it at 100,000). `pause_class`, `paused_until` and `backoff_count` describe a pause after errors or rate limits. `lease_owner`, `lease_until` and `work_version` coordinate the worker.
- `relay_allowlist` holds exceptions to the network policy: `host_port`, an optional `cidr`, and flags `allow_http` and `allow_private`. `origin` is `manual` or `seeded`.
- `outbox` is the delivery queue, one row per `(profile, destination, event, operation)`. `state` is `pending`, `retryable`, `failed`, `sent` or `discarded`. `delivery_key` makes a delivery idempotent, and `relay_generation` ties the row to the destination generation. `outbox_details` stores the payload and is purged on a schedule.
- `relay_sent_archive` keeps one daily count of delivered items per profile after the sent rows are deleted. A trigger moves the count into it before the delete.
- `relay_delivery_gate` is a one-row switch with the policy `legacy` or `durable`. It can only move from legacy to durable.

## Guard triggers worth knowing

The database enforces many rules itself, so application code cannot skip them:

- A profile always has an owner, and the owner cannot change. A key cannot move to another profile.
- The last active administrator cannot be deleted or demoted.
- A receipt's original snapshot, its resolver variant and a listen's origin cannot be edited after insert.
- A listen written by the old writer is refused for a profile converted to occurrence writing (version 11). This is the one-way cut. A trigger on the conversion flag refuses turning it back off.
- Moving an occurrence to a different recording adds a revision row to every affected receipt in the same transaction.
- A recording whose MusicBrainz ID was validated cannot be changed through the old writers.

## How the tables connect

```
users 1 ── 1 listening_profiles 1 ── * api_keys
   │                 │
   │                 ├── 1 profile_relays 1 ── 1 relay_destinations ── * outbox ── * outbox_details
   │                 │                         relay_allowlist (per profile)
   │                 ├── * integrations ── * ingest_events ── * ingest_revisions
   │                 │                          │    │            ingest_event_details
   │                 │                          │    │            ingest_observations
   │                 │                          │    └──> occurrences, recordings
   │                 │                          └──> listens (event_id)
   │                 ├── * listens ──> occurrences ──> recordings
   │                 │                      │              (musicbrainz_id)
   │                 │                      ├──> releases ── * tracks (legacy)
   │                 │                      └── * occurrence_credits ──> artists
   │                 ├── ingest_discards, ingest_inbox_daily, ingest_source_summary
   │                 └── archive_catalog, archive_events, listen_export_order
   └── 1 account_tokens, * sessions

artists ── * artist_aliases        releases ── * release_aliases        tracks ── * track_aliases
artists ── * artist_releases ── releases           artists ── * artist_tracks ── tracks
```

In words: a user owns one profile. The profile owns API keys, an optional relay, sources (`integrations`), receipts and listens. A receipt points to the listen, occurrence and recording it resolved to. A listen points to an occurrence, an occurrence to a recording and a release, and the credits of an occurrence point to artists. The catalog tables from version 2 (artists, releases, tracks and their aliases) are shared by all profiles.

## From submission to a row in listens

1. A client posts a listen to the ListenBrainz endpoint with an API key, or a file is imported. The key identifies the profile, and `api_key_id` is stored on the receipt.
2. The server writes a receipt with state `received` and answers the client. Nothing else has to finish first.
3. The resolver takes a lease on the receipt, matches the artist, release and recording, and decides whether the listen is new. It writes its reasons into `reason_codes` and `evidence`, and each decision adds a row to `ingest_revisions`.
4. If the match is clear, the receipt becomes `accepted`, a row appears in `listens` with the origin of the source, and the receipt gets its `event_id`. If the listen repeats an existing one, the state is `duplicate`. If the case is unclear, the receipt waits as `pending_identity` or `pending_duplicate` for a person to decide. If the resolver keeps failing, the receipt is parked after the retry budget.
5. If the profile has a relay, an `outbox` row is created for the new listen. The worker delivers it and marks it `sent`.

## Schema 21 (not released)

Schema 21 is part of ISSUE-172, which removes the old writer. It is on the `chore/issue-172-remover-o-escritor-legado-e` branch and is not in release 0.62. It changes these things:

- Drops `profile_ingest_flags`, the triggers that read it and `idx_listens_legacy_identity`. Occurrence ingestion becomes the only path for every profile.
- Adds `profile_ingest_review (profile_id, review)`. A profile without a row has duplicate review on, so new profiles start with it on.
- Registers a `manual` origin for listens added by hand.
- Adds `relay_destinations.awaiting_approval`. Saving a relay now creates a draft that sends nothing and queues nothing until a root administrator approves the network destination. Listens received before approval are never sent afterwards.
- Adds `ingest_events.relay_admission_generation`, which records which destination generation a receipt was admitted under.
- Refuses to run while there are shadow receipts without a linked listen or unresolved receipts of the old resolver. It tells the operator to resolve them with the previous release and to take a backup first. A database coming from version 2 has none, so it is not affected.
- Allows no downgrade. The way back is the backup taken before the upgrade.

## Schema 22 (not released)

Schema 22 is part of ISSUE-202 (avatar background color). It is on the `feat/issue-202-cor-de-fundo-do-avatar` branch. It changes one thing:

- Adds `listening_profiles.avatar_color`, a nullable text column. `NULL` keeps the theme colour. A `CHECK` accepts only a lower-case `#rrggbb` value; the application accepts only the twelve colours of the fixed palette in `internal/models/avatar_color.go`, so the column never holds a free-form value. Existing profiles get `NULL`, and their avatar photo and icon are untouched.
- The upgrade takes the usual backup first (`tunetrail.db.pre-ingest-*.bak`, at schema 21).
- Allows no downgrade. The way back is the backup taken before the upgrade.
