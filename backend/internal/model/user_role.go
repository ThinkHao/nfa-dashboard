package model

import "time"

// UserRole is the association between users and roles
// Table: user_roles
type UserRole struct {
	ID        uint64    `gorm:"column:id;primaryKey;autoIncrement"`
	UserID    uint64    `gorm:"column:user_id;index;not null"`
	RoleID    uint64    `gorm:"column:role_id;index;not null"`
	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime"`
}

func (UserRole) TableName() string { return "user_roles" }
