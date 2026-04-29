package repository

import (
	"context"
	"time"

	"github.com/Daeseong-Yu/MonthlyGoalTracker/backend/internal/domain"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type GoalCheckRepository struct {
	db *gorm.DB
}

func NewGoalCheckRepository(database *gorm.DB) *GoalCheckRepository {
	return &GoalCheckRepository{db: database}
}

func (r *GoalCheckRepository) SetCompleted(ctx context.Context, goalID uint, date time.Time, completed bool) error {
	date = normalizeDate(date)

	if !completed {
		return r.db.WithContext(ctx).
			Where("goal_id = ? AND date = ?", goalID, date).
			Delete(&domain.GoalCheck{}).Error
	}

	goalCheck := domain.GoalCheck{
		GoalID: goalID,
		Date:   date,
	}

	return r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "goal_id"}, {Name: "date"}},
			DoNothing: true,
		}).
		Create(&goalCheck).Error
}

func (r *GoalCheckRepository) Exists(ctx context.Context, goalID uint, date time.Time) (bool, error) {
	date = normalizeDate(date)

	var count int64
	if err := r.db.WithContext(ctx).
		Model(&domain.GoalCheck{}).
		Where("goal_id = ? AND date = ?", goalID, date).
		Count(&count).Error; err != nil {
		return false, err
	}

	return count > 0, nil
}

func (r *GoalCheckRepository) ListByDateRange(ctx context.Context, startDate, endDate time.Time) ([]domain.GoalCheck, error) {
	startDate = normalizeDate(startDate)
	endDate = normalizeDate(endDate)

	var goalChecks []domain.GoalCheck
	if err := r.db.WithContext(ctx).
		Where("date BETWEEN ? AND ?", startDate, endDate).
		Order("date ASC, goal_id ASC").
		Find(&goalChecks).Error; err != nil {
		return nil, err
	}

	return goalChecks, nil
}
