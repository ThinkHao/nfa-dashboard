package service

import (
	"testing"

	"nfa-dashboard/internal/model"
)

type systemSettingsRepoStub struct {
	hide bool
	cfg  TrafficSettings
}

func (s *systemSettingsRepoStub) Get() (*model.SystemSettings, error) {
	return &model.SystemSettings{
		HideNonSettlementSchoolsInTraffic: s.hide,
		TrafficByteUnitBase:               s.cfg.TrafficByteUnitBase,
		SettlementResultUnitBase:          s.cfg.SettlementResultUnitBase,
		SettlementDataRateUnit:            s.cfg.SettlementDataRateUnit,
		SettlementDailyDetailRateUnit:     s.cfg.SettlementDailyDetailRateUnit,
		SettlementSingleUserRateUnit:      s.cfg.SettlementSingleUserRateUnit,
	}, nil
}

func (s *systemSettingsRepoStub) Upsert(cfg *model.SystemSettings) (*model.SystemSettings, error) {
	if cfg != nil {
		s.hide = cfg.HideNonSettlementSchoolsInTraffic
		s.cfg = TrafficSettings{
			HideNonSettlementSchoolsInTraffic: cfg.HideNonSettlementSchoolsInTraffic,
			TrafficByteUnitBase:               cfg.TrafficByteUnitBase,
			SettlementResultUnitBase:          cfg.SettlementResultUnitBase,
			SettlementDataRateUnit:            cfg.SettlementDataRateUnit,
			SettlementDailyDetailRateUnit:     cfg.SettlementDailyDetailRateUnit,
			SettlementSingleUserRateUnit:      cfg.SettlementSingleUserRateUnit,
		}
	}
	return &model.SystemSettings{
		HideNonSettlementSchoolsInTraffic: s.hide,
		TrafficByteUnitBase:               s.cfg.TrafficByteUnitBase,
		SettlementResultUnitBase:          s.cfg.SettlementResultUnitBase,
		SettlementDataRateUnit:            s.cfg.SettlementDataRateUnit,
		SettlementDailyDetailRateUnit:     s.cfg.SettlementDailyDetailRateUnit,
		SettlementSingleUserRateUnit:      s.cfg.SettlementSingleUserRateUnit,
	}, nil
}

func TestSystemSettingsService_UpdateTrafficSettings(t *testing.T) {
	repo := &systemSettingsRepoStub{
		cfg: TrafficSettings{
			TrafficByteUnitBase:           1024,
			SettlementResultUnitBase:      1024,
			SettlementDataRateUnit:        TrafficRateUnitMbps,
			SettlementDailyDetailRateUnit: TrafficRateUnitMbps,
			SettlementSingleUserRateUnit:  TrafficRateUnitGbps,
		},
	}
	svc := NewSystemSettingsService(repo)

	cfg, err := svc.GetTrafficSettings()
	if err != nil {
		t.Fatalf("GetTrafficSettings() error = %v", err)
	}
	if cfg.HideNonSettlementSchoolsInTraffic {
		t.Fatalf("expected default false, got true")
	}
	if cfg.TrafficByteUnitBase != 1024 || cfg.SettlementResultUnitBase != 1024 {
		t.Fatalf("unexpected default unit base: %+v", cfg)
	}
	if cfg.SettlementSingleUserRateUnit != TrafficRateUnitGbps {
		t.Fatalf("expected default single-user unit Gbps, got %s", cfg.SettlementSingleUserRateUnit)
	}

	updated, err := svc.UpdateTrafficSettings(TrafficSettings{
		HideNonSettlementSchoolsInTraffic: true,
		TrafficByteUnitBase:               1000,
		SettlementResultUnitBase:          1000,
		SettlementDataRateUnit:            TrafficRateUnitGbps,
		SettlementDailyDetailRateUnit:     TrafficRateUnitGbps,
		SettlementSingleUserRateUnit:      TrafficRateUnitMbps,
	})
	if err != nil {
		t.Fatalf("UpdateTrafficSettings() error = %v", err)
	}
	if !updated.HideNonSettlementSchoolsInTraffic {
		t.Fatalf("expected updated true, got false")
	}
	if updated.TrafficByteUnitBase != 1000 || updated.SettlementResultUnitBase != 1000 {
		t.Fatalf("expected updated bases = 1000, got %+v", updated)
	}
	if updated.SettlementDataRateUnit != TrafficRateUnitGbps || updated.SettlementDailyDetailRateUnit != TrafficRateUnitGbps {
		t.Fatalf("expected updated rate unit Gbps, got %+v", updated)
	}
	if updated.SettlementSingleUserRateUnit != TrafficRateUnitMbps {
		t.Fatalf("expected updated single-user rate unit Mbps, got %+v", updated)
	}

	cfg, err = svc.GetTrafficSettings()
	if err != nil {
		t.Fatalf("GetTrafficSettings() after update error = %v", err)
	}
	if !cfg.HideNonSettlementSchoolsInTraffic {
		t.Fatalf("expected persisted true, got false")
	}
	if cfg.TrafficByteUnitBase != 1000 || cfg.SettlementResultUnitBase != 1000 {
		t.Fatalf("expected persisted bases = 1000, got %+v", cfg)
	}
}

func TestSystemSettingsService_UpdateTrafficSettings_InvalidInput(t *testing.T) {
	repo := &systemSettingsRepoStub{
		cfg: TrafficSettings{
			TrafficByteUnitBase:           1024,
			SettlementResultUnitBase:      1024,
			SettlementDataRateUnit:        TrafficRateUnitMbps,
			SettlementDailyDetailRateUnit: TrafficRateUnitMbps,
			SettlementSingleUserRateUnit:  TrafficRateUnitGbps,
		},
	}
	svc := NewSystemSettingsService(repo)

	_, err := svc.UpdateTrafficSettings(TrafficSettings{
		TrafficByteUnitBase:           999,
		SettlementResultUnitBase:      1024,
		SettlementDataRateUnit:        TrafficRateUnitMbps,
		SettlementDailyDetailRateUnit: TrafficRateUnitMbps,
		SettlementSingleUserRateUnit:  TrafficRateUnitGbps,
	})
	if err == nil {
		t.Fatalf("expected invalid unit base error")
	}
	if err != nil && err.Error() == "" {
		t.Fatalf("expected non-empty error")
	}
}

func TestSystemSettingsService_UpdateTrafficSettings_DefaultFallbackForMissingFields(t *testing.T) {
	repo := &systemSettingsRepoStub{
		cfg: TrafficSettings{
			TrafficByteUnitBase:           1024,
			SettlementResultUnitBase:      1024,
			SettlementDataRateUnit:        TrafficRateUnitMbps,
			SettlementDailyDetailRateUnit: TrafficRateUnitMbps,
			SettlementSingleUserRateUnit:  TrafficRateUnitGbps,
		},
	}
	svc := NewSystemSettingsService(repo)

	updated, err := svc.UpdateTrafficSettings(TrafficSettings{
		HideNonSettlementSchoolsInTraffic: true,
	})
	if err != nil {
		t.Fatalf("UpdateTrafficSettings() error = %v", err)
	}
	if updated.TrafficByteUnitBase != 1024 || updated.SettlementResultUnitBase != 1024 {
		t.Fatalf("expected fallback base to 1024, got %+v", updated)
	}
	if updated.SettlementDataRateUnit != TrafficRateUnitMbps || updated.SettlementDailyDetailRateUnit != TrafficRateUnitMbps {
		t.Fatalf("expected fallback rate units to Mbps, got %+v", updated)
	}
	if updated.SettlementSingleUserRateUnit != TrafficRateUnitGbps {
		t.Fatalf("expected fallback single user rate unit to Gbps, got %+v", updated)
	}
}
