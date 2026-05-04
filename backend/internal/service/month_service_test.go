package service

import (
	"context"
	"testing"
	"time"

	"github.com/Daeseong-Yu/MonthlyGoalTracker/backend/internal/domain"
)

func TestGetMonthViewBuildsDailyStatsAndChart(t *testing.T) {
	endDate := time.Date(2026, time.April, 2, 0, 0, 0, 0, time.UTC)
	goalRepo := &stubGoalRepository{
		listOverlappingDateRangeFunc: func(context.Context, time.Time, time.Time) ([]domain.Goal, error) {
			return []domain.Goal{
				{
					ID:        1,
					Title:     "all month",
					StartDate: time.Date(2026, time.April, 1, 0, 0, 0, 0, time.UTC),
				},
				{
					ID:        2,
					Title:     "starts later",
					StartDate: time.Date(2026, time.April, 2, 0, 0, 0, 0, time.UTC),
				},
				{
					ID:        3,
					Title:     "ends early",
					StartDate: time.Date(2026, time.April, 1, 0, 0, 0, 0, time.UTC),
					EndDate:   &endDate,
				},
			}, nil
		},
	}
	memoRepo := &stubMemoRepository{
		listByDateRangeFunc: func(context.Context, time.Time, time.Time) ([]domain.DailyMemo, error) {
			return []domain.DailyMemo{
				{Date: time.Date(2026, time.April, 2, 0, 0, 0, 0, time.UTC), Memo: "memo"},
			}, nil
		},
	}
	checkRepo := &stubGoalCheckRepository{
		listByDateRangeFunc: func(context.Context, time.Time, time.Time) ([]domain.GoalCheck, error) {
			return []domain.GoalCheck{
				{GoalID: 1, Date: time.Date(2026, time.April, 1, 0, 0, 0, 0, time.UTC)},
				{GoalID: 1, Date: time.Date(2026, time.April, 2, 0, 0, 0, 0, time.UTC)},
				{GoalID: 2, Date: time.Date(2026, time.April, 2, 0, 0, 0, 0, time.UTC)},
				{GoalID: 3, Date: time.Date(2026, time.April, 3, 0, 0, 0, 0, time.UTC)},
			}, nil
		},
	}
	service := NewMonthService(goalRepo, memoRepo, checkRepo)

	view, err := service.GetMonthView(context.Background(), "2026-04")
	if err != nil {
		t.Fatalf("expected month view to succeed, got %v", err)
	}

	if len(view.Days) != 30 {
		t.Fatalf("expected 30 days, got %d", len(view.Days))
	}
	if len(view.Chart) != 30 {
		t.Fatalf("expected 30 chart points, got %d", len(view.Chart))
	}

	april1 := findDayEntry(t, view.Days, time.Date(2026, time.April, 1, 0, 0, 0, 0, time.UTC))
	assertDayStats(t, april1, 2, 1, 0.5)

	april2 := findDayEntry(t, view.Days, time.Date(2026, time.April, 2, 0, 0, 0, 0, time.UTC))
	assertDayStats(t, april2, 3, 2, float64(2)/float64(3))
	if april2.Memo != "memo" {
		t.Fatalf("expected memo %q, got %q", "memo", april2.Memo)
	}

	april3 := findDayEntry(t, view.Days, time.Date(2026, time.April, 3, 0, 0, 0, 0, time.UTC))
	assertDayStats(t, april3, 2, 0, 0)
}

func TestEnsureMonthCopiesPreviousActiveGoalsAndEndsOriginal(t *testing.T) {
	monthStart := time.Date(2026, time.April, 1, 0, 0, 0, 0, time.UTC)
	previousMonthEnd := time.Date(2026, time.March, 31, 0, 0, 0, 0, time.UTC)
	previousGoal := domain.Goal{
		ID:        7,
		Title:     "Exercise",
		StartDate: time.Date(2026, time.March, 1, 0, 0, 0, 0, time.UTC),
	}

	listCalls := 0
	goalRepo := &stubGoalRepository{
		listOverlappingDateRangeFunc: func(context.Context, time.Time, time.Time) ([]domain.Goal, error) {
			listCalls++
			switch listCalls {
			case 1:
				return nil, nil
			case 2:
				return []domain.Goal{previousGoal}, nil
			default:
				return []domain.Goal{
					{ID: 8, Title: previousGoal.Title, StartDate: monthStart},
				}, nil
			}
		},
	}
	service := NewMonthService(goalRepo, &stubMemoRepository{}, &stubGoalCheckRepository{})

	view, err := service.EnsureMonth(context.Background(), "2026-04")
	if err != nil {
		t.Fatalf("expected ensure month to succeed, got %v", err)
	}
	if view == nil {
		t.Fatal("expected month view")
	}

	if goalRepo.createCalls != 1 {
		t.Fatalf("expected one copied goal, got %d", goalRepo.createCalls)
	}
	if goalRepo.createdGoal == nil {
		t.Fatal("expected copied goal to be recorded")
	}
	if goalRepo.createdGoal.Title != previousGoal.Title {
		t.Fatalf("expected copied title %q, got %q", previousGoal.Title, goalRepo.createdGoal.Title)
	}
	assertDateEqual(t, goalRepo.createdGoal.StartDate, monthStart)

	if goalRepo.setEndDateCalls != 1 {
		t.Fatalf("expected previous goal to be ended once, got %d", goalRepo.setEndDateCalls)
	}
	if goalRepo.lastSetEndDateGoalID != previousGoal.ID {
		t.Fatalf("expected end date to target goal %d, got %d", previousGoal.ID, goalRepo.lastSetEndDateGoalID)
	}
	if goalRepo.lastSetEndDate == nil {
		t.Fatal("expected end date to be set")
	}
	assertDateEqual(t, *goalRepo.lastSetEndDate, previousMonthEnd)
}

func findDayEntry(t *testing.T, days []DayEntry, date time.Time) DayEntry {
	t.Helper()
	for _, day := range days {
		if day.Date.Equal(date) {
			return day
		}
	}

	t.Fatalf("expected day %s", date.Format(time.DateOnly))
	return DayEntry{}
}

func assertDayStats(t *testing.T, day DayEntry, activeGoalCount int, completedCount int, completionRate float64) {
	t.Helper()
	if day.ActiveGoalCount != activeGoalCount {
		t.Fatalf("expected %s active goal count %d, got %d", day.Date.Format(time.DateOnly), activeGoalCount, day.ActiveGoalCount)
	}
	if day.CompletedCount != completedCount {
		t.Fatalf("expected %s completed count %d, got %d", day.Date.Format(time.DateOnly), completedCount, day.CompletedCount)
	}
	if day.CompletionRate != completionRate {
		t.Fatalf("expected %s completion rate %v, got %v", day.Date.Format(time.DateOnly), completionRate, day.CompletionRate)
	}
}
