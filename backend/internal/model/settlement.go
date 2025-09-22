package model

import (
	"time"
)

// SchoolSettlement 对应nfa_school_settlement表
type SchoolSettlement struct {
	ID              int64     `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	SchoolID        string    `gorm:"column:school_id;not null" json:"school_id"`
	SchoolName      string    `gorm:"column:school_name;not null" json:"school_name"`
	Region          string    `gorm:"column:region;not null" json:"region"`
	CP              string    `gorm:"column:cp;not null" json:"cp"`
	SettlementValue int64     `gorm:"column:settlement_value;not null;default:0" json:"settlement_value"`
	SettlementTime  time.Time `gorm:"column:settlement_time;not null" json:"settlement_time"`
	SettlementDate  time.Time `gorm:"column:settlement_date;not null;type:date" json:"settlement_date"`
	CreateTime      time.Time `gorm:"column:create_time;not null;default:CURRENT_TIMESTAMP" json:"create_time"`
	UpdateTime      time.Time `gorm:"column:update_time;not null;default:CURRENT_TIMESTAMP;autoUpdateTime" json:"update_time"`
}

// TableName 设置表名
func (SchoolSettlement) TableName() string {
	return "nfa_school_settlement"
}

// SettlementFilter 结算数据查询过滤条件
type SettlementFilter struct {
	StartDate  time.Time `form:"start_date"`
	EndDate    time.Time `form:"end_date"`
	SchoolID   string    `form:"school_id"`
	SchoolName string    `form:"school_name"`
	Region     string    `form:"region"`
	CP         string    `form:"cp"`
	Limit      int       `form:"limit,default=100"`
	Offset     int       `form:"offset,default=0"`
	UserID     *uint64   `form:"user_id" json:"user_id"` // v2：按用户可见院校范围过滤（nil/0 表示不启用）
}

// SettlementConfig 结算配置
type SettlementConfig struct {
	ID              int64     `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	DailyTime       string    `gorm:"column:daily_time;not null" json:"daily_time"`         // 每日结算时间，格式为"02:00"
	WeeklyDay       int       `gorm:"column:weekly_day;not null" json:"weekly_day"`         // 每周结算日，1-7表示周一到周日
	WeeklyTime      string    `gorm:"column:weekly_time;not null" json:"weekly_time"`       // 每周结算时间，格式为"02:00"
	Enabled         bool      `gorm:"column:enabled;not null;default:true" json:"enabled"`  // 是否启用
	LastExecuteTime time.Time `gorm:"column:last_execute_time" json:"last_execute_time"`    // 上次执行时间
	UpdateTime      time.Time `gorm:"column:update_time;autoUpdateTime" json:"update_time"` // 更新时间
}

// TableName 设置表名
func (SettlementConfig) TableName() string {
	return "nfa_settlement_config"
}

// SettlementTask 结算任务记录
type SettlementTask struct {
	ID             int64     `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	TaskType       string    `gorm:"column:task_type;not null" json:"task_type"`              // 任务计算周期：daily(每日计算前一天)、weekly(每周计算前一周每天)
	TaskDate       time.Time `gorm:"column:task_date;not null;type:date" json:"task_date"`    // 任务日期
	Status         string    `gorm:"column:status;not null" json:"status"`                    // 状态：pending、running、success、failed
	StartTime      *time.Time `gorm:"column:start_time" json:"start_time"`                    // 开始时间
	EndTime        *time.Time `gorm:"column:end_time" json:"end_time"`                        // 结束时间
	ProcessedCount int       `gorm:"column:processed_count;default:0" json:"processed_count"` // 处理记录数
	ErrorMessage   string    `gorm:"column:error_message" json:"error_message"`               // 错误信息
	CreateTime     time.Time `gorm:"column:create_time;not null;default:CURRENT_TIMESTAMP" json:"create_time"`
	UpdateTime     time.Time `gorm:"column:update_time;not null;default:CURRENT_TIMESTAMP;autoUpdateTime" json:"update_time"`
}

// TableName 设置表名
func (SettlementTask) TableName() string {
	return "nfa_settlement_task"
}

// SettlementTaskResponse 结算任务响应结构
type SettlementTaskResponse struct {
	ID             int64     `json:"id"`
	TaskType       string    `json:"task_type"`
	TaskDate       time.Time `json:"task_date"`
	Status         string    `json:"status"`
	StartTime      time.Time `json:"start_time"`
	EndTime        time.Time `json:"end_time"`
	ProcessedCount int       `json:"processed_count"`
	ErrorMessage   string    `json:"error_message"`
	CreateTime     time.Time `json:"create_time"`
	UpdateTime     time.Time `json:"update_time"`
}

// SettlementResponse 结算数据响应结构
type SettlementResponse struct {
	ID              int64     `json:"id"`
	SchoolID        string    `json:"school_id"`
	SchoolName      string    `json:"school_name"`
	Region          string    `json:"region"`
	CP              string    `json:"cp"`
	SettlementValue int64     `json:"settlement_value"`
	SettlementTime  time.Time `json:"settlement_time"`
	SettlementDate  time.Time `json:"settlement_date"`
	CreateTime      time.Time `json:"create_time"`
}

// DailySettlementDetail 对应日95明细数据，可能来自 nfa_school_settlement 或类似表
// 假设它与 SchoolSettlement 结构相似，但代表单日数据
type DailySettlementDetail struct {
	ID             int64     `gorm:"column:id;primaryKey;autoIncrement" json:"id,omitempty"`
	DailyDate      time.Time `gorm:"column:settlement_date;not null;type:date" json:"daily_date"`
	SchoolID       string    `gorm:"column:school_id;not null" json:"school_id"`
	SchoolName     string    `gorm:"column:school_name;not null" json:"school_name"`
	Region         string    `gorm:"column:region;not null" json:"region"`
	CP             string    `gorm:"column:cp;not null" json:"cp"`
	Daily95Value   int64     `gorm:"column:settlement_value;not null;default:0" json:"daily_95_value"` // 对应原始的 settlement_value
	CreateTime     time.Time `gorm:"column:create_time;not null;default:CURRENT_TIMESTAMP" json:"create_time,omitempty"`
	// UpdateTime  time.Time `gorm:"column:update_time;not null;default:CURRENT_TIMESTAMP;autoUpdateTime" json:"update_time,omitempty"` // 可选
}

// TableName 方法可以根据实际情况决定是否需要，如果 DailySettlementDetail 直接映射 nfa_school_settlement，则可以不单独定义
// func (DailySettlementDetail) TableName() string {
// 	 return "nfa_school_settlement" // 或者其他表名
// }

