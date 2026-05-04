package config

import (
	"errors"
	"testing"
)

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
	t.Setenv("APP_HOST", "localhost")
	t.Setenv("APP_PORT", "9000")
	t.Setenv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/monthly_goal_tracker?sslmode=disable")

	cfg := Load()

	if cfg.Host != "localhost" {
		t.Fatalf("expected host from environment, got %q", cfg.Host)
	}

	if cfg.Port != "9000" {
		t.Fatalf("expected port from environment, got %q", cfg.Port)
	}

	if cfg.Addr() != "localhost:9000" {
		t.Fatalf("expected addr from environment, got %q", cfg.Addr())
	}

	if cfg.DatabaseURL != "postgres://postgres:postgres@localhost:5432/monthly_goal_tracker?sslmode=disable" {
		t.Fatalf("expected database URL from environment, got %q", cfg.DatabaseURL)
	}
}

func TestValidateAllowsLoopbackHosts(t *testing.T) {
	testCases := []struct {
		host string
		addr string
	}{
		{host: "127.0.0.1", addr: "127.0.0.1:8080"},
		{host: "localhost", addr: "localhost:8080"},
		{host: "::1", addr: "[::1]:8080"},
		{host: "[::1]", addr: "[::1]:8080"},
		{host: " localhost ", addr: "localhost:8080"},
	}

	for _, host := range testCases {
		t.Run(host.host, func(t *testing.T) {
			cfg := Config{Host: host.host, Port: "8080"}

			if err := cfg.Validate(); err != nil {
				t.Fatalf("expected loopback host %q to be allowed, got %v", host.host, err)
			}

			if cfg.Addr() != host.addr {
				t.Fatalf("expected addr %q, got %q", host.addr, cfg.Addr())
			}
		})
	}
}

func TestValidateRejectsNonLoopbackHost(t *testing.T) {
	cfg := Config{Host: "0.0.0.0", Port: "8080"}

	err := cfg.Validate()
	if !errors.Is(err, ErrUnsafeHost) {
		t.Fatalf("expected ErrUnsafeHost, got %v", err)
	}
}
