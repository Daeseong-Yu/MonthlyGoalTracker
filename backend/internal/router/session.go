package router

import (
	"context"
	"crypto/subtle"
	"errors"
	"net/http"

	"github.com/Daeseong-Yu/MonthlyGoalTracker/backend/internal/domain"
	"github.com/Daeseong-Yu/MonthlyGoalTracker/backend/internal/principal"
	"github.com/Daeseong-Yu/MonthlyGoalTracker/backend/internal/service"
	"github.com/gin-gonic/gin"
)

type sessionContextKey struct{}

func sessionMiddleware(authService *service.AuthService, cookieName string) gin.HandlerFunc {
	return func(c *gin.Context) {
		token, err := c.Cookie(cookieName)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		session, err := authService.Authenticate(c.Request.Context(), token)
		if errors.Is(err, service.ErrInvalidSession) {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
			return
		}

		ctx := c.Request.Context()
		ctx = principal.WithContext(ctx, principal.NewAuthenticated(session.User.Username))
		ctx = principal.WithUser(ctx, session.User)
		ctx = withSession(ctx, session)
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}

func csrfMiddleware(csrfCookieName string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if isSafeMethod(c.Request.Method) {
			c.Next()
			return
		}

		session, ok := sessionFromContext(c.Request.Context())
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		headerToken := c.GetHeader("X-CSRF-Token")
		cookieToken, err := c.Cookie(csrfCookieName)
		if err != nil || subtle.ConstantTimeCompare([]byte(headerToken), []byte(cookieToken)) != 1 {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "csrf token required"})
			return
		}
		if !service.ValidCSRFToken(session, headerToken) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "csrf token required"})
			return
		}

		c.Next()
	}
}

func withSession(ctx context.Context, session *domain.Session) context.Context {
	return context.WithValue(ctx, sessionContextKey{}, session)
}

func sessionFromContext(ctx context.Context) (*domain.Session, bool) {
	session, ok := ctx.Value(sessionContextKey{}).(*domain.Session)
	return session, ok
}

func isSafeMethod(method string) bool {
	return method == http.MethodGet || method == http.MethodHead || method == http.MethodOptions
}
