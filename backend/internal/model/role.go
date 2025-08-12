package model

import "time"

// Role represents a role in the system
// Table: roles
type Role struct {
	ID          uint64    `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	Name        string    `gorm:"column:name;size:64;uniqueIndex;not null" json:"name"`
	Description *string   `gorm:"column:description;size:255" json:"description,omitempty"`
	CreatedAt   time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt   time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

func (Role) TableName() string { return "roles" }
