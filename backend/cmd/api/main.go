package main

import (
	"log"

	"github.com/Daeseong-Yu/MonthlyGoalTracker/backend/internal/config"
	"github.com/Daeseong-Yu/MonthlyGoalTracker/backend/internal/router"
)

func main() {
	cfg := config.Load()
	r := router.SetupRouter()

	if err := r.Run(cfg.Addr()); err != nil {
		log.Fatal(err)
	}
}
