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
completed submissions the importer recognized. Duplicate review is off by
default. With review off, a repeat of the same track or occurrence in the same
second links to the existing listen. The occurrence resolver keeps possible
matches up to two seconds apart as separate listens.

## Review possible duplicates

In **Health → Pending reviews**, the profile owner or root can turn on
duplicate review for an active profile and open **View evidence** on a receipt.
Choose an existing listen and select **Accept as duplicate** to link the
receipt to it. **Keep as a separate listen** adds a listen at the submitted
time and queues relay delivery when eligible. Pending receipts do not count
toward listening statistics. Each decision records the signed-in account's ID
in the revision history.

At the top of **Health**, the counter shows the number of pending receipts for
the selected profile. **Accept all as duplicates** processes only receipts with
one recorded candidate that still exists. **Keep all as separate listens**
processes only receipts with a resolved occurrence and a valid submitted time.
Each action processes up to 25 eligible receipts in a single transaction. If
the batch changes while the page is open, refresh before submitting again.

Turning review off changes how new decisions are made. Receipts already in
the queue remain available. A submission conflict without a resolved track or
an existing candidate remains visible without decision buttons.

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


## Occurrence-based ingestion

The occurrence resolver preserves the album and track position supplied by the
source when there is enough evidence to identify the occurrence. It supports
ListenBrainz submissions and ZIP archives, Last.fm, Spotify, Maloja, and
TuneTrail/Koito imports. Existing active profiles continue to use the
compatibility resolver.

The occurrence resolver is opt-in and currently restricted to disposable test
databases. Profiles with a configured ListenBrainz relay cannot convert.
Conversion requires clearing unlinked shadow receipts and draining compatible
active receipts. After conversion, disabling the master switch pauses new
admissions while already accepted batches continue resolving. Identity conflicts
remain pending, and turning on duplicate review can also hold possible duplicate
listens. Pending items are excluded from listening counts.

Upgrading a v9 database to v11 saves a `.pre-occurrence-ingest-*.bak` backup.
TuneTrail 0.46 and earlier cannot open a v11 database.

## Ingestion health

**Health** shows what ingestion has done for one profile. Open it from the
**+** menu: root finds it under **Administration**, above **Server
configuration**, and profile owners find it with their account links. The page
is also at `/admin/health?profile=<id>`. Only the profile's owner and root can
open it, always signed in, even when the profile is public.

- **Pending reviews** shows submissions awaiting a duplicate decision and
  lets you turn review on or off for the active profile.
- **Sources and last submission** lists the latest submission for each
  integration, source format and declared client, with the received time, time
  precision, and current relay delivery state. Use the next page to see older
  sources when the list spans multiple pages.
- **Relay queue and failures** shows the number of queued deliveries, the pause
  reason when the destination is paused, and the most recent open or failed
  deliveries with their attempt count and reason code.
- **Recent deduplication decisions** lists the latest 25 decisions with the
  mode, decision, reason code, rule version, time precision, and the recorded
  evidence: candidate count, time difference, matching recording ID, trusted
  external ID, and whether the album title differs from MusicBrainz.

The management page of a listen adds **Why is this listen here?** with its
source, received time, time precision, rule, reason code, and delivery state.
Listens saved before receipts existed show only their recorded origin, and the
missing fields read **Not recorded**. Times are shown in UTC.

The pages never show API keys, relay URLs or tokens, raw request bodies, or the
stored evidence documents. Deliveries and decisions of one profile never appear
on another profile's page.

## Durable ListenBrainz relay

For active profiles, the submission acknowledgement means the local receipts
were committed. Accepted API events are queued for delivery. Pending,
invalid and duplicate decisions are skipped. Legacy and shadow submissions
still forward the original request, as does playing now. Files and integrated
sources do not create relay deliveries.

The worker sends up to 100 events at a time. A destination that is unavailable
or rate-limited pauses independently. Shutdown preserves queued work for the
next run. Delivery is at least once, so a remote service can receive the same
event twice. This can happen after a crash following remote acceptance, a
failed local commit of the 2xx result, or cancellation while the remote
outcome is unknown. TuneTrail still tries to record a 2xx observed before
cancellation.

| Destination | Duplicate handling for an exact re-delivery |
| --- | --- |
| ListenBrainz | Unique combination of user, track, and `listened_at`. |
| Koito | Primary key on `track_id` and `listened_at`. |
| TuneTrail | v9 deduplicates by profile, track, and `listened_at`. With review enabled, either resolver can hold a possible duplicate for a decision. |
| Maloja and multi-scrobbler | To be confirmed in the relay trial. |

These keys describe receiver behavior. TuneTrail cannot guarantee that an
individual destination will suppress a re-delivery.
Changing the URL stops eligibility for the former destination. A response to
an old URL that arrives after the change leaves the item failed as
`destination_changed`, even if the old server accepted it. Explicit retry
can send it to the new destination.

For a local relay, save its URL and sign in as an instance administrator.
Open the profile's relay settings or **Server → Relay exceptions by
destination**, choose **Allow this relay**, then confirm the address, port, and
permissions shown. The button grants only the HTTP or private-network
permissions that this destination needs and resumes delivery. For a DNS name,
the confirmation lists its current addresses. HTTP permission follows the
configured name and port. Private access stays limited to the listed
addresses, so confirm again if the private address changes. Loopback and
link-local always require a destination exception. Cloud metadata and other
forbidden ranges are never eligible. The official ListenBrainz service and
other public HTTPS destinations need no exception.

The panel lists and removes existing exceptions. Advanced CIDR rules are
available through the root relay permissions API.

**Upgrading from v11:** By default, HTTP and LAN relays on non-administrator
profiles stop delivering until an instance administrator grants the required
permissions. The upgrade can seed eligible exceptions for administrator
profiles only. A DNS lookup failure at startup postpones delivery briefly and
does not seed an exception. The next attempt applies the current network policy.
If an existing administrator relay on the LAN needed a seeded exception, root
must add it when the next attempt reports `private_network_not_allowed`.

Larger databases need more free disk space and may take longer to upgrade.
Budget for the live database, WAL, and physical backup to coexist, and plan a
maintenance window before serving requests.

The **Relay** section above the exceptions offers two instance-wide permissions.
Each defaults to false and requires restart. The saved and running values and
their origins are displayed in the panel. The file equivalent is:

```toml
[relay]
allow_http = false
allow_private_network = false
```

Both flags apply to **every profile, including non-root owners**. With
`allow_private_network` enabled, any profile owner can reach LAN services
via GET `{base}/validate-token` and POST `{base}/submit-listens`. Prefer
destination exceptions. Loopback and link-local addresses still need
per-destination exceptions. Permanently forbidden ranges remain blocked.

These restrictions apply only to durable delivery and the connection test.
The legacy/shadow raw relay and `playing_now` use the v0.48 Sender with
the default transport, outside the global flags and destination exceptions.

The dialer checks permissions in this order:

1. Forbidden endpoints and ranges always stay blocked, including cloud metadata
   and the blocked IPv6 translation ranges.
2. A global permission applies to its corresponding restriction, except that loopback and link-local access always needs a destination exception.
3. A matching host:port exception can allow HTTP or a specific reserved CIDR.
4. Without permission, cleartext and reserved networks stay blocked.

The API also supports changing exceptions and resuming a saved relay under
`/apis/web/v1/profiles/{slug}/relay`. Root can convert an already-active profile
at `POST /apis/web/v1/profiles/{slug}/ingest-activation/occurrence` once its
configured relay passes validation and has no current pause. Open v9 receipts
and unlinked shadow receipts still block the cut.

Relay warnings include the configured scheme, host and port, a stable reason,
and the range category when applicable. They omit DNS answers, paths, queries
and credentials. The root panel shows the last pause reason and an instruction
after failed revalidation. Non-root users receive only the policy or network
class. The legacy raw relay retains its previous network behavior.

The saved token is omitted from the admin response and from the HTML field.
Leave it blank to keep the same credential for the displayed URL, or supply a
new token when changing the URL.

Every physical SQLite backup or full database copy is as sensitive as the
live database. It preserves password hashes, API keys, and any configured
relay token in plaintext, where present in the source schema. TuneTrail creates
automatic backups alongside the database with owner-only permissions
(`0600`), including `.pre-relay-delivery-*.bak`, `.pre-occurrence-ingest-*.bak`,
and `koito.db.pre-tunetrail-*.bak`. Preserve those permissions on copies. Keep
backups out of shared locations, do not send them to third parties, and protect
or delete them using the same procedures as the live database. History exports,
the administrative relay GET, and relay diagnostics omit the token.

Deleting history removes unsent relay work for those events and keeps at most
30 days of daily aggregate sent counts. Purging all history removes those
aggregates too.

Shutdown stops new delivery claims and gives in-flight HTTP a bounded grace
period. The supplied Compose deployment allows 30 seconds for shutdown.
Outside Compose, Docker's default 10-second stop can interrupt shutdown if the
host is slow. Configure a longer stop timeout when deploying that way.

### Relay configuration recovery on schema 12

The relay editor saves a canonical URL. Scheme and host are lowercase, the
default port is omitted, and the endpoint path is preserved. The admin GET
returns the saved form.

Use TuneTrail's relay editor to repair the saved URL or revoke the relay.
A direct update to `profile_relays` through the standalone `sqlite3` CLI or
Python can fail with `no such function: tt_relay_canonical`. The schema uses
that function to compare endpoint identities, and TuneTrail registers it on
its own SQLite connections. Keep the relay triggers in place.

If the application cannot be recovered, stop TuneTrail and restore the
SQLite-consistent `.pre-relay-delivery-*.bak` snapshot into a clean data
directory before starting the schema 11 build. Keep the newer database
separately because restoring the backup rolls back later local changes.

## Permanent failures

The active resolver makes up to five automatic attempts for transient failures,
with waits of 1, 2, 4, and 8 minutes. After the fifth failure it parks the
receipt and keeps the original submission and its idempotency key. An unreadable
listen time or snapshot is recorded as invalid and closes without further
automatic attempts.

Open **Health** in the administration menu and look under **Permanent failures**.
The heading shows the total and the list shows the 25 most recently parked
receipts. The profile owner can view their own queue, and root can select any
profile. After correcting the cause, select **Reprocess** to queue the receipt
immediately with a fresh five-attempt budget. The displayed attempt count tracks
all claims for that receipt, including recoveries after an interrupted attempt.
