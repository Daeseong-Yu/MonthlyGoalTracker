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

func TestAuthServiceRequestPasswordResetSendsTokenForVerifiedEmail(t *testing.T) {
	now := date(2026, time.May, 8)
	verifiedAt := now.Add(-time.Hour)
	users := &fakeAuthUserRepository{
		findByEmailUser: &domain.User{
			ID:              7,
			Email:           "owner@example.com",
			PasswordHash:    "existing-password-hash",
			EmailVerifiedAt: &verifiedAt,
			Locale:          "en",
		},
	}
	resetTokens := &fakePasswordResetRepository{}
	resetSender := &fakePasswordResetEmailSender{}
	authService := NewAuthService(users, &fakeAuthSessionRepository{}, time.Hour, "")
	authService.now = func() time.Time { return now }
	authService.EnableEmailVerification(&fakeEmailVerificationRepository{}, &fakeVerificationEmailSender{}, 24*time.Hour)
	authService.EnablePasswordReset(resetTokens, resetSender, time.Hour)

	result, err := authService.RequestPasswordReset(context.Background(), " Owner@Example.com ", "en")
	if err != nil {
		t.Fatalf("expected password reset request to succeed, got %v", err)
	}
	if result.Locale != "en" {
		t.Fatalf("expected locale en, got %q", result.Locale)
	}
	if users.findByEmailEmail != "owner@example.com" {
		t.Fatalf("expected normalized lookup email, got %q", users.findByEmailEmail)
	}
	if resetTokens.createdToken == nil {
		t.Fatal("expected password reset token to be stored")
	}
	if resetTokens.createdToken.UserID != 7 {
		t.Fatalf("expected token for user 7, got %d", resetTokens.createdToken.UserID)
	}
	if !resetTokens.createdToken.ExpiresAt.Equal(now.Add(time.Hour)) {
		t.Fatalf("expected 1 hour token expiry, got %s", resetTokens.createdToken.ExpiresAt)
	}
	if resetSender.to != "owner@example.com" {
		t.Fatalf("expected reset email recipient owner@example.com, got %q", resetSender.to)
	}
	if resetSender.locale != "en" {
		t.Fatalf("expected reset email locale en, got %q", resetSender.locale)
	}
	if resetSender.token == "" {
		t.Fatal("expected raw password reset token to be sent")
	}
	if resetTokens.createdToken.TokenHash != hashToken(resetSender.token) {
		t.Fatal("expected stored reset token hash to match emailed token")
	}
}

func TestAuthServiceRequestPasswordResetHidesMissingOrUnverifiedEmail(t *testing.T) {
	verifiedAt := date(2026, time.May, 8)
	testCases := []struct {
		name string
		user *domain.User
	}{
		{
			name: "missing email",
		},
		{
			name: "unverified email",
			user: &domain.User{
				ID:           7,
				Email:        "owner@example.com",
				PasswordHash: "existing-password-hash",
			},
		},
		{
			name: "passwordless account",
			user: &domain.User{
				ID:              7,
				Email:           "owner@example.com",
				EmailVerifiedAt: &verifiedAt,
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			users := &fakeAuthUserRepository{findByEmailUser: testCase.user}
			resetTokens := &fakePasswordResetRepository{}
			resetSender := &fakePasswordResetEmailSender{}
			authService := NewAuthService(users, &fakeAuthSessionRepository{}, time.Hour, "")
			authService.EnableEmailVerification(&fakeEmailVerificationRepository{}, &fakeVerificationEmailSender{}, 24*time.Hour)
			authService.EnablePasswordReset(resetTokens, resetSender, time.Hour)

			result, err := authService.RequestPasswordReset(context.Background(), "owner@example.com", "ko")
			if err != nil {
				t.Fatalf("expected request to hide account state, got %v", err)
			}
			if result.Locale != "ko" {
				t.Fatalf("expected locale ko, got %q", result.Locale)
			}
			if resetTokens.createdToken != nil {
				t.Fatal("expected no token for hidden account state")
			}
			if resetSender.token != "" {
				t.Fatal("expected no reset email for hidden account state")
			}
		})
	}
}

func TestAuthServiceResetPasswordConsumesTokenInvalidatesSessionsAndCreatesSession(t *testing.T) {
	now := date(2026, time.May, 8)
	sessions := &fakeAuthSessionRepository{}
	resetTokens := &fakePasswordResetRepository{
		consumeUser: &domain.User{
			ID:     7,
			Email:  "owner@example.com",
			Locale: "ko",
		},
	}
	authService := NewAuthService(&fakeAuthUserRepository{}, sessions, time.Hour, "")
	authService.now = func() time.Time { return now }
	authService.hashPassword = func(password string) (string, error) {
		if password != "new-secret" {
			t.Fatalf("expected password new-secret, got %q", password)
		}
		return "hashed-new-secret", nil
	}
	authService.EnablePasswordReset(resetTokens, &fakePasswordResetEmailSender{}, time.Hour)

	result, err := authService.ResetPassword(context.Background(), " raw-token ", "new-secret")
	if err != nil {
		t.Fatalf("expected password reset to succeed, got %v", err)
	}
	if resetTokens.consumedHash != hashToken("raw-token") {
		t.Fatal("expected reset token hash to be consumed")
	}
	if resetTokens.consumedPasswordHash != "hashed-new-secret" {
		t.Fatalf("expected consumed password hash, got %q", resetTokens.consumedPasswordHash)
	}
	if !resetTokens.consumedAt.Equal(now) {
		t.Fatalf("expected consume time %s, got %s", now, resetTokens.consumedAt)
	}
	if sessions.deletedUserID != 7 {
		t.Fatalf("expected existing sessions for user 7 to be deleted, got %d", sessions.deletedUserID)
	}
	if sessions.createdSession == nil {
		t.Fatal("expected new session to be created")
	}
	if result.User.PasswordHash != "hashed-new-secret" {
		t.Fatalf("expected returned user to have updated password hash, got %q", result.User.PasswordHash)
	}
}

func TestAuthServiceResetPasswordRejectsInvalidToken(t *testing.T) {
	sessions := &fakeAuthSessionRepository{}
	resetTokens := &fakePasswordResetRepository{consumeErr: gorm.ErrRecordNotFound}
	authService := NewAuthService(&fakeAuthUserRepository{}, sessions, time.Hour, "")
	authService.hashPassword = func(password string) (string, error) {
		return "hashed-new-secret", nil
	}
	authService.EnablePasswordReset(resetTokens, &fakePasswordResetEmailSender{}, time.Hour)

	result, err := authService.ResetPassword(context.Background(), "raw-token", "new-secret")
	if result != nil {
		t.Fatal("expected nil auth result")
	}
	if !errors.Is(err, ErrInvalidPasswordResetToken) {
		t.Fatalf("expected ErrInvalidPasswordResetToken, got %v", err)
	}
	if sessions.deletedUserID != 0 {
		t.Fatal("expected invalid token to avoid deleting sessions")
	}
	if sessions.createdSession != nil {
		t.Fatal("expected invalid token to avoid creating a session")
	}
}

func TestAuthServiceChangePasswordVerifiesCurrentPasswordUpdatesHashAndRefreshesSession(t *testing.T) {
	currentPasswordHash, err := bcryptHashPassword("current-secret")
	if err != nil {
		t.Fatalf("failed to hash current password: %v", err)
	}

	users := &fakeAuthUserRepository{
		findByIDUser: &domain.User{
			ID:           7,
			Email:        "owner@example.com",
			PasswordHash: currentPasswordHash,
			Locale:       "ko",
		},
	}
	sessions := &fakeAuthSessionRepository{}
	authService := NewAuthService(users, sessions, time.Hour, "")
	authService.hashPassword = func(password string) (string, error) {
		if password != "new-secret" {
			t.Fatalf("expected password new-secret, got %q", password)
		}
		return "hashed-new-secret", nil
	}

	result, err := authService.ChangePassword(context.Background(), 7, "current-secret", "new-secret")
	if err != nil {
		t.Fatalf("expected password change to succeed, got %v", err)
	}
	if users.findByIDID != 7 {
		t.Fatalf("expected user 7 lookup, got %d", users.findByIDID)
	}
	if !users.updatePasswordHashCalled {
		t.Fatal("expected password hash update")
	}
	if users.updatePasswordHashID != 7 {
		t.Fatalf("expected password hash update for user 7, got %d", users.updatePasswordHashID)
	}
	if users.updatedPasswordHash != "hashed-new-secret" {
		t.Fatalf("expected updated password hash, got %q", users.updatedPasswordHash)
	}
	if !users.replaceSessionsCalled {
		t.Fatal("expected password hash update and session replacement")
	}
	if users.replaceSessionsID != 7 {
		t.Fatalf("expected session replacement for user 7, got %d", users.replaceSessionsID)
	}
	if users.replacedSession == nil {
		t.Fatal("expected replacement session")
	}
	if users.replacedSession.UserID != 7 {
		t.Fatalf("expected replacement session for user 7, got %d", users.replacedSession.UserID)
	}
	if result.User.PasswordHash != "hashed-new-secret" {
		t.Fatalf("expected returned user to have updated password hash, got %q", result.User.PasswordHash)
	}
	if result.Token == "" || result.CSRFToken == "" {
		t.Fatal("expected new session and CSRF tokens")
	}
}

func TestAuthServiceChangePasswordRejectsWrongCurrentPassword(t *testing.T) {
	currentPasswordHash, err := bcryptHashPassword("current-secret")
	if err != nil {
		t.Fatalf("failed to hash current password: %v", err)
	}

	users := &fakeAuthUserRepository{
		findByIDUser: &domain.User{
			ID:           7,
			Email:        "owner@example.com",
			PasswordHash: currentPasswordHash,
			Locale:       "ko",
		},
	}
	sessions := &fakeAuthSessionRepository{}
	authService := NewAuthService(users, sessions, time.Hour, "")
	authService.hashPassword = func(password string) (string, error) {
		t.Fatalf("expected wrong current password to stop before hashing new password, got %q", password)
		return "", nil
	}

	result, err := authService.ChangePassword(context.Background(), 7, "wrong-secret", "new-secret")
	if result != nil {
		t.Fatal("expected nil auth result")
	}
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials, got %v", err)
	}
	if users.updatePasswordHashCalled {
		t.Fatal("expected wrong current password to avoid updating password hash")
	}
	if users.replaceSessionsCalled {
		t.Fatal("expected wrong current password to avoid replacing sessions")
	}
	if sessions.deletedUserID != 0 {
		t.Fatal("expected wrong current password to avoid deleting sessions")
	}
	if sessions.createdSession != nil {
		t.Fatal("expected wrong current password to avoid creating a session")
	}
}

func TestAuthServiceChangePasswordRejectsWeakNewPasswordBeforeLookup(t *testing.T) {
	users := &fakeAuthUserRepository{}
	sessions := &fakeAuthSessionRepository{}
	authService := NewAuthService(users, sessions, time.Hour, "")

	result, err := authService.ChangePassword(context.Background(), 7, "current-secret", "short")
	if result != nil {
		t.Fatal("expected nil auth result")
	}
	if !errors.Is(err, ErrWeakPassword) {
		t.Fatalf("expected ErrWeakPassword, got %v", err)
	}
	if users.findByIDCalled {
		t.Fatal("expected weak new password to stop before user lookup")
	}
	if sessions.createdSession != nil {
		t.Fatal("expected weak new password to avoid creating a session")
	}
}

func TestAuthServiceLogoutOtherSessionsKeepsCurrentSession(t *testing.T) {
	sessions := &fakeAuthSessionRepository{}
	authService := NewAuthService(&fakeAuthUserRepository{}, sessions, time.Hour, "")

	err := authService.LogoutOtherSessions(context.Background(), 7, " current-token ")
	if err != nil {
		t.Fatalf("expected other sessions logout to succeed, got %v", err)
	}
	if sessions.deletedOtherSessionsUserID != 7 {
		t.Fatalf("expected user 7 sessions to be deleted, got %d", sessions.deletedOtherSessionsUserID)
	}
	if sessions.deletedOtherSessionsTokenHash == "" {
		t.Fatal("expected current session token hash to be recorded")
	}
	if sessions.deletedOtherSessionsTokenHash == "current-token" {
		t.Fatal("expected raw current token not to be passed to the repository")
	}
	if sessions.deletedOtherSessionsTokenHash != hashToken("current-token") {
		t.Fatal("expected current token hash to exclude the current session")
	}
}

func TestAuthServiceLogoutOtherSessionsRejectsBlankToken(t *testing.T) {
	sessions := &fakeAuthSessionRepository{}
	authService := NewAuthService(&fakeAuthUserRepository{}, sessions, time.Hour, "")

	err := authService.LogoutOtherSessions(context.Background(), 7, "   ")
	if !errors.Is(err, ErrInvalidSession) {
		t.Fatalf("expected ErrInvalidSession, got %v", err)
	}
	if sessions.deletedOtherSessionsUserID != 0 {
		t.Fatalf("expected blank token to avoid deleting sessions, got user %d", sessions.deletedOtherSessionsUserID)
	}
	if sessions.deletedOtherSessionsTokenHash != "" {
		t.Fatal("expected blank token to avoid sending a token hash")
	}
}

type fakeAuthUserRepository struct {
	findByEmailUser          *domain.User
	findByEmailErr           error
	findByIDUser             *domain.User
	findByIDErr              error
	createErr                error
	createCalled             bool
	findByEmailCalled        bool
	findByIDCalled           bool
	hashDone                 *bool
	hashDoneAtFind           bool
	email                    string
	findByEmailEmail         string
	findByIDID               uint
	locale                   string
	passwordHash             string
	claimLegacy              bool
	updatePasswordHashCalled bool
	updatePasswordHashID     uint
	updatedPasswordHash      string
	updatePasswordHashErr    error
	replaceSessionsCalled    bool
	replaceSessionsID        uint
	replacedSession          *domain.Session
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
	r.findByEmailCalled = true
	r.findByEmailEmail = email
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
	r.findByIDCalled = true
	r.findByIDID = id
	if r.findByIDErr != nil {
		return nil, r.findByIDErr
	}
	if r.findByIDUser != nil {
		return r.findByIDUser, nil
	}

	return nil, gorm.ErrRecordNotFound
}

func (r *fakeAuthUserRepository) UpdateLocale(ctx context.Context, id uint, locale string) (*domain.User, error) {
	_ = ctx
	_ = id
	_ = locale
	return nil, gorm.ErrRecordNotFound
}

func (r *fakeAuthUserRepository) UpdatePasswordHashAndReplaceSessions(ctx context.Context, id uint, passwordHash string, session *domain.Session) (*domain.User, error) {
	_ = ctx
	r.updatePasswordHashCalled = true
	r.updatePasswordHashID = id
	r.updatedPasswordHash = passwordHash
	r.replaceSessionsCalled = true
	r.replaceSessionsID = id
	if session != nil {
		sessionCopy := *session
		r.replacedSession = &sessionCopy
	}
	if r.updatePasswordHashErr != nil {
		return nil, r.updatePasswordHashErr
	}
	if r.findByIDUser != nil {
		user := *r.findByIDUser
		user.PasswordHash = passwordHash
		if session != nil {
			session.UserID = user.ID
			session.User = user
		}
		return &user, nil
	}

	return &domain.User{
		ID:           id,
		Email:        r.email,
		PasswordHash: passwordHash,
		Locale:       r.locale,
	}, nil
}

type fakeAuthSessionRepository struct {
	createdSession                *domain.Session
	createErr                     error
	deletedUserID                 uint
	deletedOtherSessionsUserID    uint
	deletedOtherSessionsTokenHash string
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

type fakePasswordResetRepository struct {
	createdToken         *domain.PasswordResetToken
	createErr            error
	consumeUser          *domain.User
	consumeErr           error
	consumedHash         string
	consumedPasswordHash string
	consumedAt           time.Time
}

func (r *fakePasswordResetRepository) Create(ctx context.Context, token *domain.PasswordResetToken) error {
	_ = ctx
	if r.createErr != nil {
		return r.createErr
	}

	tokenCopy := *token
	r.createdToken = &tokenCopy
	return nil
}

func (r *fakePasswordResetRepository) Consume(ctx context.Context, tokenHash, passwordHash string, now time.Time) (*domain.User, error) {
	_ = ctx
	r.consumedHash = tokenHash
	r.consumedPasswordHash = passwordHash
	r.consumedAt = now
	if r.consumeErr != nil {
		return nil, r.consumeErr
	}
	if r.consumeUser != nil {
		user := *r.consumeUser
		user.PasswordHash = passwordHash
		return &user, nil
	}

	return nil, gorm.ErrRecordNotFound
}

type fakePasswordResetEmailSender struct {
	to     string
	locale string
	token  string
	err    error
}

func (s *fakePasswordResetEmailSender) SendPasswordResetEmail(ctx context.Context, to, locale, token string) error {
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

func (r *fakeAuthSessionRepository) DeleteByUserID(ctx context.Context, userID uint) error {
	_ = ctx
	r.deletedUserID = userID
	return nil
}

func (r *fakeAuthSessionRepository) DeleteOthersByUserIDAndTokenHash(ctx context.Context, userID uint, tokenHash string) error {
	_ = ctx
	r.deletedOtherSessionsUserID = userID
	r.deletedOtherSessionsTokenHash = tokenHash
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
