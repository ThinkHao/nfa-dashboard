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
	if err := model.DB.Model(&model.SystemSettings{}).
		Where("id = ?", current.ID).
		Update("hide_non_settlement_schools_in_traffic", current.HideNonSettlementSchoolsInTraffic).Error; err != nil {
		return nil, err
	}
	return current, nil
}
