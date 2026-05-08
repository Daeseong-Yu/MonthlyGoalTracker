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

const (
	maxAuthRateLimitBodyBytes   = 64 * 1024
	authRateLimitPruneEvery     = time.Minute
	defaultAuthRateLimitMaxKeys = 10000
)

type rateBucket struct {
	count   int
	resetAt time.Time
}

type fixedWindowRateLimiter struct {
	mu       sync.Mutex
	limit    int
	window   time.Duration
	maxKeys  int
	now      func() time.Time
	buckets  map[string]rateBucket
	prunedAt time.Time
}

func newFixedWindowRateLimiter(limit int, window time.Duration) *fixedWindowRateLimiter {
	return newFixedWindowRateLimiterWithMaxKeys(limit, window, defaultAuthRateLimitMaxKeys)
}

func newFixedWindowRateLimiterWithMaxKeys(limit int, window time.Duration, maxKeys int) *fixedWindowRateLimiter {
	if maxKeys <= 0 {
		maxKeys = defaultAuthRateLimitMaxKeys
	}

	return &fixedWindowRateLimiter{
		limit:   limit,
		window:  window,
		maxKeys: maxKeys,
		now:     time.Now,
		buckets: make(map[string]rateBucket),
	}
}

func (l *fixedWindowRateLimiter) allow(key string) bool {
	now := l.now()

	l.mu.Lock()
	defer l.mu.Unlock()

	bucket, ok := l.buckets[key]
	if ok {
		if now.Before(bucket.resetAt) {
			if bucket.count >= l.limit {
				return false
			}

			bucket.count++
			l.buckets[key] = bucket
			return true
		}

		delete(l.buckets, key)
	}

	if l.shouldPrune(now) {
		l.pruneExpired(now)
	}
	if l.maxKeys > 0 && len(l.buckets) >= l.maxKeys {
		l.pruneExpired(now)
		if len(l.buckets) >= l.maxKeys {
			return false
		}
	}

	l.buckets[key] = rateBucket{count: 1, resetAt: now.Add(l.window)}
	return true
}

func (l *fixedWindowRateLimiter) shouldPrune(now time.Time) bool {
	return l.prunedAt.IsZero() || !now.Before(l.prunedAt.Add(authRateLimitPruneEvery))
}

func (l *fixedWindowRateLimiter) pruneExpired(now time.Time) {
	for key, bucket := range l.buckets {
		if !now.Before(bucket.resetAt) {
			delete(l.buckets, key)
		}
	}
	l.prunedAt = now
}

type authRateLimiter struct {
	ipLimiter        *fixedWindowRateLimiter
	principalLimiter *fixedWindowRateLimiter
}

func newAuthRateLimiter(limit int, window time.Duration) *authRateLimiter {
	return newAuthRateLimiterWithMaxKeys(limit, window, defaultAuthRateLimitMaxKeys)
}

func newAuthRateLimiterWithMaxKeys(limit int, window time.Duration, maxKeys int) *authRateLimiter {
	return &authRateLimiter{
		ipLimiter:        newFixedWindowRateLimiterWithMaxKeys(limit, window, maxKeys),
		principalLimiter: newFixedWindowRateLimiterWithMaxKeys(limit, window, maxKeys),
	}
}

func (l *authRateLimiter) Middleware(scope string) gin.HandlerFunc {
	return func(c *gin.Context) {
		principal := authRateLimitPrincipal(c)
		if !l.ipLimiter.allow(scope + ":ip:" + requestClientAddress(c)) {
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

func requestClientAddress(c *gin.Context) string {
	clientIP := strings.TrimSpace(c.ClientIP())
	if clientIP != "" {
		return clientIP
	}

	return requestRemoteAddress(c.Request)
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
