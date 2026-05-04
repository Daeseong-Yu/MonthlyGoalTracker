package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/Daeseong-Yu/MonthlyGoalTracker/backend/internal/domain"
	"github.com/gin-gonic/gin"
)

type MemoService interface {
	SaveMemo(ctx context.Context, date time.Time, memo string) (*domain.DailyMemo, error)
}

type MemoHandler struct {
	service MemoService
}

func NewMemoHandler(service MemoService) *MemoHandler {
	return &MemoHandler{service: service}
}

type saveMemoRequest struct {
	Memo *string `json:"memo"`
}

func (h *MemoHandler) Save(c *gin.Context) {
	date, err := parseDate(c.Param("date"))
	if err != nil {
		writeServiceError(c, err)
		return
	}

	var req saveMemoRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.Memo == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	memo, err := h.service.SaveMemo(c.Request.Context(), date, *req.Memo)
	if err != nil {
		writeServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, toMemoResponse(*memo))
}
