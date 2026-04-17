package model

import "time"

// SystemSettings stores global feature toggles used by system pages and runtime behavior.
type SystemSettings struct {
	ID                                int64     `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	HideNonSettlementSchoolsInTraffic bool      `gorm:"column:hide_non_settlement_schools_in_traffic;not null;default:0" json:"hide_non_settlement_schools_in_traffic"`
	CreatedAt                         time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt                         time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

func (SystemSettings) TableName() string { return "nfa_system_settings" }
