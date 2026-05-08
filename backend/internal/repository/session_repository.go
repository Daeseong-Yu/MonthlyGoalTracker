package repository

import (
	"context"
	"time"

	"github.com/Daeseong-Yu/MonthlyGoalTracker/backend/internal/domain"
	"gorm.io/gorm"
)

type SessionRepository struct {
	db *gorm.DB
}

func NewSessionRepository(database *gorm.DB) *SessionRepository {
	return &SessionRepository{db: database}
}

func (r *SessionRepository) Create(ctx context.Context, session *domain.Session) error {
	return r.db.WithContext(ctx).Create(session).Error
}

func (r *SessionRepository) FindByTokenHash(ctx context.Context, tokenHash string, now time.Time) (*domain.Session, error) {
	var session domain.Session
	if err := r.db.WithContext(ctx).
		Preload("User").
		Where("token_hash = ? AND expires_at > ?", tokenHash, now).
		First(&session).Error; err != nil {
		return nil, err
	}

	return &session, nil
}

func (r *SessionRepository) DeleteByTokenHash(ctx context.Context, tokenHash string) error {
	return r.db.WithContext(ctx).Where("token_hash = ?", tokenHash).Delete(&domain.Session{}).Error
}

func (r *SessionRepository) DeleteByUserID(ctx context.Context, userID uint) error {
	return r.db.WithContext(ctx).Where("user_id = ?", userID).Delete(&domain.Session{}).Error
}

func (r *SessionRepository) UpdateLastUsedAt(ctx context.Context, id uint, lastUsedAt time.Time) error {
	return r.db.WithContext(ctx).
		Model(&domain.Session{}).
		Where("id = ?", id).
		Update("last_used_at", lastUsedAt).
		Error
}

func (r *SessionRepository) UpdateCSRFTokenHash(ctx context.Context, id uint, csrfTokenHash string) error {
	return r.db.WithContext(ctx).
		Model(&domain.Session{}).
		Where("id = ?", id).
		Update("csrf_token_hash", csrfTokenHash).
		Error
}
