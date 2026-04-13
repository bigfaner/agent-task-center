// Package config loads application configuration from environment variables.
package config

import (
	"os"
	"strconv"
)

// Config holds application configuration loaded from environment variables.
type Config struct {
	DBDriver  string // "sqlite" | "postgres"
	DBPath    string // SQLite file path
	DBConnStr string // PostgreSQL DSN
	Port      int    // HTTP port
	StaticDir string // Web UI static files directory
}

// Load reads configuration from environment variables with sensible defaults.
func Load() *Config {
	return &Config{
		DBDriver:  envOr("DB_DRIVER", "sqlite"),
		DBPath:    envOr("DB_PATH", "./data/tasks.db"),
		DBConnStr: os.Getenv("DB_CONN_STR"),
		Port:      envIntOr("PORT", 8080),
		StaticDir: os.Getenv("STATIC_DIR"),
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func envIntOr(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return n
}
