package service

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/Daeseong-Yu/MonthlyGoalTracker/backend/internal/db"
	"github.com/Daeseong-Yu/MonthlyGoalTracker/backend/internal/domain"
	"github.com/Daeseong-Yu/MonthlyGoalTracker/backend/internal/principal"
	"github.com/Daeseong-Yu/MonthlyGoalTracker/backend/internal/repository"
	"gorm.io/gorm"
)

func openIntegrationDatabase(t *testing.T) *gorm.DB {
	t.Helper()

	if os.Getenv("RUN_DB_INTEGRATION") != "1" {
		t.Skip("set RUN_DB_INTEGRATION=1 to run service integration tests")
	}

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Fatal("DATABASE_URL is required for service integration tests")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var (
		database *gorm.DB
		err      error
	)
	for {
		database, err = db.Connect(ctx, databaseURL)
		if err == nil {
			break
		}
		if ctx.Err() != nil {
			t.Fatalf("expected database connection, got %v", err)
		}

		time.Sleep(500 * time.Millisecond)
	}

	if err := db.Migrate(ctx, database); err != nil {
		t.Fatalf("expected migration to succeed, got %v", err)
	}

	sqlDB, err := database.DB()
	if err != nil {
		t.Fatalf("expected sql database handle, got %v", err)
	}
	t.Cleanup(func() {
		if err := sqlDB.Close(); err != nil {
			t.Fatalf("failed to close database connection: %v", err)
		}
	})

	return database
}

func integrationUserContext(t *testing.T, database *gorm.DB, username string) context.Context {
	t.Helper()

	user, err := repository.NewUserRepository(database).EnsureByUsername(context.Background(), username)
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

func cleanupIntegrationGoalsByTitlePrefix(t *testing.T, database *gorm.DB, prefix string) {
	t.Helper()

	if err := database.Where("title LIKE ?", prefix+"%").Delete(&domain.Goal{}).Error; err != nil {
		t.Fatalf("failed to clean goals: %v", err)
	}
}

func date(year int, month time.Month, day int) time.Time {
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
}

func assertUTCDateOnlyEqual(t *testing.T, actual time.Time, expected time.Time) {
	t.Helper()

	if !actual.Equal(expected) {
		t.Fatalf("expected date %s, got %s", expected.Format(time.RFC3339), actual.Format(time.RFC3339))
	}
	if actual.Location() != time.UTC {
		t.Fatalf("expected UTC location, got %s", actual.Location())
	}
}
