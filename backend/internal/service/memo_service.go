package service

import (
	"context"
	"strings"
	"time"

	"github.com/Daeseong-Yu/MonthlyGoalTracker/backend/internal/domain"
)

type MemoRepository interface {
	Upsert(ctx context.Context, date time.Time, memo string) (*domain.DailyMemo, error)
	FindByDate(ctx context.Context, date time.Time) (*domain.DailyMemo, error)
	ListByDateRange(ctx context.Context, startDate, endDate time.Time) ([]domain.DailyMemo, error)
}

type MemoService struct {
	repo MemoRepository
}

func NewMemoService(repo MemoRepository) *MemoService {
	return &MemoService{repo: repo}
}

func (s *MemoService) SaveMemo(ctx context.Context, date time.Time, memo string) (*domain.DailyMemo, error) {
	normalizedDate := normalizeDateUTC(date)
	trimmedMemo := strings.TrimSpace(memo)

	return s.repo.Upsert(ctx, normalizedDate, trimmedMemo)
}

func (s *MemoService) GetMemo(ctx context.Context, date time.Time) (*domain.DailyMemo, error) {
	normalizedDate := normalizeDateUTC(date)
	return s.repo.FindByDate(ctx, normalizedDate)
}

func (s *MemoService) ListMemosForMonth(ctx context.Context, month string) ([]domain.DailyMemo, error) {
	monthStart, monthEnd, err := parseMonthRange(month)
	if err != nil {
		return nil, err
	}

	return s.repo.ListByDateRange(ctx, monthStart, monthEnd)
}
