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
	if result.User.ID != 7 {
		t.Fatalf("expected created user ID 7, got %d", result.User.ID)
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

type fakeAuthUserRepository struct {
	findByEmailUser *domain.User
	findByEmailErr  error
	createErr       error
	createCalled    bool
	email           string
	locale          string
	passwordHash    string
	claimLegacy     bool
}

func (r *fakeAuthUserRepository) CreateWithPassword(ctx context.Context, email, passwordHash, locale string, claimLegacy bool) (*domain.User, error) {
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
		ID:           7,
		Username:     email,
		Email:        email,
		PasswordHash: passwordHash,
		Locale:       locale,
		CreatedAt:    date(2026, time.May, 8),
	}, nil
}

func (r *fakeAuthUserRepository) FindByEmail(ctx context.Context, email string) (*domain.User, error) {
	_ = ctx
	_ = email
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
