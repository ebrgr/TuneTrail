"""Shared helpers for rehearsals against disposable local databases."""

import contextlib
import math
import sqlite3
import subprocess
import threading
import time

from validate_multiuser import LBZ

SUBMIT = LBZ + "/submit-listens"


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
