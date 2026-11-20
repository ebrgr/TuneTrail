#!/usr/bin/env python3
"""Measure ISSUE-134 on disposable v0.46 and v11 servers."""

import argparse
from concurrent.futures import ThreadPoolExecutor
from contextlib import closing, contextmanager, suppress
from datetime import datetime, timezone
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
import io
import json
from pathlib import Path
import shutil
import sqlite3
import tempfile
import threading
import time
import urllib.error
import urllib.request
import zipfile

from rehearse_issue081 import PROFILE, SUBMIT, TOKEN, boot, database, init, percentiles, sample_resources, wait_for
from rehearse_issue134_e1 import verify_replay
from rehearse_issue134_e3 import catalog_query_plans
from validate_multiuser import ROOT, Server, scrobble


def post(server, payload, key):
    request = urllib.request.Request(server.origin + SUBMIT, json.dumps(payload).encode(),
                                     headers={"Content-Type": "application/json", "Authorization": "Token " + TOKEN,
                                              "Idempotency-Key": key})
    try:
        response = urllib.request.urlopen(request, timeout=30)
    except urllib.error.HTTPError as error:
        response = error
    with response:
        return response.status, response.headers.get("Retry-After"), response.read()


def large_history(conn):
    conn.executescript("""
        PRAGMA foreign_keys=ON;
        PRAGMA wal_autocheckpoint=0;
        CREATE TEMP TABLE sequence(n INTEGER PRIMARY KEY);
        INSERT INTO sequence WITH RECURSIVE seq(n) AS (SELECT 1 UNION ALL SELECT n+1 FROM seq WHERE n<300000) SELECT n FROM seq;
        INSERT INTO artists(id) VALUES(1);
        INSERT INTO artist_aliases VALUES(1,'Archive Artist','Fixture',1);
        INSERT INTO releases(id) SELECT n FROM sequence WHERE n<=200;
        INSERT INTO release_aliases SELECT n,'Archive Album '||n,'Fixture',1 FROM sequence WHERE n<=200;
        INSERT INTO artist_releases SELECT 1,n,1 FROM sequence WHERE n<=200;
        INSERT INTO tracks(id,release_id,duration) SELECT n,n,180 FROM sequence WHERE n<=200;
        INSERT INTO track_aliases SELECT n,'Archive Song '||n,'Fixture',1 FROM sequence WHERE n<=200;
        INSERT INTO artist_tracks SELECT 1,n,1 FROM sequence WHERE n<=200;
        INSERT INTO integrations(id,profile_id,source_type,created_at) VALUES('listenbrainz:1',1,'listenbrainz',0);
        INSERT INTO listens(profile_id,track_id,listened_at,event_id,client)
        SELECT 1,(n-1)%200+1,1640995200+n*4,'history-'||n,'Fixture' FROM sequence;
        INSERT INTO ingest_events(receipt_id,profile_id,source_instance_id,schema_version,source_type,adapter_version,
            batch_id,item_ordinal,operation,received_at,listened_at,original_time,time_precision,time_semantics,
            original_snapshot,snapshot_digest,digest_version,mode,state,identity_status,duplicate_decision,
            resolution_version,rule_version,occurrence_id,recording_id,event_id,closed_at)
        SELECT 'receipt-'||n,1,'listenbrainz:1',1,'listenbrainz','1','imported-history',n,'import',l.listened_at,
            strftime('%Y-%m-%dT%H:%M:%S',l.listened_at,'unixepoch')||'.000000000Z',CAST(l.listened_at AS TEXT),
            'second','start',json_object('title','Archive Song '||l.track_id,'artist','Archive Artist',
            'release','Archive Album '||l.track_id,'original_time',CAST(l.listened_at AS TEXT),'duration_ms',180000),
            'history-digest-'||n,'sha256-json-v2','active','accepted','resolved','different',1,'active-v1',
            l.track_id,o.recording_id,l.event_id,l.listened_at
        FROM sequence JOIN listens l ON l.event_id='history-'||n JOIN occurrences o ON o.id=l.track_id;
        INSERT INTO ingest_revisions
        SELECT profile_id,receipt_id,1,'automatic','different','["fixture"]','{}','active-v1',received_at FROM ingest_events;
        DROP TABLE sequence;
    """)


def benchmark_load(server, directory, duration, rate):
    first = 1781000000
    count = duration * rate
    arrivals, acknowledgements, visible, statuses = {}, {}, {}, {}
    stop = threading.Event()

    def observe():
        with database(directory) as conn:
            while not stop.is_set():
                now = time.monotonic()
                for (stamp,) in conn.execute("SELECT listened_at FROM listens WHERE profile_id=1 AND listened_at>=?", (first,)):
                    visible.setdefault(stamp, now)
                stop.wait(.025)

    def send(stamp):
        arrivals[stamp] = time.monotonic()
        status, _, _ = post(server, scrobble(stamp), f"load-{stamp}")
        acknowledgements[stamp] = time.monotonic()
        statuses[stamp] = status

    observer = threading.Thread(target=observe)
    observer.start()
    initial_wal = (directory / "tunetrail.db-wal").stat().st_size
    try:
        with sample_resources(server) as peaks:
            beginning = time.monotonic()
            with ThreadPoolExecutor(max_workers=16) as pool:
                futures = []
                for i in range(count):
                    remaining = beginning + i / rate - time.monotonic()
                    if remaining > 0:
                        time.sleep(remaining)
                    futures.append(pool.submit(send, first + i * 4))
                for future in futures:
                    future.result()
            assert set(statuses.values()) == {200}, statuses
            wait_for(lambda: len(visible) == count)
            elapsed = max(visible.values()) - beginning
    finally:
        stop.set()
        observer.join()
    with database(directory, timeout=0) as lock:
        lock.execute("BEGIN IMMEDIATE")
        before_busy = time.monotonic()
        status, retry, _ = post(server, scrobble(first + count * 4), "busy-probe")
        busy_ms = (time.monotonic() - before_busy) * 1000
        assert (status, retry) == (503, "1")
        lock.rollback()
    assert post(server, scrobble(first + count * 4), "busy-probe")[0] == 200
    wait_for(lambda: event_count(directory, first) == count + 1)
    return {"count": count, "rate_target_per_second": rate, "duration_target_s": duration,
            "visible_per_second": round(count / elapsed, 2), "elapsed_s": round(elapsed, 3),
            "ack": percentiles([acknowledgements[s] - arrivals[s] for s in arrivals]),
            "visibility_since_send": percentiles([visible[s] - arrivals[s] for s in arrivals]),
            "visibility_since_ack": percentiles([max(0, visible[s] - acknowledgements[s]) for s in arrivals]),
            "busy_ms": round(busy_ms, 1), "retry_after_seconds": retry, "initial_wal_bytes": initial_wal, **peaks}


def event_count(directory, first):
    with database(directory) as conn:
        return conn.execute("SELECT count(*) FROM listens WHERE profile_id=1 AND listened_at>=?", (first,)).fetchone()[0]


def claim_query_plans(conn):
    code = (ROOT / "internal/db/sqlite/ingest_resolve.go").read_text()
    selection = code.split("const nextReceipt = `", 1)[1].split("`", 1)[0]
    claim_prefix = "UPDATE ingest_events SET lease_owner"
    claim = claim_prefix + code.split("`" + claim_prefix, 1)[1].split("`", 1)[0]
    return {
        "queue_query_plan": [row[3] for row in conn.execute("EXPLAIN QUERY PLAN " + selection, (0, 0))],
        "claim_update_query_plan": [row[3] for row in conn.execute("EXPLAIN QUERY PLAN " + claim,
                                      ("fixture", 120, "receipt", 0, 0))],
    }


def sustained(binary, baseline, root, duration, rate):
    directory = root / "load"
    init(baseline, directory)
    with closing(sqlite3.connect(directory / "tunetrail.db")) as conn:
        large_history(conn)
        conn.commit()
        original_wal = (directory / "tunetrail.db-wal").stat().st_size
        assert original_wal > 0
        start = time.monotonic()
        server = boot(binary, directory, True)
        upgrade_seconds = time.monotonic() - start
        upgraded_bytes = sum(path.stat().st_size for path in directory.iterdir() if path.is_file())
    try:
        with database(directory) as conn:
            assert conn.execute("SELECT count(*) FROM ingest_events WHERE profile_id=1").fetchone() == (300000,)
            conn.execute("INSERT INTO profile_ingest_flags(profile_id,active,occurrence_write) VALUES(1,1,1)")
            plans = catalog_query_plans(conn)
            plans.update(claim_query_plans(conn))
        workload = benchmark_load(server, directory, duration, rate)
    finally:
        server.stop()
    backup, = directory.glob("*.pre-occurrence-ingest-*.bak")
    restored = root / "restored"
    restored.mkdir()
    shutil.copyfile(backup, restored / "tunetrail.db")
    server = boot(baseline, restored)
    try:
        with database(restored) as conn:
            assert conn.execute("SELECT max(version_id) FROM goose_db_version").fetchone() == (9,)
            assert conn.execute("SELECT count(*) FROM ingest_events WHERE profile_id=1").fetchone() == (300000,)
            assert conn.execute("PRAGMA foreign_key_check").fetchall() == []
    finally:
        server.stop()
    old = Server(baseline, directory)
    try:
        old.start()
    except RuntimeError:
        assert "unsupported schema version 11" in (directory / "server.log").read_text()
    else:
        raise AssertionError("v0.46 opened schema v11")
    finally:
        old.stop()
    return {"retained_same_profile": 300000, "upgrade_health_s": round(upgrade_seconds, 3),
            "v9_wal_bytes": original_wal, "upgrade_bytes_at_health": upgraded_bytes,
            "backup_bytes": backup.stat().st_size, "restore_v046": True, "v046_refuses_v11": True,
            "workload": workload, **plans}


@contextmanager
def loopback(handler):
    server = ThreadingHTTPServer(("127.0.0.1", 0), handler)
    worker = threading.Thread(target=server.serve_forever, daemon=True)
    worker.start()
    try:
        yield f"http://127.0.0.1:{server.server_port}"
    finally:
        server.shutdown()
        server.server_close()
        worker.join()


def parity(binary, baseline, root):
    projection = verify_replay(binary, baseline, root)
    relayed = []

    class Relay(BaseHTTPRequestHandler):
        def log_message(self, *_):
            pass

        def do_POST(self):
            body = self.rfile.read(int(self.headers["Content-Length"]))
            relayed.append((self.path, self.headers.get("Authorization"), body))
            self.send_response(200)
            self.end_headers()

    payload = scrobble(1783000000)
    with loopback(Relay) as url:
        for index, executable in enumerate((baseline, binary)):
            directory = root / f"relay-{index}"
            init(executable, directory)
            with database(directory) as conn:
                conn.execute("INSERT INTO profile_relays VALUES(1,?,'fixture-relay')", (url,))
            server = boot(executable, directory)
            try:
                assert post(server, payload, "legacy-relay")[0] == 200
                wait_for(lambda: len(relayed) == index + 1)
            finally:
                server.stop()
    assert relayed[0] == relayed[1]
    assert relayed[1] == ("/submit-listens", "Token fixture-relay", json.dumps(payload).encode())
    projection["restore_v046"] = projection.pop("restore_v044")
    return {**projection, "relay_body_and_headers_equal": True}


def overlap_files(stamp):
    artist, album, title = "Shared Artist", "Shared Album", "Shared Song"
    meta = {"artist_name": artist, "release_name": album, "track_name": title}
    archive = io.BytesIO()
    with zipfile.ZipFile(archive, "w") as zipped:
        zipped.writestr("listens/fixture.jsonl", '{"listened_at":' + str(stamp) + '.125,"track_metadata":'
                       + json.dumps(meta) + "}\n")
    yield "listenbrainz", archive.getvalue()
    yield "lastfm", json.dumps([{"track": [{"artist": {"#text": artist}, "album": {"#text": album},
                                          "name": title, "date": {"uts": str(stamp)}}]}]).encode()
    clock = datetime.fromtimestamp(stamp, timezone.utc).strftime("%Y-%m-%dT%H:%M:%S")
    yield "spotify", json.dumps([{"ts": clock + ".2500Z", "master_metadata_track_name": title,
                                  "master_metadata_album_artist_name": artist,
                                  "master_metadata_album_album_name": album, "reason_end": "trackdone",
                                  "ms_played": 180000}]).encode()
    yield "maloja", json.dumps({"scrobbles": [{"time": stamp, "track": {"artists": [artist], "title": title,
                                  "album": {"albumtitle": album}}}]}).encode()

    def aliases(value):
        return {"aliases": [{"alias": value, "is_primary": True}]}

    yield "koito", json.dumps({"version": "1", "listens": [{"listened_at": clock + ".100Z",
                                "track": aliases(title), "album": aliases(album),
                                "artists": [aliases(artist)]}]}).encode()


def occurrence_fixture(binary, directory, **env):
    init(binary, directory)
    with database(directory) as conn:
        conn.execute("INSERT INTO profile_ingest_flags(profile_id,active,occurrence_write) VALUES(1,1,1)")
    return boot(binary, directory, True, **env)


def overlap(binary, root):
    directory = root / "overlap"
    server = occurrence_fixture(binary, directory)
    try:
        server.login()
        stamp = 1783001000
        for index in range(20):
            first = stamp + index * 10
            payload = scrobble(first)
            payload["payload"][0]["track_metadata"].update(artist_name="Shared Artist", release_name="Shared Album", track_name="Shared Song")
            assert post(server, payload, f"primary-{index}")[0] == 200
        wait_for(lambda: event_count(directory, stamp) == 20)
        grouped = {}
        for index in range(20):
            for format, data in overlap_files(stamp + index * 10):
                grouped.setdefault(format, []).append(data)
        # Upload separate files to retain the genuine source digest and time representation.
        for format, files in grouped.items():
            for data in files:
                server.upload(PROFILE + "/import", format, data)

        def pending():
            with database(directory) as conn:
                return conn.execute("SELECT count(*) FROM ingest_events WHERE state='pending_duplicate'").fetchone()[0]

        wait_for(lambda: pending() == 100)
        assert event_count(directory, stamp) == 20
        with database(directory) as conn:
            distribution = conn.execute("""SELECT source_type,coalesce(file_format,'api'),state,time_precision,reason_codes,count(*)
                FROM ingest_events GROUP BY 1,2,3,4,5""").fetchall()
            evidence = conn.execute("""SELECT count(*) FROM ingest_events WHERE state='pending_duplicate'
                AND json_array_length(json_extract(evidence,'$.candidates'))=1
                AND json_extract(evidence,'$.candidates[0]') LIKE 'event:%'""").fetchone()[0]
            assert evidence == 100
            assert conn.execute("SELECT count(*) FROM outbox").fetchone() == (0,)
        return {"events": 20, "pending": 100, "one_candidate_each": evidence, "distribution": distribution}
    finally:
        server.stop()


class MusicBrainzFixture(BaseHTTPRequestHandler):
    """Serve typed MusicBrainz release proofs with per-release fault controls."""

    fixtures = {}
    entered = threading.Event()
    unblock = threading.Event()

    def log_message(self, *_):
        pass

    def do_GET(self):
        identity = self.path.split("?", 1)[0].rsplit("/", 1)[-1]
        case = self.fixtures.get(identity)
        if case is None:
            self.send_response(404)
            self.end_headers()
            return
        album, title, recording, mode = case
        if mode == "failure":
            self.send_response(503)
            self.end_headers()
            return
        if mode == "slow":
            time.sleep(.6)
        if mode == "hold":
            self.entered.set()
            self.unblock.wait(35)
        data = {"id": identity, "title": album, "media": [{"position": 1, "tracks": [
            {"id": fix_id(int(identity[-3:], 16) + 100), "title": title, "position": 1,
             "recording": {"id": recording}}]}]}
        self.send_response(200)
        self.end_headers()
        with suppress(OSError):
            self.wfile.write(json.dumps(data).encode())


def fix_id(number):
    return f"fa000000-0000-4000-8000-{number:012x}"


def submit_identity(server, stamp, key, album, title, release=None, recording=None, artist="Fixture Artist"):
    payload = scrobble(stamp)
    meta = payload["payload"][0]["track_metadata"]
    meta.update(track_name=title, release_name=album, artist_name=artist)
    info = meta["additional_info"]
    info.update(discnumber=1, tracknumber=1)
    if release:
        info["release_mbid"] = release
    if recording:
        info["recording_mbid"] = recording
    return post(server, payload, key)


def receipt_state(directory, stamp):
    with database(directory) as conn:
        return conn.execute("""SELECT state,attempt_count,lease_until,event_id,reason_codes FROM ingest_events
            WHERE profile_id=1 AND original_time=? ORDER BY received_at DESC,receipt_id LIMIT 1""", (str(stamp),)).fetchone()


def wait_state(directory, stamp, state, timeout=150):
    return wait_for(lambda: (row := receipt_state(directory, stamp)) and row[0] == state and row, timeout=timeout)


def mbz_env(url):
    return {"TUNETRAIL_DISABLE_MUSICBRAINZ": "false", "TUNETRAIL_MUSICBRAINZ_URL": url}


def fragmentation(binary, root):
    directory = root / "fragmentation"
    title = "Fragmented song"
    common = fix_id(90)
    fixtures = {
        fix_id(1): ("Album A", title, common, "failure"),
        fix_id(2): ("Album B", title, common, "failure"),
        fix_id(3): ("Slow Album", "Slow song", fix_id(91), "slow"),
        fix_id(4): ("Timeout Album", "Timeout song", fix_id(92), "hold"),
        fix_id(5): ("Proven A", "Proven song", fix_id(93), "proof"),
        fix_id(6): ("Proven B", "Proven song", fix_id(93), "proof"),
    }

    class Stub(MusicBrainzFixture):
        pass

    Stub.fixtures = fixtures
    Stub.entered, Stub.unblock = threading.Event(), threading.Event()
    with loopback(Stub) as url:
        server = occurrence_fixture(binary, directory, **mbz_env(url))
        stamp = 1783010000

        def send(i, key, state="accepted"):
            nonlocal stamp
            stamp += 10
            release = fix_id(i)
            album, song, recording, _ = fixtures[release]
            start = time.monotonic()
            assert submit_identity(server, stamp, key, album, song, release, recording)[0] == 200
            row = wait_state(directory, stamp, state, 40)
            return round(time.monotonic() - start, 3), row

        try:
            send(1, "fail-a")
            send(2, "fail-b")
            with database(directory) as conn:
                provisional = conn.execute("""SELECT count(DISTINCT r.id),count(o.id)
                    FROM recordings r JOIN occurrences o ON o.recording_id=r.id
                    JOIN track_aliases t ON t.track_id=o.id WHERE t.alias=? AND r.musicbrainz_id IS NULL""",
                    (title,)).fetchone()
                assert provisional == (2, 2)
            for i in (1, 2):
                fixtures[fix_id(i)] = (*fixtures[fix_id(i)][:3], "proof")
            send(2, "recover-b")
            _, conflict = send(1, "recover-a", "pending_identity")
            delay, slow = send(3, "slow")
            timeout, timed_out = send(4, "timeout")
            Stub.unblock.set()
            with database(directory) as conn:
                timeout_provisional = conn.execute("""SELECT r.musicbrainz_id IS NULL FROM ingest_events e
                    JOIN recordings r ON r.id=e.recording_id WHERE e.original_time=?""", (str(stamp),)).fetchone()[0]
                assert timeout_provisional == 1

            for artist in ("Performer One", "Performer Two"):
                stamp += 10
                assert submit_identity(server, stamp, str(stamp), "Compilation", "Intro", artist=artist)[0] == 200
                wait_state(directory, stamp, "accepted")
            for album in ("Night Sessions", "Night Sessions Deluxe"):
                stamp += 10
                assert submit_identity(server, stamp, str(stamp), album, "Intro")[0] == 200
                wait_state(directory, stamp, "accepted")
            with database(directory) as conn:
                compilation = conn.execute("SELECT count(DISTINCT release_id) FROM release_aliases WHERE alias='Compilation'").fetchone()[0]
                similar = conn.execute("""SELECT count(DISTINCT release_id) FROM release_aliases
                    WHERE alias IN ('Night Sessions','Night Sessions Deluxe')""").fetchone()[0]
                assert (compilation, similar) == (2, 2)
                conn.execute("""INSERT INTO release_aliases SELECT release_id,'Shared alias','Fixture',0
                    FROM release_aliases WHERE alias IN ('Night Sessions','Night Sessions Deluxe')""")

            stamp += 10
            assert submit_identity(server, stamp, "alias", "Shared alias", "Intro")[0] == 200
            wait_state(directory, stamp, "pending_identity")
            with database(directory) as conn:
                alias_evidence = conn.execute("""SELECT reason_codes,json_extract(evidence,'$.candidates')
                    FROM ingest_events WHERE original_time=?""", (str(stamp),)).fetchone()
            send(5, "proven-a")
            send(6, "proven-b")
            stamp_b = stamp
            with database(directory) as conn:
                count_b = conn.execute("""SELECT count(*),count(DISTINCT o.recording_id),count(DISTINCT o.release_id)
                    FROM listens l JOIN occurrences o ON o.id=l.track_id JOIN track_aliases t ON t.track_id=o.id
                    WHERE t.alias='Proven song'""").fetchone()
                assert count_b == (2, 1, 2)
                try:
                    conn.execute("UPDATE profile_ingest_flags SET occurrence_write=0 WHERE profile_id=1")
                except sqlite3.IntegrityError as error:
                    assert "cannot be disabled" in str(error)
                else:
                    raise AssertionError("converted profile opted out")
        finally:
            Stub.unblock.set()
            server.stop()
        disabled = boot(binary, directory, False)
        try:
            status, retry, _ = submit_identity(disabled, stamp_b, "proven-b", "Proven B", "Proven song", fix_id(6), fix_id(93))
            assert (status, retry) == (503, "1")
            with database(directory) as conn:
                assert conn.execute("SELECT count(*) FROM listens WHERE listened_at=?", (stamp_b,)).fetchone() == (1,)
                assert conn.execute("SELECT count(*) FROM ingest_events WHERE original_time=?", (str(stamp_b),)).fetchone() == (1,)
        finally:
            disabled.stop()
    return {"unavailable_provisional_recordings": provisional[0], "unavailable_occurrences": provisional[1],
            "recovery_a_state": conflict[0], "recovery_a_reason": conflict[4],
            "slow_s_including_queue": delay, "slow_attempts": slow[1],
            "timeout_s_including_queue": timeout, "timeout_attempts": timed_out[1],
            "timeout_provisional": bool(timeout_provisional), "compilation_releases": compilation,
            "similar_titles_releases": similar, "shared_alias_evidence": alias_evidence,
            "proven_ab_listens_recordings_releases": count_b, "one_way_flag": True, "master_off_503": True}


def faults(binary, root):
    directory = root / "faults"
    release = fix_id(20)
    album, title, recording = "Crash album", "Crash song", fix_id(99)

    class Stub(MusicBrainzFixture):
        fixtures = {release: (album, title, recording, "hold")}
        entered, unblock = threading.Event(), threading.Event()

    stamp = 1783020000
    with loopback(Stub) as url:
        server = occurrence_fixture(binary, directory, **mbz_env(url))
        try:
            assert submit_identity(server, stamp, "crash", album, title, release, recording)[0] == 200
            assert Stub.entered.wait(15)
            state, attempts, lease, event, _ = receipt_state(directory, stamp)
            assert (state, attempts, event) == ("received", 1, None)
            server.process.kill()
            server.process.wait()
            assert server.process.returncode == -9
            killed_at = time.monotonic()
            lease_left = lease - time.time()
        finally:
            Stub.unblock.set()
            server.stop()
    with database(directory) as conn:
        assert conn.execute("SELECT count(*) FROM tracks").fetchone() == (0,)
    server = boot(binary, directory, False)
    try:
        assert receipt_state(directory, stamp)[:2] == ("received", 1)
        resumed = wait_state(directory, stamp, "accepted")
        assert resumed[1] == 2 and resumed[3]
        crash_recovery = time.monotonic() - killed_at
        assert crash_recovery >= lease_left - 1
        assert event_count(directory, stamp) == 1
    finally:
        server.stop()

    server = boot(binary, directory, True)
    fault_stamp = stamp + 20
    try:
        with database(directory) as conn:
            conn.executescript("""CREATE TABLE e4_commit_failure(id INTEGER REFERENCES listening_profiles(id) DEFERRABLE INITIALLY DEFERRED);
                CREATE TRIGGER e4_commit_failure_inject AFTER INSERT ON ingest_revisions WHEN NEW.rule_version='occurrence-v1'
                BEGIN INSERT INTO e4_commit_failure VALUES(-1); END;""")
        assert submit_identity(server, fault_stamp, "commit", "Rollback album", "Rollback song")[0] == 200
        failed = wait_state(directory, fault_stamp, "retryable")
        assert failed[1] == 1 and failed[3] is None
        with database(directory) as conn:
            assert conn.execute("SELECT count(*) FROM track_aliases WHERE alias='Rollback song'").fetchone() == (0,)
            assert conn.execute("SELECT count(*) FROM listens WHERE listened_at=?", (fault_stamp,)).fetchone() == (0,)
            conn.execute("DROP TRIGGER e4_commit_failure_inject")
            conn.execute("UPDATE ingest_events SET next_attempt_at=0 WHERE original_time=?", (str(fault_stamp),))
        resumed_commit = wait_state(directory, fault_stamp, "accepted")
        assert resumed_commit[1] == 2
        assert event_count(directory, stamp) == 2
    finally:
        server.stop()
    return {"sigkill": True, "lease_remaining_at_kill_s": round(lease_left, 2),
            "crash_to_accepted_s": round(crash_recovery, 2), "attempts_after_crash": resumed[1],
            "commit_rollback_without_catalog_residue": True, "commit_retry_attempts": resumed_commit[1]}


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--binary", type=Path, required=True)
    parser.add_argument("--baseline", type=Path, required=True)
    parser.add_argument("--scenario", choices=("load", "parity", "overlap", "fragmentation", "faults"), default="load")
    parser.add_argument("--duration", type=int, default=30)
    parser.add_argument("--rate", type=int, default=20)
    args = parser.parse_args()
    with tempfile.TemporaryDirectory(prefix="tunetrail-e4-") as tmp:
        root = Path(tmp)
        binary, baseline = args.binary.resolve(), args.baseline.resolve()
        if args.scenario == "load":
            result = sustained(binary, baseline, root, args.duration, args.rate)
        elif args.scenario == "parity":
            result = parity(binary, baseline, root)
        elif args.scenario == "overlap":
            result = overlap(binary, root)
        elif args.scenario == "fragmentation":
            result = fragmentation(binary, root)
        else:
            result = faults(binary, root)
        print(json.dumps(result), flush=True)


if __name__ == "__main__":
    main()
