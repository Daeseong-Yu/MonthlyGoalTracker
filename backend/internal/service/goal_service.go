package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Daeseong-Yu/MonthlyGoalTracker/backend/internal/domain"
)

var (
	ErrInvalidMonth            = errors.New("invalid month")
	ErrEmptyTitle              = errors.New("empty title")
	ErrStartDateOutsideMonth   = errors.New("start date outside month")
	ErrInvalidEndDate          = errors.New("invalid end date")
	ErrActiveGoalLimitExceeded = errors.New("active goal limit exceeded")
)

const maxActiveGoalsPerDay = 5

type GoalRepository interface {
	Create(ctx context.Context, goal *domain.Goal) error
	FindByID(ctx context.Context, id uint) (*domain.Goal, error)
	UpdateTitle(ctx context.Context, id uint, title string) (*domain.Goal, error)
	SetEndDate(ctx context.Context, id uint, endDate *time.Time) (*domain.Goal, error)
	ListOverlappingDateRange(ctx context.Context, startDate, endDate time.Time) ([]domain.Goal, error)
}

type GoalService struct {
	repo GoalRepository
}

func NewGoalService(repo GoalRepository) *GoalService {
	return &GoalService{repo: repo}
}

func (s *GoalService) CreateGoal(ctx context.Context, month string, title string, startDate time.Time) (*domain.Goal, error) {
	monthStart, monthEnd, err := parseMonthRange(month)
	if err != nil {
		return nil, err
	}

	normalizedTitle, err := normalizeTitle(title)
	if err != nil {
		return nil, err
	}

	normalizedStartDate := normalizeDateUTC(startDate)
	if normalizedStartDate.Before(monthStart) || normalizedStartDate.After(monthEnd) {
		return nil, ErrStartDateOutsideMonth
	}

	// MVP policy: the active goal cap is enforced only within the selected month.
	overlappingGoals, err := s.repo.ListOverlappingDateRange(ctx, normalizedStartDate, monthEnd)
	if err != nil {
		return nil, err
	}

	if exceedsActiveGoalLimit(overlappingGoals, normalizedStartDate, monthEnd) {
		return nil, ErrActiveGoalLimitExceeded
	}

	goal := &domain.Goal{
		Title:     normalizedTitle,
		StartDate: normalizedStartDate,
	}
	if err := s.repo.Create(ctx, goal); err != nil {
		return nil, err
	}

	return goal, nil
}

func (s *GoalService) UpdateGoalTitle(ctx context.Context, goalID uint, title string) (*domain.Goal, error) {
	normalizedTitle, err := normalizeTitle(title)
	if err != nil {
		return nil, err
	}

	return s.repo.UpdateTitle(ctx, goalID, normalizedTitle)
}

func (s *GoalService) DeactivateGoal(ctx context.Context, goalID uint, endDate time.Time) (*domain.Goal, error) {
	goal, err := s.repo.FindByID(ctx, goalID)
	if err != nil {
		return nil, err
	}

	normalizedStartDate := normalizeDateUTC(goal.StartDate)
	normalizedEndDate := normalizeDateUTC(endDate)
	if normalizedEndDate.Before(normalizedStartDate) {
		return nil, ErrInvalidEndDate
	}

	return s.repo.SetEndDate(ctx, goalID, &normalizedEndDate)
}

func (s *GoalService) ListGoalsForMonth(ctx context.Context, month string) ([]domain.Goal, error) {
	monthStart, monthEnd, err := parseMonthRange(month)
	if err != nil {
		return nil, err
	}

	return s.repo.ListOverlappingDateRange(ctx, monthStart, monthEnd)
}

func parseMonthRange(month string) (time.Time, time.Time, error) {
	parsedMonth, err := time.Parse("2006-01", month)
	if err != nil || parsedMonth.Format("2006-01") != month {
		return time.Time{}, time.Time{}, fmt.Errorf("%w: %q", ErrInvalidMonth, month)
	}

	monthStart := time.Date(parsedMonth.Year(), parsedMonth.Month(), 1, 0, 0, 0, 0, time.UTC)
	monthEnd := monthStart.AddDate(0, 1, -1)

	return monthStart, monthEnd, nil
}

func normalizeTitle(title string) (string, error) {
	trimmedTitle := strings.TrimSpace(title)
	if trimmedTitle == "" {
		return "", ErrEmptyTitle
	}

	return trimmedTitle, nil
}

func normalizeDateUTC(value time.Time) time.Time {
	year, month, day := value.Date()
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
}

func exceedsActiveGoalLimit(goals []domain.Goal, startDate time.Time, endDate time.Time) bool {
	for day := startDate; !day.After(endDate); day = day.AddDate(0, 0, 1) {
		activeGoals := 0
		for _, goal := range goals {
			if isGoalActiveOnDate(goal, day) {
				activeGoals++
				if activeGoals >= maxActiveGoalsPerDay {
					return true
				}
			}
		}
	}

	return false
}

func isGoalActiveOnDate(goal domain.Goal, day time.Time) bool {
	goalStartDate := normalizeDateUTC(goal.StartDate)
	if day.Before(goalStartDate) {
		return false
	}

	if goal.EndDate == nil {
		return true
	}

	goalEndDate := normalizeDateUTC(*goal.EndDate)
	return !day.After(goalEndDate)
}
