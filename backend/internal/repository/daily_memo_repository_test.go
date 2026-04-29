package repository

import (
	"context"
	"testing"
	"time"

	"github.com/Daeseong-Yu/MonthlyGoalTracker/backend/internal/domain"
	"gorm.io/gorm"
)

func TestDailyMemoRepositoryIntegration(t *testing.T) {
	database := openIntegrationDatabase(t)
	repo := NewDailyMemoRepository(database)
	ctx := context.Background()

	startDate := uniqueIntegrationDate()
	memoDate := startDate.AddDate(0, 0, 14)
	endDate := startDate.AddDate(0, 0, 30)
	testDates := []time.Time{startDate, memoDate, endDate}

	cleanupDailyMemosByDates(t, database, testDates)
	t.Cleanup(func() {
		cleanupDailyMemosByDates(t, database, testDates)
	})

	created, err := repo.Upsert(ctx, memoDate, "first memo")
	if err != nil {
		t.Fatalf("expected first memo upsert to succeed, got %v", err)
	}
	if created.Memo != "first memo" {
		t.Fatalf("expected first memo, got %q", created.Memo)
	}

	updated, err := repo.Upsert(ctx, memoDate, "updated memo")
	if err != nil {
		t.Fatalf("expected second memo upsert to succeed, got %v", err)
	}
	if updated.ID != created.ID {
		t.Fatalf("expected upsert to keep ID %d, got %d", created.ID, updated.ID)
	}
	if updated.Memo != "updated memo" {
		t.Fatalf("expected updated memo, got %q", updated.Memo)
	}

	found, err := repo.FindByDate(ctx, memoDate)
	if err != nil {
		t.Fatalf("expected memo lookup to succeed, got %v", err)
	}
	if found.Memo != "updated memo" {
		t.Fatalf("expected found memo to be updated, got %q", found.Memo)
	}

	if _, err := repo.Upsert(ctx, startDate, "start boundary memo"); err != nil {
		t.Fatalf("expected start boundary memo upsert to succeed, got %v", err)
	}
	if _, err := repo.Upsert(ctx, endDate, "end boundary memo"); err != nil {
		t.Fatalf("expected end boundary memo upsert to succeed, got %v", err)
	}

	dailyMemos, err := repo.ListByDateRange(ctx, startDate, endDate)
	if err != nil {
		t.Fatalf("expected memo list to succeed, got %v", err)
	}

	memosByDate := dailyMemosByDate(dailyMemos)
	assertMemoForDate(t, memosByDate, startDate, "start boundary memo")
	assertMemoForDate(t, memosByDate, memoDate, "updated memo")
	assertMemoForDate(t, memosByDate, endDate, "end boundary memo")
}

func cleanupDailyMemosByDates(t *testing.T, database *gorm.DB, dates []time.Time) {
	t.Helper()

	if err := database.
		Where("date IN ?", dates).
		Delete(&domain.DailyMemo{}).Error; err != nil {
		t.Fatalf("failed to clean daily memos: %v", err)
	}
}

func dailyMemosByDate(dailyMemos []domain.DailyMemo) map[string]string {
	memosByDate := make(map[string]string)
	for _, dailyMemo := range dailyMemos {
		memosByDate[dailyMemo.Date.Format(time.DateOnly)] = dailyMemo.Memo
	}

	return memosByDate
}

func assertMemoForDate(t *testing.T, memosByDate map[string]string, date time.Time, expectedMemo string) {
	t.Helper()

	key := date.Format(time.DateOnly)
	if memosByDate[key] != expectedMemo {
		t.Fatalf("expected memo for %s to be %q, got %q", key, expectedMemo, memosByDate[key])
	}
}

func uniqueIntegrationDate() time.Time {
	offset := int(time.Now().UnixNano() % 20000)
	return date(2100, time.January, 1).AddDate(0, 0, offset)
}
