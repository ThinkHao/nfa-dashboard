package controller

import (
	"testing"
	"time"
)

func TestResolveDailySettlementDate_WithExplicitDate_UsesLocalMidnight(t *testing.T) {
	got, err := resolveDailySettlementDate("2026-03-28", time.Now())
	if err != nil {
		t.Fatalf("resolveDailySettlementDate returned error: %v", err)
	}
	if got.Format("2006-01-02 15:04:05") != "2026-03-28 00:00:00" {
		t.Fatalf("unexpected datetime: %s", got.Format("2006-01-02 15:04:05"))
	}
	if got.Location().String() != time.Local.String() {
		t.Fatalf("unexpected location: got=%s want=%s", got.Location().String(), time.Local.String())
	}
}

func TestResolveDailySettlementDate_WithoutDate_UsesYesterdayLocalMidnight(t *testing.T) {
	loc := time.Local
	now := time.Date(2026, 4, 21, 10, 11, 12, 0, loc)
	got, err := resolveDailySettlementDate("", now)
	if err != nil {
		t.Fatalf("resolveDailySettlementDate returned error: %v", err)
	}
	if got.Format("2006-01-02 15:04:05") != "2026-04-20 00:00:00" {
		t.Fatalf("unexpected datetime: %s", got.Format("2006-01-02 15:04:05"))
	}
}

func TestResolveDailySettlementDate_InvalidDate_ReturnsError(t *testing.T) {
	if _, err := resolveDailySettlementDate("2026/03/28", time.Now()); err == nil {
		t.Fatalf("expected error for invalid date format")
	}
}
