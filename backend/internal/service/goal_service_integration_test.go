package service

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/Daeseong-Yu/MonthlyGoalTracker/backend/internal/db"
	"github.com/Daeseong-Yu/MonthlyGoalTracker/backend/internal/domain"
	"github.com/Daeseong-Yu/MonthlyGoalTracker/backend/internal/repository"
	"gorm.io/gorm"
)

func TestGoalServiceIntegrationPreservesNonUTCLocalCalendarDay(t *testing.T) {
	database := openServiceIntegrationDatabase(t)
	repo := repository.NewGoalRepository(database)
	service := NewGoalService(repo)
	ctx := context.Background()
	kst := time.FixedZone("KST", 9*60*60)

	prefix := "goal service integration " + time.Now().UTC().Format("20060102150405.000000000")
	title := prefix + " create"
	t.Cleanup(func() {
		cleanupServiceGoalsByTitlePrefix(t, database, prefix)
	})
	cleanupServiceGoalsByTitlePrefix(t, database, prefix)

	created, err := service.CreateGoal(ctx, "2199-04", title, time.Date(2199, time.April, 1, 0, 30, 0, 0, kst))
	if err != nil {
		t.Fatalf("expected create to succeed, got %v", err)
	}
	if created == nil {
		t.Fatal("expected created goal")
	}
	assertUTCDateOnlyEqual(t, created.StartDate, time.Date(2199, time.April, 1, 0, 0, 0, 0, time.UTC))

	storedAfterCreate, err := repo.FindByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("expected created goal lookup to succeed, got %v", err)
	}
	assertUTCDateOnlyEqual(t, storedAfterCreate.StartDate, time.Date(2199, time.April, 1, 0, 0, 0, 0, time.UTC))

	deactivated, err := service.DeactivateGoal(ctx, created.ID, time.Date(2199, time.April, 3, 8, 0, 0, 0, kst))
	if err != nil {
		t.Fatalf("expected deactivate to succeed, got %v", err)
	}
	if deactivated == nil || deactivated.EndDate == nil {
		t.Fatal("expected deactivated goal with end date")
	}
	assertUTCDateOnlyEqual(t, *deactivated.EndDate, time.Date(2199, time.April, 3, 0, 0, 0, 0, time.UTC))

	storedAfterDeactivate, err := repo.FindByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("expected deactivated goal lookup to succeed, got %v", err)
	}
	if storedAfterDeactivate.EndDate == nil {
		t.Fatal("expected stored end date")
	}
	assertUTCDateOnlyEqual(t, *storedAfterDeactivate.EndDate, time.Date(2199, time.April, 3, 0, 0, 0, 0, time.UTC))
}

func openServiceIntegrationDatabase(t *testing.T) *gorm.DB {
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

func cleanupServiceGoalsByTitlePrefix(t *testing.T, database *gorm.DB, prefix string) {
	t.Helper()

	if err := database.Where("title LIKE ?", prefix+"%").Delete(&domain.Goal{}).Error; err != nil {
		t.Fatalf("failed to clean goals: %v", err)
	}
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
