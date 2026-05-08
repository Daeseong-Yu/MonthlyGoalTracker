package repository

import (
	"context"
	"errors"
	"testing"
)

func TestCurrentUserIDRequiresResolvedUser(t *testing.T) {
	userID, err := currentUserID(context.Background())
	if userID != 0 {
		t.Fatalf("expected zero user ID, got %d", userID)
	}
	if !errors.Is(err, ErrUserContextRequired) {
		t.Fatalf("expected ErrUserContextRequired, got %v", err)
	}
}

func TestCurrentUserIDReturnsResolvedUser(t *testing.T) {
	userID, err := currentUserID(scopedUserContext())
	if err != nil {
		t.Fatalf("expected resolved user ID, got %v", err)
	}
	if userID != testUserID {
		t.Fatalf("expected user ID %d, got %d", testUserID, userID)
	}
}
