package service

import (
	"errors"
	"fmt"
	"nfa-dashboard/internal/model"
	"nfa-dashboard/internal/repository"
)

const (
	TrafficRateUnitMbps = "Mbps"
	TrafficRateUnitGbps = "Gbps"
)

var ErrInvalidTrafficSettings = errors.New("invalid traffic settings")

type TrafficSettings struct {
	HideNonSettlementSchoolsInTraffic bool   `json:"hide_non_settlement_schools_in_traffic"`
	TrafficByteUnitBase               int    `json:"traffic_byte_unit_base"`
	SettlementResultUnitBase          int    `json:"settlement_result_unit_base"`
	SettlementDataRateUnit            string `json:"settlement_data_rate_unit"`
	SettlementDailyDetailRateUnit     string `json:"settlement_daily_detail_rate_unit"`
	SettlementSingleUserRateUnit      string `json:"settlement_single_user_rate_unit"`
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
	sanitized := sanitizeTrafficSettings(TrafficSettings{
		HideNonSettlementSchoolsInTraffic: cfg.HideNonSettlementSchoolsInTraffic,
		TrafficByteUnitBase:               cfg.TrafficByteUnitBase,
		SettlementResultUnitBase:          cfg.SettlementResultUnitBase,
		SettlementDataRateUnit:            cfg.SettlementDataRateUnit,
		SettlementDailyDetailRateUnit:     cfg.SettlementDailyDetailRateUnit,
		SettlementSingleUserRateUnit:      cfg.SettlementSingleUserRateUnit,
	})
	return &TrafficSettings{
		HideNonSettlementSchoolsInTraffic: sanitized.HideNonSettlementSchoolsInTraffic,
		TrafficByteUnitBase:               sanitized.TrafficByteUnitBase,
		SettlementResultUnitBase:          sanitized.SettlementResultUnitBase,
		SettlementDataRateUnit:            sanitized.SettlementDataRateUnit,
		SettlementDailyDetailRateUnit:     sanitized.SettlementDailyDetailRateUnit,
		SettlementSingleUserRateUnit:      sanitized.SettlementSingleUserRateUnit,
	}, nil
}

func (s *systemSettingsService) UpdateTrafficSettings(input TrafficSettings) (*TrafficSettings, error) {
	if err := validateTrafficSettings(input); err != nil {
		return nil, err
	}
	sanitized := sanitizeTrafficSettings(input)
	cfg := &model.SystemSettings{
		HideNonSettlementSchoolsInTraffic: sanitized.HideNonSettlementSchoolsInTraffic,
		TrafficByteUnitBase:               sanitized.TrafficByteUnitBase,
		SettlementResultUnitBase:          sanitized.SettlementResultUnitBase,
		SettlementDataRateUnit:            sanitized.SettlementDataRateUnit,
		SettlementDailyDetailRateUnit:     sanitized.SettlementDailyDetailRateUnit,
		SettlementSingleUserRateUnit:      sanitized.SettlementSingleUserRateUnit,
	}
	out, err := s.repo.Upsert(cfg)
	if err != nil {
		return nil, err
	}
	sanitizedOut := sanitizeTrafficSettings(TrafficSettings{
		HideNonSettlementSchoolsInTraffic: out.HideNonSettlementSchoolsInTraffic,
		TrafficByteUnitBase:               out.TrafficByteUnitBase,
		SettlementResultUnitBase:          out.SettlementResultUnitBase,
		SettlementDataRateUnit:            out.SettlementDataRateUnit,
		SettlementDailyDetailRateUnit:     out.SettlementDailyDetailRateUnit,
		SettlementSingleUserRateUnit:      out.SettlementSingleUserRateUnit,
	})
	return &TrafficSettings{
		HideNonSettlementSchoolsInTraffic: sanitizedOut.HideNonSettlementSchoolsInTraffic,
		TrafficByteUnitBase:               sanitizedOut.TrafficByteUnitBase,
		SettlementResultUnitBase:          sanitizedOut.SettlementResultUnitBase,
		SettlementDataRateUnit:            sanitizedOut.SettlementDataRateUnit,
		SettlementDailyDetailRateUnit:     sanitizedOut.SettlementDailyDetailRateUnit,
		SettlementSingleUserRateUnit:      sanitizedOut.SettlementSingleUserRateUnit,
	}, nil
}

func sanitizeTrafficSettings(input TrafficSettings) TrafficSettings {
	out := input
	if out.TrafficByteUnitBase != 1000 && out.TrafficByteUnitBase != 1024 {
		out.TrafficByteUnitBase = 1024
	}
	if out.SettlementResultUnitBase != 1000 && out.SettlementResultUnitBase != 1024 {
		out.SettlementResultUnitBase = 1024
	}
	if out.SettlementDataRateUnit == "" {
		out.SettlementDataRateUnit = TrafficRateUnitMbps
	}
	if out.SettlementDailyDetailRateUnit == "" {
		out.SettlementDailyDetailRateUnit = TrafficRateUnitMbps
	}
	if out.SettlementSingleUserRateUnit == "" {
		out.SettlementSingleUserRateUnit = TrafficRateUnitGbps
	}
	return out
}

func validateTrafficSettings(input TrafficSettings) error {
	if input.TrafficByteUnitBase != 0 && input.TrafficByteUnitBase != 1000 && input.TrafficByteUnitBase != 1024 {
		return fmt.Errorf("%w: traffic_byte_unit_base must be 1000 or 1024", ErrInvalidTrafficSettings)
	}
	if input.SettlementResultUnitBase != 0 && input.SettlementResultUnitBase != 1000 && input.SettlementResultUnitBase != 1024 {
		return fmt.Errorf("%w: settlement_result_unit_base must be 1000 or 1024", ErrInvalidTrafficSettings)
	}
	if input.SettlementDataRateUnit != "" && input.SettlementDataRateUnit != TrafficRateUnitMbps && input.SettlementDataRateUnit != TrafficRateUnitGbps {
		return fmt.Errorf("%w: settlement_data_rate_unit must be Mbps or Gbps", ErrInvalidTrafficSettings)
	}
	if input.SettlementDailyDetailRateUnit != "" && input.SettlementDailyDetailRateUnit != TrafficRateUnitMbps && input.SettlementDailyDetailRateUnit != TrafficRateUnitGbps {
		return fmt.Errorf("%w: settlement_daily_detail_rate_unit must be Mbps or Gbps", ErrInvalidTrafficSettings)
	}
	if input.SettlementSingleUserRateUnit != "" && input.SettlementSingleUserRateUnit != TrafficRateUnitMbps && input.SettlementSingleUserRateUnit != TrafficRateUnitGbps {
		return fmt.Errorf("%w: settlement_single_user_rate_unit must be Mbps or Gbps", ErrInvalidTrafficSettings)
	}
	return nil
}
