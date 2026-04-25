package main

import (
	"github.com/Daeseong-Yu/MonthlyGoalTracker/backend/internal/router"
)

func main() {
	r := router.SetupRouter()

	r.Run("127.0.0.1:8080")
}
