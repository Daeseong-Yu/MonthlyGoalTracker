package principal

import (
	"context"
	"testing"
)

func TestFromContextReturnsDefaultWhenMissing(t *testing.T) {
	if got := FromContext(context.Background()); got != Default() {
		t.Fatalf("expected default principal %+v, got %+v", Default(), got)
	}
}

func TestWithContextStoresNormalizedPrincipal(t *testing.T) {
	ctx := WithContext(context.Background(), Principal{Username: " app-user ", Authenticated: true})

	got := FromContext(ctx)
	if got.Username != "app-user" {
		t.Fatalf("expected username app-user, got %q", got.Username)
	}
	if !got.Authenticated {
		t.Fatal("expected principal to remain authenticated")
	}
}

func TestNewAuthenticatedFallsBackForBlankUsername(t *testing.T) {
	if got := NewAuthenticated("   "); got != Default() {
		t.Fatalf("expected default principal %+v, got %+v", Default(), got)
	}
}
