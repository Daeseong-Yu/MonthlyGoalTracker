package main

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Daeseong-Yu/MonthlyGoalTracker/backend/internal/config"
	"gorm.io/gorm"
)

func TestRunWithDepsReturnsConnectError(t *testing.T) {
	expectedErr := errors.New("connect failed")
	deps := failFastAppDeps(t)
	deps.connect = func(context.Context, string) (*gorm.DB, error) {
		return nil, expectedErr
	}

	err := runWithDeps(testConfig(), deps, runOptions{mode: runModeServe})

	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected connect error, got %v", err)
	}
}

func TestRunWithDepsRejectsNonLoopbackHostBeforeConnect(t *testing.T) {
	cfg := testConfig()
	cfg.Host = "0.0.0.0"

	err := runWithDeps(cfg, failFastAppDeps(t), runOptions{mode: runModeServe})

	if !errors.Is(err, config.ErrUnsafeHost) {
		t.Fatalf("expected unsafe host error, got %v", err)
	}
}

func TestRunWithDepsClosesDatabaseOnMigrateError(t *testing.T) {
	expectedErr := errors.New("migrate failed")
	testDeps := newTestAppDeps(t)
	testDeps.deps.migrate = func(context.Context, *gorm.DB) error {
		return expectedErr
	}
	testDeps.deps.serve = failServe(t)

	err := runWithDeps(testConfig(), testDeps.deps, runOptions{mode: runModeServe})

	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected migrate error, got %v", err)
	}

	if !testDeps.sqlDB.closed {
		t.Fatal("expected database to be closed")
	}
}

func TestRunWithDepsServesConfiguredAddress(t *testing.T) {
	testDeps := newTestAppDeps(t)
	cfg := testConfig()
	servedAddr := ""

	testDeps.deps.serve = func(database *gorm.DB, addr string) error {
		if database != testDeps.database {
			t.Fatal("expected connected database to be served")
		}

		servedAddr = addr
		return nil
	}

	err := runWithDeps(cfg, testDeps.deps, runOptions{mode: runModeServe})

	if err != nil {
		t.Fatalf("expected run to succeed, got %v", err)
	}

	if servedAddr != cfg.Addr() {
		t.Fatalf("expected serve addr %q, got %q", cfg.Addr(), servedAddr)
	}

	if !testDeps.sqlDB.closed {
		t.Fatal("expected database to be closed")
	}
}

func TestRunWithDepsUsesSeparateConnectAndMigrateTimeouts(t *testing.T) {
	testDeps := newTestAppDeps(t)

	var connectBudget time.Duration
	var migrateBudget time.Duration

	testDeps.deps.connect = func(ctx context.Context, _ string) (*gorm.DB, error) {
		deadline, ok := ctx.Deadline()
		if !ok {
			t.Fatal("expected connect context deadline")
		}

		connectBudget = time.Until(deadline)
		return testDeps.database, nil
	}
	testDeps.deps.migrate = func(ctx context.Context, _ *gorm.DB) error {
		deadline, ok := ctx.Deadline()
		if !ok {
			t.Fatal("expected migrate context deadline")
		}

		migrateBudget = time.Until(deadline)
		return nil
	}

	err := runWithDeps(testConfig(), testDeps.deps, runOptions{mode: runModeServe})

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

func TestRunWithDepsMigrateOnlySkipsServe(t *testing.T) {
	testDeps := newTestAppDeps(t)
	testDeps.deps.serve = failServe(t)

	err := runWithDeps(testConfig(), testDeps.deps, runOptions{mode: runModeMigrateOnly})

	if err != nil {
		t.Fatalf("expected migrate-only run to succeed, got %v", err)
	}

	if !testDeps.sqlDB.closed {
		t.Fatal("expected database to be closed")
	}
}
