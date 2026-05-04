package router

import (
	"github.com/Daeseong-Yu/MonthlyGoalTracker/backend/internal/handler"
	"github.com/Daeseong-Yu/MonthlyGoalTracker/backend/internal/repository"
	"github.com/Daeseong-Yu/MonthlyGoalTracker/backend/internal/service"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func SetupRouter(database *gorm.DB) *gin.Engine {
	r := gin.Default()

	goalRepo := repository.NewGoalRepository(database)
	memoRepo := repository.NewDailyMemoRepository(database)
	checkRepo := repository.NewGoalCheckRepository(database)

	goalService := service.NewGoalService(goalRepo)
	memoService := service.NewMemoService(memoRepo)
	checkService := service.NewCheckService(goalRepo, checkRepo)
	monthService := service.NewMonthService(goalRepo, memoRepo, checkRepo)

	goalHandler := handler.NewGoalHandler(goalService)
	memoHandler := handler.NewMemoHandler(memoService)
	checkHandler := handler.NewCheckHandler(checkService)
	monthHandler := handler.NewMonthHandler(monthService)

	r.GET("/api/health", handler.Health)
	r.POST("/api/months/:month/ensure", monthHandler.Ensure)
	r.GET("/api/months/:month", monthHandler.Get)
	r.POST("/api/months/:month/goals", goalHandler.Create)
	r.PATCH("/api/goals/:id", goalHandler.Update)
	r.POST("/api/goals/:id/deactivate", goalHandler.Deactivate)
	r.PUT("/api/memos/:date", memoHandler.Save)
	r.PUT("/api/checks", checkHandler.Set)

	return r
}
