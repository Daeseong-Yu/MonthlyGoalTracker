package repository

import (
	"context"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestUserRepositoryEnsureByUsernameDefaultsBlankPrincipalToSingleUser(t *testing.T) {
	repo, mock, closeDB := newMockUserRepository(t)
	defer closeDB()

	mock.ExpectQuery(`INSERT INTO "users" \("username","created_at","updated_at"\) VALUES \(\$1,\$2,\$3\) ON CONFLICT \("username"\) DO NOTHING RETURNING "id"`).
		WithArgs("single-user", fixedNow(), fixedNow()).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(11))

	user, err := repo.EnsureByUsername(context.Background(), "   ")
	if err != nil {
		t.Fatalf("expected ensure by username to succeed, got %v", err)
	}
	if user.ID != 11 {
		t.Fatalf("expected user ID 11, got %d", user.ID)
	}
	if user.Username != "single-user" {
		t.Fatalf("expected single-user username, got %q", user.Username)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}

func TestUserRepositoryEnsureByUsernameReturnsExistingUserAfterConflict(t *testing.T) {
	repo, mock, closeDB := newMockUserRepository(t)
	defer closeDB()

	mock.ExpectQuery(`INSERT INTO "users" \("username","created_at","updated_at"\) VALUES \(\$1,\$2,\$3\) ON CONFLICT \("username"\) DO NOTHING RETURNING "id"`).
		WithArgs("app-user", fixedNow(), fixedNow()).
		WillReturnRows(sqlmock.NewRows([]string{"id"}))
	mock.ExpectQuery(`SELECT \* FROM "users" WHERE username = \$1 ORDER BY "users"."id" LIMIT \$2`).
		WithArgs("app-user", 1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "username", "created_at", "updated_at"}).
			AddRow(7, "app-user", fixedNow(), fixedNow()))

	user, err := repo.EnsureByUsername(context.Background(), " app-user ")
	if err != nil {
		t.Fatalf("expected ensure by username to succeed, got %v", err)
	}
	if user.ID != 7 {
		t.Fatalf("expected user ID 7, got %d", user.ID)
	}
	if user.Username != "app-user" {
		t.Fatalf("expected app-user username, got %q", user.Username)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}

func TestUserRepositoryEnsureByUsernamePropagatesInsertError(t *testing.T) {
	repo, mock, closeDB := newMockUserRepository(t)
	defer closeDB()

	expectedErr := errors.New("insert failed")
	mock.ExpectQuery(`INSERT INTO "users" \("username","created_at","updated_at"\) VALUES \(\$1,\$2,\$3\) ON CONFLICT \("username"\) DO NOTHING RETURNING "id"`).
		WithArgs("app-user", fixedNow(), fixedNow()).
		WillReturnError(expectedErr)

	user, err := repo.EnsureByUsername(context.Background(), "app-user")
	if user != nil {
		t.Fatal("expected nil user on insert error")
	}
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected insert error, got %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}

func newMockUserRepository(t *testing.T) (*UserRepository, sqlmock.Sqlmock, func()) {
	t.Helper()

	database, mock, closeDB := newMockDatabase(t)
	return NewUserRepository(database), mock, closeDB
}
