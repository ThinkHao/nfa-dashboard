package model

import "time"

// RolePermission is the association between roles and permissions
// Table: role_permissions
type RolePermission struct {
	ID           uint64    `gorm:"column:id;primaryKey;autoIncrement"`
	RoleID       uint64    `gorm:"column:role_id;index;not null"`
	PermissionID uint64    `gorm:"column:permission_id;index;not null"`
	CreatedAt    time.Time `gorm:"column:created_at;autoCreateTime"`
}

func (RolePermission) TableName() string { return "role_permissions" }
