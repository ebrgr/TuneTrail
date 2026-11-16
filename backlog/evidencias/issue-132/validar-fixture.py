"""Validate a conceptual identity contract in temporary SQLite, without application data."""
from pathlib import Path
import sqlite3


# main checks observable contract invariants, not production resolver behavior.
def main():
    connection = sqlite3.connect(":memory:")
    try:
        connection.executescript(Path(__file__).with_name("fixture-identidade.sql").read_text())
        checks = 0

        # verify reports each independently observable invariant.
        def verify(condition, message):
            nonlocal checks
            if not condition:
                raise AssertionError(message)
            checks += 1
            print("PASS", message)

        # scalar keeps each verification query scoped to this disposable fixture.
        def scalar(query, args=()):
            return connection.execute(query, args).fetchone()[0]

        verify(scalar("SELECT COUNT(*) FROM events WHERE profile_id='P1' AND listened_at<2000") == 2,
               "P1 A/B: two events")
        for profile, expected in (("P1", 2), ("P2", 1)):
            verify(scalar("SELECT COUNT(*) FROM events e JOIN occurrences o ON o.id=e.occurrence_id "
                          "WHERE profile_id=? AND recording_id='R' AND listened_at<2000", (profile,)) == expected,
                   f"{profile}: recording count")
        for profile, release, expected in (("P1", "A", 1), ("P1", "B", 1), ("P2", "A", 1), ("P2", "B", 0)):
            verify(scalar("SELECT COUNT(*) FROM events e JOIN occurrences o ON o.id=e.occurrence_id "
                          "WHERE profile_id=? AND release_id=? AND listened_at<2000", (profile, release)) == expected,
                   f"{profile}: album {release} count")
        verify(scalar("SELECT COUNT(*) FROM events WHERE profile_id='P1' AND listened_at=2000") == 2,
               "distinct source events in same second survive")
        verify(scalar("SELECT COUNT(DISTINCT occurrence_id) FROM events WHERE listened_at=4000") == 2,
               "two Intro occurrences in same album remain distinct")
        verify(scalar("SELECT COUNT(*) FROM receipts WHERE status LIKE 'pending_%'") == 3
               and scalar("SELECT COUNT(*) FROM events") == 8, "pending receipts do not count as listens")
        verify(scalar("SELECT COUNT(*) FROM receipts WHERE event_id='E4'") == 2
               and scalar("SELECT COUNT(*) FROM events WHERE id='E4'") == 1, "two receipts can identify one event")
        verify(scalar("SELECT COUNT(*) FROM events e JOIN occurrences o ON o.id=e.occurrence_id WHERE o.release_id IS NULL") == 1,
               "missing album remains unassigned")
        for profile in ("P1", "P2"):
            total = scalar("SELECT COUNT(*) FROM events WHERE profile_id=?", (profile,))
            groups = connection.execute("SELECT o.release_id, COUNT(*) FROM events e JOIN occurrences o ON o.id=e.occurrence_id "
                                        "WHERE profile_id=? GROUP BY o.release_id", (profile,)).fetchall()
            verify(sum(count for _, count in groups) == total, f"{profile}: album sum including unknown equals total")
        try:
            with connection:
                connection.execute("INSERT INTO receipts(id,profile_id,source_instance,external_id,status,event_id) "
                                   "VALUES('reimport','P1','player-1','source-1','accepted','E1')")
        except sqlite3.IntegrityError:
            pass
        else:
            raise AssertionError("source identity was duplicated")
        verify(scalar("SELECT COUNT(*) FROM events") == 8, "same receipt cannot create another listen")
        try:
            with connection:
                connection.execute("UPDATE receipts SET status='accepted',event_id='E1' WHERE id='C3'")
        except sqlite3.IntegrityError:
            pass
        else:
            raise AssertionError("cross-profile event link accepted")
        verify(scalar("SELECT event_id FROM receipts WHERE id='C3'") == "E3", "cross-profile link rejected")
        before = scalar("SELECT COUNT(*) FROM events")
        try:
            with connection:
                connection.execute("INSERT INTO events VALUES('rolled-back','P1','RB',5000)")
                connection.execute("UPDATE receipts SET status='accepted',event_id='missing-event' WHERE id='C9'")
        except sqlite3.IntegrityError:
            pass
        else:
            raise AssertionError("invalid decision was committed")
        verify(scalar("SELECT COUNT(*) FROM events") == before, "failed resolution rolls back event insertion")
        # Review uses profile + expected version; replay sees no pending receipt.
        for _ in range(2):
            with connection:
                row = connection.execute("SELECT version FROM receipts WHERE id='C9' AND profile_id='P1' "
                                         "AND status='pending_identity' AND version=1").fetchone()
                if row:
                    connection.execute("INSERT INTO events VALUES('reviewed','P1','RB',5000)")
                    connection.execute("UPDATE receipts SET status='accepted',event_id='reviewed',version=2 "
                                       "WHERE id='C9' AND profile_id='P1' AND version=1")
        verify(scalar("SELECT COUNT(*) FROM events") == before + 1, "review replay produces one event")
        verify(connection.execute("UPDATE receipts SET reason='wrong-profile' WHERE id='C10' AND profile_id='P2'").rowcount == 0,
               "review selector does not cross profiles")
        verify(connection.execute("PRAGMA foreign_key_check").fetchall() == [], "foreign keys hold")
        verify(scalar("PRAGMA integrity_check") == "ok", "SQLite integrity holds")
        print(f"{checks} contract checks passed; no application database opened")
    finally:
        connection.close()


if __name__ == "__main__":
    main()
