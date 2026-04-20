package controller

import (
	"fmt"
	"strings"
	"time"
)

const (
	dateOnlyLayout     = "2006-01-02"
	dateTimeOnlyLayout = "2006-01-02 15:04:05"
)

func parseDateBoundary(raw string, endOfDay bool) (time.Time, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return time.Time{}, fmt.Errorf("empty date")
	}

	// Interpret non-timezone inputs in local time to keep date boundaries
	// consistent with business queries and DB session timezone.
	if t, err := time.ParseInLocation(dateTimeOnlyLayout, trimmed, time.Local); err == nil {
		return t, nil
	}

	if t, err := time.Parse(time.RFC3339, trimmed); err == nil {
		return t.In(time.Local), nil
	}

	t, err := time.ParseInLocation(dateOnlyLayout, trimmed, time.Local)
	if err != nil {
		return time.Time{}, err
	}
	if endOfDay {
		return time.Date(t.Year(), t.Month(), t.Day(), 23, 59, 59, 0, t.Location()), nil
	}
	return t, nil
}

func parseOptionalDateBoundary(raw string, endOfDay bool) (*time.Time, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil, nil
	}
	t, err := parseDateBoundary(trimmed, endOfDay)
	if err != nil {
		return nil, err
	}
	return &t, nil
}
