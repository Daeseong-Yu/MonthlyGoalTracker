package config

import (
	"errors"
	"testing"

	"golang.org/x/crypto/bcrypt"
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

	if cfg.Auth.Enabled() {
		t.Fatal("expected basic auth to be disabled by default")
	}
}

func TestLoadUsesEnvironmentValues(t *testing.T) {
	passwordHash := basicAuthHash(t, "secret")

	t.Setenv("APP_HOST", "localhost")
	t.Setenv("APP_PORT", "9000")
	t.Setenv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/monthly_goal_tracker?sslmode=disable")
	t.Setenv("APP_BASIC_AUTH_USERNAME", "app-user")
	t.Setenv("APP_BASIC_AUTH_PASSWORD_HASH", passwordHash)

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

	if cfg.Auth.Username != "app-user" {
		t.Fatalf("expected basic auth username from environment, got %q", cfg.Auth.Username)
	}

	if cfg.Auth.PasswordHash != passwordHash {
		t.Fatal("expected basic auth password hash from environment")
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

func TestValidateAllowsCompleteBasicAuthConfig(t *testing.T) {
	cfg := Config{
		Host: "127.0.0.1",
		Port: "8080",
		Auth: BasicAuthConfig{
			Username:     "app-user",
			PasswordHash: basicAuthHash(t, "secret"),
		},
	}

	if err := cfg.Validate(); err != nil {
		t.Fatalf("expected complete basic auth config to be allowed, got %v", err)
	}
}

func TestValidateRejectsPartialBasicAuthConfig(t *testing.T) {
	testCases := []struct {
		name string
		auth BasicAuthConfig
	}{
		{
			name: "missing password hash",
			auth: BasicAuthConfig{Username: "app-user"},
		},
		{
			name: "missing username",
			auth: BasicAuthConfig{PasswordHash: basicAuthHash(t, "secret")},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			cfg := Config{Host: "127.0.0.1", Port: "8080", Auth: testCase.auth}

			err := cfg.Validate()
			if !errors.Is(err, ErrInvalidAuthConfig) {
				t.Fatalf("expected ErrInvalidAuthConfig, got %v", err)
			}
		})
	}
}

func TestValidateRejectsInvalidBasicAuthPasswordHash(t *testing.T) {
	cfg := Config{
		Host: "127.0.0.1",
		Port: "8080",
		Auth: BasicAuthConfig{
			Username:     "app-user",
			PasswordHash: "not-a-bcrypt-hash",
		},
	}

	err := cfg.Validate()
	if !errors.Is(err, ErrInvalidAuthConfig) {
		t.Fatalf("expected ErrInvalidAuthConfig, got %v", err)
	}
}

func TestValidateRejectsWeakBasicAuthPasswordHash(t *testing.T) {
	cfg := Config{
		Host: "127.0.0.1",
		Port: "8080",
		Auth: BasicAuthConfig{
			Username:     "app-user",
			PasswordHash: basicAuthHashWithCost(t, "secret", bcrypt.MinCost),
		},
	}

	err := cfg.Validate()
	if !errors.Is(err, ErrInvalidAuthConfig) {
		t.Fatalf("expected ErrInvalidAuthConfig, got %v", err)
	}
}

func basicAuthHash(t *testing.T, password string) string {
	return basicAuthHashWithCost(t, password, minimumBcryptCost)
}

func basicAuthHashWithCost(t *testing.T, password string, cost int) string {
	t.Helper()

	hash, err := bcrypt.GenerateFromPassword([]byte(password), cost)
	if err != nil {
		t.Fatalf("failed to generate bcrypt hash: %v", err)
	}

	return string(hash)
}
