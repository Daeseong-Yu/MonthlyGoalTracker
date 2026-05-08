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

	"github.com/Daeseong-Yu/MonthlyGoalTracker/backend/internal/domain"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

var (
	ErrInvalidEmail        = errors.New("invalid email")
	ErrWeakPassword        = errors.New("weak password")
	ErrEmailAlreadyExists  = errors.New("email already exists")
	ErrInvalidCredentials  = errors.New("invalid credentials")
	ErrInvalidLocale       = errors.New("invalid locale")
	ErrInvalidSession      = errors.New("invalid session")
	ErrInvalidLegacyClaim  = errors.New("invalid legacy claim")
	ErrLegacyClaimRequired = domain.ErrLegacyClaimRequired
)

const minPasswordLength = 8

type AuthUserRepository interface {
	CreateWithPassword(ctx context.Context, email, passwordHash, locale string, claimLegacy bool) (*domain.User, error)
	FindByEmail(ctx context.Context, email string) (*domain.User, error)
	FindByID(ctx context.Context, id uint) (*domain.User, error)
	UpdateLocale(ctx context.Context, id uint, locale string) (*domain.User, error)
}

type AuthSessionRepository interface {
	Create(ctx context.Context, session *domain.Session) error
	FindByTokenHash(ctx context.Context, tokenHash string, now time.Time) (*domain.Session, error)
	DeleteByTokenHash(ctx context.Context, tokenHash string) error
	UpdateLastUsedAt(ctx context.Context, id uint, lastUsedAt time.Time) error
	UpdateCSRFTokenHash(ctx context.Context, id uint, csrfTokenHash string) error
}

type AuthResult struct {
	User      domain.User
	Session   domain.Session
	Token     string
	CSRFToken string
}

type AuthService struct {
	users                AuthUserRepository
	sessions             AuthSessionRepository
	ttl                  time.Duration
	now                  func() time.Time
	hashPassword         func(password string) (string, error)
	legacyClaimTokenHash string
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

func (s *AuthService) SignUp(ctx context.Context, email, password, locale, legacyClaimToken string) (*AuthResult, error) {
	normalizedEmail, err := normalizeEmailAddress(email)
	if err != nil {
		return nil, err
	}
	if len(password) < minPasswordLength {
		return nil, ErrWeakPassword
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
		return nil, ErrEmailAlreadyExists
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	user, err := s.users.CreateWithPassword(ctx, normalizedEmail, passwordHash, normalizedLocale, claimLegacy)
	if err != nil {
		return nil, err
	}

	return s.createSession(ctx, *user)
}

func bcryptHashPassword(password string) (string, error) {
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}

	return string(passwordHash), nil
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
	sessionToken, err := generateToken()
	if err != nil {
		return nil, err
	}
	csrfToken, err := generateToken()
	if err != nil {
		return nil, err
	}

	now := s.now()
	session := domain.Session{
		UserID:        user.ID,
		User:          user,
		TokenHash:     hashToken(sessionToken),
		CSRFTokenHash: hashToken(csrfToken),
		ExpiresAt:     now.Add(s.ttl),
		LastUsedAt:    now,
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
