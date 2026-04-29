package db

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/Daeseong-Yu/MonthlyGoalTracker/backend/internal/domain"
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

func TestMigrateRequiresDatabase(t *testing.T) {
	err := Migrate(nil)

	if !errors.Is(err, ErrDatabaseRequired) {
		t.Fatalf("expected ErrDatabaseRequired, got %v", err)
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

	if err := Migrate(database); err != nil {
		t.Fatalf("expected migration to succeed, got %v", err)
	}

	assertMigratedConstraints(t, database)
}

func assertMigratedConstraints(t *testing.T, database *gorm.DB) {
	t.Helper()

	baseDate := time.Date(2099, time.January, 1, 0, 0, 0, 0, time.UTC).
		AddDate(0, 0, int(time.Now().Unix()%10000))

	cleanupIntegrationRows(t, database, baseDate)
	t.Cleanup(func() {
		cleanupIntegrationRows(t, database, baseDate)
	})

	memo := domain.DailyMemo{
		Date: baseDate,
		Memo: "integration memo",
	}
	if err := database.Create(&memo).Error; err != nil {
		t.Fatalf("expected first memo insert to succeed, got %v", err)
	}

	duplicateMemo := domain.DailyMemo{
		Date: baseDate,
		Memo: "integration duplicate memo",
	}
	if err := database.Create(&duplicateMemo).Error; err == nil {
		t.Fatal("expected duplicate memo date to fail")
	}

	goal := domain.Goal{
		Title:     "integration constraint goal",
		StartDate: baseDate,
		EndDate:   nil,
	}
	if err := database.Create(&goal).Error; err != nil {
		t.Fatalf("expected open-ended goal insert to succeed, got %v", err)
	}

	invalidEndDate := baseDate
	invalidGoal := domain.Goal{
		Title:     "integration invalid date range goal",
		StartDate: baseDate.AddDate(0, 0, 1),
		EndDate:   &invalidEndDate,
	}
	if err := database.Create(&invalidGoal).Error; err == nil {
		t.Fatal("expected invalid goal date range to fail")
	}

	check := domain.GoalCheck{
		GoalID: goal.ID,
		Date:   baseDate,
	}
	if err := database.Create(&check).Error; err != nil {
		t.Fatalf("expected first goal check insert to succeed, got %v", err)
	}

	duplicateCheck := domain.GoalCheck{
		GoalID: goal.ID,
		Date:   baseDate,
	}
	if err := database.Create(&duplicateCheck).Error; err == nil {
		t.Fatal("expected duplicate goal check to fail")
	}

	orphanCheck := domain.GoalCheck{
		GoalID: goal.ID + 1000000,
		Date:   baseDate.AddDate(0, 0, 1),
	}
	if err := database.Create(&orphanCheck).Error; err == nil {
		t.Fatal("expected orphan goal check to fail")
	}
}

func cleanupIntegrationRows(t *testing.T, database *gorm.DB, baseDate time.Time) {
	t.Helper()

	dates := []time.Time{
		baseDate,
		baseDate.AddDate(0, 0, 1),
	}

	if err := database.Where("date IN ?", dates).Delete(&domain.GoalCheck{}).Error; err != nil {
		t.Fatalf("failed to clean goal checks: %v", err)
	}

	if err := database.Where("date IN ?", dates).Delete(&domain.DailyMemo{}).Error; err != nil {
		t.Fatalf("failed to clean daily memos: %v", err)
	}

	if err := database.Where("title IN ?", []string{
		"integration constraint goal",
		"integration invalid date range goal",
	}).Delete(&domain.Goal{}).Error; err != nil {
		t.Fatalf("failed to clean goals: %v", err)
	}
}
