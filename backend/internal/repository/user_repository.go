package repository

import (
	"context"
	"errors"
	"strings"
	"time"

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
		Select("Username").
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

func (r *UserRepository) CreateWithPassword(ctx context.Context, email, passwordHash, locale string, claimLegacy bool, emailVerifiedAt *time.Time) (*domain.User, error) {
	normalizedEmail := normalizeEmail(email)
	var user domain.User

	if err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		legacyUser, hasUnclaimedLegacyUser, err := unclaimedLegacyUser(ctx, tx)
		if err != nil {
			return err
		}
		if hasUnclaimedLegacyUser {
			if !claimLegacy {
				return domain.ErrLegacyClaimRequired
			}

			updates := map[string]any{
				"username":          normalizedEmail,
				"email":             normalizedEmail,
				"password_hash":     passwordHash,
				"locale":            locale,
				"email_verified_at": emailVerifiedAt,
			}
			if err := tx.WithContext(ctx).Model(&domain.User{}).
				Where("id = ?", legacyUser.ID).
				Updates(updates).Error; err != nil {
				return err
			}

			return tx.WithContext(ctx).First(&user, legacyUser.ID).Error
		}

		user = domain.User{
			Username:        normalizedEmail,
			Email:           normalizedEmail,
			PasswordHash:    passwordHash,
			Locale:          locale,
			EmailVerifiedAt: emailVerifiedAt,
		}

		return tx.WithContext(ctx).Create(&user).Error
	}); err != nil {
		return nil, err
	}

	return &user, nil
}

func (r *UserRepository) FindByEmail(ctx context.Context, email string) (*domain.User, error) {
	var user domain.User
	if err := r.db.WithContext(ctx).
		Where("email = ?", normalizeEmail(email)).
		First(&user).Error; err != nil {
		return nil, err
	}

	return &user, nil
}

func (r *UserRepository) FindByID(ctx context.Context, id uint) (*domain.User, error) {
	var user domain.User
	if err := r.db.WithContext(ctx).First(&user, id).Error; err != nil {
		return nil, err
	}

	return &user, nil
}

func (r *UserRepository) UpdateLocale(ctx context.Context, id uint, locale string) (*domain.User, error) {
	var user domain.User
	if err := r.db.WithContext(ctx).First(&user, id).Error; err != nil {
		return nil, err
	}

	if err := r.db.WithContext(ctx).Model(&user).Update("locale", locale).Error; err != nil {
		return nil, err
	}
	user.Locale = locale

	return &user, nil
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func unclaimedLegacyUser(ctx context.Context, tx *gorm.DB) (*domain.User, bool, error) {
	var legacyUser domain.User
	err := tx.WithContext(ctx).
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("username = ?", principal.Default().Username).
		First(&legacyUser).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	if strings.TrimSpace(legacyUser.Email) != "" || strings.TrimSpace(legacyUser.PasswordHash) != "" {
		return nil, false, nil
	}

	return &legacyUser, true, nil
}
