#!/usr/bin/env python3
"""Run the private ISSUE-172 rehearsal under a loopback-only process sandbox."""
from collections import Counter
from concurrent.futures import ThreadPoolExecutor
from hashlib import sha256
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from pathlib import Path
import argparse
import contextlib
import json
import math
import os
import secrets
import shlex
import shutil
import sqlite3
import sys
import subprocess
import threading
import time
import urllib.error
import urllib.request

from rehearsal_support import SUBMIT, database, wait_for
from validate_multiuser import Server, WEB, scrobble
from validate_admin_webui import approve_saved_relay

RELAY = "127.0.0.1:4121"
SANDBOX = '''(version 1)
(allow default)
(deny network*)
(allow network-bind (local ip "localhost:*"))
(allow network-inbound (local ip "localhost:*"))
(allow network-outbound (remote ip "localhost:4121"))
'''


def snapshot(directory):
    """Return profile counts and fingerprints of the existing event IDs."""
    with database(directory) as db:
        rows = db.execute("SELECT profile_id,event_id FROM listens ORDER BY 1,2").fetchall()
        profiles = [profile for (profile,) in db.execute("SELECT id FROM listening_profiles ORDER BY id")]
    counts = Counter({profile: 0 for profile in profiles})
    counts.update(profile for profile, _ in rows)
    digests = {profile: sha256(json.dumps([event for p, event in rows if p == profile]).encode()).hexdigest()
               for profile in counts}
    return rows, counts, digests


def prepare(source, directory):
    """Remove inherited credentials and destinations before application startup."""
    directory.mkdir()
    target = directory / "tunetrail.db"
    with contextlib.closing(sqlite3.connect(source.as_uri() + "?mode=ro", uri=True)) as src:
        with contextlib.closing(sqlite3.connect(target)) as dst:
            src.backup(dst)
    password = secrets.token_urlsafe(18)
    htpasswd = subprocess.run(["htpasswd", "-niB", "ensaio-172"], input=password + "\n",
                             text=True, capture_output=True, check=True)
    password_hash = htpasswd.stdout.strip().split(":", 1)[1].encode().replace(b"$2y$", b"$2a$", 1)
    with database(directory) as db:
        db.executescript("""
            PRAGMA foreign_keys=ON;
            PRAGMA secure_delete=ON;
            DELETE FROM profile_relays;
            DELETE FROM relay_allowlist;
            UPDATE legacy_relay_binding SET profile_id=NULL;
            UPDATE integrations SET token_ref=NULL, revoked_at=coalesce(revoked_at,unixepoch());
            DELETE FROM api_keys;
            DELETE FROM account_tokens;
            DELETE FROM sessions;
        """)
        for uid, status in db.execute("SELECT id,status FROM users ORDER BY id").fetchall():
            db.execute("UPDATE users SET username=?,password=? WHERE id=?",
                       (f"ensaio-172-{uid}", password_hash if status == "active" else b"", uid))
        db.execute("""UPDATE users SET username='ensaio-172' WHERE id=(SELECT min(id) FROM users
                      WHERE role='admin' AND status='active')""")
        db.commit()
        db.execute("VACUUM")
    (directory / "login.txt").write_text(f"Usuário: ensaio-172\nSenha: {password}\n")
    return password


class RelayMock(BaseHTTPRequestHandler):
    """Return one rate limit response, then acknowledge deliveries without storing bodies."""
    statuses = Counter()
    events = 0
    lock = threading.Lock()

    def log_message(self, *_):
        pass

    def do_POST(self):
        payload = json.loads(self.rfile.read(int(self.headers["Content-Length"])))
        with self.lock:
            status = 429 if not self.statuses else 200
            self.statuses[status] += 1
            if status == 200:
                type(self).events += len(payload.get("payload", []))
        self.send_response(status)
        if status == 429:
            self.send_header("Retry-After", "12")
        self.end_headers()


def configure(directory):
    """Configure mock drafts and synthetic keys on the migrated copy."""
    with database(directory) as db:
        profiles = db.execute("SELECT id,slug FROM listening_profiles ORDER BY id").fetchall()
        credentials = []
        for index, (profile, _) in enumerate(profiles):
            for client in range(3):
                token = secrets.token_urlsafe(24)
                db.execute("INSERT INTO api_keys(key,profile_id,created_at,label) VALUES(?,?,unixepoch(),?)",
                           (token, profile, f"Ensaio cliente {client + 1}"))
                credentials.append(token)
            if index != 0:
                db.execute("INSERT INTO profile_ingest_review VALUES(?,0)", (profile,))
            db.execute("INSERT INTO profile_relays VALUES(?,?,?)",
                       (profile, "http://" + RELAY, "fixture-relay-only"))
        return profiles, credentials


def post(server, token, stamp, index, *, labeled=True, complete=False):
    """Submit one synthetic listen, recording HTTP status and elapsed time only."""
    payload = scrobble(stamp)
    metadata = payload["payload"][0]["track_metadata"]
    metadata.update(track_name=f"Ensaio 172 Faixa {index % 11}", artist_name="Ensaio 172 Artista",
                    release_name="Ensaio 172 Álbum")
    metadata["additional_info"]["submission_client"] = f"Ensaio cliente {index % 3}"
    if not complete and index % 11 == 1:
        metadata.pop("release_name")
    if not complete and index % 11 == 2:
        metadata.pop("artist_name")
        metadata.pop("release_name")
    headers = {"Content-Type": "application/json", "Authorization": "Token " + token}
    if labeled:
        headers["Idempotency-Key"] = f"issue172-{stamp}"
    request = urllib.request.Request(server.origin + SUBMIT, json.dumps(payload).encode(), headers)
    started = time.monotonic()
    try:
        response = server.public.open(request, timeout=30)
    except urllib.error.HTTPError as error:
        response = error
    with response:
        response.read()
        elapsed = time.monotonic() - started
        return response.status, started + elapsed, elapsed


def observe(directory, profile, first, stop, observations):
    """Sample first appearance in the persisted history every 50 ms."""
    with database(directory) as db:
        while not stop.is_set():
            for (stamp,) in db.execute("SELECT listened_at FROM listens WHERE profile_id=? AND listened_at>=?", (profile, first)):
                observations.setdefault(stamp, time.monotonic())
            stop.wait(0.05)


def p95(values):
    """Return nearest-rank p95 in milliseconds."""
    return round(sorted(values)[math.ceil(len(values) * .95) - 1] * 1000, 2) if values else None


def load(server, directory, credentials, first, total, rate, profile, *, complete=False):
    """Offer identical paced traffic to each mode and reconcile each acknowledged timestamp."""
    stop = threading.Event()
    observations = {}
    observer = threading.Thread(target=observe, args=(directory, profile, first, stop, observations))
    observer.start()
    try:
        with ThreadPoolExecutor(max_workers=12) as pool:
            start = time.monotonic()
            requests = []
            for index in range(total):
                time.sleep(max(0, start + index / rate - time.monotonic()))
                requests.append(pool.submit(post, server, credentials[index % 3], first + index, index, complete=complete))
            responses = [future.result() for future in requests]
        submit_seconds = time.monotonic() - start
        statuses = Counter(status for status, _, _ in responses)
        accepted = {first + index: ack for index, (status, ack, _) in enumerate(responses) if status == 200}
        deadline = time.monotonic() + 30
        while accepted.keys() - observations.keys() and time.monotonic() < deadline:
            time.sleep(.1)
        visible = [max(0, observations[stamp] - ack) for stamp, ack in accepted.items() if stamp in observations]
        with database(directory) as db:
            receipt_keys = db.execute("""SELECT client_idempotency_key,state FROM ingest_events
                                         WHERE profile_id=? AND mode='active'""", (profile,)).fetchall()
        receipt_states = dict(receipt_keys)
        not_visible = accepted.keys() - observations.keys()
        pending = Counter(receipt_states.get(idempotency_digest(stamp), "no_receipt") for stamp in not_visible)
        return dict(statuses=statuses, p95_ack_ms=p95([elapsed for _, _, elapsed in responses]),
                    p95_visible_ms=p95(visible), max_visible_ms=round(max(visible, default=0) * 1000, 2),
                    missing=len(not_visible), missing_states=pending, submitted_per_second=round(total / submit_seconds, 2),
                    seconds=round(time.monotonic() - start, 2))
    finally:
        stop.set()
        observer.join()


def main():
    """Prepare two copies, run parity traffic and keep the active copy available."""
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("binary", type=Path)
    parser.add_argument("source", type=Path)
    parser.add_argument("work", type=Path)
    parser.add_argument("--total", type=int, default=2250)
    parser.add_argument("--keep-preview", action="store_true")
    parser.add_argument("--first", type=int, help="Reuse a synthetic timestamp range for a comparison")
    args = parser.parse_args()
    if args.total < 1:
        parser.error("--total must be positive")
    if args.work.resolve().is_relative_to(Path(__file__).resolve().parents[1]):
        parser.error("work directory must be outside the repository")
    os.umask(0o077)
    args.work.mkdir()
    sandbox = args.work / "relay-only.sb"
    sandbox.write_text(SANDBOX)
    wrapper = args.work / "api"
    # Argument quoting keeps paths with spaces out of shell expansion.
    executable = args.work / "tunetrail"
    shutil.copy2(args.binary, executable)
    wrapper.write_text("#!/bin/sh\nexec /usr/bin/sandbox-exec -f " + shlex.quote(str(sandbox.resolve())) +
                       " " + shlex.quote(str(executable.resolve())) + ' "$@"\n')
    wrapper.chmod(0o700)
    first = args.first if args.first is not None else int(time.time()) - 3 * args.total
    receiver = ThreadingHTTPServer(("127.0.0.1", 4121), RelayMock)
    threading.Thread(target=receiver.serve_forever, daemon=True).start()
    directory = args.work / "occurrence"
    server = Server(wrapper.resolve(), directory.resolve(), port=4120)
    try:
        password = prepare(args.source.resolve(), directory)
        original, count, digest = snapshot(directory)
        server.start()
        server.stop()
        profiles, keys = configure(directory)
        server.start()
        server.login("ensaio-172", password)
        with database(directory) as db:
            assert db.execute("SELECT max(version_id) FROM goose_db_version WHERE is_applied").fetchone() == (21,)
        for profile, _ in profiles:
            approve_saved_relay(server, profile)
        print("occurrence ready", flush=True)
        result = load(server, directory, keys, first, args.total, 15, profiles[0][0])
        rows, counts, after_digest = snapshot(directory)
        projection = project(directory, first)
        baseline_preserved = set(original) <= set(rows)
        print(json.dumps(dict(mode="occurrence", before=count, after=counts, baseline_preserved=baseline_preserved,
                              before_digest=digest, after_digest=after_digest,
                              projection_digest=sha256(json.dumps(projection).encode()).hexdigest(),
                              **result)), flush=True)
        if not baseline_preserved or result["missing"] or result["p95_ack_ms"] > 200:
            raise RuntimeError("rehearsal stop condition")
        edge_cases(server, directory, profiles, keys, first - 30)
        permanent_failure(directory)
        complete_pilot(directory)
        with RelayMock.lock:
            print(json.dumps(dict(relay_responses=dict(RelayMock.statuses), relay_listens=RelayMock.events)), flush=True)
        if args.keep_preview:
            print("Preview ready on 127.0.0.1:4120", flush=True)
            server.process.wait()
    finally:
        server.stop()
        receiver.shutdown()
        receiver.server_close()


def project(directory, first):
    """Project only fields shared by the two writer paths."""
    with database(directory) as db:
        return db.execute("SELECT profile_id,listened_at,client FROM listens WHERE listened_at>=? ORDER BY 1,2,3",
                          (first,)).fetchall()


def edge_cases(server, directory, profiles, keys, stamp):
    """Exercise replay, review flags and duplicate file rows."""
    statuses = []
    for index, (_, slug) in enumerate(profiles):
        token = keys[index*3]
        statuses.extend(post(server, token, stamp, 0)[0] for _ in range(2))
        statuses.append(post(server, token, stamp, 0, labeled=False)[0])
        file_stamp = stamp - 1
        row = {"artist":{"#text":"Ensaio 172 Arquivo"}, "name":"Faixa importada",
               "album":{"#text":"Álbum importado"}, "date":{"uts":str(file_stamp)}}
        server.upload(WEB + f"/profiles/{slug}/import", "lastfm", json.dumps([{"track":[row,row]}]).encode())
    print(json.dumps(dict(edge_statuses=statuses)), flush=True)


def idempotency_digest(stamp):
    """Match the hash the handler stores for a synthetic Idempotency-Key."""
    return sha256(f"issue172-{stamp}".encode()).hexdigest()


def permanent_failure(directory):
    """Force one resolver rollback and advance only this receipt's retry clock."""
    server = Server(None, directory)
    server.origin = "http://127.0.0.1:4120"
    stamp = int(time.time()) - 20_000
    key = idempotency_digest(stamp)
    with database(directory) as db:
        token, profile = db.execute("SELECT key,profile_id FROM api_keys WHERE label='Ensaio cliente 1' ORDER BY profile_id LIMIT 1").fetchone()
        db.execute(f"""CREATE TRIGGER issue172_fail_resolution BEFORE INSERT ON ingest_revisions
                    WHEN EXISTS(SELECT 1 FROM ingest_events WHERE receipt_id=NEW.receipt_id
                                AND client_idempotency_key='{key}' AND profile_id={profile})
                    BEGIN SELECT RAISE(ABORT,'ISSUE-172 synthetic resolver failure'); END""")
    try:
        assert post(server, token, stamp, 10)[0] == 200

        def progress():
            with database(directory) as db:
                return db.execute("""SELECT attempt_count,park_reason FROM ingest_events
                    WHERE client_idempotency_key=? AND profile_id=? AND state='retryable'
                      AND lease_owner IS NULL""", (key, profile)).fetchone()

        for attempt in range(1, 6):
            outcome = wait_for(lambda: (row if (row := progress()) and row[0] == attempt else None), timeout=15)
            if not outcome[1]:
                with database(directory) as db:
                    db.execute("""UPDATE ingest_events SET next_attempt_at=0
                                  WHERE client_idempotency_key=? AND profile_id=?""", (key, profile))
        assert outcome == (5, "retry_exhausted")
        print(json.dumps(dict(fault_ack=200, attempts=outcome[0], park_reason=outcome[1])))
    finally:
        with database(directory) as db:
            db.execute("DROP TRIGGER issue172_fail_resolution")


def complete_pilot(directory):
    """Measure one minute at the pilot rate with sufficient metadata on every request."""
    server = Server(None, directory)
    server.origin = "http://127.0.0.1:4120"
    with database(directory) as db:
        profile = db.execute("SELECT min(id) FROM listening_profiles").fetchone()[0]
        credentials = [key for (key,) in db.execute("""SELECT key FROM api_keys WHERE profile_id=?
            AND label LIKE 'Ensaio cliente %' ORDER BY label""", (profile,))]
    assert len(credentials) == 3
    first = int(time.time()) - 1000
    result = load(server, directory, credentials, first, 900, 15, profile, complete=True)
    print(json.dumps(dict(complete_pilot=True, **result)), flush=True)
    if result["missing"] or result["p95_ack_ms"] > 200:
        raise RuntimeError("pilot stop condition")


if __name__ == "__main__":
    if len(sys.argv) == 3 and sys.argv[1] == "--permanent-failure":
        permanent_failure(Path(sys.argv[2]))
    elif len(sys.argv) == 3 and sys.argv[1] == "--complete-pilot":
        complete_pilot(Path(sys.argv[2]))
    else:
        main()
