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
	auditConfirmation = "audit-migration"
	dbConnectTimeout  = 10 * time.Second
	dbAuditTimeout    = 30 * time.Second
)

var (
	errAuditConfirmationRequired = errors.New("audit confirmation required")
	errAuditFailed               = errors.New("audit failed")
	errAuditExpectationMismatch  = errors.New("audit expectation mismatch")
)

var auditTableNames = []string{
	"users",
	"sessions",
	"email_verification_tokens",
	"password_reset_tokens",
	"goals",
	"daily_memos",
	"goal_checks",
}

type auditEvent struct {
	Confirm  string           `json:"confirm,omitempty"`
	Expected map[string]int64 `json:"expected,omitempty"`
}

type auditResult struct {
	Status                  string           `json:"status"`
	Counts                  map[string]int64 `json:"counts,omitempty"`
	OwnershipViolationCount int64            `json:"ownershipViolationCount,omitempty"`
}

type closer interface {
	Close() error
}

type auditDeps struct {
	connect func(context.Context, string) (*gorm.DB, error)
	audit   func(context.Context, *gorm.DB) (auditResult, error)
	sqlDB   func(*gorm.DB) (closer, error)
}

type auditRuntime struct {
	cfg  config.Config
	deps auditDeps
}

func main() {
	runtime := auditRuntime{
		cfg: config.Load(),
		deps: auditDeps{
			connect: db.Connect,
			audit:   auditDatabase,
			sqlDB: func(database *gorm.DB) (closer, error) {
				return database.DB()
			},
		},
	}

	lambda.Start(runtime.Handle)
}

func (r auditRuntime) Handle(ctx context.Context, event auditEvent) (auditResult, error) {
	if event.Confirm != auditConfirmation {
		return auditResult{Status: "rejected"}, errAuditConfirmationRequired
	}

	result, err := runAudit(ctx, r.cfg, event.Expected, r.deps)
	if err != nil {
		log.Printf("database audit failed: %T", err)
		return auditResult{Status: "failed"}, errAuditFailed
	}

	return result, nil
}

func runAudit(ctx context.Context, cfg config.Config, expected map[string]int64, deps auditDeps) (auditResult, error) {
	if deps.connect == nil {
		return auditResult{}, errors.New("audit database connector is required")
	}
	if deps.audit == nil {
		return auditResult{}, errors.New("audit function is required")
	}
	if deps.sqlDB == nil {
		return auditResult{}, errors.New("audit sql database resolver is required")
	}
	if err := cfg.Validate(); err != nil {
		return auditResult{}, err
	}

	connectCtx, cancelConnect := context.WithTimeout(ctx, dbConnectTimeout)
	databaseURL, err := cfg.ResolveDatabaseURL(connectCtx)
	if err != nil {
		cancelConnect()
		return auditResult{}, err
	}

	database, err := deps.connect(connectCtx, databaseURL)
	cancelConnect()
	if err != nil {
		return auditResult{}, err
	}

	sqlDB, err := deps.sqlDB(database)
	if err != nil {
		return auditResult{}, err
	}
	defer func() {
		if err := sqlDB.Close(); err != nil {
			log.Printf("failed to close audit database connection: %T", err)
		}
	}()

	auditCtx, cancelAudit := context.WithTimeout(ctx, dbAuditTimeout)
	defer cancelAudit()

	result, err := deps.audit(auditCtx, database)
	if err != nil {
		return auditResult{}, err
	}
	if err := validateExpectedCounts(result.Counts, expected); err != nil {
		return auditResult{}, err
	}
	if result.OwnershipViolationCount != 0 {
		return auditResult{}, errAuditExpectationMismatch
	}

	result.Status = "completed"
	return result, nil
}

func auditDatabase(ctx context.Context, database *gorm.DB) (auditResult, error) {
	counts := make(map[string]int64, len(auditTableNames))
	for _, tableName := range auditTableNames {
		count, err := tableRowCount(ctx, database, tableName)
		if err != nil {
			return auditResult{}, err
		}
		counts[tableName] = count
	}

	ownershipViolationCount, err := ownershipViolationCount(ctx, database)
	if err != nil {
		return auditResult{}, err
	}

	return auditResult{
		Counts:                  counts,
		OwnershipViolationCount: ownershipViolationCount,
	}, nil
}

func tableRowCount(ctx context.Context, database *gorm.DB, tableName string) (int64, error) {
	var count int64
	err := database.WithContext(ctx).Raw("SELECT count(*) FROM " + tableName).Scan(&count).Error
	return count, err
}

func ownershipViolationCount(ctx context.Context, database *gorm.DB) (int64, error) {
	var count int64
	err := database.WithContext(ctx).Raw(`
		SELECT
			(SELECT count(*) FROM goals LEFT JOIN users ON users.id = goals.user_id WHERE users.id IS NULL) +
			(SELECT count(*) FROM daily_memos LEFT JOIN users ON users.id = daily_memos.user_id WHERE users.id IS NULL) +
			(SELECT count(*) FROM goal_checks LEFT JOIN users ON users.id = goal_checks.user_id WHERE users.id IS NULL) +
			(SELECT count(*) FROM goal_checks LEFT JOIN goals ON goals.id = goal_checks.goal_id WHERE goals.id IS NULL) +
			(SELECT count(*) FROM goal_checks INNER JOIN goals ON goals.id = goal_checks.goal_id WHERE goal_checks.user_id <> goals.user_id)
	`).Scan(&count).Error
	return count, err
}

func validateExpectedCounts(actual, expected map[string]int64) error {
	for tableName, expectedCount := range expected {
		actualCount, ok := actual[tableName]
		if !ok || actualCount != expectedCount {
			return errAuditExpectationMismatch
		}
	}

	return nil
}
