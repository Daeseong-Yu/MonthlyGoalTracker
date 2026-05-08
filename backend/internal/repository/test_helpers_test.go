package repository

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/Daeseong-Yu/MonthlyGoalTracker/backend/internal/db"
	"github.com/Daeseong-Yu/MonthlyGoalTracker/backend/internal/domain"
	"github.com/Daeseong-Yu/MonthlyGoalTracker/backend/internal/principal"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

const testUserID uint = 7

func scopedUserContext() context.Context {
	return principal.WithUser(context.Background(), domain.User{
		ID:       testUserID,
		Username: "app-user",
	})
}

func newMockDatabase(t *testing.T) (*gorm.DB, sqlmock.Sqlmock, func()) {
	t.Helper()

	sqlDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sql mock: %v", err)
	}

	database, err := gorm.Open(postgres.New(postgres.Config{
		Conn:                 sqlDB,
		PreferSimpleProtocol: true,
	}), &gorm.Config{
		DisableAutomaticPing:   true,
		SkipDefaultTransaction: true,
		NowFunc:                fixedNow,
	})
	if err != nil {
		t.Fatalf("failed to create gorm database: %v", err)
	}

	return database, mock, func() {
		_ = sqlDB.Close()
	}
}

func openIntegrationDatabase(t *testing.T) *gorm.DB {
	t.Helper()

	if os.Getenv("RUN_DB_INTEGRATION") != "1" {
		t.Skip("set RUN_DB_INTEGRATION=1 to run repository integration tests")
	}

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Fatal("DATABASE_URL is required for repository integration tests")
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

func date(year int, month time.Month, day int) time.Time {
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
}

func uniqueIntegrationDate() time.Time {
	offset := int(time.Now().UnixNano() % 20000)
	return date(2100, time.January, 1).AddDate(0, 0, offset)
}
