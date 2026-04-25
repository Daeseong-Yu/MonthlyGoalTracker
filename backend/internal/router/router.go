package router

import (
	"github.com/Daeseong-Yu/MonthlyGoalTracker/backend/internal/handler"
	"github.com/gin-gonic/gin"
)

func SetupRouter() *gin.Engine {
	r := gin.Default()

	r.GET("/api/health", handler.Health)

	return r
}
