package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// StageConfig describes one radio stage.
//
// Playlists is the frozen list of Plex playlist titles that feed this stage.
// The disk importer uses it to decide which files on disk belong to which stage
// (etage-0 aggregates "ETAGE 0" + "Etage 0 - FAST DARK MINIMAL"; fontanna-laputa
// aggregates "FONTANNA" + "MINIMAL"; all others are 1:1). **This mapping is
// frozen** — never add or remove entries per user directive 2026-04-18. Plex
// playlists that aren't in any stage's list (BPM, MIRROR FÉVRIER, MOME, MOME 2,
// TRA, XO) are ignored by the bridge even though Python still syncs them to disk.
type StageConfig struct {
	ID          string   `yaml:"id"`
	Name        string   `yaml:"name"`
	Description string   `yaml:"description"`
	MPDPort     int      `yaml:"mpd_port"`
	StreamPort  int      `yaml:"stream_port"`
	Genre       string   `yaml:"genre"`
	Color       string   `yaml:"color"`
	BPMMin      int      `yaml:"bpm_min"`
	BPMMax      int      `yaml:"bpm_max"`
	Visible     bool     `yaml:"visible"`
	Order       int      `yaml:"order"`
	Playlists   []string `yaml:"playlists" json:"playlists"`
}

type Config struct {
	Host         string
	Port         int
	MPDHost      string
	DatabaseURL  string
	RedisURL     string
	MeiliURL     string
	MeiliKey     string
	PlexURL      string
	PlexToken    string
	LastFMKey    string
	DiscogsToken string
	AdminToken   string
	MusicBasePath string
	Stages       []StageConfig
}

func Load() *Config {
	return &Config{
		Host:          envOr("HOST", "0.0.0.0"),
		Port:          envIntOr("PORT", 3001),
		MPDHost:       envOr("MPD_HOST", "localhost"),
		DatabaseURL:   envOr("DATABASE_URL", "postgres://gaende:gaende@localhost:15432/gaende?sslmode=disable"),
		RedisURL:      envOr("REDIS_URL", "redis://localhost:16379"),
		MeiliURL:      envOr("MEILI_URL", "http://localhost:7700"),
		MeiliKey:      envOr("MEILI_KEY", ""),
		PlexURL:       envOr("PLEX_URL", "https://127.0.0.1:31711"),
		PlexToken:     envOr("PLEX_TOKEN", ""),
		LastFMKey:     envOr("LASTFM_API_KEY", ""),
		DiscogsToken:  envOr("DISCOGS_TOKEN", ""),
		AdminToken:    envOr("ADMIN_TOKEN", ""),
		MusicBasePath: envOr("MUSIC_BASE_PATH", ""),
		Stages:        defaultStages(),
	}
}

func (c *Config) Addr() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}

func (c *Config) StageByID(id string) *StageConfig {
	for i := range c.Stages {
		if c.Stages[i].ID == id {
			return &c.Stages[i]
		}
	}
	return nil
}

func (c *Config) VisibleStages() []StageConfig {
	var out []StageConfig
	for _, s := range c.Stages {
		if s.Visible {
			out = append(out, s)
		}
	}
	return out
}

func defaultStages() []StageConfig {
	raw := envOr("STAGES", "")
	if raw != "" {
		return parseStagesEnv(raw)
	}

	// Default 9 stages matching the existing MPD instances on Whatbox.
	// MPD control ports are 6600-6608, HTTP stream ports are 14000-14008.
	// Playlists names must match Plex exactly (case-insensitive) — they're the
	// frozen stage ↔ Plex mapping per user directive (2026-04-18).
	return []StageConfig{
		{ID: "gaende-favorites", Name: "Main Stage", Description: "Progressive melodic techno, the heart of GAENDE", MPDPort: 6600, StreamPort: 14000, Genre: "Progressive Melodic Techno", Color: "#00ffc8", BPMMin: 118, BPMMax: 128, Visible: true, Order: 1, Playlists: []string{"❤️ Tracks"}},
		{ID: "etage-0", Name: "Techno Bunker", Description: "Raw, industrial, uncompromising", MPDPort: 6601, StreamPort: 14001, Genre: "Techno", Color: "#ff0066", BPMMin: 130, BPMMax: 145, Visible: true, Order: 2, Playlists: []string{"ETAGE 0", "Etage 0 - FAST DARK MINIMAL"}},
		{ID: "ambiance-safe", Name: "Ambient Horizon", Description: "Ambient, downtempo, introspective", MPDPort: 6602, StreamPort: 14002, Genre: "Ambient", Color: "#00ddff", BPMMin: 70, BPMMax: 110, Visible: true, Order: 3, Playlists: []string{"AMBIANCE"}},
		{ID: "palac-dance", Name: "Indie Floor", Description: "Indie dance, nu-disco, groovy", MPDPort: 6603, StreamPort: 14003, Genre: "Indie Dance", Color: "#ffaa00", BPMMin: 110, BPMMax: 125, Visible: true, Order: 4, Playlists: []string{"PALAC - DANCE"}},
		{ID: "fontanna-laputa", Name: "Deep Current", Description: "Deep house, organic, hypnotic", MPDPort: 6604, StreamPort: 14004, Genre: "Deep House", Color: "#7b7bff", BPMMin: 115, BPMMax: 124, Visible: true, Order: 5, Playlists: []string{"FONTANNA", "MINIMAL"}},
		{ID: "palac-slow-hypno", Name: "Chill Terrace", Description: "Lo-fi, chillout, balearic", MPDPort: 6605, StreamPort: 14005, Genre: "Chillout", Color: "#44ddff", BPMMin: 80, BPMMax: 110, Visible: true, Order: 6, Playlists: []string{"PALAC - SLOW AND HYPNOTIC - POETIC"}},
		{ID: "bermuda-night", Name: "Bass Cave", Description: "DnB, breakbeat, UK garage", MPDPort: 6606, StreamPort: 14006, Genre: "DnB", Color: "#ff44ff", BPMMin: 140, BPMMax: 174, Visible: true, Order: 7, Playlists: []string{"BERMUDA - AFTER 6"}},
		{ID: "bermuda-day", Name: "World Frequencies", Description: "Afro house, global bass", MPDPort: 6607, StreamPort: 14007, Genre: "Afro House", Color: "#ff4466", BPMMin: 115, BPMMax: 130, Visible: true, Order: 8, Playlists: []string{"BERMUDA - BEFORE 6"}},
		{ID: "closing", Name: "Live Sets", Description: "Recorded live sets and festival recordings", MPDPort: 6608, StreamPort: 14008, Genre: "Live", Color: "#00ff88", BPMMin: 0, BPMMax: 0, Visible: true, Order: 9, Playlists: []string{"Outro"}},
	}
}

// PlaylistToStage returns a map from each configured Plex playlist title (lowercased,
// for case-insensitive match) to its stage ID. Used by the disk importer to decide
// which files belong to which stage. Playlists that appear in multiple stages (none
// do today) would resolve to whichever iteration wins — log a warn if that changes.
func (c *Config) PlaylistToStage() map[string]string {
	m := make(map[string]string, len(c.Stages)*2)
	for _, s := range c.VisibleStages() {
		for _, p := range s.Playlists {
			m[strings.ToLower(p)] = s.ID
		}
	}
	return m
}

func parseStagesEnv(raw string) []StageConfig {
	// Format: "id:name:port:streamport,id:name:port:streamport,..."
	var stages []StageConfig
	for i, entry := range strings.Split(raw, ",") {
		parts := strings.Split(strings.TrimSpace(entry), ":")
		if len(parts) < 4 {
			continue
		}
		mpdPort, _ := strconv.Atoi(parts[2])
		streamPort, _ := strconv.Atoi(parts[3])
		stages = append(stages, StageConfig{
			ID:         parts[0],
			Name:       parts[1],
			MPDPort:    mpdPort,
			StreamPort: streamPort,
			Visible:    true,
			Order:      i + 1,
		})
	}
	return stages
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func envIntOr(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		n, err := strconv.Atoi(v)
		if err == nil {
			return n
		}
	}
	return fallback
}
