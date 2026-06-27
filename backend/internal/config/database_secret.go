package config

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	awscfg "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
)

type DatabaseSecret struct {
	Username string
	Password string
	Host     string
	Port     string
	Name     string
}

type DatabaseSecretLoader interface {
	LoadDatabaseSecret(context.Context, string) (DatabaseSecret, error)
}

type defaultDatabaseSecretLoader struct{}

func (c Config) ResolveDatabaseURL(ctx context.Context) (string, error) {
	return c.ResolveDatabaseURLWithLoader(ctx, defaultDatabaseSecretLoader{})
}

func (c Config) ResolveDatabaseURLWithLoader(ctx context.Context, loader DatabaseSecretLoader) (string, error) {
	databaseConfig := c.Database
	if strings.TrimSpace(databaseConfig.URL) == "" {
		databaseConfig.URL = c.DatabaseURL
	}

	return databaseConfig.ResolveURL(ctx, loader)
}

func (c DatabaseConfig) ResolveURL(ctx context.Context, loader DatabaseSecretLoader) (string, error) {
	cfg := c
	cfg.URL = strings.TrimSpace(cfg.URL)
	cfg.SecretARN = strings.TrimSpace(cfg.SecretARN)
	cfg.Host = strings.TrimSpace(cfg.Host)
	cfg.Port = strings.TrimSpace(cfg.Port)
	cfg.Name = strings.TrimSpace(cfg.Name)
	cfg.SSLMode = strings.TrimSpace(cfg.SSLMode)
	if cfg.URL != "" {
		return cfg.URL, nil
	}
	if cfg.SecretARN == "" {
		return "", fmt.Errorf("%w: DATABASE_URL or DATABASE_SECRET_ARN is required", ErrInvalidDatabase)
	}
	if loader == nil {
		return "", fmt.Errorf("%w: database secret loader is required", ErrInvalidDatabase)
	}

	secret, err := loader.LoadDatabaseSecret(ctx, cfg.SecretARN)
	if err != nil {
		return "", err
	}

	username := strings.TrimSpace(secret.Username)
	password := strings.TrimSpace(secret.Password)
	host := firstNonEmpty(cfg.Host, secret.Host)
	port := firstNonEmpty(cfg.Port, secret.Port, "5432")
	name := firstNonEmpty(cfg.Name, secret.Name)
	sslMode := firstNonEmpty(cfg.SSLMode, "require")
	if username == "" || password == "" || host == "" || name == "" {
		return "", fmt.Errorf("%w: database secret, host, and name are required", ErrInvalidDatabase)
	}

	databaseURL := url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(username, password),
		Host:   net.JoinHostPort(host, port),
		Path:   "/" + strings.TrimPrefix(name, "/"),
	}
	query := databaseURL.Query()
	query.Set("sslmode", sslMode)
	databaseURL.RawQuery = query.Encode()

	return databaseURL.String(), nil
}

func (defaultDatabaseSecretLoader) LoadDatabaseSecret(ctx context.Context, secretARN string) (DatabaseSecret, error) {
	awsConfig, err := awscfg.LoadDefaultConfig(ctx)
	if err != nil {
		return DatabaseSecret{}, err
	}

	output, err := secretsmanager.NewFromConfig(awsConfig).GetSecretValue(ctx, &secretsmanager.GetSecretValueInput{
		SecretId: aws.String(secretARN),
	})
	if err != nil {
		return DatabaseSecret{}, err
	}
	if output.SecretString == nil {
		return DatabaseSecret{}, fmt.Errorf("%w: database secret string is required", ErrInvalidDatabase)
	}

	return parseDatabaseSecret(*output.SecretString)
}

func parseDatabaseSecret(rawSecret string) (DatabaseSecret, error) {
	payload := map[string]any{}
	if err := json.Unmarshal([]byte(rawSecret), &payload); err != nil {
		return DatabaseSecret{}, fmt.Errorf("%w: database secret must be JSON", ErrInvalidDatabase)
	}

	return DatabaseSecret{
		Username: secretStringField(payload, "username"),
		Password: secretStringField(payload, "password"),
		Host:     secretStringField(payload, "host"),
		Port:     secretStringField(payload, "port"),
		Name:     firstNonEmpty(secretStringField(payload, "dbname"), secretStringField(payload, "database")),
	}, nil
}

func secretStringField(payload map[string]any, key string) string {
	value, ok := payload[key]
	if !ok || value == nil {
		return ""
	}

	switch typedValue := value.(type) {
	case string:
		return strings.TrimSpace(typedValue)
	case float64:
		return strconv.FormatInt(int64(typedValue), 10)
	default:
		return strings.TrimSpace(fmt.Sprint(typedValue))
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}

	return ""
}
