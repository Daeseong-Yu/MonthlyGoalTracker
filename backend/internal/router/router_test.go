package router

import (
	"testing"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func TestSetupRouterRegistersAPIRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)

	engine := SetupRouter(&gorm.DB{})
	routes := routeSet(engine.Routes())

	expectedRoutes := []string{
		"GET /api/health",
		"POST /api/months/:month/ensure",
		"GET /api/months/:month",
		"POST /api/months/:month/goals",
		"PATCH /api/goals/:id",
		"POST /api/goals/:id/deactivate",
		"PUT /api/memos/:date",
		"PUT /api/checks",
	}

	for _, expectedRoute := range expectedRoutes {
		if !routes[expectedRoute] {
			t.Fatalf("expected route %s to be registered", expectedRoute)
		}
	}
}

func routeSet(routes []gin.RouteInfo) map[string]bool {
	values := make(map[string]bool, len(routes))
	for _, route := range routes {
		values[route.Method+" "+route.Path] = true
	}

	return values
}
