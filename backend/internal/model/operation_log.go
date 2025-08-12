package model

import "time"

// OperationLog records auditable operations
// Table: operation_logs
type OperationLog struct {
	ID           uint64     `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	UserID       *uint64    `gorm:"column:user_id;index" json:"user_id"`
	Method       string     `gorm:"column:method;size:16;not null" json:"method"`
	Path         string     `gorm:"column:path;size:255;not null" json:"path"`
	Resource     *string    `gorm:"column:resource;size:128" json:"resource,omitempty"`
	Action       *string    `gorm:"column:action;size:64" json:"action,omitempty"`
	StatusCode   int        `gorm:"column:status_code;not null" json:"status_code"`
	Success      int8       `gorm:"column:success;not null;default:1" json:"success"`
	LatencyMS    *int       `gorm:"column:latency_ms" json:"latency_ms,omitempty"`
	IP           *string    `gorm:"column:ip;size:64" json:"ip,omitempty"`
	UserAgent    *string    `gorm:"column:user_agent;size:255" json:"user_agent,omitempty"`
	ErrorMessage *string    `gorm:"column:error_message;size:512" json:"error_message,omitempty"`
	CreatedAt    time.Time  `gorm:"column:created_at;autoCreateTime" json:"created_at"`
}

func (OperationLog) TableName() string { return "operation_logs" }
