package config

import (
	"log"
	"os"
	"strconv"
)

// Config holds all runtime configuration loaded from environment variables.
type Config struct {
	ServerURL           string
	HostName            string
	Deployment          string
	PostIntervalSeconds int
	ListenAddr          string
}

// Load reads configuration from environment variables and applies defaults.
func Load() *Config {
	cfg := &Config{
		ServerURL:           os.Getenv("MOTADATA_SERVER_URL"),
		HostName:            os.Getenv("HOST_NAME"),
		Deployment:          os.Getenv("DEPLOYMENT"),
		PostIntervalSeconds: 60,
		ListenAddr:          ":8181",
	}

	if v := os.Getenv("POST_INTERVAL_SECONDS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			cfg.PostIntervalSeconds = n
		} else {
			log.Printf("invalid POST_INTERVAL_SECONDS %q, using default 60", v)
		}
	}

	if cfg.HostName == "" {
		if h, err := os.Hostname(); err == nil {
			cfg.HostName = h
		}
	}

	return cfg
}
