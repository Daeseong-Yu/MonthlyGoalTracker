package service

import (
	"testing"
	"time"

	"github.com/Daeseong-Yu/MonthlyGoalTracker/backend/internal/repository"
)

func TestGoalServiceIntegrationPreservesNonUTCLocalCalendarDay(t *testing.T) {
	database := openIntegrationDatabase(t)
	repo := repository.NewGoalRepository(database)
	service := NewGoalService(repo)
	kst := time.FixedZone("KST", 9*60*60)

	prefix := "goal service integration " + time.Now().UTC().Format("20060102150405.000000000")
	username := prefix + " user"
	ctx := integrationUserContext(t, database, username)
	title := prefix + " create"
	t.Cleanup(func() {
		cleanupIntegrationGoalsByTitlePrefix(t, database, prefix)
		cleanupIntegrationUserByUsername(t, database, username)
	})
	cleanupIntegrationGoalsByTitlePrefix(t, database, prefix)

	created, err := service.CreateGoal(ctx, "2199-04", title, time.Date(2199, time.April, 1, 0, 30, 0, 0, kst))
	if err != nil {
		t.Fatalf("expected create to succeed, got %v", err)
	}
	if created == nil {
		t.Fatal("expected created goal")
	}
	assertUTCDateOnlyEqual(t, created.StartDate, date(2199, time.April, 1))

	storedAfterCreate, err := repo.FindByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("expected created goal lookup to succeed, got %v", err)
	}
	assertUTCDateOnlyEqual(t, storedAfterCreate.StartDate, date(2199, time.April, 1))

	deactivated, err := service.DeactivateGoal(ctx, created.ID, time.Date(2199, time.April, 3, 8, 0, 0, 0, kst))
	if err != nil {
		t.Fatalf("expected deactivate to succeed, got %v", err)
	}
	if deactivated == nil || deactivated.EndDate == nil {
		t.Fatal("expected deactivated goal with end date")
	}
	assertUTCDateOnlyEqual(t, *deactivated.EndDate, date(2199, time.April, 3))

	storedAfterDeactivate, err := repo.FindByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("expected deactivated goal lookup to succeed, got %v", err)
	}
	if storedAfterDeactivate.EndDate == nil {
		t.Fatal("expected stored end date")
	}
	assertUTCDateOnlyEqual(t, *storedAfterDeactivate.EndDate, date(2199, time.April, 3))
}
