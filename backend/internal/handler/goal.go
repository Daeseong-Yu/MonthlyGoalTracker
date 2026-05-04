package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/Daeseong-Yu/MonthlyGoalTracker/backend/internal/domain"
	"github.com/gin-gonic/gin"
)

type GoalService interface {
	CreateGoal(ctx context.Context, month string, title string, startDate time.Time) (*domain.Goal, error)
	UpdateGoalTitle(ctx context.Context, goalID uint, title string) (*domain.Goal, error)
	DeactivateGoal(ctx context.Context, goalID uint, endDate time.Time) (*domain.Goal, error)
}

type GoalHandler struct {
	service GoalService
}

func NewGoalHandler(service GoalService) *GoalHandler {
	return &GoalHandler{service: service}
}

type createGoalRequest struct {
	Title     string `json:"title"`
	StartDate string `json:"startDate"`
}

type updateGoalRequest struct {
	Title string `json:"title"`
}

type deactivateGoalRequest struct {
	EndDate string `json:"endDate"`
}

func (h *GoalHandler) Create(c *gin.Context) {
	var req createGoalRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	startDate, err := parseDate(req.StartDate)
	if err != nil {
		writeServiceError(c, err)
		return
	}

	goal, err := h.service.CreateGoal(c.Request.Context(), c.Param("month"), req.Title, startDate)
	if err != nil {
		writeServiceError(c, err)
		return
	}

	c.JSON(http.StatusCreated, toGoalResponse(*goal))
}

func (h *GoalHandler) Update(c *gin.Context) {
	goalID, ok := parseIDParam(c, "id")
	if !ok {
		return
	}

	var req updateGoalRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	goal, err := h.service.UpdateGoalTitle(c.Request.Context(), goalID, req.Title)
	if err != nil {
		writeServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, toGoalResponse(*goal))
}

func (h *GoalHandler) Deactivate(c *gin.Context) {
	goalID, ok := parseIDParam(c, "id")
	if !ok {
		return
	}

	var req deactivateGoalRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	endDate, err := parseDate(req.EndDate)
	if err != nil {
		writeServiceError(c, err)
		return
	}

	goal, err := h.service.DeactivateGoal(c.Request.Context(), goalID, endDate)
	if err != nil {
		writeServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, toGoalResponse(*goal))
}
