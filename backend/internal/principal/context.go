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
	trimmedUsername := strings.TrimSpace(username)
	if trimmedUsername == "" {
		return Default()
	}

	return Principal{
		Username:      trimmedUsername,
		Authenticated: true,
	}
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
	username := strings.TrimSpace(current.Username)
	if username == "" {
		return Default()
	}

	return Principal{Username: username, Authenticated: current.Authenticated}
}
