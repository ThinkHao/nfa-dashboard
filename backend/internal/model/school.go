package model

import (
	"time"
)

// School 对应nfa_school表
type School struct {
	ID              int64     `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	SchoolID        string    `gorm:"column:school_id;not null" json:"school_id"`
	SchoolName      string    `gorm:"column:school_name;not null" json:"school_name"`
	Region          string    `gorm:"column:region;not null" json:"region"`
	CP              string    `gorm:"column:cp;not null" json:"cp"`
	HashUUIDs       string    `gorm:"column:hash_uuids;not null" json:"hash_uuids"`
	PrimaryHashUUID string    `gorm:"column:primary_hash_uuid;not null" json:"primary_hash_uuid"`
	HashCount       int       `gorm:"column:hash_count;not null;default:0" json:"hash_count"`
	UpdateTime      time.Time `gorm:"column:update_time;autoUpdateTime" json:"update_time"`
	DataHash        string    `gorm:"column:data_hash;not null" json:"data_hash"`
}

// TableName 设置表名
func (School) TableName() string {
	return "nfa_school"
}

// SchoolTraffic 对应nfa_school_traffic表
type SchoolTraffic struct {
	ID         int64     `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	CreateTime time.Time `gorm:"column:create_time;not null" json:"create_time"`
	SchoolID   string    `gorm:"column:school_id;not null" json:"school_id"`
	SchoolName string    `gorm:"column:school_name;not null" json:"school_name"`
	Region     string    `gorm:"column:region;not null" json:"region"`
	CP         string    `gorm:"column:cp;not null" json:"cp"`
	HashUUID   string    `gorm:"column:hash_uuid;not null" json:"hash_uuid"`
	TotalRecv  int64     `gorm:"column:total_recv;not null;default:0" json:"total_recv"`
	TotalSend  int64     `gorm:"column:total_send;not null;default:0" json:"total_send"`
}

// TableName 设置表名
func (SchoolTraffic) TableName() string {
	return "nfa_school_traffic"
}

// TrafficResponse 流量数据响应结构
type TrafficResponse struct {
	CreateTime time.Time `json:"create_time"`
	SchoolID   string    `json:"school_id,omitempty"`
	SchoolName string    `json:"school_name,omitempty"`
	Region     string    `json:"region,omitempty"`
	CP         string    `json:"cp,omitempty"`
	TotalRecv  int64     `json:"total_recv"`
	TotalSend  int64     `json:"total_send"`
	Total      int64     `json:"total"`
}

// TrafficFilter 流量查询过滤条件
type TrafficFilter struct {
	StartTime              time.Time `form:"start_time"`
	EndTime                time.Time `form:"end_time"`
	SchoolName             string    `form:"school_name"`
	Region                 string    `form:"region"`
	CP                     string    `form:"cp"`
	Interval               string    `form:"interval" binding:"oneof=hour day week month 15m "` // 时间间隔：小时、天、周、月、15分钟
	Granularity            string    `form:"granularity"`                                       // 时间粒度：5m、5分钟；15m、15分钟；hour、1小时；day、1天
	Limit                  int       `form:"limit,default=100"`
	Offset                 int       `form:"offset,default=0"`
	SamplingInterval       int       `form:"-"` // 采样间隔，内部使用，不从表单获取
	UseSampling            bool      `form:"-"` // 是否使用采样，内部使用，不从表单获取
	OriginalExpectedPoints int       `form:"-"` // 原始预期数据点数量，内部使用，不从表单获取
	UserID                 *uint64   `form:"user_id" json:"user_id"` // v2：按用户可见院校范围过滤（nil/0 表示不启用）
}

// UserSchool 对应 user_schools 表，用于 v2 过滤用途（按用户过滤）
type UserSchool struct {
	ID        uint64    `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	UserID    uint64    `gorm:"column:user_id;not null" json:"user_id"`
	SchoolID  string    `gorm:"column:school_id;not null" json:"school_id"`
	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
}

func (UserSchool) TableName() string { return "user_schools" }
