#!/usr/bin/env python3
"""
Backfill BPM / Camelot key / year / genre into existing sidecars from ID3 tags.

Rationale
---------
The Plex sync path only learned to read ID3 tags at commit ~2026-04-19. Every
sidecar written before that has null bpm / camelot_key and (usually) empty
genres. Running the full Plex sync fills them in but takes minutes and
contacts Plex; this script stays local — it walks the existing sidecars,
reads the sibling audio file, normalizes the tags, and rewrites the sidecar
atomically.

Design (codex-reviewed 2026-04-19)
----------------------------------
- Shares tag_extract.py with plex_webradio_sync.py. No second normalization
  implementation to diverge.
- Writes sidecars with the same atomic tempfile + os.replace() discipline
  the main sync uses, so a partial run leaves a coherent state.
- Regenerates sync_manifest.json.sidecars.aggregate_sha256 at the end,
  using the SAME canonical implementation (sidecar_hash.py) the Go importer
  verifies against. Without this the bridge's manifest-hash check would
  reject the whole batch after a backfill run.
- Holds an flock on the manifest path while mutating, so a concurrent Plex
  sync can't interleave and ship a half-old manifest.
- Only TOUCHES sidecars whose current tags differ from what the ID3 extract
  produces, or whose bpm/camelot_key slots are empty. Idempotent: second run
  is a no-op.

Usage
-----
    python3 id3_backfill.py /home/yolan/files/Webradio
    # --dry-run   : don't write
    # --verbose   : per-file log lines
"""
from __future__ import annotations

import argparse
import contextlib
import fcntl
import hashlib
import json
import os
import sys
import tempfile
import time
from pathlib import Path
from typing import Optional

# Import shared helpers — both must resolve relative to this file's dir.
sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from tag_extract import read_audio_tags, mutagen_available  # noqa: E402
from sidecar_hash import sidecar_aggregate_sha256  # noqa: E402


def atomic_write_json(path: str, data) -> None:
    """Tempfile in same dir → fsync → os.replace, matching plex_webradio_sync."""
    tmp_dir = os.path.dirname(path) or "."
    fd, tmp_path = tempfile.mkstemp(prefix=".tmp.", suffix=".json", dir=tmp_dir)
    try:
        with os.fdopen(fd, "w", encoding="utf-8") as f:
            json.dump(data, f, ensure_ascii=False, indent=2)
            f.flush()
            os.fsync(f.fileno())
        os.replace(tmp_path, path)
    except Exception:
        with contextlib.suppress(OSError):
            os.unlink(tmp_path)
        raise


def find_audio_file(sidecar_path: str) -> Optional[str]:
    """Sidecar `<x>.mp3.json` → audio `<x>.mp3` (same pattern for .flac)."""
    for ext in (".mp3.json", ".flac.json"):
        if sidecar_path.endswith(ext):
            return sidecar_path[: -len(ext)] + ext[: -len(".json")]
    return None


def walk_sidecars(root: str):
    for dirpath, _, filenames in os.walk(root):
        for name in filenames:
            if name.endswith(".mp3.json") or name.endswith(".flac.json"):
                yield os.path.join(dirpath, name)


def needs_update(side: dict, tags: dict) -> bool:
    """True if applying `tags` would change anything on `side["track"]`."""
    track = side.get("track") or {}
    if tags.get("bpm") is not None and track.get("bpm") != tags["bpm"]:
        return True
    if tags.get("camelot_key") and track.get("camelot_key") != tags["camelot_key"]:
        return True
    if tags.get("year") is not None and not track.get("year"):
        return True  # Plex didn't set year; fill from ID3
    if tags.get("genre") and not (track.get("genres") or []):
        return True  # Plex didn't set genre; fill from ID3
    return False


def apply_tags(side: dict, tags: dict) -> dict:
    track = side.setdefault("track", {})
    if tags.get("bpm") is not None:
        track["bpm"] = tags["bpm"]
    if tags.get("camelot_key"):
        track["camelot_key"] = tags["camelot_key"]
    if tags.get("year") is not None and not track.get("year"):
        track["year"] = tags["year"]
    if tags.get("genre") and not (track.get("genres") or []):
        track["genres"] = [tags["genre"]]
    # Bump updated_at so a downstream reader can tell a refresh happened.
    meta = side.setdefault("metadata", {})
    meta["updated_at"] = time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime())
    return side


def rewrite_manifest(webradio_folder: str, verbose: bool = False) -> None:
    """Regenerate sync_manifest.json.sidecars.{count,aggregate_sha256}.

    We only touch the sidecars sub-object — the top-level artifact hashes
    (artists.json, etc.) stay untouched so a concurrent Plex sync's write
    of those artifacts isn't stomped. If the manifest doesn't exist yet,
    we refuse (bridge would reject a manifest-less snapshot anyway).
    """
    manifest_path = os.path.join(webradio_folder, "sync_manifest.json")
    if not os.path.exists(manifest_path):
        print(f"no manifest at {manifest_path}; refusing to create one out-of-band")
        return

    # flock the manifest so a concurrent sync doesn't interleave.
    lock_path = manifest_path + ".lock"
    with open(lock_path, "w") as lock_f:
        fcntl.flock(lock_f.fileno(), fcntl.LOCK_EX)
        try:
            with open(manifest_path, "r", encoding="utf-8") as f:
                manifest = json.load(f)
            count, agg = sidecar_aggregate_sha256(webradio_folder)
            manifest.setdefault("sidecars", {})
            manifest["sidecars"]["count"] = count
            manifest["sidecars"]["aggregate_sha256"] = agg
            manifest["written_at"] = time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime())
            atomic_write_json(manifest_path, manifest)
            if verbose:
                print(f"manifest refreshed: sidecars.count={count} agg={agg[:12]}…")
        finally:
            fcntl.flock(lock_f.fileno(), fcntl.LOCK_UN)


def main() -> int:
    ap = argparse.ArgumentParser()
    ap.add_argument("webradio_folder")
    ap.add_argument("--dry-run", action="store_true")
    ap.add_argument("--verbose", action="store_true")
    ap.add_argument("--limit", type=int, default=0,
                    help="Stop after N updates (0 = no limit). Useful for smoke tests.")
    args = ap.parse_args()

    if not mutagen_available():
        print("ERROR: mutagen not installed. pip install mutagen")
        return 2

    root = args.webradio_folder
    if not os.path.isdir(root):
        print(f"ERROR: not a directory: {root}")
        return 2

    scanned = 0
    updated = 0
    skipped_no_audio = 0
    skipped_unreadable = 0
    started = time.time()

    for sidecar_path in walk_sidecars(root):
        scanned += 1
        audio_path = find_audio_file(sidecar_path)
        if not audio_path or not os.path.exists(audio_path):
            skipped_no_audio += 1
            continue

        try:
            with open(sidecar_path, "r", encoding="utf-8") as f:
                side = json.load(f)
        except Exception:
            skipped_unreadable += 1
            continue

        tags = read_audio_tags(audio_path)
        if not any(tags.values()):
            if args.verbose:
                print(f"no tags extracted: {audio_path}")
            continue

        if not needs_update(side, tags):
            continue

        if args.verbose:
            print(f"updating: {sidecar_path}  tags={tags}")

        if not args.dry_run:
            apply_tags(side, tags)
            atomic_write_json(sidecar_path, side)

        updated += 1
        if args.limit and updated >= args.limit:
            print(f"reached --limit={args.limit}, stopping")
            break

    elapsed = time.time() - started
    print(f"scanned={scanned} updated={updated} "
          f"skipped_no_audio={skipped_no_audio} skipped_unreadable={skipped_unreadable} "
          f"elapsed={elapsed:.1f}s")

    if updated > 0 and not args.dry_run:
        rewrite_manifest(root, verbose=args.verbose)

    return 0


if __name__ == "__main__":
    sys.exit(main())
