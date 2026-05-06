package repository

import (
	"context"
	"testing"

	"github.com/Daeseong-Yu/MonthlyGoalTracker/backend/internal/domain"
	"github.com/Daeseong-Yu/MonthlyGoalTracker/backend/internal/principal"
	"gorm.io/gorm"
)

const testUserID uint = 7

func scopedUserContext() context.Context {
	return principal.WithUser(context.Background(), domain.User{
		ID:       testUserID,
		Username: "app-user",
	})
}

func integrationUserContext(t *testing.T, database *gorm.DB, username string) context.Context {
	t.Helper()

	user, err := NewUserRepository(database).EnsureByUsername(context.Background(), username)
	if err != nil {
		t.Fatalf("expected integration user %q, got %v", username, err)
	}

	return principal.WithUser(context.Background(), *user)
}

func cleanupIntegrationUserByUsername(t *testing.T, database *gorm.DB, username string) {
	t.Helper()

	if err := database.Where("username = ?", username).Delete(&domain.User{}).Error; err != nil {
		t.Fatalf("failed to clean user %q: %v", username, err)
	}
}
