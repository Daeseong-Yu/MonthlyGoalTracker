package service

import (
	"context"
	"errors"
	"time"

	"github.com/Daeseong-Yu/MonthlyGoalTracker/backend/internal/domain"
)

var ErrGoalNotActiveOnDate = errors.New("goal not active on date")

type CheckGoalRepository interface {
	FindByID(ctx context.Context, id uint) (*domain.Goal, error)
}

type GoalCheckRepository interface {
	SetCompleted(ctx context.Context, goalID uint, date time.Time, completed bool) error
	ListByDateRange(ctx context.Context, startDate, endDate time.Time) ([]domain.GoalCheck, error)
}

type CheckService struct {
	goalRepo  CheckGoalRepository
	checkRepo GoalCheckRepository
}

func NewCheckService(goalRepo CheckGoalRepository, checkRepo GoalCheckRepository) *CheckService {
	return &CheckService{
		goalRepo:  goalRepo,
		checkRepo: checkRepo,
	}
}

func (s *CheckService) SetGoalCompleted(ctx context.Context, goalID uint, date time.Time, completed bool) error {
	goal, err := s.goalRepo.FindByID(ctx, goalID)
	if err != nil {
		return err
	}

	normalizedDate := normalizeDateUTC(date)
	if goal == nil || !isGoalActiveOnDate(*goal, normalizedDate) {
		return ErrGoalNotActiveOnDate
	}

	return s.checkRepo.SetCompleted(ctx, goalID, normalizedDate, completed)
}

func (s *CheckService) ListChecksForMonth(ctx context.Context, month string) ([]domain.GoalCheck, error) {
	monthStart, monthEnd, err := parseMonthRange(month)
	if err != nil {
		return nil, err
	}

	return s.checkRepo.ListByDateRange(ctx, monthStart, monthEnd)
}
