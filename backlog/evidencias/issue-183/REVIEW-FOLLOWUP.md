# ISSUE-183 follow-up verification

- The owner upload test uses the archive from a real owner session and replaces
  its occurrence ID with one from another profile's history. The upload
  creates a new occurrence and both the HTML and JSON routes in the owner's
  profile still return 404 for the other occurrence. The test also verifies
  reuse of an already-listened occurrence, reuse of that owner's saved mapping,
  and local ID reuse by root in a separate destination profile.
- The export concurrency test pauses a 1,002-event export after page one,
  deletes the listen with the highest SQLite rowid, then inserts two events
  sorting on opposite sides of the event-ID cursor. The in-flight export
  contains each surviving initial event once. The next export includes both
  new events. The 300,000-event plan still seeks on `(profile_id,event_id)`
  and uses the covering event index on `listen_export_order` to apply the
  ceiling. No temporary sort was reported. The first and deep pages,
  including catalog metadata, took 3.55 ms and 2.93 ms.
- The undo retention tests cover a stage-1 row upgraded from v14, a new row
  given 30 days, rejection and pruning after expiry, and continued preservation
  of the merged state and its events. Occurrence deletion continues to
  cascade its own undo entries.
- The regenerated ISSUE-075 reference matched all 176 fixture files.

## Recording reconciliation timing

Command:

```sh
go test ./internal/db/sqlite -run '^TestRecordingMergeLargeFixture$' -v -count=1
```

Local arm64 SQLite fixture with 5,001 moved occurrences and 10,000 accepted
receipts, two distinct same-second events per generated occurrence. Timings
include the reconciliation transaction and its commit, excluding setup and
assertions.

| Schema | Merge | Undo |
| --- | ---: | ---: |
| v14, before follow-up | 182.47 ms | 196.38 ms |
| v15, after follow-up | 183.86 ms | 200.85 ms |

The fixture checks all 10,000 linked receipts have a matching revision at
version 2 after merge and version 3 after undo. Both operations were below the
approximately 300 ms threshold on this fixture. Reconciliation remains atomic.

The HTTP parity rehearsal against v0.55 also passed on 3,000 listens after
the upgrade to v15 and after writer activation for two profiles. All 39
normalized responses matched in each pass.
