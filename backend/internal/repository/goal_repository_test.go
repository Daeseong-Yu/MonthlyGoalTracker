package repository

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/Daeseong-Yu/MonthlyGoalTracker/backend/internal/domain"
	"gorm.io/gorm"
)

func TestGoalRepositoryIntegration(t *testing.T) {
	database := openIntegrationDatabase(t)
	repo := NewGoalRepository(database)

	prefix := "goal repository integration " + time.Now().UTC().Format("20060102150405.000000000")
	username := prefix + " user"
	ctx := integrationUserContext(t, database, username)
	t.Cleanup(func() {
		cleanupGoalsByTitlePrefix(t, database, prefix)
		cleanupIntegrationUserByUsername(t, database, username)
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

	createGoal(t, ctx, repo, prefix+" before open", rangeStart.AddDate(0, 0, -10), nil)
	createGoal(t, ctx, repo, prefix+" inside open", rangeStart.AddDate(0, 0, 5), nil)

	beforeRangeEnd := rangeStart.AddDate(0, 0, -1)
	createGoal(t, ctx, repo, prefix+" before ended", rangeStart.AddDate(0, 0, -10), &beforeRangeEnd)

	afterRangeStart := rangeEnd.AddDate(0, 0, 1)
	createGoal(t, ctx, repo, prefix+" after open", afterRangeStart, nil)

	endsOnRangeStart := rangeStart
	createGoal(t, ctx, repo, prefix+" ends on range start", rangeStart.AddDate(0, 0, -10), &endsOnRangeStart)

	startsOnRangeEnd := rangeEnd
	createGoal(t, ctx, repo, prefix+" starts on range end", startsOnRangeEnd, nil)

	oneDayBoundary := rangeStart
	createGoal(t, ctx, repo, prefix+" one day on range start", rangeStart, &oneDayBoundary)

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

func createGoal(t *testing.T, ctx context.Context, repo *GoalRepository, title string, startDate time.Time, endDate *time.Time) *domain.Goal {
	t.Helper()

	goal := &domain.Goal{
		Title:     title,
		StartDate: startDate,
		EndDate:   endDate,
	}
	if err := repo.Create(ctx, goal); err != nil {
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

func sameDate(left, right time.Time) bool {
	return left.Year() == right.Year() && left.Month() == right.Month() && left.Day() == right.Day()
}
