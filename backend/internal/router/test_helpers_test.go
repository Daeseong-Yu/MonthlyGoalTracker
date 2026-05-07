package router

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/Daeseong-Yu/MonthlyGoalTracker/backend/internal/config"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

const protectedRoutePath = "/protected"

type requestOption func(*http.Request)

func setGinTestMode(t *testing.T) {
	t.Helper()

	gin.SetMode(gin.TestMode)
}

func newTestEngine(t *testing.T) *gin.Engine {
	t.Helper()

	setGinTestMode(t)
	return gin.New()
}

func newProtectedTestEngine(t *testing.T, middleware ...gin.HandlerFunc) *gin.Engine {
	t.Helper()

	engine := newTestEngine(t)
	engine.Use(middleware...)

	return engine
}

func registerProtectedRoute(t *testing.T, engine *gin.Engine, handler gin.HandlerFunc) {
	t.Helper()

	engine.GET(protectedRoutePath, handler)
}

func performRequest(t *testing.T, engine *gin.Engine, method, target string, opts ...requestOption) *httptest.ResponseRecorder {
	t.Helper()

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(method, target, nil)
	for _, opt := range opts {
		opt(request)
	}

	engine.ServeHTTP(recorder, request)

	return recorder
}

func withBasicAuth(username, password string) requestOption {
	return func(request *http.Request) {
		request.SetBasicAuth(username, password)
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

func expectResolvedUserInsertError(mock sqlmock.Sqlmock, username string, err error) {
	mock.ExpectQuery(`INSERT INTO "users" \("username","created_at","updated_at"\) VALUES \(\$1,\$2,\$3\) ON CONFLICT \("username"\) DO NOTHING RETURNING "id"`).
		WithArgs(username, routerFixedNow(), routerFixedNow()).
		WillReturnError(err)
}

func routerFixedNow() time.Time {
	return time.Date(2099, time.January, 1, 1, 2, 3, 0, time.UTC)
}
