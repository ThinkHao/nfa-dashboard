package model

import "time"

type EDCEntity struct {
	ID          uint64    `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	EDCName     string    `gorm:"column:edc_name;not null" json:"edc_name"`
	SN          string    `gorm:"column:sn" json:"sn"`
	DisplayName string    `gorm:"column:display_name;not null" json:"display_name"`
	Region      string    `gorm:"column:region;not null" json:"region"`
	CP          string    `gorm:"column:cp;not null" json:"cp"`
	IsBackup    bool      `gorm:"column:is_backup;not null;default:false" json:"is_backup"`
	Enabled     bool      `gorm:"column:enabled;not null;default:true" json:"enabled"`
	Remark      string    `gorm:"column:remark" json:"remark"`
	DataHash    string    `gorm:"column:data_hash;not null" json:"data_hash"`
	CreatedAt   time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt   time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

func (EDCEntity) TableName() string { return "edc_entities" }

type EDCTraffic5m struct {
	Bucket5m    time.Time `gorm:"column:bucket_5m;primaryKey" json:"bucket_5m"`
	EntityID    uint64    `gorm:"column:entity_id;primaryKey" json:"entity_id"`
	Region      string    `gorm:"column:region;not null" json:"region"`
	CP          string    `gorm:"column:cp;not null" json:"cp"`
	DisplayName string    `gorm:"column:display_name;not null" json:"display_name"`
	ServiceSize int64     `gorm:"column:service_size;not null;default:0" json:"service_size"`
	CacheSize   int64     `gorm:"column:cache_size;not null;default:0" json:"cache_size"`
	RecordCount int       `gorm:"column:record_count;not null;default:0" json:"record_count"`
	UpdatedAt   time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

func (EDCTraffic5m) TableName() string { return "edc_traffic_5m" }

type EDCNodeTrafficPoint struct {
	Bucket5m    time.Time `gorm:"column:bucket_5m" json:"bucket_5m"`
	EntityID    uint64    `gorm:"column:entity_id" json:"entity_id"`
	Region      string    `gorm:"column:region" json:"region"`
	CP          string    `gorm:"column:cp" json:"cp"`
	DisplayName string    `gorm:"column:display_name" json:"display_name"`
	ServiceSize int64     `gorm:"column:service_size" json:"service_size"`
	CacheSize   int64     `gorm:"column:cache_size" json:"cache_size"`
	RecordCount int       `gorm:"column:record_count" json:"record_count"`
}

type EDCEntityFilter struct {
	DisplayName      string
	Region           string
	CP               string
	EnabledOnly      bool
	AllowedEntityIDs []uint64
	Limit            int
	Offset           int
}

type EDCTrafficFilter struct {
	StartTime        time.Time `form:"start_time"`
	EndTime          time.Time `form:"end_time"`
	DisplayName      string    `form:"display_name"`
	Region           string    `form:"region"`
	CP               string    `form:"cp"`
	AllowedEntityIDs []uint64  `form:"-" json:"-"`
	ScopeSource      string    `form:"-" json:"-"`
}

type EDCTrafficResponse struct {
	Bucket5m    time.Time `gorm:"column:bucket_5m" json:"create_time"`
	EntityID    uint64    `gorm:"column:entity_id" json:"entity_id,omitempty"`
	DisplayName string    `gorm:"column:display_name" json:"display_name,omitempty"`
	Region      string    `gorm:"column:region" json:"region,omitempty"`
	CP          string    `gorm:"column:cp" json:"cp,omitempty"`
	ServiceSize int64     `gorm:"column:service_size" json:"service_size"`
	CacheSize   int64     `gorm:"column:cache_size" json:"cache_size"`
	Total       int64     `gorm:"-" json:"total"`
}

const (
	EDCTrafficScopeRuleTypeAllow = "allow"
	EDCTrafficScopeRuleTypeDeny  = "deny"

	EDCTrafficScopeDimensionRegion = "region"
	EDCTrafficScopeDimensionCP     = "cp"
	EDCTrafficScopeDimensionEntity = "entity"

	EDCTrafficScopeSourceNone             = "none"
	EDCTrafficScopeSourcePolicyRule       = "policy_rule"
	EDCTrafficScopeSourceDefaultAdminRole = "default_admin_role"
)

type EDCTrafficScopeRuleGroup struct {
	ID         uint64                     `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	UserID     uint64                     `gorm:"column:user_id;not null" json:"user_id"`
	RuleType   string                     `gorm:"column:rule_type;not null" json:"rule_type"`
	Conditions []EDCTrafficScopeCondition `gorm:"foreignKey:GroupID" json:"conditions"`
	CreatedAt  time.Time                  `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt  time.Time                  `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

func (EDCTrafficScopeRuleGroup) TableName() string { return "edc_traffic_scope_rule_groups" }

type EDCTrafficScopeCondition struct {
	ID             uint64    `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	GroupID        uint64    `gorm:"column:group_id;not null" json:"group_id"`
	DimensionType  string    `gorm:"column:dimension_type;not null" json:"dimension_type"`
	DimensionValue string    `gorm:"column:dimension_value;not null" json:"dimension_value"`
	CreatedAt      time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
}

func (EDCTrafficScopeCondition) TableName() string { return "edc_traffic_scope_rule_conditions" }

type EffectiveEDCTrafficScope struct {
	Source           string   `json:"source"`
	AllowedEntityIDs []uint64 `json:"allowed_entity_ids"`
}
