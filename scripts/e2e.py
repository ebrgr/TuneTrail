#!/usr/bin/env python3
"""Run TuneTrail's end-to-end journeys against a real binary on disposable data.

Each journey starts its own loopback server(s) with fictional SQLite fixtures and
external services disabled; nothing reads .env or the operational database.
Usage: python3 scripts/e2e.py --binary ./tunetrail [--only NAME ...] [--list]
"""
import argparse
import subprocess
import sys
import tempfile
import time
from pathlib import Path

SCRIPTS = Path(__file__).resolve().parent

# (name, script, needs binary, extra arguments, what the journey covers).
# "{work}" is replaced by a new directory inside the run's temporary root.
JOURNEYS = [
    ("multiuser", "validate_multiuser.py", True, [],
     "fresh and migrated databases: administration, keys, concurrent ingestion, public reads, import/export, revocation, restart"),
    ("admin", "validate_admin_webui.py", True, ["--work-dir", "{work}"],
     "HTML administration: login, CSRF/origin, profiles, keys, relay, import formats, catalog editing and review"),
    ("public", "validate_webui.py", True, ["--work-dir", "{work}"],
     "public Go pages: scoping, API/HTML parity, invalid filters, media, rewind, search, interest bounds"),
    ("permissions", "validate_permissions.py", True, [],
     "authorization, sessions, origin checks and secret handling"),
    ("isolation", "validate_isolation.py", True, [],
     "isolation between profiles for listens, keys, caches and aggregates"),
    ("migration", "validate_migration.py", True, [],
     "v2 to v3 migrations (including WAL), backup restore, route matrix and journeys after migration"),
    ("standalone", "validate_standalone.py", True, [],
     "standalone binary from an empty directory: Go interface, routing and SQLite v3"),
    ("shutdown", "validate_shutdown.py", True, [],
     "graceful shutdown with a stalled relay receiver"),
    ("vscode", "test_vscode_launch.py", False, [],
     "VS Code launch configuration starts the Go server"),
    ("tools", None, False, [],
     "unit tests of the version, issue workflow and fixture scripts"),
]


def run(name, script, needs_binary, extra, binary, root, timeout):
    # Run one journey in its own process so a failure cannot leak servers or state.
    if script is None:
        cmd = [sys.executable, "-m", "unittest", "-q", "test_version", "test_issue", "test_roadmap06_fixture"]
    else:
        cmd = [sys.executable, str(SCRIPTS / script)]
        if needs_binary:
            cmd += ["--binary", str(binary)]
        cmd += [arg.replace("{work}", str(root / name)) for arg in extra]
    started = time.monotonic()
    try:
        result = subprocess.run(cmd, cwd=SCRIPTS, capture_output=True, text=True, timeout=timeout)
        ok, output = result.returncode == 0, result.stdout + result.stderr
    except subprocess.TimeoutExpired as exc:
        ok, output = False, f"timeout after {timeout}s\n{exc.stdout or ''}{exc.stderr or ''}"
    return ok, time.monotonic() - started, output


def main():
    parser = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    parser.add_argument("--binary", type=Path, help="TuneTrail binary built from this checkout")
    parser.add_argument("--only", nargs="+", metavar="NAME", help="run only these journeys")
    parser.add_argument("--list", action="store_true", help="list journeys and exit")
    parser.add_argument("--timeout", type=int, default=300, help="seconds per journey")
    args = parser.parse_args()
    if args.list:
        for name, _, _, _, about in JOURNEYS:
            print(f"{name:12} {about}")
        return
    selected = [j for j in JOURNEYS if not args.only or j[0] in args.only]
    unknown = set(args.only or []) - {j[0] for j in JOURNEYS}
    if unknown:
        sys.exit(f"e2e: unknown journey(s): {', '.join(sorted(unknown))}")
    if any(j[2] for j in selected):
        if not args.binary:
            sys.exit("e2e: --binary is required (make e2e builds it)")
        binary = args.binary.resolve(strict=True)
    else:
        binary = None
    failures = []
    with tempfile.TemporaryDirectory(prefix="tunetrail-e2e-") as tmp:
        for name, script, needs_binary, extra, _ in selected:
            ok, seconds, output = run(name, script, needs_binary, extra, binary, Path(tmp), args.timeout)
            print(f"{'PASS' if ok else 'FAIL'}  {name:12} {seconds:5.1f}s", flush=True)
            if not ok:
                failures.append(name)
                print("\n".join("      " + line for line in output.strip().splitlines()[-25:]), flush=True)
    print(f"e2e: {len(selected) - len(failures)}/{len(selected)} journeys passed")
    if failures:
        sys.exit(1)


if __name__ == "__main__":
    main()
