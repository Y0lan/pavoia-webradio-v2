# MPD Multi-Stream Deployment System

A complete automated system for deploying and managing multiple MPD instances for web radio streaming.

## Quick Start

1. **Make scripts executable:**
```bash
chmod +x deploy-mpd-streams.sh
chmod +x mpd-control.sh
chmod +x fix-deploy.sh
chmod +x install.sh
```

2. **Deploy all streams:**
```bash
./deploy-mpd-streams.sh deploy
```

3. **Check status:**
```bash
./mpd-control.sh status
```

## Stream Configuration

| Stream Name | Stream ID | Control Port | Stream Port | Folders |
|-------------|-----------|--------------|-------------|---------|
| gaende's favorites (all genres) | `gaende-favorites` | :6600 | :14000 | ❤️ Tracks |
| Ambiance / Safe Chillroom | `ambiance-safe` | :6601 | :14001 | AMBIANCE |
| ETAGE 0 | `etage-0` | :6602 | :14002 | ETAGE 0, Etage 0 - FAST DARK MINIMAL |
| FONTANNA / LAPUTA | `fontanna-laputa` | :6603 | :14003 | FONTANNA, MINIMAL |
| PALAC-DANCE | `palac-dance` | :6604 | :14004 | PALAC - DANCE |
| PALAC-SLOW/HYPNOTIQUE | `palac-slow-hypno` | :6605 | :14005 | PALAC - SLOW AND HYPNOTIC - POETIC |
| BERMUDA (6PM–00h) | `bermuda-night` | :6606 | :14006 | BERMUDA - AFTER 6 |
| BERMUDA (BEFORE 6PM) / OAZA | `bermuda-day` | :6607 | :14007 | BERMUDA - BEFORE 6 |
| CLOSING | `closing` | :6608 | :14008 | Outro |

## File Structure

```
~/
├── deploy-mpd-streams.sh    # Main deployment script
├── mpd-control.sh           # Control script for managing streams
├── .config/
│   ├── mpd/
│   │   ├── gaende-favorites/
│   │   │   └── mpd.conf
│   │   ├── ambiance-safe/
│   │   │   └── mpd.conf
│   │   └── ... (other stream configs)
│   └── systemd/
│       └── user/
│           └── mpd-streams.service  # Optional systemd service
└── .local/
    └── share/
        └── mpd/
            ├── gaende-favorites/
            │   ├── database
            │   ├── music/     # Symlinks to actual folders
            │   └── playlists/
            └── ... (other stream data)
```

## Usage Examples

### Stream Control

```bash
# Start individual stream
./mpd-control.sh start palac-dance

# Stop individual stream
./mpd-control.sh stop bermuda-night

# Restart stream
./mpd-control.sh restart fontanna-laputa

# Start all streams
./mpd-control.sh start-all

# Stop all streams
./mpd-control.sh stop-all

# Check status of all streams
./mpd-control.sh status

# Validate configuration
./mpd-control.sh validate
```

### Playback Control

```bash
# Skip to next track
./mpd-control.sh next palac-dance

# Pause playback
./mpd-control.sh pause etage-0

# Resume playback
./mpd-control.sh play etage-0

# Show currently playing
./mpd-control.sh current

# Update music database
./mpd-control.sh update ambiance-safe
```

## Accessing Streams

Once deployed, streams are accessible at:

- **gaende's favorites**: `http://YOUR_SERVER_IP:14000`
- **Ambiance**: `http://YOUR_SERVER_IP:14001`
- **ETAGE 0**: `http://YOUR_SERVER_IP:14002`
- **FONTANNA**: `http://YOUR_SERVER_IP:14003`
- **PALAC-DANCE**: `http://YOUR_SERVER_IP:14004`
- **PALAC-SLOW**: `http://YOUR_SERVER_IP:14005`
- **BERMUDA Night**: `http://YOUR_SERVER_IP:14006`
- **BERMUDA Day**: `http://YOUR_SERVER_IP:14007`
- **CLOSING**: `http://YOUR_SERVER_IP:14008`

## Optional: Systemd Service

To run streams automatically at startup:

1. **Copy service file:**
```bash
mkdir -p ~/.config/systemd/user/
cp mpd-streams.service ~/.config/systemd/user/
```

2. **Enable and start service:**
```bash
systemctl --user daemon-reload
systemctl --user enable mpd-streams.service
systemctl --user start mpd-streams.service
```

3. **Check service status:**
```bash
systemctl --user status mpd-streams.service
```

## Advanced Usage

### Deployment Script Commands

```bash
# Deploy all streams
./deploy-mpd-streams.sh deploy

# Stop all streams
./deploy-mpd-streams.sh stop

# Restart all streams
./deploy-mpd-streams.sh restart

# Regenerate documentation only
./deploy-mpd-streams.sh docs
```

### Testing Streams

```bash
# Test if stream is accessible
curl -I http://localhost:14000

# Play stream with mpv
mpv http://localhost:14000

# Play stream with VLC
vlc http://localhost:14000
```

### Monitoring

```bash
# View logs for a specific stream
tail -f ~/.local/share/mpd/palac-dance/log

# Check MPD database update
mpc -p 6604 stats

# See playlist length
mpc -p 6604 playlist | wc -l
```

## Troubleshooting

### Quick Fix for Common Issues
If you encounter any problems, run:
```bash
./fix-deploy.sh
```
This will:
- Remove deprecated configuration parameters
- Create missing database files
- Test IP detection
- Show current system status

### Database errors on first run
- The "Failed to open database" messages are **normal** on first deployment
- MPD creates the database files automatically after starting
- These errors will not appear on subsequent runs

### Stream won't start
- Check if port is already in use: `lsof -i :14000`
- Check logs: `tail ~/.local/share/mpd/STREAM_ID/log`
- Verify music folder exists: `ls ~/files/Webradio/`

### No music playing
- Update database: `./mpd-control.sh update STREAM_ID`
- Check if files are accessible: `ls -la ~/.local/share/mpd/STREAM_ID/music/`
- Verify playlist has songs: `mpc -p PORT playlist`

### Can't connect to stream
- Check firewall settings for ports 14000-14008
- Verify MPD is listening: `netstat -tlnp | grep 14000`
- Test locally first: `curl http://localhost:14000`

## Features

- **Automatic Configuration**: All MPD instances configured automatically
- **Multi-folder Support**: Handles streams that use multiple source folders
- **Port Organization**: Control ports (6600-6608) and stream ports (14000-14008)
- **Status Monitoring**: Real-time status with currently playing tracks
- **Auto-shuffle**: All streams start with shuffle, repeat, and random enabled
- **Documentation Generation**: Auto-generates reference documentation
- **Systemd Integration**: Optional automatic startup on boot
- **Colored Output**: Clear visual feedback for all operations

## Requirements

- MPD (Music Player Daemon) installed at `~/bin/mpd`
- mpc (MPD client) installed
- Music files in `~/files/Webradio/`
- Bash 4.0 or higher
- Basic Unix tools (pgrep, pkill, curl)

## License

This deployment system is provided as-is for personal use.
