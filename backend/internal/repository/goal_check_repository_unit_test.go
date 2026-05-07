package repository

import (
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"gorm.io/gorm"
)

func TestGoalCheckRepositorySetCompletedTrueUpsertsCheck(t *testing.T) {
	repo, mock, closeDB := newMockGoalCheckRepository(t)
	defer closeDB()

	const goalID uint = 42
	checkDate := time.Date(2099, time.April, 1, 18, 30, 0, 0, time.FixedZone("KST", 9*60*60))
	normalizedDate := date(2099, time.April, 1)

	mock.ExpectQuery(`SELECT count\(\*\) FROM "goals" WHERE user_id = \$1 AND id = \$2`).
		WithArgs(testUserID, goalID).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	mock.ExpectQuery(`INSERT INTO "goal_checks" \("user_id","goal_id","date","created_at"\) VALUES \(\$1,\$2,\$3,\$4\) ON CONFLICT \("goal_id","date"\) DO NOTHING RETURNING "id"`).
		WithArgs(testUserID, goalID, normalizedDate, fixedNow()).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(11))

	if err := repo.SetCompleted(scopedUserContext(), goalID, checkDate, true); err != nil {
		t.Fatalf("expected set completed true to succeed, got %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}

func TestGoalCheckRepositorySetCompletedTrueRejectsGoalOutsideCurrentUser(t *testing.T) {
	repo, mock, closeDB := newMockGoalCheckRepository(t)
	defer closeDB()

	const goalID uint = 42
	checkDate := date(2099, time.April, 1)

	mock.ExpectQuery(`SELECT count\(\*\) FROM "goals" WHERE user_id = \$1 AND id = \$2`).
		WithArgs(testUserID, goalID).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

	err := repo.SetCompleted(scopedUserContext(), goalID, checkDate, true)
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("expected ErrRecordNotFound, got %v", err)
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

	mock.ExpectExec(`DELETE FROM "goal_checks" WHERE user_id = \$1 AND \(goal_id = \$2 AND date = \$3\)`).
		WithArgs(testUserID, goalID, normalizedDate).
		WillReturnResult(sqlmock.NewResult(0, 1))

	if err := repo.SetCompleted(scopedUserContext(), goalID, checkDate, false); err != nil {
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

	mock.ExpectQuery(`SELECT count\(\*\) FROM "goal_checks" WHERE user_id = \$1 AND \(goal_id = \$2 AND date = \$3\)`).
		WithArgs(testUserID, goalID, checkDate).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	exists, err := repo.Exists(scopedUserContext(), goalID, checkDate)
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

	mock.ExpectQuery(`SELECT count\(\*\) FROM "goal_checks" WHERE user_id = \$1 AND \(goal_id = \$2 AND date = \$3\)`).
		WithArgs(testUserID, goalID, checkDate).
		WillReturnError(expectedErr)

	exists, err := repo.Exists(scopedUserContext(), goalID, checkDate)
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

	mock.ExpectQuery(`SELECT \* FROM "goal_checks" WHERE user_id = \$1 AND \(date BETWEEN \$2 AND \$3\) ORDER BY date ASC, goal_id ASC`).
		WithArgs(testUserID, normalizedStartDate, normalizedEndDate).
		WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "goal_id", "date", "created_at"}).
			AddRow(1, testUserID, 10, normalizedStartDate, fixedNow()).
			AddRow(2, testUserID, 11, normalizedEndDate, fixedNow()))

	goalChecks, err := repo.ListByDateRange(scopedUserContext(), startDate, endDate)
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

	database, mock, closeDB := newMockDatabase(t)
	return NewGoalCheckRepository(database), mock, closeDB
}
