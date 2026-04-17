package service

import (
	"testing"

	"nfa-dashboard/internal/model"
)

type systemSettingsRepoStub struct {
	hide bool
}

func (s *systemSettingsRepoStub) Get() (*model.SystemSettings, error) {
	return &model.SystemSettings{HideNonSettlementSchoolsInTraffic: s.hide}, nil
}

func (s *systemSettingsRepoStub) Upsert(cfg *model.SystemSettings) (*model.SystemSettings, error) {
	if cfg != nil {
		s.hide = cfg.HideNonSettlementSchoolsInTraffic
	}
	return &model.SystemSettings{HideNonSettlementSchoolsInTraffic: s.hide}, nil
}

func TestSystemSettingsService_UpdateTrafficSettings(t *testing.T) {
	repo := &systemSettingsRepoStub{}
	svc := NewSystemSettingsService(repo)

	cfg, err := svc.GetTrafficSettings()
	if err != nil {
		t.Fatalf("GetTrafficSettings() error = %v", err)
	}
	if cfg.HideNonSettlementSchoolsInTraffic {
		t.Fatalf("expected default false, got true")
	}

	updated, err := svc.UpdateTrafficSettings(TrafficSettings{HideNonSettlementSchoolsInTraffic: true})
	if err != nil {
		t.Fatalf("UpdateTrafficSettings() error = %v", err)
	}
	if !updated.HideNonSettlementSchoolsInTraffic {
		t.Fatalf("expected updated true, got false")
	}

	cfg, err = svc.GetTrafficSettings()
	if err != nil {
		t.Fatalf("GetTrafficSettings() after update error = %v", err)
	}
	if !cfg.HideNonSettlementSchoolsInTraffic {
		t.Fatalf("expected persisted true, got false")
	}
}
