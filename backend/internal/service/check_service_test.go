package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Daeseong-Yu/MonthlyGoalTracker/backend/internal/domain"
)

func TestSetGoalCompletedSavesForActiveGoal(t *testing.T) {
	service, goalRepo, checkRepo := newCheckServiceFixture(t, activeAprilGoal())

	err := service.SetGoalCompleted(context.Background(), 42, aprilDateTime(12, 18), true)
	if err != nil {
		t.Fatalf("expected set completed to succeed, got %v", err)
	}
	if goalRepo.findCalls != 1 {
		t.Fatalf("expected goal lookup once, got %d", goalRepo.findCalls)
	}
	if checkRepo.setCalls != 1 {
		t.Fatalf("expected set completed once, got %d", checkRepo.setCalls)
	}
	if checkRepo.lastSetGoalID != 42 {
		t.Fatalf("expected goal id 42, got %d", checkRepo.lastSetGoalID)
	}
	if !checkRepo.lastSetCompleted {
		t.Fatal("expected completed=true")
	}
	assertDateEqual(t, checkRepo.lastSetDate, date(2026, time.April, 12))
}

func TestSetGoalCompletedRejectsBeforeStartDate(t *testing.T) {
	service, _, checkRepo := newCheckServiceFixture(t, activeAprilGoal())

	err := service.SetGoalCompleted(context.Background(), 42, aprilDateTime(9, 23), true)
	if !errors.Is(err, ErrGoalNotActiveOnDate) {
		t.Fatalf("expected ErrGoalNotActiveOnDate, got %v", err)
	}
	if checkRepo.setCalls != 0 {
		t.Fatalf("expected set completed not to be called, got %d", checkRepo.setCalls)
	}
}

func TestSetGoalCompletedRejectsAfterEndDate(t *testing.T) {
	service, _, checkRepo := newCheckServiceFixture(t, activeAprilGoalWithEnd(aprilDateTime(15, 22)))

	err := service.SetGoalCompleted(context.Background(), 42, aprilDateTime(16, 8), true)
	if !errors.Is(err, ErrGoalNotActiveOnDate) {
		t.Fatalf("expected ErrGoalNotActiveOnDate, got %v", err)
	}
	if checkRepo.setCalls != 0 {
		t.Fatalf("expected set completed not to be called, got %d", checkRepo.setCalls)
	}
}

func TestSetGoalCompletedAllowsEndDateDay(t *testing.T) {
	service, _, checkRepo := newCheckServiceFixture(t, activeAprilGoalWithEnd(aprilDateTime(15, 22)))

	err := service.SetGoalCompleted(context.Background(), 42, aprilDateTime(15, 8), true)
	if err != nil {
		t.Fatalf("expected end date day to be allowed, got %v", err)
	}
	if checkRepo.setCalls != 1 {
		t.Fatalf("expected set completed once, got %d", checkRepo.setCalls)
	}
	assertDateEqual(t, checkRepo.lastSetDate, date(2026, time.April, 15))
}

func TestSetGoalCompletedFalseUsesSetCompletedDeletePath(t *testing.T) {
	service, goalRepo, checkRepo := newCheckServiceFixture(t, activeAprilGoal())

	err := service.SetGoalCompleted(context.Background(), 42, aprilDateTime(12, 18), false)
	if err != nil {
		t.Fatalf("expected set completed false to succeed, got %v", err)
	}
	if goalRepo.findCalls != 1 {
		t.Fatalf("expected goal lookup once, got %d", goalRepo.findCalls)
	}
	if checkRepo.setCalls != 1 {
		t.Fatalf("expected set completed once, got %d", checkRepo.setCalls)
	}
	if checkRepo.lastSetCompleted {
		t.Fatal("expected completed=false")
	}
	assertDateEqual(t, checkRepo.lastSetDate, date(2026, time.April, 12))
}

func TestSetGoalCompletedNormalizesFixedZoneInputDate(t *testing.T) {
	kst := time.FixedZone("KST", 9*60*60)
	service, _, checkRepo := newCheckServiceFixture(t, &domain.Goal{
		ID:        42,
		Title:     "goal",
		StartDate: time.Date(2026, time.April, 1, 0, 30, 0, 0, kst),
	})

	err := service.SetGoalCompleted(context.Background(), 42, time.Date(2026, time.April, 1, 0, 30, 0, 0, kst), true)
	if err != nil {
		t.Fatalf("expected fixed-zone date to succeed, got %v", err)
	}
	assertDateEqual(t, checkRepo.lastSetDate, date(2026, time.April, 1))
}

func TestSetGoalCompletedPropagatesGoalLookupError(t *testing.T) {
	expectedErr := errors.New("lookup failed")
	goalRepo := &stubCheckGoalRepository{
		findByIDFunc: func(context.Context, uint) (*domain.Goal, error) {
			return nil, expectedErr
		},
	}
	checkRepo := &stubGoalCheckRepository{}
	service := NewCheckService(goalRepo, checkRepo)

	err := service.SetGoalCompleted(context.Background(), 42, aprilDateTime(12, 18), true)
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected goal lookup error, got %v", err)
	}
	if checkRepo.setCalls != 0 {
		t.Fatalf("expected set completed not to be called, got %d", checkRepo.setCalls)
	}
}

func TestSetGoalCompletedPropagatesCheckRepositoryError(t *testing.T) {
	expectedErr := errors.New("save failed")
	checkRepo := &stubGoalCheckRepository{
		setCompletedFunc: func(context.Context, uint, time.Time, bool) error {
			return expectedErr
		},
	}
	service := newCheckServiceWithRepos(activeAprilGoal(), checkRepo)

	err := service.SetGoalCompleted(context.Background(), 42, aprilDateTime(12, 18), true)
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected check repository error, got %v", err)
	}
}

func TestListChecksForMonthRejectsInvalidMonth(t *testing.T) {
	checkRepo := &stubGoalCheckRepository{}
	service := NewCheckService(&stubCheckGoalRepository{}, checkRepo)

	checks, err := service.ListChecksForMonth(context.Background(), "2026/04")
	if checks != nil {
		t.Fatal("expected nil checks")
	}
	if !errors.Is(err, ErrInvalidMonth) {
		t.Fatalf("expected ErrInvalidMonth, got %v", err)
	}
	if checkRepo.listCalls != 0 {
		t.Fatalf("expected list not to be called, got %d", checkRepo.listCalls)
	}
}

func TestListChecksForMonthPassesCorrectRange(t *testing.T) {
	expectedChecks := []domain.GoalCheck{{ID: 1, GoalID: 42}}
	checkRepo := &stubGoalCheckRepository{
		listByDateRangeFunc: func(context.Context, time.Time, time.Time) ([]domain.GoalCheck, error) {
			return expectedChecks, nil
		},
	}
	service := NewCheckService(&stubCheckGoalRepository{}, checkRepo)

	checks, err := service.ListChecksForMonth(context.Background(), "2026-02")
	if err != nil {
		t.Fatalf("expected list to succeed, got %v", err)
	}
	if len(checks) != 1 || checks[0].ID != expectedChecks[0].ID {
		t.Fatalf("expected checks %v, got %v", expectedChecks, checks)
	}
	assertDateEqual(t, checkRepo.lastListStartDate, date(2026, time.February, 1))
	assertDateEqual(t, checkRepo.lastListEndDate, date(2026, time.February, 28))
}

func TestListChecksForMonthPropagatesRepositoryError(t *testing.T) {
	expectedErr := errors.New("list failed")
	checkRepo := &stubGoalCheckRepository{
		listByDateRangeFunc: func(context.Context, time.Time, time.Time) ([]domain.GoalCheck, error) {
			return nil, expectedErr
		},
	}
	service := NewCheckService(&stubCheckGoalRepository{}, checkRepo)

	checks, err := service.ListChecksForMonth(context.Background(), "2026-04")
	if checks != nil {
		t.Fatal("expected nil checks")
	}
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected repository error, got %v", err)
	}
}

func aprilDateTime(day int, hour int) time.Time {
	return time.Date(2026, time.April, day, hour, 0, 0, 0, time.UTC)
}

func activeAprilGoal() *domain.Goal {
	return &domain.Goal{
		ID:        42,
		Title:     "goal",
		StartDate: aprilDateTime(10, 9),
	}
}

func activeAprilGoalWithEnd(endDate time.Time) *domain.Goal {
	goal := activeAprilGoal()
	goal.EndDate = &endDate
	return goal
}

func newCheckServiceFixture(t *testing.T, goal *domain.Goal) (*CheckService, *stubCheckGoalRepository, *stubGoalCheckRepository) {
	t.Helper()
	checkRepo := &stubGoalCheckRepository{}
	goalRepo := goalRepoReturning(goal)
	return NewCheckService(goalRepo, checkRepo), goalRepo, checkRepo
}

func newCheckServiceWithRepos(goal *domain.Goal, checkRepo *stubGoalCheckRepository) *CheckService {
	return NewCheckService(goalRepoReturning(goal), checkRepo)
}

func goalRepoReturning(goal *domain.Goal) *stubCheckGoalRepository {
	return &stubCheckGoalRepository{
		findByIDFunc: func(context.Context, uint) (*domain.Goal, error) {
			return goal, nil
		},
	}
}

type stubCheckGoalRepository struct {
	findByIDFunc func(ctx context.Context, id uint) (*domain.Goal, error)

	findCalls      int
	lastFindGoalID uint
}

func (s *stubCheckGoalRepository) FindByID(ctx context.Context, id uint) (*domain.Goal, error) {
	s.findCalls++
	s.lastFindGoalID = id
	if s.findByIDFunc != nil {
		return s.findByIDFunc(ctx, id)
	}
	return nil, nil
}

type stubGoalCheckRepository struct {
	setCompletedFunc    func(ctx context.Context, goalID uint, date time.Time, completed bool) error
	listByDateRangeFunc func(ctx context.Context, startDate, endDate time.Time) ([]domain.GoalCheck, error)

	setCalls  int
	listCalls int

	lastSetGoalID     uint
	lastSetDate       time.Time
	lastSetCompleted  bool
	lastListStartDate time.Time
	lastListEndDate   time.Time
}

func (s *stubGoalCheckRepository) SetCompleted(ctx context.Context, goalID uint, date time.Time, completed bool) error {
	s.setCalls++
	s.lastSetGoalID = goalID
	s.lastSetDate = date
	s.lastSetCompleted = completed
	if s.setCompletedFunc != nil {
		return s.setCompletedFunc(ctx, goalID, date, completed)
	}
	return nil
}

func (s *stubGoalCheckRepository) ListByDateRange(ctx context.Context, startDate, endDate time.Time) ([]domain.GoalCheck, error) {
	s.listCalls++
	s.lastListStartDate = startDate
	s.lastListEndDate = endDate
	if s.listByDateRangeFunc != nil {
		return s.listByDateRangeFunc(ctx, startDate, endDate)
	}
	return nil, nil
}
