# ISSUE-183 verification

## Occurrence writes and restore

- `TestOccurrenceArchiveRoundTrip` uses two profiles playing the same two
  releases at the same timestamp. It checks occurrence-local alias edits,
  recording reconciliation and undo with provisional MBIDs, profile-scoped
  cursor pages, independent same-second events, release and artist links after
  restore, and a standalone occurrence with neither a release nor a legacy
  track. It restores both profiles into a separate database with occurrence
  writing active and retries both restores. It also checks exact deletion,
  rejection of a cross-profile event ID, and occurrence-local cascades.
- `TestOccurrenceReconciliationRevisesActiveReceipts` creates two accepted
  receipts through the occurrence writer. Direct SQL reassignment remains
  guarded. Merge and undo keep event and occurrence IDs stable, advancing the
  receipt revision and updating its recording link atomically.
- `TestOccurrenceHTTPCompatibility183` compares profile readers before and
  after activation, exercises both export versions and the undo route, deletes
  an exact event, then uploads the v2 archive twice over HTTP. The second
  profile's readers remain identical throughout.
- `scripts/rehearse_issue183.py` repeats the ISSUE-133 AC3 comparison against
  the v0.55 binary (commit `b8b74c6`) and the current binary. The synthetic v13
  database contained 3,000 listens. All 39 HTTP responses matched across v13,
  migrated v14, and v14 with occurrence writing active on two profiles.
  Normalization removes request-time fields and compares v1 export entries and
  aliases as sets with multiplicity. See `HTTP-PARITY.json`.
- The ISSUE-075 UI reference was regenerated with `make ui.reference`.
  The reference comparator found 176 matching files after regeneration.
  The patch changes templates and links and adds no CSS.

## Export query plan

Run `go test -v -count=1 ./internal/db/sqlite -run '^TestOccurrenceExportPlan$'`.
The SQLite fixture inserts 300,000 events split evenly across two profiles,
all in the same second and on the same occurrence. Four additional legacy
events cover two occurrences. The test runs `ANALYZE`, checks page size 1,000,
and measures a first page and a deep cursor on the same machine.

Both `EXPLAIN QUERY PLAN` results were identical:

```text
SEARCH l USING INDEX sqlite_autoindex_listens_2 (profile_id=? AND event_id>?)
SCAN n
SEARCH o USING INTEGER PRIMARY KEY (rowid=?)
SEARCH rec USING INTEGER PRIMARY KEY (rowid=?)
SEARCH t USING INTEGER PRIMARY KEY (rowid=?) LEFT-JOIN
SEARCH r USING INTEGER PRIMARY KEY (rowid=?) LEFT-JOIN
SEARCH a USING INDEX sqlite_autoindex_archive_events_2 (profile_id=? AND event_id=?) LEFT-JOIN
CORRELATED SCALAR SUBQUERY 1
SEARCH e USING INDEX idx_ingest_event (profile_id=? AND event_id=?)
```

Neither page requires a temporary sort. Measured page reads including aliases
and credits were 2.83 ms at the start and 2.37 ms after
`event-000290000` on local arm64. The synthetic fixture intentionally repeats
one occurrence, so these times characterize repeated catalog lookups and
keyset depth.

## Operational scope

Schema v14 adds archive identity and reconciliation records. It retains legacy
same-second uniqueness for the legacy writer while permitting independent
event IDs in other origins. The export streams a series of consistent pages.
A physical backup is required for point-in-time recovery of queues, pending
receipts and raw ingestion data. Native restores retain accepted receipt
provenance as historical data and do not queue ingest or relay work.
