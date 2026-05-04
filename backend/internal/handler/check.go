package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type CheckService interface {
	SetGoalCompleted(ctx context.Context, goalID uint, date time.Time, completed bool) error
}

type CheckHandler struct {
	service CheckService
}

func NewCheckHandler(service CheckService) *CheckHandler {
	return &CheckHandler{service: service}
}

type setCheckRequest struct {
	GoalID    uint   `json:"goalId"`
	Date      string `json:"date"`
	Completed *bool  `json:"completed"`
}

func (h *CheckHandler) Set(c *gin.Context) {
	var req setCheckRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.GoalID == 0 || req.Completed == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	date, err := parseDate(req.Date)
	if err != nil {
		writeServiceError(c, err)
		return
	}

	if err := h.service.SetGoalCompleted(c.Request.Context(), req.GoalID, date, *req.Completed); err != nil {
		writeServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, goalCheckResponse{
		GoalID:    req.GoalID,
		Date:      date.Format(dateLayout),
		Completed: *req.Completed,
	})
}
