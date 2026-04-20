package controller

import (
	"testing"
	"time"
)

func TestParseDateBoundary_DateOnlyStartUsesStartOfDay(t *testing.T) {
	got, err := parseDateBoundary("2026-02-01", false)
	if err != nil {
		t.Fatalf("parseDateBoundary returned error: %v", err)
	}
	if got.Format("2006-01-02 15:04:05") != "2026-02-01 00:00:00" {
		t.Fatalf("unexpected start boundary: %s", got.Format("2006-01-02 15:04:05"))
	}
}

func TestParseDateBoundary_DateOnlyEndUsesEndOfDay(t *testing.T) {
	got, err := parseDateBoundary("2026-02-28", true)
	if err != nil {
		t.Fatalf("parseDateBoundary returned error: %v", err)
	}
	if got.Format("2006-01-02 15:04:05") != "2026-02-28 23:59:59" {
		t.Fatalf("unexpected end boundary: %s", got.Format("2006-01-02 15:04:05"))
	}
}

func TestParseDateBoundary_DateTimeKeepsExplicitClock(t *testing.T) {
	got, err := parseDateBoundary("2026-02-28 12:34:56", true)
	if err != nil {
		t.Fatalf("parseDateBoundary returned error: %v", err)
	}
	if got.Format("2006-01-02 15:04:05") != "2026-02-28 12:34:56" {
		t.Fatalf("unexpected datetime boundary: %s", got.Format("2006-01-02 15:04:05"))
	}
}

func TestParseDateBoundary_UsesLocalLocation(t *testing.T) {
	got, err := parseDateBoundary("2026-02-28 12:34:56", true)
	if err != nil {
		t.Fatalf("parseDateBoundary returned error: %v", err)
	}
	if got.Location().String() != time.Local.String() {
		t.Fatalf("expected local location, got=%s want=%s", got.Location().String(), time.Local.String())
	}
}
