package config

import (
	"errors"
	"fmt"
	"net"
	"net/netip"
	"os"
	"strconv"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

var (
	ErrUnsafeHost        = errors.New("unsafe host")
	ErrInvalidAuthConfig = errors.New("invalid auth config")
	ErrInvalidSession    = errors.New("invalid session config")
	ErrInvalidAuthFlow   = errors.New("invalid auth flow config")
)

const (
	minimumBcryptCost                 = bcrypt.DefaultCost
	defaultSessionTTLHours            = 24 * 30
	defaultSignupRateLimitPerMinute   = 5
	defaultLoginRateLimitPerMinute    = 10
	minimumLegacyClaimTokenCharacters = 16
)

type Config struct {
	Host        string
	Port        string
	DatabaseURL string
	Auth        BasicAuthConfig
	Session     SessionConfig
	AuthFlow    AuthFlowConfig
}

type BasicAuthConfig struct {
	Username     string
	PasswordHash string
}

func (c BasicAuthConfig) Enabled() bool {
	return strings.TrimSpace(c.Username) != "" || strings.TrimSpace(c.PasswordHash) != ""
}

type SessionConfig struct {
	CookieName     string
	CSRFCookieName string
	TTLHours       int
	Secure         bool
	SameSite       string
}

func (c SessionConfig) WithDefaults() SessionConfig {
	if strings.TrimSpace(c.CookieName) == "" {
		c.CookieName = "mgt_session"
	}
	if strings.TrimSpace(c.CSRFCookieName) == "" {
		c.CSRFCookieName = "mgt_csrf"
	}
	if c.TTLHours == 0 {
		c.TTLHours = defaultSessionTTLHours
	}
	if strings.TrimSpace(c.SameSite) == "" {
		c.SameSite = "lax"
	}

	return c
}

type AuthFlowConfig struct {
	LegacyClaimToken         string
	SignupRateLimitPerMinute int
	LoginRateLimitPerMinute  int
}

func (c AuthFlowConfig) WithDefaults() AuthFlowConfig {
	if c.SignupRateLimitPerMinute == 0 {
		c.SignupRateLimitPerMinute = defaultSignupRateLimitPerMinute
	}
	if c.LoginRateLimitPerMinute == 0 {
		c.LoginRateLimitPerMinute = defaultLoginRateLimitPerMinute
	}
	c.LegacyClaimToken = strings.TrimSpace(c.LegacyClaimToken)

	return c
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
		Session: SessionConfig{
			CookieName:     getEnv("APP_SESSION_COOKIE_NAME", "mgt_session"),
			CSRFCookieName: getEnv("APP_CSRF_COOKIE_NAME", "mgt_csrf"),
			TTLHours:       getEnvInt("APP_SESSION_TTL_HOURS", defaultSessionTTLHours),
			Secure:         getEnvBool("APP_COOKIE_SECURE", false),
			SameSite:       getEnv("APP_COOKIE_SAMESITE", "lax"),
		},
		AuthFlow: AuthFlowConfig{
			LegacyClaimToken:         getEnv("APP_LEGACY_CLAIM_TOKEN", ""),
			SignupRateLimitPerMinute: getEnvInt("APP_SIGNUP_RATE_LIMIT_PER_MINUTE", defaultSignupRateLimitPerMinute),
			LoginRateLimitPerMinute:  getEnvInt("APP_LOGIN_RATE_LIMIT_PER_MINUTE", defaultLoginRateLimitPerMinute),
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
	if err := c.Session.Validate(); err != nil {
		return err
	}
	if err := c.AuthFlow.Validate(); err != nil {
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

func (c SessionConfig) Validate() error {
	cfg := c.WithDefaults()
	if strings.TrimSpace(cfg.CookieName) == "" || strings.TrimSpace(cfg.CSRFCookieName) == "" {
		return fmt.Errorf("%w: cookie names are required", ErrInvalidSession)
	}
	if cfg.TTLHours <= 0 {
		return fmt.Errorf("%w: APP_SESSION_TTL_HOURS must be positive", ErrInvalidSession)
	}

	switch strings.ToLower(strings.TrimSpace(cfg.SameSite)) {
	case "lax", "strict", "none":
	default:
		return fmt.Errorf("%w: APP_COOKIE_SAMESITE must be lax, strict, or none", ErrInvalidSession)
	}

	if strings.EqualFold(strings.TrimSpace(cfg.SameSite), "none") && !cfg.Secure {
		return fmt.Errorf("%w: APP_COOKIE_SECURE must be true when APP_COOKIE_SAMESITE is none", ErrInvalidSession)
	}

	return nil
}

func (c AuthFlowConfig) Validate() error {
	cfg := c.WithDefaults()
	if cfg.SignupRateLimitPerMinute <= 0 {
		return fmt.Errorf("%w: APP_SIGNUP_RATE_LIMIT_PER_MINUTE must be positive", ErrInvalidAuthFlow)
	}
	if cfg.LoginRateLimitPerMinute <= 0 {
		return fmt.Errorf("%w: APP_LOGIN_RATE_LIMIT_PER_MINUTE must be positive", ErrInvalidAuthFlow)
	}
	if cfg.LegacyClaimToken != "" && len(cfg.LegacyClaimToken) < minimumLegacyClaimTokenCharacters {
		return fmt.Errorf("%w: APP_LEGACY_CLAIM_TOKEN must be at least %d characters", ErrInvalidAuthFlow, minimumLegacyClaimTokenCharacters)
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

func getEnvInt(key string, fallback int) int {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}

	parsed, err := strconv.Atoi(value)
	if err != nil {
		return -1
	}

	return parsed
}

func getEnvBool(key string, fallback bool) bool {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}

	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return fallback
	}

	return parsed
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
