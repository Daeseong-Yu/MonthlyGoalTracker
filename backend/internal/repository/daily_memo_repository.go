package repository

import (
	"context"
	"time"

	"github.com/Daeseong-Yu/MonthlyGoalTracker/backend/internal/domain"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type DailyMemoRepository struct {
	db *gorm.DB
}

func NewDailyMemoRepository(database *gorm.DB) *DailyMemoRepository {
	return &DailyMemoRepository{db: database}
}

func (r *DailyMemoRepository) Upsert(ctx context.Context, date time.Time, memo string) (*domain.DailyMemo, error) {
	date = normalizeDate(date)

	dailyMemo := domain.DailyMemo{
		Date: date,
		Memo: memo,
	}

	if err := r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "date"}},
			DoUpdates: clause.AssignmentColumns([]string{"memo", "updated_at"}),
		}).
		Create(&dailyMemo).Error; err != nil {
		return nil, err
	}

	return r.FindByDate(ctx, date)
}

func (r *DailyMemoRepository) FindByDate(ctx context.Context, date time.Time) (*domain.DailyMemo, error) {
	date = normalizeDate(date)

	var dailyMemo domain.DailyMemo
	if err := r.db.WithContext(ctx).
		Where("date = ?", date).
		First(&dailyMemo).Error; err != nil {
		return nil, err
	}

	return &dailyMemo, nil
}

func (r *DailyMemoRepository) ListByDateRange(ctx context.Context, startDate, endDate time.Time) ([]domain.DailyMemo, error) {
	startDate = normalizeDate(startDate)
	endDate = normalizeDate(endDate)

	var dailyMemos []domain.DailyMemo
	if err := r.db.WithContext(ctx).
		Where("date BETWEEN ? AND ?", startDate, endDate).
		Order("date ASC").
		Find(&dailyMemos).Error; err != nil {
		return nil, err
	}

	return dailyMemos, nil
}

func normalizeDate(value time.Time) time.Time {
	year, month, day := value.Date()
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
}
