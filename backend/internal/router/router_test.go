package router

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Daeseong-Yu/MonthlyGoalTracker/backend/internal/config"
	"github.com/Daeseong-Yu/MonthlyGoalTracker/backend/internal/principal"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

func TestSetupRouterRegistersAPIRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)

	engine := SetupRouter(&gorm.DB{}, config.BasicAuthConfig{})
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

func TestSetupRouterKeepsHealthPublicWhenBasicAuthEnabled(t *testing.T) {
	gin.SetMode(gin.TestMode)

	engine := SetupRouter(&gorm.DB{}, basicAuthConfigForTest(t))

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	engine.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}
}

func TestSetupRouterProtectsAPIRoutesWhenBasicAuthEnabled(t *testing.T) {
	gin.SetMode(gin.TestMode)

	engine := SetupRouter(&gorm.DB{}, basicAuthConfigForTest(t))

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/months/2026-05", nil)
	engine.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, recorder.Code)
	}

	if recorder.Header().Get("WWW-Authenticate") != basicAuthChallenge {
		t.Fatalf("expected basic auth challenge header, got %q", recorder.Header().Get("WWW-Authenticate"))
	}
}

func TestBasicAuthMiddlewareAllowsValidCredentials(t *testing.T) {
	gin.SetMode(gin.TestMode)

	engine := gin.New()
	engine.Use(principalMiddleware(principal.Default()))
	engine.Use(basicAuthMiddleware(basicAuthConfigForTest(t)))
	engine.GET("/protected", func(c *gin.Context) {
		current := principal.FromContext(c.Request.Context())
		if current.Username != "app-user" {
			t.Fatalf("expected username app-user, got %q", current.Username)
		}
		if !current.Authenticated {
			t.Fatal("expected principal to be authenticated")
		}

		c.Status(http.StatusNoContent)
	})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/protected", nil)
	request.SetBasicAuth("app-user", "secret")
	engine.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d", http.StatusNoContent, recorder.Code)
	}
}

func TestPrincipalMiddlewareAssignsDefaultPrincipal(t *testing.T) {
	gin.SetMode(gin.TestMode)

	engine := gin.New()
	engine.Use(principalMiddleware(principal.Default()))
	engine.GET("/protected", func(c *gin.Context) {
		if got := principal.FromContext(c.Request.Context()); got != principal.Default() {
			t.Fatalf("expected default principal %+v, got %+v", principal.Default(), got)
		}

		c.Status(http.StatusNoContent)
	})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/protected", nil)
	engine.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d", http.StatusNoContent, recorder.Code)
	}
}

func TestBasicAuthMiddlewareRejectsInvalidCredentials(t *testing.T) {
	gin.SetMode(gin.TestMode)

	engine := gin.New()
	engine.Use(principalMiddleware(principal.Default()))
	engine.Use(basicAuthMiddleware(basicAuthConfigForTest(t)))
	engine.GET("/protected", func(c *gin.Context) {
		t.Fatal("protected handler should not be called")
	})

	testCases := []struct {
		name     string
		username string
		password string
	}{
		{name: "wrong username", username: "wrong-user", password: "secret"},
		{name: "wrong password", username: "app-user", password: "wrong-secret"},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodGet, "/protected", nil)
			request.SetBasicAuth(testCase.username, testCase.password)
			engine.ServeHTTP(recorder, request)

			if recorder.Code != http.StatusUnauthorized {
				t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, recorder.Code)
			}
		})
	}
}

func routeSet(routes []gin.RouteInfo) map[string]bool {
	values := make(map[string]bool, len(routes))
	for _, route := range routes {
		values[route.Method+" "+route.Path] = true
	}

	return values
}

func basicAuthConfigForTest(t *testing.T) config.BasicAuthConfig {
	t.Helper()

	hash, err := bcrypt.GenerateFromPassword([]byte("secret"), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("failed to generate bcrypt hash: %v", err)
	}

	return config.BasicAuthConfig{
		Username:     "app-user",
		PasswordHash: string(hash),
	}
}
