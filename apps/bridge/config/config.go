package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

type StageConfig struct {
	ID          string `yaml:"id"`
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
	MPDPort     int    `yaml:"mpd_port"`
	StreamPort  int    `yaml:"stream_port"`
	Genre       string `yaml:"genre"`
	Color       string `yaml:"color"`
	BPMMin      int    `yaml:"bpm_min"`
	BPMMax      int    `yaml:"bpm_max"`
	Visible     bool   `yaml:"visible"`
	Order       int    `yaml:"order"`
}

type Config struct {
	Host         string
	Port         int
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

	// Default 9 stages matching the existing MPD instances on Whatbox
	return []StageConfig{
		{ID: "gaende-favorites", Name: "Main Stage", Description: "Progressive melodic techno, the heart of GAENDE", MPDPort: 14000, StreamPort: 18000, Genre: "Progressive Melodic Techno", Color: "#00ffc8", BPMMin: 118, BPMMax: 128, Visible: true, Order: 1},
		{ID: "etage-0", Name: "Techno Bunker", Description: "Raw, industrial, uncompromising", MPDPort: 14001, StreamPort: 18001, Genre: "Techno", Color: "#ff0066", BPMMin: 130, BPMMax: 145, Visible: true, Order: 2},
		{ID: "ambiance-safe", Name: "Ambient Horizon", Description: "Ambient, downtempo, introspective", MPDPort: 14002, StreamPort: 18002, Genre: "Ambient", Color: "#00ddff", BPMMin: 70, BPMMax: 110, Visible: true, Order: 3},
		{ID: "palac-dance", Name: "Indie Floor", Description: "Indie dance, nu-disco, groovy", MPDPort: 14003, StreamPort: 18003, Genre: "Indie Dance", Color: "#ffaa00", BPMMin: 110, BPMMax: 125, Visible: true, Order: 4},
		{ID: "fontanna-laputa", Name: "Deep Current", Description: "Deep house, organic, hypnotic", MPDPort: 14004, StreamPort: 18004, Genre: "Deep House", Color: "#7b7bff", BPMMin: 115, BPMMax: 124, Visible: true, Order: 5},
		{ID: "palac-slow-hypno", Name: "Chill Terrace", Description: "Lo-fi, chillout, balearic", MPDPort: 14005, StreamPort: 18005, Genre: "Chillout", Color: "#44ddff", BPMMin: 80, BPMMax: 110, Visible: true, Order: 6},
		{ID: "bermuda-night", Name: "Bass Cave", Description: "DnB, breakbeat, UK garage", MPDPort: 14006, StreamPort: 18006, Genre: "DnB", Color: "#ff44ff", BPMMin: 140, BPMMax: 174, Visible: true, Order: 7},
		{ID: "bermuda-day", Name: "World Frequencies", Description: "Afro house, global bass", MPDPort: 14007, StreamPort: 18007, Genre: "Afro House", Color: "#ff4466", BPMMin: 115, BPMMax: 130, Visible: true, Order: 8},
		{ID: "closing", Name: "Live Sets", Description: "Recorded live sets and festival recordings", MPDPort: 14008, StreamPort: 18008, Genre: "Live", Color: "#00ff88", BPMMin: 0, BPMMax: 0, Visible: true, Order: 9},
	}
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
