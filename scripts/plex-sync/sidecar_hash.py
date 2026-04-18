"""Sidecar aggregate hashing — canonical implementation.

Imported by plex_webradio_sync.py AND by the Go cross-language test so there
is exactly one implementation. Kept in its own module (no side effects at
import) so tests can `import sidecar_hash` without needing Plex credentials.
"""
import hashlib
import os


def sidecar_aggregate_sha256(webradio_folder):
    """Walk webradio_folder, hash every *.mp3.json / *.flac.json, and aggregate.

    Algorithm:
      1. Collect every path ending in '.mp3.json' or '.flac.json'.
      2. Relative paths from webradio_folder, sorted ascending.
      3. For each sidecar: write rel_path + b'\\0' + file_sha256_hex + b'\\n'
         into a running sha256. The NUL + LF prevent concatenation ambiguity.
      4. Return (count, hex(running_hash)).

    Go's disk.ComputeSidecarAggregate must replicate this exactly — any
    change here is a cross-language contract change and needs a Go-side
    matching edit.
    """
    entries = []
    for root, _, files in os.walk(webradio_folder):
        for name in files:
            if not (name.endswith('.mp3.json') or name.endswith('.flac.json')):
                continue
            full = os.path.join(root, name)
            rel = os.path.relpath(full, webradio_folder)
            h = hashlib.sha256()
            try:
                with open(full, 'rb') as f:
                    for chunk in iter(lambda: f.read(65536), b''):
                        h.update(chunk)
            except OSError:
                # Vanished between walk and open; skip. Next run reconverges.
                continue
            entries.append((rel, h.hexdigest()))
    entries.sort()
    agg = hashlib.sha256()
    for rel, hexdigest in entries:
        agg.update(rel.encode('utf-8'))
        agg.update(b'\0')
        agg.update(hexdigest.encode('ascii'))
        agg.update(b'\n')
    return len(entries), agg.hexdigest()
