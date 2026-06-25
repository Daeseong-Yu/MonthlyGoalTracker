package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/Daeseong-Yu/MonthlyGoalTracker/backend/internal/config"
	"github.com/Daeseong-Yu/MonthlyGoalTracker/backend/internal/db"
	"github.com/Daeseong-Yu/MonthlyGoalTracker/backend/internal/router"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	ginadapter "github.com/awslabs/aws-lambda-go-api-proxy/gin"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

const dbConnectTimeout = 10 * time.Second

type lambdaDeps struct {
	connect     func(context.Context, string) (*gorm.DB, error)
	setupRouter func(*gorm.DB, config.Config) *gin.Engine
}

type lambdaRuntime struct {
	mu      sync.Mutex
	init    func(context.Context) (*ginadapter.GinLambdaV2, error)
	adapter *ginadapter.GinLambdaV2
}

func main() {
	runtime := newLambdaRuntime(func(ctx context.Context) (*ginadapter.GinLambdaV2, error) {
		return newLambdaAdapter(ctx, config.Load(), lambdaDeps{
			connect:     db.Connect,
			setupRouter: router.SetupRouter,
		})
	})

	lambda.Start(runtime.Handle)
}

func newLambdaRuntime(init func(context.Context) (*ginadapter.GinLambdaV2, error)) *lambdaRuntime {
	return &lambdaRuntime{
		init: init,
	}
}

func newLambdaAdapter(ctx context.Context, cfg config.Config, deps lambdaDeps) (*ginadapter.GinLambdaV2, error) {
	if deps.connect == nil {
		return nil, errors.New("lambda database connector is required")
	}
	if deps.setupRouter == nil {
		return nil, errors.New("lambda router setup is required")
	}
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	connectCtx, cancelConnect := context.WithTimeout(ctx, dbConnectTimeout)
	defer cancelConnect()

	databaseURL, err := cfg.ResolveDatabaseURL(connectCtx)
	if err != nil {
		return nil, err
	}

	database, err := deps.connect(connectCtx, databaseURL)
	if err != nil {
		return nil, err
	}

	return ginadapter.NewV2(deps.setupRouter(database, cfg)), nil
}

func (r *lambdaRuntime) Handle(ctx context.Context, req events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
	adapter, err := r.adapterFor(ctx)
	if err != nil {
		log.Printf("failed to initialize lambda API handler: %T", err)
		return internalServerErrorResponse(), nil
	}

	return adapter.ProxyWithContext(ctx, req)
}

func (r *lambdaRuntime) adapterFor(ctx context.Context) (*ginadapter.GinLambdaV2, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.adapter != nil {
		return r.adapter, nil
	}

	adapter, err := r.init(ctx)
	if err != nil {
		return nil, err
	}

	r.adapter = adapter
	return adapter, nil
}

func internalServerErrorResponse() events.APIGatewayV2HTTPResponse {
	return events.APIGatewayV2HTTPResponse{
		StatusCode: http.StatusInternalServerError,
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
		Body: `{"error":"internal server error"}`,
	}
}
