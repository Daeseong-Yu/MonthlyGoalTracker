package router

import (
	"crypto/sha256"
	"crypto/subtle"
	"net/http"
	"strings"

	"github.com/Daeseong-Yu/MonthlyGoalTracker/backend/internal/config"
	"github.com/Daeseong-Yu/MonthlyGoalTracker/backend/internal/principal"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

const basicAuthChallenge = `Basic realm="Monthly Goal Tracker", charset="UTF-8"`

func basicAuthMiddleware(auth config.BasicAuthConfig) gin.HandlerFunc {
	expectedUsername := strings.TrimSpace(auth.Username)
	expectedPasswordHash := strings.TrimSpace(auth.PasswordHash)
	expectedUsernameDigest := sha256.Sum256([]byte(expectedUsername))

	return func(c *gin.Context) {
		username, password, ok := c.Request.BasicAuth()
		if !ok {
			rejectBasicAuth(c)
			return
		}

		usernameDigest := sha256.Sum256([]byte(username))
		usernameMatches := subtle.ConstantTimeCompare(usernameDigest[:], expectedUsernameDigest[:]) == 1
		passwordMatches := bcrypt.CompareHashAndPassword([]byte(expectedPasswordHash), []byte(password)) == nil

		if !usernameMatches || !passwordMatches {
			rejectBasicAuth(c)
			return
		}

		attachPrincipal(c, principal.NewAuthenticated(username))
		c.Next()
	}
}

func principalMiddleware(current principal.Principal) gin.HandlerFunc {
	return func(c *gin.Context) {
		attachPrincipal(c, current)
		c.Next()
	}
}

func attachPrincipal(c *gin.Context, current principal.Principal) {
	c.Request = c.Request.WithContext(principal.WithContext(c.Request.Context(), current))
}

func rejectBasicAuth(c *gin.Context) {
	c.Header("WWW-Authenticate", basicAuthChallenge)
	c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
}
