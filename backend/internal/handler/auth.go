package handler

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/Daeseong-Yu/MonthlyGoalTracker/backend/internal/config"
	"github.com/Daeseong-Yu/MonthlyGoalTracker/backend/internal/domain"
	"github.com/Daeseong-Yu/MonthlyGoalTracker/backend/internal/principal"
	"github.com/Daeseong-Yu/MonthlyGoalTracker/backend/internal/service"
	"github.com/gin-gonic/gin"
)

type AuthHandler struct {
	service *service.AuthService
	cookies config.SessionConfig
}

func NewAuthHandler(authService *service.AuthService, cookies config.SessionConfig) *AuthHandler {
	return &AuthHandler{
		service: authService,
		cookies: cookies.WithDefaults(),
	}
}

type authRequest struct {
	Email      string `json:"email"`
	Password   string `json:"password"`
	Locale     string `json:"locale"`
	ClaimToken string `json:"claimToken"`
}

type updateLocaleRequest struct {
	Locale string `json:"locale"`
}

type verifyEmailRequest struct {
	Token string `json:"token"`
}

type userSessionResponse struct {
	ID        uint      `json:"id"`
	Email     string    `json:"email"`
	Locale    string    `json:"locale"`
	CreatedAt time.Time `json:"createdAt"`
}

type authResponse struct {
	User      userSessionResponse `json:"user"`
	CSRFToken string              `json:"csrfToken"`
	Locale    string              `json:"locale"`
}

type signupAcceptedResponse struct {
	Status string `json:"status"`
	Locale string `json:"locale"`
}

type bootstrapResponse struct {
	Authenticated bool                 `json:"authenticated"`
	Locale        string               `json:"locale"`
	LocaleSource  string               `json:"localeSource"`
	User          *userSessionResponse `json:"user,omitempty"`
	CSRFToken     string               `json:"csrfToken,omitempty"`
}

func (h *AuthHandler) Bootstrap(c *gin.Context) {
	locale := service.InferLocale(requestCountry(c), c.GetHeader("Accept-Language"))
	response := bootstrapResponse{
		Locale:       locale,
		LocaleSource: "region",
	}

	session, err := h.sessionFromCookie(c)
	if err != nil {
		if errors.Is(err, service.ErrInvalidSession) || errors.Is(err, http.ErrNoCookie) {
			h.clearCookies(c)
			c.JSON(http.StatusOK, response)
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	csrfToken, err := h.csrfTokenForSession(c, session)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	user := toUserSessionResponse(session.User)
	response.Locale = service.LocaleOrDefault(session.User.Locale)
	response.LocaleSource = "user"
	response.Authenticated = true
	response.User = &user
	response.CSRFToken = csrfToken
	h.setCSRFCookie(c, csrfToken)
	c.JSON(http.StatusOK, response)
}

func (h *AuthHandler) SignUp(c *gin.Context) {
	var req authRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	result, err := h.service.SignUp(c.Request.Context(), req.Email, req.Password, req.Locale, req.ClaimToken)
	if err != nil {
		writeAuthError(c, err)
		return
	}

	if result.Auth == nil {
		c.JSON(http.StatusAccepted, signupAcceptedResponse{
			Status: "verification_required",
			Locale: service.LocaleOrDefault(result.Locale),
		})
		return
	}

	h.setAuthCookies(c, result.Auth)
	c.JSON(http.StatusCreated, toAuthResponse(result.Auth))
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req authRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	result, err := h.service.Login(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		writeAuthError(c, err)
		return
	}

	h.setAuthCookies(c, result)
	c.JSON(http.StatusOK, toAuthResponse(result))
}

func (h *AuthHandler) VerifyEmail(c *gin.Context) {
	var req verifyEmailRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	result, err := h.service.VerifyEmail(c.Request.Context(), req.Token)
	if err != nil {
		writeAuthError(c, err)
		return
	}

	h.setAuthCookies(c, result)
	c.JSON(http.StatusOK, toAuthResponse(result))
}

func (h *AuthHandler) Logout(c *gin.Context) {
	token, err := c.Cookie(h.cookies.CookieName)
	if err != nil && !errors.Is(err, http.ErrNoCookie) {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}
	if err := h.service.Logout(c.Request.Context(), token); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	h.clearCookies(c)
	c.Status(http.StatusNoContent)
}

func (h *AuthHandler) Me(c *gin.Context) {
	user, ok := principal.UserFromContext(c.Request.Context())
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	c.JSON(http.StatusOK, toUserSessionResponse(user))
}

func (h *AuthHandler) UpdateLocale(c *gin.Context) {
	currentUser, ok := principal.UserFromContext(c.Request.Context())
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req updateLocaleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	user, err := h.service.UpdateLocale(c.Request.Context(), currentUser.ID, req.Locale)
	if err != nil {
		writeAuthError(c, err)
		return
	}

	c.JSON(http.StatusOK, toUserSessionResponse(*user))
}

func (h *AuthHandler) sessionFromCookie(c *gin.Context) (*domain.Session, error) {
	token, err := c.Cookie(h.cookies.CookieName)
	if err != nil {
		return nil, err
	}

	return h.service.Authenticate(c.Request.Context(), token)
}

func (h *AuthHandler) csrfTokenForSession(c *gin.Context, session *domain.Session) (string, error) {
	token, err := c.Cookie(h.cookies.CSRFCookieName)
	if err == nil && service.ValidCSRFToken(session, token) {
		return token, nil
	}

	return h.service.RefreshCSRFToken(c.Request.Context(), session)
}

func (h *AuthHandler) setAuthCookies(c *gin.Context, result *service.AuthResult) {
	h.setCookie(c, h.cookies.CookieName, result.Token, true)
	h.setCSRFCookie(c, result.CSRFToken)
}

func (h *AuthHandler) setCSRFCookie(c *gin.Context, token string) {
	h.setCookie(c, h.cookies.CSRFCookieName, token, false)
}

func (h *AuthHandler) clearCookies(c *gin.Context) {
	h.setExpiredCookie(c, h.cookies.CookieName, true)
	h.setExpiredCookie(c, h.cookies.CSRFCookieName, false)
}

func (h *AuthHandler) setCookie(c *gin.Context, name, value string, httpOnly bool) {
	c.SetSameSite(sameSiteMode(h.cookies.SameSite))
	c.SetCookie(name, value, int((time.Duration(h.cookies.TTLHours) * time.Hour).Seconds()), "/", "", h.cookies.Secure, httpOnly)
}

func (h *AuthHandler) setExpiredCookie(c *gin.Context, name string, httpOnly bool) {
	c.SetSameSite(sameSiteMode(h.cookies.SameSite))
	c.SetCookie(name, "", -1, "/", "", h.cookies.Secure, httpOnly)
}

func toAuthResponse(result *service.AuthResult) authResponse {
	return authResponse{
		User:      toUserSessionResponse(result.User),
		CSRFToken: result.CSRFToken,
		Locale:    service.LocaleOrDefault(result.User.Locale),
	}
}

func toUserSessionResponse(user domain.User) userSessionResponse {
	return userSessionResponse{
		ID:        user.ID,
		Email:     user.Email,
		Locale:    service.LocaleOrDefault(user.Locale),
		CreatedAt: user.CreatedAt,
	}
}

func writeAuthError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrInvalidEmail), errors.Is(err, service.ErrWeakPassword), errors.Is(err, service.ErrInvalidLocale):
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	case errors.Is(err, service.ErrEmailAlreadyExists):
		c.JSON(http.StatusBadRequest, gin.H{"error": "signup failed"})
	case errors.Is(err, service.ErrInvalidLegacyClaim):
		c.JSON(http.StatusForbidden, gin.H{"error": "invalid legacy claim"})
	case errors.Is(err, service.ErrEmailNotVerified):
		c.JSON(http.StatusForbidden, gin.H{"error": "email not verified"})
	case errors.Is(err, service.ErrLegacyClaimRequired):
		c.JSON(http.StatusConflict, gin.H{"error": "legacy claim required"})
	case errors.Is(err, service.ErrInvalidVerificationToken):
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid verification token"})
	case errors.Is(err, service.ErrInvalidCredentials), errors.Is(err, service.ErrInvalidSession):
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
	}
}

func requestCountry(c *gin.Context) string {
	for _, header := range []string{"CF-IPCountry", "X-Vercel-IP-Country", "X-App-Country", "X-Country-Code"} {
		value := strings.TrimSpace(c.GetHeader(header))
		if value != "" {
			return value
		}
	}

	return ""
}

func sameSiteMode(value string) http.SameSite {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "strict":
		return http.SameSiteStrictMode
	case "none":
		return http.SameSiteNoneMode
	default:
		return http.SameSiteLaxMode
	}
}
