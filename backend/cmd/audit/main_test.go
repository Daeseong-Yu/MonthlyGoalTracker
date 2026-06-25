package main

import (
	"context"
	"errors"
	"testing"

	"github.com/Daeseong-Yu/MonthlyGoalTracker/backend/internal/config"
	"gorm.io/gorm"
)

func TestAuditHandlerRequiresConfirmation(t *testing.T) {
	calledConnect := false
	runtime := auditRuntime{
		cfg: testAuditConfig(),
		deps: auditDeps{
			connect: func(context.Context, string) (*gorm.DB, error) {
				calledConnect = true
				return &gorm.DB{}, nil
			},
		},
	}

	result, err := runtime.Handle(context.Background(), auditEvent{})
	if !errors.Is(err, errAuditConfirmationRequired) {
		t.Fatalf("expected confirmation error, got %v", err)
	}
	if result.Status != "rejected" {
		t.Fatalf("expected rejected status, got %q", result.Status)
	}
	if calledConnect {
		t.Fatal("expected audit to skip database connection without confirmation")
	}
}

func TestRunAuditValidatesConfigBeforeConnecting(t *testing.T) {
	calledConnect := false
	_, err := runAudit(context.Background(), config.Config{}, nil, auditDeps{
		connect: func(context.Context, string) (*gorm.DB, error) {
			calledConnect = true
			return &gorm.DB{}, nil
		},
		audit: func(context.Context, *gorm.DB) (auditResult, error) {
			return auditResult{}, nil
		},
		sqlDB: func(*gorm.DB) (closer, error) {
			return fakeAuditCloser{}, nil
		},
	})

	if err == nil {
		t.Fatal("expected invalid config error")
	}
	if calledConnect {
		t.Fatal("expected config validation before database connection")
	}
}

func TestAuditHandlerRunsAuditAndClosesDatabase(t *testing.T) {
	cfg := testAuditConfig()
	closed := false
	audited := false
	runtime := auditRuntime{
		cfg: cfg,
		deps: auditDeps{
			connect: func(ctx context.Context, databaseURL string) (*gorm.DB, error) {
				if err := ctx.Err(); err != nil {
					return nil, err
				}
				if databaseURL != cfg.DatabaseURL {
					t.Fatalf("expected database URL from config, got %q", databaseURL)
				}
				return &gorm.DB{}, nil
			},
			audit: func(ctx context.Context, database *gorm.DB) (auditResult, error) {
				if err := ctx.Err(); err != nil {
					return auditResult{}, err
				}
				if database == nil {
					t.Fatal("expected database")
				}
				audited = true
				return auditResult{
					Counts: map[string]int64{
						"users": 2,
						"goals": 4,
					},
				}, nil
			},
			sqlDB: func(*gorm.DB) (closer, error) {
				return fakeAuditCloser{close: func() {
					closed = true
				}}, nil
			},
		},
	}

	result, err := runtime.Handle(context.Background(), auditEvent{
		Confirm: auditConfirmation,
		Expected: map[string]int64{
			"users": 2,
			"goals": 4,
		},
	})
	if err != nil {
		t.Fatalf("expected audit success, got %v", err)
	}
	if result.Status != "completed" {
		t.Fatalf("expected completed status, got %q", result.Status)
	}
	if result.Counts["users"] != 2 || result.Counts["goals"] != 4 {
		t.Fatalf("expected safe row counts in result, got %v", result.Counts)
	}
	if !audited {
		t.Fatal("expected audit to run")
	}
	if !closed {
		t.Fatal("expected database connection to close")
	}
}

func TestAuditHandlerRejectsOwnershipViolation(t *testing.T) {
	runtime := auditRuntime{
		cfg: testAuditConfig(),
		deps: auditDeps{
			connect: func(context.Context, string) (*gorm.DB, error) {
				return &gorm.DB{}, nil
			},
			audit: func(context.Context, *gorm.DB) (auditResult, error) {
				return auditResult{
					Counts:                  map[string]int64{"users": 1},
					OwnershipViolationCount: 1,
				}, nil
			},
			sqlDB: func(*gorm.DB) (closer, error) {
				return fakeAuditCloser{}, nil
			},
		},
	}

	result, err := runtime.Handle(context.Background(), auditEvent{Confirm: auditConfirmation})
	if !errors.Is(err, errAuditFailed) {
		t.Fatalf("expected generic audit failure, got %v", err)
	}
	if result.Status != "failed" {
		t.Fatalf("expected failed status, got %q", result.Status)
	}
}

func TestAuditHandlerRejectsExpectedCountMismatch(t *testing.T) {
	runtime := auditRuntime{
		cfg: testAuditConfig(),
		deps: auditDeps{
			connect: func(context.Context, string) (*gorm.DB, error) {
				return &gorm.DB{}, nil
			},
			audit: func(context.Context, *gorm.DB) (auditResult, error) {
				return auditResult{
					Counts: map[string]int64{"users": 1},
				}, nil
			},
			sqlDB: func(*gorm.DB) (closer, error) {
				return fakeAuditCloser{}, nil
			},
		},
	}

	result, err := runtime.Handle(context.Background(), auditEvent{
		Confirm: auditConfirmation,
		Expected: map[string]int64{
			"users": 2,
		},
	})
	if !errors.Is(err, errAuditFailed) {
		t.Fatalf("expected generic audit failure, got %v", err)
	}
	if result.Status != "failed" {
		t.Fatalf("expected failed status, got %q", result.Status)
	}
}

func TestAuditHandlerReturnsGenericFailure(t *testing.T) {
	runtime := auditRuntime{
		cfg: testAuditConfig(),
		deps: auditDeps{
			connect: func(context.Context, string) (*gorm.DB, error) {
				return &gorm.DB{}, nil
			},
			audit: func(context.Context, *gorm.DB) (auditResult, error) {
				return auditResult{}, errors.New("database detail")
			},
			sqlDB: func(*gorm.DB) (closer, error) {
				return fakeAuditCloser{}, nil
			},
		},
	}

	result, err := runtime.Handle(context.Background(), auditEvent{Confirm: auditConfirmation})
	if !errors.Is(err, errAuditFailed) {
		t.Fatalf("expected generic audit failure, got %v", err)
	}
	if result.Status != "failed" {
		t.Fatalf("expected failed status, got %q", result.Status)
	}
}

func testAuditConfig() config.Config {
	cfg := config.Config{
		Host:        "127.0.0.1",
		Port:        "8080",
		DatabaseURL: "postgres://example",
	}
	cfg.Session = cfg.Session.WithDefaults()
	cfg.AuthFlow = cfg.AuthFlow.WithDefaults()
	return cfg
}

type fakeAuditCloser struct {
	close func()
}

func (c fakeAuditCloser) Close() error {
	if c.close != nil {
		c.close()
	}
	return nil
}
