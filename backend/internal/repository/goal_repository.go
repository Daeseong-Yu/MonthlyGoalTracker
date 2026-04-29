package repository

import (
	"context"
	"time"

	"github.com/Daeseong-Yu/MonthlyGoalTracker/backend/internal/domain"
	"gorm.io/gorm"
)

type GoalRepository struct {
	db *gorm.DB
}

func NewGoalRepository(database *gorm.DB) *GoalRepository {
	return &GoalRepository{db: database}
}

func (r *GoalRepository) Create(ctx context.Context, goal *domain.Goal) error {
	return r.db.WithContext(ctx).Create(goal).Error
}

func (r *GoalRepository) FindByID(ctx context.Context, id uint) (*domain.Goal, error) {
	var goal domain.Goal
	if err := r.db.WithContext(ctx).First(&goal, id).Error; err != nil {
		return nil, err
	}

	return &goal, nil
}

func (r *GoalRepository) UpdateTitle(ctx context.Context, id uint, title string) (*domain.Goal, error) {
	result := r.db.WithContext(ctx).
		Model(&domain.Goal{}).
		Where("id = ?", id).
		Update("title", title)
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, gorm.ErrRecordNotFound
	}

	goal, err := r.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	return goal, nil
}

func (r *GoalRepository) SetEndDate(ctx context.Context, id uint, endDate *time.Time) (*domain.Goal, error) {
	result := r.db.WithContext(ctx).
		Model(&domain.Goal{}).
		Where("id = ?", id).
		Update("end_date", endDate)
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, gorm.ErrRecordNotFound
	}

	goal, err := r.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	return goal, nil
}

func (r *GoalRepository) ListOverlappingDateRange(ctx context.Context, startDate, endDate time.Time) ([]domain.Goal, error) {
	var goals []domain.Goal
	if err := r.db.WithContext(ctx).
		Where("start_date <= ? AND (end_date IS NULL OR end_date >= ?)", endDate, startDate).
		Order("start_date ASC, id ASC").
		Find(&goals).Error; err != nil {
		return nil, err
	}

	return goals, nil
}
