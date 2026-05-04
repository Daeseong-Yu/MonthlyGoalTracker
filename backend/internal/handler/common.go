package handler

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/Daeseong-Yu/MonthlyGoalTracker/backend/internal/domain"
	"github.com/Daeseong-Yu/MonthlyGoalTracker/backend/internal/service"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

const dateLayout = "2006-01-02"

type goalResponse struct {
	ID        uint    `json:"id"`
	Title     string  `json:"title"`
	StartDate string  `json:"startDate"`
	EndDate   *string `json:"endDate"`
}

type memoResponse struct {
	Date string `json:"date"`
	Memo string `json:"memo"`
}

type goalCheckResponse struct {
	GoalID    uint   `json:"goalId"`
	Date      string `json:"date"`
	Completed bool   `json:"completed"`
}

type dayEntryResponse struct {
	Date            string  `json:"date"`
	Memo            string  `json:"memo"`
	ActiveGoalCount int     `json:"activeGoalCount"`
	CompletedCount  int     `json:"completedCount"`
	CompletionRate  float64 `json:"completionRate"`
}

type chartPointResponse struct {
	Date            string  `json:"date"`
	ActiveGoalCount int     `json:"activeGoalCount"`
	CompletedCount  int     `json:"completedCount"`
	CompletionRate  float64 `json:"completionRate"`
}

type monthViewResponse struct {
	Month  string               `json:"month"`
	Goals  []goalResponse       `json:"goals"`
	Days   []dayEntryResponse   `json:"days"`
	Checks []goalCheckResponse  `json:"checks"`
	Chart  []chartPointResponse `json:"chart"`
}

func parseDate(value string) (time.Time, error) {
	parsedDate, err := time.Parse(dateLayout, value)
	if err != nil || parsedDate.Format(dateLayout) != value {
		return time.Time{}, errInvalidDate
	}

	return parsedDate, nil
}

func parseIDParam(c *gin.Context, name string) (uint, bool) {
	rawID := c.Param(name)
	parsedID, err := strconv.ParseUint(rawID, 10, 64)
	if err != nil || parsedID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid " + name})
		return 0, false
	}

	return uint(parsedID), true
}

func writeServiceError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, errInvalidDate):
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid date"})
	case errors.Is(err, service.ErrInvalidMonth):
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid month"})
	case errors.Is(err, service.ErrEmptyTitle):
		c.JSON(http.StatusBadRequest, gin.H{"error": "empty title"})
	case errors.Is(err, service.ErrStartDateOutsideMonth):
		c.JSON(http.StatusBadRequest, gin.H{"error": "start date outside month"})
	case errors.Is(err, service.ErrInvalidEndDate):
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid end date"})
	case errors.Is(err, service.ErrGoalNotActiveOnDate):
		c.JSON(http.StatusBadRequest, gin.H{"error": "goal not active on date"})
	case errors.Is(err, service.ErrActiveGoalLimitExceeded):
		c.JSON(http.StatusConflict, gin.H{"error": "active goal limit exceeded"})
	case errors.Is(err, gorm.ErrRecordNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
	}
}

func toGoalResponse(goal domain.Goal) goalResponse {
	var endDate *string
	if goal.EndDate != nil {
		formattedEndDate := goal.EndDate.Format(dateLayout)
		endDate = &formattedEndDate
	}

	return goalResponse{
		ID:        goal.ID,
		Title:     goal.Title,
		StartDate: goal.StartDate.Format(dateLayout),
		EndDate:   endDate,
	}
}

func toGoalResponses(goals []domain.Goal) []goalResponse {
	responses := make([]goalResponse, 0, len(goals))
	for _, goal := range goals {
		responses = append(responses, toGoalResponse(goal))
	}

	return responses
}

func toMemoResponse(memo domain.DailyMemo) memoResponse {
	return memoResponse{
		Date: memo.Date.Format(dateLayout),
		Memo: memo.Memo,
	}
}

func toGoalCheckResponse(check domain.GoalCheck) goalCheckResponse {
	return goalCheckResponse{
		GoalID:    check.GoalID,
		Date:      check.Date.Format(dateLayout),
		Completed: true,
	}
}

func toMonthViewResponse(view service.MonthView) monthViewResponse {
	days := make([]dayEntryResponse, 0, len(view.Days))
	for _, day := range view.Days {
		days = append(days, dayEntryResponse{
			Date:            day.Date.Format(dateLayout),
			Memo:            day.Memo,
			ActiveGoalCount: day.ActiveGoalCount,
			CompletedCount:  day.CompletedCount,
			CompletionRate:  day.CompletionRate,
		})
	}

	checks := make([]goalCheckResponse, 0, len(view.Checks))
	for _, check := range view.Checks {
		checks = append(checks, toGoalCheckResponse(check))
	}

	chart := make([]chartPointResponse, 0, len(view.Chart))
	for _, point := range view.Chart {
		chart = append(chart, chartPointResponse{
			Date:            point.Date.Format(dateLayout),
			ActiveGoalCount: point.ActiveGoalCount,
			CompletedCount:  point.CompletedCount,
			CompletionRate:  point.CompletionRate,
		})
	}

	return monthViewResponse{
		Month:  view.Month,
		Goals:  toGoalResponses(view.Goals),
		Days:   days,
		Checks: checks,
		Chart:  chart,
	}
}

var errInvalidDate = errors.New("invalid date")
