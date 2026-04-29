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

func TestDailyMemoRepositoryUpsertUsesDateConflictAndReturnsSavedMemo(t *testing.T) {
	repo, mock, closeDB := newMockDailyMemoRepository(t)
	defer closeDB()

	memoDate := time.Date(2099, time.February, 1, 15, 30, 0, 0, time.FixedZone("KST", 9*60*60))
	normalizedDate := date(2099, time.February, 1)

	mock.ExpectQuery(`INSERT INTO "daily_memos" \("date","memo","created_at","updated_at"\) VALUES \(\$1,\$2,\$3,\$4\) ON CONFLICT \("date"\) DO UPDATE SET "memo"="excluded"."memo","updated_at"="excluded"."updated_at" RETURNING "id"`).
		WithArgs(normalizedDate, "updated memo", fixedNow(), fixedNow()).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(11))
	mock.ExpectQuery(`SELECT \* FROM "daily_memos" WHERE date = \$1 ORDER BY "daily_memos"."id" LIMIT \$2`).
		WithArgs(normalizedDate, 1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "date", "memo", "created_at", "updated_at"}).
			AddRow(11, normalizedDate, "updated memo", fixedNow(), fixedNow()))

	dailyMemo, err := repo.Upsert(context.Background(), memoDate, "updated memo")
	if err != nil {
		t.Fatalf("expected upsert to succeed, got %v", err)
	}
	if dailyMemo.Memo != "updated memo" {
		t.Fatalf("expected updated memo, got %q", dailyMemo.Memo)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}

func TestDailyMemoRepositoryFindByDateReturnsNotFound(t *testing.T) {
	repo, mock, closeDB := newMockDailyMemoRepository(t)
	defer closeDB()

	memoDate := date(2099, time.February, 1)

	mock.ExpectQuery(`SELECT \* FROM "daily_memos" WHERE date = \$1 ORDER BY "daily_memos"."id" LIMIT \$2`).
		WithArgs(memoDate, 1).
		WillReturnError(gorm.ErrRecordNotFound)

	dailyMemo, err := repo.FindByDate(context.Background(), memoDate)
	if dailyMemo != nil {
		t.Fatal("expected nil memo")
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("expected ErrRecordNotFound, got %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}

func TestDailyMemoRepositoryListByDateRangeUsesInclusiveRange(t *testing.T) {
	repo, mock, closeDB := newMockDailyMemoRepository(t)
	defer closeDB()

	startDate := time.Date(2099, time.February, 1, 23, 59, 0, 0, time.FixedZone("KST", 9*60*60))
	endDate := time.Date(2099, time.February, 28, 6, 30, 0, 0, time.FixedZone("EST", -5*60*60))
	normalizedStartDate := date(2099, time.February, 1)
	normalizedEndDate := date(2099, time.February, 28)

	mock.ExpectQuery(`SELECT \* FROM "daily_memos" WHERE date BETWEEN \$1 AND \$2 ORDER BY date ASC`).
		WithArgs(normalizedStartDate, normalizedEndDate).
		WillReturnRows(sqlmock.NewRows([]string{"id", "date", "memo", "created_at", "updated_at"}).
			AddRow(1, normalizedStartDate, "first memo", fixedNow(), fixedNow()).
			AddRow(2, normalizedEndDate, "last memo", fixedNow(), fixedNow()))

	dailyMemos, err := repo.ListByDateRange(context.Background(), startDate, endDate)
	if err != nil {
		t.Fatalf("expected list by date range to succeed, got %v", err)
	}
	if len(dailyMemos) != 2 {
		t.Fatalf("expected 2 memos, got %d", len(dailyMemos))
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}

func newMockDailyMemoRepository(t *testing.T) (*DailyMemoRepository, sqlmock.Sqlmock, func()) {
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

	return NewDailyMemoRepository(database), mock, func() {
		_ = sqlDB.Close()
	}
}
