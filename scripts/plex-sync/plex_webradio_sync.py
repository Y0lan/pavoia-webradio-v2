#!/usr/bin/env python3
"""
Plex → Webradio sync.

Reads creds from env (set via ~/.config/gaende/plex.env, sourced by start.sh):
  PLEX_USERNAME    — required
  PLEX_PASSWORD    — required
  PLEX_SERVERNAME  — optional, default "AEGIR"
  WEBRADIO_FOLDER  — optional, default /home/yolan/files/Webradio
  SERVER_MUSIC     — optional, default /home/yolan/files/plex_music_library/opus/

Writes artifacts atomically via tempfile + os.replace. Emits sync_manifest.json
LAST with sha256 of each top-level artifact so a downstream Go disk importer can
refuse to ingest a partially-written generation.
"""
import contextlib
import fcntl
import hashlib
import json
import os
import pathlib
import subprocess
import sys
import tempfile
import threading
from concurrent.futures import ThreadPoolExecutor, as_completed
from datetime import datetime
from subprocess import call
from time import sleep
from multiprocessing import Pool, cpu_count
from collections import defaultdict
from plexapi.myplex import MyPlexAccount
from tqdm import tqdm

# --- Configuration from env ---
SERVER_MUSIC = os.environ.get('SERVER_MUSIC', '/home/yolan/files/plex_music_library/opus/')
WEBRADIO_FOLDER = os.environ.get('WEBRADIO_FOLDER', '/home/yolan/files/Webradio')
USERNAME = os.environ.get('PLEX_USERNAME')
PASSWORD = os.environ.get('PLEX_PASSWORD')
SERVERNAME = os.environ.get('PLEX_SERVERNAME', 'AEGIR')

if not USERNAME or not PASSWORD:
    sys.stderr.write(
        "ERROR: PLEX_USERNAME and PLEX_PASSWORD must be set.\n"
        "  Put them in ~/.config/gaende/plex.env (sourced by start.sh).\n"
    )
    sys.exit(1)

# --- Run lock: prevent overlapping syncs (Phase C5 hardening, F5.1).
# Two concurrent runs would race on shared global dicts and on the final
# artifact writes, producing a manifest whose sha256 entries describe one
# generation's JSON but a different generation's sidecars.
_LOCK_PATH = os.path.join(WEBRADIO_FOLDER, '.sync.lock')
_LOCK_FH = None
try:
    os.makedirs(WEBRADIO_FOLDER, exist_ok=True)
    _LOCK_FH = open(_LOCK_PATH, 'w')
    fcntl.flock(_LOCK_FH.fileno(), fcntl.LOCK_EX | fcntl.LOCK_NB)
    _LOCK_FH.write(f"{os.getpid()}\n")
    _LOCK_FH.flush()
except BlockingIOError:
    sys.stderr.write(
        f"ERROR: another sync is already running (lock: {_LOCK_PATH}).\n"
        "  Wait for it to finish, or remove the lock file if the process is dead.\n"
    )
    sys.exit(2)
except Exception as e:
    sys.stderr.write(f"WARN: could not acquire run lock ({e}); proceeding without exclusion.\n")

PLEX = MyPlexAccount(USERNAME, PASSWORD).resource(SERVERNAME).connect()

# Plex server URL and token. _TOKEN is kept internally for Plex API calls (via plexapi)
# but MUST NOT be written to disk: the produced JSON artifacts live on a filesystem
# with a public webradio surface, and a leaked Plex token is a reusable Plex auth cred.
# We only persist path fragments (parentThumb, parentArt, …) plus the Plex ratingKey;
# the bridge can re-sign URLs at serve time from its own PLEX_TOKEN env if needed.
PLEX_URL = PLEX._baseurl
_PLEX_TOKEN_SECRET = PLEX._token  # SECRET — do not serialize. See F5.5.

# --- Thread-safety for the shared collection dicts (Phase C5 hardening, F5.3).
# save_metadata_json runs under ThreadPoolExecutor; check-then-act on shared dicts
# produces duplicates, torn sets, and lost playlist associations without a lock.
_SHARED_LOCK = threading.Lock()


# --- Atomic I/O helpers (Phase C5) ---

def atomic_write_json(path, data):
    """Write JSON to path atomically via tempfile + os.replace, with fsync on both
    the file and its containing directory.

    Durability story:
      1. Write to a hidden tempfile in the same directory (so os.replace is atomic
         on the same filesystem).
      2. fsync(file) — data on stable storage.
      3. os.replace(tmp, path) — atomic rename.
      4. fsync(dir) — rename on stable storage; without this, a crash between #3
         and the next directory flush can revert the rename.
    """
    dirname = os.path.dirname(path) or '.'
    fd, tmp_path = tempfile.mkstemp(
        prefix='.' + os.path.basename(path) + '.',
        suffix='.tmp',
        dir=dirname,
    )
    try:
        with os.fdopen(fd, 'w', encoding='utf-8') as f:
            json.dump(data, f, indent=2, ensure_ascii=False)
            f.flush()
            os.fsync(f.fileno())
        os.replace(tmp_path, path)
        # Durability for the rename itself (F5.7).
        try:
            dir_fd = os.open(dirname, os.O_DIRECTORY)
            try:
                os.fsync(dir_fd)
            finally:
                os.close(dir_fd)
        except OSError:
            # Some filesystems (FUSE, certain network mounts) don't support O_DIRECTORY
            # or fsync on dirs. The rename is still atomic — just not crash-proof.
            pass
    except Exception:
        with contextlib.suppress(FileNotFoundError):
            os.remove(tmp_path)
        raise


def file_sha256(path, block_size=65536):
    h = hashlib.sha256()
    with open(path, 'rb') as f:
        for chunk in iter(lambda: f.read(block_size), b''):
            h.update(chunk)
    return h.hexdigest()


def write_sync_manifest(artifact_paths, generation_id, counts):
    """Emit sync_manifest.json LAST, after all artifacts are written.

    Downstream readers verify `artifacts[name].sha256` before trusting a generation.
    A mismatch (or missing manifest) means a partial write — reject the batch.
    """
    artifacts = {}
    for name, path in artifact_paths.items():
        if not os.path.exists(path):
            continue
        artifacts[name] = {
            "path": os.path.relpath(path, WEBRADIO_FOLDER),
            "sha256": file_sha256(path),
            "size_bytes": os.path.getsize(path),
            "mtime": datetime.fromtimestamp(
                os.path.getmtime(path)
            ).strftime('%Y-%m-%d %H:%M:%S'),
        }
    manifest = {
        "generation_id": generation_id,
        "written_at": datetime.now().strftime('%Y-%m-%d %H:%M:%S'),
        "counts": counts,
        "artifacts": artifacts,
    }
    atomic_write_json(
        os.path.join(WEBRADIO_FOLDER, 'sync_manifest.json'), manifest,
    )

# Playlists to ignore (using contains matching)
IGNORE_PLAYLISTS = [
    "All Music",
    "Fresh",
    "Recently Added",
    "Recently Played",
    "Listen",
    "Again",
    "Acapella",
    "Intro"
]

# Global dictionaries to collect all data
ARTISTS_DATA = {}  # artist_key -> artist info with albums and tracks
PLAYLISTS_DATA = {}  # playlist_name -> playlist info with artists and tracks
TRACKS_DATA = {}  # track_key -> track info with playlist associations
ALBUMS_DATA = {}  # album_key -> album info

def get_local_path(song):
    """Extract the actual file path from a Plex media item."""
    try:
        for media in song.media:
            for part in media.parts:
                return part.file
    except Exception as e:
        print(f"Error getting path for {song.title}: {e}")
        return None

def extract_track_metadata(track, playlist_name):
    """Extract simplified metadata from a Plex track object, including playlist association.

    Thread-safety: ThreadPoolExecutor runs this concurrently across hundreds of tracks,
    and the body is full of check-then-act operations against shared dicts. We hold
    _SHARED_LOCK for the entire body — lock overhead is one acquire per track (~7k/run),
    and the serialized section is mostly in-memory work, so throughput cost is negligible.
    Plex I/O (PLEX.fetchItem for artist enrichment) happens under the lock as well —
    accepting that tradeoff to keep correctness obviously correct.
    """
    _SHARED_LOCK.acquire()
    try:
        track_key = track.ratingKey

        # If we've already processed this track, just add the playlist association
        if track_key in TRACKS_DATA:
            if playlist_name not in TRACKS_DATA[track_key]["playlists"]:
                TRACKS_DATA[track_key]["playlists"].append(playlist_name)
            return TRACKS_DATA[track_key]["metadata"]
        
        metadata = {
            "track": {
                "title": track.title,
                "artist": getattr(track, 'originalTitle', None) or getattr(track, 'grandparentTitle', 'Unknown Artist'),
                "album_artist": getattr(track, 'grandparentTitle', 'Unknown Artist'),
                "album": getattr(track, 'parentTitle', 'Unknown Album'),
                "year": getattr(track, 'year', None),
                "duration_ms": getattr(track, 'duration', None),
                "labels": [],
                "genres": [],
                "moods": [],
                "track_number": getattr(track, 'index', None),
                "disc_number": getattr(track, 'parentIndex', None)
            },
            "album": {
                "cover_path": None,
                "art_path": None,
                "rating_key": getattr(track, 'parentRatingKey', None)
            },
            "artist": {
                "name": getattr(track, 'grandparentTitle', 'Unknown Artist'),
                "thumb_path": None,
                "art_path": None,
                "rating_key": getattr(track, 'grandparentRatingKey', None)
            },
            "file": {
                "path": None,
                "original_path": get_local_path(track)
            },
            "metadata": {
                "plex_rating_key": track.ratingKey,
                "plex_guid": track.guid,
                "added_to_webradio": datetime.now().strftime('%Y-%m-%d %H:%M:%S'),
                "updated_at": datetime.now().strftime('%Y-%m-%d %H:%M:%S'),
                "playlists": [playlist_name]  # Track which playlists contain this track
            }
        }

        # Extract labels
        if hasattr(track, 'labels') and track.labels:
            metadata["track"]["labels"] = [label.tag for label in track.labels]

        # Extract genres
        if hasattr(track, 'genres') and track.genres:
            metadata["track"]["genres"] = [genre.tag for genre in track.genres]

        # Extract moods
        if hasattr(track, 'moods') and track.moods:
            metadata["track"]["moods"] = [mood.tag for mood in track.moods]

        # Store only the Plex path fragments (no auth token). A consumer that wants
        # to fetch the image signs its own URL: f"{PLEX_URL}{path}?X-Plex-Token={its-token}".
        # See F5.5 — tokens must never be serialized to disk.
        if hasattr(track, 'parentThumb') and track.parentThumb:
            metadata["album"]["cover_path"] = track.parentThumb
        if hasattr(track, 'parentArt') and track.parentArt:
            metadata["album"]["art_path"] = track.parentArt
        if hasattr(track, 'grandparentThumb') and track.grandparentThumb:
            metadata["artist"]["thumb_path"] = track.grandparentThumb
        if hasattr(track, 'grandparentArt') and track.grandparentArt:
            metadata["artist"]["art_path"] = track.grandparentArt

        # Store track data globally
        TRACKS_DATA[track_key] = {
            "metadata": metadata,
            "playlists": [playlist_name]
        }

        # Collect album data
        album_key = getattr(track, 'parentRatingKey', None)
        if album_key and album_key not in ALBUMS_DATA:
            ALBUMS_DATA[album_key] = {
                "title": getattr(track, 'parentTitle', 'Unknown Album'),
                "artist": getattr(track, 'grandparentTitle', 'Unknown Artist'),
                "artist_key": getattr(track, 'grandparentRatingKey', None),
                "year": getattr(track, 'year', None),
                "cover_path": metadata["album"]["cover_path"],
                "art_path": metadata["album"]["art_path"],
                "rating_key": album_key,
                "tracks": []
            }
        
        # Add track to album
        if album_key:
            track_info = {
                "rating_key": track.ratingKey,
                "title": track.title,
                "track_number": getattr(track, 'index', None),
                "disc_number": getattr(track, 'parentIndex', None),
                "duration_ms": getattr(track, 'duration', None),
                "playlists": [playlist_name]
            }
            # Check if track already exists in album
            existing_track = next((t for t in ALBUMS_DATA[album_key]["tracks"] if t["rating_key"] == track.ratingKey), None)
            if existing_track:
                if playlist_name not in existing_track["playlists"]:
                    existing_track["playlists"].append(playlist_name)
            else:
                ALBUMS_DATA[album_key]["tracks"].append(track_info)

        # Collect artist data with albums and tracks
        artist_key = getattr(track, 'grandparentRatingKey', None)
        if artist_key:
            if artist_key not in ARTISTS_DATA:
                # Fetch the full artist object for more details
                try:
                    artist_obj = PLEX.fetchItem(artist_key)
                    ARTISTS_DATA[artist_key] = {
                        "name": getattr(artist_obj, 'title', metadata["artist"]["name"]),
                        "bio": getattr(artist_obj, 'summary', None),
                        "thumb_path": metadata["artist"]["thumb_path"],
                        "art_path": metadata["artist"]["art_path"],
                        "rating_key": artist_key,
                        "genres": [genre.tag for genre in getattr(artist_obj, 'genres', [])] if hasattr(artist_obj, 'genres') else [],
                        "moods": [mood.tag for mood in getattr(artist_obj, 'moods', [])] if hasattr(artist_obj, 'moods') else [],
                        "similar": [similar.tag for similar in getattr(artist_obj, 'similar', [])] if hasattr(artist_obj, 'similar') else [],
                        "albums": {},  # album_key -> album info
                        "tracks": {},  # track_key -> track info
                        "playlists": set()  # playlists this artist appears in
                    }
                except:
                    # If we can't fetch full artist, use what we have
                    ARTISTS_DATA[artist_key] = {
                        "name": metadata["artist"]["name"],
                        "bio": None,
                        "thumb_path": metadata["artist"]["thumb_path"],
                        "art_path": metadata["artist"]["art_path"],
                        "rating_key": artist_key,
                        "genres": [],
                        "moods": [],
                        "similar": [],
                        "albums": {},
                        "tracks": {},
                        "playlists": set()
                    }
            
            # Add album to artist if not already there
            if album_key and album_key not in ARTISTS_DATA[artist_key]["albums"]:
                ARTISTS_DATA[artist_key]["albums"][album_key] = {
                    "title": getattr(track, 'parentTitle', 'Unknown Album'),
                    "year": getattr(track, 'year', None),
                    "cover_path": metadata["album"]["cover_path"],
                    "tracks": []
                }
            
            # Add track to artist
            if track.ratingKey not in ARTISTS_DATA[artist_key]["tracks"]:
                ARTISTS_DATA[artist_key]["tracks"][track.ratingKey] = {
                    "title": track.title,
                    "album": getattr(track, 'parentTitle', 'Unknown Album'),
                    "album_key": album_key,
                    "track_number": getattr(track, 'index', None),
                    "duration_ms": getattr(track, 'duration', None),
                    "playlists": [playlist_name]
                }
            else:
                # Track already exists, just add playlist
                if playlist_name not in ARTISTS_DATA[artist_key]["tracks"][track.ratingKey]["playlists"]:
                    ARTISTS_DATA[artist_key]["tracks"][track.ratingKey]["playlists"].append(playlist_name)
            
            # Add track to album within artist
            if album_key and track.ratingKey not in ARTISTS_DATA[artist_key]["albums"][album_key]["tracks"]:
                ARTISTS_DATA[artist_key]["albums"][album_key]["tracks"].append(track.ratingKey)
            
            # Add playlist to artist
            ARTISTS_DATA[artist_key]["playlists"].add(playlist_name)

        return metadata
    except Exception as e:
        print(f"Error extracting metadata for {track.title}: {e}")
        return None
    finally:
        _SHARED_LOCK.release()

def save_metadata_json(metadata, json_path, audio_file_path):
    """Save metadata to JSON file, preserving added_to_webradio date if file exists."""
    try:
        # If JSON already exists, preserve the original added_to_webradio date
        if os.path.exists(json_path):
            try:
                with open(json_path, 'r', encoding='utf-8') as f:
                    existing_data = json.load(f)
                    if 'metadata' in existing_data and 'added_to_webradio' in existing_data['metadata']:
                        metadata['metadata']['added_to_webradio'] = existing_data['metadata']['added_to_webradio']
                    # Merge playlists
                    if 'playlists' in existing_data['metadata']:
                        existing_playlists = existing_data['metadata']['playlists']
                        new_playlists = metadata['metadata'].get('playlists', [])
                        metadata['metadata']['playlists'] = list(set(existing_playlists + new_playlists))
            except:
                pass  # If we can't read existing file, use new data
        elif audio_file_path and os.path.exists(audio_file_path):
            # For new JSON files, use the creation/modification time of the audio file
            file_stat = os.stat(audio_file_path)
            creation_time = file_stat.st_mtime
            metadata['metadata']['added_to_webradio'] = datetime.fromtimestamp(creation_time).strftime('%Y-%m-%d %H:%M:%S')
        
        # Update the updated_at timestamp to current time
        metadata['metadata']['updated_at'] = datetime.now().strftime('%Y-%m-%d %H:%M:%S')

        atomic_write_json(json_path, metadata)
        return True
    except Exception as e:
        print(f"Error saving JSON to {json_path}: {e}")
        return False

def save_global_metadata():
    """Save all global metadata files atomically, then emit sync_manifest.json LAST.

    Ordering matters: each top-level artifact is written atomically (tempfile +
    os.replace), then the manifest is written atomically LAST containing sha256
    of every artifact. A downstream reader that sees a manifest is guaranteed
    the artifacts it points at are complete and consistent with that generation.
    A reader that sees no manifest (or a manifest whose sha256 entries don't
    match the files on disk) must reject the batch.
    """
    try:
        # Convert sets to lists for JSON serialization
        for artist_key in ARTISTS_DATA:
            ARTISTS_DATA[artist_key]["playlists"] = sorted(list(ARTISTS_DATA[artist_key]["playlists"]))

        written_at = datetime.now().strftime('%Y-%m-%d %H:%M:%S')

        # --- artists.json ---
        artists_path = os.path.join(WEBRADIO_FOLDER, 'artists.json')
        artists_list = sorted(
            ARTISTS_DATA.values(), key=lambda x: x.get('name', '').lower()
        )
        atomic_write_json(artists_path, {
            "updated_at": written_at,
            "total_artists": len(artists_list),
            "artists": artists_list,
        })
        print(f"✅ Saved {len(artists_list)} artists to artists.json")

        # --- playlists.json ---
        playlists_path = os.path.join(WEBRADIO_FOLDER, 'playlists.json')
        playlists_list = []
        for playlist_name, playlist_data in PLAYLISTS_DATA.items():
            playlists_list.append({
                "name": playlist_name,
                "track_count": playlist_data["track_count"],
                "artists": sorted(list(playlist_data["artists"])),
                "artist_keys": sorted(list(playlist_data["artist_keys"])),
                "tracks": playlist_data["tracks"],
            })
        atomic_write_json(playlists_path, {
            "updated_at": written_at,
            "total_playlists": len(playlists_list),
            "playlists": playlists_list,
        })
        print(f"✅ Saved {len(playlists_list)} playlists to playlists.json")

        # --- albums.json ---
        albums_path = os.path.join(WEBRADIO_FOLDER, 'albums.json')
        albums_list = sorted(
            ALBUMS_DATA.values(),
            key=lambda x: (x.get('artist', '').lower(), x.get('title', '').lower()),
        )
        atomic_write_json(albums_path, {
            "updated_at": written_at,
            "total_albums": len(albums_list),
            "albums": albums_list,
        })
        print(f"✅ Saved {len(albums_list)} albums to albums.json")

        # --- tracks_index.json ---
        tracks_index_path = os.path.join(WEBRADIO_FOLDER, 'tracks_index.json')
        tracks_index = {}
        for track_key, track_data in TRACKS_DATA.items():
            metadata = track_data["metadata"]
            tracks_index[track_key] = {
                "title": metadata["track"]["title"],
                "artist": metadata["artist"]["name"],
                "album": metadata["track"]["album"],
                "playlists": track_data["playlists"],
            }
        atomic_write_json(tracks_index_path, {
            "updated_at": written_at,
            "total_tracks": len(tracks_index),
            "tracks": tracks_index,
        })
        print(f"✅ Saved {len(tracks_index)} tracks to tracks_index.json")

        # --- sync_manifest.json (LAST) ---
        generation_id = datetime.now().strftime('%Y%m%dT%H%M%S')
        write_sync_manifest(
            {
                "artists": artists_path,
                "playlists": playlists_path,
                "albums": albums_path,
                "tracks_index": tracks_index_path,
            },
            generation_id=generation_id,
            counts={
                "artists": len(artists_list),
                "playlists": len(playlists_list),
                "albums": len(albums_list),
                "tracks": len(tracks_index),
            },
        )
        print(f"📜 Wrote sync_manifest.json (generation {generation_id})")

    except Exception as e:
        # Re-raise so main() exits non-zero; cron picks this up instead of silently
        # printing "Sync complete!" over a half-written generation (F5.4).
        print(f"ERROR saving global metadata files: {e}", file=sys.stderr)
        raise

def should_ignore_playlist(playlist_name):
    """Check if playlist should be ignored based on contains matching."""
    playlist_lower = playlist_name.lower()
    for ignore_term in IGNORE_PLAYLISTS:
        if ignore_term.lower() in playlist_lower:
            return True
    return False

def input_number(message, maximum):
    try:
        choice = int(input(message))
        if choice >= maximum or choice < 0:
            return False
        return choice
    except ValueError:
        print("Error, try again")
        return False

def spacing(x):
    for i in range(x):
        print()

def clear():
    _ = call('clear')

def select_playlist():
    playlists = [p for p in PLEX.playlists() if not should_ignore_playlist(p.title)]

    print("[0] - Download all playlists")
    for number, p in enumerate(playlists, 1):
        print("[{}] - {} ({})".format(number, p.title, p.leafCount))
    choice = None

    while choice is None:
        choice = input_number("choose a playlist -> ", len(playlists) + 1)
        if choice is False:
            continue
        elif choice == 0:
            return "all", playlists
        else:
            return playlists[choice - 1], playlists

def prepare_playlist_folder(playlist_name):
    """Create the playlist folder in Webradio directory."""
    folder_path = os.path.join(WEBRADIO_FOLDER, playlist_name)
    pathlib.Path(folder_path).mkdir(parents=True, exist_ok=True)
    return folder_path

def convert_flac_to_mp3(flac_path, mp3_path):
    """Convert FLAC file to MP3 using ffmpeg."""
    try:
        cmd = [
            'ffmpeg', '-i', flac_path,
            '-acodec', 'mp3', '-b:a', '320k',
            '-y',  # Overwrite output file if it exists
            mp3_path
        ]
        subprocess.run(cmd, check=True, capture_output=True, text=True)
        return True
    except subprocess.CalledProcessError as e:
        print(f"Error converting {flac_path} to MP3: {e}")
        return False
    except FileNotFoundError:
        print("ffmpeg not found. Please install ffmpeg to convert FLAC files.")
        return False

def create_symlink(source_path, link_path):
    """Create a symlink, handling existing links."""
    try:
        if os.path.lexists(link_path):
            if os.path.islink(link_path):
                # Check if existing symlink points to the same source
                if os.readlink(link_path) == source_path:
                    return "exists"
                else:
                    # Remove old symlink and create new one
                    os.remove(link_path)
            else:
                # If it's a regular file, keep it (might be a converted FLAC)
                return "file_exists"

        os.symlink(source_path, link_path)
        return "created"
    except Exception as e:
        print(f"Error creating symlink: {e}")
        return "error"

def process_song(song, dest_folder, playlist_name):
    """Process a single song - create symlink/convert and save metadata JSON."""
    song_path = get_local_path(song)
    if not song_path:
        return f"❌ Could not get path for: {song.title}"

    if not os.path.exists(song_path):
        return f"❌ File not found: {song_path}"

    filename = os.path.basename(song_path)
    file_ext = os.path.splitext(filename)[1].lower()
    dest_path = os.path.join(dest_folder, filename)
    
    # Extract metadata for JSON file (this also updates global collections)
    metadata = extract_track_metadata(song, playlist_name)
    if metadata:
        # Update the file path in metadata to be the destination filename
        metadata["file"]["path"] = filename
    
    audio_result = None
    json_result = None
    audio_file_for_metadata = None

    # Handle MP3 and Opus files - create symlinks
    if file_ext in ['.mp3', '.opus']:
        result = create_symlink(song_path, dest_path)
        if result == "created":
            audio_result = f"✅ Symlinked: {filename}"
        elif result == "exists":
            audio_result = f"⭐️ Already linked: {filename}"
        elif result == "file_exists":
            audio_result = f"📄 File exists: {filename}"
        else:
            audio_result = f"❌ Failed to symlink: {filename}"
        
        # Save JSON metadata for symlinked file
        json_path = f"{dest_path}.json"
        audio_file_for_metadata = dest_path

    # Handle FLAC files - convert to MP3
    elif file_ext == '.flac':
        mp3_filename = os.path.splitext(filename)[0] + '.mp3'
        mp3_path = os.path.join(dest_folder, mp3_filename)

        if os.path.exists(mp3_path):
            audio_result = f"⭐️ MP3 already exists: {mp3_filename}"
        else:
            print(f"Converting FLAC to MP3: {filename}")
            if convert_flac_to_mp3(song_path, mp3_path):
                audio_result = f"✅ Converted to MP3: {mp3_filename}"
            else:
                audio_result = f"❌ Failed to convert: {filename}"
        
        # Update metadata for converted file
        if metadata:
            metadata["file"]["path"] = mp3_filename
        
        # Save JSON metadata for converted file
        json_path = f"{mp3_path}.json"
        audio_file_for_metadata = mp3_path

    else:
        return f"⚠️ Unsupported format ({file_ext}): {filename}"

    # Save metadata JSON
    if metadata:
        if save_metadata_json(metadata, json_path, audio_file_for_metadata):
            json_result = "📝 JSON saved"
        else:
            json_result = "⚠️ JSON failed"
    else:
        json_result = "⚠️ No metadata"

    # Combine results
    return f"{audio_result} | {json_result}"

def process_playlist(playlist):
    """Process all songs in a playlist."""
    if playlist is None:
        print("Error: Received None instead of a playlist")
        return

    playlist_name = playlist.title
    print(f"\n{'='*50}")
    print(f"Processing playlist: {playlist_name}")
    print(f"{'='*50}")

    # Initialize playlist data
    PLAYLISTS_DATA[playlist_name] = {
        "track_count": 0,
        "artists": set(),  # Artist names
        "artist_keys": set(),  # Artist rating keys
        "tracks": []  # Track rating keys
    }

    dest_folder = prepare_playlist_folder(playlist_name)
    songs = playlist.items()

    results = {
        'created': 0,
        'exists': 0,
        'converted': 0,
        'failed': 0,
        'json_saved': 0,
        'json_failed': 0
    }

    with tqdm(total=len(songs), desc=f"Processing {playlist_name}", unit="song") as pbar:
        with ThreadPoolExecutor(max_workers=4) as executor:
            futures = {executor.submit(process_song, song, dest_folder, playlist_name): song for song in songs}

            for future in as_completed(futures):
                song = futures[future]
                try:
                    result = future.result()
                    
                    # Update playlist data
                    PLAYLISTS_DATA[playlist_name]["track_count"] += 1
                    PLAYLISTS_DATA[playlist_name]["tracks"].append(song.ratingKey)
                    
                    artist_name = getattr(song, 'grandparentTitle', 'Unknown Artist')
                    artist_key = getattr(song, 'grandparentRatingKey', None)
                    PLAYLISTS_DATA[playlist_name]["artists"].add(artist_name)
                    if artist_key:
                        PLAYLISTS_DATA[playlist_name]["artist_keys"].add(artist_key)
                    
                    # Parse the combined result
                    if "✅" in result:
                        if "Converted" in result:
                            results['converted'] += 1
                        else:
                            results['created'] += 1
                    elif "⭐️" in result or "📄" in result:
                        results['exists'] += 1
                    elif "❌" in result:
                        results['failed'] += 1
                        print(f"\n{result}")
                    
                    # Count JSON results
                    if "📝 JSON saved" in result:
                        results['json_saved'] += 1
                    elif "JSON failed" in result:
                        results['json_failed'] += 1
                        
                except Exception as e:
                    results['failed'] += 1
                    print(f"\nError processing {song.title}: {e}")
                pbar.update(1)

    print(f"\nPlaylist '{playlist_name}' summary:")
    print(f"  Audio files:")
    print(f"    - New symlinks created: {results['created']}")
    print(f"    - Files converted: {results['converted']}")
    print(f"    - Already existing: {results['exists']}")
    print(f"    - Failed: {results['failed']}")
    print(f"  Metadata:")
    print(f"    - JSON files saved: {results['json_saved']}")
    print(f"    - JSON files failed: {results['json_failed']}")

    return results

def main():
    """Main function to process playlists."""
    print("="*60)
    print("Plex Playlist to Webradio Sync Tool")
    print("With Complete Metadata & Cross-References")
    print("="*60)

    # Get all playlists
    all_playlists = PLEX.playlists()

    # Filter out ignored playlists
    filtered_playlists = [p for p in all_playlists if not should_ignore_playlist(p.title)]

    print(f"\nFound {len(all_playlists)} playlists total")
    print(f"Processing {len(filtered_playlists)} playlists (ignoring {len(all_playlists) - len(filtered_playlists)})")
    print(f"\nPlex Server: {PLEX_URL}")
    print(f"Webradio Folder: {WEBRADIO_FOLDER}")

    playlist_choice, available_playlists = select_playlist()

    if playlist_choice == "all":
        print("\nProcessing all non-ignored playlists...")
        total_results = {
            'playlists': 0,
            'created': 0,
            'exists': 0,
            'converted': 0,
            'failed': 0,
            'json_saved': 0,
            'json_failed': 0
        }

        for playlist in available_playlists:
            results = process_playlist(playlist)
            if results:
                total_results['playlists'] += 1
                total_results['created'] += results['created']
                total_results['exists'] += results['exists']
                total_results['converted'] += results['converted']
                total_results['failed'] += results['failed']
                total_results['json_saved'] += results['json_saved']
                total_results['json_failed'] += results['json_failed']

        print("\n" + "="*60)
        print("FINAL SUMMARY")
        print("="*60)
        print(f"Processed {total_results['playlists']} playlists")
        print(f"Audio files:")
        print(f"  - Total new symlinks: {total_results['created']}")
        print(f"  - Total converted files: {total_results['converted']}")
        print(f"  - Total already existing: {total_results['exists']}")
        print(f"  - Total failed: {total_results['failed']}")
        print(f"Metadata:")
        print(f"  - Total JSON files saved: {total_results['json_saved']}")
        print(f"  - Total JSON files failed: {total_results['json_failed']}")

    elif playlist_choice:
        process_playlist(playlist_choice)
    else:
        print("No playlist selected or an error occurred.")

    # Save all global metadata
    print("\nSaving comprehensive metadata database...")
    save_global_metadata()

    # Print statistics
    print("\n" + "🎉"*20)
    print("Sync complete! Your playlists are ready in:", WEBRADIO_FOLDER)
    print("\nDatabase Statistics:")
    print(f"  - {len(ARTISTS_DATA)} artists with albums and tracks")
    print(f"  - {len(ALBUMS_DATA)} albums")
    print(f"  - {len(TRACKS_DATA)} unique tracks")
    print(f"  - {len(PLAYLISTS_DATA)} playlists")
    print("\nAvailable resources:")
    print("  - artists.json: Complete artist database with albums, tracks, and playlist associations")
    print("  - playlists.json: All playlists with their artists and tracks")
    print("  - albums.json: All albums with track listings")
    print("  - tracks_index.json: Quick lookup index for all tracks")
    print("  - Individual track .json files with full metadata")
    print("\nYour webradio can now provide:")
    print("  - Artist wall with filtering by playlist(s)")
    print("  - Complete discography browsing")
    print("  - Cross-playlist track discovery")
    print("  - Rich metadata for all content!")
    print("🎉"*20)

if __name__ == '__main__':
    try:
        main()
    except KeyboardInterrupt:
        print("\n\nProcess interrupted by user. Exiting...")
        sys.exit(0)
    except Exception as e:
        print(f"\nUnexpected error: {e}")
        sys.exit(1)
