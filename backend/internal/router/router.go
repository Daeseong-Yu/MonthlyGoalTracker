package router

import (
	"github.com/Daeseong-Yu/MonthlyGoalTracker/backend/internal/config"
	"github.com/Daeseong-Yu/MonthlyGoalTracker/backend/internal/handler"
	"github.com/Daeseong-Yu/MonthlyGoalTracker/backend/internal/principal"
	"github.com/Daeseong-Yu/MonthlyGoalTracker/backend/internal/repository"
	"github.com/Daeseong-Yu/MonthlyGoalTracker/backend/internal/service"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func SetupRouter(database *gorm.DB, auth config.BasicAuthConfig) *gin.Engine {
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

	api := r.Group("/api")
	api.Use(principalMiddleware(principal.Default()))
	if auth.Enabled() {
		api.Use(basicAuthMiddleware(auth))
	}

	api.POST("/months/:month/ensure", monthHandler.Ensure)
	api.GET("/months/:month", monthHandler.Get)
	api.POST("/months/:month/goals", goalHandler.Create)
	api.PATCH("/goals/:id", goalHandler.Update)
	api.POST("/goals/:id/deactivate", goalHandler.Deactivate)
	api.PUT("/memos/:date", memoHandler.Save)
	api.PUT("/checks", checkHandler.Set)

	return r
}
