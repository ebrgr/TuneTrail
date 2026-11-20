---
title: Active ingestion pilot
description: Enable and pause active ingestion for a listening profile.
---

Active ingestion is off by default. It takes effect when the server starts with
`TUNETRAIL_INGEST_ACTIVE=true` and an administrator enables it for the specific
listening profile. It covers ListenBrainz submissions and history imports.
Manual additions from the administration panel still use the existing writer.

For new active submissions, HTTP 200 means TuneTrail has saved the submission
for processing. The listen may take time to appear in the history. For a 503
response, respect `Retry-After` before resending. The import summary shows
**Pending** for submissions waiting on processing and **Duplicates** for
completed submissions the importer recognized. The review option is locked
until there is an operator queue for resolving ambiguous submissions. Until
then, a repeat of the same track in the same second follows the existing
one-listen rule.

## Prepare a pilot

Keep the master switch off during the agreed shadow measurement period. In
shadow mode, TuneTrail records information about submissions while the existing
writer continues to update the history. Keep a verified backup of the data
directory, including its SQLite files. When upgrading from SQLite schema 8,
TuneTrail creates a `.pre-ingest-activation-*.bak` backup beside the database.
The old build requires a restored backup to roll back from schema 9.

Plan a maintenance window. Pause ListenBrainz writers and wait for import jobs
for the profile to finish. Give the database writer time to finish any remaining
work. Choose one profile for the pilot.

## Enable the profile

Use a signed-in root administrator's session for both requests. Send the
requests to your own TuneTrail instance, using the same origin as the
administrator session. Replace `<slug>` with the profile's URL name.

1. Send `GET /apis/web/v1/profiles/<slug>/ingest-activation`. Save the response
   as a record of the shadow measurements. It contains aggregate counts and a
   confirmation digest.
2. Send `PUT` to the same path with the digest from that response.

   ```json
   {"active": true, "review": false, "digest": "<preview digest>"}
   ```

   A 204 response confirms the profile switch and removal of its unlinked
   shadow receipts. The operation preserves the listening history and receipts
   linked to events. If you receive 409, get a new preview before confirming.
3. Ensure the server is running with `TUNETRAIL_INGEST_ACTIVE=true`. Changing
   the environment variable requires a restart. Check the preview again and
   confirm that `active` and `master_active` are both true before resuming
   clients. Verify that new test submissions appear in the history.

## Pause or resume

To pause the pilot for one profile, send `PUT` to the same path with
`{"active": false, "review": false}`. TuneTrail keeps processing submissions
already admitted by the active writer.

Before resuming, drain the shadow writers, start a new preview and confirm
activation again. Turn off the profile switch before using the master switch
as a longer-term pause. If the master switch alone is turned off, the profile
flag remains on and shadow receipts can accumulate. The new preview and purge
give you a clean starting point for measuring the resumed pilot.

If a client retries a submission whose key still belongs to an unlinked shadow
receipt, active ingestion takes over that submission. TuneTrail keeps the
existing listen if it matches the same track and second, or saves the listen if
it is missing. Shadow receipts already linked to a listen remain available for
replay.
