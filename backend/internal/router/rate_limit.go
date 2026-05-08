package router

import (
	"bytes"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

const maxAuthRateLimitBodyBytes = 64 * 1024

type rateBucket struct {
	count   int
	resetAt time.Time
}

type fixedWindowRateLimiter struct {
	mu      sync.Mutex
	limit   int
	window  time.Duration
	now     func() time.Time
	buckets map[string]rateBucket
}

func newFixedWindowRateLimiter(limit int, window time.Duration) *fixedWindowRateLimiter {
	return &fixedWindowRateLimiter{
		limit:   limit,
		window:  window,
		now:     time.Now,
		buckets: make(map[string]rateBucket),
	}
}

func (l *fixedWindowRateLimiter) allow(key string) bool {
	now := l.now()

	l.mu.Lock()
	defer l.mu.Unlock()

	bucket, ok := l.buckets[key]
	if !ok || !now.Before(bucket.resetAt) {
		l.buckets[key] = rateBucket{count: 1, resetAt: now.Add(l.window)}
		return true
	}
	if bucket.count >= l.limit {
		return false
	}

	bucket.count++
	l.buckets[key] = bucket
	return true
}

type authRateLimiter struct {
	ipLimiter        *fixedWindowRateLimiter
	principalLimiter *fixedWindowRateLimiter
}

func newAuthRateLimiter(limit int, window time.Duration) *authRateLimiter {
	return &authRateLimiter{
		ipLimiter:        newFixedWindowRateLimiter(limit, window),
		principalLimiter: newFixedWindowRateLimiter(limit, window),
	}
}

func (l *authRateLimiter) Middleware(scope string) gin.HandlerFunc {
	return func(c *gin.Context) {
		principal := authRateLimitPrincipal(c)
		if !l.ipLimiter.allow(scope + ":ip:" + requestRemoteAddress(c.Request)) {
			rejectRateLimited(c)
			return
		}
		if principal != "" && !l.principalLimiter.allow(scope+":principal:"+principal) {
			rejectRateLimited(c)
			return
		}

		c.Next()
	}
}

func authRateLimitPrincipal(c *gin.Context) string {
	if c.Request.Body == nil {
		return ""
	}

	body, err := io.ReadAll(io.LimitReader(c.Request.Body, maxAuthRateLimitBodyBytes))
	if err != nil {
		c.Request.Body = io.NopCloser(bytes.NewReader(nil))
		return ""
	}
	c.Request.Body = io.NopCloser(bytes.NewReader(body))

	var payload struct {
		Email string `json:"email"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return ""
	}

	return strings.ToLower(strings.TrimSpace(payload.Email))
}

func requestRemoteAddress(request *http.Request) string {
	remoteAddress := strings.TrimSpace(request.RemoteAddr)
	host, _, err := net.SplitHostPort(remoteAddress)
	if err == nil {
		return host
	}
	if remoteAddress == "" {
		return "unknown"
	}

	return remoteAddress
}

func rejectRateLimited(c *gin.Context) {
	c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "too many requests"})
}
