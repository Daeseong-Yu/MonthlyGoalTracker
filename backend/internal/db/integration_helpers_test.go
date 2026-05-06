package db

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"gorm.io/gorm"
)

const (
	integrationTestEnv     = "RUN_DB_INTEGRATION"
	integrationDatabaseEnv = "DATABASE_URL"
	integrationTimeout     = 30 * time.Second
	integrationRetryDelay  = 500 * time.Millisecond
)

func requireDatabaseIntegration(t *testing.T, description string) (context.Context, context.CancelFunc, string) {
	t.Helper()

	if os.Getenv(integrationTestEnv) != "1" {
		t.Skipf("set %s=1 to run %s", integrationTestEnv, description)
	}

	databaseURL := os.Getenv(integrationDatabaseEnv)
	if databaseURL == "" {
		t.Fatalf("%s is required for %s", integrationDatabaseEnv, description)
	}

	ctx, cancel := context.WithTimeout(context.Background(), integrationTimeout)
	return ctx, cancel, databaseURL
}

func openIntegrationDatabase(t *testing.T, ctx context.Context, databaseURL string) *gorm.DB {
	t.Helper()

	var (
		database *gorm.DB
		err      error
	)
	for {
		database, err = Connect(ctx, databaseURL)
		if err == nil {
			return database
		}
		if ctx.Err() != nil {
			t.Fatalf("expected database connection, got %v", err)
		}

		time.Sleep(integrationRetryDelay)
	}
}

func closeIntegrationDatabase(t *testing.T, database *gorm.DB) {
	t.Helper()

	sqlDB, err := database.DB()
	if err != nil {
		t.Fatalf("expected sql database handle, got %v", err)
	}

	if err := sqlDB.Close(); err != nil {
		t.Fatalf("failed to close database connection: %v", err)
	}
}

func openIntegrationSchemaDatabase(
	t *testing.T,
	ctx context.Context,
	adminDB *gorm.DB,
	databaseURL string,
	schemaPrefix string,
) *gorm.DB {
	t.Helper()

	schemaName := fmt.Sprintf("%s_%d", schemaPrefix, time.Now().UTC().UnixNano())
	createSchema(t, adminDB, schemaName)

	schemaDB := openIntegrationDatabase(t, ctx, databaseURLWithSearchPath(t, databaseURL, schemaName))
	t.Cleanup(func() {
		dropSchema(t, adminDB, schemaName)
	})
	t.Cleanup(func() {
		closeIntegrationDatabase(t, schemaDB)
	})

	return schemaDB
}

func createSchema(t *testing.T, database *gorm.DB, schemaName string) {
	t.Helper()

	if err := database.Exec(`CREATE SCHEMA ` + quoteIdentifier(schemaName)).Error; err != nil {
		t.Fatalf("failed to create schema %q: %v", schemaName, err)
	}
}

func dropSchema(t *testing.T, database *gorm.DB, schemaName string) {
	t.Helper()

	if err := database.Exec(`DROP SCHEMA IF EXISTS ` + quoteIdentifier(schemaName) + ` CASCADE`).Error; err != nil {
		t.Fatalf("failed to drop schema %q: %v", schemaName, err)
	}
}

func databaseURLWithSearchPath(t *testing.T, databaseURL, schemaName string) string {
	t.Helper()

	parsedURL, err := url.Parse(databaseURL)
	if err != nil {
		t.Fatalf("failed to parse database URL: %v", err)
	}

	query := parsedURL.Query()
	query.Set("search_path", schemaName)
	parsedURL.RawQuery = query.Encode()
	return parsedURL.String()
}

func quoteIdentifier(value string) string {
	return `"` + strings.ReplaceAll(value, `"`, `""`) + `"`
}
