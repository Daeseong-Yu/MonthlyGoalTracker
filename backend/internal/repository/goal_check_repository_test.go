package repository

import (
	"errors"
	"strconv"
	"testing"
	"time"

	"github.com/Daeseong-Yu/MonthlyGoalTracker/backend/internal/domain"
	"gorm.io/gorm"
)

func TestGoalCheckRepositoryIntegration(t *testing.T) {
	database := openIntegrationDatabase(t)
	goalRepo := NewGoalRepository(database)
	checkRepo := NewGoalCheckRepository(database)

	prefix := "goal check repository integration " + time.Now().UTC().Format("20060102150405.000000000")
	username := prefix + " user"
	otherUsername := prefix + " other user"
	ctx := integrationUserContext(t, database, username)
	otherCtx := integrationUserContext(t, database, otherUsername)
	t.Cleanup(func() {
		cleanupGoalsByTitlePrefix(t, database, prefix)
		cleanupIntegrationUserByUsername(t, database, username)
		cleanupIntegrationUserByUsername(t, database, otherUsername)
	})
	cleanupGoalsByTitlePrefix(t, database, prefix)

	rangeStart := uniqueIntegrationDate()
	rangeEnd := rangeStart.AddDate(0, 0, 30)
	checkDate := rangeStart.AddDate(0, 0, 14)

	goal := createGoal(t, ctx, goalRepo, prefix+" goal", rangeStart, nil)

	if err := checkRepo.SetCompleted(ctx, goal.ID, checkDate, true); err != nil {
		t.Fatalf("expected check insert to succeed, got %v", err)
	}
	if err := checkRepo.SetCompleted(ctx, goal.ID, checkDate, true); err != nil {
		t.Fatalf("expected duplicate check insert to be ignored, got %v", err)
	}

	exists, err := checkRepo.Exists(ctx, goal.ID, checkDate)
	if err != nil {
		t.Fatalf("expected check exists lookup to succeed, got %v", err)
	}
	if !exists {
		t.Fatal("expected check to exist")
	}

	otherGoal := createGoal(t, ctx, goalRepo, prefix+" other goal", rangeStart, nil)
	if err := checkRepo.SetCompleted(ctx, otherGoal.ID, rangeStart, true); err != nil {
		t.Fatalf("expected range start check insert to succeed, got %v", err)
	}
	if err := checkRepo.SetCompleted(ctx, otherGoal.ID, rangeEnd, true); err != nil {
		t.Fatalf("expected range end check insert to succeed, got %v", err)
	}

	otherUserGoal := createGoal(t, otherCtx, goalRepo, prefix+" isolated other user goal", rangeStart, nil)
	if err := checkRepo.SetCompleted(otherCtx, otherUserGoal.ID, rangeStart, true); err != nil {
		t.Fatalf("expected other user check insert to succeed, got %v", err)
	}
	if err := checkRepo.SetCompleted(ctx, otherUserGoal.ID, rangeStart, true); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("expected other user goal check write to be rejected, got %v", err)
	}

	goalChecks, err := checkRepo.ListByDateRange(ctx, rangeStart, rangeEnd)
	if err != nil {
		t.Fatalf("expected goal checks list to succeed, got %v", err)
	}
	checksByGoalAndDate := goalChecksByGoalAndDate(goalChecks, map[uint]bool{
		goal.ID:      true,
		otherGoal.ID: true,
	})
	assertExactGoalChecks(t, checksByGoalAndDate, []string{
		goalCheckKey(goal.ID, checkDate),
		goalCheckKey(otherGoal.ID, rangeStart),
		goalCheckKey(otherGoal.ID, rangeEnd),
	})

	otherGoalChecks, err := checkRepo.ListByDateRange(otherCtx, rangeStart, rangeEnd)
	if err != nil {
		t.Fatalf("expected other user goal checks list to succeed, got %v", err)
	}
	otherChecksByGoalAndDate := goalChecksByGoalAndDate(otherGoalChecks, map[uint]bool{
		otherUserGoal.ID: true,
	})
	assertExactGoalChecks(t, otherChecksByGoalAndDate, []string{
		goalCheckKey(otherUserGoal.ID, rangeStart),
	})

	if err := checkRepo.SetCompleted(ctx, goal.ID, checkDate, false); err != nil {
		t.Fatalf("expected check delete to succeed, got %v", err)
	}

	exists, err = checkRepo.Exists(ctx, goal.ID, checkDate)
	if err != nil {
		t.Fatalf("expected check exists lookup after delete to succeed, got %v", err)
	}
	if exists {
		t.Fatal("expected check to be deleted")
	}
}

func goalChecksByGoalAndDate(goalChecks []domain.GoalCheck, goalIDs map[uint]bool) map[string]bool {
	values := make(map[string]bool)
	for _, goalCheck := range goalChecks {
		if !goalIDs[goalCheck.GoalID] {
			continue
		}

		values[goalCheckKey(goalCheck.GoalID, goalCheck.Date)] = true
	}

	return values
}

func assertExactGoalChecks(t *testing.T, checks map[string]bool, expectedKeys []string) {
	t.Helper()

	if len(checks) != len(expectedKeys) {
		t.Fatalf("expected %d goal checks, got %d: %v", len(expectedKeys), len(checks), checks)
	}

	for _, key := range expectedKeys {
		if !checks[key] {
			t.Fatalf("expected goal check %s to exist", key)
		}
	}
}

func goalCheckKey(goalID uint, date time.Time) string {
	return strconv.FormatUint(uint64(goalID), 10) + ":" + date.Format(time.DateOnly)
}
