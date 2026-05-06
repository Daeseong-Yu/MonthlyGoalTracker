package router

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/Daeseong-Yu/MonthlyGoalTracker/backend/internal/config"
	"github.com/Daeseong-Yu/MonthlyGoalTracker/backend/internal/principal"
	"github.com/Daeseong-Yu/MonthlyGoalTracker/backend/internal/repository"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/postgres"
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

	database, mock, closeDB := newMockRouterDatabase(t)
	defer closeDB()
	expectResolvedUserInsert(mock, "app-user", 7)

	engine := gin.New()
	engine.Use(principalMiddleware(principal.Default()))
	engine.Use(basicAuthMiddleware(basicAuthConfigForTest(t)))
	engine.Use(userMiddleware(repository.NewUserRepository(database)))
	engine.GET("/protected", func(c *gin.Context) {
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

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/protected", nil)
	request.SetBasicAuth("app-user", "secret")
	engine.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d", http.StatusNoContent, recorder.Code)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}

func TestPrincipalMiddlewareAssignsDefaultPrincipalAndUser(t *testing.T) {
	gin.SetMode(gin.TestMode)

	database, mock, closeDB := newMockRouterDatabase(t)
	defer closeDB()
	expectResolvedUserInsert(mock, "single-user", 11)

	engine := gin.New()
	engine.Use(principalMiddleware(principal.Default()))
	engine.Use(userMiddleware(repository.NewUserRepository(database)))
	engine.GET("/protected", func(c *gin.Context) {
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

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/protected", nil)
	engine.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d", http.StatusNoContent, recorder.Code)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
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

func TestUserMiddlewareReturnsInternalServerErrorOnRepositoryFailure(t *testing.T) {
	gin.SetMode(gin.TestMode)

	database, mock, closeDB := newMockRouterDatabase(t)
	defer closeDB()

	mock.ExpectQuery(`INSERT INTO "users" \("username","created_at","updated_at"\) VALUES \(\$1,\$2,\$3\) ON CONFLICT \("username"\) DO NOTHING RETURNING "id"`).
		WithArgs("single-user", routerFixedNow(), routerFixedNow()).
		WillReturnError(gorm.ErrInvalidDB)

	engine := gin.New()
	engine.Use(principalMiddleware(principal.Default()))
	engine.Use(userMiddleware(repository.NewUserRepository(database)))
	engine.GET("/protected", func(c *gin.Context) {
		t.Fatal("protected handler should not be called")
	})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/protected", nil)
	engine.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, recorder.Code)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
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

func newMockRouterDatabase(t *testing.T) (*gorm.DB, sqlmock.Sqlmock, func()) {
	t.Helper()

	sqlDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sql mock: %v", err)
	}

	database, err := gorm.Open(postgres.New(postgres.Config{
		Conn:                 sqlDB,
		PreferSimpleProtocol: true,
	}), &gorm.Config{
		DisableAutomaticPing:   true,
		SkipDefaultTransaction: true,
		NowFunc:                routerFixedNow,
	})
	if err != nil {
		t.Fatalf("failed to create gorm database: %v", err)
	}

	return database, mock, func() {
		_ = sqlDB.Close()
	}
}

func expectResolvedUserInsert(mock sqlmock.Sqlmock, username string, id uint) {
	mock.ExpectQuery(`INSERT INTO "users" \("username","created_at","updated_at"\) VALUES \(\$1,\$2,\$3\) ON CONFLICT \("username"\) DO NOTHING RETURNING "id"`).
		WithArgs(username, routerFixedNow(), routerFixedNow()).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(id))
}

func routerFixedNow() time.Time {
	return time.Date(2099, time.January, 1, 1, 2, 3, 0, time.UTC)
}
