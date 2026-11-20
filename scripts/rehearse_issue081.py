#!/usr/bin/env python3
"""Rehearse ISSUE-081 against disposable loopback servers and synthetic data."""
import argparse
import concurrent.futures
import contextlib
import datetime
import io
import shutil
import zipfile
import http.server
import json
import math
from pathlib import Path
import sqlite3
import subprocess
import tempfile
import threading
import time
import urllib.error
import urllib.request

from rehearse_issue133 import fill
from validate_multiuser import LBZ, WEB, Server, scrobble

SUBMIT = LBZ + "/submit-listens"
PROFILE = WEB + "/profiles/public-fixture"
ACTIVATE = PROFILE + "/ingest-activation"
TOKEN = "fixture-api"


@contextlib.contextmanager
def database(directory, **kwargs):
    with contextlib.closing(sqlite3.connect(directory / "tunetrail.db", **kwargs)) as conn, conn:
        yield conn


def wait_for(probe, timeout=150):
    end = time.monotonic() + timeout
    while time.monotonic() < end:
        value = probe()
        if value:
            return value
        time.sleep(.05)
    raise AssertionError("timed out waiting for synthetic fixture")


def boot(binary, directory, master=False, **env):
    server = Server(binary, directory, environment={"TUNETRAIL_INGEST_ACTIVE": str(master).lower(), **env})
    server.start()
    return server


def init(binary, directory, history=0):
    server = boot(binary, directory)
    server.stop()
    with database(directory) as conn:
        if history:
            fill(conn, history)
        else:
            conn.execute("UPDATE listening_profiles SET slug='public-fixture' WHERE id=1")
            conn.execute("INSERT INTO api_keys(key,user_id,profile_id,created_at,label) VALUES(?,1,1,0,'Fixture')", (TOKEN,))
        conn.commit()
        conn.execute("PRAGMA wal_checkpoint(TRUNCATE)")


def set_flag(server, enabled):
    server.login()
    body = {"active": enabled, "review": False}
    if enabled:
        body["digest"] = server.request("GET", ACTIVATE, admin=True)["digest"]
    server.request("PUT", ACTIVATE, body, admin=True, expected=204)


def submit(server, stamp, expected=200, key=None, recording=None, title=None):
    payload = scrobble(stamp)
    meta = payload["payload"][0]["track_metadata"]
    if recording:
        meta["additional_info"] = {"recording_mbid": recording}
    if title:
        meta["track_name"] = title
    return server.request("POST", SUBMIT, payload, token=TOKEN, expected=expected,
                          headers={"Idempotency-Key": key} if key else None)


def receipt(directory, stamp):
    with database(directory) as conn:
        return conn.execute("""SELECT state,attempt_count,lease_until,next_attempt_at,event_id
            FROM ingest_events WHERE mode='active' AND json_extract(original_snapshot,'$.original_time')=?""",
                            (str(stamp),)).fetchone()


def slow_faults(binary, directory):
    init(binary, directory)
    recording = "f1000000-0000-4000-a000-000000000081"
    release = threading.Event()
    entered = threading.Event()
    requests = []

    class MBZ(http.server.BaseHTTPRequestHandler):
        def log_message(self, *_):
            pass

        def do_GET(self):
            if recording in self.path:
                requests.append(time.monotonic())
                entered.set()
                release.wait(15)
            data = b'{"title":"Shared validation song","length":180000}'
            self.send_response(200)
            self.end_headers()
            with contextlib.suppress(OSError):
                self.wfile.write(data)

    stub = http.server.ThreadingHTTPServer(("127.0.0.1", 0), MBZ)
    worker = threading.Thread(target=stub.serve_forever, daemon=True)
    worker.start()
    server = None
    try:
        server = boot(binary, directory, True, TUNETRAIL_DISABLE_MUSICBRAINZ="false",
                      TUNETRAIL_MUSICBRAINZ_URL=f"http://127.0.0.1:{stub.server_port}")
        set_flag(server, True)
        stamp = 1790000000
        ack = time.monotonic()
        submit(server, stamp, recording=recording)
        ack = time.monotonic() - ack
        assert entered.wait(12), "worker did not enter loopback MusicBrainz"
        held = receipt(directory, stamp)
        assert held[0:2] == ("received", 1) and held[4] is None
        # Observe a slow request that hits the real client's ten-second timeout.
        resolved = wait_for(lambda: (r := receipt(directory, stamp)) and r[0] == "accepted" and r, timeout=25)
        assert resolved[1] == 1
        slow_seconds = time.monotonic() - requests[0]
        release.set()
        entered.clear()
        release.clear()
        # A different recording guarantees a new lookup and a fresh lease.
        recording = "f2000000-0000-4000-a000-000000000081"
        stamp += 100
        submit(server, stamp, recording=recording, title="Crash before decision")
        assert entered.wait(12), "second receipt did not enter loopback MusicBrainz"
        held = receipt(directory, stamp)
        assert held[0:2] == ("received", 1) and held[4] is None, held
        server.process.kill()
        server.process.wait()
        crashed = time.monotonic()
        remaining_lease = held[2]-int(time.time())
        server.stop()
        release.set()
        server = boot(binary, directory, False)
        set_flag(server, False)
        resumed = wait_for(lambda: (r := receipt(directory, stamp)) and r[0] == "accepted" and r)
        assert resumed[1] == 2 and resumed[4]
        with database(directory) as conn:
            assert conn.execute("SELECT count(*) FROM listens WHERE listened_at IN (?,?)", (stamp-100, stamp)).fetchone() == (2,)
        return {"slow_admission_ms": round(ack*1000, 2), "slow_seconds": round(slow_seconds, 2),
                "slow_attempts": resolved[1], "crash_to_accepted_s": round(time.monotonic()-crashed, 2),
                "lease_at_crash_s": remaining_lease,
                "post_crash_attempts": resumed[1]}
    finally:
        release.set()
        if server:
            server.stop()
        stub.shutdown()
        stub.server_close()


def retry_fault(binary, directory):
    init(binary, directory)
    server = boot(binary, directory, True)
    try:
        set_flag(server, True)
        with database(directory) as conn:
            conn.execute("""CREATE TRIGGER fixture_fail_decision BEFORE INSERT ON ingest_revisions
                BEGIN SELECT RAISE(ABORT,'synthetic persistent decision failure'); END""")
        stamp = 1791000000
        start = time.monotonic()
        submit(server, stamp)
        attempts = []
        for i in range(1, 4):
            failed = wait_for(lambda: (r := receipt(directory, stamp)) and r[0] == "retryable"
                              and r[1] == i and r, timeout=75)
            assert failed[2] is None and failed[4] is None
            attempts.append({"attempt": i, "elapsed_s": round(time.monotonic()-start, 2), "next_epoch": failed[3]})
        with database(directory) as conn:
            assert conn.execute("SELECT count(*) FROM listens").fetchone() == (0,)
            assert conn.execute("SELECT count(*) FROM ingest_revisions").fetchone() == (0,)
            conn.execute("DROP TRIGGER fixture_fail_decision")
        accepted = wait_for(lambda: (r := receipt(directory, stamp)) and r[0] == "accepted" and r, timeout=75)
        assert accepted[1] == 4 and accepted[4]
        return {"failed": attempts, "accepted_attempt": accepted[1], "accepted_elapsed_s": round(time.monotonic()-start, 2)}
    finally:
        server.stop()


@contextlib.contextmanager
def sample_resources(server):
    peaks = {"rss_kib": 0, "data_bytes": 0, "wal_bytes": 0}
    stop = threading.Event()

    def sample():
        while not stop.is_set():
            size = 0
            for path in server.directory.rglob("*"):
                try:
                    if path.is_file() and path.name != "server.log":
                        size += path.stat().st_size
                except FileNotFoundError:
                    pass
            peaks["data_bytes"] = max(peaks["data_bytes"], size)
            wal = server.directory / "tunetrail.db-wal"
            with contextlib.suppress(FileNotFoundError):
                peaks["wal_bytes"] = max(peaks["wal_bytes"], wal.stat().st_size)
            rss = subprocess.run(["ps", "-o", "rss=", "-p", str(server.process.pid)],
                                 capture_output=True, text=True, check=False).stdout.strip()
            if rss.isdigit():
                peaks["rss_kib"] = max(peaks["rss_kib"], int(rss))
            stop.wait(.025)

    sampler = threading.Thread(target=sample)
    sampler.start()
    try:
        yield peaks
    finally:
        stop.set()
        sampler.join()


def percentiles(samples):
    ordered = sorted(samples)
    return {f"p{p}_ms": round(1000 * ordered[math.ceil(p*len(ordered)/100)-1], 2) for p in (50, 95, 99)}


def file_fixtures(base, count):
    for index, form in enumerate(("spotify", "lastfm", "maloja", "koito", "listenbrainz")):
        stamps = [base + index*1000 + 4*n for n in range(count)]
        artist, title, album = f"Fixture {form}", "Track", "Album"
        iso = [datetime.datetime.fromtimestamp(t, datetime.timezone.utc).isoformat() for t in stamps]
        if form == "spotify":
            data = [{"ts": t, "master_metadata_track_name": title, "master_metadata_album_artist_name": artist,
                     "master_metadata_album_album_name": album, "reason_end": "trackdone", "ms_played": 180000} for t in iso]
        elif form == "lastfm":
            data = [{"track": [{"artist": {"#text": artist}, "album": {"#text": album},
                                "name": title, "date": {"uts": str(t)}} for t in stamps]}]
        elif form == "maloja":
            data = {"scrobbles": [{"time": t, "track": {"artists": [artist], "title": title,
                    "album": {"albumtitle": album}}} for t in stamps]}
        elif form == "koito":
            def aliases(value):
                return {"aliases": [{"alias": value, "is_primary": True}]}
            data = {"version": "1", "listens": [{"listened_at": t, "track": aliases(title),
                    "album": aliases(album), "artists": [aliases(artist)]} for t in iso]}
        else:
            new = io.BytesIO()
            lines = [{"listened_at": t, "track_metadata": {"artist_name": artist,
                      "track_name": title, "release_name": album}} for t in stamps]
            with zipfile.ZipFile(new, "w") as archive:
                archive.writestr("listens/synthetic.jsonl", "\n".join(map(json.dumps, lines)) + "\n")
            yield form, stamps, new.getvalue()
            continue
        yield form, stamps, json.dumps(data).encode()


def observed_events(directory, first):
    with database(directory) as conn:
        return set(conn.execute("SELECT profile_id,listened_at FROM listens WHERE listened_at>=?", (first,)))


def workload(binary, baseline, root):
    source = root / "source"
    init(baseline, source, 30000)
    result = {}
    baseline_projection = None
    first, api_count, file_count = 1780000000, 160, 50
    files = list(file_fixtures(first + 10000, file_count))
    api_stamps = [first + 4*i for i in range(api_count)]
    expected = {(1, t) for t in api_stamps + [t for _, stamps, _ in files for t in stamps]}
    for label, executable, master, active in (("v043", baseline, False, False),
                                                ("v9_master_off", binary, False, False),
                                                ("v9_profile_off", binary, True, False),
                                                ("v9_active", binary, True, True)):
        directory = root / label
        directory.mkdir()
        shutil.copyfile(source / "tunetrail.db", directory / "tunetrail.db")
        server = boot(executable, directory, master)
        try:
            server.login()
            if label == "v9_master_off":
                set_flag(server, True)
            if active:
                set_flag(server, True)
            admissions, completions = {}, {}
            with sample_resources(server) as peaks:
                start = time.monotonic()
                stop = threading.Event()

                def observe():
                    while not stop.is_set():
                        now = time.monotonic()
                        for _, stamp in observed_events(directory, first):
                            completions.setdefault(stamp, now)
                        stop.wait(.02)

                observer = threading.Thread(target=observe)
                observer.start()
                try:
                    def send(stamp):
                        began = time.monotonic()
                        submit(server, stamp, key=f"api-{stamp}")
                        admissions[stamp] = (began, time.monotonic())
                        return admissions[stamp][1] - began

                    with concurrent.futures.ThreadPoolExecutor(max_workers=8) as pool:
                        latencies = list(pool.map(send, api_stamps))
                    api_ack = time.monotonic() - start
                    api_end = wait_for(lambda: all(t in completions for t in api_stamps) and max(completions[t] for t in api_stamps))
                    import_seconds = {}
                    imports_start = time.monotonic()
                    for form, stamps, data in files:
                        began = time.monotonic()
                        server.upload(PROFILE + "/import", form, data)
                        ended = time.monotonic()
                        admissions.update({t: (began, ended) for t in stamps})
                        import_seconds[form] = round(ended-began, 4)
                    import_admission = time.monotonic() - imports_start
                    wait_for(lambda: len(completions) >= len(expected))
                finally:
                    stop.set()
                    observer.join()
                assert observed_events(directory, first) == expected
                with database(directory) as conn:
                    versions = conn.execute("SELECT max(version_id) FROM goose_db_version").fetchone()
                    heard = conn.execute("SELECT count(*) FROM listens WHERE listened_at>=?", (first,)).fetchone()[0]
                    assert heard == len(expected)
                    receipt_count = conn.execute("""SELECT count(*) FROM ingest_events
                        WHERE closed_at IS NOT NULL AND state='accepted'""").fetchone()[0]
                    assert receipt_count == (len(expected) if active else 0)
                    profiles = conn.execute("SELECT count(*) FROM listening_profiles").fetchone()[0]
                    projection = conn.execute("""SELECT l.profile_id,l.listened_at,l.client,t.title,l.origin
                        FROM listens l JOIN tracks_with_title t ON t.id=l.track_id WHERE l.listened_at>=?
                        ORDER BY l.profile_id,l.listened_at,l.client,t.title,l.origin""", (first,)).fetchall()
                    if baseline_projection is None:
                        baseline_projection = projection
                    else:
                        assert projection == baseline_projection
                file_stamps = [t for _, stamps, _ in files for t in stamps]
                result[label] = {
                    "schema": versions[0], "profiles": profiles, "events": heard, "active_accepted": receipt_count,
                    "api_ack_per_s": round(api_count/api_ack, 1), "api_ack": percentiles(latencies),
                    "api_visible_per_s": round(api_count/(api_end-start), 1),
                    "api_to_visible": percentiles([completions[t]-admissions[t][0] for t in api_stamps]),
                    "api_ack_to_visible": percentiles([max(0, completions[t]-admissions[t][1]) for t in api_stamps]),
                    "import_admit_per_s": round(len(file_stamps)/import_admission, 1),
                    "import_visible_per_s": round(len(file_stamps)/(max(completions[t] for t in file_stamps)-imports_start), 1),
                    "import_ack_s": import_seconds,
                    "import_ack_to_visible": percentiles([max(0, completions[t]-admissions[t][1]) for t in file_stamps]),
                    **peaks,
                }
            if active:
                set_flag(server, False)
                off = first + 50000
                submit(server, off)
                assert (1, off) in observed_events(directory, first)
                set_flag(server, True)
                submit(server, off+10)
                wait_for(lambda: (1, off+10) in observed_events(directory, first))
                server.stop()
                server = boot(binary, directory, False)
                submit(server, off+20)
                assert observed_events(directory, first) == expected | {(1, off), (1, off+10), (1, off+20)}
                result[label]["switch_events_verified"] = 3
                server.stop()
                server = boot(binary, directory, True)
                submit(server, off+10)
                def duplicated():
                    with database(directory) as conn:
                        return conn.execute("""SELECT count(*) FROM ingest_events WHERE mode='active'
                            AND state='duplicate' AND json_extract(original_snapshot,'$.original_time')=?
                            AND json_extract(reason_codes,'$[0]')='review_deferred'""", (str(off+10),)).fetchone()[0]

                wait_for(duplicated)
                assert observed_events(directory, first) == expected | {(1, off), (1, off+10), (1, off+20)}
                result[label]["triplet_collision"] = "duplicate_without_extra_listen"
        finally:
            server.stop()
    return result


def timed_request(server, stamp, expected):
    start = time.monotonic()
    data = json.dumps(scrobble(stamp)).encode()
    request = urllib.request.Request(server.origin + SUBMIT, data,
                                     {"Content-Type": "application/json", "Authorization": f"Token {TOKEN}"})
    try:
        response = urllib.request.urlopen(request, timeout=30)
    except urllib.error.HTTPError as error:
        response = error
    with response:
        response.read()
        assert response.status == expected, response.status
        return {"status": response.status, "retry_after": response.headers.get("Retry-After"),
                "elapsed_s": round(time.monotonic()-start, 3)}


def purge(binary, directory):
    init(binary, directory, 30000)
    with database(directory) as conn:
        conn.execute("DELETE FROM ingest_events WHERE profile_id=1")
        conn.execute("""WITH RECURSIVE n(i) AS (VALUES(1) UNION ALL SELECT i+1 FROM n WHERE i<300000)
            INSERT INTO ingest_events(receipt_id,profile_id,source_instance_id,schema_version,source_type,adapter_version,
              batch_id,item_ordinal,operation,received_at,time_precision,time_semantics,original_snapshot,snapshot_digest,digest_version,state,rule_version)
            SELECT printf('purge-%06d',i),1,'listenbrainz:1',1,'listenbrainz','1','bulk',i,'single',1700000000+i,
              'second','unknown',json_object('artist','Synthetic Artist','title',printf('Track %06d',i)),
              printf('%064d',i),'sha256-json-v1','received','shadow-v2' FROM n""")
        conn.execute("""INSERT INTO ingest_observations(id,profile_id,receipt_id,schema_version,snapshot_digest,digest_version,original_snapshot,received_at)
            SELECT 'obs-'||receipt_id,1,receipt_id,1,'variant','sha256-json-v1',original_snapshot,received_at
            FROM ingest_events WHERE profile_id=1 AND item_ordinal%10=0""")
        conn.commit()
        conn.execute("PRAGMA wal_checkpoint(TRUNCATE)")
    server = boot(binary, directory, True)
    try:
        server.login()
        with sample_resources(server) as peaks:
            started = time.monotonic()
            preview = server.request("GET", ACTIVATE, admin=True)
            preview_time = time.monotonic() - started
            assert preview["receipts"] == 300000
            assert sum(row["observations"] for row in preview["groups"]) == 30000
            stop = threading.Event()
            attempts = []

            def writers():
                with database(directory, timeout=.025) as conn:
                    while not stop.is_set():
                        began = time.monotonic()
                        try:
                            conn.execute("BEGIN IMMEDIATE")
                            conn.rollback()
                            ok = True
                        except sqlite3.OperationalError as error:
                            assert error.sqlite_errorcode == sqlite3.SQLITE_BUSY
                            ok = False
                        attempts.append((began, time.monotonic(), ok))
                        stop.wait(.015)

            worker = threading.Thread(target=writers)
            worker.start()
            try:
                began = time.monotonic()
                server.request("PUT", ACTIVATE, {"active": True, "review": False, "digest": preview["digest"]},
                               admin=True, expected=204)
                ended = time.monotonic()
                wait_for(lambda: attempts and attempts[-1][2] and attempts[-1][1] > ended)
            finally:
                stop.set()
                worker.join()
            busy = [a for a in attempts if not a[2]]
            assert busy
            with database(directory) as conn:
                assert conn.execute("SELECT count(*) FROM ingest_events WHERE profile_id=1").fetchone() == (0,)
                assert conn.execute("SELECT count(*) FROM listens").fetchone() == (30000,)
            # Force the production five-second busy timeout while issuing four API writes.
            busy_stamps = [1786000000 + 4*i for i in range(4)]
            with concurrent.futures.ThreadPoolExecutor(max_workers=4) as pool:
                with database(directory) as lock:
                    lock.execute("BEGIN IMMEDIATE")
                    rejected = list(pool.map(lambda t: timed_request(server, t, 503), busy_stamps))
                    assert all(r["retry_after"] == "1" for r in rejected)
            assert not observed_events(directory, min(busy_stamps))
            for stamp in busy_stamps:
                submit(server, stamp)
            wait_for(lambda: observed_events(directory, min(busy_stamps)) == {(1,t) for t in busy_stamps})
        return {"shadow": preview["receipts"], "observations": sum(row["observations"] for row in preview["groups"]),
                "preview_s": round(preview_time, 3), "confirm_s": round(ended-began, 3),
                "blocked_probe_span_s": round(busy[-1][1]-busy[0][0], 3), "writer_busy_probes": len(busy),
                "http_busy": rejected, "retry_events": len(busy_stamps), **peaks}
    finally:
        server.stop()


def replay(binary, directory):
    init(binary, directory)
    server = boot(binary, directory, False)
    try:
        # The profile remains opted in while the master switch is off.
        set_flag(server, True)
        old, gap = 1785000000, 1785000020
        submit(server, old, key="shadow-with-listen")
        with database(directory) as conn:
            conn.execute("""CREATE TRIGGER fixture_gap BEFORE INSERT ON listens
                BEGIN SELECT RAISE(ABORT,'synthetic legacy listen failure after shadow commit'); END""")
        submit(server, gap, key="shadow-without-listen", expected=500)
        with database(directory) as conn:
            conn.execute("DROP TRIGGER fixture_gap")
        server.stop()
        server = boot(binary, directory, True)
        for stamp, key in ((old, "shadow-with-listen"), (gap, "shadow-without-listen")):
            submit(server, stamp, key=key)
        # Another active receipt closes, proving the worker made progress after these ACKs.
        submit(server, gap+20, key="active-control")
        wait_for(lambda: (1, gap+20) in observed_events(directory, old))
        with database(directory) as conn:
            retained = conn.execute("SELECT count(*) FROM ingest_events WHERE mode='shadow' AND event_id IS NULL").fetchone()[0]
        heard = observed_events(directory, old)
        assert retained == 2 and heard == {(1, old), (1, gap+20)}
        result = {"active_replay_http_200": 2, "retained_shadow_without_link": retained,
                  "legacy_events_for_replayed_keys": 1, "confirmed_without_event": 1, "active_control_events": 1}
        set_flag(server, False)
        # Simulate a generic database error before the inactive profile flag can be read.
        with database(directory) as conn:
            conn.execute("ALTER TABLE profile_ingest_flags RENAME TO fixture_hidden_flags")
        before_flag = timed_request(server, gap+40, 500)
        with database(directory) as conn:
            conn.execute("ALTER TABLE fixture_hidden_flags RENAME TO profile_ingest_flags")
        assert (1, gap+40) not in observed_events(directory, old)
        submit(server, gap+40)
        assert (1, gap+40) in observed_events(directory, old)
        result["unknown_mode_generic_error"] = before_flag
        return result
    finally:
        server.stop()


def pinned_shadow_replay(binary, directory):
    init(binary, directory)
    server = boot(binary, directory, True, TUNETRAIL_THROTTLE_IMPORTS_MS="25")
    try:
        server.login()
        form, stamps, data = next(file_fixtures(1787000000, 160))
        with concurrent.futures.ThreadPoolExecutor(max_workers=1) as pool:
            upload = pool.submit(server.upload, PROFILE + "/import", form, data)
            wait_for(lambda: len(observed_events(directory, stamps[0])) >= 10)
            assert not upload.done()
            set_flag(server, True)
            assert not upload.done()
            upload.result()
        with database(directory) as conn:
            seen = conn.execute("""SELECT count(*) FROM ingest_events
                WHERE mode='shadow' AND file_format=?""", (form,)).fetchone()[0]
        assert seen == len(stamps)
        server.upload(PROFILE + "/import", form, data)
        assert observed_events(directory, stamps[0]) == {(1, t) for t in stamps}
        with database(directory) as conn:
            assert conn.execute("SELECT count(*) FROM ingest_events WHERE mode='active'").fetchone()[0] == 0
        return {"shadow_after_purge": seen, "replayed_ordinals": len(stamps),
                "existing_events": len(stamps), "confirmed_without_event": 0}
    finally:
        server.stop()


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--binary", type=Path, required=True)
    parser.add_argument("--baseline", type=Path)
    parser.add_argument("--scenario", choices=("faults", "retry", "slow-crash", "workload", "purge", "replay"), required=True)
    args = parser.parse_args()
    args.binary = args.binary.resolve(strict=True)
    with tempfile.TemporaryDirectory(prefix="tt081-synthetic-") as tmp:
        root = Path(tmp)
        if args.scenario == "faults":
            with concurrent.futures.ThreadPoolExecutor(max_workers=2) as pool:
                runs = {name: pool.submit(fn, args.binary, root / name)
                        for name, fn in (("retry", retry_fault), ("slow_crash", slow_faults))}
                result = {name: task.result() for name, task in runs.items()}
        elif args.scenario == "retry":
            result = retry_fault(args.binary, root / "retry")
        elif args.scenario == "slow-crash":
            result = slow_faults(args.binary, root / "slow")
        elif args.scenario == "workload":
            result = workload(args.binary, args.baseline.resolve(strict=True), root)
        elif args.scenario == "purge":
            result = purge(args.binary, root / "purge")
        else:
            result = {"master_restart": replay(args.binary, root / "replay"),
                      "pinned_import": pinned_shadow_replay(args.binary, root / "pinned")}
        print(json.dumps(result, indent=2), flush=True)


if __name__ == "__main__":
    main()
