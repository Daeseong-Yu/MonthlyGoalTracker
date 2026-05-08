package config

import (
	"errors"
	"fmt"
	"net"
	"net/mail"
	"net/netip"
	"net/url"
	"os"
	"strconv"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

var (
	ErrUnsafeHost         = errors.New("unsafe host")
	ErrInvalidAuthConfig  = errors.New("invalid auth config")
	ErrInvalidSession     = errors.New("invalid session config")
	ErrInvalidAuthFlow    = errors.New("invalid auth flow config")
	ErrInvalidEmailConfig = errors.New("invalid email config")
	ErrInvalidProxy       = errors.New("invalid proxy config")
)

const (
	minimumBcryptCost                 = bcrypt.DefaultCost
	defaultSessionTTLHours            = 24 * 30
	defaultSignupRateLimitPerMinute   = 5
	defaultLoginRateLimitPerMinute    = 10
	defaultAuthRateLimitMaxBuckets    = 10000
	defaultEmailVerificationTTLHours  = 24
	defaultPasswordResetTTLHours      = 1
	defaultSMTPPort                   = 587
	minimumLegacyClaimTokenCharacters = 16
)

type Config struct {
	Host           string
	Port           string
	DatabaseURL    string
	Auth           BasicAuthConfig
	Session        SessionConfig
	AuthFlow       AuthFlowConfig
	Email          EmailConfig
	TrustedProxies []string
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
	LegacyClaimToken          string
	SignupRateLimitPerMinute  int
	LoginRateLimitPerMinute   int
	RateLimitMaxBuckets       int
	EmailVerificationTTLHours int
	PasswordResetTTLHours     int
}

func (c AuthFlowConfig) WithDefaults() AuthFlowConfig {
	if c.SignupRateLimitPerMinute == 0 {
		c.SignupRateLimitPerMinute = defaultSignupRateLimitPerMinute
	}
	if c.LoginRateLimitPerMinute == 0 {
		c.LoginRateLimitPerMinute = defaultLoginRateLimitPerMinute
	}
	if c.RateLimitMaxBuckets == 0 {
		c.RateLimitMaxBuckets = defaultAuthRateLimitMaxBuckets
	}
	if c.EmailVerificationTTLHours == 0 {
		c.EmailVerificationTTLHours = defaultEmailVerificationTTLHours
	}
	if c.PasswordResetTTLHours == 0 {
		c.PasswordResetTTLHours = defaultPasswordResetTTLHours
	}
	c.LegacyClaimToken = strings.TrimSpace(c.LegacyClaimToken)

	return c
}

type EmailConfig struct {
	From                 string
	SMTPHost             string
	SMTPPort             int
	SMTPUsername         string
	SMTPPassword         string
	VerificationBaseURL  string
	PasswordResetBaseURL string
}

func (c EmailConfig) WithDefaults() EmailConfig {
	c.From = strings.TrimSpace(c.From)
	c.SMTPHost = strings.TrimSpace(c.SMTPHost)
	c.SMTPUsername = strings.TrimSpace(c.SMTPUsername)
	c.SMTPPassword = strings.TrimSpace(c.SMTPPassword)
	c.VerificationBaseURL = strings.TrimSpace(c.VerificationBaseURL)
	c.PasswordResetBaseURL = strings.TrimSpace(c.PasswordResetBaseURL)
	if c.SMTPPort == 0 {
		c.SMTPPort = defaultSMTPPort
	}
	if c.PasswordResetBaseURL == "" {
		c.PasswordResetBaseURL = c.VerificationBaseURL
	}

	return c
}

func (c EmailConfig) Enabled() bool {
	cfg := c.WithDefaults()
	return cfg.From != "" || cfg.SMTPHost != "" || cfg.VerificationBaseURL != "" || cfg.PasswordResetBaseURL != ""
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
			LegacyClaimToken:          getEnv("APP_LEGACY_CLAIM_TOKEN", ""),
			SignupRateLimitPerMinute:  getEnvInt("APP_SIGNUP_RATE_LIMIT_PER_MINUTE", defaultSignupRateLimitPerMinute),
			LoginRateLimitPerMinute:   getEnvInt("APP_LOGIN_RATE_LIMIT_PER_MINUTE", defaultLoginRateLimitPerMinute),
			RateLimitMaxBuckets:       getEnvInt("APP_AUTH_RATE_LIMIT_MAX_BUCKETS", defaultAuthRateLimitMaxBuckets),
			EmailVerificationTTLHours: getEnvInt("APP_EMAIL_VERIFICATION_TTL_HOURS", defaultEmailVerificationTTLHours),
			PasswordResetTTLHours:     getEnvInt("APP_PASSWORD_RESET_TTL_HOURS", defaultPasswordResetTTLHours),
		},
		Email: EmailConfig{
			From:                 getEnv("APP_EMAIL_FROM", ""),
			SMTPHost:             getEnv("APP_SMTP_HOST", ""),
			SMTPPort:             getEnvInt("APP_SMTP_PORT", defaultSMTPPort),
			SMTPUsername:         getEnv("APP_SMTP_USERNAME", ""),
			SMTPPassword:         getEnv("APP_SMTP_PASSWORD", ""),
			VerificationBaseURL:  getEnv("APP_EMAIL_VERIFICATION_BASE_URL", ""),
			PasswordResetBaseURL: getEnv("APP_PASSWORD_RESET_BASE_URL", ""),
		},
		TrustedProxies: getEnvList("APP_TRUSTED_PROXIES"),
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
	if err := c.Email.Validate(); err != nil {
		return err
	}
	if err := validateTrustedProxies(c.TrustedProxies); err != nil {
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
	if cfg.RateLimitMaxBuckets <= 0 {
		return fmt.Errorf("%w: APP_AUTH_RATE_LIMIT_MAX_BUCKETS must be positive", ErrInvalidAuthFlow)
	}
	if cfg.EmailVerificationTTLHours <= 0 {
		return fmt.Errorf("%w: APP_EMAIL_VERIFICATION_TTL_HOURS must be positive", ErrInvalidAuthFlow)
	}
	if cfg.PasswordResetTTLHours <= 0 {
		return fmt.Errorf("%w: APP_PASSWORD_RESET_TTL_HOURS must be positive", ErrInvalidAuthFlow)
	}
	if cfg.LegacyClaimToken != "" && len(cfg.LegacyClaimToken) < minimumLegacyClaimTokenCharacters {
		return fmt.Errorf("%w: APP_LEGACY_CLAIM_TOKEN must be at least %d characters", ErrInvalidAuthFlow, minimumLegacyClaimTokenCharacters)
	}

	return nil
}

func (c EmailConfig) Validate() error {
	cfg := c.WithDefaults()
	configured := cfg.Enabled() || cfg.SMTPUsername != "" || cfg.SMTPPassword != ""
	if !configured {
		return nil
	}

	if cfg.From == "" || cfg.SMTPHost == "" || cfg.VerificationBaseURL == "" {
		return fmt.Errorf("%w: APP_EMAIL_FROM, APP_SMTP_HOST, and APP_EMAIL_VERIFICATION_BASE_URL must be set together", ErrInvalidEmailConfig)
	}
	if _, err := mail.ParseAddress(cfg.From); err != nil {
		return fmt.Errorf("%w: APP_EMAIL_FROM must be a valid email address", ErrInvalidEmailConfig)
	}
	if cfg.SMTPPort <= 0 {
		return fmt.Errorf("%w: APP_SMTP_PORT must be positive", ErrInvalidEmailConfig)
	}
	if (cfg.SMTPUsername == "") != (cfg.SMTPPassword == "") {
		return fmt.Errorf("%w: APP_SMTP_USERNAME and APP_SMTP_PASSWORD must be set together", ErrInvalidEmailConfig)
	}

	if err := validateEmailActionBaseURL("APP_EMAIL_VERIFICATION_BASE_URL", cfg.VerificationBaseURL); err != nil {
		return err
	}
	if err := validateEmailActionBaseURL("APP_PASSWORD_RESET_BASE_URL", cfg.PasswordResetBaseURL); err != nil {
		return err
	}

	return nil
}

func validateEmailActionBaseURL(envName, rawURL string) error {
	actionURL, err := url.Parse(rawURL)
	if err != nil || actionURL.Scheme == "" || actionURL.Host == "" {
		return fmt.Errorf("%w: %s must be an absolute URL", ErrInvalidEmailConfig, envName)
	}
	if !safeEmailActionBaseURL(actionURL) {
		return fmt.Errorf("%w: %s must use https outside localhost", ErrInvalidEmailConfig, envName)
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

func getEnvList(key string) []string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return nil
	}

	parts := strings.Split(value, ",")
	values := make([]string, 0, len(parts))
	for _, part := range parts {
		item := strings.TrimSpace(part)
		if item != "" {
			values = append(values, item)
		}
	}

	return values
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

func safeEmailActionBaseURL(actionURL *url.URL) bool {
	switch strings.ToLower(actionURL.Scheme) {
	case "https":
		return true
	case "http":
		return isLoopbackHost(normalizeHost(actionURL.Hostname()))
	default:
		return false
	}
}

func validateTrustedProxies(values []string) error {
	for _, value := range values {
		if _, err := netip.ParseAddr(value); err == nil {
			continue
		}
		if _, err := netip.ParsePrefix(value); err == nil {
			continue
		}

		return fmt.Errorf("%w: APP_TRUSTED_PROXIES contains invalid proxy %q", ErrInvalidProxy, value)
	}

	return nil
}
