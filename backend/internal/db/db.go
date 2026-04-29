package db

import (
	"context"
	"errors"
	"strings"

	"github.com/Daeseong-Yu/MonthlyGoalTracker/backend/internal/domain"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var (
	ErrDatabaseURLRequired = errors.New("database URL is required")
	ErrDatabaseRequired    = errors.New("database is required")
)

func Connect(ctx context.Context, databaseURL string) (*gorm.DB, error) {
	if strings.TrimSpace(databaseURL) == "" {
		return nil, ErrDatabaseURLRequired
	}

	database, err := gorm.Open(postgres.Open(databaseURL), &gorm.Config{
		DisableAutomaticPing: true,
	})
	if err != nil {
		return nil, err
	}

	sqlDB, err := database.DB()
	if err != nil {
		return nil, err
	}

	if err := sqlDB.PingContext(ctx); err != nil {
		_ = sqlDB.Close()
		return nil, err
	}

	return database, nil
}

func Migrate(ctx context.Context, database *gorm.DB) error {
	if database == nil {
		return ErrDatabaseRequired
	}

	return database.WithContext(ctx).AutoMigrate(
		&domain.Goal{},
		&domain.DailyMemo{},
		&domain.GoalCheck{},
	)
}
