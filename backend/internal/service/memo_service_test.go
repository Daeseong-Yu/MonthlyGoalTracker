package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Daeseong-Yu/MonthlyGoalTracker/backend/internal/domain"
)

func TestSaveMemoTrimsMemoBeforeSaving(t *testing.T) {
	repo := &stubMemoRepository{}
	service := NewMemoService(repo)

	dailyMemo, err := service.SaveMemo(context.Background(), time.Date(2026, time.April, 10, 15, 4, 5, 0, time.UTC), "  Ship note  ")
	if err != nil {
		t.Fatalf("expected save to succeed, got %v", err)
	}
	if dailyMemo == nil {
		t.Fatal("expected memo")
	}
	if repo.upsertCalls != 1 {
		t.Fatalf("expected upsert to be called once, got %d", repo.upsertCalls)
	}
	if repo.lastUpsertMemo != "Ship note" {
		t.Fatalf("expected trimmed memo %q, got %q", "Ship note", repo.lastUpsertMemo)
	}
	if dailyMemo.Memo != "Ship note" {
		t.Fatalf("expected returned memo %q, got %q", "Ship note", dailyMemo.Memo)
	}
	assertDateEqual(t, repo.lastUpsertDate, time.Date(2026, time.April, 10, 0, 0, 0, 0, time.UTC))
}

func TestSaveMemoAllowsEmptyMemoAfterTrim(t *testing.T) {
	repo := &stubMemoRepository{}
	service := NewMemoService(repo)

	dailyMemo, err := service.SaveMemo(context.Background(), time.Date(2026, time.April, 10, 15, 4, 5, 0, time.UTC), "   \n\t  ")
	if err != nil {
		t.Fatalf("expected save to succeed, got %v", err)
	}
	if dailyMemo == nil {
		t.Fatal("expected memo")
	}
	if repo.lastUpsertMemo != "" {
		t.Fatalf("expected empty memo, got %q", repo.lastUpsertMemo)
	}
	if dailyMemo.Memo != "" {
		t.Fatalf("expected returned memo to be empty, got %q", dailyMemo.Memo)
	}
}

func TestSaveMemoPreservesFixedZoneLocalCalendarDay(t *testing.T) {
	repo := &stubMemoRepository{}
	service := NewMemoService(repo)
	kst := time.FixedZone("KST", 9*60*60)

	dailyMemo, err := service.SaveMemo(context.Background(), time.Date(2026, time.April, 1, 0, 30, 0, 0, kst), "memo")
	if err != nil {
		t.Fatalf("expected save to succeed, got %v", err)
	}
	if dailyMemo == nil {
		t.Fatal("expected memo")
	}
	assertDateEqual(t, repo.lastUpsertDate, time.Date(2026, time.April, 1, 0, 0, 0, 0, time.UTC))
	assertDateEqual(t, dailyMemo.Date, time.Date(2026, time.April, 1, 0, 0, 0, 0, time.UTC))
}

func TestSaveMemoPropagatesRepositoryError(t *testing.T) {
	expectedErr := errors.New("upsert failed")
	repo := &stubMemoRepository{
		upsertFunc: func(context.Context, time.Time, string) (*domain.DailyMemo, error) {
			return nil, expectedErr
		},
	}
	service := NewMemoService(repo)

	dailyMemo, err := service.SaveMemo(context.Background(), time.Date(2026, time.April, 10, 15, 4, 5, 0, time.UTC), "memo")
	if dailyMemo != nil {
		t.Fatal("expected nil memo")
	}
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected repository error, got %v", err)
	}
}

func TestGetMemoPreservesFixedZoneLocalCalendarDay(t *testing.T) {
	repo := &stubMemoRepository{}
	service := NewMemoService(repo)
	kst := time.FixedZone("KST", 9*60*60)

	dailyMemo, err := service.GetMemo(context.Background(), time.Date(2026, time.April, 1, 8, 0, 0, 0, kst))
	if err != nil {
		t.Fatalf("expected get to succeed, got %v", err)
	}
	if dailyMemo == nil {
		t.Fatal("expected memo")
	}
	assertDateEqual(t, repo.lastFindDate, time.Date(2026, time.April, 1, 0, 0, 0, 0, time.UTC))
	assertDateEqual(t, dailyMemo.Date, time.Date(2026, time.April, 1, 0, 0, 0, 0, time.UTC))
}

func TestGetMemoPropagatesRepositoryError(t *testing.T) {
	expectedErr := errors.New("not found")
	repo := &stubMemoRepository{
		findByDateFunc: func(context.Context, time.Time) (*domain.DailyMemo, error) {
			return nil, expectedErr
		},
	}
	service := NewMemoService(repo)

	dailyMemo, err := service.GetMemo(context.Background(), time.Date(2026, time.April, 10, 15, 4, 5, 0, time.UTC))
	if dailyMemo != nil {
		t.Fatal("expected nil memo")
	}
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected repository error, got %v", err)
	}
}

func TestListMemosForMonthRejectsInvalidMonth(t *testing.T) {
	repo := &stubMemoRepository{}
	service := NewMemoService(repo)

	dailyMemos, err := service.ListMemosForMonth(context.Background(), "2026/04")
	if dailyMemos != nil {
		t.Fatal("expected nil memos")
	}
	if !errors.Is(err, ErrInvalidMonth) {
		t.Fatalf("expected ErrInvalidMonth, got %v", err)
	}
	if repo.listCalls != 0 {
		t.Fatal("expected list not to be called")
	}
}

func TestListMemosForMonthUsesMonthRange(t *testing.T) {
	expectedMemos := []domain.DailyMemo{{ID: 1, Memo: "memo"}}
	repo := &stubMemoRepository{
		listByDateRangeFunc: func(context.Context, time.Time, time.Time) ([]domain.DailyMemo, error) {
			return expectedMemos, nil
		},
	}
	service := NewMemoService(repo)

	dailyMemos, err := service.ListMemosForMonth(context.Background(), "2026-02")
	if err != nil {
		t.Fatalf("expected list to succeed, got %v", err)
	}
	if len(dailyMemos) != 1 || dailyMemos[0].ID != expectedMemos[0].ID {
		t.Fatalf("expected memos %v, got %v", expectedMemos, dailyMemos)
	}
	assertDateEqual(t, repo.lastListStartDate, time.Date(2026, time.February, 1, 0, 0, 0, 0, time.UTC))
	assertDateEqual(t, repo.lastListEndDate, time.Date(2026, time.February, 28, 0, 0, 0, 0, time.UTC))
}

func TestListMemosForMonthPropagatesRepositoryError(t *testing.T) {
	expectedErr := errors.New("list failed")
	repo := &stubMemoRepository{
		listByDateRangeFunc: func(context.Context, time.Time, time.Time) ([]domain.DailyMemo, error) {
			return nil, expectedErr
		},
	}
	service := NewMemoService(repo)

	dailyMemos, err := service.ListMemosForMonth(context.Background(), "2026-04")
	if dailyMemos != nil {
		t.Fatal("expected nil memos")
	}
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected repository error, got %v", err)
	}
}

type stubMemoRepository struct {
	upsertFunc          func(ctx context.Context, date time.Time, memo string) (*domain.DailyMemo, error)
	findByDateFunc      func(ctx context.Context, date time.Time) (*domain.DailyMemo, error)
	listByDateRangeFunc func(ctx context.Context, startDate, endDate time.Time) ([]domain.DailyMemo, error)

	upsertCalls int
	findCalls   int
	listCalls   int

	lastUpsertDate    time.Time
	lastUpsertMemo    string
	lastFindDate      time.Time
	lastListStartDate time.Time
	lastListEndDate   time.Time
}

func (s *stubMemoRepository) Upsert(ctx context.Context, date time.Time, memo string) (*domain.DailyMemo, error) {
	s.upsertCalls++
	s.lastUpsertDate = date
	s.lastUpsertMemo = memo
	if s.upsertFunc != nil {
		return s.upsertFunc(ctx, date, memo)
	}
	return &domain.DailyMemo{ID: 1, Date: date, Memo: memo}, nil
}

func (s *stubMemoRepository) FindByDate(ctx context.Context, date time.Time) (*domain.DailyMemo, error) {
	s.findCalls++
	s.lastFindDate = date
	if s.findByDateFunc != nil {
		return s.findByDateFunc(ctx, date)
	}
	return &domain.DailyMemo{ID: 1, Date: date, Memo: "memo"}, nil
}

func (s *stubMemoRepository) ListByDateRange(ctx context.Context, startDate, endDate time.Time) ([]domain.DailyMemo, error) {
	s.listCalls++
	s.lastListStartDate = startDate
	s.lastListEndDate = endDate
	if s.listByDateRangeFunc != nil {
		return s.listByDateRangeFunc(ctx, startDate, endDate)
	}
	return nil, nil
}
