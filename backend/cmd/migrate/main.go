package main

import (
	"context"
	"errors"
	"log"
	"time"

	"github.com/Daeseong-Yu/MonthlyGoalTracker/backend/internal/config"
	"github.com/Daeseong-Yu/MonthlyGoalTracker/backend/internal/db"
	"github.com/aws/aws-lambda-go/lambda"
	"gorm.io/gorm"
)

const (
	migrationConfirmation = "run-migrations"
	dbConnectTimeout      = 10 * time.Second
	dbMigrateTimeout      = 60 * time.Second
)

var (
	errMigrationConfirmationRequired = errors.New("migration confirmation required")
	errMigrationFailed               = errors.New("migration failed")
)

type migrationEvent struct {
	Confirm string `json:"confirm,omitempty"`
}

type migrationResult struct {
	Status string `json:"status"`
}

type closer interface {
	Close() error
}

type migrationDeps struct {
	connect func(context.Context, string) (*gorm.DB, error)
	migrate func(context.Context, *gorm.DB) error
	sqlDB   func(*gorm.DB) (closer, error)
}

type migrationRuntime struct {
	cfg  config.Config
	deps migrationDeps
}

func main() {
	runtime := migrationRuntime{
		cfg: config.Load(),
		deps: migrationDeps{
			connect: db.Connect,
			migrate: db.Migrate,
			sqlDB: func(database *gorm.DB) (closer, error) {
				return database.DB()
			},
		},
	}

	lambda.Start(runtime.Handle)
}

func (r migrationRuntime) Handle(ctx context.Context, event migrationEvent) (migrationResult, error) {
	if event.Confirm != migrationConfirmation {
		return migrationResult{Status: "rejected"}, errMigrationConfirmationRequired
	}

	if err := runMigration(ctx, r.cfg, r.deps); err != nil {
		log.Printf("database migration failed: %T", err)
		return migrationResult{Status: "failed"}, errMigrationFailed
	}

	return migrationResult{Status: "completed"}, nil
}

func runMigration(ctx context.Context, cfg config.Config, deps migrationDeps) error {
	if deps.connect == nil {
		return errors.New("migration database connector is required")
	}
	if deps.migrate == nil {
		return errors.New("migration function is required")
	}
	if deps.sqlDB == nil {
		return errors.New("migration sql database resolver is required")
	}
	if err := cfg.Validate(); err != nil {
		return err
	}

	connectCtx, cancelConnect := context.WithTimeout(ctx, dbConnectTimeout)
	databaseURL, err := cfg.ResolveDatabaseURL(connectCtx)
	if err != nil {
		cancelConnect()
		return err
	}

	database, err := deps.connect(connectCtx, databaseURL)
	cancelConnect()
	if err != nil {
		return err
	}

	sqlDB, err := deps.sqlDB(database)
	if err != nil {
		return err
	}
	defer func() {
		if err := sqlDB.Close(); err != nil {
			log.Printf("failed to close migration database connection: %T", err)
		}
	}()

	migrateCtx, cancelMigrate := context.WithTimeout(ctx, dbMigrateTimeout)
	defer cancelMigrate()

	return deps.migrate(migrateCtx, database)
}
