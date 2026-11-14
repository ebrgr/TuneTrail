#!/usr/bin/env python3
"""Verify the standalone Go binary from an empty directory."""
import argparse
import json
import os
import signal
import socket
import sqlite3
import subprocess
import tempfile
import time
import urllib.error

from validate_multiuser import clean_environment
import urllib.request
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]


def request(origin, path, method="GET"):
    """Return an HTTP status, headers and body, including error responses."""
    try:
        response = urllib.request.urlopen(urllib.request.Request(origin + path, method=method), timeout=3)
    except urllib.error.HTTPError as error:
        response = error
    with response:
        return response.status, response.headers, response.read()


def run(binary, root, mode, legacy_env=False):
    """Start one disposable server and inspect its actual HTTP responses."""
    directory = root / mode
    directory.mkdir()
    # Go runs from an empty working directory, so client/build cannot exist there.
    cwd = directory
    with socket.socket() as sock:
        sock.bind(("127.0.0.1", 0))
        port = sock.getsockname()[1]
    origin = f"http://127.0.0.1:{port}"
    env = clean_environment()  # both prefixes, see validate_multiuser.py
    # TUNETRAIL_ names; --legacy-env uses the deprecated KOITO_ names (ISSUE-111).
    prefix = "KOITO_" if legacy_env else "TUNETRAIL_"
    env.update({prefix + k: v for k, v in dict(
        CONFIG_DIR=str(directory), BIND_ADDR="127.0.0.1", LISTEN_PORT=str(port),
        DEFAULT_USERNAME="validation-admin", DEFAULT_PASSWORD="validation-password",
        SKIP_IMPORT="true", DISABLE_MUSICBRAINZ="true", DISABLE_DEEZER="true",
        DISABLE_COVER_ART_ARCHIVE="true", LOG_LEVEL="warn").items()})
    with (directory / "server.log").open("w") as log:
        process = subprocess.Popen([str(binary)], cwd=cwd, env=env, stdout=log, stderr=log)
        try:
            for _ in range(100):
                if process.poll() is not None:
                    raise RuntimeError(f"{mode} server stopped before readiness")
                try:
                    if request(origin, "/apis/web/v1/health")[0] == 200:
                        break
                except OSError:
                    pass
                time.sleep(.1)
            else:
                raise TimeoutError(f"{mode} server not ready")
            config = json.loads(request(origin, "/apis/web/v1/config")[2])
            expected = "go"
            assert config["web_ui"] == expected
            status, _, body = request(origin, "/")
            assert status == 200
            if mode == "default":
                assert b'data-assets="/web-assets"' in body
                for path in ("/theme-helper", "/index.html", "/assets/unknown.js", "/default/unknown"):
                    for method in ("GET", "HEAD"):
                        assert request(origin, path, method)[0] == 404, (path, method)
                for path in ("/default", "/admin", "/favicon.svg", "/web-assets/ui.css", "/apis/web/v1/profiles", "/public/yuu.jpg", "/public/site.webmanifest"):
                    assert request(origin, path)[0] == 200, path
                manifest = json.loads(request(origin, '/public/site.webmanifest')[2])
                assert manifest['start_url'] == '/'
                for icon in manifest['icons']:
                    assert request(origin, icon['src'])[0] == 200, icon['src']
                status, headers, body = request(origin, "/favicon.svg", "HEAD")
                assert status == 200 and not body and "image/svg+xml" in headers.get("Content-Type", "")
            with sqlite3.connect(directory / "tunetrail.db") as conn:
                assert conn.execute("PRAGMA integrity_check").fetchall() == [("ok",)]
                assert conn.execute("PRAGMA foreign_key_check").fetchall() == []
                assert conn.execute("SELECT max(version_id) FROM goose_db_version WHERE is_applied=1").fetchone() == (3,)
            log.flush()
            warned = "is deprecated; rename it to TUNETRAIL_CONFIG_DIR" in (directory / "server.log").read_text(errors="ignore")
            # Deprecated names must warn; current names must not.
            assert warned == legacy_env, f"deprecation warning present={warned}, legacy_env={legacy_env}"
            print(f"PASS {mode}: {expected} mode, routing and SQLite v3, {prefix}* names", flush=True)
        finally:
            if process.poll() is None:
                process.send_signal(signal.SIGINT)
                try:
                    process.wait(timeout=10)
                except subprocess.TimeoutExpired:
                    process.kill()
                    process.wait()
                    raise RuntimeError(f"{mode} server failed to shut down")


def main():
    """Run the standalone server without touching operational data."""
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--binary", type=Path, required=True)
    parser.add_argument("--legacy-env", action="store_true", help="use the deprecated KOITO_ variable names")
    args = parser.parse_args()
    binary = args.binary.resolve(strict=True)
    with tempfile.TemporaryDirectory(prefix="koito-027-") as directory:
        for mode in ("default",):
            run(binary, Path(directory), mode, args.legacy_env)


if __name__ == "__main__":
    main()
