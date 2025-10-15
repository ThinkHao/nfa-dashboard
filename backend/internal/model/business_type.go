package model

import "time"

// BusinessType 对应 business_types 表
// 作为业务对象类型的枚举来源
// 约束：code 唯一且不可修改；enabled 标识是否启用
// 其他模块（如 EntitiesService）仅允许选择 enabled=true 的类型

type BusinessType struct {
    ID          uint64     `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
    Code        string     `gorm:"column:code;size:64;unique;not null" json:"code"`
    Name        string     `gorm:"column:name;size:128;not null" json:"name"`
    Description *string    `gorm:"column:description;size:255" json:"description,omitempty"`
    Enabled     bool       `gorm:"column:enabled;not null;default:true" json:"enabled"`
    CreatedAt   time.Time  `gorm:"column:created_at;autoCreateTime" json:"created_at"`
    UpdatedAt   time.Time  `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

func (BusinessType) TableName() string { return "business_types" }
