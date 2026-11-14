#!/usr/bin/env python3
"""Start an isolated Go server with newly created fictional SQLite data (for captures)."""

import argparse
import json
import signal
import threading
from pathlib import Path

from roadmap06_fixture import create
from validate_multiuser import Server, WEB


def main():
    """Create fixtures and keep only this local server until interrupted."""
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--binary", type=Path, required=True)
    parser.add_argument("--work-dir", type=Path, required=True, help="new directory; never an existing configuration")
    parser.add_argument("--smoke", action="store_true", help="check startup and stop immediately")
    args = parser.parse_args()
    binary = args.binary.resolve(strict=True)
    root = args.work_dir.absolute()
    create(root)
    servers = []
    stop = threading.Event()
    old_handlers = {}
    try:
        for sig in (signal.SIGINT, signal.SIGTERM):
            old_handlers[sig] = signal.signal(sig, lambda *_: stop.set())
        report = {"schema": 3, "timezone": "America/Sao_Paulo", "fixture_version": 1, "modes": {}}
        # React was removed in ISSUE-030; only the Go interface remains to serve.
        for mode in ("go",):
            server = Server(binary, root / mode, environment={"KOITO_WEB_UI": mode, "TUNETRAIL_FORCE_TZ": "America/Sao_Paulo"})
            servers.append(server)
            server.start()
            if server.request("GET", WEB + "/config")["web_ui"] != mode:
                raise AssertionError("server mode does not match the fixture")
            actual = {profile["slug"]: server.request("GET", WEB + "/profiles/" + profile["slug"] + "/stats?period=all_time")["listen_count"] for profile in server.request("GET", WEB + "/profiles")}
            if actual != {"alpha": 7, "beta": 2, "empty": 0}:
                raise AssertionError(f"unexpected fixture counts: {actual}")
            report["modes"][mode] = {"origin": server.origin, "counts": actual}
        print(json.dumps(report, ensure_ascii=False), flush=True)
        if not args.smoke:
            stop.wait()
    finally:
        for server in reversed(servers):
            server.stop()
        for sig, previous in old_handlers.items():
            signal.signal(sig, previous)


if __name__ == "__main__":
    main()
