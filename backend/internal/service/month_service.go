package service

import (
	"context"
	"time"

	"github.com/Daeseong-Yu/MonthlyGoalTracker/backend/internal/domain"
)

type MonthGoalRepository interface {
	ApplyMonthRollover(
		ctx context.Context,
		copiedGoals []domain.Goal,
		previousGoalIDs []uint,
		previousMonthEnd time.Time,
	) error
	ListOverlappingDateRange(ctx context.Context, startDate, endDate time.Time) ([]domain.Goal, error)
}

type MonthMemoRepository interface {
	ListByDateRange(ctx context.Context, startDate, endDate time.Time) ([]domain.DailyMemo, error)
}

type MonthCheckRepository interface {
	ListByDateRange(ctx context.Context, startDate, endDate time.Time) ([]domain.GoalCheck, error)
}

type MonthService struct {
	goalRepo  MonthGoalRepository
	memoRepo  MonthMemoRepository
	checkRepo MonthCheckRepository
}

type MonthView struct {
	Month  string
	Goals  []domain.Goal
	Days   []DayEntry
	Checks []domain.GoalCheck
	Chart  []ChartPoint
}

type DayEntry struct {
	Date            time.Time
	Memo            string
	ActiveGoalCount int
	CompletedCount  int
	CompletionRate  float64
}

type ChartPoint struct {
	Date            time.Time
	ActiveGoalCount int
	CompletedCount  int
	CompletionRate  float64
}

func NewMonthService(goalRepo MonthGoalRepository, memoRepo MonthMemoRepository, checkRepo MonthCheckRepository) *MonthService {
	return &MonthService{
		goalRepo:  goalRepo,
		memoRepo:  memoRepo,
		checkRepo: checkRepo,
	}
}

func (s *MonthService) EnsureMonth(ctx context.Context, month string) (*MonthView, error) {
	monthStart, _, err := parseMonthRange(month)
	if err != nil {
		return nil, err
	}

	previousMonthEnd := monthStart.AddDate(0, 0, -1)

	currentGoals, err := s.ListGoalsForMonth(ctx, month)
	if err != nil {
		return nil, err
	}

	previousGoals, err := s.goalRepo.ListOverlappingDateRange(ctx, previousMonthEnd, previousMonthEnd)
	if err != nil {
		return nil, err
	}

	replacementCounts := countMonthStartGoalsByTitle(currentGoals, monthStart)
	copiedGoals := make([]domain.Goal, 0, len(previousGoals))
	previousGoalIDsToEnd := make([]uint, 0, len(previousGoals))
	for _, previousGoal := range previousGoals {
		if replacementCounts[previousGoal.Title] > 0 {
			replacementCounts[previousGoal.Title]--
		} else {
			copiedGoals = append(copiedGoals, domain.Goal{
				Title:     previousGoal.Title,
				StartDate: monthStart,
			})
		}

		if previousGoal.EndDate == nil || normalizeDateUTC(*previousGoal.EndDate).After(previousMonthEnd) {
			previousGoalIDsToEnd = append(previousGoalIDsToEnd, previousGoal.ID)
		}
	}
	if err := s.goalRepo.ApplyMonthRollover(ctx, copiedGoals, previousGoalIDsToEnd, previousMonthEnd); err != nil {
		return nil, err
	}

	return s.GetMonthView(ctx, month)
}

func (s *MonthService) GetMonthView(ctx context.Context, month string) (*MonthView, error) {
	monthStart, monthEnd, err := parseMonthRange(month)
	if err != nil {
		return nil, err
	}

	goals, err := s.goalRepo.ListOverlappingDateRange(ctx, monthStart, monthEnd)
	if err != nil {
		return nil, err
	}

	memos, err := s.memoRepo.ListByDateRange(ctx, monthStart, monthEnd)
	if err != nil {
		return nil, err
	}

	checks, err := s.checkRepo.ListByDateRange(ctx, monthStart, monthEnd)
	if err != nil {
		return nil, err
	}

	days := buildDayEntries(monthStart, monthEnd, goals, memos, checks)
	chart := make([]ChartPoint, 0, len(days))
	for _, day := range days {
		chart = append(chart, ChartPoint{
			Date:            day.Date,
			ActiveGoalCount: day.ActiveGoalCount,
			CompletedCount:  day.CompletedCount,
			CompletionRate:  day.CompletionRate,
		})
	}

	return &MonthView{
		Month:  month,
		Goals:  goals,
		Days:   days,
		Checks: checks,
		Chart:  chart,
	}, nil
}

func (s *MonthService) ListGoalsForMonth(ctx context.Context, month string) ([]domain.Goal, error) {
	monthStart, monthEnd, err := parseMonthRange(month)
	if err != nil {
		return nil, err
	}

	return s.goalRepo.ListOverlappingDateRange(ctx, monthStart, monthEnd)
}

func buildDayEntries(monthStart, monthEnd time.Time, goals []domain.Goal, memos []domain.DailyMemo, checks []domain.GoalCheck) []DayEntry {
	memosByDate := make(map[string]string, len(memos))
	for _, memo := range memos {
		memosByDate[dateKey(memo.Date)] = memo.Memo
	}

	checksByDate := make(map[string]map[uint]bool)
	for _, check := range checks {
		key := dateKey(check.Date)
		if checksByDate[key] == nil {
			checksByDate[key] = make(map[uint]bool)
		}
		checksByDate[key][check.GoalID] = true
	}

	days := make([]DayEntry, 0, int(monthEnd.Sub(monthStart).Hours()/24)+1)
	for day := monthStart; !day.After(monthEnd); day = day.AddDate(0, 0, 1) {
		key := dateKey(day)
		activeGoalCount := 0
		completedCount := 0

		for _, goal := range goals {
			if !isGoalActiveOnDate(goal, day) {
				continue
			}

			activeGoalCount++
			if checksByDate[key][goal.ID] {
				completedCount++
			}
		}

		completionRate := 0.0
		if activeGoalCount > 0 {
			completionRate = float64(completedCount) / float64(activeGoalCount)
		}

		days = append(days, DayEntry{
			Date:            day,
			Memo:            memosByDate[key],
			ActiveGoalCount: activeGoalCount,
			CompletedCount:  completedCount,
			CompletionRate:  completionRate,
		})
	}

	return days
}

func countMonthStartGoalsByTitle(goals []domain.Goal, monthStart time.Time) map[string]int {
	counts := make(map[string]int)
	for _, goal := range goals {
		if normalizeDateUTC(goal.StartDate).Equal(monthStart) {
			counts[goal.Title]++
		}
	}

	return counts
}

func dateKey(date time.Time) string {
	return normalizeDateUTC(date).Format(time.DateOnly)
}
