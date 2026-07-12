package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"net/mail"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/Daeseong-Yu/MonthlyGoalTracker/backend/internal/domain"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

var (
	ErrInvalidEmail              = errors.New("invalid email")
	ErrWeakPassword              = errors.New("weak password")
	ErrEmailAlreadyExists        = errors.New("email already exists")
	ErrInvalidCredentials        = errors.New("invalid credentials")
	ErrEmailNotVerified          = errors.New("email not verified")
	ErrInvalidLocale             = errors.New("invalid locale")
	ErrInvalidSession            = errors.New("invalid session")
	ErrInvalidVerificationToken  = errors.New("invalid verification token")
	ErrInvalidPasswordResetToken = errors.New("invalid password reset token")
	ErrInvalidLegacyClaim        = errors.New("invalid legacy claim")
	ErrLegacyClaimRequired       = domain.ErrLegacyClaimRequired
)

const (
	minPasswordCharacters = 8
	maxPasswordBytes      = 72
)

type AuthUserRepository interface {
	CreateWithPassword(ctx context.Context, email, passwordHash, locale string, claimLegacy bool, emailVerifiedAt *time.Time) (*domain.User, error)
	FindByEmail(ctx context.Context, email string) (*domain.User, error)
	FindByID(ctx context.Context, id uint) (*domain.User, error)
	UpdateLocale(ctx context.Context, id uint, locale string) (*domain.User, error)
	UpdatePasswordHashAndReplaceSessions(ctx context.Context, id uint, passwordHash string, session *domain.Session) (*domain.User, error)
}

type AuthSessionRepository interface {
	Create(ctx context.Context, session *domain.Session) error
	FindByTokenHash(ctx context.Context, tokenHash string, now time.Time) (*domain.Session, error)
	DeleteByTokenHash(ctx context.Context, tokenHash string) error
	DeleteByUserID(ctx context.Context, userID uint) error
	DeleteOthersByUserIDAndTokenHash(ctx context.Context, userID uint, tokenHash string) error
	UpdateLastUsedAt(ctx context.Context, id uint, lastUsedAt time.Time) error
	UpdateCSRFTokenHash(ctx context.Context, id uint, csrfTokenHash string) error
}

type AuthEmailVerificationRepository interface {
	Create(ctx context.Context, token *domain.EmailVerificationToken) error
	Consume(ctx context.Context, tokenHash string, now time.Time) (*domain.User, error)
}

type AuthPasswordResetRepository interface {
	Create(ctx context.Context, token *domain.PasswordResetToken) error
	Consume(ctx context.Context, tokenHash, passwordHash string, now time.Time) (*domain.User, error)
}

type VerificationEmailSender interface {
	SendVerificationEmail(ctx context.Context, to, locale, token string) error
}

type PasswordResetEmailSender interface {
	SendPasswordResetEmail(ctx context.Context, to, locale, token string) error
}

type AuthResult struct {
	User      domain.User
	Session   domain.Session
	Token     string
	CSRFToken string
}

type SignupResult struct {
	Auth                 *AuthResult
	VerificationRequired bool
	Locale               string
}

type PasswordResetRequestResult struct {
	Locale string
}

type AuthService struct {
	users                    AuthUserRepository
	sessions                 AuthSessionRepository
	emailVerificationTokens  AuthEmailVerificationRepository
	verificationEmailSender  VerificationEmailSender
	passwordResetTokens      AuthPasswordResetRepository
	passwordResetEmailSender PasswordResetEmailSender
	ttl                      time.Duration
	emailVerificationTTL     time.Duration
	passwordResetTTL         time.Duration
	now                      func() time.Time
	hashPassword             func(password string) (string, error)
	legacyClaimTokenHash     string
}

func NewAuthService(users AuthUserRepository, sessions AuthSessionRepository, ttl time.Duration, legacyClaimToken string) *AuthService {
	authService := &AuthService{
		users:        users,
		sessions:     sessions,
		ttl:          ttl,
		now:          time.Now,
		hashPassword: bcryptHashPassword,
	}

	if token := strings.TrimSpace(legacyClaimToken); token != "" {
		authService.legacyClaimTokenHash = hashToken(token)
	}

	return authService
}

func (s *AuthService) EnableEmailVerification(tokens AuthEmailVerificationRepository, sender VerificationEmailSender, ttl time.Duration) {
	s.emailVerificationTokens = tokens
	if sender == nil {
		sender = noopVerificationEmailSender{}
	}
	s.verificationEmailSender = sender
	s.emailVerificationTTL = ttl
}

func (s *AuthService) EnablePasswordReset(tokens AuthPasswordResetRepository, sender PasswordResetEmailSender, ttl time.Duration) {
	s.passwordResetTokens = tokens
	if sender == nil {
		sender = noopPasswordResetEmailSender{}
	}
	s.passwordResetEmailSender = sender
	s.passwordResetTTL = ttl
}

func (s *AuthService) SignUp(ctx context.Context, email, password, locale, legacyClaimToken string) (*SignupResult, error) {
	normalizedEmail, err := normalizeEmailAddress(email)
	if err != nil {
		return nil, err
	}
	if err := validatePassword(password); err != nil {
		return nil, err
	}
	normalizedLocale, err := NormalizeLocale(locale)
	if err != nil {
		return nil, err
	}
	claimLegacy, err := s.shouldClaimLegacy(legacyClaimToken)
	if err != nil {
		return nil, err
	}

	passwordHash, err := s.hashPassword(password)
	if err != nil {
		return nil, err
	}

	existingUser, err := s.users.FindByEmail(ctx, normalizedEmail)
	if err == nil && existingUser.ID != 0 {
		if s.emailVerificationEnabled() {
			if existingUser.EmailVerifiedAt == nil {
				if err := s.createEmailVerificationToken(ctx, *existingUser, normalizedLocale); err != nil {
					return nil, err
				}
			}

			return &SignupResult{
				VerificationRequired: true,
				Locale:               normalizedLocale,
			}, nil
		}
		return nil, ErrEmailAlreadyExists
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	var emailVerifiedAt *time.Time
	if !s.emailVerificationEnabled() {
		verifiedAt := s.now()
		emailVerifiedAt = &verifiedAt
	}

	user, err := s.users.CreateWithPassword(ctx, normalizedEmail, passwordHash, normalizedLocale, claimLegacy, emailVerifiedAt)
	if err != nil {
		return nil, err
	}

	if s.emailVerificationEnabled() {
		if err := s.createEmailVerificationToken(ctx, *user, normalizedLocale); err != nil {
			return nil, err
		}

		return &SignupResult{
			VerificationRequired: true,
			Locale:               normalizedLocale,
		}, nil
	}

	authResult, err := s.createSession(ctx, *user)
	if err != nil {
		return nil, err
	}

	return &SignupResult{
		Auth:   authResult,
		Locale: normalizedLocale,
	}, nil
}

func bcryptHashPassword(password string) (string, error) {
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}

	return string(passwordHash), nil
}

func validatePassword(password string) error {
	if utf8.RuneCountInString(password) < minPasswordCharacters || len(password) > maxPasswordBytes {
		return ErrWeakPassword
	}

	return nil
}

func (s *AuthService) shouldClaimLegacy(legacyClaimToken string) (bool, error) {
	token := strings.TrimSpace(legacyClaimToken)
	if token == "" {
		return false, nil
	}
	if s.legacyClaimTokenHash == "" {
		return false, ErrInvalidLegacyClaim
	}

	tokenHash := hashToken(token)
	if subtle.ConstantTimeCompare([]byte(tokenHash), []byte(s.legacyClaimTokenHash)) != 1 {
		return false, ErrInvalidLegacyClaim
	}

	return true, nil
}

func (s *AuthService) Login(ctx context.Context, email, password string) (*AuthResult, error) {
	normalizedEmail, err := normalizeEmailAddress(email)
	if err != nil {
		return nil, ErrInvalidCredentials
	}

	user, err := s.users.FindByEmail(ctx, normalizedEmail)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrInvalidCredentials
	}
	if err != nil {
		return nil, err
	}
	if user.PasswordHash == "" {
		return nil, ErrInvalidCredentials
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, ErrInvalidCredentials
	}
	if s.emailVerificationEnabled() && user.EmailVerifiedAt == nil {
		return nil, ErrEmailNotVerified
	}

	return s.createSession(ctx, *user)
}

func (s *AuthService) RequestPasswordReset(ctx context.Context, email, locale string) (*PasswordResetRequestResult, error) {
	normalizedEmail, err := normalizeEmailAddress(email)
	if err != nil {
		return nil, err
	}
	normalizedLocale, err := NormalizeLocale(locale)
	if err != nil {
		return nil, err
	}

	result := &PasswordResetRequestResult{Locale: normalizedLocale}
	if !s.passwordResetEnabled() {
		return result, nil
	}

	user, err := s.users.FindByEmail(ctx, normalizedEmail)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return result, nil
	}
	if err != nil {
		return nil, err
	}
	if user.PasswordHash == "" || user.EmailVerifiedAt == nil {
		return result, nil
	}

	if err := s.createPasswordResetToken(ctx, *user, normalizedLocale); err != nil {
		return nil, err
	}

	return result, nil
}

func (s *AuthService) ResetPassword(ctx context.Context, token, password string) (*AuthResult, error) {
	if !s.passwordResetEnabled() || strings.TrimSpace(token) == "" {
		return nil, ErrInvalidPasswordResetToken
	}
	if err := validatePassword(password); err != nil {
		return nil, err
	}

	passwordHash, err := s.hashPassword(password)
	if err != nil {
		return nil, err
	}

	user, err := s.passwordResetTokens.Consume(ctx, hashToken(strings.TrimSpace(token)), passwordHash, s.now())
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrInvalidPasswordResetToken
	}
	if err != nil {
		return nil, err
	}
	if err := s.sessions.DeleteByUserID(ctx, user.ID); err != nil {
		return nil, err
	}

	return s.createSession(ctx, *user)
}

func (s *AuthService) ChangePassword(ctx context.Context, userID uint, currentPassword, newPassword string) (*AuthResult, error) {
	if err := validatePassword(newPassword); err != nil {
		return nil, err
	}

	user, err := s.users.FindByID(ctx, userID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrInvalidSession
	}
	if err != nil {
		return nil, err
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(currentPassword)); err != nil {
		return nil, ErrInvalidCredentials
	}

	passwordHash, err := s.hashPassword(newPassword)
	if err != nil {
		return nil, err
	}

	session, sessionToken, csrfToken, err := s.newSession(*user)
	if err != nil {
		return nil, err
	}

	updatedUser, err := s.users.UpdatePasswordHashAndReplaceSessions(ctx, user.ID, passwordHash, &session)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrInvalidSession
	}
	if err != nil {
		return nil, err
	}

	session.User = *updatedUser
	return &AuthResult{
		User:      *updatedUser,
		Session:   session,
		Token:     sessionToken,
		CSRFToken: csrfToken,
	}, nil
}

func (s *AuthService) VerifyEmail(ctx context.Context, token string) (*AuthResult, error) {
	if !s.emailVerificationEnabled() || strings.TrimSpace(token) == "" {
		return nil, ErrInvalidVerificationToken
	}

	now := s.now()
	user, err := s.emailVerificationTokens.Consume(ctx, hashToken(strings.TrimSpace(token)), now)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrInvalidVerificationToken
	}
	if err != nil {
		return nil, err
	}

	return s.createSession(ctx, *user)
}

func (s *AuthService) Authenticate(ctx context.Context, token string) (*domain.Session, error) {
	if strings.TrimSpace(token) == "" {
		return nil, ErrInvalidSession
	}

	now := s.now()
	session, err := s.sessions.FindByTokenHash(ctx, hashToken(token), now)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrInvalidSession
	}
	if err != nil {
		return nil, err
	}
	if err := s.sessions.UpdateLastUsedAt(ctx, session.ID, now); err != nil {
		return nil, err
	}
	session.LastUsedAt = now

	return session, nil
}

func (s *AuthService) Logout(ctx context.Context, token string) error {
	if strings.TrimSpace(token) == "" {
		return nil
	}

	return s.sessions.DeleteByTokenHash(ctx, hashToken(token))
}

func (s *AuthService) LogoutOtherSessions(ctx context.Context, userID uint, currentToken string) error {
	token := strings.TrimSpace(currentToken)
	if token == "" {
		return ErrInvalidSession
	}

	return s.sessions.DeleteOthersByUserIDAndTokenHash(ctx, userID, hashToken(token))
}

func (s *AuthService) UpdateLocale(ctx context.Context, userID uint, locale string) (*domain.User, error) {
	normalizedLocale, err := NormalizeLocale(locale)
	if err != nil {
		return nil, err
	}

	return s.users.UpdateLocale(ctx, userID, normalizedLocale)
}

func (s *AuthService) RefreshCSRFToken(ctx context.Context, session *domain.Session) (string, error) {
	token, err := generateToken()
	if err != nil {
		return "", err
	}

	tokenHash := hashToken(token)
	if err := s.sessions.UpdateCSRFTokenHash(ctx, session.ID, tokenHash); err != nil {
		return "", err
	}
	session.CSRFTokenHash = tokenHash

	return token, nil
}

func (s *AuthService) createEmailVerificationToken(ctx context.Context, user domain.User, locale string) error {
	token, err := generateToken()
	if err != nil {
		return err
	}

	now := s.now()
	verificationToken := domain.EmailVerificationToken{
		UserID:    user.ID,
		TokenHash: hashToken(token),
		ExpiresAt: now.Add(s.emailVerificationTTL),
	}
	if err := s.emailVerificationTokens.Create(ctx, &verificationToken); err != nil {
		return err
	}

	return s.verificationEmailSender.SendVerificationEmail(ctx, user.Email, locale, token)
}

func (s *AuthService) createPasswordResetToken(ctx context.Context, user domain.User, locale string) error {
	token, err := generateToken()
	if err != nil {
		return err
	}

	now := s.now()
	resetToken := domain.PasswordResetToken{
		UserID:    user.ID,
		TokenHash: hashToken(token),
		ExpiresAt: now.Add(s.passwordResetTTL),
	}
	if err := s.passwordResetTokens.Create(ctx, &resetToken); err != nil {
		return err
	}

	return s.passwordResetEmailSender.SendPasswordResetEmail(ctx, user.Email, locale, token)
}

func (s *AuthService) emailVerificationEnabled() bool {
	return s.emailVerificationTokens != nil && s.emailVerificationTTL > 0
}

func (s *AuthService) passwordResetEnabled() bool {
	return s.passwordResetTokens != nil && s.passwordResetTTL > 0
}

type noopVerificationEmailSender struct{}

func (noopVerificationEmailSender) SendVerificationEmail(context.Context, string, string, string) error {
	return nil
}

type noopPasswordResetEmailSender struct{}

func (noopPasswordResetEmailSender) SendPasswordResetEmail(context.Context, string, string, string) error {
	return nil
}

func ValidCSRFToken(session *domain.Session, token string) bool {
	if session == nil || strings.TrimSpace(token) == "" {
		return false
	}

	return subtle.ConstantTimeCompare([]byte(session.CSRFTokenHash), []byte(hashToken(token))) == 1
}

func NormalizeLocale(locale string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(locale)) {
	case "", "ko", "ko-kr":
		return "ko", nil
	case "en", "en-us", "en-ca", "en-gb":
		return "en", nil
	default:
		return "", ErrInvalidLocale
	}
}

func LocaleOrDefault(locale string) string {
	normalizedLocale, err := NormalizeLocale(locale)
	if err != nil {
		return "ko"
	}

	return normalizedLocale
}

func InferLocale(countryCode, acceptLanguage string) string {
	country := strings.ToUpper(strings.TrimSpace(countryCode))
	if country == "KR" {
		return "ko"
	}
	if country != "" {
		return "en"
	}

	for _, part := range strings.Split(strings.ToLower(acceptLanguage), ",") {
		language := strings.TrimSpace(strings.Split(part, ";")[0])
		if strings.HasPrefix(language, "ko") {
			return "ko"
		}
		if strings.HasPrefix(language, "en") {
			return "en"
		}
	}

	return "ko"
}

func (s *AuthService) createSession(ctx context.Context, user domain.User) (*AuthResult, error) {
	session, sessionToken, csrfToken, err := s.newSession(user)
	if err != nil {
		return nil, err
	}
	if err := s.sessions.Create(ctx, &session); err != nil {
		return nil, err
	}

	return &AuthResult{
		User:      user,
		Session:   session,
		Token:     sessionToken,
		CSRFToken: csrfToken,
	}, nil
}

func (s *AuthService) newSession(user domain.User) (domain.Session, string, string, error) {
	sessionToken, err := generateToken()
	if err != nil {
		return domain.Session{}, "", "", err
	}
	csrfToken, err := generateToken()
	if err != nil {
		return domain.Session{}, "", "", err
	}

	now := s.now()
	return domain.Session{
		UserID:        user.ID,
		User:          user,
		TokenHash:     hashToken(sessionToken),
		CSRFTokenHash: hashToken(csrfToken),
		ExpiresAt:     now.Add(s.ttl),
		LastUsedAt:    now,
	}, sessionToken, csrfToken, nil
}

func normalizeEmailAddress(email string) (string, error) {
	normalizedEmail := strings.ToLower(strings.TrimSpace(email))
	parsedAddress, err := mail.ParseAddress(normalizedEmail)
	if err != nil || parsedAddress.Address != normalizedEmail || !strings.Contains(normalizedEmail, "@") {
		return "", ErrInvalidEmail
	}

	return normalizedEmail, nil
}

func generateToken() (string, error) {
	value := make([]byte, 32)
	if _, err := rand.Read(value); err != nil {
		return "", err
	}

	return base64.RawURLEncoding.EncodeToString(value), nil
}

func hashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}
