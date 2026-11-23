---
title: Listening profiles
description: Share one Koito instance using public profiles and profile-specific API keys.
---

## Browse without an account

The root `/` lists profiles. Open `/<slug>` to view a person's history, charts,
statistics and currently playing track, or use the sidebar profile selector.
The URL determines the profile, so two browser tabs can show different people.
Switching profiles keeps the screen and period where possible and drops filters
that refer to the previous person's media.

Profiles and listening data are public to anyone who can reach the instance.
Selecting a profile does not grant permission to change data. Private profiles
and separate listener passwords are not provided.

## Manage profiles

Every signed-in user manages their own account and profile in one place,
**My account** (`/account`), which has five tabs: **Account** (username,
password, session), **Profile** (name, URL name, listen protection, avatar),
**API keys**, **History** (import and export) and **Relay**. The address never
carries an ID: the target is always the signed-in user's own profile. The old
`/admin?profile=<id>` address of your own profile redirects to `/account/profile`.

Administrators also have **Administration** (`/admin?section=profiles`), a list of
accounts and profiles with the form to create a new one. Opening another account
there shows the same five tabs under a banner naming that person and
saying you are operating as administrator; its Account tab holds the root tools
(recovery token, administrator role, deletion review). Forms bind their
destination to the profile ID and slug, independently of other open tabs.

Create or edit a display name, URL name and optional photo or fallback icon.
Renaming a URL keeps the profile's ID, history, keys and forwarding configuration;
the previous URL returns 404. Administrator login credentials are independent of
the profile name. Photos accept JPEG, PNG and static WebP; animated images and
other formats are not supported. The complete upload, including multipart data,
is limited to 5 MiB.

Each profile starts with an API key. Create additional labelled keys for players,
copy them into the appropriate player's settings, or revoke them to stop future
submissions. Keys cannot be transferred between people. A player's key selects
its profile; it does not authorize profile management or other administration.
See [Setting up the Scrobbler](/guides/scrobbler).

Administrators can import/export a selected profile's history and configure its
ListenBrainz forwarding.

The **Health** page of a profile shows recent submissions, relay failures and
deduplication decisions. See [Listening history ingestion](/guides/active-ingestion#ingestion-health).

Deleting a profile requires confirmation of its name
and listen count and removes that profile's listens, keys and relay. Other
profiles and shared catalog entries remain intact. Deleting a profile does not
delete the administrator account.

## Upgrade an existing SQLite installation

1. Stop the existing Koito process and back up its complete configuration/data
   directory, including the SQLite database. Keep the backup for rollback.
2. Install a build containing listening-profile support and reuse that directory.
3. Start Koito. Migration `003_listening_profiles.sql` upgrades schema 2 to 3 in
   one transaction. Do not edit the schema version manually.
4. Sign in to Administration and verify existing history and keys before adding
   more people. A single existing user's history becomes `/default`; multiple
   existing users receive `user-<id>` slugs. Existing credentials and keys are
   preserved.

Existing installations must already use SQLite. The schema migration cannot be
downgraded in place: stop Koito, restore the pre-upgrade directory backup and run
the previous version to roll back.

## Track rankings

Open **Top Tracks** in a profile to see plays grouped **By recording**. Select
**By release** to see how many times the profile played each recording on each
release. Plays without a known release appear under **Unknown release**.
With the same filters, the two views add up to the same play total.

The release view can be bookmarked with `?group_by=release`. Tracks with
unconfirmed recording identities stay separate, even if their titles match.
A recording is new to a profile in the period when that profile first heard
any of its occurrences.

## API compatibility

Players continue to submit to `/apis/listenbrainz/1/submit-listens` with
`Authorization: Token YOUR_KEY`. The server resolves the destination from the
key rather than from a profile supplied in the payload.

Web reads use `/apis/web/v1/profiles/<slug>/...`. Legacy unscoped read routes
resolve only when exactly one profile exists; otherwise they return `409` asking
for explicit profile selection. Administrative actions require an administrator
session, not a player API key.

In the Go interface, profile selection and personal search also work without
JavaScript. Appearance settings require JavaScript.

## Go administration forms

Forms work without JavaScript. Copy buttons and image drag-and-drop are optional
enhancements; a file input and manual key selection remain available. A submitted
form cannot be reused: refresh the page after a conflict or after restarting the
server. Sessions expire after 24 hours or 30 days with Remember me. Changing the
password revokes all sessions; changing the username does not rename a profile.

The HTML and JSON login endpoints share the existing global limit of ten attempts
per minute. Administrative pages and downloads are marked `no-store`.
