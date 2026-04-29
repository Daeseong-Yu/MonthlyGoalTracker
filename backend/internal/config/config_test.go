package config

import "testing"

func TestLoadUsesDefaultValues(t *testing.T) {
	t.Setenv("APP_HOST", "")
	t.Setenv("APP_PORT", "")
	t.Setenv("DATABASE_URL", "")

	cfg := Load()

	if cfg.Host != "127.0.0.1" {
		t.Fatalf("expected default host 127.0.0.1, got %q", cfg.Host)
	}

	if cfg.Port != "8080" {
		t.Fatalf("expected default port 8080, got %q", cfg.Port)
	}

	if cfg.Addr() != "127.0.0.1:8080" {
		t.Fatalf("expected default addr 127.0.0.1:8080, got %q", cfg.Addr())
	}

	if cfg.DatabaseURL != "" {
		t.Fatalf("expected empty default database URL, got %q", cfg.DatabaseURL)
	}
}

func TestLoadUsesEnvironmentValues(t *testing.T) {
	t.Setenv("APP_HOST", "0.0.0.0")
	t.Setenv("APP_PORT", "9000")
	t.Setenv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/monthly_goal_tracker?sslmode=disable")

	cfg := Load()

	if cfg.Host != "0.0.0.0" {
		t.Fatalf("expected host from environment, got %q", cfg.Host)
	}

	if cfg.Port != "9000" {
		t.Fatalf("expected port from environment, got %q", cfg.Port)
	}

	if cfg.Addr() != "0.0.0.0:9000" {
		t.Fatalf("expected addr from environment, got %q", cfg.Addr())
	}

	if cfg.DatabaseURL != "postgres://postgres:postgres@localhost:5432/monthly_goal_tracker?sslmode=disable" {
		t.Fatalf("expected database URL from environment, got %q", cfg.DatabaseURL)
	}
}
