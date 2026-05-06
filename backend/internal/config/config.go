package config

import (
	"errors"
	"fmt"
	"net"
	"net/netip"
	"os"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

var (
	ErrUnsafeHost        = errors.New("unsafe host")
	ErrInvalidAuthConfig = errors.New("invalid auth config")
)

const minimumBcryptCost = bcrypt.DefaultCost

type Config struct {
	Host        string
	Port        string
	DatabaseURL string
	Auth        BasicAuthConfig
}

type BasicAuthConfig struct {
	Username     string
	PasswordHash string
}

func (c BasicAuthConfig) Enabled() bool {
	return strings.TrimSpace(c.Username) != "" || strings.TrimSpace(c.PasswordHash) != ""
}

func Load() Config {
	return Config{
		Host:        getEnv("APP_HOST", "127.0.0.1"),
		Port:        getEnv("APP_PORT", "8080"),
		DatabaseURL: getEnv("DATABASE_URL", ""),
		Auth: BasicAuthConfig{
			Username:     getEnv("APP_BASIC_AUTH_USERNAME", ""),
			PasswordHash: getEnv("APP_BASIC_AUTH_PASSWORD_HASH", ""),
		},
	}
}

func (c Config) Addr() string {
	return net.JoinHostPort(normalizeHost(c.Host), c.Port)
}

func (c Config) Validate() error {
	if err := c.Auth.Validate(); err != nil {
		return err
	}

	if !isLoopbackHost(normalizeHost(c.Host)) {
		return fmt.Errorf("%w: APP_HOST must be loopback", ErrUnsafeHost)
	}

	return nil
}

func (c BasicAuthConfig) Validate() error {
	username := strings.TrimSpace(c.Username)
	passwordHash := strings.TrimSpace(c.PasswordHash)

	if username == "" && passwordHash == "" {
		return nil
	}

	if username == "" || passwordHash == "" {
		return fmt.Errorf("%w: APP_BASIC_AUTH_USERNAME and APP_BASIC_AUTH_PASSWORD_HASH must be set together", ErrInvalidAuthConfig)
	}

	cost, err := bcrypt.Cost([]byte(passwordHash))
	if err != nil {
		return fmt.Errorf("%w: APP_BASIC_AUTH_PASSWORD_HASH must be a bcrypt hash", ErrInvalidAuthConfig)
	}

	if cost < minimumBcryptCost {
		return fmt.Errorf("%w: APP_BASIC_AUTH_PASSWORD_HASH cost must be at least %d", ErrInvalidAuthConfig, minimumBcryptCost)
	}

	return nil
}

func getEnv(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	return value
}

func normalizeHost(host string) string {
	trimmedHost := strings.TrimSpace(host)
	return strings.Trim(trimmedHost, "[]")
}

func isLoopbackHost(host string) bool {
	if host == "localhost" {
		return true
	}

	addr, err := netip.ParseAddr(host)
	if err != nil {
		return false
	}

	return addr.IsLoopback()
}
