package controller

import (
	"testing"
	"time"
)

func TestParseTrafficDateRangeUsesHalfOpenLocalDays(t *testing.T) {
	start, end, err := parseTrafficDateRange("2026-07-01", "2026-07-03")
	if err != nil {
		t.Fatalf("parseTrafficDateRange() error = %v", err)
	}
	if got := start.Format("2006-01-02 15:04:05"); got != "2026-07-01 00:00:00" {
		t.Fatalf("start = %s", got)
	}
	if got := end.Format("2006-01-02 15:04:05"); got != "2026-07-04 00:00:00" {
		t.Fatalf("end = %s", got)
	}
	if start.Location() != time.Local || end.Location() != time.Local {
		t.Fatalf("date range location must be time.Local")
	}
}

func TestParseTrafficDateRangeRejectsInvalidRange(t *testing.T) {
	if _, _, err := parseTrafficDateRange("2026-07-03", "2026-07-01"); err == nil {
		t.Fatal("expected reversed date range to fail")
	}
	if _, _, err := parseTrafficDateRange("", "2026-07-01"); err == nil {
		t.Fatal("expected missing start_date to fail")
	}
}
