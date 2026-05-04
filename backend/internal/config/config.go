package config

import (
	"errors"
	"fmt"
	"net"
	"net/netip"
	"os"
	"strings"
)

var ErrUnsafeHost = errors.New("unsafe host")

type Config struct {
	Host        string
	Port        string
	DatabaseURL string
}

func Load() Config {
	return Config{
		Host:        getEnv("APP_HOST", "127.0.0.1"),
		Port:        getEnv("APP_PORT", "8080"),
		DatabaseURL: getEnv("DATABASE_URL", ""),
	}
}

func (c Config) Addr() string {
	return net.JoinHostPort(normalizeHost(c.Host), c.Port)
}

func (c Config) Validate() error {
	if !isLoopbackHost(normalizeHost(c.Host)) {
		return fmt.Errorf("%w: APP_HOST must be loopback while authentication is disabled", ErrUnsafeHost)
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
