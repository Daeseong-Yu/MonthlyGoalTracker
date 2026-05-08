package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Daeseong-Yu/MonthlyGoalTracker/backend/internal/domain"
	"gorm.io/gorm"
)

func TestAuthServiceSignUpDoesNotClaimLegacyDataWithoutToken(t *testing.T) {
	users := &fakeAuthUserRepository{findByEmailErr: gorm.ErrRecordNotFound}
	sessions := &fakeAuthSessionRepository{}
	authService := NewAuthService(users, sessions, time.Hour, "owner-claim-token-123")
	authService.now = func() time.Time { return date(2026, time.May, 8) }

	result, err := authService.SignUp(context.Background(), " Owner@Example.com ", "strong-password", "en", "")
	if err != nil {
		t.Fatalf("expected signup to succeed, got %v", err)
	}
	if !users.createCalled {
		t.Fatal("expected user to be created")
	}
	if users.claimLegacy {
		t.Fatal("expected signup without a claim token to create a fresh user")
	}
	if users.email != "owner@example.com" {
		t.Fatalf("expected normalized email, got %q", users.email)
	}
	if users.locale != "en" {
		t.Fatalf("expected locale en, got %q", users.locale)
	}
	if result.Auth == nil {
		t.Fatal("expected signup to create an auth result")
	}
	if result.Auth.User.ID != 7 {
		t.Fatalf("expected created user ID 7, got %d", result.Auth.User.ID)
	}
	if result.Auth.User.EmailVerifiedAt == nil {
		t.Fatal("expected immediate signup user to be treated as email verified")
	}
	if sessions.createdSession == nil {
		t.Fatal("expected session to be created")
	}
}

func TestAuthServiceSignUpClaimsLegacyDataWithConfiguredToken(t *testing.T) {
	users := &fakeAuthUserRepository{findByEmailErr: gorm.ErrRecordNotFound}
	sessions := &fakeAuthSessionRepository{}
	authService := NewAuthService(users, sessions, time.Hour, "owner-claim-token-123")

	_, err := authService.SignUp(context.Background(), "owner@example.com", "strong-password", "ko", " owner-claim-token-123 ")
	if err != nil {
		t.Fatalf("expected signup to succeed, got %v", err)
	}
	if !users.claimLegacy {
		t.Fatal("expected valid claim token to request legacy data claim")
	}
}

func TestAuthServiceSignUpRejectsInvalidLegacyClaimToken(t *testing.T) {
	users := &fakeAuthUserRepository{findByEmailErr: gorm.ErrRecordNotFound}
	sessions := &fakeAuthSessionRepository{}
	authService := NewAuthService(users, sessions, time.Hour, "owner-claim-token-123")

	result, err := authService.SignUp(context.Background(), "owner@example.com", "strong-password", "ko", "wrong-token")
	if result != nil {
		t.Fatal("expected nil auth result")
	}
	if !errors.Is(err, ErrInvalidLegacyClaim) {
		t.Fatalf("expected ErrInvalidLegacyClaim, got %v", err)
	}
	if users.createCalled {
		t.Fatal("expected invalid claim token to stop before creating a user")
	}
	if sessions.createdSession != nil {
		t.Fatal("expected invalid claim token to stop before creating a session")
	}
}

func TestAuthServiceSignUpHashesPasswordBeforeDuplicateEmailCheck(t *testing.T) {
	hashDone := false
	users := &fakeAuthUserRepository{
		findByEmailUser: &domain.User{ID: 9, Email: "owner@example.com"},
		hashDone:        &hashDone,
	}
	sessions := &fakeAuthSessionRepository{}
	authService := NewAuthService(users, sessions, time.Hour, "")
	authService.hashPassword = func(password string) (string, error) {
		hashDone = true
		return "hashed-password", nil
	}

	result, err := authService.SignUp(context.Background(), "owner@example.com", "strong-password", "ko", "")
	if result != nil {
		t.Fatal("expected nil auth result")
	}
	if !errors.Is(err, ErrEmailAlreadyExists) {
		t.Fatalf("expected ErrEmailAlreadyExists, got %v", err)
	}
	if !users.findByEmailCalled {
		t.Fatal("expected duplicate email check to run")
	}
	if !users.hashDoneAtFind {
		t.Fatal("expected password hash to be computed before duplicate email check")
	}
	if users.createCalled {
		t.Fatal("expected duplicate email to stop before creating a user")
	}
	if sessions.createdSession != nil {
		t.Fatal("expected duplicate email to stop before creating a session")
	}
}

func TestAuthServiceSignUpWithEmailVerificationReturnsAcceptedResult(t *testing.T) {
	now := date(2026, time.May, 8)
	users := &fakeAuthUserRepository{findByEmailErr: gorm.ErrRecordNotFound}
	sessions := &fakeAuthSessionRepository{}
	tokens := &fakeEmailVerificationRepository{}
	sender := &fakeVerificationEmailSender{}
	authService := NewAuthService(users, sessions, time.Hour, "")
	authService.now = func() time.Time { return now }
	authService.EnableEmailVerification(tokens, sender, 24*time.Hour)

	result, err := authService.SignUp(context.Background(), "owner@example.com", "strong-password", "en", "")
	if err != nil {
		t.Fatalf("expected signup to succeed, got %v", err)
	}
	if result.Auth != nil {
		t.Fatal("expected signup to wait for email verification")
	}
	if !result.VerificationRequired {
		t.Fatal("expected verification required result")
	}
	if result.Locale != "en" {
		t.Fatalf("expected locale en, got %q", result.Locale)
	}
	if sessions.createdSession != nil {
		t.Fatal("expected no session before email verification")
	}
	if tokens.createdToken == nil {
		t.Fatal("expected verification token to be stored")
	}
	if tokens.createdToken.UserID != 7 {
		t.Fatalf("expected token for user 7, got %d", tokens.createdToken.UserID)
	}
	if !tokens.createdToken.ExpiresAt.Equal(now.Add(24 * time.Hour)) {
		t.Fatalf("expected 24 hour token expiry, got %s", tokens.createdToken.ExpiresAt)
	}
	if sender.to != "owner@example.com" {
		t.Fatalf("expected verification email recipient owner@example.com, got %q", sender.to)
	}
	if sender.locale != "en" {
		t.Fatalf("expected verification email locale en, got %q", sender.locale)
	}
	if sender.token == "" {
		t.Fatal("expected raw verification token to be sent")
	}
	if tokens.createdToken.TokenHash != hashToken(sender.token) {
		t.Fatal("expected stored token hash to match emailed token")
	}
}

func TestAuthServiceSignUpWithEmailVerificationResendsForExistingUnverifiedEmail(t *testing.T) {
	hashDone := false
	users := &fakeAuthUserRepository{
		findByEmailUser: &domain.User{ID: 9, Email: "owner@example.com"},
		hashDone:        &hashDone,
	}
	sessions := &fakeAuthSessionRepository{}
	tokens := &fakeEmailVerificationRepository{}
	sender := &fakeVerificationEmailSender{}
	authService := NewAuthService(users, sessions, time.Hour, "")
	authService.hashPassword = func(password string) (string, error) {
		hashDone = true
		return "hashed-password", nil
	}
	authService.EnableEmailVerification(tokens, sender, 24*time.Hour)

	result, err := authService.SignUp(context.Background(), "owner@example.com", "strong-password", "ko", "")
	if err != nil {
		t.Fatalf("expected duplicate signup to be accepted, got %v", err)
	}
	if result.Auth != nil {
		t.Fatal("expected duplicate signup to avoid creating an auth result")
	}
	if !result.VerificationRequired {
		t.Fatal("expected verification required result")
	}
	if !users.hashDoneAtFind {
		t.Fatal("expected password hash to be computed before duplicate email check")
	}
	if users.createCalled {
		t.Fatal("expected duplicate signup to avoid creating a user")
	}
	if tokens.createdToken == nil {
		t.Fatal("expected duplicate signup to create a new verification token")
	}
	if tokens.createdToken.UserID != 9 {
		t.Fatalf("expected token for existing user 9, got %d", tokens.createdToken.UserID)
	}
	if sender.token == "" {
		t.Fatal("expected duplicate signup to resend a verification token")
	}
	if tokens.createdToken.TokenHash != hashToken(sender.token) {
		t.Fatal("expected stored token hash to match resent token")
	}
	if sessions.createdSession != nil {
		t.Fatal("expected duplicate signup to avoid creating a session")
	}
}

func TestAuthServiceSignUpWithEmailVerificationDoesNotResendForExistingVerifiedEmail(t *testing.T) {
	verifiedAt := date(2026, time.May, 8)
	users := &fakeAuthUserRepository{
		findByEmailUser: &domain.User{
			ID:              9,
			Email:           "owner@example.com",
			EmailVerifiedAt: &verifiedAt,
		},
	}
	sessions := &fakeAuthSessionRepository{}
	tokens := &fakeEmailVerificationRepository{}
	sender := &fakeVerificationEmailSender{}
	authService := NewAuthService(users, sessions, time.Hour, "")
	authService.EnableEmailVerification(tokens, sender, 24*time.Hour)

	result, err := authService.SignUp(context.Background(), "owner@example.com", "strong-password", "ko", "")
	if err != nil {
		t.Fatalf("expected duplicate signup to be accepted, got %v", err)
	}
	if result.Auth != nil {
		t.Fatal("expected duplicate signup to avoid creating an auth result")
	}
	if !result.VerificationRequired {
		t.Fatal("expected verification required result")
	}
	if users.createCalled {
		t.Fatal("expected duplicate signup to avoid creating a user")
	}
	if tokens.createdToken != nil {
		t.Fatal("expected verified duplicate signup to avoid creating a new verification token")
	}
	if sender.token != "" {
		t.Fatal("expected verified duplicate signup to avoid sending a verification token")
	}
	if sessions.createdSession != nil {
		t.Fatal("expected duplicate signup to avoid creating a session")
	}
}

func TestAuthServiceSignUpWithEmailVerificationCanRecoverAfterSendFailure(t *testing.T) {
	users := &fakeAuthUserRepository{findByEmailErr: gorm.ErrRecordNotFound}
	sessions := &fakeAuthSessionRepository{}
	tokens := &fakeEmailVerificationRepository{}
	sender := &fakeVerificationEmailSender{err: errors.New("smtp unavailable")}
	authService := NewAuthService(users, sessions, time.Hour, "")
	authService.EnableEmailVerification(tokens, sender, 24*time.Hour)

	result, err := authService.SignUp(context.Background(), "owner@example.com", "strong-password", "en", "")
	if result != nil {
		t.Fatal("expected failed email send to return nil result")
	}
	if err == nil {
		t.Fatal("expected failed email send to return an error")
	}
	if !users.createCalled {
		t.Fatal("expected user creation before send failure")
	}
	if tokens.createdToken == nil {
		t.Fatal("expected failed send to have created a recoverable token record")
	}

	retryUsers := &fakeAuthUserRepository{
		findByEmailUser: &domain.User{ID: 7, Email: "owner@example.com", Locale: "en"},
	}
	retryTokens := &fakeEmailVerificationRepository{}
	retrySender := &fakeVerificationEmailSender{}
	retryService := NewAuthService(retryUsers, sessions, time.Hour, "")
	retryService.EnableEmailVerification(retryTokens, retrySender, 24*time.Hour)

	retryResult, retryErr := retryService.SignUp(context.Background(), "owner@example.com", "strong-password", "en", "")
	if retryErr != nil {
		t.Fatalf("expected retry signup to resend verification, got %v", retryErr)
	}
	if retryResult.Auth != nil {
		t.Fatal("expected retry signup to avoid creating an auth result")
	}
	if !retryResult.VerificationRequired {
		t.Fatal("expected retry signup to require verification")
	}
	if retryUsers.createCalled {
		t.Fatal("expected retry signup to avoid creating another user")
	}
	if retryTokens.createdToken == nil {
		t.Fatal("expected retry signup to create a new verification token")
	}
	if retrySender.token == "" {
		t.Fatal("expected retry signup to resend a verification email")
	}
}

func TestAuthServiceLoginRejectsUnverifiedEmailWhenVerificationEnabled(t *testing.T) {
	passwordHash, err := bcryptHashPassword("strong-password")
	if err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}

	users := &fakeAuthUserRepository{
		findByEmailUser: &domain.User{
			ID:           7,
			Email:        "owner@example.com",
			PasswordHash: passwordHash,
			Locale:       "ko",
		},
	}
	sessions := &fakeAuthSessionRepository{}
	authService := NewAuthService(users, sessions, time.Hour, "")
	authService.EnableEmailVerification(&fakeEmailVerificationRepository{}, &fakeVerificationEmailSender{}, 24*time.Hour)

	result, err := authService.Login(context.Background(), "owner@example.com", "strong-password")
	if result != nil {
		t.Fatal("expected nil auth result")
	}
	if !errors.Is(err, ErrEmailNotVerified) {
		t.Fatalf("expected ErrEmailNotVerified, got %v", err)
	}
	if sessions.createdSession != nil {
		t.Fatal("expected unverified user to stop before creating a session")
	}
}

func TestAuthServiceVerifyEmailConsumesTokenAndCreatesSession(t *testing.T) {
	now := date(2026, time.May, 8)
	verifiedAt := now
	sessions := &fakeAuthSessionRepository{}
	tokens := &fakeEmailVerificationRepository{
		consumeUser: &domain.User{
			ID:              7,
			Email:           "owner@example.com",
			EmailVerifiedAt: &verifiedAt,
			Locale:          "en",
		},
	}
	authService := NewAuthService(&fakeAuthUserRepository{}, sessions, time.Hour, "")
	authService.now = func() time.Time { return now }
	authService.EnableEmailVerification(tokens, &fakeVerificationEmailSender{}, 24*time.Hour)

	result, err := authService.VerifyEmail(context.Background(), "raw-token")
	if err != nil {
		t.Fatalf("expected email verification to succeed, got %v", err)
	}
	if result == nil {
		t.Fatal("expected auth result")
	}
	if tokens.consumedHash != hashToken("raw-token") {
		t.Fatal("expected verification token hash to be consumed")
	}
	if !tokens.consumedAt.Equal(now) {
		t.Fatalf("expected consume time %s, got %s", now, tokens.consumedAt)
	}
	if sessions.createdSession == nil {
		t.Fatal("expected session to be created")
	}
	if result.User.ID != 7 {
		t.Fatalf("expected verified user ID 7, got %d", result.User.ID)
	}
}

type fakeAuthUserRepository struct {
	findByEmailUser   *domain.User
	findByEmailErr    error
	createErr         error
	createCalled      bool
	findByEmailCalled bool
	hashDone          *bool
	hashDoneAtFind    bool
	email             string
	locale            string
	passwordHash      string
	claimLegacy       bool
}

func (r *fakeAuthUserRepository) CreateWithPassword(ctx context.Context, email, passwordHash, locale string, claimLegacy bool, emailVerifiedAt *time.Time) (*domain.User, error) {
	_ = ctx
	r.createCalled = true
	r.email = email
	r.locale = locale
	r.passwordHash = passwordHash
	r.claimLegacy = claimLegacy
	if r.createErr != nil {
		return nil, r.createErr
	}

	return &domain.User{
		ID:              7,
		Username:        email,
		Email:           email,
		PasswordHash:    passwordHash,
		Locale:          locale,
		EmailVerifiedAt: emailVerifiedAt,
		CreatedAt:       date(2026, time.May, 8),
	}, nil
}

func (r *fakeAuthUserRepository) FindByEmail(ctx context.Context, email string) (*domain.User, error) {
	_ = ctx
	_ = email
	r.findByEmailCalled = true
	if r.hashDone != nil {
		r.hashDoneAtFind = *r.hashDone
	}
	if r.findByEmailUser != nil {
		return r.findByEmailUser, nil
	}
	if r.findByEmailErr != nil {
		return nil, r.findByEmailErr
	}

	return nil, gorm.ErrRecordNotFound
}

func (r *fakeAuthUserRepository) FindByID(ctx context.Context, id uint) (*domain.User, error) {
	_ = ctx
	_ = id
	return nil, gorm.ErrRecordNotFound
}

func (r *fakeAuthUserRepository) UpdateLocale(ctx context.Context, id uint, locale string) (*domain.User, error) {
	_ = ctx
	_ = id
	_ = locale
	return nil, gorm.ErrRecordNotFound
}

type fakeAuthSessionRepository struct {
	createdSession *domain.Session
	createErr      error
}

type fakeEmailVerificationRepository struct {
	createdToken *domain.EmailVerificationToken
	createErr    error
	consumeUser  *domain.User
	consumeErr   error
	consumedHash string
	consumedAt   time.Time
}

func (r *fakeEmailVerificationRepository) Create(ctx context.Context, token *domain.EmailVerificationToken) error {
	_ = ctx
	if r.createErr != nil {
		return r.createErr
	}

	tokenCopy := *token
	r.createdToken = &tokenCopy
	return nil
}

func (r *fakeEmailVerificationRepository) Consume(ctx context.Context, tokenHash string, now time.Time) (*domain.User, error) {
	_ = ctx
	r.consumedHash = tokenHash
	r.consumedAt = now
	if r.consumeErr != nil {
		return nil, r.consumeErr
	}
	if r.consumeUser != nil {
		return r.consumeUser, nil
	}

	return nil, gorm.ErrRecordNotFound
}

type fakeVerificationEmailSender struct {
	to     string
	locale string
	token  string
	err    error
}

func (s *fakeVerificationEmailSender) SendVerificationEmail(ctx context.Context, to, locale, token string) error {
	_ = ctx
	s.to = to
	s.locale = locale
	s.token = token
	return s.err
}

func (r *fakeAuthSessionRepository) Create(ctx context.Context, session *domain.Session) error {
	_ = ctx
	if r.createErr != nil {
		return r.createErr
	}

	session.ID = 17
	r.createdSession = session
	return nil
}

func (r *fakeAuthSessionRepository) FindByTokenHash(ctx context.Context, tokenHash string, now time.Time) (*domain.Session, error) {
	_ = ctx
	_ = tokenHash
	_ = now
	return nil, gorm.ErrRecordNotFound
}

func (r *fakeAuthSessionRepository) DeleteByTokenHash(ctx context.Context, tokenHash string) error {
	_ = ctx
	_ = tokenHash
	return nil
}

func (r *fakeAuthSessionRepository) UpdateLastUsedAt(ctx context.Context, id uint, lastUsedAt time.Time) error {
	_ = ctx
	_ = id
	_ = lastUsedAt
	return nil
}

func (r *fakeAuthSessionRepository) UpdateCSRFTokenHash(ctx context.Context, id uint, csrfTokenHash string) error {
	_ = ctx
	_ = id
	_ = csrfTokenHash
	return nil
}
