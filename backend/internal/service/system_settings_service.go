package service

import (
	"nfa-dashboard/internal/model"
	"nfa-dashboard/internal/repository"
)

type TrafficSettings struct {
	HideNonSettlementSchoolsInTraffic bool `json:"hide_non_settlement_schools_in_traffic"`
}

type SystemSettingsService interface {
	GetTrafficSettings() (*TrafficSettings, error)
	UpdateTrafficSettings(cfg TrafficSettings) (*TrafficSettings, error)
}

type systemSettingsService struct {
	repo repository.SystemSettingsRepository
}

func NewSystemSettingsService(repo repository.SystemSettingsRepository) SystemSettingsService {
	return &systemSettingsService{repo: repo}
}

func (s *systemSettingsService) GetTrafficSettings() (*TrafficSettings, error) {
	cfg, err := s.repo.Get()
	if err != nil {
		return nil, err
	}
	return &TrafficSettings{
		HideNonSettlementSchoolsInTraffic: cfg.HideNonSettlementSchoolsInTraffic,
	}, nil
}

func (s *systemSettingsService) UpdateTrafficSettings(input TrafficSettings) (*TrafficSettings, error) {
	cfg := &model.SystemSettings{
		HideNonSettlementSchoolsInTraffic: input.HideNonSettlementSchoolsInTraffic,
	}
	out, err := s.repo.Upsert(cfg)
	if err != nil {
		return nil, err
	}
	return &TrafficSettings{
		HideNonSettlementSchoolsInTraffic: out.HideNonSettlementSchoolsInTraffic,
	}, nil
}
