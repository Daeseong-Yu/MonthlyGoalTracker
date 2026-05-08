package repository

import (
	"context"
	"time"

	"github.com/Daeseong-Yu/MonthlyGoalTracker/backend/internal/domain"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type EmailVerificationRepository struct {
	db *gorm.DB
}

func NewEmailVerificationRepository(database *gorm.DB) *EmailVerificationRepository {
	return &EmailVerificationRepository{db: database}
}

func (r *EmailVerificationRepository) Create(ctx context.Context, token *domain.EmailVerificationToken) error {
	return r.db.WithContext(ctx).Create(token).Error
}

func (r *EmailVerificationRepository) Consume(ctx context.Context, tokenHash string, now time.Time) (*domain.User, error) {
	var token domain.EmailVerificationToken
	var user domain.User

	if err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.WithContext(ctx).
			Clauses(clause.Locking{Strength: "UPDATE"}).
			Preload("User").
			Where("token_hash = ? AND used_at IS NULL AND expires_at > ?", tokenHash, now).
			First(&token).Error; err != nil {
			return err
		}

		usedAt := now
		if err := tx.WithContext(ctx).Model(&token).Update("used_at", usedAt).Error; err != nil {
			return err
		}

		user = token.User
		if user.ID == 0 {
			return gorm.ErrRecordNotFound
		}
		if user.EmailVerifiedAt == nil {
			if err := tx.WithContext(ctx).
				Model(&domain.User{}).
				Where("id = ? AND email_verified_at IS NULL", user.ID).
				Update("email_verified_at", usedAt).
				Error; err != nil {
				return err
			}
			user.EmailVerifiedAt = &usedAt
		}

		return nil
	}); err != nil {
		return nil, err
	}

	return &user, nil
}
