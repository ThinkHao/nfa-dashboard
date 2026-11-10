package model

import (
	"time"

	"gorm.io/datatypes"
)

const (
	TodoTypeCustomerRatePrices = "customer_rate_prices"
	TodoStatusPending          = "pending"
	TodoStatusCompleted        = "completed"
)

type Todo struct {
	ID          uint64     `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	TodoType    string     `gorm:"column:todo_type;size:64;not null" json:"todo_type"`
	TargetID    uint64     `gorm:"column:target_id;not null" json:"target_id"`
	Status      string     `gorm:"column:status;size:16;not null;default:pending" json:"status"`
	CompletedAt *time.Time `gorm:"column:completed_at" json:"completed_at,omitempty"`
	CreatedAt   time.Time  `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt   time.Time  `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

func (Todo) TableName() string { return "nfa_todos" }

type TodoAssignee struct {
	ID        uint64     `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	TodoID    uint64     `gorm:"column:todo_id;not null" json:"todo_id"`
	UserID    uint64     `gorm:"column:user_id;not null" json:"user_id"`
	Ignored   bool       `gorm:"column:ignored;not null;default:0" json:"ignored"`
	IgnoredAt *time.Time `gorm:"column:ignored_at" json:"ignored_at,omitempty"`
	CreatedAt time.Time  `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt time.Time  `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

func (TodoAssignee) TableName() string { return "nfa_todo_assignees" }

type TodoEvent struct {
	ID          uint64    `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	TodoID      uint64    `gorm:"column:todo_id;not null" json:"todo_id"`
	ActorUserID *uint64   `gorm:"column:actor_user_id" json:"actor_user_id,omitempty"`
	EventType   string    `gorm:"column:event_type;size:32;not null" json:"event_type"`
	OldStatus   *string   `gorm:"column:old_status;size:16" json:"old_status,omitempty"`
	NewStatus   *string   `gorm:"column:new_status;size:16" json:"new_status,omitempty"`
	Detail      *string   `gorm:"column:detail;size:255" json:"detail,omitempty"`
	CreatedAt   time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
}

func (TodoEvent) TableName() string { return "nfa_todo_events" }

type TodoEventListItem struct {
	ID          uint64    `json:"id"`
	ActorUserID *uint64   `json:"actor_user_id,omitempty"`
	ActorName   string    `json:"actor_name"`
	EventType   string    `json:"event_type"`
	OldStatus   *string   `json:"old_status,omitempty"`
	NewStatus   *string   `json:"new_status,omitempty"`
	Detail      *string   `json:"detail,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

type TodoListItem struct {
	ID            uint64    `json:"id"`
	Status        string    `json:"status"`
	Region        string    `json:"region"`
	CP            string    `json:"cp"`
	SchoolName    string    `json:"school_name"`
	MissingFields []string  `json:"missing_fields"`
	Ignored       bool      `json:"ignored"`
	CreatedAt     time.Time `json:"created_at"`
}

type SystemTodoSettings struct {
	ID                  int64          `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	CustomerRateUserIDs datatypes.JSON `gorm:"column:customer_rate_user_ids;type:json" json:"-"`
	CreatedAt           time.Time      `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt           time.Time      `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

func (SystemTodoSettings) TableName() string { return "nfa_todo_settings" }

type TodoSettingsResponse struct {
	CustomerRateUserIDs []uint64 `json:"customer_rate_user_ids"`
}
