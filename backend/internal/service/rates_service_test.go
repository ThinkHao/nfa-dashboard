package service

import (
	"testing"
	"time"

	"nfa-dashboard/internal/model"
)

func TestNormalizeIncrementConfig_NoIncrementStartForcesDefault(t *testing.T) {
	stock := 0.3
	increment := 0.4
	rate := &model.RateCustomer{
		StockRatio:     &stock,
		IncrementRatio: &increment,
	}

	if err := normalizeIncrementConfig(rate); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if rate.StockRatio == nil || *rate.StockRatio != 1 {
		t.Fatalf("expected stock_ratio=1, got %+v", rate.StockRatio)
	}
	if rate.IncrementRatio == nil || *rate.IncrementRatio != 0 {
		t.Fatalf("expected increment_ratio=0, got %+v", rate.IncrementRatio)
	}
}

func TestNormalizeIncrementConfig_IncrementStartAllowsIndependentRatios(t *testing.T) {
	now := time.Now()
	stock := 0.9
	increment := 0.9
	rate := &model.RateCustomer{
		IncrementStartAt: &now,
		StockRatio:       &stock,
		IncrementRatio:   &increment,
	}

	if err := normalizeIncrementConfig(rate); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestNormalizeIncrementConfig_IncrementStartOneSideMissingAutoFillZero(t *testing.T) {
	now := time.Now()
	stock := 0.3
	rate := &model.RateCustomer{
		IncrementStartAt: &now,
		StockRatio:       &stock,
	}

	if err := normalizeIncrementConfig(rate); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if rate.IncrementRatio == nil || *rate.IncrementRatio != 0 {
		t.Fatalf("expected increment_ratio=0, got %+v", rate.IncrementRatio)
	}
}

func TestNormalizeIncrementConfig_IncrementStartBothMissingShouldFail(t *testing.T) {
	now := time.Now()
	rate := &model.RateCustomer{
		IncrementStartAt: &now,
	}

	if err := normalizeIncrementConfig(rate); err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestNormalizeIncrementConfig_RatioOutOfRangeShouldFail(t *testing.T) {
	now := time.Now()
	stock := 1.1
	increment := 0.2
	rate := &model.RateCustomer{
		IncrementStartAt: &now,
		StockRatio:       &stock,
		IncrementRatio:   &increment,
	}

	if err := normalizeIncrementConfig(rate); err == nil {
		t.Fatalf("expected error, got nil")
	}
}

