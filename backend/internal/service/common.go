package service

import (
	"errors"
	"fmt"
	"time"
)

var ErrInvalidMonth = errors.New("invalid month")

func parseMonthRange(month string) (time.Time, time.Time, error) {
	parsedMonth, err := time.Parse("2006-01", month)
	if err != nil || parsedMonth.Format("2006-01") != month {
		return time.Time{}, time.Time{}, fmt.Errorf("%w: %q", ErrInvalidMonth, month)
	}

	monthStart := time.Date(parsedMonth.Year(), parsedMonth.Month(), 1, 0, 0, 0, 0, time.UTC)
	monthEnd := monthStart.AddDate(0, 1, -1)

	return monthStart, monthEnd, nil
}

func normalizeDateUTC(value time.Time) time.Time {
	year, month, day := value.Date()
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
}
