package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Daeseong-Yu/MonthlyGoalTracker/backend/internal/domain"
)

func TestCreateGoalRejectsStartDateOutsideMonth(t *testing.T) {
	repo := &stubGoalRepository{}
	service := NewGoalService(repo)

	goal, err := service.CreateGoal(context.Background(), "2026-04", "Valid title", time.Date(2026, time.May, 1, 12, 0, 0, 0, time.UTC))
	if goal != nil {
		t.Fatal("expected nil goal")
	}
	if !errors.Is(err, ErrStartDateOutsideMonth) {
		t.Fatalf("expected ErrStartDateOutsideMonth, got %v", err)
	}
	if repo.createCalls != 0 {
		t.Fatal("expected create not to be called")
	}
}

func TestCreateGoalAcceptsFixedZoneStartDateWithinMonth(t *testing.T) {
	repo := &stubGoalRepository{}
	service := NewGoalService(repo)
	kst := time.FixedZone("KST", 9*60*60)

	goal, err := service.CreateGoal(context.Background(), "2026-04", "Valid title", time.Date(2026, time.April, 1, 0, 30, 0, 0, kst))
	if err != nil {
		t.Fatalf("expected create to succeed, got %v", err)
	}
	if goal == nil {
		t.Fatal("expected goal")
	}
	assertDateEqual(t, goal.StartDate, time.Date(2026, time.April, 1, 0, 0, 0, 0, time.UTC))
	assertDateEqual(t, repo.lastListStartDate, time.Date(2026, time.April, 1, 0, 0, 0, 0, time.UTC))
	assertDateEqual(t, repo.lastListEndDate, time.Date(2026, time.April, 30, 0, 0, 0, 0, time.UTC))
	if repo.createdGoal == nil {
		t.Fatal("expected created goal to be recorded")
	}
	assertDateEqual(t, repo.createdGoal.StartDate, time.Date(2026, time.April, 1, 0, 0, 0, 0, time.UTC))
}

func TestCreateGoalRejectsWhitespaceTitle(t *testing.T) {
	repo := &stubGoalRepository{}
	service := NewGoalService(repo)

	goal, err := service.CreateGoal(context.Background(), "2026-04", "   \n\t  ", time.Date(2026, time.April, 10, 12, 0, 0, 0, time.UTC))
	if goal != nil {
		t.Fatal("expected nil goal")
	}
	if !errors.Is(err, ErrEmptyTitle) {
		t.Fatalf("expected ErrEmptyTitle, got %v", err)
	}
	if repo.createCalls != 0 {
		t.Fatal("expected create not to be called")
	}
}

func TestCreateGoalRejectsInvalidMonth(t *testing.T) {
	repo := &stubGoalRepository{}
	service := NewGoalService(repo)

	goal, err := service.CreateGoal(context.Background(), "2026/04", "Valid title", time.Date(2026, time.April, 10, 12, 0, 0, 0, time.UTC))
	if goal != nil {
		t.Fatal("expected nil goal")
	}
	if !errors.Is(err, ErrInvalidMonth) {
		t.Fatalf("expected ErrInvalidMonth, got %v", err)
	}
	if repo.createCalls != 0 {
		t.Fatal("expected create not to be called")
	}
}

func TestCreateGoalRejectsWhenSelectedMonthActiveGoalLimitExceeded(t *testing.T) {
	repo := &stubGoalRepository{
		listOverlappingDateRangeFunc: func(context.Context, time.Time, time.Time) ([]domain.Goal, error) {
			goals := make([]domain.Goal, 0, 5)
			for i := 0; i < 5; i++ {
				goals = append(goals, domain.Goal{
					ID:        uint(i + 1),
					Title:     "existing goal",
					StartDate: time.Date(2026, time.April, 1, 0, 0, 0, 0, time.UTC),
				})
			}
			return goals, nil
		},
	}
	service := NewGoalService(repo)

	goal, err := service.CreateGoal(context.Background(), "2026-04", "New goal", time.Date(2026, time.April, 10, 8, 30, 0, 0, time.UTC))
	if goal != nil {
		t.Fatal("expected nil goal")
	}
	if !errors.Is(err, ErrActiveGoalLimitExceeded) {
		t.Fatalf("expected ErrActiveGoalLimitExceeded, got %v", err)
	}
	if repo.createCalls != 0 {
		t.Fatal("expected create not to be called")
	}
	assertDateEqual(t, repo.lastListStartDate, time.Date(2026, time.April, 10, 0, 0, 0, 0, time.UTC))
	assertDateEqual(t, repo.lastListEndDate, time.Date(2026, time.April, 30, 0, 0, 0, 0, time.UTC))
}

func TestCreateGoalAllowsFifthActiveGoalAtInclusiveLimitBoundary(t *testing.T) {
	repo := &stubGoalRepository{
		listOverlappingDateRangeFunc: func(context.Context, time.Time, time.Time) ([]domain.Goal, error) {
			goals := make([]domain.Goal, 0, 4)
			for i := 0; i < 4; i++ {
				goals = append(goals, domain.Goal{
					ID:        uint(i + 1),
					Title:     "existing goal",
					StartDate: time.Date(2026, time.April, 10, 0, 0, 0, 0, time.UTC),
				})
			}
			return goals, nil
		},
	}
	service := NewGoalService(repo)

	goal, err := service.CreateGoal(context.Background(), "2026-04", "New goal", time.Date(2026, time.April, 10, 8, 30, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("expected create to succeed at inclusive limit boundary, got %v", err)
	}
	if goal == nil {
		t.Fatal("expected goal")
	}
	if repo.createCalls != 1 {
		t.Fatalf("expected create to be called once, got %d", repo.createCalls)
	}
	assertDateEqual(t, goal.StartDate, time.Date(2026, time.April, 10, 0, 0, 0, 0, time.UTC))
	assertDateEqual(t, repo.lastListStartDate, time.Date(2026, time.April, 10, 0, 0, 0, 0, time.UTC))
	assertDateEqual(t, repo.lastListEndDate, time.Date(2026, time.April, 30, 0, 0, 0, 0, time.UTC))
}

func TestCreateGoalChecksActiveGoalLimitWithinSelectedMonthOnly(t *testing.T) {
	repo := &stubGoalRepository{
		listOverlappingDateRangeFunc: func(_ context.Context, startDate, endDate time.Time) ([]domain.Goal, error) {
			assertDateEqual(t, startDate, time.Date(2026, time.April, 28, 0, 0, 0, 0, time.UTC))
			assertDateEqual(t, endDate, time.Date(2026, time.April, 30, 0, 0, 0, 0, time.UTC))
			return []domain.Goal{
				{
					ID:        1,
					Title:     "future goal",
					StartDate: time.Date(2026, time.May, 1, 0, 0, 0, 0, time.UTC),
				},
			}, nil
		},
	}
	service := NewGoalService(repo)

	goal, err := service.CreateGoal(context.Background(), "2026-04", "New goal", time.Date(2026, time.April, 28, 9, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("expected create to succeed, got %v", err)
	}
	if goal == nil {
		t.Fatal("expected goal")
	}
	if repo.listCalls != 1 {
		t.Fatalf("expected one overlapping-range query, got %d", repo.listCalls)
	}
	if repo.createCalls != 1 {
		t.Fatalf("expected create to be called once, got %d", repo.createCalls)
	}
}

func TestCreateGoalCreatesTrimmedGoal(t *testing.T) {
	repo := &stubGoalRepository{}
	service := NewGoalService(repo)

	goal, err := service.CreateGoal(context.Background(), "2026-04", "  Ship feature  ", time.Date(2026, time.April, 10, 15, 4, 5, 0, time.UTC))
	if err != nil {
		t.Fatalf("expected create to succeed, got %v", err)
	}
	if goal == nil {
		t.Fatal("expected goal")
	}
	if goal.Title != "Ship feature" {
		t.Fatalf("expected trimmed title, got %q", goal.Title)
	}
	assertDateEqual(t, goal.StartDate, time.Date(2026, time.April, 10, 0, 0, 0, 0, time.UTC))
	if repo.createCalls != 1 {
		t.Fatalf("expected create to be called once, got %d", repo.createCalls)
	}
	if repo.createdGoal == nil {
		t.Fatal("expected created goal to be recorded")
	}
	if repo.createdGoal.Title != "Ship feature" {
		t.Fatalf("expected repository title %q, got %q", "Ship feature", repo.createdGoal.Title)
	}
	assertDateEqual(t, repo.createdGoal.StartDate, time.Date(2026, time.April, 10, 0, 0, 0, 0, time.UTC))
}

func TestUpdateGoalTitleRejectsEmptyTitle(t *testing.T) {
	repo := &stubGoalRepository{}
	service := NewGoalService(repo)

	goal, err := service.UpdateGoalTitle(context.Background(), 42, "   ")
	if goal != nil {
		t.Fatal("expected nil goal")
	}
	if !errors.Is(err, ErrEmptyTitle) {
		t.Fatalf("expected ErrEmptyTitle, got %v", err)
	}
	if repo.updateTitleCalls != 0 {
		t.Fatal("expected update title not to be called")
	}
}

func TestUpdateGoalTitleTrimsTitle(t *testing.T) {
	repo := &stubGoalRepository{
		updateTitleFunc: func(_ context.Context, id uint, title string) (*domain.Goal, error) {
			return &domain.Goal{ID: id, Title: title}, nil
		},
	}
	service := NewGoalService(repo)

	goal, err := service.UpdateGoalTitle(context.Background(), 42, "  Renamed goal  ")
	if err != nil {
		t.Fatalf("expected update to succeed, got %v", err)
	}
	if goal == nil {
		t.Fatal("expected goal")
	}
	if goal.Title != "Renamed goal" {
		t.Fatalf("expected trimmed title, got %q", goal.Title)
	}
	if repo.lastUpdatedGoalID != 42 {
		t.Fatalf("expected goal id 42, got %d", repo.lastUpdatedGoalID)
	}
	if repo.lastUpdatedTitle != "Renamed goal" {
		t.Fatalf("expected repository title %q, got %q", "Renamed goal", repo.lastUpdatedTitle)
	}
}

func TestDeactivateGoalRejectsEndDateBeforeStartDate(t *testing.T) {
	repo := &stubGoalRepository{
		findByIDFunc: func(context.Context, uint) (*domain.Goal, error) {
			return &domain.Goal{ID: 42, Title: "goal", StartDate: time.Date(2026, time.April, 10, 12, 0, 0, 0, time.UTC)}, nil
		},
	}
	service := NewGoalService(repo)

	goal, err := service.DeactivateGoal(context.Background(), 42, time.Date(2026, time.April, 9, 23, 59, 0, 0, time.UTC))
	if goal != nil {
		t.Fatal("expected nil goal")
	}
	if !errors.Is(err, ErrInvalidEndDate) {
		t.Fatalf("expected ErrInvalidEndDate, got %v", err)
	}
	if repo.setEndDateCalls != 0 {
		t.Fatal("expected set end date not to be called")
	}
}

func TestDeactivateGoalSetsInclusiveNormalizedEndDate(t *testing.T) {
	repo := &stubGoalRepository{
		findByIDFunc: func(context.Context, uint) (*domain.Goal, error) {
			return &domain.Goal{ID: 42, Title: "goal", StartDate: time.Date(2026, time.April, 10, 12, 0, 0, 0, time.UTC)}, nil
		},
		setEndDateFunc: func(_ context.Context, id uint, endDate *time.Time) (*domain.Goal, error) {
			return &domain.Goal{ID: id, Title: "goal", StartDate: time.Date(2026, time.April, 10, 12, 0, 0, 0, time.UTC), EndDate: endDate}, nil
		},
	}
	service := NewGoalService(repo)

	goal, err := service.DeactivateGoal(context.Background(), 42, time.Date(2026, time.April, 10, 23, 59, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("expected deactivate to succeed, got %v", err)
	}
	if goal == nil || goal.EndDate == nil {
		t.Fatal("expected goal with end date")
	}
	assertDateEqual(t, *goal.EndDate, time.Date(2026, time.April, 10, 0, 0, 0, 0, time.UTC))
	if repo.lastFindGoalID != 42 {
		t.Fatalf("expected find by id to use 42, got %d", repo.lastFindGoalID)
	}
	if repo.lastSetEndDateGoalID != 42 {
		t.Fatalf("expected set end date to use 42, got %d", repo.lastSetEndDateGoalID)
	}
	if repo.lastSetEndDate == nil {
		t.Fatal("expected end date to be recorded")
	}
	assertDateEqual(t, *repo.lastSetEndDate, time.Date(2026, time.April, 10, 0, 0, 0, 0, time.UTC))
}

func TestDeactivateGoalPreservesFixedZoneLocalCalendarDay(t *testing.T) {
	kst := time.FixedZone("KST", 9*60*60)
	repo := &stubGoalRepository{
		findByIDFunc: func(context.Context, uint) (*domain.Goal, error) {
			return &domain.Goal{
				ID:        42,
				Title:     "goal",
				StartDate: time.Date(2026, time.April, 1, 0, 30, 0, 0, kst),
			}, nil
		},
		setEndDateFunc: func(_ context.Context, id uint, endDate *time.Time) (*domain.Goal, error) {
			return &domain.Goal{ID: id, Title: "goal", EndDate: endDate}, nil
		},
	}
	service := NewGoalService(repo)

	goal, err := service.DeactivateGoal(context.Background(), 42, time.Date(2026, time.April, 1, 8, 0, 0, 0, kst))
	if err != nil {
		t.Fatalf("expected deactivate to succeed, got %v", err)
	}
	if goal == nil || goal.EndDate == nil {
		t.Fatal("expected goal with end date")
	}
	assertDateEqual(t, *goal.EndDate, time.Date(2026, time.April, 1, 0, 0, 0, 0, time.UTC))
	if repo.lastSetEndDate == nil {
		t.Fatal("expected end date to be recorded")
	}
	assertDateEqual(t, *repo.lastSetEndDate, time.Date(2026, time.April, 1, 0, 0, 0, 0, time.UTC))
}

func TestListGoalsForMonthUsesMonthRange(t *testing.T) {
	expectedGoals := []domain.Goal{{ID: 1, Title: "goal"}}
	repo := &stubGoalRepository{
		listOverlappingDateRangeFunc: func(context.Context, time.Time, time.Time) ([]domain.Goal, error) {
			return expectedGoals, nil
		},
	}
	service := NewGoalService(repo)

	goals, err := service.ListGoalsForMonth(context.Background(), "2026-02")
	if err != nil {
		t.Fatalf("expected list to succeed, got %v", err)
	}
	if len(goals) != 1 || goals[0].ID != expectedGoals[0].ID {
		t.Fatalf("expected goals %v, got %v", expectedGoals, goals)
	}
	assertDateEqual(t, repo.lastListStartDate, time.Date(2026, time.February, 1, 0, 0, 0, 0, time.UTC))
	assertDateEqual(t, repo.lastListEndDate, time.Date(2026, time.February, 28, 0, 0, 0, 0, time.UTC))
}

type stubGoalRepository struct {
	createFunc                   func(ctx context.Context, goal *domain.Goal) error
	findByIDFunc                 func(ctx context.Context, id uint) (*domain.Goal, error)
	updateTitleFunc              func(ctx context.Context, id uint, title string) (*domain.Goal, error)
	setEndDateFunc               func(ctx context.Context, id uint, endDate *time.Time) (*domain.Goal, error)
	listOverlappingDateRangeFunc func(ctx context.Context, startDate, endDate time.Time) ([]domain.Goal, error)

	createCalls      int
	updateTitleCalls int
	setEndDateCalls  int
	listCalls        int

	createdGoal          *domain.Goal
	lastFindGoalID       uint
	lastUpdatedGoalID    uint
	lastUpdatedTitle     string
	lastSetEndDateGoalID uint
	lastSetEndDate       *time.Time
	lastListStartDate    time.Time
	lastListEndDate      time.Time
}

func (s *stubGoalRepository) Create(ctx context.Context, goal *domain.Goal) error {
	s.createCalls++
	copied := *goal
	s.createdGoal = &copied
	if s.createFunc != nil {
		return s.createFunc(ctx, goal)
	}
	return nil
}

func (s *stubGoalRepository) FindByID(ctx context.Context, id uint) (*domain.Goal, error) {
	s.lastFindGoalID = id
	if s.findByIDFunc != nil {
		return s.findByIDFunc(ctx, id)
	}
	return nil, nil
}

func (s *stubGoalRepository) UpdateTitle(ctx context.Context, id uint, title string) (*domain.Goal, error) {
	s.updateTitleCalls++
	s.lastUpdatedGoalID = id
	s.lastUpdatedTitle = title
	if s.updateTitleFunc != nil {
		return s.updateTitleFunc(ctx, id, title)
	}
	return &domain.Goal{ID: id, Title: title}, nil
}

func (s *stubGoalRepository) SetEndDate(ctx context.Context, id uint, endDate *time.Time) (*domain.Goal, error) {
	s.setEndDateCalls++
	s.lastSetEndDateGoalID = id
	if endDate != nil {
		copied := *endDate
		s.lastSetEndDate = &copied
	}
	if s.setEndDateFunc != nil {
		return s.setEndDateFunc(ctx, id, endDate)
	}
	return &domain.Goal{ID: id, EndDate: endDate}, nil
}

func (s *stubGoalRepository) ListOverlappingDateRange(ctx context.Context, startDate, endDate time.Time) ([]domain.Goal, error) {
	s.listCalls++
	s.lastListStartDate = startDate
	s.lastListEndDate = endDate
	if s.listOverlappingDateRangeFunc != nil {
		return s.listOverlappingDateRangeFunc(ctx, startDate, endDate)
	}
	return nil, nil
}

func assertDateEqual(t *testing.T, actual time.Time, expected time.Time) {
	t.Helper()
	if !actual.Equal(expected) {
		t.Fatalf("expected date %s, got %s", expected.Format(time.RFC3339), actual.Format(time.RFC3339))
	}
}
