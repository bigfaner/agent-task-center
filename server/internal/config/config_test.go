package config

import (
	"testing"
)

func TestLoadDefaults(t *testing.T) {
	// Clear all relevant env vars to test defaults
	t.Setenv("DB_DRIVER", "")
	t.Setenv("DB_PATH", "")
	t.Setenv("DB_CONN_STR", "")
	t.Setenv("PORT", "")
	t.Setenv("STATIC_DIR", "")

	cfg := Load()

	if cfg.DBDriver != "sqlite" {
		t.Errorf("expected DBDriver default 'sqlite', got %q", cfg.DBDriver)
	}
	if cfg.DBPath != "./data/tasks.db" {
		t.Errorf("expected DBPath default './data/tasks.db', got %q", cfg.DBPath)
	}
	if cfg.DBConnStr != "" {
		t.Errorf("expected DBConnStr default '', got %q", cfg.DBConnStr)
	}
	if cfg.Port != 8080 {
		t.Errorf("expected Port default 8080, got %d", cfg.Port)
	}
	if cfg.StaticDir != "" {
		t.Errorf("expected StaticDir default '', got %q", cfg.StaticDir)
	}
}

func TestLoadFromEnv(t *testing.T) {
	tests := []struct {
		name     string
		envKey   string
		envVal   string
		getField func(*Config) string
		want     string
	}{
		{
			name: "DB_DRIVER env", envKey: "DB_DRIVER", envVal: "postgres",
			getField: func(c *Config) string { return c.DBDriver }, want: "postgres",
		},
		{
			name: "DB_PATH env", envKey: "DB_PATH", envVal: "/tmp/test.db",
			getField: func(c *Config) string { return c.DBPath }, want: "/tmp/test.db",
		},
		{
			name: "DB_CONN_STR env", envKey: "DB_CONN_STR", envVal: "postgres://user:pass@localhost/db",
			getField: func(c *Config) string { return c.DBConnStr }, want: "postgres://user:pass@localhost/db",
		},
		{
			name: "STATIC_DIR env", envKey: "STATIC_DIR", envVal: "/var/www",
			getField: func(c *Config) string { return c.StaticDir }, want: "/var/www",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv(tt.envKey, tt.envVal)

			cfg := Load()
			got := tt.getField(cfg)
			if got != tt.want {
				t.Errorf("expected %s=%q, got %q", tt.envKey, tt.want, got)
			}
		})
	}
}

func TestLoadPortFromEnv(t *testing.T) {
	t.Setenv("PORT", "3000")

	cfg := Load()
	if cfg.Port != 3000 {
		t.Errorf("expected Port=3000, got %d", cfg.Port)
	}
}

func TestLoadInvalidPortFallsBack(t *testing.T) {
	t.Setenv("PORT", "not-a-number")

	cfg := Load()
	if cfg.Port != 8080 {
		t.Errorf("expected Port fallback to 8080, got %d", cfg.Port)
	}
}
