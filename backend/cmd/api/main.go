package main

import (
	"context"
	"flag"
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
	migrateOnly := flag.Bool("migrate-only", false, "run database migrations and exit")
	flag.Parse()

	opts := runOptions{mode: runModeServe}
	if *migrateOnly {
		opts.mode = runModeMigrateOnly
	}

	cfg := config.Load()

	if err := run(cfg, opts); err != nil {
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
	serve   func(*gorm.DB, string) error
}

type runMode int

const (
	runModeServe runMode = iota
	runModeMigrateOnly
)

type runOptions struct {
	mode runMode
}

func run(cfg config.Config, opts runOptions) error {
	return runWithDeps(cfg, appDeps{
		connect: db.Connect,
		migrate: db.Migrate,
		sqlDB: func(database *gorm.DB) (closer, error) {
			return database.DB()
		},
		serve: func(database *gorm.DB, addr string) error {
			return router.SetupRouter(database, cfg).Run(addr)
		},
	}, opts)
}

func runWithDeps(cfg config.Config, deps appDeps, opts runOptions) error {
	if err := cfg.Validate(); err != nil {
		return err
	}

	connectCtx, cancelConnect := context.WithTimeout(context.Background(), dbConnectTimeout)
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
			log.Printf("failed to close database connection: %v", err)
		}
	}()

	migrateCtx, cancelMigrate := context.WithTimeout(context.Background(), dbMigrateTimeout)
	defer cancelMigrate()

	if err := deps.migrate(migrateCtx, database); err != nil {
		return err
	}

	if opts.mode == runModeMigrateOnly {
		return nil
	}

	return deps.serve(database, cfg.Addr())
}
