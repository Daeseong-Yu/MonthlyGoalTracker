package repository

import (
	"context"

	"github.com/Daeseong-Yu/MonthlyGoalTracker/backend/internal/domain"
	"github.com/Daeseong-Yu/MonthlyGoalTracker/backend/internal/principal"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type UserRepository struct {
	db *gorm.DB
}

func NewUserRepository(database *gorm.DB) *UserRepository {
	return &UserRepository{db: database}
}

func (r *UserRepository) EnsureByUsername(ctx context.Context, username string) (*domain.User, error) {
	normalizedUsername := principal.NormalizeUsername(username)

	user := domain.User{
		Username: normalizedUsername,
	}

	if err := r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "username"}},
			DoNothing: true,
		}).
		Create(&user).Error; err != nil {
		return nil, err
	}

	if user.ID != 0 {
		return &user, nil
	}

	if err := r.db.WithContext(ctx).
		Where("username = ?", normalizedUsername).
		First(&user).Error; err != nil {
		return nil, err
	}

	return &user, nil
}
