package repository

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/Daeseong-Yu/MonthlyGoalTracker/backend/internal/db"
	"github.com/Daeseong-Yu/MonthlyGoalTracker/backend/internal/domain"
	"gorm.io/gorm"
)

func TestGoalRepositoryIntegration(t *testing.T) {
	database := openIntegrationDatabase(t)
	repo := NewGoalRepository(database)
	ctx := context.Background()

	prefix := "goal repository integration " + time.Now().UTC().Format("20060102150405.000000000")
	t.Cleanup(func() {
		cleanupGoalsByTitlePrefix(t, database, prefix)
	})
	cleanupGoalsByTitlePrefix(t, database, prefix)

	rangeStart := date(2099, time.January, 1)
	rangeEnd := date(2099, time.January, 31)

	goal := &domain.Goal{
		Title:     prefix + " create",
		StartDate: rangeStart,
	}
	if err := repo.Create(ctx, goal); err != nil {
		t.Fatalf("expected goal create to succeed, got %v", err)
	}
	if goal.ID == 0 {
		t.Fatal("expected created goal ID to be set")
	}

	found, err := repo.FindByID(ctx, goal.ID)
	if err != nil {
		t.Fatalf("expected goal lookup to succeed, got %v", err)
	}
	if found.Title != goal.Title {
		t.Fatalf("expected title %q, got %q", goal.Title, found.Title)
	}

	updated, err := repo.UpdateTitle(ctx, goal.ID, prefix+" updated")
	if err != nil {
		t.Fatalf("expected title update to succeed, got %v", err)
	}
	if updated.Title != prefix+" updated" {
		t.Fatalf("expected updated title, got %q", updated.Title)
	}

	endDate := rangeStart.AddDate(0, 0, 10)
	ended, err := repo.SetEndDate(ctx, goal.ID, &endDate)
	if err != nil {
		t.Fatalf("expected end date update to succeed, got %v", err)
	}
	if ended.EndDate == nil || !sameDate(*ended.EndDate, endDate) {
		t.Fatalf("expected end date %s, got %v", endDate.Format(time.DateOnly), ended.EndDate)
	}

	createGoal(t, repo, prefix+" before open", rangeStart.AddDate(0, 0, -10), nil)
	createGoal(t, repo, prefix+" inside open", rangeStart.AddDate(0, 0, 5), nil)

	beforeRangeEnd := rangeStart.AddDate(0, 0, -1)
	createGoal(t, repo, prefix+" before ended", rangeStart.AddDate(0, 0, -10), &beforeRangeEnd)

	afterRangeStart := rangeEnd.AddDate(0, 0, 1)
	createGoal(t, repo, prefix+" after open", afterRangeStart, nil)

	endsOnRangeStart := rangeStart
	createGoal(t, repo, prefix+" ends on range start", rangeStart.AddDate(0, 0, -10), &endsOnRangeStart)

	startsOnRangeEnd := rangeEnd
	createGoal(t, repo, prefix+" starts on range end", startsOnRangeEnd, nil)

	oneDayBoundary := rangeStart
	createGoal(t, repo, prefix+" one day on range start", rangeStart, &oneDayBoundary)

	goals, err := repo.ListOverlappingDateRange(ctx, rangeStart, rangeEnd)
	if err != nil {
		t.Fatalf("expected overlapping goals lookup to succeed, got %v", err)
	}

	titles := matchingTitles(goals, prefix)
	assertContainsTitle(t, titles, prefix+" updated")
	assertContainsTitle(t, titles, prefix+" before open")
	assertContainsTitle(t, titles, prefix+" inside open")
	assertContainsTitle(t, titles, prefix+" ends on range start")
	assertContainsTitle(t, titles, prefix+" starts on range end")
	assertContainsTitle(t, titles, prefix+" one day on range start")
	assertNotContainsTitle(t, titles, prefix+" before ended")
	assertNotContainsTitle(t, titles, prefix+" after open")
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

func createGoal(t *testing.T, repo *GoalRepository, title string, startDate time.Time, endDate *time.Time) *domain.Goal {
	t.Helper()

	goal := &domain.Goal{
		Title:     title,
		StartDate: startDate,
		EndDate:   endDate,
	}
	if err := repo.Create(context.Background(), goal); err != nil {
		t.Fatalf("expected goal %q create to succeed, got %v", title, err)
	}

	return goal
}

func cleanupGoalsByTitlePrefix(t *testing.T, database *gorm.DB, prefix string) {
	t.Helper()

	if err := database.Where("title LIKE ?", prefix+"%").Delete(&domain.Goal{}).Error; err != nil {
		t.Fatalf("failed to clean goals: %v", err)
	}
}

func matchingTitles(goals []domain.Goal, prefix string) map[string]bool {
	titles := make(map[string]bool)
	for _, goal := range goals {
		if strings.HasPrefix(goal.Title, prefix) {
			titles[goal.Title] = true
		}
	}

	return titles
}

func assertContainsTitle(t *testing.T, titles map[string]bool, title string) {
	t.Helper()

	if !titles[title] {
		t.Fatalf("expected title %q in results", title)
	}
}

func assertNotContainsTitle(t *testing.T, titles map[string]bool, title string) {
	t.Helper()

	if titles[title] {
		t.Fatalf("expected title %q to be absent from results", title)
	}
}

func date(year int, month time.Month, day int) time.Time {
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
}

func sameDate(left, right time.Time) bool {
	return left.Year() == right.Year() && left.Month() == right.Month() && left.Day() == right.Day()
}
