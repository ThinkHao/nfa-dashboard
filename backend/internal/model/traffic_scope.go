package model

import "time"

const (
	TrafficScopeRuleTypeAllow = "allow"
	TrafficScopeRuleTypeDeny  = "deny"

	TrafficScopeDimensionRegion = "region"
	TrafficScopeDimensionCP     = "cp"
	TrafficScopeDimensionSchool = "school"

	TrafficScopeSourcePolicyRule       = "policy_rule"
	TrafficScopeSourceLegacyUserSchool = "legacy_user_school"
	TrafficScopeSourceDefaultAdminRole = "default_admin_role"
	TrafficScopeSourceNone             = "none"
)

type TrafficScopeRuleGroup struct {
	ID           uint64                  `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	UserID       uint64                  `gorm:"column:user_id;not null;index:idx_traffic_scope_group_user" json:"user_id"`
	RuleType     string                  `gorm:"column:rule_type;size:16;not null" json:"rule_type"`
	LegacyRuleID *uint64                 `gorm:"column:legacy_rule_id;uniqueIndex" json:"-"`
	CreatedAt    time.Time               `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt    time.Time               `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
	Conditions   []TrafficScopeCondition `gorm:"foreignKey:GroupID;references:ID" json:"conditions"`
}

func (TrafficScopeRuleGroup) TableName() string { return "traffic_scope_rule_groups" }

type TrafficScopeCondition struct {
	ID             uint64    `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	GroupID        uint64    `gorm:"column:group_id;not null;index:idx_traffic_scope_condition_group" json:"group_id"`
	DimensionType  string    `gorm:"column:dimension_type;size:16;not null" json:"dimension_type"`
	DimensionValue string    `gorm:"column:dimension_value;size:255;not null" json:"dimension_value"`
	CreatedAt      time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
}

func (TrafficScopeCondition) TableName() string { return "traffic_scope_rule_conditions" }

type TrafficScopeSchoolKey struct {
	SchoolID string `json:"school_id"`
	Region   string `json:"region"`
	CP       string `json:"cp"`
}

type EffectiveTrafficScope struct {
	UserID            uint64                  `json:"user_id"`
	Source            string                  `json:"source"`
	AllowedSchoolKeys []TrafficScopeSchoolKey `json:"allowed_school_keys"`
	AllowedSchoolIDs  []string                `json:"allowed_school_ids"`
}

type EffectiveTrafficScopePreview struct {
	UserID          uint64                  `json:"user_id"`
	Source          string                  `json:"source"`
	Rules           []TrafficScopeRuleGroup `json:"rules"`
	LegacySchoolIDs []string                `json:"legacy_school_ids"`
	AllowedSchools  []School                `json:"allowed_schools"`
}
