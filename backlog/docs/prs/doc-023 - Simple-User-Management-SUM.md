---
id: doc-023
title: Simple User Management (SUM)
type: other
created_date: '2026-11-12T23:40:00Z'
updated_date: '2026-11-12T23:40:00Z'
tags:
  - pr
---
> Importado de `docs-dev/PR_MULTIUSER.md`. Original removido em 12/11/2026; consulte o histórico do Git.

# Simple User Management (SUM)

## Description

This is a proof of concept, is a full working code and running on my machine to all my family members, running with one instance of koito.

Allow several people to scrobble to one Koito instance while keeping separate listening histories and the existing interface. Listeners select a public profile without signing in; one password-protected administrator manages profiles and credentials.

- Resolve each incoming ListenBrainz API key to exactly one profile. Scope history, statistics, charts, search, now-playing, imports and exports to that profile while sharing catalog metadata.
- Add `/` profile selection, `/<slug>` profile routes and a sidebar selector with names and avatars.
- Add profile, avatar and API-key management inside **Settings → Administration**. Player API keys cannot authorize administrative changes.
- Add per-profile ListenBrainz forwarding with a visible external token, saved-state feedback and connection testing without saving credentials or submitting listens.
- Migrate SQLite schema 2 → 3 transactionally, preserving existing credentials and history. A single existing user's profile becomes `/default`.

**Upgrade:** back up the database first. Migration rollback requires restoring the pre-migration backup. Legacy unscoped reads return `409` when profile selection is ambiguous. Profiles and listening data are public to anyone with access to the instance; private profiles are outside this implementation.

## Related Issue(s)

None linked.

## Type of Change

- [ ] Bug fix
- [x] New feature
- [x] Breaking change — profile-scoped routes, administrative authorization and schema migration
- [x] Documentation update

## How Has This Been Tested?

- [x] Unit Tests — full Go suite, migration/isolation/authorization tests, relay tests using local mock servers, frontend type checking and four frontend tests passed. Production frontend build passed.
- [x] Manual Testing — local validation recorded in the design documents covers fresh installation, upgrading a copy of a schema-2 database, restart persistence, two independent browser tabs, profile switching, imports/exports and key revocation.

Migration and HTTP validation used disposable databases or copies. External relay delivery was simulated locally. The focused PR branch passed `make check`, the production frontend build, fresh/schema-2 HTTP journeys and migration/profile/admin tests with `-race`. Fork-specific GHCR publishing, deployment and SUM versioning changes are excluded.

https://github.com/ebrgr/TuneTrail/blob/main/docs-dev/MULTIUSER_DESIGN.en.md


## Checklist

- [X] I have performed a self-review of my own code
- [X] I have made corresponding changes to the documentation
- [x] I have ensured my changes generate no new warnings

## AI/LLM Usage

- [x] I disclose that approximately 70% of the implementation work used OpenAI Codex assistance; this is an estimate.
- [X] I fully understand all changes made by LLM-generated code.
- [ ] I have not and will not use an LLM to write any part of this PR description or PR comments.

This description was prepared with OpenAI Codex assistance; the no-LLM-description declaration is intentionally unchecked. Maintainer agreement on this upstream requirement is needed before proceeding beyond this draft.

## Screenshots (if applicable)

All screenshots use a fresh disposable database with fictitious profiles, music, artwork and demonstration-only tokens. No personal data or existing browser profile was used.

**Public profile selection**

![Public profile selection](https://raw.githubusercontent.com/ebrgr/TuneTrail/c37e37c9560aef3af33a3a0f4971930ffb67f88f/docs/screenshots/multiuser/01-profile-selection.png?download=1)

**Profile dashboard**

![Profile dashboard](https://raw.githubusercontent.com/ebrgr/TuneTrail/c37e37c9560aef3af33a3a0f4971930ffb67f88f/docs/screenshots/multiuser/02-profile-dashboard.png?download=1)

**Sidebar profile selector**

![Sidebar profile selector](https://raw.githubusercontent.com/ebrgr/TuneTrail/c37e37c9560aef3af33a3a0f4971930ffb67f88f/docs/screenshots/multiuser/03-profile-selector.png?download=1)

**Administration inside Settings**

![Administration inside Settings](https://raw.githubusercontent.com/ebrgr/TuneTrail/c37e37c9560aef3af33a3a0f4971930ffb67f88f/docs/screenshots/multiuser/04-profile-administration.png?download=1)

**Per-profile ListenBrainz forwarding**

![Per-profile ListenBrainz forwarding](https://raw.githubusercontent.com/ebrgr/TuneTrail/c37e37c9560aef3af33a3a0f4971930ffb67f88f/docs/screenshots/multiuser/05-profile-forwarding.png?download=1)
