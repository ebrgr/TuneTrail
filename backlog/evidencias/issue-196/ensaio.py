"""Reexecuta percursos dos E2E na porta 4130 e emite somente método, caminho e status."""
import argparse
from contextlib import redirect_stdout
from collections import Counter
from pathlib import Path
import re
import socket
import sys
from tempfile import TemporaryDirectory

ROOT = Path(__file__).resolve().parents[3]
sys.path.insert(0, str(ROOT / "scripts"))

from validate_multiuser import Server, validate_journey
from validate_permissions import run_mode
from validate_webui import seed, validate


def fixture(binary, directory, **options):
    environment = options.pop("environment", {})
    with socket.socket() as probe:
        probe.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
        probe.bind(("127.0.0.1", 4130))
    return Server(binary, directory, port=4130,
                  environment={**environment, "TUNETRAIL_LOG_LEVEL": "debug"}, **options)


def with_server(binary, directory, journey):
    server = fixture(binary, directory, environment={"TUNETRAIL_FORCE_TZ": "America/Sao_Paulo"}
                     if directory.name == "public" else {})
    try:
        server.start()
        journey(server)
    finally:
        server.stop()


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--binary", type=Path, required=True)
    args = parser.parse_args()
    binary = args.binary.resolve(strict=True)
    failed = False
    with TemporaryDirectory(prefix="tt196-", dir="/tmp") as work:
        root = Path(work)
        # run_mode constructs its own Server; enforce the same fixture settings.
        run_mode.__globals__["Server"] = fixture
        journeys = {
            "public": lambda: with_server(binary, root / "public", lambda s: validate(s, seed(s))),
            "permissions": lambda: run_mode(binary, root / "permissions", "go"),
            "multiuser": lambda: with_server(binary, root / "multiuser", validate_journey),
        }
        for name, run in journeys.items():
            try:
                with redirect_stdout(sys.stderr):
                    run()
            except Exception as error:
                print(f"{name}: FAIL ({type(error).__name__})", file=sys.stderr)
                failed = True
            else:
                print(f"{name}: PASS", file=sys.stderr)
            logs = (root / name).rglob("server.log")
            hits = Counter(match for log in logs for match in re.findall(
                r"Received (\w+) (/[^\s?]*) - Responded with (\d+)", log.read_text(errors="replace")))
            for (method, path, status), count in sorted(hits.items()):
                print(name, method, path, status, count, sep="\t")
    return int(failed)


if __name__ == "__main__":
    sys.exit(main())
