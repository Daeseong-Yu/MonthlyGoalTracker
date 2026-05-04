package handler

import (
	"context"
	"net/http"

	"github.com/Daeseong-Yu/MonthlyGoalTracker/backend/internal/service"
	"github.com/gin-gonic/gin"
)

type MonthService interface {
	EnsureMonth(ctx context.Context, month string) (*service.MonthView, error)
	GetMonthView(ctx context.Context, month string) (*service.MonthView, error)
}

type MonthHandler struct {
	service MonthService
}

func NewMonthHandler(service MonthService) *MonthHandler {
	return &MonthHandler{service: service}
}

func (h *MonthHandler) Ensure(c *gin.Context) {
	view, err := h.service.EnsureMonth(c.Request.Context(), c.Param("month"))
	if err != nil {
		writeServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, toMonthViewResponse(*view))
}

func (h *MonthHandler) Get(c *gin.Context) {
	view, err := h.service.GetMonthView(c.Request.Context(), c.Param("month"))
	if err != nil {
		writeServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, toMonthViewResponse(*view))
}
