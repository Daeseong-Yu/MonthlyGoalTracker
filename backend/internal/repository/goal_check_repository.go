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
	scopedDB, userID, err := scopedByUser(ctx, r.db)
	if err != nil {
		return err
	}

	if !completed {
		return scopedDB.
			Where("goal_id = ? AND date = ?", goalID, date).
			Delete(&domain.GoalCheck{}).Error
	}

	if err := ensureGoalOwnedByCurrentUser(scopedDB, goalID); err != nil {
		return err
	}

	goalCheck := domain.GoalCheck{
		UserID: userID,
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
	scopedDB, _, err := scopedByUser(ctx, r.db)
	if err != nil {
		return false, err
	}

	var count int64
	if err := scopedDB.
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
	scopedDB, _, err := scopedByUser(ctx, r.db)
	if err != nil {
		return nil, err
	}

	var goalChecks []domain.GoalCheck
	if err := scopedDB.
		Where("date BETWEEN ? AND ?", startDate, endDate).
		Order("date ASC, goal_id ASC").
		Find(&goalChecks).Error; err != nil {
		return nil, err
	}

	return goalChecks, nil
}

func ensureGoalOwnedByCurrentUser(scopedDB *gorm.DB, goalID uint) error {
	var count int64
	if err := scopedDB.
		Model(&domain.Goal{}).
		Where("id = ?", goalID).
		Count(&count).Error; err != nil {
		return err
	}
	if count == 0 {
		return gorm.ErrRecordNotFound
	}

	return nil
}
