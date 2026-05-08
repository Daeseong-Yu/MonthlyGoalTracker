package router

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestAuthRateLimiterLimitsByRemoteAddress(t *testing.T) {
	limiter := newAuthRateLimiter(1, time.Minute)
	engine := newTestEngine(t)
	engine.POST("/auth", limiter.Middleware("login"), func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	first := performAuthRateLimitRequest(t, engine, `{"email":"first@example.com"}`, "198.51.100.1:1000")
	if first.Code != http.StatusNoContent {
		t.Fatalf("expected first request status %d, got %d", http.StatusNoContent, first.Code)
	}

	second := performAuthRateLimitRequest(t, engine, `{"email":"second@example.com"}`, "198.51.100.1:1001")
	if second.Code != http.StatusTooManyRequests {
		t.Fatalf("expected second request status %d, got %d", http.StatusTooManyRequests, second.Code)
	}
}

func TestAuthRateLimiterLimitsByPrincipal(t *testing.T) {
	limiter := newAuthRateLimiter(1, time.Minute)
	engine := newTestEngine(t)
	engine.POST("/auth", limiter.Middleware("signup"), func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	first := performAuthRateLimitRequest(t, engine, `{"email":"Owner@Example.com"}`, "198.51.100.1:1000")
	if first.Code != http.StatusNoContent {
		t.Fatalf("expected first request status %d, got %d", http.StatusNoContent, first.Code)
	}

	second := performAuthRateLimitRequest(t, engine, `{"email":" owner@example.com "}`, "198.51.100.2:1000")
	if second.Code != http.StatusTooManyRequests {
		t.Fatalf("expected second request status %d, got %d", http.StatusTooManyRequests, second.Code)
	}
}

func TestAuthRateLimiterRestoresRequestBody(t *testing.T) {
	limiter := newAuthRateLimiter(1, time.Minute)
	engine := newTestEngine(t)
	engine.POST("/auth", limiter.Middleware("signup"), func(c *gin.Context) {
		var payload struct {
			Email string `json:"email"`
		}
		if err := c.ShouldBindJSON(&payload); err != nil {
			t.Fatalf("expected restored request body to bind, got %v", err)
		}
		if payload.Email != "owner@example.com" {
			t.Fatalf("expected owner@example.com email, got %q", payload.Email)
		}

		c.Status(http.StatusNoContent)
	})

	recorder := performAuthRateLimitRequest(t, engine, `{"email":"owner@example.com"}`, "198.51.100.3:1000")
	if recorder.Code != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d", http.StatusNoContent, recorder.Code)
	}
}

func performAuthRateLimitRequest(t *testing.T, engine *gin.Engine, body, remoteAddr string) *httptest.ResponseRecorder {
	t.Helper()

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/auth", strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	request.RemoteAddr = remoteAddr

	engine.ServeHTTP(recorder, request)

	return recorder
}
