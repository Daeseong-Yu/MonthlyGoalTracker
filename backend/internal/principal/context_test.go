package principal

import (
	"context"
	"testing"

	"github.com/Daeseong-Yu/MonthlyGoalTracker/backend/internal/domain"
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

func TestWithUserStoresResolvedUser(t *testing.T) {
	expectedUser := domain.User{ID: 7, Username: "app-user"}

	ctx := WithUser(context.Background(), expectedUser)

	got, ok := UserFromContext(ctx)
	if !ok {
		t.Fatal("expected user in context")
	}
	if got != expectedUser {
		t.Fatalf("expected user %+v, got %+v", expectedUser, got)
	}
}

func TestUserFromContextRejectsZeroIDUser(t *testing.T) {
	ctx := WithUser(context.Background(), domain.User{Username: "app-user"})

	if _, ok := UserFromContext(ctx); ok {
		t.Fatal("expected zero-ID user to be treated as missing")
	}
}
