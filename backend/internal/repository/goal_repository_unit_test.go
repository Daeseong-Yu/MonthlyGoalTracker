package repository

import (
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/Daeseong-Yu/MonthlyGoalTracker/backend/internal/domain"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func TestGoalRepositoryUpdateTitleReturnsNotFound(t *testing.T) {
	repo, mock, closeDB := newMockGoalRepository(t)
	defer closeDB()

	const goalID uint = 42
	mock.ExpectExec(`UPDATE "goals" SET "title"=\$1,"updated_at"=\$2 WHERE user_id = \$3 AND id = \$4`).
		WithArgs("updated title", fixedNow(), testUserID, goalID).
		WillReturnResult(sqlmock.NewResult(0, 0))

	goal, err := repo.UpdateTitle(scopedUserContext(), goalID, "updated title")
	if goal != nil {
		t.Fatal("expected nil goal")
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("expected ErrRecordNotFound, got %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}

func TestGoalRepositorySetEndDateAllowsClearingEndDate(t *testing.T) {
	repo, mock, closeDB := newMockGoalRepository(t)
	defer closeDB()

	const goalID uint = 42
	startDate := date(2099, time.January, 1)

	mock.ExpectExec(`UPDATE "goals" SET "end_date"=\$1,"updated_at"=\$2 WHERE user_id = \$3 AND id = \$4`).
		WithArgs(nil, fixedNow(), testUserID, goalID).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(`SELECT \* FROM "goals" WHERE user_id = \$1 AND id = \$2 ORDER BY "goals"."id" LIMIT \$3`).
		WithArgs(testUserID, goalID, 1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "title", "start_date", "end_date", "created_at", "updated_at"}).
			AddRow(goalID, testUserID, "goal without end date", startDate, nil, fixedNow(), fixedNow()))

	goal, err := repo.SetEndDate(scopedUserContext(), goalID, nil)
	if err != nil {
		t.Fatalf("expected end date clear to succeed, got %v", err)
	}
	if goal.EndDate != nil {
		t.Fatalf("expected nil end date, got %v", goal.EndDate)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}

func TestGoalRepositoryCreatePropagatesConstraintError(t *testing.T) {
	repo, mock, closeDB := newMockGoalRepository(t)
	defer closeDB()

	expectedErr := errors.New("check constraint failed")
	goal := &domain.Goal{
		Title:     "invalid range",
		StartDate: date(2099, time.January, 2),
		EndDate:   ptrTime(date(2099, time.January, 1)),
	}

	mock.ExpectQuery(`INSERT INTO "goals" \("user_id","title","start_date","end_date","created_at","updated_at"\) VALUES \(\$1,\$2,\$3,\$4,\$5,\$6\) RETURNING "id"`).
		WithArgs(testUserID, goal.Title, goal.StartDate, goal.EndDate, fixedNow(), fixedNow()).
		WillReturnError(expectedErr)

	err := repo.Create(scopedUserContext(), goal)
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected constraint error, got %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}

func TestGoalRepositoryListOverlappingDateRangeUsesInclusiveOverlapPredicate(t *testing.T) {
	repo, mock, closeDB := newMockGoalRepository(t)
	defer closeDB()

	rangeStart := date(2099, time.January, 1)
	rangeEnd := date(2099, time.January, 31)

	mock.ExpectQuery(`SELECT \* FROM "goals" WHERE user_id = \$1 AND \(start_date <= \$2 AND \(end_date IS NULL OR end_date >= \$3\)\) ORDER BY start_date ASC, id ASC`).
		WithArgs(testUserID, rangeEnd, rangeStart).
		WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "title", "start_date", "end_date", "created_at", "updated_at"}).
			AddRow(1, testUserID, "starts on range end", rangeEnd, nil, fixedNow(), fixedNow()).
			AddRow(2, testUserID, "ends on range start", rangeStart.AddDate(0, 0, -10), rangeStart, fixedNow(), fixedNow()))

	goals, err := repo.ListOverlappingDateRange(scopedUserContext(), rangeStart, rangeEnd)
	if err != nil {
		t.Fatalf("expected overlapping range lookup to succeed, got %v", err)
	}
	if len(goals) != 2 {
		t.Fatalf("expected 2 goals, got %d", len(goals))
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}

func newMockGoalRepository(t *testing.T) (*GoalRepository, sqlmock.Sqlmock, func()) {
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

	return NewGoalRepository(database), mock, func() {
		_ = sqlDB.Close()
	}
}

func fixedNow() time.Time {
	return time.Date(2099, time.January, 1, 1, 2, 3, 0, time.UTC)
}

func ptrTime(value time.Time) *time.Time {
	return &value
}
