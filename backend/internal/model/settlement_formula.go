package model

import (
	"time"
)

// SettlementFormula 映射 nfa_settlement_formulas 表
// tokens 以 JSON 字符串存储（前端传入/返回为 JSON 数组）
type SettlementFormula struct {
	ID          uint64    `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	Name        string    `gorm:"column:name;type:varchar(64);not null" json:"name"`
	Description string    `gorm:"column:description;type:varchar(255)" json:"description"`
	Tokens      string    `gorm:"column:tokens;type:json;not null" json:"tokens"`
	Enabled     bool      `gorm:"column:enabled;not null;default:true" json:"enabled"`
	UpdatedBy   string    `gorm:"column:updated_by;type:varchar(64)" json:"updated_by"`
	CreateTime  time.Time `gorm:"column:create_time;autoCreateTime" json:"create_time"`
	UpdateTime  time.Time `gorm:"column:update_time;autoUpdateTime" json:"update_time"`
}

func (SettlementFormula) TableName() string { return "nfa_settlement_formulas" }
