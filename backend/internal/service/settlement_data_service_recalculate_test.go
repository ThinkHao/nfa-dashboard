package service

import (
	"testing"
	"time"

	"nfa-dashboard/internal/repository"
)

type recalculateSettlementRepoStub struct {
	repository.SettlementRepository
	calls *[]string
}

func (s *recalculateSettlementRepoStub) RecalculateDaily95Range(region, cp, school string, start, end time.Time) (int64, error) {
	*s.calls = append(*s.calls, "source")
	return 2, nil
}

type recalculateDataRepoStub struct {
	repository.SettlementDataRepository
	calls *[]string
}

func (s *recalculateDataRepoStub) BackfillFromSchoolSettlement(region, cp, school string, start, end time.Time, markRecalc bool, progress func(int64, map[string]int64)) (int64, error) {
	*s.calls = append(*s.calls, "backfill")
	if !markRecalc {
		panic("recalculate backfill must set markRecalc")
	}
	return 2, nil
}

func TestRecalculateRebuildsDaily95BeforeCustomerBackfill(t *testing.T) {
	calls := []string{}
	dataRepo := &recalculateDataRepoStub{calls: &calls}
	settlementRepo := &recalculateSettlementRepoStub{calls: &calls}
	svc := NewSettlementDataService(dataRepo, nil, nil, settlementRepo)
	start := time.Date(2026, 7, 28, 0, 0, 0, 0, time.Local)
	end := time.Date(2026, 7, 29, 23, 59, 59, 0, time.Local)

	affected, err := svc.RecalculateWithProgress(SettlementCustomerFilter{
		Region: "天津市", CP: "ali", School: "天津大学城", Start: &start, End: &end,
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if affected != 2 {
		t.Fatalf("affected=%d, want 2", affected)
	}
	if len(calls) != 2 || calls[0] != "source" || calls[1] != "backfill" {
		t.Fatalf("calls=%v, want [source backfill]", calls)
	}
}
