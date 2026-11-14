#!/usr/bin/env python3
"""Check or increment the TuneTrail 0.x version before committing."""
import argparse
from pathlib import Path
import re
import subprocess

ROOT = Path(__file__).resolve().parent.parent
SOURCE = ROOT / "internal/version/VERSION"
FILES = ["Dockerfile", "deploy/compose.yaml", "deploy/.env.example", "README.md", "DEPLOYMENT.md"]
PATTERN = r"0\.(0|[1-9]\d*)"
LEGACY_PATTERN = r"\d+\.\d+\.\d+-SUM-RC[1-9]\d*"


def version_pattern(value):
    # Match whole versions without touching historical values such as 0.3.2.
    return re.compile(r"(?<![\d.])" + re.escape(value) + r"(?![\d.])")


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--bump", action="store_true", help="Increment the 0.x version and synchronize current files")
    parser.add_argument("--check-parent", action="store_true", help="Require a higher version than the parent commit (publish commits)")
    # CI checks ordinary commits: since ISSUE-060 the version rises once per published task, not per commit.
    parser.add_argument("--check-not-lower", action="store_true", help="Allow the parent's version but reject a lower one (CI)")
    args = parser.parse_args()
    current = SOURCE.read_text().strip()
    match = re.fullmatch(PATTERN, current)
    if not match:
        raise SystemExit("Invalid TuneTrail 0.x version: " + current)
    pattern = version_pattern(current)
    contents = {name: (ROOT / name).read_text() for name in FILES}
    for name, text in contents.items():
        if not pattern.search(text):
            raise SystemExit(f"Version out of sync: {name}")
    if args.check_parent or args.check_not_lower:
        parent = subprocess.run(["git", "show", "HEAD^:internal/version/VERSION"], cwd=ROOT, capture_output=True, text=True)
        if parent.returncode == 0:
            parent_version = parent.stdout.strip()
            previous = re.fullmatch(PATTERN, parent_version)
            if previous and args.check_parent and int(match[1]) <= int(previous[1]):
                raise SystemExit("Increment the version before committing: make version.bump")
            if previous and int(match[1]) < int(previous[1]):
                raise SystemExit(f"Version decreased from {parent_version} to {current}")
            # Allow the first TuneTrail commit to advance beyond the initial 0.3.
            if not previous and not (re.fullmatch(LEGACY_PATTERN, parent_version) and int(match[1]) >= 3):
                raise SystemExit("Unsupported previous version or invalid SUM migration: " + parent_version)
    if args.bump:
        next_version = f"0.{int(match[1]) + 1}"
        for name, text in contents.items():
            (ROOT / name).write_text(pattern.sub(next_version, text))
        # Preserve historical release notes; new releases need their own notes.
        SOURCE.write_text(next_version + "\n")
        print(f"TuneTrail {next_version}: review synchronized files before committing")
    else:
        print(f"TuneTrail {current}: version files synchronized")


if __name__ == "__main__":
    main()
