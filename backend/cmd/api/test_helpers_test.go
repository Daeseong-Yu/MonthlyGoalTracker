package main

import (
	"context"
	"testing"

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

type testAppDeps struct {
	database *gorm.DB
	sqlDB    *testCloser
	deps     appDeps
}

func newTestAppDeps(t *testing.T) *testAppDeps {
	t.Helper()

	testDeps := &testAppDeps{
		database: &gorm.DB{},
		sqlDB:    &testCloser{},
	}

	testDeps.deps = appDeps{
		connect: func(context.Context, string) (*gorm.DB, error) {
			return testDeps.database, nil
		},
		migrate: func(context.Context, *gorm.DB) error {
			return nil
		},
		sqlDB: func(database *gorm.DB) (closer, error) {
			if database != testDeps.database {
				t.Fatal("expected connected database")
			}

			return testDeps.sqlDB, nil
		},
		serve: func(database *gorm.DB, _ string) error {
			if database != testDeps.database {
				t.Fatal("expected connected database to be served")
			}

			return nil
		},
	}

	return testDeps
}

func failFastAppDeps(t *testing.T) appDeps {
	t.Helper()

	return appDeps{
		connect: failConnect(t),
		migrate: failMigrate(t),
		sqlDB:   failSQLDB(t),
		serve:   failServe(t),
	}
}

func failConnect(t *testing.T) func(context.Context, string) (*gorm.DB, error) {
	t.Helper()

	return func(context.Context, string) (*gorm.DB, error) {
		t.Helper()
		t.Fatal("connect should not be called")
		return nil, nil
	}
}

func failMigrate(t *testing.T) func(context.Context, *gorm.DB) error {
	t.Helper()

	return func(context.Context, *gorm.DB) error {
		t.Helper()
		t.Fatal("migrate should not be called")
		return nil
	}
}

func failSQLDB(t *testing.T) func(*gorm.DB) (closer, error) {
	t.Helper()

	return func(*gorm.DB) (closer, error) {
		t.Helper()
		t.Fatal("sqlDB should not be called")
		return nil, nil
	}
}

func failServe(t *testing.T) func(*gorm.DB, string) error {
	t.Helper()

	return func(*gorm.DB, string) error {
		t.Helper()
		t.Fatal("serve should not be called")
		return nil
	}
}

func testConfig() config.Config {
	return config.Config{
		Host:        "127.0.0.1",
		Port:        "8080",
		DatabaseURL: "test-database-url",
	}
}
