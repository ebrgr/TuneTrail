#!/usr/bin/env python3
"""ISSUE-215 spike: measure MusicBrainz by-name vs ListenBrainz metadata lookup.

Goals (see backlog task ISSUE-215 and design doc-048):
  * Take a sample of listens whose recording_mbid was sent by the client
    (track_metadata.additional_info.recording_mbid, never the one from
    ListenBrainz's own mapper) and query both sources by name, duration and
    album.
  * Measure hit / wrong / ambiguous / absent per source plus per-query time.
  * Sweep MusicBrainz acceptance rules (duration tolerance, score threshold,
    album and artist-credit checks) into precision/coverage tables.
  * Measure the ISRC bridge on a subsample of reference MBIDs.

Safety rules honored here:
  * The ListenBrainz token is read from the token file only; it is never
    embedded, printed or logged. The user name is fetched but never printed.
  * Artist/track/album names and MBIDs never reach stdout or the summary
    files; raw data lives only under /tmp/issue215-raw and is deleted later.
  * MusicBrainz: at most 1 request per second; on 429/503 wait and retry.
  * ListenBrainz: X-RateLimit-* headers respected; on 429/503 wait and retry.

Run: python3 medir_issue_215.py --phase all
"""

from __future__ import annotations

import argparse
import json
import os
import random
import re
import sys
import time
import urllib.parse
import urllib.request
from collections import Counter
from pathlib import Path

USER_AGENT = "TuneTrail/0.76 ( https://github.com/ebrgr/TuneTrail )"
MB_BASE = "https://musicbrainz.org/ws/2"
LB_BASE = "https://api.listenbrainz.org/1"
DEFAULT_TOKEN_FILE = "~/.cache/tunetrail-215/lb_token"
DEFAULT_RAW_DIR = "/tmp/issue215-raw"

# Acceptance-rule sweep grid (duration tolerance seconds, score threshold,
# album check on/off, artist-credit check on/off).
TOLS = [2, 3, 5, 10]
THRS = [90, 95, 100]
FIELD_COMBOS = [(True, True), (True, False), (False, True), (False, False)]


def norm(text: str) -> str:
    """Normalize a name for equality checks: casefold, strip, collapse spaces."""
    return re.sub(r"\s+", " ", (text or "").casefold().strip())


def lucene_escape(text: str) -> str:
    """Escape Lucene special characters for MusicBrainz search phrases."""
    out = []
    for ch in text:
        if ch in '+-&|!(){}[]^"~*?:\\/':
            out.append("\\" + ch)
        else:
            out.append(ch)
    return "".join(out)


def read_token(token_file: str) -> str:
    """Read the ListenBrainz token from the token file (never printed)."""
    return Path(token_file).expanduser().read_text().strip()


def http_get(url: str, headers: dict, timeout: int = 30):
    """Single GET; returns (status, headers, text) or raises on network error."""
    req = urllib.request.Request(url, headers=headers)
    with urllib.request.urlopen(req, timeout=timeout) as resp:
        hdrs = {k: v for k, v in resp.headers.items()}
        return resp.status, hdrs, resp.read().decode("utf-8", "replace")


def get_with_retry(url: str, headers: dict, base_wait: float = 5.0,
                   retries: int = 3):
    """GET with 429/503 handling; returns (status, hdrs, text, elapsed_ms)
    or (None, None, None, None) when all attempts fail."""
    for attempt in range(retries + 1):
        t0 = time.monotonic()
        try:
            status, hdrs, text = http_get(url, headers)
        except Exception:
            elapsed = round((time.monotonic() - t0) * 1000)
            wait = base_wait * (2 ** attempt)
            if attempt >= retries:
                return None, None, None, elapsed
            time.sleep(wait)
            continue
        elapsed = round((time.monotonic() - t0) * 1000)
        if status in (429, 503):
            wait = None
            ra = hdrs.get("Retry-After")
            if ra is not None:
                try:
                    wait = int(ra)
                except ValueError:
                    wait = None
            if wait is None:
                reset = hdrs.get("X-RateLimit-Reset-In")
                if reset is not None:
                    try:
                        wait = int(reset) + 1
                    except ValueError:
                        wait = None
            if wait is None:
                wait = base_wait * (2 ** attempt)
            if attempt >= retries:
                return status, hdrs, text, elapsed
            time.sleep(max(1.0, float(wait)))
            continue
        return status, hdrs, text, elapsed
    return None, None, None, None


def read_jsonl(path: Path):
    """Read a JSONL file; malformed trailing lines are skipped."""
    out = []
    if not path.exists():
        return out
    with path.open("r", encoding="utf-8") as fh:
        for line in fh:
            line = line.strip()
            if not line:
                continue
            try:
                out.append(json.loads(line))
            except json.JSONDecodeError:
                continue
    return out


def append_jsonl(path: Path, rec: dict) -> None:
    with path.open("a", encoding="utf-8") as fh:
        fh.write(json.dumps(rec, ensure_ascii=False) + "\n")


# --------------------------------------------------------------------------
# Phase 1: collect the sample from the ListenBrainz history.
# --------------------------------------------------------------------------

def phase_collect(args) -> None:
    raw = Path(args.raw_dir)
    raw.mkdir(parents=True, exist_ok=True)
    sample_path = raw / "sample.json"
    if sample_path.exists() and not args.redo:
        print("[collect] sample.json already exists; skipping (use --redo to rebuild)")
        return

    token = read_token(args.token_file)
    auth = {"Authorization": "Token " + token, "User-Agent": USER_AGENT}

    status, _hdrs, text, _el = get_with_retry(
        LB_BASE + "/validate-token", auth, retries=2)
    if status != 200:
        print(f"[collect] validate-token failed: status={status}")
        sys.exit(1)
    data = json.loads(text)
    user = data.get("user_name")
    if not user:
        print("[collect] validate-token returned no user_name")
        sys.exit(1)
    print("[collect] token valid (user hidden)")

    combo_refs = {}  # key -> {"mbids": Counter, "nav": bool, "dur": Counter, ...}
    pages = 0
    scanned = 0
    with_mbid = 0
    client_counts = Counter()  # submission_client -> number of listens
    nav_total = 0
    nav_with_ref = 0
    max_ts = None
    stop_reason = "pages exhausted"

    while pages < args.max_pages:
        params = {"count": "100"}
        if max_ts is not None:
            params["max_ts"] = str(max_ts)
        url = (LB_BASE + "/user/" + urllib.parse.quote(user, safe="")
               + "/listens?" + urllib.parse.urlencode(params))
        status, hdrs, text, _el = get_with_retry(url, auth, retries=3)
        if status != 200:
            print(f"[collect] listens page failed: status={status}; stopping")
            stop_reason = f"http {status}"
            break
        payload = json.loads(text).get("payload") or {}
        listens = payload.get("listens") or []
        scanned += len(listens)
        pages += 1
        for ls in listens:
            tm = ls.get("track_metadata") or {}
            ai = tm.get("additional_info") or {}
            # The submission client lives in additional_info, not in
            # track_metadata directly.
            client = ai.get("submission_client") or "(unknown)"
            client_counts[client] += 1
            if client == "navidrome":
                nav_total += 1
            ref = ai.get("recording_mbid")
            if not ref:
                continue
            if client == "navidrome":
                nav_with_ref += 1
            artist = tm.get("artist_name")
            track = tm.get("track_name")
            if not artist or not track:
                continue
            with_mbid += 1
            release = tm.get("release_name") or ""
            key = (norm(artist), norm(track), norm(release))
            entry = combo_refs.setdefault(key, {
                "mbids": Counter(), "nav": False, "durs": Counter(),
                "artist": artist, "track": track, "album": release,
            })
            entry["mbids"][ref] += 1
            if client == "navidrome":
                entry["nav"] = True
            dur = ai.get("duration_ms")
            if dur is not None:
                entry["durs"][dur] += 1
        print(f"[collect] page {pages}: scanned={scanned} combos_with_mbid={len(combo_refs)}")
        if len(listens) < 100:
            stop_reason = "end of history"
            break
        oldest = min(int(ls["listened_at"]) for ls in listens)
        if max_ts is not None and oldest >= max_ts:
            stop_reason = "no older listens"
            break
        max_ts = oldest - 1
        if len(combo_refs) >= 2.5 * args.target:
            stop_reason = f"pool >= 2.5x target ({len(combo_refs)} combos)"
            break

    conflicts = 0
    nav_pool = []
    other_pool = []
    for key, entry in combo_refs.items():
        mbids = entry["mbids"]
        if len(mbids) > 1:
            conflicts += 1
            continue
        ref = next(iter(mbids))  # single reference MBID, guaranteed above
        dur = None
        if entry["durs"]:
            dur = entry["durs"].most_common(1)[0][0]
        item = {
            "artist": entry["artist"], "track": entry["track"],
            "album": entry["album"], "ref": ref, "dur": dur,
            "nav": entry["nav"],
        }
        (nav_pool if entry["nav"] else other_pool).append(item)

    rng = random.Random(args.seed)
    rng.shuffle(nav_pool)
    rng.shuffle(other_pool)
    picked = nav_pool[: args.target] + other_pool[: max(0, args.target - len(nav_pool))]
    items = []
    for i, it in enumerate(picked):
        it["i"] = i
        items.append(it)

    meta = {
        "generated_at": time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime()),
        "seed": args.seed,
        "target": args.target,
        "actual": len(items),
        "pages_scanned": pages,
        "stop_reason": stop_reason,
        "listens_scanned": scanned,
        "listens_with_recording_mbid": with_mbid,
        "listens_by_client": dict(client_counts),
        "navidrome_total": nav_total,
        "navidrome_with_ref": nav_with_ref,
        "combos_total": len(combo_refs),
        "combos_conflict_excluded": conflicts,
        "nav_pool": len(nav_pool),
        "other_pool": len(other_pool),
        "picked_nav": sum(1 for it in items if it["nav"]),
        "picked_other": sum(1 for it in items if not it["nav"]),
        "picked_with_duration": sum(1 for it in items if it["dur"] is not None),
        "picked_with_album": sum(1 for it in items if it["album"]),
    }
    sample_path.write_text(json.dumps({"meta": meta, "items": items},
                                      ensure_ascii=False, indent=1))
    print(f"[collect] done: sample={len(items)} nav={meta['picked_nav']} "
          f"conflicts={conflicts} pages={pages}")


# --------------------------------------------------------------------------
# Phase 2: MusicBrainz by-name queries.
# --------------------------------------------------------------------------

def phase_mb(args) -> None:
    raw = Path(args.raw_dir)
    sample = json.loads((raw / "sample.json").read_text())
    items = sample["items"]
    out_path = raw / "mb_results.jsonl"
    done = {r["idx"] for r in read_jsonl(out_path)}
    last_req = 0.0
    errors = 0
    for idx, it in enumerate(items):
        if idx in done:
            continue
        q = ('recording:"%s" AND artist:"%s"'
             % (lucene_escape(it["track"]), lucene_escape(it["artist"])))
        params = {"query": q, "fmt": "json", "limit": "100"}
        url = MB_BASE + "/recording?" + urllib.parse.urlencode(params)

        now = time.monotonic()
        wait = last_req + 1.0 - now
        if wait > 0:
            time.sleep(wait)
        last_req = time.monotonic()

        status, _hdrs, text, elapsed = get_with_retry(
            url, {"User-Agent": USER_AGENT}, retries=3)
        rec = {"idx": idx, "ts": time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime()),
               "elapsed_ms": elapsed, "status": status, "error": status is None}
        if status is None or status != 200:
            errors += 1
            rec["error"] = True
            rec["n_results"] = 0
            rec["candidates"] = []
        else:
            data = json.loads(text)
            recordings = data.get("recordings") or []
            rec["n_results"] = len(recordings)
            rec["truncated"] = len(recordings) >= 100
            candidates = []
            for r in recordings:
                candidates.append({
                    "id": r.get("id"),
                    "score": r.get("score"),
                    "length": r.get("length"),
                    "credits": [c.get("name") for c in (r.get("artist-credit") or [])],
                    "releases": [rel.get("title") for rel in (r.get("releases") or [])],
                })
            rec["candidates"] = candidates
        append_jsonl(out_path, rec)
        if (idx + 1) % 25 == 0 or idx + 1 == len(items):
            print(f"[mb] {idx + 1}/{len(items)} queried, errors={errors}")


# --------------------------------------------------------------------------
# Phase 3: ListenBrainz metadata lookup.
# --------------------------------------------------------------------------

def phase_lb(args) -> None:
    raw = Path(args.raw_dir)
    sample = json.loads((raw / "sample.json").read_text())
    items = sample["items"]
    out_path = raw / "lb_results.jsonl"
    done = {r["idx"] for r in read_jsonl(out_path)}
    token = read_token(args.token_file)
    headers = {"Authorization": "Token " + token, "User-Agent": USER_AGENT}
    remaining = None
    reset_in = None
    errors = 0
    for idx, it in enumerate(items):
        if idx in done:
            continue
        if remaining is not None and remaining <= 1 and reset_in is not None:
            time.sleep(max(0, int(reset_in)) + 0.5)
        params = {"artist_name": it["artist"], "recording_name": it["track"]}
        if it["album"]:
            params["release_name"] = it["album"]
        url = LB_BASE + "/metadata/lookup/?" + urllib.parse.urlencode(params)
        status, hdrs, text, elapsed = get_with_retry(url, headers, retries=3)
        if hdrs is not None:
            try:
                remaining = int(hdrs.get("X-RateLimit-Remaining"))
            except (TypeError, ValueError):
                remaining = None
            try:
                reset_in = int(hdrs.get("X-RateLimit-Reset-In"))
            except (TypeError, ValueError):
                reset_in = None
        rec = {"idx": idx, "ts": time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime()),
               "elapsed_ms": elapsed, "status": status, "error": False, "mbid": None}
        if status is None:
            rec["error"] = True
            errors += 1
        elif status == 200:
            data = json.loads(text)
            # The lookup returns the fields at the top level of the JSON.
            rm = data.get("recording_metadata") or data
            rec["mbid"] = rm.get("recording_mbid")
        elif status == 404:
            pass  # no match: counted as absent
        else:
            rec["error"] = True
            errors += 1
        append_jsonl(out_path, rec)
        if (idx + 1) % 50 == 0 or idx + 1 == len(items):
            print(f"[lb] {idx + 1}/{len(items)} queried, errors={errors}")


# --------------------------------------------------------------------------
# Phase 4: ISRC bridge on a subsample of reference MBIDs.
# --------------------------------------------------------------------------

def phase_isrc(args) -> None:
    raw = Path(args.raw_dir)
    sample = json.loads((raw / "sample.json").read_text())
    items = sample["items"]
    rng = random.Random(args.seed)
    idxs = sorted(rng.sample(range(len(items)), min(args.isrc_size, len(items))))

    rec_path = raw / "isrc_recs.jsonl"
    map_path = raw / "isrc_map.jsonl"
    done_recs = {r["idx"]: r for r in read_jsonl(rec_path)}
    done_maps = {(r["idx"], r["isrc"]) for r in read_jsonl(map_path)}
    last_req = 0.0
    truncated = 0
    processed = 0

    for idx in idxs:
        rec = done_recs.get(idx)
        if rec is None:
            it = items[idx]
            url = MB_BASE + "/recording/" + urllib.parse.quote(it["ref"]) + "?inc=isrcs&fmt=json"
            now = time.monotonic()
            wait = last_req + 1.0 - now
            if wait > 0:
                time.sleep(wait)
            last_req = time.monotonic()
            status, _hdrs, text, elapsed = get_with_retry(
                url, {"User-Agent": USER_AGENT}, retries=3)
            rec = {"idx": idx, "elapsed_ms": elapsed, "status": status,
                   "error": status is None or status != 200, "isrcs": []}
            if status == 200:
                rec["isrcs"] = (json.loads(text).get("isrcs") or [])
            append_jsonl(rec_path, rec)

        isrcs = rec["isrcs"][:10]
        if len(rec["isrcs"]) > 10:
            truncated += 1
        for isrc in isrcs:
            if (idx, isrc) in done_maps:
                continue
            url = MB_BASE + "/isrc/" + urllib.parse.quote(isrc) + "?fmt=json"
            now = time.monotonic()
            wait = last_req + 1.0 - now
            if wait > 0:
                time.sleep(wait)
            last_req = time.monotonic()
            status, _hdrs, text, elapsed = get_with_retry(
                url, {"User-Agent": USER_AGENT}, retries=3)
            mrec = {"idx": idx, "isrc": isrc, "elapsed_ms": elapsed,
                    "status": status, "error": status is None or status != 200,
                    "recs": []}
            if status == 200:
                data = json.loads(text)
                # The isrc endpoint returns the recordings at the top level.
                recs = data.get("recordings")
                if recs is None:
                    block = (data.get("isrcs") or [{}])[0]
                    recs = block.get("recordings") or []
                mrec["recs"] = [r.get("id") for r in recs]
            append_jsonl(map_path, mrec)
        processed += 1
        print(f"[isrc] recording {idx} done ({len(isrcs)} isrcs), total={processed}")


# --------------------------------------------------------------------------
# Phase 5: summarize (sanitized: counts only, no names, no MBIDs).
# --------------------------------------------------------------------------

def _pct(num: int, den: int):
    return None if den == 0 else round(100.0 * num / den, 1)


def _timing(elapsed_list):
    vals = sorted(elapsed_list)
    if not vals:
        return {"n": 0, "min": None, "median": None, "p95": None, "max": None}
    n = len(vals)
    return {
        "n": n,
        "min": vals[0],
        "median": vals[n // 2] if n % 2 else (vals[n // 2 - 1] + vals[n // 2]) // 2,
        "p95": vals[min(n - 1, int(0.95 * (n - 1)))],
        "max": vals[-1],
    }


def phase_summarize(args) -> None:
    raw = Path(args.raw_dir)
    sample = json.loads((raw / "sample.json").read_text())
    items = sample["items"]
    meta = sample["meta"]
    n = len(items)
    mb_recs = {r["idx"]: r for r in read_jsonl(raw / "mb_results.jsonl")}
    lb_recs = {r["idx"]: r for r in read_jsonl(raw / "lb_results.jsonl")}

    # ---- MusicBrainz rule sweep ----
    grid = []
    for tol in TOLS:
        for thr in THRS:
            for alb, art in FIELD_COMBOS:
                counts = Counter()
                for idx, it in enumerate(items):
                    rec = mb_recs.get(idx)
                    if rec is None or rec.get("error"):
                        counts["error"] += 1
                        continue
                    ref = it["ref"]
                    ref_dur = it.get("dur")
                    album_norm = norm(it["album"]) if it["album"] else None
                    artist_norm = norm(it["artist"])
                    passes = []
                    for c in rec.get("candidates", []):
                        score = c.get("score") or 0
                        if score < thr:
                            continue
                        if ref_dur is not None:
                            clen = c.get("length")
                            if clen is None or abs(clen - ref_dur) > tol * 1000:
                                continue
                        if alb and album_norm is not None and \
                                album_norm not in {norm(t) for t in c["releases"]}:
                            continue
                        if art and artist_norm not in {norm(x) for x in c["credits"]}:
                            continue
                        passes.append(c)
                    ids = sorted({c["id"] for c in passes if c.get("id")})
                    if not ids:
                        counts["absent"] += 1
                    elif len(ids) == 1:
                        counts["hit" if ids[0] == ref else "wrong"] += 1
                    else:
                        counts["ambiguous"] += 1
                        if ref in ids:
                            counts["ambiguous_with_ref"] += 1
                decisive = counts["hit"] + counts["wrong"]
                n_eval = n - counts["error"]
                grid.append({
                    "tol": tol, "thr": thr, "album": alb, "artist": art,
                    "n_eval": n_eval, "errors": counts["error"],
                    "hit": counts["hit"], "wrong": counts["wrong"],
                    "ambiguous": counts["ambiguous"],
                    "ambiguous_with_ref": counts["ambiguous_with_ref"],
                    "absent": counts["absent"],
                    "precision": _pct(counts["hit"], decisive),
                    "coverage": _pct(counts["hit"], n_eval),
                })

    # Proposed rule: best precision among combos with >= 10 decisive answers,
    # ties broken by coverage then by higher threshold.
    viable = [g for g in grid if g["hit"] + g["wrong"] >= 10]
    if viable:
        chosen = max(viable, key=lambda g: (g["precision"], g["coverage"], g["thr"]))
    else:
        chosen = max(grid, key=lambda g: (g["hit"] + g["wrong"], g["precision"] or 0))

    def mb_outcome_for(idx, it, rule):
        rec = mb_recs.get(idx)
        if rec is None or rec.get("error"):
            return "error"
        tol, thr, alb, art = (rule["tol"], rule["thr"],
                              rule["album"], rule["artist"])
        ref_dur = it.get("dur")
        album_norm = norm(it["album"]) if it["album"] else None
        artist_norm = norm(it["artist"])
        ids = []
        for c in rec.get("candidates", []):
            if (c.get("score") or 0) < thr:
                continue
            if ref_dur is not None:
                clen = c.get("length")
                if clen is None or abs(clen - ref_dur) > tol * 1000:
                    continue
            if alb and album_norm is not None and \
                    album_norm not in {norm(t) for t in c["releases"]}:
                continue
            if art and artist_norm not in {norm(x) for x in c["credits"]}:
                continue
            if c.get("id"):
                ids.append(c["id"])
        ids = sorted(set(ids))
        if not ids:
            return "absent"
        if len(ids) > 1:
            return "ambiguous"
        return "hit" if ids[0] == it["ref"] else "wrong"

    mb_counts = Counter()
    lb_counts = Counter()
    cross = Counter()
    mb_top1_hits = 0
    mb_top1_eval = 0
    for idx, it in enumerate(items):
        mo = mb_outcome_for(idx, it, chosen)
        mb_counts[mo] += 1
        lr = lb_recs.get(idx)
        if lr is None or lr.get("error"):
            lo = "error"
        elif lr.get("mbid"):
            lo = "hit" if lr["mbid"] == it["ref"] else "wrong"
        else:
            lo = "absent"
        lb_counts[lo] += 1
        cross[(mo, lo)] += 1
        rec = mb_recs.get(idx)
        if rec is not None and not rec.get("error") and rec.get("candidates"):
            mb_top1_eval += 1
            if rec["candidates"][0].get("id") == it["ref"]:
                mb_top1_hits += 1

    # ---- How many listens actually separate the 2 s from the 5 s tolerance
    # (both at score 95 with album and credits on). Used to show how thin the
    # in-sample rule selection is.
    rule2 = {"tol": 2, "thr": 95, "album": True, "artist": True}
    rule5 = {"tol": 5, "thr": 95, "album": True, "artist": True}
    tol_diff = 0
    wrong2_to_amb5 = 0
    for idx, it in enumerate(items):
        o2 = mb_outcome_for(idx, it, rule2)
        o5 = mb_outcome_for(idx, it, rule5)
        if o2 != o5:
            tol_diff += 1
            if o2 == "wrong" and o5 == "ambiguous":
                wrong2_to_amb5 += 1

    # ---- ISRC bridge ----
    isrc_recs = {r["idx"]: r for r in read_jsonl(raw / "isrc_recs.jsonl")}
    isrc_maps = read_jsonl(raw / "isrc_map.jsonl")
    isrc_by_idx = {}
    for m in isrc_maps:
        isrc_by_idx.setdefault(m["idx"], []).append(m)
    n_sub = len(isrc_recs)
    sub_with_isrc = 0
    sub_isrc_errors = 0
    total_isrcs = 0
    truncated_recs = 0
    lookups_ok = 0
    lookups_err = 0
    isrc_single = 0
    isrc_with_ref = 0
    isrc_ambiguous_with_ref = 0
    sub_unambiguous = 0
    sub_wrong_only = 0
    sub_any_isrc = 0
    for idx, rec in isrc_recs.items():
        it = items[idx]
        ref = it["ref"]
        if rec.get("error"):
            sub_isrc_errors += 1
            continue
        isrcs = rec.get("isrcs") or []
        if len(isrcs) > 10:
            truncated_recs += 1
        if not isrcs:
            continue
        sub_with_isrc += 1
        total_isrcs += len(isrcs)
        found_ref = False
        found_wrong = False
        for m in isrc_by_idx.get(idx, []):
            sub_any_isrc += 1
            if m.get("error"):
                lookups_err += 1
                continue
            lookups_ok += 1
            rec_ids = m.get("recs") or []
            if len(rec_ids) == 1:
                isrc_single += 1
                if rec_ids[0] == ref:
                    found_ref = True
                else:
                    found_wrong = True
            if ref in rec_ids:
                isrc_with_ref += 1
                if len(rec_ids) > 1:
                    isrc_ambiguous_with_ref += 1
        if found_ref:
            sub_unambiguous += 1
        elif found_wrong:
            sub_wrong_only += 1

    # ---- Timing ----
    mb_times = [r["elapsed_ms"] for r in mb_recs.values()
                if not r.get("error") and r.get("elapsed_ms") is not None]
    lb_times = [r["elapsed_ms"] for r in lb_recs.values()
                if not r.get("error") and r.get("elapsed_ms") is not None]
    isrc_times = [m["elapsed_ms"] for m in isrc_maps
                  if not m.get("error") and m.get("elapsed_ms") is not None]

    summary = {
        "meta": meta,
        "sample_n": n,
        "mb": {
            "rule": chosen,
            "counts": dict(mb_counts),
            "top1_hits": mb_top1_hits,
            "top1_eval": mb_top1_eval,
            "timing_ms": _timing(mb_times),
            "grid": grid,
        },
        "lb": {"counts": dict(lb_counts), "timing_ms": _timing(lb_times)},
        "cross": {f"{k[0]}|{k[1]}": v for k, v in cross.items()},
        "tol_2s_vs_5s": {
            "diff_outcomes": tol_diff,
            "wrong2_to_amb5": wrong2_to_amb5,
        },
        "isrc": {
            "subsample_n": n_sub,
            "with_isrc": sub_with_isrc,
            "without_isrc": n_sub - sub_with_isrc - sub_isrc_errors,
            "fetch_errors": sub_isrc_errors,
            "total_isrcs": total_isrcs,
            "truncated_recs": truncated_recs,
            "lookups_ok": lookups_ok,
            "lookups_err": lookups_err,
            "isrc_single_recording": isrc_single,
            "isrc_with_ref": isrc_with_ref,
            "isrc_ambiguous_with_ref": isrc_ambiguous_with_ref,
            "sub_unambiguous": sub_unambiguous,
            "sub_wrong_only": sub_wrong_only,
            "timing_ms": _timing(isrc_times),
        },
    }
    (raw / "summary.json").write_text(
        json.dumps(summary, ensure_ascii=False, indent=1))

    # ---- tables.md (sanitized markdown for the report) ----
    lines = []
    lines.append(f"# Tabelas ISSUE-215 (geradas em {meta['generated_at']})\n")

    lines.append("## Amostra\n")
    lines.append("| Métrica | Valor |")
    lines.append("|---|---|")
    lines.append(f"| Alvo | {meta['target']} |")
    lines.append(f"| Amostra sorteada | {n} |")
    lines.append(f"| Escutas varridas do histórico | {meta['listens_scanned']} |")
    lines.append(f"| Páginas do histórico lidas | {meta['pages_scanned']} ({meta['stop_reason']}) |")
    lines.append(f"| Escutas com recording_mbid do cliente | {meta['listens_with_recording_mbid']} |")
    clients = ", ".join(f"{name} {count}" for name, count in
                         sorted(meta["listens_by_client"].items(), key=lambda kv: -kv[1]))
    lines.append(f"| Escutas por cliente (submission_client) | {clients} |")
    lines.append(f"| Escutas do Navidrome com recording_mbid | {meta['navidrome_with_ref']} de {meta['navidrome_total']} |")
    lines.append(f"| Escutas do Navidrome sem recording_mbid | {meta['navidrome_total'] - meta['navidrome_with_ref']} de {meta['navidrome_total']} |")
    lines.append(f"| Combinações distintas (artista, faixa, álbum) | {meta['combos_total']} |")
    lines.append(f"| Combinações excluídas (MBIDs de referência conflitantes) | {meta['combos_conflict_excluded']} |")
    lines.append(f"| Amostra vinda do Navidrome | {meta['picked_nav']} ({_pct(meta['picked_nav'], n)}%) |")
    lines.append(f"| Amostra com duração conhecida | {meta['picked_with_duration']} ({_pct(meta['picked_with_duration'], n)}%) |")
    lines.append(f"| Amostra com álbum preenchido | {meta['picked_with_album']} ({_pct(meta['picked_with_album'], n)}%) |")
    lines.append("")

    lines.append("## Resultados por fonte (regra proposta para o MusicBrainz)\n")
    lines.append(f"| Fonte | Acertou | Errou | Ambíguo | Ausente | Erro HTTP |")
    lines.append("|---|---|---|---|---|---|")
    mc = mb_counts
    lines.append(f"| MusicBrainz (regra {chosen['tol']}s/{chosen['thr']}/album={chosen['album']}/creditos={chosen['artist']}) "
                 f"| {mc['hit']} ({_pct(mc['hit'], n)}%) | {mc['wrong']} ({_pct(mc['wrong'], n)}%) | "
                 f"{mc['ambiguous']} ({_pct(mc['ambiguous'], n)}%) | {mc['absent']} ({_pct(mc['absent'], n)}%) | "
                 f"{mc['error']} ({_pct(mc['error'], n)}%) |")
    lc = lb_counts
    lines.append(f"| ListenBrainz lookup | {lc['hit']} ({_pct(lc['hit'], n)}%) | {lc['wrong']} ({_pct(lc['wrong'], n)}%) | "
                 f"0 (0%) | {lc['absent']} ({_pct(lc['absent'], n)}%) | {lc['error']} ({_pct(lc['error'], n)}%) |")
    lines.append("")
    lines.append(f"MusicBrainz, sem regra (1º resultado por score): acertos "
                 f"{mb_top1_hits}/{mb_top1_eval} ({_pct(mb_top1_hits, mb_top1_eval)}%).")
    lines.append("")

    lines.append("## Varredura de regras do MusicBrainz (álbum e créditos ligados)\n")
    lines.append("| Tolerância (s) | Limiar | Avaliadas | Decisivas | Acertos | Falsos | "
                 "Ambíguas | Amb. c/ ref | Ausentes | Precisão % | Cobertura % |")
    lines.append("|---|---|---|---|---|---|---|---|---|---|---|")
    for g in grid:
        if g["album"] and g["artist"]:
            lines.append(f"| {g['tol']} | {g['thr']} | {g['n_eval']} | "
                         f"{g['hit'] + g['wrong']} | {g['hit']} | {g['wrong']} | "
                         f"{g['ambiguous']} | {g['ambiguous_with_ref']} | {g['absent']} | "
                         f"{g['precision']} | {g['coverage']} |")
    lines.append("")

    lines.append("## Diferença entre tolerâncias 2 s e 5 s (limiar 95, álbum e créditos ligados)\n")
    td = summary["tol_2s_vs_5s"]
    lines.append(f"| Escutas com desfecho diferente entre as duas tolerâncias | {td['diff_outcomes']} |")
    lines.append(f"| Destas, as que viram ambíguas em vez de falso positivo | {td['wrong2_to_amb5']} |")
    lines.append("")

    lines.append("## Efeito de desligar álbum e créditos (na regra proposta)\n")
    lines.append("| Álbum | Créditos | Decisivas | Acertos | Falsos | Ambíguas | "
                 "Ausentes | Precisão % | Cobertura % |")
    lines.append("|---|---|---|---|---|---|---|---|---|")
    for alb, art in FIELD_COMBOS:
        g = next(x for x in grid if x["tol"] == chosen["tol"]
                 and x["thr"] == chosen["thr"] and x["album"] == alb and x["artist"] == art)
        lines.append(f"| {'sim' if alb else 'não'} | {'sim' if art else 'não'} | "
                     f"{g['hit'] + g['wrong']} | {g['hit']} | {g['wrong']} | "
                     f"{g['ambiguous']} | {g['absent']} | {g['precision']} | {g['coverage']} |")
    lines.append("")

    lines.append("## Cruzamento MusicBrainz × ListenBrainz (regra proposta)\n")
    lines.append("| MB \\ LB | Acertou | Errou | Ausente | Erro |")
    lines.append("|---|---|---|---|---|")
    for mo in ["hit", "wrong", "ambiguous", "absent", "error"]:
        row = [cross.get((mo, lo), 0) for lo in ["hit", "wrong", "absent", "error"]]
        lines.append(f"| {mo} | {row[0]} | {row[1]} | {row[2]} | {row[3]} |")
    lines.append("")

    lines.append("## Tempo por consulta (ms, latência da requisição HTTP)\n")
    lines.append("| Fonte | n | Mín | Mediana | p95 | Máx |")
    lines.append("|---|---|---|---|---|---|")
    for name, t in [("MusicBrainz busca", summary["mb"]["timing_ms"]),
                    ("ListenBrainz lookup", summary["lb"]["timing_ms"]),
                    ("MusicBrainz ISRC", summary["isrc"]["timing_ms"])]:
        lines.append(f"| {name} | {t['n']} | {t['min']} | {t['median']} | {t['p95']} | {t['max']} |")
    lines.append("")

    lines.append("## Ponte por ISRC (subamostra)\n")
    lines.append("| Métrica | Valor |")
    lines.append("|---|---|")
    i = summary["isrc"]
    lines.append(f"| Gravações na subamostra | {i['subsample_n']} |")
    lines.append(f"| Com ao menos 1 ISRC | {i['with_isrc']} ({_pct(i['with_isrc'], i['subsample_n'])}%) |")
    lines.append(f"| Sem ISRC | {i['without_isrc']} ({_pct(i['without_isrc'], i['subsample_n'])}%) |")
    lines.append(f"| Erro ao buscar gravação | {i['fetch_errors']} |")
    lines.append(f"| Total de ISRCs | {i['total_isrcs']} |")
    lines.append(f"| Gravações com mais de 10 ISRCs (truncadas) | {i['truncated_recs']} |")
    lines.append(f"| Consultas de ISRC válidas | {i['lookups_ok']} |")
    lines.append(f"| Consultas de ISRC com erro | {i['lookups_err']} |")
    lines.append(f"| ISRCs que levam a uma única gravação | {i['isrc_single_recording']} ({_pct(i['isrc_single_recording'], i['lookups_ok'])}% das válidas) |")
    lines.append(f"| ISRCs cujo destino inclui a gravação de referência | {i['isrc_with_ref']} ({_pct(i['isrc_with_ref'], i['lookups_ok'])}% das válidas) |")
    lines.append(f"| ISRCs ambíguos que incluem a referência | {i['isrc_ambiguous_with_ref']} |")
    lines.append(f"| Gravações que ganham MBID sem ambiguidade via ISRC | {i['sub_unambiguous']} ({_pct(i['sub_unambiguous'], i['with_isrc'])}% das com ISRC) |")
    lines.append(f"| Gravações cujos ISRCs levam só a gravação errada | {i['sub_wrong_only']} ({_pct(i['sub_wrong_only'], i['with_isrc'])}% das com ISRC) |")
    lines.append("")

    (raw / "tables.md").write_text("\n".join(lines), encoding="utf-8")
    print(f"[summarize] wrote summary.json and tables.md; n={n}; "
          f"chosen rule: tol={chosen['tol']}s thr={chosen['thr']} "
          f"album={chosen['album']} artist={chosen['artist']}")


# --------------------------------------------------------------------------

def main() -> None:
    ap = argparse.ArgumentParser(description=__doc__)
    ap.add_argument("--phase", default="all",
                    choices=["all", "collect", "mb", "lb", "isrc", "summarize"])
    ap.add_argument("--target", type=int, default=300)
    ap.add_argument("--isrc-size", type=int, default=100)
    ap.add_argument("--seed", type=int, default=20261126)
    ap.add_argument("--max-pages", type=int, default=300)
    ap.add_argument("--token-file", default=os.environ.get(
        "ISSUE215_LB_TOKEN_FILE", DEFAULT_TOKEN_FILE))
    ap.add_argument("--raw-dir", default=DEFAULT_RAW_DIR)
    ap.add_argument("--redo", action="store_true",
                    help="rebuild the sample even if sample.json exists")
    args = ap.parse_args()

    if args.phase in ("all", "collect"):
        phase_collect(args)
    if args.phase in ("all", "mb"):
        phase_mb(args)
    if args.phase in ("all", "lb"):
        phase_lb(args)
    if args.phase in ("all", "isrc"):
        phase_isrc(args)
    if args.phase in ("all", "summarize"):
        phase_summarize(args)


if __name__ == "__main__":
    main()
