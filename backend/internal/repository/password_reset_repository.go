package repository

import (
	"context"
	"time"

	"github.com/Daeseong-Yu/MonthlyGoalTracker/backend/internal/domain"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type PasswordResetRepository struct {
	db *gorm.DB
}

func NewPasswordResetRepository(database *gorm.DB) *PasswordResetRepository {
	return &PasswordResetRepository{db: database}
}

func (r *PasswordResetRepository) Create(ctx context.Context, token *domain.PasswordResetToken) error {
	return r.db.WithContext(ctx).Create(token).Error
}

func (r *PasswordResetRepository) Consume(ctx context.Context, tokenHash, passwordHash string, now time.Time) (*domain.User, error) {
	var token domain.PasswordResetToken
	var user domain.User

	if err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.WithContext(ctx).
			Clauses(clause.Locking{Strength: "UPDATE"}).
			Preload("User").
			Where("token_hash = ? AND used_at IS NULL AND expires_at > ?", tokenHash, now).
			First(&token).Error; err != nil {
			return err
		}

		user = token.User
		if user.ID == 0 {
			return gorm.ErrRecordNotFound
		}

		usedAt := now
		if err := tx.WithContext(ctx).Model(&token).Update("used_at", usedAt).Error; err != nil {
			return err
		}
		if err := tx.WithContext(ctx).
			Model(&domain.PasswordResetToken{}).
			Where("user_id = ? AND used_at IS NULL", user.ID).
			Update("used_at", usedAt).
			Error; err != nil {
			return err
		}
		if err := tx.WithContext(ctx).
			Model(&domain.User{}).
			Where("id = ?", user.ID).
			Update("password_hash", passwordHash).
			Error; err != nil {
			return err
		}

		user.PasswordHash = passwordHash
		return nil
	}); err != nil {
		return nil, err
	}

	return &user, nil
}
