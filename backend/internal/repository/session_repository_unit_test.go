package repository

import (
	"context"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestSessionRepositoryDeleteOthersByUserIDAndTokenHashExcludesCurrentSession(t *testing.T) {
	database, mock, closeDB := newMockDatabase(t)
	defer closeDB()

	repo := NewSessionRepository(database)
	mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM "sessions" WHERE user_id = $1 AND token_hash <> $2`)).
		WithArgs(uint(7), "current-token-hash").
		WillReturnResult(sqlmock.NewResult(0, 2))

	if err := repo.DeleteOthersByUserIDAndTokenHash(context.Background(), 7, "current-token-hash"); err != nil {
		t.Fatalf("expected other sessions delete to succeed, got %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}
