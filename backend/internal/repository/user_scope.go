package repository

import (
	"context"
	"errors"

	"github.com/Daeseong-Yu/MonthlyGoalTracker/backend/internal/principal"
	"gorm.io/gorm"
)

var ErrUserContextRequired = errors.New("resolved user context required")

func currentUserID(ctx context.Context) (uint, error) {
	user, ok := principal.UserFromContext(ctx)
	if !ok {
		return 0, ErrUserContextRequired
	}

	return user.ID, nil
}

func scopedByUser(ctx context.Context, database *gorm.DB) (*gorm.DB, uint, error) {
	userID, err := currentUserID(ctx)
	if err != nil {
		return nil, 0, err
	}

	return database.WithContext(ctx).Where("user_id = ?", userID), userID, nil
}
