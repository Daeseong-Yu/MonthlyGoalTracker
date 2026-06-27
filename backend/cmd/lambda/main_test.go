package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/Daeseong-Yu/MonthlyGoalTracker/backend/internal/config"
	"github.com/Daeseong-Yu/MonthlyGoalTracker/backend/internal/router"
	"github.com/aws/aws-lambda-go/events"
	ginadapter "github.com/awslabs/aws-lambda-go-api-proxy/gin"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func TestNewLambdaAdapterValidatesConfigBeforeConnect(t *testing.T) {
	cfg := testConfig()
	cfg.Host = "0.0.0.0"

	_, err := newLambdaAdapter(context.Background(), cfg, lambdaDeps{
		connect: func(context.Context, string) (*gorm.DB, error) {
			t.Fatal("connect should not be called")
			return nil, nil
		},
		setupRouter: func(*gorm.DB, config.Config) *gin.Engine {
			t.Fatal("setupRouter should not be called")
			return nil
		},
	})

	if !errors.Is(err, config.ErrUnsafeHost) {
		t.Fatalf("expected unsafe host error, got %v", err)
	}
}

func TestNewLambdaAdapterConnectsAndBuildsRouterWithoutMigrationPath(t *testing.T) {
	cfg := testConfig()
	database := &gorm.DB{}
	connectCalled := false
	setupCalled := false

	adapter, err := newLambdaAdapter(context.Background(), cfg, lambdaDeps{
		connect: func(ctx context.Context, databaseURL string) (*gorm.DB, error) {
			deadline, ok := ctx.Deadline()
			if !ok {
				t.Fatal("expected connect context deadline")
			}
			if budget := time.Until(deadline); budget <= 0 || budget > dbConnectTimeout {
				t.Fatalf("expected connect budget up to %v, got %v", dbConnectTimeout, budget)
			}
			if databaseURL != cfg.DatabaseURL {
				t.Fatalf("expected database URL %q, got %q", cfg.DatabaseURL, databaseURL)
			}

			connectCalled = true
			return database, nil
		},
		setupRouter: func(actualDatabase *gorm.DB, actualConfig config.Config) *gin.Engine {
			if actualDatabase != database {
				t.Fatal("expected connected database")
			}
			if actualConfig.DatabaseURL != cfg.DatabaseURL {
				t.Fatal("expected lambda config to be passed to router")
			}

			setupCalled = true
			return gin.New()
		},
	})

	if err != nil {
		t.Fatalf("expected lambda adapter to initialize, got %v", err)
	}
	if adapter == nil {
		t.Fatal("expected lambda adapter")
	}
	if !connectCalled {
		t.Fatal("expected database connection")
	}
	if !setupCalled {
		t.Fatal("expected router setup")
	}
}

func TestLambdaRuntimeInitializesOnce(t *testing.T) {
	initCalls := 0
	runtime := newLambdaRuntime(func(context.Context) (*ginadapter.GinLambdaV2, error) {
		initCalls++

		engine := gin.New()
		engine.GET("/api/health", func(c *gin.Context) {
			c.JSON(200, gin.H{"status": "ok"})
		})

		return ginadapter.NewV2(engine), nil
	})

	for i := 0; i < 2; i++ {
		response, err := runtime.Handle(context.Background(), events.APIGatewayV2HTTPRequest{
			RequestContext: events.APIGatewayV2HTTPRequestContext{
				HTTP: events.APIGatewayV2HTTPRequestContextHTTPDescription{
					Method: "GET",
					Path:   "/api/health",
				},
			},
			RawPath: "/api/health",
		})
		if err != nil {
			t.Fatalf("expected handler response, got %v", err)
		}
		if response.StatusCode != 200 {
			t.Fatalf("expected status 200, got %d", response.StatusCode)
		}
	}

	if initCalls != 1 {
		t.Fatalf("expected one init call, got %d", initCalls)
	}
}

func TestLambdaRuntimeReturnsGenericInitError(t *testing.T) {
	expectedErr := errors.New("connect failed")
	runtime := newLambdaRuntime(func(context.Context) (*ginadapter.GinLambdaV2, error) {
		return nil, expectedErr
	})

	response, err := runtime.Handle(context.Background(), events.APIGatewayV2HTTPRequest{})

	if err != nil {
		t.Fatalf("expected generic response without lambda error, got %v", err)
	}
	if response.StatusCode != 500 {
		t.Fatalf("expected status 500, got %d", response.StatusCode)
	}
	if response.Body != `{"error":"internal server error"}` {
		t.Fatalf("expected generic error body, got %q", response.Body)
	}
}

func TestLambdaRuntimeRetriesAfterInitError(t *testing.T) {
	initCalls := 0
	runtime := newLambdaRuntime(func(context.Context) (*ginadapter.GinLambdaV2, error) {
		initCalls++
		if initCalls == 1 {
			return nil, errors.New("temporary connect failed")
		}

		engine := gin.New()
		engine.GET("/api/health", func(c *gin.Context) {
			c.JSON(200, gin.H{"status": "ok"})
		})

		return ginadapter.NewV2(engine), nil
	})

	firstResponse, err := runtime.Handle(context.Background(), events.APIGatewayV2HTTPRequest{})
	if err != nil {
		t.Fatalf("expected first handler response, got %v", err)
	}
	if firstResponse.StatusCode != 500 {
		t.Fatalf("expected first status 500, got %d", firstResponse.StatusCode)
	}

	secondResponse, err := runtime.Handle(context.Background(), events.APIGatewayV2HTTPRequest{
		RequestContext: events.APIGatewayV2HTTPRequestContext{
			HTTP: events.APIGatewayV2HTTPRequestContextHTTPDescription{
				Method: "GET",
				Path:   "/api/health",
			},
		},
		RawPath: "/api/health",
	})
	if err != nil {
		t.Fatalf("expected second handler response, got %v", err)
	}
	if secondResponse.StatusCode != 200 {
		t.Fatalf("expected second status 200, got %d", secondResponse.StatusCode)
	}
	if initCalls != 2 {
		t.Fatalf("expected retry after failed init, got %d init calls", initCalls)
	}
}

func TestLambdaRuntimeReturnsCookiesForHTTPAPIV2(t *testing.T) {
	runtime := newLambdaRuntime(func(context.Context) (*ginadapter.GinLambdaV2, error) {
		engine := gin.New()
		engine.POST("/api/auth/login", func(c *gin.Context) {
			c.SetSameSite(http.SameSiteLaxMode)
			c.SetCookie("mgt_session", "session-token", 3600, "/", "", true, true)
			c.SetCookie("mgt_csrf", "csrf-token", 3600, "/", "", true, false)
			c.JSON(200, gin.H{"csrfToken": "csrf-token"})
		})

		return ginadapter.NewV2(engine), nil
	})

	response, err := runtime.Handle(context.Background(), events.APIGatewayV2HTTPRequest{
		RequestContext: events.APIGatewayV2HTTPRequestContext{
			HTTP: events.APIGatewayV2HTTPRequestContextHTTPDescription{
				Method: "POST",
				Path:   "/api/auth/login",
			},
		},
		RawPath: "/api/auth/login",
	})
	if err != nil {
		t.Fatalf("expected login response, got %v", err)
	}
	if response.StatusCode != 200 {
		t.Fatalf("expected status 200, got %d with body %q", response.StatusCode, response.Body)
	}

	sessionCookie := cookieByName(response.Cookies, "mgt_session")
	csrfCookie := cookieByName(response.Cookies, "mgt_csrf")
	if sessionCookie == "" {
		t.Fatalf("expected session cookie in HTTP API v2 cookies array, got %#v", response.Cookies)
	}
	if csrfCookie == "" {
		t.Fatalf("expected csrf cookie in HTTP API v2 cookies array, got %#v", response.Cookies)
	}
	assertCookieAttribute(t, sessionCookie, "Secure")
	assertCookieAttribute(t, sessionCookie, "HttpOnly")
	assertCookieAttribute(t, sessionCookie, "SameSite=Lax")
	assertCookieAttribute(t, csrfCookie, "Secure")
	assertCookieMissingAttribute(t, csrfCookie, "HttpOnly")
	assertCookieAttribute(t, csrfCookie, "SameSite=Lax")
}

func TestLambdaRuntimeMapsHTTPAPIV2RequestCookiesAndHeaders(t *testing.T) {
	runtime := newLambdaRuntime(func(context.Context) (*ginadapter.GinLambdaV2, error) {
		engine := gin.New()
		engine.PUT("/api/checks", func(c *gin.Context) {
			sessionCookie, sessionErr := c.Cookie("mgt_session")
			csrfCookie, csrfErr := c.Cookie("mgt_csrf")
			csrfHeader := c.GetHeader("X-CSRF-Token")
			if sessionErr != nil || csrfErr != nil || sessionCookie != "session-token" || csrfCookie != csrfHeader {
				c.JSON(http.StatusForbidden, gin.H{"error": "csrf mismatch"})
				return
			}

			c.Status(http.StatusNoContent)
		})

		return ginadapter.NewV2(engine), nil
	})

	response, err := runtime.Handle(context.Background(), events.APIGatewayV2HTTPRequest{
		Cookies: []string{
			"mgt_session=session-token",
			"mgt_csrf=csrf-token",
		},
		Headers: map[string]string{
			"x-csrf-token": "csrf-token",
		},
		RequestContext: events.APIGatewayV2HTTPRequestContext{
			HTTP: events.APIGatewayV2HTTPRequestContextHTTPDescription{
				Method: "PUT",
				Path:   "/api/checks",
			},
		},
		RawPath: "/api/checks",
	})
	if err != nil {
		t.Fatalf("expected checks response, got %v", err)
	}
	if response.StatusCode != http.StatusNoContent {
		t.Fatalf("expected status 204, got %d with body %q", response.StatusCode, response.Body)
	}
}

func TestLambdaAdapterServesAnonymousBootstrapThroughRouter(t *testing.T) {
	cfg := testConfig()
	adapter, err := newLambdaAdapter(context.Background(), cfg, lambdaDeps{
		connect: func(context.Context, string) (*gorm.DB, error) {
			return &gorm.DB{}, nil
		},
		setupRouter: router.SetupRouter,
	})
	if err != nil {
		t.Fatalf("expected lambda adapter to initialize, got %v", err)
	}

	response, err := adapter.ProxyWithContext(context.Background(), events.APIGatewayV2HTTPRequest{
		Headers: map[string]string{
			"accept-language": "ko-KR,ko;q=0.9,en;q=0.8",
		},
		RequestContext: events.APIGatewayV2HTTPRequestContext{
			HTTP: events.APIGatewayV2HTTPRequestContextHTTPDescription{
				Method: "GET",
				Path:   "/api/bootstrap",
			},
		},
		RawPath: "/api/bootstrap",
	})
	if err != nil {
		t.Fatalf("expected bootstrap response, got %v", err)
	}
	if response.StatusCode != 200 {
		t.Fatalf("expected status 200, got %d with body %q", response.StatusCode, response.Body)
	}

	var payload struct {
		Authenticated bool `json:"authenticated"`
	}
	if err := json.Unmarshal([]byte(response.Body), &payload); err != nil {
		t.Fatalf("expected JSON bootstrap response, got %v", err)
	}
	if payload.Authenticated {
		t.Fatal("expected anonymous bootstrap to be unauthenticated")
	}
}

func cookieByName(cookies []string, name string) string {
	for _, cookie := range cookies {
		if strings.HasPrefix(cookie, name+"=") {
			return cookie
		}
	}

	return ""
}

func assertCookieAttribute(t *testing.T, cookie string, attribute string) {
	t.Helper()
	if !strings.Contains(strings.ToLower(cookie), strings.ToLower(attribute)) {
		t.Fatalf("expected cookie %q to contain attribute %q", cookie, attribute)
	}
}

func assertCookieMissingAttribute(t *testing.T, cookie string, attribute string) {
	t.Helper()
	if strings.Contains(strings.ToLower(cookie), strings.ToLower(attribute)) {
		t.Fatalf("expected cookie %q to omit attribute %q", cookie, attribute)
	}
}

func testConfig() config.Config {
	return config.Config{
		Host:        "127.0.0.1",
		Port:        "8080",
		DatabaseURL: "test-database-url",
	}
}
