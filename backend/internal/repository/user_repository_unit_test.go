package repository

import (
	"context"
	"errors"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/Daeseong-Yu/MonthlyGoalTracker/backend/internal/domain"
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

func TestUserRepositoryCreateWithPasswordClaimsLegacyDefaultUser(t *testing.T) {
	repo, mock, closeDB := newMockUserRepository(t)
	defer closeDB()

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "users" WHERE username = $1 ORDER BY "users"."id" LIMIT $2 FOR UPDATE`)).
		WithArgs("single-user", 1).
		WillReturnRows(sqlmock.NewRows(userColumns()).
			AddRow(9, "single-user", "", "", "ko", nil, fixedNow(), fixedNow()))
	expectLegacyOwnedRowsQuery(mock, 9, 1)
	verifiedAt := fixedNow()
	mock.ExpectExec(regexp.QuoteMeta(`UPDATE "users" SET "email"=$1,"email_verified_at"=$2,"locale"=$3,"password_hash"=$4,"username"=$5,"updated_at"=$6 WHERE id = $7`)).
		WithArgs("owner@example.com", verifiedAt, "en", "hash", "owner@example.com", fixedNow(), uint(9)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "users" WHERE "users"."id" = $1 ORDER BY "users"."id" LIMIT $2`)).
		WithArgs(uint(9), 1).
		WillReturnRows(sqlmock.NewRows(userColumns()).
			AddRow(9, "owner@example.com", "owner@example.com", "hash", "en", verifiedAt, fixedNow(), fixedNow()))
	mock.ExpectCommit()

	user, err := repo.CreateWithPassword(context.Background(), " Owner@Example.com ", "hash", "en", true, &verifiedAt)
	if err != nil {
		t.Fatalf("expected create with password to succeed, got %v", err)
	}
	if user.ID != 9 {
		t.Fatalf("expected legacy user ID 9 to be preserved, got %d", user.ID)
	}
	if user.Username != "owner@example.com" {
		t.Fatalf("expected normalized username, got %q", user.Username)
	}
	if user.Email != "owner@example.com" {
		t.Fatalf("expected normalized email, got %q", user.Email)
	}
	if user.PasswordHash != "hash" {
		t.Fatalf("expected password hash to be set, got %q", user.PasswordHash)
	}
	if user.Locale != "en" {
		t.Fatalf("expected locale en, got %q", user.Locale)
	}
	if user.EmailVerifiedAt == nil || !user.EmailVerifiedAt.Equal(verifiedAt) {
		t.Fatalf("expected email verified at %s, got %v", verifiedAt, user.EmailVerifiedAt)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}

func TestUserRepositoryCreateWithPasswordIgnoresEmptyDefaultUserWithoutLegacyData(t *testing.T) {
	repo, mock, closeDB := newMockUserRepository(t)
	defer closeDB()

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "users" WHERE username = $1 ORDER BY "users"."id" LIMIT $2 FOR UPDATE`)).
		WithArgs("single-user", 1).
		WillReturnRows(sqlmock.NewRows(userColumns()).
			AddRow(9, "single-user", "", "", "ko", nil, fixedNow(), fixedNow()))
	expectLegacyOwnedRowsQuery(mock, 9, 0)
	mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "users" ("username","email","password_hash","locale","email_verified_at","created_at","updated_at") VALUES ($1,$2,$3,$4,$5,$6,$7) RETURNING "id"`)).
		WithArgs("new@example.com", "new@example.com", "hash", "ko", nil, fixedNow(), fixedNow()).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(21))
	mock.ExpectCommit()

	user, err := repo.CreateWithPassword(context.Background(), " New@Example.com ", "hash", "ko", false, nil)
	if err != nil {
		t.Fatalf("expected create with password to ignore empty default user, got %v", err)
	}
	if user.ID != 21 {
		t.Fatalf("expected new user ID 21, got %d", user.ID)
	}
	if user.Username != "new@example.com" {
		t.Fatalf("expected normalized username, got %q", user.Username)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}

func TestUserRepositoryCreateWithPasswordCreatesNewUserWithoutLegacyClaim(t *testing.T) {
	repo, mock, closeDB := newMockUserRepository(t)
	defer closeDB()

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "users" WHERE username = $1 ORDER BY "users"."id" LIMIT $2 FOR UPDATE`)).
		WithArgs("single-user", 1).
		WillReturnRows(sqlmock.NewRows(userColumns()))
	mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "users" ("username","email","password_hash","locale","email_verified_at","created_at","updated_at") VALUES ($1,$2,$3,$4,$5,$6,$7) RETURNING "id"`)).
		WithArgs("new@example.com", "new@example.com", "hash", "ko", nil, fixedNow(), fixedNow()).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(21))
	mock.ExpectCommit()

	user, err := repo.CreateWithPassword(context.Background(), " New@Example.com ", "hash", "ko", false, nil)
	if err != nil {
		t.Fatalf("expected create with password to succeed, got %v", err)
	}
	if user.ID != 21 {
		t.Fatalf("expected new user ID 21, got %d", user.ID)
	}
	if user.Username != "new@example.com" {
		t.Fatalf("expected normalized username, got %q", user.Username)
	}
	if user.Email != "new@example.com" {
		t.Fatalf("expected normalized email, got %q", user.Email)
	}
	if user.PasswordHash != "hash" {
		t.Fatalf("expected password hash to be set, got %q", user.PasswordHash)
	}
	if user.Locale != "ko" {
		t.Fatalf("expected locale ko, got %q", user.Locale)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}

func TestUserRepositoryCreateWithPasswordRequiresClaimWhenLegacyUserIsUnclaimed(t *testing.T) {
	repo, mock, closeDB := newMockUserRepository(t)
	defer closeDB()

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "users" WHERE username = $1 ORDER BY "users"."id" LIMIT $2 FOR UPDATE`)).
		WithArgs("single-user", 1).
		WillReturnRows(sqlmock.NewRows(userColumns()).
			AddRow(9, "single-user", "", "", "ko", nil, fixedNow(), fixedNow()))
	expectLegacyOwnedRowsQuery(mock, 9, 2)
	mock.ExpectRollback()

	user, err := repo.CreateWithPassword(context.Background(), " New@Example.com ", "hash", "ko", false, nil)
	if user != nil {
		t.Fatal("expected nil user when legacy claim is required")
	}
	if !errors.Is(err, domain.ErrLegacyClaimRequired) {
		t.Fatalf("expected ErrLegacyClaimRequired, got %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}

func TestUserRepositoryUpdatePasswordHashAndReplaceSessionsUsesTransaction(t *testing.T) {
	repo, mock, closeDB := newMockUserRepository(t)
	defer closeDB()

	session := &domain.Session{
		TokenHash:     "token-hash",
		CSRFTokenHash: "csrf-token-hash",
		ExpiresAt:     fixedNow().Add(time.Hour),
		LastUsedAt:    fixedNow(),
	}

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "users" WHERE "users"."id" = $1 ORDER BY "users"."id" LIMIT $2`)).
		WithArgs(uint(7), 1).
		WillReturnRows(sqlmock.NewRows(userColumns()).
			AddRow(7, "owner@example.com", "owner@example.com", "old-hash", "ko", fixedNow(), fixedNow(), fixedNow()))
	mock.ExpectExec(regexp.QuoteMeta(`UPDATE "users" SET "password_hash"=$1,"updated_at"=$2 WHERE "id" = $3`)).
		WithArgs("new-hash", fixedNow(), uint(7)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM "sessions" WHERE user_id = $1`)).
		WithArgs(uint(7)).
		WillReturnResult(sqlmock.NewResult(0, 2))
	mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "sessions" ("user_id","token_hash","csrf_token_hash","expires_at","last_used_at","created_at","updated_at") VALUES ($1,$2,$3,$4,$5,$6,$7) RETURNING "id"`)).
		WithArgs(uint(7), "token-hash", "csrf-token-hash", fixedNow().Add(time.Hour), fixedNow(), fixedNow(), fixedNow()).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(31))
	mock.ExpectCommit()

	user, err := repo.UpdatePasswordHashAndReplaceSessions(context.Background(), 7, "new-hash", session)
	if err != nil {
		t.Fatalf("expected password hash update and session replacement to succeed, got %v", err)
	}
	if user.ID != 7 {
		t.Fatalf("expected user ID 7, got %d", user.ID)
	}
	if user.PasswordHash != "new-hash" {
		t.Fatalf("expected new password hash, got %q", user.PasswordHash)
	}
	if user.Locale != "ko" {
		t.Fatalf("expected locale ko to be preserved, got %q", user.Locale)
	}
	if session.UserID != 7 {
		t.Fatalf("expected replacement session user ID 7, got %d", session.UserID)
	}
	if session.ID != 31 {
		t.Fatalf("expected replacement session ID 31, got %d", session.ID)
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

func userColumns() []string {
	return []string{"id", "username", "email", "password_hash", "locale", "email_verified_at", "created_at", "updated_at"}
}

func expectLegacyOwnedRowsQuery(mock sqlmock.Sqlmock, userID int, ownedRows int64) {
	mock.ExpectQuery(`SELECT\s+\(SELECT count\(\*\) FROM goals WHERE user_id = \$1\)\s+\+\s+\(SELECT count\(\*\) FROM daily_memos WHERE user_id = \$2\)\s+\+\s+\(SELECT count\(\*\) FROM goal_checks WHERE user_id = \$3\)\s+AS owned_rows`).
		WithArgs(userID, userID, userID).
		WillReturnRows(sqlmock.NewRows([]string{"owned_rows"}).AddRow(ownedRows))
}
