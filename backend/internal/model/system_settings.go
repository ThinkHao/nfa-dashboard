package model

import "time"

// SystemSettings stores global feature toggles used by system pages and runtime behavior.
type SystemSettings struct {
	ID                                int64     `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	HideNonSettlementSchoolsInTraffic bool      `gorm:"column:hide_non_settlement_schools_in_traffic;not null;default:0" json:"hide_non_settlement_schools_in_traffic"`
	TrafficByteUnitBase               int       `gorm:"column:traffic_byte_unit_base;not null;default:1024" json:"traffic_byte_unit_base"`
	SettlementResultUnitBase          int       `gorm:"column:settlement_result_unit_base;not null;default:1024" json:"settlement_result_unit_base"`
	SettlementDataRateUnit            string    `gorm:"column:settlement_data_rate_unit;type:varchar(16);not null;default:'Mbps'" json:"settlement_data_rate_unit"`
	SettlementDailyDetailRateUnit     string    `gorm:"column:settlement_daily_detail_rate_unit;type:varchar(16);not null;default:'Mbps'" json:"settlement_daily_detail_rate_unit"`
	SettlementSingleUserRateUnit      string    `gorm:"column:settlement_single_user_rate_unit;type:varchar(16);not null;default:'Gbps'" json:"settlement_single_user_rate_unit"`
	CreatedAt                         time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt                         time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

func (SystemSettings) TableName() string { return "nfa_system_settings" }
