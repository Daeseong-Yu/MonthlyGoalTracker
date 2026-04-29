package main

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Daeseong-Yu/MonthlyGoalTracker/backend/internal/config"
	"gorm.io/gorm"
)

type testCloser struct {
	closed bool
}

func (c *testCloser) Close() error {
	c.closed = true
	return nil
}

func TestRunWithDepsReturnsConnectError(t *testing.T) {
	expectedErr := errors.New("connect failed")
	served := false

	err := runWithDeps(testConfig(), appDeps{
		connect: func(context.Context, string) (*gorm.DB, error) {
			return nil, expectedErr
		},
		migrate: func(context.Context, *gorm.DB) error {
			t.Fatal("migrate should not be called")
			return nil
		},
		sqlDB: func(*gorm.DB) (closer, error) {
			t.Fatal("sqlDB should not be called")
			return nil, nil
		},
		serve: func(string) error {
			served = true
			return nil
		},
	})

	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected connect error, got %v", err)
	}

	if served {
		t.Fatal("serve should not be called")
	}
}

func TestRunWithDepsClosesDatabaseOnMigrateError(t *testing.T) {
	expectedErr := errors.New("migrate failed")
	testDB := &gorm.DB{}
	testSQL := &testCloser{}

	err := runWithDeps(testConfig(), appDeps{
		connect: func(context.Context, string) (*gorm.DB, error) {
			return testDB, nil
		},
		migrate: func(context.Context, *gorm.DB) error {
			return expectedErr
		},
		sqlDB: func(database *gorm.DB) (closer, error) {
			if database != testDB {
				t.Fatal("expected connected database")
			}

			return testSQL, nil
		},
		serve: func(string) error {
			t.Fatal("serve should not be called")
			return nil
		},
	})

	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected migrate error, got %v", err)
	}

	if !testSQL.closed {
		t.Fatal("expected database to be closed")
	}
}

func TestRunWithDepsServesConfiguredAddress(t *testing.T) {
	testDB := &gorm.DB{}
	testSQL := &testCloser{}
	cfg := testConfig()
	servedAddr := ""

	err := runWithDeps(cfg, appDeps{
		connect: func(context.Context, string) (*gorm.DB, error) {
			return testDB, nil
		},
		migrate: func(context.Context, *gorm.DB) error {
			return nil
		},
		sqlDB: func(*gorm.DB) (closer, error) {
			return testSQL, nil
		},
		serve: func(addr string) error {
			servedAddr = addr
			return nil
		},
	})

	if err != nil {
		t.Fatalf("expected run to succeed, got %v", err)
	}

	if servedAddr != cfg.Addr() {
		t.Fatalf("expected serve addr %q, got %q", cfg.Addr(), servedAddr)
	}

	if !testSQL.closed {
		t.Fatal("expected database to be closed")
	}
}

func TestRunWithDepsUsesSeparateConnectAndMigrateTimeouts(t *testing.T) {
	testDB := &gorm.DB{}
	testSQL := &testCloser{}

	var connectBudget time.Duration
	var migrateBudget time.Duration

	err := runWithDeps(testConfig(), appDeps{
		connect: func(ctx context.Context, _ string) (*gorm.DB, error) {
			deadline, ok := ctx.Deadline()
			if !ok {
				t.Fatal("expected connect context deadline")
			}

			connectBudget = time.Until(deadline)
			return testDB, nil
		},
		migrate: func(ctx context.Context, _ *gorm.DB) error {
			deadline, ok := ctx.Deadline()
			if !ok {
				t.Fatal("expected migrate context deadline")
			}

			migrateBudget = time.Until(deadline)
			return nil
		},
		sqlDB: func(*gorm.DB) (closer, error) {
			return testSQL, nil
		},
		serve: func(string) error {
			return nil
		},
	})

	if err != nil {
		t.Fatalf("expected run to succeed, got %v", err)
	}

	if connectBudget <= 0 || connectBudget > dbConnectTimeout {
		t.Fatalf("expected connect budget up to %v, got %v", dbConnectTimeout, connectBudget)
	}

	if migrateBudget <= dbConnectTimeout {
		t.Fatalf("expected independent migrate budget greater than connect budget, got %v", migrateBudget)
	}
}

func testConfig() config.Config {
	return config.Config{
		Host:        "127.0.0.1",
		Port:        "8080",
		DatabaseURL: "postgres://postgres:postgres@localhost:5433/monthly_goal_tracker?sslmode=disable",
	}
}
