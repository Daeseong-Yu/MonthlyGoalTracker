package db

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"gorm.io/gorm"
)

func TestConnectRequiresDatabaseURL(t *testing.T) {
	tests := []struct {
		name        string
		databaseURL string
	}{
		{
			name:        "empty",
			databaseURL: "",
		},
		{
			name:        "blank",
			databaseURL: "   ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			database, err := Connect(context.Background(), tt.databaseURL)

			if database != nil {
				t.Fatal("expected nil database when database URL is empty")
			}

			if !errors.Is(err, ErrDatabaseURLRequired) {
				t.Fatalf("expected ErrDatabaseURLRequired, got %v", err)
			}
		})
	}
}

func TestConnectReturnsErrorForMalformedDatabaseURL(t *testing.T) {
	database, err := Connect(context.Background(), "://")

	if database != nil {
		t.Fatal("expected nil database when database URL is malformed")
	}

	if err == nil {
		t.Fatal("expected error when database URL is malformed")
	}
}

func TestConnectUsesContextForPing(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	database, err := Connect(ctx, "postgres://postgres:postgres@127.0.0.1:5432/monthly_goal_tracker?sslmode=disable")

	if database != nil {
		t.Fatal("expected nil database when context is canceled")
	}

	if err == nil {
		t.Fatal("expected error when context is canceled")
	}

	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
}

func TestConnectIntegration(t *testing.T) {
	if os.Getenv("RUN_DB_INTEGRATION") != "1" {
		t.Skip("set RUN_DB_INTEGRATION=1 to run database integration test")
	}

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Fatal("DATABASE_URL is required for database integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var (
		database *gorm.DB
		err      error
	)

	for {
		database, err = Connect(ctx, databaseURL)
		if err == nil {
			break
		}

		if ctx.Err() != nil {
			t.Fatalf("expected database connection, got %v", err)
		}

		time.Sleep(500 * time.Millisecond)
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
}
