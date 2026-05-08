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

func TestAuthRateLimiterUsesTrustedForwardedClientIP(t *testing.T) {
	limiter := newAuthRateLimiter(1, time.Minute)
	engine := newTestEngine(t)
	if err := engine.SetTrustedProxies([]string{"127.0.0.1"}); err != nil {
		t.Fatalf("failed to set trusted proxies: %v", err)
	}
	engine.POST("/auth", limiter.Middleware("login"), func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	first := performAuthRateLimitRequest(t, engine, `{"email":"first@example.com"}`, "127.0.0.1:1000", withHeader("X-Forwarded-For", "198.51.100.10"))
	if first.Code != http.StatusNoContent {
		t.Fatalf("expected first request status %d, got %d", http.StatusNoContent, first.Code)
	}

	second := performAuthRateLimitRequest(t, engine, `{"email":"second@example.com"}`, "127.0.0.1:1001", withHeader("X-Forwarded-For", "198.51.100.10"))
	if second.Code != http.StatusTooManyRequests {
		t.Fatalf("expected second request status %d, got %d", http.StatusTooManyRequests, second.Code)
	}
}

func TestAuthRateLimiterIgnoresForwardedClientIPWithoutTrustedProxy(t *testing.T) {
	limiter := newAuthRateLimiter(1, time.Minute)
	engine := newTestEngine(t)
	if err := engine.SetTrustedProxies(nil); err != nil {
		t.Fatalf("failed to clear trusted proxies: %v", err)
	}
	engine.POST("/auth", limiter.Middleware("login"), func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	first := performAuthRateLimitRequest(t, engine, `{"email":"first@example.com"}`, "198.51.100.10:1000", withHeader("X-Forwarded-For", "203.0.113.7"))
	if first.Code != http.StatusNoContent {
		t.Fatalf("expected first request status %d, got %d", http.StatusNoContent, first.Code)
	}

	second := performAuthRateLimitRequest(t, engine, `{"email":"second@example.com"}`, "198.51.100.11:1000", withHeader("X-Forwarded-For", "203.0.113.7"))
	if second.Code != http.StatusNoContent {
		t.Fatalf("expected second request status %d, got %d", http.StatusNoContent, second.Code)
	}
}

func TestFixedWindowRateLimiterCapsAndPrunesBuckets(t *testing.T) {
	now := time.Date(2026, 5, 8, 12, 0, 0, 0, time.UTC)
	limiter := newFixedWindowRateLimiterWithMaxKeys(1, time.Minute, 2)
	limiter.now = func() time.Time {
		return now
	}

	if !limiter.allow("first") {
		t.Fatal("expected first bucket to be allowed")
	}
	if !limiter.allow("second") {
		t.Fatal("expected second bucket to be allowed")
	}
	if limiter.allow("third") {
		t.Fatal("expected third bucket to be rejected while bucket cap is full")
	}

	now = now.Add(time.Minute)
	if !limiter.allow("third") {
		t.Fatal("expected expired buckets to be pruned before applying the cap")
	}
	if got := len(limiter.buckets); got != 1 {
		t.Fatalf("expected one bucket after pruning, got %d", got)
	}
}

func performAuthRateLimitRequest(t *testing.T, engine *gin.Engine, body, remoteAddr string, opts ...requestOption) *httptest.ResponseRecorder {
	t.Helper()

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/auth", strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	request.RemoteAddr = remoteAddr
	for _, opt := range opts {
		opt(request)
	}

	engine.ServeHTTP(recorder, request)

	return recorder
}

func withHeader(name, value string) requestOption {
	return func(request *http.Request) {
		request.Header.Set(name, value)
	}
}
