package repository

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func TestGoalCheckRepositorySetCompletedTrueUpsertsCheck(t *testing.T) {
	repo, mock, closeDB := newMockGoalCheckRepository(t)
	defer closeDB()

	const goalID uint = 42
	checkDate := time.Date(2099, time.April, 1, 18, 30, 0, 0, time.FixedZone("KST", 9*60*60))
	normalizedDate := date(2099, time.April, 1)

	mock.ExpectQuery(`INSERT INTO "goal_checks" \("goal_id","date","created_at"\) VALUES \(\$1,\$2,\$3\) ON CONFLICT \("goal_id","date"\) DO NOTHING RETURNING "id"`).
		WithArgs(goalID, normalizedDate, fixedNow()).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(11))

	if err := repo.SetCompleted(context.Background(), goalID, checkDate, true); err != nil {
		t.Fatalf("expected set completed true to succeed, got %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}

func TestGoalCheckRepositorySetCompletedFalseDeletesCheck(t *testing.T) {
	repo, mock, closeDB := newMockGoalCheckRepository(t)
	defer closeDB()

	const goalID uint = 42
	checkDate := time.Date(2099, time.April, 1, 18, 30, 0, 0, time.FixedZone("KST", 9*60*60))
	normalizedDate := date(2099, time.April, 1)

	mock.ExpectExec(`DELETE FROM "goal_checks" WHERE goal_id = \$1 AND date = \$2`).
		WithArgs(goalID, normalizedDate).
		WillReturnResult(sqlmock.NewResult(0, 1))

	if err := repo.SetCompleted(context.Background(), goalID, checkDate, false); err != nil {
		t.Fatalf("expected set completed false to succeed, got %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}

func TestGoalCheckRepositoryExistsReturnsTrue(t *testing.T) {
	repo, mock, closeDB := newMockGoalCheckRepository(t)
	defer closeDB()

	const goalID uint = 42
	checkDate := date(2099, time.April, 1)

	mock.ExpectQuery(`SELECT count\(\*\) FROM "goal_checks" WHERE goal_id = \$1 AND date = \$2`).
		WithArgs(goalID, checkDate).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	exists, err := repo.Exists(context.Background(), goalID, checkDate)
	if err != nil {
		t.Fatalf("expected exists lookup to succeed, got %v", err)
	}
	if !exists {
		t.Fatal("expected check to exist")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}

func TestGoalCheckRepositoryExistsPropagatesError(t *testing.T) {
	repo, mock, closeDB := newMockGoalCheckRepository(t)
	defer closeDB()

	const goalID uint = 42
	checkDate := date(2099, time.April, 1)
	expectedErr := errors.New("count failed")

	mock.ExpectQuery(`SELECT count\(\*\) FROM "goal_checks" WHERE goal_id = \$1 AND date = \$2`).
		WithArgs(goalID, checkDate).
		WillReturnError(expectedErr)

	exists, err := repo.Exists(context.Background(), goalID, checkDate)
	if exists {
		t.Fatal("expected check to not exist on error")
	}
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected count error, got %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}

func TestGoalCheckRepositoryListByDateRangeUsesInclusiveRange(t *testing.T) {
	repo, mock, closeDB := newMockGoalCheckRepository(t)
	defer closeDB()

	startDate := time.Date(2099, time.April, 1, 23, 59, 0, 0, time.FixedZone("KST", 9*60*60))
	endDate := time.Date(2099, time.April, 30, 6, 30, 0, 0, time.FixedZone("EST", -5*60*60))
	normalizedStartDate := date(2099, time.April, 1)
	normalizedEndDate := date(2099, time.April, 30)

	mock.ExpectQuery(`SELECT \* FROM "goal_checks" WHERE date BETWEEN \$1 AND \$2 ORDER BY date ASC, goal_id ASC`).
		WithArgs(normalizedStartDate, normalizedEndDate).
		WillReturnRows(sqlmock.NewRows([]string{"id", "goal_id", "date", "created_at"}).
			AddRow(1, 10, normalizedStartDate, fixedNow()).
			AddRow(2, 11, normalizedEndDate, fixedNow()))

	goalChecks, err := repo.ListByDateRange(context.Background(), startDate, endDate)
	if err != nil {
		t.Fatalf("expected list by date range to succeed, got %v", err)
	}
	if len(goalChecks) != 2 {
		t.Fatalf("expected 2 goal checks, got %d", len(goalChecks))
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}

func newMockGoalCheckRepository(t *testing.T) (*GoalCheckRepository, sqlmock.Sqlmock, func()) {
	t.Helper()

	sqlDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sql mock: %v", err)
	}

	database, err := gorm.Open(postgres.New(postgres.Config{
		Conn:                 sqlDB,
		PreferSimpleProtocol: true,
	}), &gorm.Config{
		DisableAutomaticPing:   true,
		SkipDefaultTransaction: true,
		NowFunc:                fixedNow,
	})
	if err != nil {
		t.Fatalf("failed to create gorm database: %v", err)
	}

	return NewGoalCheckRepository(database), mock, func() {
		_ = sqlDB.Close()
	}
}
