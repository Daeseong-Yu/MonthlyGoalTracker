package principal

import (
	"context"
	"strings"
)

const defaultUsername = "single-user"

type Principal struct {
	Username      string
	Authenticated bool
}

type contextKey struct{}

func Default() Principal {
	return Principal{Username: defaultUsername}
}

func NewAuthenticated(username string) Principal {
	if strings.TrimSpace(username) == "" {
		return Default()
	}

	normalizedUsername := NormalizeUsername(username)
	return Principal{
		Username:      normalizedUsername,
		Authenticated: true,
	}
}

func NormalizeUsername(username string) string {
	trimmedUsername := strings.TrimSpace(username)
	if trimmedUsername == "" {
		return defaultUsername
	}

	return trimmedUsername
}

func WithContext(ctx context.Context, current Principal) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}

	return context.WithValue(ctx, contextKey{}, normalize(current))
}

func FromContext(ctx context.Context) Principal {
	if ctx == nil {
		return Default()
	}

	stored, ok := ctx.Value(contextKey{}).(Principal)
	if !ok {
		return Default()
	}

	return normalize(stored)
}

func normalize(current Principal) Principal {
	if strings.TrimSpace(current.Username) == "" {
		return Default()
	}

	username := NormalizeUsername(current.Username)
	return Principal{Username: username, Authenticated: current.Authenticated}
}
