package config

import (
	"context"
	"errors"
	"testing"
)

func TestResolveDatabaseURLPrefersExplicitURL(t *testing.T) {
	cfg := Config{
		Database: DatabaseConfig{
			URL:       "postgres://direct",
			SecretARN: "secret-arn",
		},
	}
	loader := fakeDatabaseSecretLoader{
		load: func(context.Context, string) (DatabaseSecret, error) {
			t.Fatal("secret loader should not be called when DATABASE_URL is set")
			return DatabaseSecret{}, nil
		},
	}

	databaseURL, err := cfg.ResolveDatabaseURLWithLoader(context.Background(), loader)
	if err != nil {
		t.Fatalf("expected explicit database URL, got %v", err)
	}
	if databaseURL != "postgres://direct" {
		t.Fatalf("expected explicit database URL, got %q", databaseURL)
	}
}

func TestResolveDatabaseURLFromSecret(t *testing.T) {
	cfg := Config{
		Database: DatabaseConfig{
			SecretARN: "secret-arn",
			Host:      "proxy.example.internal",
			Name:      "monthly_goal_tracker",
			SSLMode:   "require",
		},
	}
	loader := fakeDatabaseSecretLoader{
		load: func(ctx context.Context, secretARN string) (DatabaseSecret, error) {
			if err := ctx.Err(); err != nil {
				return DatabaseSecret{}, err
			}
			if secretARN != "secret-arn" {
				t.Fatalf("expected secret ARN, got %q", secretARN)
			}
			return DatabaseSecret{
				Username: "user@example",
				Password: "p@ss word",
			}, nil
		},
	}

	databaseURL, err := cfg.ResolveDatabaseURLWithLoader(context.Background(), loader)
	if err != nil {
		t.Fatalf("expected database URL from secret, got %v", err)
	}

	expectedURL := "postgres://user%40example:p%40ss%20word@proxy.example.internal:5432/monthly_goal_tracker?sslmode=require"
	if databaseURL != expectedURL {
		t.Fatalf("expected database URL %q, got %q", expectedURL, databaseURL)
	}
}

func TestResolveDatabaseURLAllowsHostAndNameFromSecret(t *testing.T) {
	cfg := Config{
		Database: DatabaseConfig{
			SecretARN: "secret-arn",
		},
	}
	loader := fakeDatabaseSecretLoader{
		load: func(context.Context, string) (DatabaseSecret, error) {
			return DatabaseSecret{
				Username: "postgres",
				Password: "secret",
				Host:     "database.example.internal",
				Port:     "6543",
				Name:     "app",
			}, nil
		},
	}

	databaseURL, err := cfg.ResolveDatabaseURLWithLoader(context.Background(), loader)
	if err != nil {
		t.Fatalf("expected database URL from secret metadata, got %v", err)
	}

	expectedURL := "postgres://postgres:secret@database.example.internal:6543/app?sslmode=require"
	if databaseURL != expectedURL {
		t.Fatalf("expected database URL %q, got %q", expectedURL, databaseURL)
	}
}

func TestResolveDatabaseURLRejectsMissingSource(t *testing.T) {
	cfg := Config{}

	_, err := cfg.ResolveDatabaseURLWithLoader(context.Background(), fakeDatabaseSecretLoader{})
	if !errors.Is(err, ErrInvalidDatabase) {
		t.Fatalf("expected invalid database config error, got %v", err)
	}
}

func TestResolveDatabaseURLRejectsIncompleteSecret(t *testing.T) {
	cfg := Config{
		Database: DatabaseConfig{
			SecretARN: "secret-arn",
			Host:      "database.example.internal",
			Name:      "app",
		},
	}
	loader := fakeDatabaseSecretLoader{
		load: func(context.Context, string) (DatabaseSecret, error) {
			return DatabaseSecret{Username: "postgres"}, nil
		},
	}

	_, err := cfg.ResolveDatabaseURLWithLoader(context.Background(), loader)
	if !errors.Is(err, ErrInvalidDatabase) {
		t.Fatalf("expected invalid database config error, got %v", err)
	}
}

func TestParseDatabaseSecret(t *testing.T) {
	secret, err := parseDatabaseSecret(`{"username":"postgres","password":"secret","host":"db.internal","port":5432,"dbname":"app"}`)
	if err != nil {
		t.Fatalf("expected parsed secret, got %v", err)
	}

	if secret.Username != "postgres" || secret.Password != "secret" || secret.Host != "db.internal" || secret.Port != "5432" || secret.Name != "app" {
		t.Fatalf("unexpected parsed secret metadata: %#v", secret)
	}
}

func TestParseDatabaseSecretRejectsInvalidJSON(t *testing.T) {
	_, err := parseDatabaseSecret(`not-json`)
	if !errors.Is(err, ErrInvalidDatabase) {
		t.Fatalf("expected invalid database config error, got %v", err)
	}
}

type fakeDatabaseSecretLoader struct {
	load func(context.Context, string) (DatabaseSecret, error)
}

func (l fakeDatabaseSecretLoader) LoadDatabaseSecret(ctx context.Context, secretARN string) (DatabaseSecret, error) {
	if l.load == nil {
		return DatabaseSecret{}, nil
	}

	return l.load(ctx, secretARN)
}
