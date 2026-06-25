package main

import (
	"context"
	"errors"
	"testing"

	"github.com/Daeseong-Yu/MonthlyGoalTracker/backend/internal/config"
	"gorm.io/gorm"
)

func TestMigrationHandlerRequiresConfirmation(t *testing.T) {
	calledConnect := false
	runtime := migrationRuntime{
		cfg: testMigrationConfig(),
		deps: migrationDeps{
			connect: func(context.Context, string) (*gorm.DB, error) {
				calledConnect = true
				return &gorm.DB{}, nil
			},
		},
	}

	result, err := runtime.Handle(context.Background(), migrationEvent{})
	if !errors.Is(err, errMigrationConfirmationRequired) {
		t.Fatalf("expected confirmation error, got %v", err)
	}
	if result.Status != "rejected" {
		t.Fatalf("expected rejected status, got %q", result.Status)
	}
	if calledConnect {
		t.Fatal("expected migration to skip database connection without confirmation")
	}
}

func TestRunMigrationValidatesConfigBeforeConnecting(t *testing.T) {
	calledConnect := false
	err := runMigration(context.Background(), config.Config{}, migrationDeps{
		connect: func(context.Context, string) (*gorm.DB, error) {
			calledConnect = true
			return &gorm.DB{}, nil
		},
		migrate: func(context.Context, *gorm.DB) error {
			return nil
		},
		sqlDB: func(*gorm.DB) (closer, error) {
			return fakeMigrationCloser{}, nil
		},
	})

	if err == nil {
		t.Fatal("expected invalid config error")
	}
	if calledConnect {
		t.Fatal("expected config validation before database connection")
	}
}

func TestMigrationHandlerRunsMigrationAndClosesDatabase(t *testing.T) {
	cfg := testMigrationConfig()
	closed := false
	migrated := false
	runtime := migrationRuntime{
		cfg: cfg,
		deps: migrationDeps{
			connect: func(ctx context.Context, databaseURL string) (*gorm.DB, error) {
				if err := ctx.Err(); err != nil {
					return nil, err
				}
				if databaseURL != cfg.DatabaseURL {
					t.Fatalf("expected database URL from config, got %q", databaseURL)
				}
				return &gorm.DB{}, nil
			},
			migrate: func(ctx context.Context, database *gorm.DB) error {
				if err := ctx.Err(); err != nil {
					return err
				}
				if database == nil {
					t.Fatal("expected database")
				}
				migrated = true
				return nil
			},
			sqlDB: func(*gorm.DB) (closer, error) {
				return fakeMigrationCloser{close: func() {
					closed = true
				}}, nil
			},
		},
	}

	result, err := runtime.Handle(context.Background(), migrationEvent{Confirm: migrationConfirmation})
	if err != nil {
		t.Fatalf("expected migration success, got %v", err)
	}
	if result.Status != "completed" {
		t.Fatalf("expected completed status, got %q", result.Status)
	}
	if !migrated {
		t.Fatal("expected migration to run")
	}
	if !closed {
		t.Fatal("expected database connection to close")
	}
}

func TestMigrationHandlerReturnsGenericFailure(t *testing.T) {
	runtime := migrationRuntime{
		cfg: testMigrationConfig(),
		deps: migrationDeps{
			connect: func(context.Context, string) (*gorm.DB, error) {
				return &gorm.DB{}, nil
			},
			migrate: func(context.Context, *gorm.DB) error {
				return errors.New("database detail")
			},
			sqlDB: func(*gorm.DB) (closer, error) {
				return fakeMigrationCloser{}, nil
			},
		},
	}

	result, err := runtime.Handle(context.Background(), migrationEvent{Confirm: migrationConfirmation})
	if !errors.Is(err, errMigrationFailed) {
		t.Fatalf("expected generic migration failure, got %v", err)
	}
	if result.Status != "failed" {
		t.Fatalf("expected failed status, got %q", result.Status)
	}
}

func testMigrationConfig() config.Config {
	cfg := config.Config{
		Host:        "127.0.0.1",
		Port:        "8080",
		DatabaseURL: "postgres://example",
	}
	cfg.Session = cfg.Session.WithDefaults()
	cfg.AuthFlow = cfg.AuthFlow.WithDefaults()
	return cfg
}

type fakeMigrationCloser struct {
	close func()
}

func (c fakeMigrationCloser) Close() error {
	if c.close != nil {
		c.close()
	}
	return nil
}
