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
	t.Setenv("DATABASE_SECRET_ARN", "")
	t.Setenv("DATABASE_HOST", "")
	t.Setenv("DATABASE_PORT", "")
	t.Setenv("DATABASE_NAME", "")
	t.Setenv("DATABASE_SSLMODE", "")

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
	if cfg.Database.URL != "" {
		t.Fatalf("expected empty default database config URL, got %q", cfg.Database.URL)
	}
	if cfg.Database.WithDefaults().Port != "5432" {
		t.Fatalf("expected default database port 5432, got %q", cfg.Database.WithDefaults().Port)
	}
	if cfg.Database.WithDefaults().SSLMode != "require" {
		t.Fatalf("expected default database sslmode require, got %q", cfg.Database.WithDefaults().SSLMode)
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
	if cfg.AuthFlow.RateLimitMaxBuckets != defaultAuthRateLimitMaxBuckets {
		t.Fatalf("expected default auth rate limit max buckets %d, got %d", defaultAuthRateLimitMaxBuckets, cfg.AuthFlow.RateLimitMaxBuckets)
	}
	if cfg.AuthFlow.EmailVerificationTTLHours != defaultEmailVerificationTTLHours {
		t.Fatalf("expected default email verification TTL %d, got %d", defaultEmailVerificationTTLHours, cfg.AuthFlow.EmailVerificationTTLHours)
	}
	if cfg.AuthFlow.PasswordResetTTLHours != defaultPasswordResetTTLHours {
		t.Fatalf("expected default password reset TTL %d, got %d", defaultPasswordResetTTLHours, cfg.AuthFlow.PasswordResetTTLHours)
	}
	if cfg.Email.Enabled() {
		t.Fatal("expected email config to be disabled by default")
	}
	if cfg.Email.WithDefaults().Provider != EmailProviderSMTP {
		t.Fatalf("expected default email provider %q, got %q", EmailProviderSMTP, cfg.Email.WithDefaults().Provider)
	}
	if cfg.Email.SMTPPort != defaultSMTPPort {
		t.Fatalf("expected default SMTP port %d, got %d", defaultSMTPPort, cfg.Email.SMTPPort)
	}
	if len(cfg.TrustedProxies) != 0 {
		t.Fatalf("expected no trusted proxies by default, got %v", cfg.TrustedProxies)
	}
}

func TestLoadUsesEnvironmentValues(t *testing.T) {
	passwordHash := basicAuthHash(t, "secret")

	t.Setenv("APP_HOST", "localhost")
	t.Setenv("APP_PORT", "9000")
	t.Setenv("DATABASE_URL", "test-database-url")
	t.Setenv("DATABASE_SECRET_ARN", "test-database-secret-arn")
	t.Setenv("DATABASE_HOST", "db.example.internal")
	t.Setenv("DATABASE_PORT", "6543")
	t.Setenv("DATABASE_NAME", "monthly_goal_tracker")
	t.Setenv("DATABASE_SSLMODE", "verify-full")
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
	t.Setenv("APP_AUTH_RATE_LIMIT_MAX_BUCKETS", "1234")
	t.Setenv("APP_EMAIL_VERIFICATION_TTL_HOURS", "48")
	t.Setenv("APP_PASSWORD_RESET_TTL_HOURS", "2")
	t.Setenv("APP_EMAIL_PROVIDER", "ses")
	t.Setenv("APP_EMAIL_FROM", "no-reply@example.com")
	t.Setenv("APP_SMTP_HOST", "smtp.example.com")
	t.Setenv("APP_SMTP_PORT", "2525")
	t.Setenv("APP_SMTP_USERNAME", "smtp-user")
	t.Setenv("APP_SMTP_PASSWORD", "smtp-password")
	t.Setenv("APP_SES_REGION", "us-east-1")
	t.Setenv("APP_EMAIL_VERIFICATION_BASE_URL", "https://app.example.com/verify")
	t.Setenv("APP_PASSWORD_RESET_BASE_URL", "https://app.example.com/reset-password")
	t.Setenv("APP_TRUSTED_PROXIES", "127.0.0.1, 10.0.0.0/8")

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
	if cfg.Database.URL != "test-database-url" {
		t.Fatalf("expected database config URL from environment, got %q", cfg.Database.URL)
	}
	if cfg.Database.SecretARN != "test-database-secret-arn" {
		t.Fatalf("expected database secret ARN from environment, got %q", cfg.Database.SecretARN)
	}
	if cfg.Database.Host != "db.example.internal" {
		t.Fatalf("expected database host from environment, got %q", cfg.Database.Host)
	}
	if cfg.Database.Port != "6543" {
		t.Fatalf("expected database port from environment, got %q", cfg.Database.Port)
	}
	if cfg.Database.Name != "monthly_goal_tracker" {
		t.Fatalf("expected database name from environment, got %q", cfg.Database.Name)
	}
	if cfg.Database.SSLMode != "verify-full" {
		t.Fatalf("expected database sslmode from environment, got %q", cfg.Database.SSLMode)
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
	if cfg.AuthFlow.RateLimitMaxBuckets != 1234 {
		t.Fatalf("expected auth rate limit max buckets from environment, got %d", cfg.AuthFlow.RateLimitMaxBuckets)
	}
	if cfg.AuthFlow.EmailVerificationTTLHours != 48 {
		t.Fatalf("expected email verification TTL from environment, got %d", cfg.AuthFlow.EmailVerificationTTLHours)
	}
	if cfg.AuthFlow.PasswordResetTTLHours != 2 {
		t.Fatalf("expected password reset TTL from environment, got %d", cfg.AuthFlow.PasswordResetTTLHours)
	}
	if cfg.Email.Provider != EmailProviderSES {
		t.Fatalf("expected email provider from environment, got %q", cfg.Email.Provider)
	}
	if cfg.Email.From != "no-reply@example.com" {
		t.Fatalf("expected email from from environment, got %q", cfg.Email.From)
	}
	if cfg.Email.SMTPHost != "smtp.example.com" {
		t.Fatalf("expected SMTP host from environment, got %q", cfg.Email.SMTPHost)
	}
	if cfg.Email.SMTPPort != 2525 {
		t.Fatalf("expected SMTP port from environment, got %d", cfg.Email.SMTPPort)
	}
	if cfg.Email.SMTPUsername != "smtp-user" {
		t.Fatalf("expected SMTP username from environment, got %q", cfg.Email.SMTPUsername)
	}
	if cfg.Email.SMTPPassword != "smtp-password" {
		t.Fatal("expected SMTP password from environment")
	}
	if cfg.Email.SESRegion != "us-east-1" {
		t.Fatalf("expected SES region from environment, got %q", cfg.Email.SESRegion)
	}
	if cfg.Email.VerificationBaseURL != "https://app.example.com/verify" {
		t.Fatalf("expected verification base URL from environment, got %q", cfg.Email.VerificationBaseURL)
	}
	if cfg.Email.PasswordResetBaseURL != "https://app.example.com/reset-password" {
		t.Fatalf("expected password reset base URL from environment, got %q", cfg.Email.PasswordResetBaseURL)
	}
	if len(cfg.TrustedProxies) != 2 || cfg.TrustedProxies[0] != "127.0.0.1" || cfg.TrustedProxies[1] != "10.0.0.0/8" {
		t.Fatalf("expected trusted proxies from environment, got %v", cfg.TrustedProxies)
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
		{
			name:     "negative max buckets",
			authFlow: AuthFlowConfig{RateLimitMaxBuckets: -1},
		},
		{
			name:     "negative email verification TTL",
			authFlow: AuthFlowConfig{EmailVerificationTTLHours: -1},
		},
		{
			name:     "negative password reset TTL",
			authFlow: AuthFlowConfig{PasswordResetTTLHours: -1},
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

func TestValidateAllowsCompleteEmailConfig(t *testing.T) {
	cfg := testConfig()
	cfg.Email = EmailConfig{
		From:                 "no-reply@example.com",
		SMTPHost:             "smtp.example.com",
		SMTPPort:             587,
		SMTPUsername:         "smtp-user",
		SMTPPassword:         "smtp-password",
		VerificationBaseURL:  "https://app.example.com/verify",
		PasswordResetBaseURL: "https://app.example.com/reset-password",
	}

	if err := cfg.Validate(); err != nil {
		t.Fatalf("expected complete email config to be allowed, got %v", err)
	}
}

func TestValidateAllowsCompleteSESEmailConfig(t *testing.T) {
	cfg := testConfig()
	cfg.Email = EmailConfig{
		Provider:            EmailProviderSES,
		From:                "no-reply@example.com",
		SESRegion:           "us-east-1",
		VerificationBaseURL: "https://app.example.com/verify",
	}

	if err := cfg.Validate(); err != nil {
		t.Fatalf("expected complete SES email config to be allowed, got %v", err)
	}
}

func TestEmailConfigWithDefaultsUsesVerificationURLForPasswordReset(t *testing.T) {
	cfg := EmailConfig{VerificationBaseURL: "https://app.example.com/verify"}.WithDefaults()

	if cfg.PasswordResetBaseURL != "https://app.example.com/verify" {
		t.Fatalf("expected verification URL fallback for password reset, got %q", cfg.PasswordResetBaseURL)
	}
}

func TestValidateAllowsLoopbackHTTPEmailVerificationURLs(t *testing.T) {
	testCases := []string{
		"http://localhost:5173/verify",
		"http://127.0.0.1:5173/verify",
		"http://[::1]:5173/verify",
	}

	for _, verificationURL := range testCases {
		t.Run(verificationURL, func(t *testing.T) {
			cfg := testConfig()
			cfg.Email = EmailConfig{
				From:                "no-reply@example.com",
				SMTPHost:            "smtp.example.com",
				VerificationBaseURL: verificationURL,
			}

			if err := cfg.Validate(); err != nil {
				t.Fatalf("expected loopback HTTP verification URL to be allowed, got %v", err)
			}
		})
	}
}

func TestValidateRejectsInvalidEmailConfig(t *testing.T) {
	testCases := []struct {
		name  string
		email EmailConfig
	}{
		{
			name: "unsupported provider",
			email: EmailConfig{
				Provider:            "mailgun",
				From:                "no-reply@example.com",
				SESRegion:           "us-east-1",
				VerificationBaseURL: "https://app.example.com/verify",
			},
		},
		{
			name: "missing SMTP host",
			email: EmailConfig{
				From:                "no-reply@example.com",
				VerificationBaseURL: "https://app.example.com/verify",
			},
		},
		{
			name: "invalid sender",
			email: EmailConfig{
				From:                "not-an-email",
				SMTPHost:            "smtp.example.com",
				VerificationBaseURL: "https://app.example.com/verify",
			},
		},
		{
			name: "negative SMTP port",
			email: EmailConfig{
				From:                "no-reply@example.com",
				SMTPHost:            "smtp.example.com",
				SMTPPort:            -1,
				VerificationBaseURL: "https://app.example.com/verify",
			},
		},
		{
			name: "partial SMTP credentials",
			email: EmailConfig{
				From:                "no-reply@example.com",
				SMTPHost:            "smtp.example.com",
				SMTPUsername:        "smtp-user",
				VerificationBaseURL: "https://app.example.com/verify",
			},
		},
		{
			name: "relative verification URL",
			email: EmailConfig{
				From:                "no-reply@example.com",
				SMTPHost:            "smtp.example.com",
				VerificationBaseURL: "/verify",
			},
		},
		{
			name: "plaintext non-loopback verification URL",
			email: EmailConfig{
				From:                "no-reply@example.com",
				SMTPHost:            "smtp.example.com",
				VerificationBaseURL: "http://app.example.com/verify",
			},
		},
		{
			name: "unsupported verification URL scheme",
			email: EmailConfig{
				From:                "no-reply@example.com",
				SMTPHost:            "smtp.example.com",
				VerificationBaseURL: "ftp://app.example.com/verify",
			},
		},
		{
			name: "relative password reset URL",
			email: EmailConfig{
				From:                 "no-reply@example.com",
				SMTPHost:             "smtp.example.com",
				VerificationBaseURL:  "https://app.example.com/verify",
				PasswordResetBaseURL: "/reset-password",
			},
		},
		{
			name: "plaintext non-loopback password reset URL",
			email: EmailConfig{
				From:                 "no-reply@example.com",
				SMTPHost:             "smtp.example.com",
				VerificationBaseURL:  "https://app.example.com/verify",
				PasswordResetBaseURL: "http://app.example.com/reset-password",
			},
		},
		{
			name: "missing SES region",
			email: EmailConfig{
				Provider:            EmailProviderSES,
				From:                "no-reply@example.com",
				VerificationBaseURL: "https://app.example.com/verify",
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			cfg := testConfig()
			cfg.Email = testCase.email

			err := cfg.Validate()
			if !errors.Is(err, ErrInvalidEmailConfig) {
				t.Fatalf("expected ErrInvalidEmailConfig, got %v", err)
			}
		})
	}
}

func TestValidateRejectsInvalidTrustedProxy(t *testing.T) {
	cfg := testConfig()
	cfg.TrustedProxies = []string{"not-a-proxy"}

	err := cfg.Validate()
	if !errors.Is(err, ErrInvalidProxy) {
		t.Fatalf("expected ErrInvalidProxy, got %v", err)
	}
}

func TestValidateAllowsTrustedProxyAddressesAndCIDRs(t *testing.T) {
	cfg := testConfig()
	cfg.TrustedProxies = []string{"127.0.0.1", "::1", "10.0.0.0/8"}

	if err := cfg.Validate(); err != nil {
		t.Fatalf("expected trusted proxies to be allowed, got %v", err)
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
