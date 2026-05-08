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

	if cfg.Session.CookieName != "mgt_session" {
		t.Fatalf("expected default session cookie name, got %q", cfg.Session.CookieName)
	}
	if cfg.Session.CSRFCookieName != "mgt_csrf" {
		t.Fatalf("expected default csrf cookie name, got %q", cfg.Session.CSRFCookieName)
	}
	if cfg.Session.TTLHours != defaultSessionTTLHours {
		t.Fatalf("expected default session TTL %d, got %d", defaultSessionTTLHours, cfg.Session.TTLHours)
	}
	if cfg.Session.SameSite != "lax" {
		t.Fatalf("expected default SameSite lax, got %q", cfg.Session.SameSite)
	}
	if cfg.AuthFlow.SignupRateLimitPerMinute != defaultSignupRateLimitPerMinute {
		t.Fatalf("expected default signup rate limit %d, got %d", defaultSignupRateLimitPerMinute, cfg.AuthFlow.SignupRateLimitPerMinute)
	}
	if cfg.AuthFlow.LoginRateLimitPerMinute != defaultLoginRateLimitPerMinute {
		t.Fatalf("expected default login rate limit %d, got %d", defaultLoginRateLimitPerMinute, cfg.AuthFlow.LoginRateLimitPerMinute)
	}
}

func TestLoadUsesEnvironmentValues(t *testing.T) {
	passwordHash := basicAuthHash(t, "secret")

	t.Setenv("APP_HOST", "localhost")
	t.Setenv("APP_PORT", "9000")
	t.Setenv("DATABASE_URL", "test-database-url")
	t.Setenv("APP_BASIC_AUTH_USERNAME", "app-user")
	t.Setenv("APP_BASIC_AUTH_PASSWORD_HASH", passwordHash)
	t.Setenv("APP_SESSION_COOKIE_NAME", "session_cookie")
	t.Setenv("APP_CSRF_COOKIE_NAME", "csrf_cookie")
	t.Setenv("APP_SESSION_TTL_HOURS", "12")
	t.Setenv("APP_COOKIE_SECURE", "true")
	t.Setenv("APP_COOKIE_SAMESITE", "strict")
	t.Setenv("APP_LEGACY_CLAIM_TOKEN", "owner-claim-token-123")
	t.Setenv("APP_SIGNUP_RATE_LIMIT_PER_MINUTE", "3")
	t.Setenv("APP_LOGIN_RATE_LIMIT_PER_MINUTE", "4")

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

	if cfg.DatabaseURL != "test-database-url" {
		t.Fatalf("expected database URL from environment, got %q", cfg.DatabaseURL)
	}

	if cfg.Auth.Username != "app-user" {
		t.Fatalf("expected basic auth username from environment, got %q", cfg.Auth.Username)
	}

	if cfg.Auth.PasswordHash != passwordHash {
		t.Fatal("expected basic auth password hash from environment")
	}

	if cfg.Session.CookieName != "session_cookie" {
		t.Fatalf("expected session cookie name from environment, got %q", cfg.Session.CookieName)
	}
	if cfg.Session.CSRFCookieName != "csrf_cookie" {
		t.Fatalf("expected csrf cookie name from environment, got %q", cfg.Session.CSRFCookieName)
	}
	if cfg.Session.TTLHours != 12 {
		t.Fatalf("expected session TTL from environment, got %d", cfg.Session.TTLHours)
	}
	if !cfg.Session.Secure {
		t.Fatal("expected secure cookie setting from environment")
	}
	if cfg.Session.SameSite != "strict" {
		t.Fatalf("expected SameSite from environment, got %q", cfg.Session.SameSite)
	}
	if cfg.AuthFlow.LegacyClaimToken != "owner-claim-token-123" {
		t.Fatalf("expected legacy claim token from environment, got %q", cfg.AuthFlow.LegacyClaimToken)
	}
	if cfg.AuthFlow.SignupRateLimitPerMinute != 3 {
		t.Fatalf("expected signup rate limit from environment, got %d", cfg.AuthFlow.SignupRateLimitPerMinute)
	}
	if cfg.AuthFlow.LoginRateLimitPerMinute != 4 {
		t.Fatalf("expected login rate limit from environment, got %d", cfg.AuthFlow.LoginRateLimitPerMinute)
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
			cfg := testConfig()
			cfg.Host = host.host

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
	cfg := testConfig()
	cfg.Host = "0.0.0.0"

	err := cfg.Validate()
	if !errors.Is(err, ErrUnsafeHost) {
		t.Fatalf("expected ErrUnsafeHost, got %v", err)
	}
}

func TestValidateAllowsCompleteBasicAuthConfig(t *testing.T) {
	cfg := testConfig()
	cfg.Auth = testBasicAuthConfig(t)

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
			cfg := testConfig()
			cfg.Auth = testCase.auth

			err := cfg.Validate()
			if !errors.Is(err, ErrInvalidAuthConfig) {
				t.Fatalf("expected ErrInvalidAuthConfig, got %v", err)
			}
		})
	}
}

func TestValidateRejectsInvalidBasicAuthPasswordHash(t *testing.T) {
	cfg := testConfig()
	cfg.Auth = BasicAuthConfig{
		Username:     "app-user",
		PasswordHash: "not-a-bcrypt-hash",
	}

	err := cfg.Validate()
	if !errors.Is(err, ErrInvalidAuthConfig) {
		t.Fatalf("expected ErrInvalidAuthConfig, got %v", err)
	}
}

func TestValidateRejectsWeakBasicAuthPasswordHash(t *testing.T) {
	cfg := testConfig()
	cfg.Auth = BasicAuthConfig{
		Username:     "app-user",
		PasswordHash: basicAuthHashWithCost(t, "secret", bcrypt.MinCost),
	}

	err := cfg.Validate()
	if !errors.Is(err, ErrInvalidAuthConfig) {
		t.Fatalf("expected ErrInvalidAuthConfig, got %v", err)
	}
}

func TestValidateRejectsSameSiteNoneWithoutSecureCookie(t *testing.T) {
	cfg := testConfig()
	cfg.Session = SessionConfig{SameSite: "none"}

	err := cfg.Validate()
	if !errors.Is(err, ErrInvalidSession) {
		t.Fatalf("expected ErrInvalidSession, got %v", err)
	}
}

func TestValidateAllowsSameSiteNoneWithSecureCookie(t *testing.T) {
	cfg := testConfig()
	cfg.Session = SessionConfig{SameSite: "none", Secure: true}

	if err := cfg.Validate(); err != nil {
		t.Fatalf("expected SameSite none with secure cookie to be allowed, got %v", err)
	}
}

func TestValidateRejectsWeakLegacyClaimToken(t *testing.T) {
	cfg := testConfig()
	cfg.AuthFlow = AuthFlowConfig{LegacyClaimToken: "short-token"}

	err := cfg.Validate()
	if !errors.Is(err, ErrInvalidAuthFlow) {
		t.Fatalf("expected ErrInvalidAuthFlow, got %v", err)
	}
}

func TestValidateRejectsInvalidAuthRateLimits(t *testing.T) {
	testCases := []struct {
		name     string
		authFlow AuthFlowConfig
	}{
		{
			name:     "negative signup rate limit",
			authFlow: AuthFlowConfig{SignupRateLimitPerMinute: -1},
		},
		{
			name:     "negative login rate limit",
			authFlow: AuthFlowConfig{LoginRateLimitPerMinute: -1},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			cfg := testConfig()
			cfg.AuthFlow = testCase.authFlow

			err := cfg.Validate()
			if !errors.Is(err, ErrInvalidAuthFlow) {
				t.Fatalf("expected ErrInvalidAuthFlow, got %v", err)
			}
		})
	}
}

func testConfig() Config {
	return Config{Host: "127.0.0.1", Port: "8080"}
}

func testBasicAuthConfig(t *testing.T) BasicAuthConfig {
	t.Helper()

	return BasicAuthConfig{
		Username:     "app-user",
		PasswordHash: basicAuthHash(t, "secret"),
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
