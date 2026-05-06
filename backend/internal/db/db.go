package db

import (
	"context"
	"errors"
	"strings"

	"github.com/Daeseong-Yu/MonthlyGoalTracker/backend/internal/domain"
	"github.com/Daeseong-Yu/MonthlyGoalTracker/backend/internal/principal"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
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

	if err := database.WithContext(ctx).AutoMigrate(&domain.User{}); err != nil {
		return err
	}

	defaultUserID, err := ensureUser(ctx, database, principal.Default().Username)
	if err != nil {
		return err
	}

	for _, migration := range []struct {
		model       interface{}
		table       string
		dropIndexes []string
	}{
		{model: &domain.Goal{}, table: "goals"},
		{model: &domain.DailyMemo{}, table: "daily_memos", dropIndexes: []string{"idx_daily_memos_date"}},
		{model: &domain.GoalCheck{}, table: "goal_checks"},
	} {
		if !database.Migrator().HasTable(migration.model) {
			continue
		}

		if err := database.WithContext(ctx).
			Exec(`ALTER TABLE ` + migration.table + ` ADD COLUMN IF NOT EXISTS user_id bigint`).
			Error; err != nil {
			return err
		}

		if err := database.WithContext(ctx).
			Exec(`UPDATE `+migration.table+` SET user_id = ? WHERE user_id IS NULL OR user_id = 0`, defaultUserID).
			Error; err != nil {
			return err
		}

		if err := database.WithContext(ctx).
			Exec(`ALTER TABLE ` + migration.table + ` ALTER COLUMN user_id SET NOT NULL`).
			Error; err != nil {
			return err
		}

		for _, index := range migration.dropIndexes {
			if err := database.WithContext(ctx).Exec(`DROP INDEX IF EXISTS ` + index).Error; err != nil {
				return err
			}
		}
	}

	return database.WithContext(ctx).AutoMigrate(
		&domain.User{},
		&domain.Goal{},
		&domain.DailyMemo{},
		&domain.GoalCheck{},
	)
}

func ensureUser(ctx context.Context, database *gorm.DB, username string) (uint, error) {
	user := domain.User{
		Username: username,
	}

	if err := database.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "username"}},
			DoNothing: true,
		}).
		Create(&user).Error; err != nil {
		return 0, err
	}

	if user.ID != 0 {
		return user.ID, nil
	}

	if err := database.WithContext(ctx).
		Where("username = ?", username).
		First(&user).Error; err != nil {
		return 0, err
	}

	return user.ID, nil
}
