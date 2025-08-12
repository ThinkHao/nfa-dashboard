package model

import "time"

// Permission represents a granular permission
// Table: permissions
type Permission struct {
	ID          uint64    `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	Code        string    `gorm:"column:code;size:128;uniqueIndex;not null" json:"code"`
	Name        string    `gorm:"column:name;size:128;not null" json:"name"`
	Description *string   `gorm:"column:description;size:255" json:"description,omitempty"`
	Enabled     bool      `gorm:"column:enabled;not null;default:true" json:"enabled"`
	CreatedAt   time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt   time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

func (Permission) TableName() string { return "permissions" }
