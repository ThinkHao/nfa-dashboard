package repository

import (
	"errors"

	"gorm.io/gorm"
	"nfa-dashboard/internal/model"
)

type SystemSettingsRepository interface {
	Get() (*model.SystemSettings, error)
	Upsert(settings *model.SystemSettings) (*model.SystemSettings, error)
}

type systemSettingsRepository struct{}

func NewSystemSettingsRepository() SystemSettingsRepository { return &systemSettingsRepository{} }

func (r *systemSettingsRepository) Get() (*model.SystemSettings, error) {
	var cfg model.SystemSettings
	err := model.DB.First(&cfg).Error
	if err == nil {
		return &cfg, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	cfg = model.SystemSettings{
		HideNonSettlementSchoolsInTraffic: false,
		TrafficByteUnitBase:               1024,
		SettlementResultUnitBase:          1024,
		SettlementDataRateUnit:            "Mbps",
		SettlementDailyDetailRateUnit:     "Mbps",
		SettlementSingleUserRateUnit:      "Gbps",
	}
	if createErr := model.DB.Create(&cfg).Error; createErr != nil {
		return nil, createErr
	}
	return &cfg, nil
}

func (r *systemSettingsRepository) Upsert(settings *model.SystemSettings) (*model.SystemSettings, error) {
	if settings == nil {
		settings = &model.SystemSettings{}
	}

	current, err := r.Get()
	if err != nil {
		return nil, err
	}

	current.HideNonSettlementSchoolsInTraffic = settings.HideNonSettlementSchoolsInTraffic
	current.TrafficByteUnitBase = settings.TrafficByteUnitBase
	current.SettlementResultUnitBase = settings.SettlementResultUnitBase
	current.SettlementDataRateUnit = settings.SettlementDataRateUnit
	current.SettlementDailyDetailRateUnit = settings.SettlementDailyDetailRateUnit
	current.SettlementSingleUserRateUnit = settings.SettlementSingleUserRateUnit
	if err := model.DB.Model(&model.SystemSettings{}).
		Where("id = ?", current.ID).
		Updates(map[string]any{
			"hide_non_settlement_schools_in_traffic": current.HideNonSettlementSchoolsInTraffic,
			"traffic_byte_unit_base":                 current.TrafficByteUnitBase,
			"settlement_result_unit_base":            current.SettlementResultUnitBase,
			"settlement_data_rate_unit":              current.SettlementDataRateUnit,
			"settlement_daily_detail_rate_unit":      current.SettlementDailyDetailRateUnit,
			"settlement_single_user_rate_unit":       current.SettlementSingleUserRateUnit,
		}).Error; err != nil {
		return nil, err
	}
	return current, nil
}
