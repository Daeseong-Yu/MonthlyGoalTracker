package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Daeseong-Yu/MonthlyGoalTracker/backend/internal/domain"
	"github.com/Daeseong-Yu/MonthlyGoalTracker/backend/internal/principal"
	"github.com/Daeseong-Yu/MonthlyGoalTracker/backend/internal/service"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func TestGoalHandlerCreateReturnsCreatedGoal(t *testing.T) {
	gin.SetMode(gin.TestMode)

	expectedDate := date(2026, time.April, 10)
	goalService := &stubGoalService{
		createGoalFunc: func(_ context.Context, month string, title string, startDate time.Time) (*domain.Goal, error) {
			if month != "2026-04" {
				t.Fatalf("expected month 2026-04, got %q", month)
			}
			if title != "Exercise" {
				t.Fatalf("expected title Exercise, got %q", title)
			}
			if !startDate.Equal(expectedDate) {
				t.Fatalf("expected start date %s, got %s", expectedDate, startDate)
			}

			return &domain.Goal{ID: 7, Title: title, StartDate: startDate}, nil
		},
	}
	goalHandler := NewGoalHandler(goalService)

	recorder := performRequest(http.MethodPost, "/api/months/2026-04/goals", `{"title":"Exercise","startDate":"2026-04-10"}`, func(router *gin.Engine) {
		router.POST("/api/months/:month/goals", goalHandler.Create)
	})

	if recorder.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d: %s", http.StatusCreated, recorder.Code, recorder.Body.String())
	}
	if goalService.createGoalCalls != 1 {
		t.Fatalf("expected create goal to be called once, got %d", goalService.createGoalCalls)
	}

	var response goalResponse
	decodeJSON(t, recorder.Body.Bytes(), &response)
	if response.ID != 7 || response.Title != "Exercise" || response.StartDate != "2026-04-10" {
		t.Fatalf("unexpected response: %+v", response)
	}
	if response.EndDate != nil {
		t.Fatalf("expected nil end date, got %q", *response.EndDate)
	}
}

func TestGoalHandlerCreateRejectsInvalidStartDate(t *testing.T) {
	gin.SetMode(gin.TestMode)

	goalService := &stubGoalService{}
	goalHandler := NewGoalHandler(goalService)

	recorder := performRequest(http.MethodPost, "/api/months/2026-04/goals", `{"title":"Exercise","startDate":"2026/04/10"}`, func(router *gin.Engine) {
		router.POST("/api/months/:month/goals", goalHandler.Create)
	})

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, recorder.Code)
	}
	if goalService.createGoalCalls != 0 {
		t.Fatalf("expected create goal not to be called, got %d", goalService.createGoalCalls)
	}
}

func TestGoalHandlerCreateForwardsRequestPrincipal(t *testing.T) {
	gin.SetMode(gin.TestMode)

	expectedPrincipal := principal.NewAuthenticated("app-user")
	goalService := &stubGoalService{
		createGoalFunc: func(ctx context.Context, month string, title string, startDate time.Time) (*domain.Goal, error) {
			if got := principal.FromContext(ctx); got != expectedPrincipal {
				t.Fatalf("expected principal %+v, got %+v", expectedPrincipal, got)
			}

			return &domain.Goal{ID: 8, Title: title, StartDate: startDate}, nil
		},
	}
	goalHandler := NewGoalHandler(goalService)

	router := gin.New()
	router.POST("/api/months/:month/goals", goalHandler.Create)

	request := httptest.NewRequest(http.MethodPost, "/api/months/2026-04/goals", bytes.NewReader([]byte(`{"title":"Exercise","startDate":"2026-04-10"}`)))
	request.Header.Set("Content-Type", "application/json")
	request = request.WithContext(principal.WithContext(request.Context(), expectedPrincipal))

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d: %s", http.StatusCreated, recorder.Code, recorder.Body.String())
	}
	if goalService.createGoalCalls != 1 {
		t.Fatalf("expected create goal to be called once, got %d", goalService.createGoalCalls)
	}

	var response goalResponse
	decodeJSON(t, recorder.Body.Bytes(), &response)
	if response.ID != 8 || response.Title != "Exercise" || response.StartDate != "2026-04-10" {
		t.Fatalf("unexpected response: %+v", response)
	}
}

func TestGoalHandlerUpdateMapsNotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)

	goalService := &stubGoalService{
		updateGoalTitleFunc: func(context.Context, uint, string) (*domain.Goal, error) {
			return nil, gorm.ErrRecordNotFound
		},
	}
	goalHandler := NewGoalHandler(goalService)

	recorder := performRequest(http.MethodPatch, "/api/goals/42", `{"title":"Updated"}`, func(router *gin.Engine) {
		router.PATCH("/api/goals/:id", goalHandler.Update)
	})

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, recorder.Code)
	}
}

func TestMemoHandlerSaveReturnsSavedMemo(t *testing.T) {
	gin.SetMode(gin.TestMode)

	expectedDate := date(2026, time.April, 12)
	memoService := &stubMemoService{
		saveMemoFunc: func(_ context.Context, actualDate time.Time, memo string) (*domain.DailyMemo, error) {
			if !actualDate.Equal(expectedDate) {
				t.Fatalf("expected date %s, got %s", expectedDate, actualDate)
			}
			if memo != "Evening walk" {
				t.Fatalf("expected memo Evening walk, got %q", memo)
			}

			return &domain.DailyMemo{Date: actualDate, Memo: memo}, nil
		},
	}
	memoHandler := NewMemoHandler(memoService)

	recorder := performRequest(http.MethodPut, "/api/memos/2026-04-12", `{"memo":"Evening walk"}`, func(router *gin.Engine) {
		router.PUT("/api/memos/:date", memoHandler.Save)
	})

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, recorder.Code, recorder.Body.String())
	}

	var response memoResponse
	decodeJSON(t, recorder.Body.Bytes(), &response)
	if response.Date != "2026-04-12" || response.Memo != "Evening walk" {
		t.Fatalf("unexpected response: %+v", response)
	}
}

func TestMemoHandlerSaveRejectsMissingMemo(t *testing.T) {
	gin.SetMode(gin.TestMode)

	memoService := &stubMemoService{}
	memoHandler := NewMemoHandler(memoService)

	recorder := performRequest(http.MethodPut, "/api/memos/2026-04-12", `{}`, func(router *gin.Engine) {
		router.PUT("/api/memos/:date", memoHandler.Save)
	})

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, recorder.Code)
	}
	if memoService.saveMemoCalls != 0 {
		t.Fatalf("expected save memo not to be called, got %d", memoService.saveMemoCalls)
	}
}

func TestCheckHandlerSetAcceptsCompletedFalse(t *testing.T) {
	gin.SetMode(gin.TestMode)

	expectedDate := date(2026, time.April, 12)
	checkService := &stubCheckService{
		setGoalCompletedFunc: func(_ context.Context, goalID uint, actualDate time.Time, completed bool) error {
			if goalID != 7 {
				t.Fatalf("expected goal ID 7, got %d", goalID)
			}
			if !actualDate.Equal(expectedDate) {
				t.Fatalf("expected date %s, got %s", expectedDate, actualDate)
			}
			if completed {
				t.Fatal("expected completed false")
			}

			return nil
		},
	}
	checkHandler := NewCheckHandler(checkService)

	recorder := performRequest(http.MethodPut, "/api/checks", `{"goalId":7,"date":"2026-04-12","completed":false}`, func(router *gin.Engine) {
		router.PUT("/api/checks", checkHandler.Set)
	})

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, recorder.Code, recorder.Body.String())
	}

	var response goalCheckResponse
	decodeJSON(t, recorder.Body.Bytes(), &response)
	if response.GoalID != 7 || response.Date != "2026-04-12" || response.Completed {
		t.Fatalf("unexpected response: %+v", response)
	}
}

func TestCheckHandlerSetMapsInactiveGoal(t *testing.T) {
	gin.SetMode(gin.TestMode)

	checkService := &stubCheckService{
		setGoalCompletedFunc: func(context.Context, uint, time.Time, bool) error {
			return service.ErrGoalNotActiveOnDate
		},
	}
	checkHandler := NewCheckHandler(checkService)

	recorder := performRequest(http.MethodPut, "/api/checks", `{"goalId":7,"date":"2026-04-12","completed":true}`, func(router *gin.Engine) {
		router.PUT("/api/checks", checkHandler.Set)
	})

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, recorder.Code)
	}
}

func TestMonthHandlerGetReturnsMonthView(t *testing.T) {
	gin.SetMode(gin.TestMode)

	monthService := &stubMonthService{
		getMonthViewFunc: func(_ context.Context, month string) (*service.MonthView, error) {
			if month != "2026-04" {
				t.Fatalf("expected month 2026-04, got %q", month)
			}

			return &service.MonthView{
				Month: "2026-04",
				Goals: []domain.Goal{
					{ID: 7, Title: "Exercise", StartDate: date(2026, time.April, 1)},
				},
				Days: []service.DayEntry{
					{Date: date(2026, time.April, 1), Memo: "Start", ActiveGoalCount: 1, CompletedCount: 1, CompletionRate: 1},
				},
				Checks: []domain.GoalCheck{
					{GoalID: 7, Date: date(2026, time.April, 1)},
				},
				Chart: []service.ChartPoint{
					{Date: date(2026, time.April, 1), ActiveGoalCount: 1, CompletedCount: 1, CompletionRate: 1},
				},
			}, nil
		},
	}
	monthHandler := NewMonthHandler(monthService)

	recorder := performRequest(http.MethodGet, "/api/months/2026-04", "", func(router *gin.Engine) {
		router.GET("/api/months/:month", monthHandler.Get)
	})

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, recorder.Code, recorder.Body.String())
	}

	var response monthViewResponse
	decodeJSON(t, recorder.Body.Bytes(), &response)
	if response.Month != "2026-04" || len(response.Goals) != 1 || len(response.Days) != 1 || len(response.Checks) != 1 || len(response.Chart) != 1 {
		t.Fatalf("unexpected response: %+v", response)
	}
}

func TestMonthHandlerEnsureMapsInvalidMonth(t *testing.T) {
	gin.SetMode(gin.TestMode)

	monthService := &stubMonthService{
		ensureMonthFunc: func(context.Context, string) (*service.MonthView, error) {
			return nil, service.ErrInvalidMonth
		},
	}
	monthHandler := NewMonthHandler(monthService)

	recorder := performRequest(http.MethodPost, "/api/months/invalid/ensure", "", func(router *gin.Engine) {
		router.POST("/api/months/:month/ensure", monthHandler.Ensure)
	})

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, recorder.Code)
	}
}

type stubGoalService struct {
	createGoalFunc      func(ctx context.Context, month string, title string, startDate time.Time) (*domain.Goal, error)
	updateGoalTitleFunc func(ctx context.Context, goalID uint, title string) (*domain.Goal, error)
	deactivateGoalFunc  func(ctx context.Context, goalID uint, endDate time.Time) (*domain.Goal, error)

	createGoalCalls int
}

func (s *stubGoalService) CreateGoal(ctx context.Context, month string, title string, startDate time.Time) (*domain.Goal, error) {
	s.createGoalCalls++
	if s.createGoalFunc != nil {
		return s.createGoalFunc(ctx, month, title, startDate)
	}

	return nil, errors.New("unexpected CreateGoal call")
}

func (s *stubGoalService) UpdateGoalTitle(ctx context.Context, goalID uint, title string) (*domain.Goal, error) {
	if s.updateGoalTitleFunc != nil {
		return s.updateGoalTitleFunc(ctx, goalID, title)
	}

	return nil, errors.New("unexpected UpdateGoalTitle call")
}

func (s *stubGoalService) DeactivateGoal(ctx context.Context, goalID uint, endDate time.Time) (*domain.Goal, error) {
	if s.deactivateGoalFunc != nil {
		return s.deactivateGoalFunc(ctx, goalID, endDate)
	}

	return nil, errors.New("unexpected DeactivateGoal call")
}

type stubMemoService struct {
	saveMemoFunc func(ctx context.Context, date time.Time, memo string) (*domain.DailyMemo, error)

	saveMemoCalls int
}

func (s *stubMemoService) SaveMemo(ctx context.Context, date time.Time, memo string) (*domain.DailyMemo, error) {
	s.saveMemoCalls++
	if s.saveMemoFunc != nil {
		return s.saveMemoFunc(ctx, date, memo)
	}

	return nil, errors.New("unexpected SaveMemo call")
}

type stubCheckService struct {
	setGoalCompletedFunc func(ctx context.Context, goalID uint, date time.Time, completed bool) error
}

func (s *stubCheckService) SetGoalCompleted(ctx context.Context, goalID uint, date time.Time, completed bool) error {
	if s.setGoalCompletedFunc != nil {
		return s.setGoalCompletedFunc(ctx, goalID, date, completed)
	}

	return errors.New("unexpected SetGoalCompleted call")
}

type stubMonthService struct {
	ensureMonthFunc  func(ctx context.Context, month string) (*service.MonthView, error)
	getMonthViewFunc func(ctx context.Context, month string) (*service.MonthView, error)
}

func (s *stubMonthService) EnsureMonth(ctx context.Context, month string) (*service.MonthView, error) {
	if s.ensureMonthFunc != nil {
		return s.ensureMonthFunc(ctx, month)
	}

	return nil, errors.New("unexpected EnsureMonth call")
}

func (s *stubMonthService) GetMonthView(ctx context.Context, month string) (*service.MonthView, error) {
	if s.getMonthViewFunc != nil {
		return s.getMonthViewFunc(ctx, month)
	}

	return nil, errors.New("unexpected GetMonthView call")
}

func performRequest(method string, path string, body string, register func(*gin.Engine)) *httptest.ResponseRecorder {
	router := gin.New()
	register(router)

	var requestBody *bytes.Reader
	if body == "" {
		requestBody = bytes.NewReader(nil)
	} else {
		requestBody = bytes.NewReader([]byte(body))
	}

	request := httptest.NewRequest(method, path, requestBody)
	request.Header.Set("Content-Type", "application/json")

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	return recorder
}

func decodeJSON(t *testing.T, data []byte, target any) {
	t.Helper()

	if err := json.Unmarshal(data, target); err != nil {
		t.Fatalf("failed to decode JSON %q: %v", string(data), err)
	}
}

func date(year int, month time.Month, day int) time.Time {
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
}
