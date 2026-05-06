package db

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/Daeseong-Yu/MonthlyGoalTracker/backend/internal/domain"
	"github.com/Daeseong-Yu/MonthlyGoalTracker/backend/internal/principal"
	"github.com/Daeseong-Yu/MonthlyGoalTracker/backend/internal/repository"
	"gorm.io/gorm"
)

func TestMigrateRehearsesLegacySchemaIntegration(t *testing.T) {
	if os.Getenv("RUN_DB_INTEGRATION") != "1" {
		t.Skip("set RUN_DB_INTEGRATION=1 to run database migration rehearsal test")
	}

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Fatal("DATABASE_URL is required for database migration rehearsal test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	adminDB := openIntegrationDatabaseWithoutMigration(t, ctx, databaseURL)
	t.Cleanup(func() {
		closeIntegrationDatabase(t, adminDB)
	})

	schemaName := fmt.Sprintf("migration_rehearsal_%d", time.Now().UTC().UnixNano())
	createSchema(t, adminDB, schemaName)

	schemaDB := openIntegrationDatabaseWithoutMigration(t, ctx, databaseURLWithSearchPath(t, databaseURL, schemaName))
	t.Cleanup(func() {
		dropSchema(t, adminDB, schemaName)
	})
	t.Cleanup(func() {
		closeIntegrationDatabase(t, schemaDB)
	})

	legacy := seedLegacySchema(t, schemaDB)

	if err := Migrate(ctx, schemaDB); err != nil {
		t.Fatalf("expected migration rehearsal to succeed, got %v", err)
	}

	assertDailyMemoIndexesMigrated(t, schemaDB)

	var defaultUser domain.User
	if err := schemaDB.WithContext(ctx).
		Where("username = ?", principal.Default().Username).
		First(&defaultUser).Error; err != nil {
		t.Fatalf("expected default user to exist after migration, got %v", err)
	}
	if defaultUser.ID == 0 {
		t.Fatal("expected default user ID to be set after migration")
	}

	assertBackfilledRecordUserID(t, schemaDB, &domain.Goal{}, legacy.goalID, defaultUser.ID)
	assertBackfilledRecordUserID(t, schemaDB, &domain.DailyMemo{}, legacy.memoID, defaultUser.ID)
	assertBackfilledRecordUserID(t, schemaDB, &domain.GoalCheck{}, legacy.goalCheckID, defaultUser.ID)

	userRepo := repository.NewUserRepository(schemaDB)
	dailyMemoRepo := repository.NewDailyMemoRepository(schemaDB)
	goalRepo := repository.NewGoalRepository(schemaDB)
	goalCheckRepo := repository.NewGoalCheckRepository(schemaDB)

	defaultCtx := principal.WithUser(context.Background(), defaultUser)
	otherUser, err := userRepo.EnsureByUsername(context.Background(), legacy.prefix+" other user")
	if err != nil {
		t.Fatalf("expected other user creation to succeed, got %v", err)
	}
	otherCtx := principal.WithUser(context.Background(), *otherUser)

	legacyMemo, err := dailyMemoRepo.FindByDate(defaultCtx, legacy.memoDate)
	if err != nil {
		t.Fatalf("expected legacy memo lookup for default user to succeed, got %v", err)
	}
	if legacyMemo.ID != legacy.memoID {
		t.Fatalf("expected legacy memo ID %d, got %d", legacy.memoID, legacyMemo.ID)
	}

	if _, err := dailyMemoRepo.FindByDate(otherCtx, legacy.memoDate); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("expected legacy memo to be hidden from other user, got %v", err)
	}

	otherUserMemo := domain.DailyMemo{
		UserID: otherUser.ID,
		Date:   legacy.memoDate,
		Memo:   legacy.prefix + " other user memo",
	}
	if err := schemaDB.WithContext(ctx).Create(&otherUserMemo).Error; err != nil {
		t.Fatalf("expected same memo date for other user to succeed after migration, got %v", err)
	}

	duplicateLegacyUserMemo := domain.DailyMemo{
		UserID: defaultUser.ID,
		Date:   legacy.memoDate,
		Memo:   legacy.prefix + " duplicate default user memo",
	}
	if err := schemaDB.WithContext(ctx).Create(&duplicateLegacyUserMemo).Error; err == nil {
		t.Fatal("expected duplicate memo date for same user to fail after migration")
	}

	otherFoundMemo, err := dailyMemoRepo.FindByDate(otherCtx, legacy.memoDate)
	if err != nil {
		t.Fatalf("expected other user memo lookup to succeed, got %v", err)
	}
	if otherFoundMemo.ID != otherUserMemo.ID {
		t.Fatalf("expected other user memo ID %d, got %d", otherUserMemo.ID, otherFoundMemo.ID)
	}

	legacyGoal, err := goalRepo.FindByID(defaultCtx, legacy.goalID)
	if err != nil {
		t.Fatalf("expected legacy goal lookup for default user to succeed, got %v", err)
	}
	if legacyGoal.UserID != defaultUser.ID {
		t.Fatalf("expected legacy goal user ID %d, got %d", defaultUser.ID, legacyGoal.UserID)
	}

	if _, err := goalRepo.FindByID(otherCtx, legacy.goalID); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("expected legacy goal to be hidden from other user, got %v", err)
	}

	otherGoal := &domain.Goal{
		Title:     legacy.prefix + " other user goal",
		StartDate: legacy.memoDate,
	}
	if err := goalRepo.Create(otherCtx, otherGoal); err != nil {
		t.Fatalf("expected other user goal creation to succeed, got %v", err)
	}

	if err := goalCheckRepo.SetCompleted(otherCtx, otherGoal.ID, legacy.memoDate, true); err != nil {
		t.Fatalf("expected other user goal check creation to succeed, got %v", err)
	}

	defaultChecks, err := goalCheckRepo.ListByDateRange(defaultCtx, legacy.memoDate, legacy.memoDate)
	if err != nil {
		t.Fatalf("expected default user goal check lookup to succeed, got %v", err)
	}
	if !containsGoalCheck(defaultChecks, legacy.goalCheckID, legacy.goalID, defaultUser.ID, legacy.memoDate) {
		t.Fatalf("expected legacy goal check %d for default user in results", legacy.goalCheckID)
	}
	if containsGoalCheckForGoal(defaultChecks, otherGoal.ID, otherUser.ID, legacy.memoDate) {
		t.Fatal("expected default user goal checks to exclude other user's rows")
	}

	otherChecks, err := goalCheckRepo.ListByDateRange(otherCtx, legacy.memoDate, legacy.memoDate)
	if err != nil {
		t.Fatalf("expected other user goal check lookup to succeed, got %v", err)
	}
	if !containsGoalCheckForGoal(otherChecks, otherGoal.ID, otherUser.ID, legacy.memoDate) {
		t.Fatalf("expected other user goal check for goal %d in results", otherGoal.ID)
	}
	if containsGoalCheck(otherChecks, legacy.goalCheckID, legacy.goalID, defaultUser.ID, legacy.memoDate) {
		t.Fatal("expected other user goal checks to exclude legacy default user rows")
	}

	if err := goalCheckRepo.SetCompleted(defaultCtx, otherGoal.ID, legacy.memoDate.AddDate(0, 0, 1), true); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("expected cross-user goal check write to be rejected, got %v", err)
	}
}

type legacyFixture struct {
	prefix      string
	goalID      uint
	memoID      uint
	goalCheckID uint
	memoDate    time.Time
}

func openIntegrationDatabaseWithoutMigration(t *testing.T, ctx context.Context, databaseURL string) *gorm.DB {
	t.Helper()

	var (
		database *gorm.DB
		err      error
	)
	for {
		database, err = Connect(ctx, databaseURL)
		if err == nil {
			return database
		}
		if ctx.Err() != nil {
			t.Fatalf("expected database connection, got %v", err)
		}

		time.Sleep(500 * time.Millisecond)
	}
}

func closeIntegrationDatabase(t *testing.T, database *gorm.DB) {
	t.Helper()

	sqlDB, err := database.DB()
	if err != nil {
		t.Fatalf("expected sql database handle, got %v", err)
	}

	if err := sqlDB.Close(); err != nil {
		t.Fatalf("failed to close database connection: %v", err)
	}
}

func createSchema(t *testing.T, database *gorm.DB, schemaName string) {
	t.Helper()

	if err := database.Exec(`CREATE SCHEMA ` + quoteIdentifier(schemaName)).Error; err != nil {
		t.Fatalf("failed to create schema %q: %v", schemaName, err)
	}
}

func dropSchema(t *testing.T, database *gorm.DB, schemaName string) {
	t.Helper()

	if err := database.Exec(`DROP SCHEMA IF EXISTS ` + quoteIdentifier(schemaName) + ` CASCADE`).Error; err != nil {
		t.Fatalf("failed to drop schema %q: %v", schemaName, err)
	}
}

func databaseURLWithSearchPath(t *testing.T, databaseURL, schemaName string) string {
	t.Helper()

	parsedURL, err := url.Parse(databaseURL)
	if err != nil {
		t.Fatalf("failed to parse database URL: %v", err)
	}

	query := parsedURL.Query()
	query.Set("search_path", schemaName)
	parsedURL.RawQuery = query.Encode()
	return parsedURL.String()
}

func seedLegacySchema(t *testing.T, database *gorm.DB) legacyFixture {
	t.Helper()

	prefix := fmt.Sprintf("migration rehearsal %d", time.Now().UTC().UnixNano())
	memoDate := time.Date(2099, time.March, 14, 0, 0, 0, 0, time.UTC)

	for _, statement := range []string{
		`CREATE TABLE goals (
			id bigserial PRIMARY KEY,
			title varchar(100) NOT NULL,
			start_date date NOT NULL,
			end_date date,
			created_at timestamptz NOT NULL,
			updated_at timestamptz NOT NULL,
			CONSTRAINT chk_goals_date_range CHECK (end_date IS NULL OR start_date <= end_date)
		)`,
		`CREATE INDEX idx_goals_start_date ON goals (start_date)`,
		`CREATE INDEX idx_goals_end_date ON goals (end_date)`,
		`CREATE TABLE daily_memos (
			id bigserial PRIMARY KEY,
			date date NOT NULL,
			memo text NOT NULL,
			created_at timestamptz NOT NULL,
			updated_at timestamptz NOT NULL
		)`,
		`CREATE UNIQUE INDEX idx_daily_memos_date ON daily_memos (date)`,
		`CREATE TABLE goal_checks (
			id bigserial PRIMARY KEY,
			goal_id bigint NOT NULL REFERENCES goals(id) ON UPDATE CASCADE ON DELETE CASCADE,
			date date NOT NULL,
			created_at timestamptz NOT NULL
		)`,
		`CREATE UNIQUE INDEX idx_goal_checks_goal_date ON goal_checks (goal_id, date)`,
		`CREATE INDEX idx_goal_checks_date ON goal_checks (date)`,
	} {
		if err := database.Exec(statement).Error; err != nil {
			t.Fatalf("failed to create legacy schema: %v", err)
		}
	}

	timestamp := time.Date(2099, time.March, 10, 9, 0, 0, 0, time.UTC)
	legacyGoal := struct {
		ID uint
	}{}
	if err := database.Raw(
		`INSERT INTO goals (title, start_date, end_date, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?)
		 RETURNING id`,
		prefix+" legacy goal",
		memoDate,
		nil,
		timestamp,
		timestamp,
	).Scan(&legacyGoal).Error; err != nil {
		t.Fatalf("failed to insert legacy goal: %v", err)
	}

	legacyMemo := struct {
		ID uint
	}{}
	if err := database.Raw(
		`INSERT INTO daily_memos (date, memo, created_at, updated_at)
		 VALUES (?, ?, ?, ?)
		 RETURNING id`,
		memoDate,
		prefix+" legacy memo",
		timestamp,
		timestamp,
	).Scan(&legacyMemo).Error; err != nil {
		t.Fatalf("failed to insert legacy daily memo: %v", err)
	}

	legacyGoalCheck := struct {
		ID uint
	}{}
	if err := database.Raw(
		`INSERT INTO goal_checks (goal_id, date, created_at)
		 VALUES (?, ?, ?)
		 RETURNING id`,
		legacyGoal.ID,
		memoDate,
		timestamp,
	).Scan(&legacyGoalCheck).Error; err != nil {
		t.Fatalf("failed to insert legacy goal check: %v", err)
	}

	return legacyFixture{
		prefix:      prefix,
		goalID:      legacyGoal.ID,
		memoID:      legacyMemo.ID,
		goalCheckID: legacyGoalCheck.ID,
		memoDate:    memoDate,
	}
}

func assertDailyMemoIndexesMigrated(t *testing.T, database *gorm.DB) {
	t.Helper()

	var oldIndexCount int64
	if err := database.Raw(
		`SELECT COUNT(*) FROM pg_indexes WHERE schemaname = current_schema() AND indexname = 'idx_daily_memos_date'`,
	).Scan(&oldIndexCount).Error; err != nil {
		t.Fatalf("failed to inspect legacy daily memo index: %v", err)
	}
	if oldIndexCount != 0 {
		t.Fatalf("expected legacy idx_daily_memos_date index to be removed, found %d entries", oldIndexCount)
	}

	var indexDefinition string
	if err := database.Raw(
		`SELECT indexdef FROM pg_indexes WHERE schemaname = current_schema() AND indexname = 'idx_daily_memos_user_date'`,
	).Scan(&indexDefinition).Error; err != nil {
		t.Fatalf("failed to inspect migrated daily memo index: %v", err)
	}
	if indexDefinition == "" {
		t.Fatal("expected idx_daily_memos_user_date to exist after migration")
	}
	if !strings.Contains(indexDefinition, "UNIQUE INDEX") || !strings.Contains(indexDefinition, "(user_id, date)") {
		t.Fatalf("expected idx_daily_memos_user_date to enforce unique (user_id, date), got %q", indexDefinition)
	}
}

func assertBackfilledRecordUserID(t *testing.T, database *gorm.DB, model interface{}, id uint, expectedUserID uint) {
	t.Helper()

	switch typedModel := model.(type) {
	case *domain.Goal:
		if err := database.First(typedModel, "id = ?", id).Error; err != nil {
			t.Fatalf("expected goal %d to exist, got %v", id, err)
		}
		if typedModel.UserID != expectedUserID {
			t.Fatalf("expected goal %d to be backfilled with user ID %d, got %d", id, expectedUserID, typedModel.UserID)
		}
	case *domain.DailyMemo:
		if err := database.First(typedModel, "id = ?", id).Error; err != nil {
			t.Fatalf("expected daily memo %d to exist, got %v", id, err)
		}
		if typedModel.UserID != expectedUserID {
			t.Fatalf("expected daily memo %d to be backfilled with user ID %d, got %d", id, expectedUserID, typedModel.UserID)
		}
	case *domain.GoalCheck:
		if err := database.First(typedModel, "id = ?", id).Error; err != nil {
			t.Fatalf("expected goal check %d to exist, got %v", id, err)
		}
		if typedModel.UserID != expectedUserID {
			t.Fatalf("expected goal check %d to be backfilled with user ID %d, got %d", id, expectedUserID, typedModel.UserID)
		}
	default:
		t.Fatalf("unsupported model type %T", model)
	}
}

func containsGoalCheck(goalChecks []domain.GoalCheck, goalCheckID, goalID, userID uint, date time.Time) bool {
	for _, goalCheck := range goalChecks {
		if goalCheck.ID == goalCheckID &&
			goalCheck.GoalID == goalID &&
			goalCheck.UserID == userID &&
			goalCheck.Date.Equal(date) {
			return true
		}
	}

	return false
}

func containsGoalCheckForGoal(goalChecks []domain.GoalCheck, goalID, userID uint, date time.Time) bool {
	for _, goalCheck := range goalChecks {
		if goalCheck.GoalID == goalID &&
			goalCheck.UserID == userID &&
			goalCheck.Date.Equal(date) {
			return true
		}
	}

	return false
}

func quoteIdentifier(value string) string {
	return `"` + strings.ReplaceAll(value, `"`, `""`) + `"`
}
