---
title: Multiuser operations
description: Manage accounts and listen visibility, upgrade a Koito database, and restore a backup.
---

## Accounts and permissions

Each account owns one listening profile. The sign-in username is independent
of the profile's public name and URL.

| Who | Read listening history | Manage data |
| --- | --- | --- |
| Visitor | Public profiles | |
| Signed-in user | Public profiles and their own protected profile | Their own account, profile, listens, API keys, imports, exports and relay |
| Administrator (root) | All profiles | All profiles, accounts, roles, server settings and the shared music catalog |

Users sign in at `/login`. **My account** opens the tabs for their own account
and profile. Administrators use **Administration → Accounts and profiles** to
create a profile with a pending account and an initial player key, or to manage
an existing account. An active administrator can grant or remove administrator
access. The last active administrator must remain. Deleting an account also
removes its profile, keys, history and relay configuration. Review the impact
shown before confirming.

## Activate an account or recover access

A pending account can receive listens through its player key and appears in
the directory. It becomes active after its owner sets a password.

1. In **Administration → Accounts and profiles**, open the person's account
   and issue an activation token for a pending account or a recovery token for
   an active account. Activation tokens expire after 24 hours and recovery
   tokens after 30 minutes. Issuing another token replaces the previous one.
2. Give the recipient the sign-in username and token privately. The token is
   displayed once. The recipient opens `/admin/redeem`, enters the username and
   token, and sets a password. Use the form to enter the token.
3. Sign in with the new password. Redemption consumes the token and revokes
   the account's old sessions. A user who can still sign in can change their
   password from **My account**. Activation, recovery and password changes
   use the existing limits of at least eight characters and at most 72 bytes.

An administrator can issue a token for another administrator. If the instance
has a single administrator and that person loses access, stop the service and run `tunetrail reset-password`
locally using the existing data directory and configuration. The command
generates a password, revokes that account's sessions and prints the password
to the local process log. Protect that output, then restart the service and
sign in.

## Protect listening history

Existing and newly created profiles start public. In **My account → Profile**,
enable **Require login to view listens** to limit local listening history and
its derived statistics, search results and artwork to the owner and
administrators. The directory still shows the public profile name, URL, avatar
and protection indicator. Protected reads use a private, non-storing cache
policy. Copies of artwork served while public may remain in browsers or
intermediary caches for up to 30 days.

Player keys continue to accept submissions. A configured outgoing relay also
continues to send eligible listens. Check privacy at the external destination
separately, or remove the relay from the profile to stop forwarding. Changing
local visibility or deleting a local listen leaves previous deliveries at the
destination.

## Upgrade Koito SQLite schema 2 to TuneTrail 0.72

1. Schedule downtime and stop all processes that can write to the data
   directory. Copy the entire directory, including configuration, images and
   any SQLite recovery files. Keep that copy private and retain the old binary
   or image. Allow disk space for the old database, the backup, the new database
   and SQLite working files.
2. Rehearse the upgrade with a second copy in its own data directory. Use
   TuneTrail 0.72 (schema 22), or the release you intend to deploy. For a
   standalone binary, point `TUNETRAIL_CONFIG_DIR` at the copy. For Compose,
   point `KOITO_DATA_DIR` in your deployment environment at the copy and select
   the image tag with `KOITO_IMAGE_TAG`. Use a separate port and isolate the
   rehearsal from external relay destinations.
3. On first startup TuneTrail makes a consistent
   `koito.db.pre-tunetrail-*.bak` snapshot, including committed WAL data, and
   migrates the copy it publishes as `tunetrail.db`. It retains the original
   `koito.db`. Existing credentials, listens and player keys carry over. A
   single legacy user's profile is `/default`. Multiple legacy users receive
   separate profiles. Profiles without an existing account get a pending
   account and can be activated by an administrator.
4. Confirm that the server starts, sign in again, check listening counts and
   player key assignments, and review profile visibility and relay settings
   before reopening access. Repeat the same upgrade using the production
   directory. Keep the original backup until the upgrade is accepted.

## Restore or roll back

Stop TuneTrail before switching databases. Preserve the current directory as
a separate copy first, especially if it contains writes made since the backup.
Those writes are outside the restored snapshot.

To restore a complete backup, restore the directory to a new location and
point the matching application version at it. To restore an automatic SQLite
backup, copy it into a clean data directory as `koito.db` for schema 2 or as
`tunetrail.db` for a canonical TuneTrail backup. Restore the matching
configuration and images from the full directory backup. Use only recovery
files that belong to that snapshot. Keep the backup and its copies readable
only by the operator or service account.

Start the version appropriate to the snapshot to roll back, or the target
TuneTrail version to repeat the migration. Check startup, history totals,
account access and key assignments before switching traffic. Keep in mind that
a restored relay queue reflects the snapshot time and external deliveries
already made still exist.

See the [installation guide](/guides/installation) for deployment settings and
the [listening history ingestion guide](/guides/active-ingestion) for relay
delivery and queue behavior.
