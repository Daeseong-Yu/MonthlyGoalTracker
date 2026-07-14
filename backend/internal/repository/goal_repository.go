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
	userID, err := currentUserID(ctx)
	if err != nil {
		return err
	}

	goal.UserID = userID
	return r.db.WithContext(ctx).Create(goal).Error
}

func (r *GoalRepository) ApplyMonthRollover(
	ctx context.Context,
	copiedGoals []domain.Goal,
	previousGoalIDs []uint,
	previousMonthEnd time.Time,
) error {
	userID, err := currentUserID(ctx)
	if err != nil {
		return err
	}
	if len(copiedGoals) == 0 && len(previousGoalIDs) == 0 {
		return nil
	}

	goalsToCreate := append([]domain.Goal(nil), copiedGoals...)
	for index := range goalsToCreate {
		goalsToCreate[index].UserID = userID
	}

	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if len(goalsToCreate) > 0 {
			if err := tx.Create(&goalsToCreate).Error; err != nil {
				return err
			}
		}

		if len(previousGoalIDs) == 0 {
			return nil
		}

		result := tx.
			Model(&domain.Goal{}).
			Where("user_id = ? AND id IN ?", userID, previousGoalIDs).
			Update("end_date", previousMonthEnd)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != int64(len(previousGoalIDs)) {
			return gorm.ErrRecordNotFound
		}

		return nil
	})
}

func (r *GoalRepository) FindByID(ctx context.Context, id uint) (*domain.Goal, error) {
	scopedDB, _, err := scopedByUser(ctx, r.db)
	if err != nil {
		return nil, err
	}

	var goal domain.Goal
	if err := scopedDB.First(&goal, "id = ?", id).Error; err != nil {
		return nil, err
	}

	return &goal, nil
}

func (r *GoalRepository) UpdateTitle(ctx context.Context, id uint, title string) (*domain.Goal, error) {
	scopedDB, _, err := scopedByUser(ctx, r.db)
	if err != nil {
		return nil, err
	}

	result := scopedDB.
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
	scopedDB, _, err := scopedByUser(ctx, r.db)
	if err != nil {
		return nil, err
	}

	result := scopedDB.
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
	scopedDB, _, err := scopedByUser(ctx, r.db)
	if err != nil {
		return nil, err
	}

	var goals []domain.Goal
	if err := scopedDB.
		Where("start_date <= ? AND (end_date IS NULL OR end_date >= ?)", endDate, startDate).
		Order("start_date ASC, id ASC").
		Find(&goals).Error; err != nil {
		return nil, err
	}

	return goals, nil
}
