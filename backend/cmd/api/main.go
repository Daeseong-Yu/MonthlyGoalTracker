package main

import (
	"context"
	"log"
	"time"

	"github.com/Daeseong-Yu/MonthlyGoalTracker/backend/internal/config"
	"github.com/Daeseong-Yu/MonthlyGoalTracker/backend/internal/db"
	"github.com/Daeseong-Yu/MonthlyGoalTracker/backend/internal/router"
	"gorm.io/gorm"
)

const (
	dbConnectTimeout = 10 * time.Second
	dbMigrateTimeout = 30 * time.Second
)

func main() {
	cfg := config.Load()

	if err := run(cfg); err != nil {
		log.Fatal(err)
	}
}

type closer interface {
	Close() error
}

type appDeps struct {
	connect func(context.Context, string) (*gorm.DB, error)
	migrate func(context.Context, *gorm.DB) error
	sqlDB   func(*gorm.DB) (closer, error)
	serve   func(string) error
}

func run(cfg config.Config) error {
	return runWithDeps(cfg, appDeps{
		connect: db.Connect,
		migrate: db.Migrate,
		sqlDB: func(database *gorm.DB) (closer, error) {
			return database.DB()
		},
		serve: func(addr string) error {
			return router.SetupRouter().Run(addr)
		},
	})
}

func runWithDeps(cfg config.Config, deps appDeps) error {
	connectCtx, cancelConnect := context.WithTimeout(context.Background(), dbConnectTimeout)

	database, err := deps.connect(connectCtx, cfg.DatabaseURL)
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
			log.Printf("failed to close database connection: %v", err)
		}
	}()

	migrateCtx, cancelMigrate := context.WithTimeout(context.Background(), dbMigrateTimeout)
	defer cancelMigrate()

	if err := deps.migrate(migrateCtx, database); err != nil {
		return err
	}

	return deps.serve(cfg.Addr())
}
