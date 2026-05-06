package service

import (
	"testing"
	"time"
)

func date(year int, month time.Month, day int) time.Time {
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
}

func assertDateEqual(t *testing.T, actual time.Time, expected time.Time) {
	t.Helper()
	if !actual.Equal(expected) {
		t.Fatalf("expected date %s, got %s", expected.Format(time.RFC3339), actual.Format(time.RFC3339))
	}
}
