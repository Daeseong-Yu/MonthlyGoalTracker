package db

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/Daeseong-Yu/MonthlyGoalTracker/backend/internal/domain"
	"gorm.io/gorm"
)

func TestConnectRequiresDatabaseURL(t *testing.T) {
	tests := []struct {
		name        string
		databaseURL string
	}{
		{
			name:        "empty",
			databaseURL: "",
		},
		{
			name:        "blank",
			databaseURL: "   ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			database, err := Connect(context.Background(), tt.databaseURL)

			if database != nil {
				t.Fatal("expected nil database when database URL is empty")
			}

			if !errors.Is(err, ErrDatabaseURLRequired) {
				t.Fatalf("expected ErrDatabaseURLRequired, got %v", err)
			}
		})
	}
}

func TestMigrateRequiresDatabase(t *testing.T) {
	err := Migrate(context.Background(), nil)

	if !errors.Is(err, ErrDatabaseRequired) {
		t.Fatalf("expected ErrDatabaseRequired, got %v", err)
	}
}

func TestConnectReturnsErrorForMalformedDatabaseURL(t *testing.T) {
	database, err := Connect(context.Background(), "://")

	if database != nil {
		t.Fatal("expected nil database when database URL is malformed")
	}

	if err == nil {
		t.Fatal("expected error when database URL is malformed")
	}
}

func TestConnectUsesContextForPing(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	database, err := Connect(ctx, "postgres://postgres:postgres@127.0.0.1:5432/monthly_goal_tracker?sslmode=disable")

	if database != nil {
		t.Fatal("expected nil database when context is canceled")
	}

	if err == nil {
		t.Fatal("expected error when context is canceled")
	}

	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
}

func TestConfigureConnectionPoolLimitsDirectRDSConnections(t *testing.T) {
	database, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("expected sql mock database, got %v", err)
	}
	defer database.Close()

	configureConnectionPool(database)

	if got := database.Stats().MaxOpenConnections; got != 1 {
		t.Fatalf("expected max open connections 1, got %d", got)
	}
	if got := defaultMaxIdleConnections; got != 1 {
		t.Fatalf("expected max idle connections 1, got %d", got)
	}
	if got := defaultConnectionIdleTime; got != time.Minute {
		t.Fatalf("expected connection idle time %s, got %s", time.Minute, got)
	}
	if got := defaultConnectionLifetime; got != 5*time.Minute {
		t.Fatalf("expected connection lifetime %s, got %s", 5*time.Minute, got)
	}
}

func TestConnectIntegration(t *testing.T) {
	ctx, cancel, databaseURL := requireDatabaseIntegration(t, "database integration test")
	defer cancel()

	database := openIntegrationDatabase(t, ctx, databaseURL)
	t.Cleanup(func() {
		closeIntegrationDatabase(t, database)
	})

	if err := Migrate(ctx, database); err != nil {
		t.Fatalf("expected migration to succeed, got %v", err)
	}

	assertMigratedConstraints(t, database)
}

func assertMigratedConstraints(t *testing.T, database *gorm.DB) {
	t.Helper()

	baseDate := time.Date(2099, time.January, 1, 0, 0, 0, 0, time.UTC).
		AddDate(0, 0, int(time.Now().Unix()%10000))
	username := "integration-user-" + baseDate.Format("20060102")
	otherUsername := username + "-other"

	cleanupIntegrationRows(t, database, baseDate, username, otherUsername)
	t.Cleanup(func() {
		cleanupIntegrationRows(t, database, baseDate, username, otherUsername)
	})

	user := domain.User{
		Username: username,
	}
	if err := database.Create(&user).Error; err != nil {
		t.Fatalf("expected first user insert to succeed, got %v", err)
	}

	duplicateUser := domain.User{
		Username: username,
	}
	if err := database.Create(&duplicateUser).Error; err == nil {
		t.Fatal("expected duplicate username to fail")
	}

	otherUser := domain.User{
		Username: otherUsername,
	}
	if err := database.Create(&otherUser).Error; err != nil {
		t.Fatalf("expected other user insert to succeed, got %v", err)
	}

	memo := domain.DailyMemo{
		UserID: user.ID,
		Date:   baseDate,
		Memo:   "integration memo",
	}
	if err := database.Create(&memo).Error; err != nil {
		t.Fatalf("expected first memo insert to succeed, got %v", err)
	}

	duplicateMemo := domain.DailyMemo{
		UserID: user.ID,
		Date:   baseDate,
		Memo:   "integration duplicate memo",
	}
	if err := database.Create(&duplicateMemo).Error; err == nil {
		t.Fatal("expected duplicate memo date to fail")
	}

	otherUserMemo := domain.DailyMemo{
		UserID: otherUser.ID,
		Date:   baseDate,
		Memo:   "integration other user memo",
	}
	if err := database.Create(&otherUserMemo).Error; err != nil {
		t.Fatalf("expected same memo date for other user to succeed, got %v", err)
	}

	goal := domain.Goal{
		UserID:    user.ID,
		Title:     "integration constraint goal",
		StartDate: baseDate,
		EndDate:   nil,
	}
	if err := database.Create(&goal).Error; err != nil {
		t.Fatalf("expected open-ended goal insert to succeed, got %v", err)
	}

	invalidEndDate := baseDate
	invalidGoal := domain.Goal{
		UserID:    user.ID,
		Title:     "integration invalid date range goal",
		StartDate: baseDate.AddDate(0, 0, 1),
		EndDate:   &invalidEndDate,
	}
	if err := database.Create(&invalidGoal).Error; err == nil {
		t.Fatal("expected invalid goal date range to fail")
	}

	check := domain.GoalCheck{
		UserID: user.ID,
		GoalID: goal.ID,
		Date:   baseDate,
	}
	if err := database.Create(&check).Error; err != nil {
		t.Fatalf("expected first goal check insert to succeed, got %v", err)
	}

	duplicateCheck := domain.GoalCheck{
		UserID: user.ID,
		GoalID: goal.ID,
		Date:   baseDate,
	}
	if err := database.Create(&duplicateCheck).Error; err == nil {
		t.Fatal("expected duplicate goal check to fail")
	}

	orphanCheck := domain.GoalCheck{
		UserID: user.ID,
		GoalID: goal.ID + 1000000,
		Date:   baseDate.AddDate(0, 0, 1),
	}
	if err := database.Create(&orphanCheck).Error; err == nil {
		t.Fatal("expected orphan goal check to fail")
	}

	crossUserCheck := domain.GoalCheck{
		UserID: otherUser.ID,
		GoalID: goal.ID,
		Date:   baseDate.AddDate(0, 0, 2),
	}
	if err := database.Create(&crossUserCheck).Error; err == nil {
		t.Fatal("expected cross-user goal check to fail")
	}
}

func cleanupIntegrationRows(t *testing.T, database *gorm.DB, baseDate time.Time, usernames ...string) {
	t.Helper()

	dates := []time.Time{
		baseDate,
		baseDate.AddDate(0, 0, 1),
		baseDate.AddDate(0, 0, 2),
	}

	if err := database.Where("date IN ?", dates).Delete(&domain.GoalCheck{}).Error; err != nil {
		t.Fatalf("failed to clean goal checks: %v", err)
	}

	if err := database.Where("date IN ?", dates).Delete(&domain.DailyMemo{}).Error; err != nil {
		t.Fatalf("failed to clean daily memos: %v", err)
	}

	if err := database.Where("title IN ?", []string{
		"integration constraint goal",
		"integration invalid date range goal",
	}).Delete(&domain.Goal{}).Error; err != nil {
		t.Fatalf("failed to clean goals: %v", err)
	}

	if err := database.Where("username IN ?", usernames).Delete(&domain.User{}).Error; err != nil {
		t.Fatalf("failed to clean users: %v", err)
	}
}
