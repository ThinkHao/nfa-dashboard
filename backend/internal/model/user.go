package model

import "time"

// User represents a system user for authentication and authorization
// Table: users
type User struct {
	ID           uint64     `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	Username     string     `gorm:"column:username;uniqueIndex;size:64;not null" json:"username"`
	PasswordHash string     `gorm:"column:password_hash;size:255;not null" json:"-"`
	Email        *string    `gorm:"column:email;size:128" json:"email,omitempty"`
	Phone        *string    `gorm:"column:phone;size:32" json:"phone,omitempty"`
	Status       int8       `gorm:"column:status;not null;default:1" json:"status"`
	LastLoginAt  *time.Time `gorm:"column:last_login_at" json:"last_login_at,omitempty"`
	CreatedAt    time.Time  `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt    time.Time  `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

func (User) TableName() string { return "users" }
