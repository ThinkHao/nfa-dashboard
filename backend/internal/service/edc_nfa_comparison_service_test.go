package service

import (
	"context"
	"testing"
	"time"

	"nfa-dashboard/internal/model"
)

type comparisonRepoStub struct{}

func (comparisonRepoStub) ListGroups(context.Context, []uint64) ([]model.EDCNFAComparisonGroup, error) {
	return nil, nil
}

func (comparisonRepoStub) GetComparison(context.Context, model.EDCNFAComparisonFilter) ([]model.EDCNFAComparisonPoint, error) {
	return []model.EDCNFAComparisonPoint{}, nil
}

func TestEDCNFAComparisonServiceAccepts366DayWindow(t *testing.T) {
	svc := NewEDCNFAComparisonService(comparisonRepoStub{})
	points, err := svc.GetComparison(context.Background(), model.EDCNFAComparisonFilter{
		GroupID:   1,
		StartTime: time.Date(2026, 1, 1, 0, 0, 0, 0, time.Local),
		EndTime:   time.Date(2027, 1, 2, 0, 0, 0, 0, time.Local),
	})
	if err != nil || points == nil {
		t.Fatalf("error=%v points=%v, want accepted 366-day window", err, points)
	}
}

func TestEDCNFAComparisonServiceRejectsOver366DayWindow(t *testing.T) {
	svc := NewEDCNFAComparisonService(comparisonRepoStub{})
	_, err := svc.GetComparison(context.Background(), model.EDCNFAComparisonFilter{
		GroupID:   1,
		StartTime: time.Date(2026, 1, 1, 0, 0, 0, 0, time.Local),
		EndTime:   time.Date(2027, 1, 3, 0, 0, 0, 0, time.Local),
	})
	if !IsBadRequest(err) {
		t.Fatalf("error=%v, want bad request", err)
	}
}

func TestEDCNFAComparisonServiceDefaultsEmptyWindow(t *testing.T) {
	svc := NewEDCNFAComparisonService(comparisonRepoStub{})
	points, err := svc.GetComparison(context.Background(), model.EDCNFAComparisonFilter{GroupID: 1})
	if err != nil {
		t.Fatalf("GetComparison() error=%v", err)
	}
	if points == nil {
		t.Fatalf("points=nil, want empty slice")
	}
}
