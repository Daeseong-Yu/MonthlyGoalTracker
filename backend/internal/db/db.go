package db

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/Daeseong-Yu/MonthlyGoalTracker/backend/internal/domain"
	"github.com/Daeseong-Yu/MonthlyGoalTracker/backend/internal/principal"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var (
	ErrDatabaseURLRequired        = errors.New("database URL is required")
	ErrDatabaseRequired           = errors.New("database is required")
	ErrGoalCheckOwnershipMismatch = errors.New("goal check ownership mismatch")
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

	emailVerificationTableExists := database.Migrator().HasTable(&domain.EmailVerificationToken{})

	if err := database.WithContext(ctx).AutoMigrate(&domain.User{}); err != nil {
		return err
	}
	if err := database.WithContext(ctx).
		Exec(`UPDATE users SET locale = 'ko' WHERE locale IS NULL OR locale = ''`).
		Error; err != nil {
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

	if err := database.WithContext(ctx).AutoMigrate(
		&domain.User{},
		&domain.EmailVerificationToken{},
		&domain.Session{},
		&domain.Goal{},
		&domain.DailyMemo{},
		&domain.GoalCheck{},
	); err != nil {
		return err
	}
	if !emailVerificationTableExists {
		if err := database.WithContext(ctx).
			Model(&domain.User{}).
			Where("email <> '' AND password_hash <> '' AND email_verified_at IS NULL").
			Update("email_verified_at", gorm.Expr("created_at")).
			Error; err != nil {
			return err
		}
	}

	return enforceGoalCheckOwnershipConstraint(ctx, database)
}

func enforceGoalCheckOwnershipConstraint(ctx context.Context, database *gorm.DB) error {
	if err := database.WithContext(ctx).Exec(`CREATE UNIQUE INDEX IF NOT EXISTS idx_goals_id_user ON goals (id, user_id)`).Error; err != nil {
		return err
	}

	var mismatchCount int64
	if err := database.WithContext(ctx).Raw(`
		SELECT count(*)
		FROM goal_checks
		INNER JOIN goals ON goals.id = goal_checks.goal_id
		WHERE goal_checks.user_id <> goals.user_id`,
	).Scan(&mismatchCount).Error; err != nil {
		return err
	}
	if mismatchCount > 0 {
		return fmt.Errorf("%w: %d goal_checks rows reference goals owned by another user", ErrGoalCheckOwnershipMismatch, mismatchCount)
	}

	return database.WithContext(ctx).Exec(`DO $$
BEGIN
	IF NOT EXISTS (
		SELECT 1
		FROM pg_constraint
		WHERE conname = 'fk_goal_checks_goal_user'
			AND conrelid = 'goal_checks'::regclass
	) THEN
		ALTER TABLE goal_checks
			ADD CONSTRAINT fk_goal_checks_goal_user
			FOREIGN KEY (goal_id, user_id)
			REFERENCES goals (id, user_id)
			ON UPDATE CASCADE
			ON DELETE CASCADE;
	END IF;
END $$;`).Error
}

func ensureUser(ctx context.Context, database *gorm.DB, username string) (uint, error) {
	user := domain.User{
		Username: username,
	}

	if err := database.WithContext(ctx).
		Select("Username").
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
