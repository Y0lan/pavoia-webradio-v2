"""
Read audio-file ID3 tags and normalize them into sidecar-friendly fields.

Plex exposes its own metadata (title/artist/album/year/genre) but not the
Mixed-In-Key-style tags we care about for /stats: BPM (TBPM) and Camelot key
(TKEY / INITIALKEY). This module is the single source of the normalization
rules — both plex_webradio_sync.py (per-sync extraction) and the one-off
backfill script share it. Codex round-1 on 2026-04-19 was specific about
this: if two scripts duplicate the rules they will eventually disagree and
we silently corrupt the DB on one path.

Contract:
    read_audio_tags(path) -> {
        "bpm":         int | None,        # 40..250 inclusive, else None
        "camelot_key": str | None,        # canonical form "1A".."12A"/"1B".."12B"
        "genre":       str | None,        # trimmed; Plex takes precedence when present
        "year":        int | None,        # 1900..current+1; else None
    }

Decisions (deliberate, non-heuristic):
  - BPM: reject <40 or >250 outright. Round decimal BPM (e.g. 120.5) to nearest
    int. "120;123"-style multi-value fields: pick the first value that passes
    range, reject the rest.
  - CamelotKey: accept only "NNA" / "NNB" with N in 1..12 after leading-zero
    strip. "Am"/"C"/other musical-key notations are NOT converted — Mixed In
    Key writes INITIALKEY as Camelot when available, so preserving the lie by
    storing a musical key under a camelot_key column would corrupt the chart.
  - Genre: trim whitespace + null-byte artifacts. No taxonomy normalization.
  - Year: pull from TDRC/TYER/date; parse "YYYY-MM-DD" → year-only.

One file with a malformed tag must not crash the whole run. On any parse
failure we return None for that field and keep going.
"""
from __future__ import annotations

import re
from datetime import datetime
from typing import Optional

try:
    from mutagen import File as MutagenFile  # type: ignore
except ImportError as _e:  # pragma: no cover - only hit in dev-without-dep
    MutagenFile = None
    _MUTAGEN_ERR = str(_e)
else:
    _MUTAGEN_ERR = None


_BPM_MIN, _BPM_MAX = 40, 250
_YEAR_MIN = 1900
_YEAR_MAX = datetime.now().year + 1  # allow "next-year release" edge case

# INITIALKEY in Camelot notation: 1-12 followed by A or B, optionally zero-padded.
_CAMELOT_RE = re.compile(r"^\s*0*([1-9]|1[0-2])([AaBb])\s*$")

# Multi-value ID3 frames sometimes join with null bytes or semicolons.
_MULTI_SPLIT = re.compile(r"[\x00;]")


def _normalize_bpm(raw) -> Optional[int]:
    """'120' / '120.5' / '120;123' / 120.0 → first in-range integer or None."""
    if raw is None:
        return None
    if isinstance(raw, (int, float)):
        candidates = [raw]
    else:
        # mutagen frame objects have a .text list on ID3; fall back to str()
        text = getattr(raw, "text", None)
        if text:
            raw = text[0] if isinstance(text, list) and text else str(raw)
        text_str = str(raw).strip()
        if not text_str:
            return None
        candidates = [c for c in _MULTI_SPLIT.split(text_str) if c.strip()]

    for cand in candidates:
        try:
            val = float(cand)
        except (TypeError, ValueError):
            continue
        # Half-up rounding (not Python's default banker's rounding — 120.5
        # should land on 121, not 120, so users reading the chart can trust
        # the bucket boundary matches the label).
        import math
        bpm = math.floor(val + 0.5)
        if _BPM_MIN <= bpm <= _BPM_MAX:
            return bpm
    return None


def _normalize_camelot(raw) -> Optional[str]:
    """'05A' / '5A' / '12b' → canonical '5A'/'12B'. Musical keys → None."""
    if raw is None:
        return None
    text = getattr(raw, "text", None)
    if text:
        raw = text[0] if isinstance(text, list) and text else str(raw)
    m = _CAMELOT_RE.match(str(raw))
    if not m:
        return None
    num, letter = m.groups()
    return f"{int(num)}{letter.upper()}"


def _normalize_year(raw) -> Optional[int]:
    """YYYY / YYYY-MM-DD / TDRC-ish objects → int year or None."""
    if raw is None:
        return None
    text = getattr(raw, "text", None)
    if text:
        raw = text[0] if isinstance(text, list) and text else str(raw)
    s = str(raw).strip()
    # YYYY prefix covers both "2021" and "2021-05-14"
    m = re.match(r"^(\d{4})", s)
    if not m:
        return None
    try:
        year = int(m.group(1))
    except ValueError:
        return None
    if _YEAR_MIN <= year <= _YEAR_MAX:
        return year
    return None


def _normalize_genre(raw) -> Optional[str]:
    """Trim whitespace + null bytes. Reject empty. Keep source casing."""
    if raw is None:
        return None
    text = getattr(raw, "text", None)
    if text:
        raw = text[0] if isinstance(text, list) and text else str(raw)
    s = str(raw).replace("\x00", "").strip()
    return s or None


def read_audio_tags(path: str) -> dict:
    """
    Return {bpm, camelot_key, genre, year}.

    Any unreadable field is None — never raises. Callers merge into the sidecar
    structure as-is; a None means "unknown, don't touch whatever is already
    there" (importer uses COALESCE on UPSERT).
    """
    result = {"bpm": None, "camelot_key": None, "genre": None, "year": None}
    if MutagenFile is None:
        # mutagen not installed; caller decides whether that's fatal.
        return result

    try:
        mf = MutagenFile(path)
    except Exception:
        return result
    if mf is None:
        return result

    tags = getattr(mf, "tags", None)
    if tags is None:
        return result

    # Helper: tolerant dict-like lookup. mutagen's ID3 tag class indexes by
    # frame ID (e.g. "TBPM"), MP4/Vorbis/etc expose a mapping with different
    # key conventions. We probe a short list of plausible keys per field.
    #
    # IMPORTANT: Mixed In Key and many tagging tools write BPM + INITIALKEY as
    # TXXX (user-defined) frames rather than the canonical TBPM/TKEY. mutagen
    # keys those as "TXXX:BPM" / "TXXX:INITIALKEY" — we MUST probe those or
    # we'll silently return None on the exact files we care about most.
    def lookup(keys):
        for k in keys:
            try:
                v = tags.get(k) if hasattr(tags, "get") else tags[k]
            except (KeyError, TypeError):
                v = None
            if v is not None:
                return v
        return None

    bpm_raw = lookup(["TBPM", "TXXX:BPM", "TXXX:bpm", "BPM", "bpm"])
    key_raw = lookup([
        "TKEY", "TXXX:INITIALKEY", "TXXX:initialkey", "TXXX:Initial Key",
        "INITIALKEY", "initialkey", "initial_key",
    ])
    genre_raw = lookup(["TCON", "TXXX:GENRE", "genre", "GENRE"])
    # Prefer TDRC (ID3v2.4 recording date) over TYER (v2.3 year). Some files
    # have neither; "date" is the FLAC/Vorbis equivalent.
    year_raw = lookup([
        "TDRC", "TYER", "TXXX:YEAR", "TXXX:RELEASETIME",
        "date", "DATE", "year", "YEAR",
    ])

    result["bpm"] = _normalize_bpm(bpm_raw)
    result["camelot_key"] = _normalize_camelot(key_raw)
    result["genre"] = _normalize_genre(genre_raw)
    result["year"] = _normalize_year(year_raw)
    return result


def mutagen_available() -> bool:
    return MutagenFile is not None


def mutagen_import_error() -> Optional[str]:
    return _MUTAGEN_ERR
