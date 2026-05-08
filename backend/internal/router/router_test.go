package router

import (
	"net/http"
	"testing"

	"github.com/Daeseong-Yu/MonthlyGoalTracker/backend/internal/config"
	"github.com/Daeseong-Yu/MonthlyGoalTracker/backend/internal/principal"
	"github.com/Daeseong-Yu/MonthlyGoalTracker/backend/internal/repository"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func TestSetupRouterRegistersAPIRoutes(t *testing.T) {
	setGinTestMode(t)

	engine := SetupRouter(&gorm.DB{}, config.Config{})
	routes := routeSet(engine.Routes())

	expectedRoutes := []string{
		"GET /api/health",
		"GET /api/bootstrap",
		"POST /api/auth/signup",
		"POST /api/auth/login",
		"POST /api/auth/verify-email",
		"POST /api/auth/password-reset/request",
		"POST /api/auth/password-reset/confirm",
		"GET /api/auth/me",
		"POST /api/auth/logout",
		"PATCH /api/auth/me/locale",
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

func TestSetupRouterKeepsHealthPublic(t *testing.T) {
	setGinTestMode(t)

	engine := SetupRouter(&gorm.DB{}, config.Config{})
	recorder := performRequest(t, engine, http.MethodGet, "/api/health")

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}
}

func TestSetupRouterProtectsAPIRoutesWithoutSession(t *testing.T) {
	setGinTestMode(t)

	engine := SetupRouter(&gorm.DB{}, config.Config{})
	recorder := performRequest(t, engine, http.MethodGet, "/api/months/2026-05")

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, recorder.Code)
	}

	if recorder.Header().Get("WWW-Authenticate") != "" {
		t.Fatalf("expected no basic auth challenge header, got %q", recorder.Header().Get("WWW-Authenticate"))
	}
}

func TestSetupRouterAppliesBasicAuthWhenConfigured(t *testing.T) {
	setGinTestMode(t)

	cfg := config.Config{Auth: basicAuthConfigForTest(t)}
	engine := SetupRouter(&gorm.DB{}, cfg)

	recorder := performRequest(t, engine, http.MethodGet, "/api/bootstrap")
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, recorder.Code)
	}
	if recorder.Header().Get("WWW-Authenticate") == "" {
		t.Fatal("expected basic auth challenge header")
	}

	recorder = performRequest(t, engine, http.MethodGet, "/api/bootstrap", withBasicAuth("app-user", "secret"))
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}

	recorder = performRequest(t, engine, http.MethodGet, "/api/health")
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected health status %d, got %d", http.StatusOK, recorder.Code)
	}
}

func TestBasicAuthMiddlewareAllowsValidCredentials(t *testing.T) {
	database, mock, closeDB := newMockRouterDatabase(t)
	defer closeDB()
	expectResolvedUserInsert(mock, "app-user", 7)

	engine := newProtectedTestEngine(t,
		principalMiddleware(principal.Default()),
		basicAuthMiddleware(basicAuthConfigForTest(t)),
		userMiddleware(repository.NewUserRepository(database)),
	)
	registerProtectedRoute(t, engine, func(c *gin.Context) {
		current := principal.FromContext(c.Request.Context())
		if current.Username != "app-user" {
			t.Fatalf("expected username app-user, got %q", current.Username)
		}
		if !current.Authenticated {
			t.Fatal("expected principal to be authenticated")
		}
		currentUser, ok := principal.UserFromContext(c.Request.Context())
		if !ok {
			t.Fatal("expected resolved user in request context")
		}
		if currentUser.ID != 7 || currentUser.Username != "app-user" {
			t.Fatalf("expected resolved user ID 7 username app-user, got %+v", currentUser)
		}

		c.Status(http.StatusNoContent)
	})

	recorder := performRequest(t, engine, http.MethodGet, protectedRoutePath, withBasicAuth("app-user", "secret"))

	if recorder.Code != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d", http.StatusNoContent, recorder.Code)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}

func TestPrincipalMiddlewareAssignsDefaultPrincipalAndUser(t *testing.T) {
	database, mock, closeDB := newMockRouterDatabase(t)
	defer closeDB()
	expectResolvedUserInsert(mock, "single-user", 11)

	engine := newProtectedTestEngine(t,
		principalMiddleware(principal.Default()),
		userMiddleware(repository.NewUserRepository(database)),
	)
	registerProtectedRoute(t, engine, func(c *gin.Context) {
		if got := principal.FromContext(c.Request.Context()); got != principal.Default() {
			t.Fatalf("expected default principal %+v, got %+v", principal.Default(), got)
		}
		currentUser, ok := principal.UserFromContext(c.Request.Context())
		if !ok {
			t.Fatal("expected resolved user in request context")
		}
		if currentUser.ID != 11 || currentUser.Username != "single-user" {
			t.Fatalf("expected resolved user ID 11 username single-user, got %+v", currentUser)
		}

		c.Status(http.StatusNoContent)
	})

	recorder := performRequest(t, engine, http.MethodGet, protectedRoutePath)

	if recorder.Code != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d", http.StatusNoContent, recorder.Code)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}

func TestBasicAuthMiddlewareRejectsInvalidCredentials(t *testing.T) {
	engine := newProtectedTestEngine(t,
		principalMiddleware(principal.Default()),
		basicAuthMiddleware(basicAuthConfigForTest(t)),
	)
	registerProtectedRoute(t, engine, func(c *gin.Context) {
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
			recorder := performRequest(t, engine, http.MethodGet, protectedRoutePath, withBasicAuth(testCase.username, testCase.password))

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

func TestUserMiddlewareReturnsInternalServerErrorOnRepositoryFailure(t *testing.T) {
	database, mock, closeDB := newMockRouterDatabase(t)
	defer closeDB()

	expectResolvedUserInsertError(mock, "single-user", gorm.ErrInvalidDB)

	engine := newProtectedTestEngine(t,
		principalMiddleware(principal.Default()),
		userMiddleware(repository.NewUserRepository(database)),
	)
	registerProtectedRoute(t, engine, func(c *gin.Context) {
		t.Fatal("protected handler should not be called")
	})

	recorder := performRequest(t, engine, http.MethodGet, protectedRoutePath)

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, recorder.Code)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}
