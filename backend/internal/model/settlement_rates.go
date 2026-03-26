package model

import (
	"time"

	"gorm.io/datatypes"
)

// BusinessEntity 对应 business_entities 表
// 费用归属对象（客户、线路、节点、销售等）
type BusinessEntity struct {
	ID          uint64    `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	EntityType  string    `gorm:"column:entity_type;size:50;not null" json:"entity_type"`
	EntityName  string    `gorm:"column:entity_name;size:100;not null" json:"entity_name"`
	ContactInfo *string   `gorm:"column:contact_info;size:255" json:"contact_info,omitempty"`
	CreatedAt   time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt   time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

func (BusinessEntity) TableName() string { return "business_entities" }

// RateCustomer 对应 rate_customer 表
// 客户业务费率（NFA）
type RateCustomer struct {
	ID                    uint64         `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	Region                string         `gorm:"column:region;size:32;not null" json:"region"`
	CP                    string         `gorm:"column:cp;size:32;not null" json:"cp"`
	SchoolName            *string        `gorm:"column:school_name;size:128" json:"school_name,omitempty"`
	CustomerFee           *float64       `gorm:"column:customer_fee" json:"customer_fee,omitempty"`
	NetworkLineFee        *float64       `gorm:"column:network_line_fee" json:"network_line_fee,omitempty"`
	GeneralFee            *float64       `gorm:"column:general_fee" json:"general_fee,omitempty"`
	CustomerFeeOwnerID    *uint64        `gorm:"column:customer_fee_owner_id" json:"customer_fee_owner_id,omitempty"`
	NetworkLineFeeOwnerID *uint64        `gorm:"column:network_line_fee_owner_id" json:"network_line_fee_owner_id,omitempty"`
	GeneralFeeOwnerID     *uint64        `gorm:"column:general_fee_owner_id" json:"general_fee_owner_id,omitempty"`
	ChannelRate           *float64       `gorm:"column:channel_rate" json:"channel_rate,omitempty"`
	ChannelOwnerUserID    *uint64        `gorm:"column:channel_owner_user_id" json:"channel_owner_user_id,omitempty"`
	StartAt               *time.Time     `gorm:"column:start_at" json:"start_at,omitempty"`
	IncrementStartAt      *time.Time     `gorm:"column:increment_start_at" json:"increment_start_at,omitempty"`
	StockRatio            *float64       `gorm:"column:stock_ratio" json:"stock_ratio,omitempty"`
	IncrementRatio        *float64       `gorm:"column:increment_ratio" json:"increment_ratio,omitempty"`
	DailyIncrementValue   *float64       `gorm:"column:daily_increment_value" json:"daily_increment_value,omitempty"`
	FeeMode               string         `gorm:"column:fee_mode;size:16;not null;default:auto" json:"fee_mode"`
	Extra                 datatypes.JSON `gorm:"column:extra" json:"extra,omitempty"`
	LastSyncTime          *time.Time     `gorm:"column:last_sync_time" json:"last_sync_time,omitempty"`
	LastSyncRuleID        *uint64        `gorm:"column:last_sync_rule_id" json:"last_sync_rule_id,omitempty"`
	CreatedAt             time.Time      `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt             time.Time      `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

func (RateCustomer) TableName() string { return "rate_customer" }

// RateNode 对应 rate_node 表
// 节点业务费率（EDC）
type RateNode struct {
	ID                         uint64    `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	Region                     string    `gorm:"column:region;size:32;not null" json:"region"`
	CP                         string    `gorm:"column:cp;size:32;not null" json:"cp"`
	CPFee                      *float64  `gorm:"column:cp_fee" json:"cp_fee,omitempty"`
	CPFeeOwnerID               *uint64   `gorm:"column:cp_fee_owner_id" json:"cp_fee_owner_id,omitempty"`
	NodeConstructionFee        *float64  `gorm:"column:node_construction_fee" json:"node_construction_fee,omitempty"`
	NodeConstructionFeeOwnerID *uint64   `gorm:"column:node_construction_fee_owner_id" json:"node_construction_fee_owner_id,omitempty"`
	RackFee                    *float64  `gorm:"column:rack_fee" json:"rack_fee,omitempty"`
	RackFeeOwnerID             *uint64   `gorm:"column:rack_fee_owner_id" json:"rack_fee_owner_id,omitempty"`
	OtherFee                   *float64  `gorm:"column:other_fee" json:"other_fee,omitempty"`
	OtherFeeOwnerID            *uint64   `gorm:"column:other_fee_owner_id" json:"other_fee_owner_id,omitempty"`
	SettlementType             string    `gorm:"column:settlement_type;size:16;not null" json:"settlement_type"`
	CreatedAt                  time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt                  time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

func (RateNode) TableName() string { return "rate_node" }

// RateFinalCustomer 对应 rate_final_customer 表
// 最终客户费率（手工/自动）
type RateFinalCustomer struct {
	ID                      uint64     `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	Region                  string     `gorm:"column:region;size:32;not null" json:"region"`
	CP                      string     `gorm:"column:cp;size:32;not null" json:"cp"`
	SchoolName              string     `gorm:"column:school_name;size:128;not null" json:"school_name"`
	FinalFee                *float64   `gorm:"column:final_fee" json:"final_fee,omitempty"`
	FeeType                 string     `gorm:"column:fee_type;size:16;not null" json:"fee_type"`
	CustomerFee             *float64   `gorm:"column:customer_fee" json:"customer_fee,omitempty"`
	CustomerFeeOwnerID      *uint64    `gorm:"column:customer_fee_owner_id" json:"customer_fee_owner_id,omitempty"`
	NetworkLineFee          *float64   `gorm:"column:network_line_fee" json:"network_line_fee,omitempty"`
	NetworkLineFeeOwnerID   *uint64    `gorm:"column:network_line_fee_owner_id" json:"network_line_fee_owner_id,omitempty"`
	NodeDeductionFee        *float64   `gorm:"column:node_deduction_fee" json:"node_deduction_fee,omitempty"`
	NodeDeductionFeeOwnerID *uint64    `gorm:"column:node_deduction_fee_owner_id" json:"node_deduction_fee_owner_id,omitempty"`
	ChannelRate             *float64   `gorm:"column:channel_rate" json:"channel_rate,omitempty"`
	ChannelOwnerUserID      *uint64    `gorm:"column:channel_owner_user_id" json:"channel_owner_user_id,omitempty"`
	LastSyncTime            *time.Time `gorm:"column:last_sync_time" json:"last_sync_time,omitempty"`
	LastSyncRuleID          *uint64    `gorm:"column:last_sync_rule_id" json:"last_sync_rule_id,omitempty"`
	CreatedAt               time.Time  `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt               time.Time  `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

func (RateFinalCustomer) TableName() string { return "rate_final_customer" }

// SettlementCustomer 对应 settlement_customer 表
// 客户结算金额
type SettlementCustomer struct {
	ID                      uint64     `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	Region                  string     `gorm:"column:region;size:32;not null" json:"region"`
	CP                      string     `gorm:"column:cp;size:32;not null" json:"cp"`
	SchoolName              string     `gorm:"column:school_name;size:128;not null" json:"school_name"`
	SettlementValue         float64    `gorm:"column:settlement_value;not null" json:"settlement_value"`
	SettlementTime          time.Time  `gorm:"column:settlement_time;not null" json:"settlement_time"`
	ServiceDate             *time.Time `gorm:"column:service_date" json:"service_date,omitempty"`
	Recalculated            bool       `gorm:"column:recalculated" json:"recalculated"`
	LastRecalcTime          *time.Time `gorm:"column:last_recalc_time" json:"last_recalc_time,omitempty"`
	CustomerFee             *float64   `gorm:"column:customer_fee" json:"customer_fee,omitempty"`
	CustomerBill            *float64   `gorm:"column:customer_bill" json:"customer_bill,omitempty"`
	CustomerFeeOwnerID      *uint64    `gorm:"column:customer_fee_owner_id" json:"customer_fee_owner_id,omitempty"`
	NetworkLineFee          *float64   `gorm:"column:network_line_fee" json:"network_line_fee,omitempty"`
	NetworkLineBill         *float64   `gorm:"column:network_line_bill" json:"network_line_bill,omitempty"`
	NetworkLineFeeOwnerID   *uint64    `gorm:"column:network_line_fee_owner_id" json:"network_line_fee_owner_id,omitempty"`
	NodeDeductionFee        *float64   `gorm:"column:node_deduction_fee" json:"node_deduction_fee,omitempty"`
	NodeDeductionBill       *float64   `gorm:"column:node_deduction_bill" json:"node_deduction_bill,omitempty"`
	NodeDeductionFeeOwnerID *uint64    `gorm:"column:node_deduction_fee_owner_id" json:"node_deduction_fee_owner_id,omitempty"`
	// 新增渠道及折损追踪字段
	ChannelRate         *float64  `gorm:"column:channel_rate" json:"channel_rate,omitempty"`
	ChannelBill         *float64  `gorm:"column:channel_bill" json:"channel_bill,omitempty"`
	ChannelOwnerUserID  *uint64   `gorm:"column:channel_owner_user_id" json:"channel_owner_user_id,omitempty"`
	StockRatio          *float64  `gorm:"column:stock_ratio" json:"stock_ratio,omitempty"`
	IncrementRatio      *float64  `gorm:"column:increment_ratio" json:"increment_ratio,omitempty"`
	DailyIncrementValue *float64  `gorm:"column:daily_increment_value" json:"daily_increment_value,omitempty"`
	DiscountRuleID      *uint64   `gorm:"column:discount_rule_id" json:"discount_rule_id,omitempty"`
	ServiceYearIndex    *int      `gorm:"column:service_year_index" json:"service_year_index,omitempty"`
	CreatedAt           time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt           time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

func (SettlementCustomer) TableName() string { return "settlement_customer" }

// SettlementCustomerMonthly 客户结算月度聚合视图（按 region/cp/school_name/月 汇总）
type SettlementCustomerMonthly struct {
	Region                  string   `json:"region"`
	CP                      string   `json:"cp"`
	SchoolName              string   `json:"school_name"`
	ServiceDate             string   `json:"service_date"`
	DataSource              string   `json:"data_source,omitempty"`
	SettlementValue         float64  `json:"settlement_value"`
	CustomerFee             *float64 `json:"customer_fee,omitempty"`
	CustomerBill            *float64 `json:"customer_bill,omitempty"`
	CustomerFeeOwnerID      *uint64  `json:"customer_fee_owner_id,omitempty"`
	NetworkLineFee          *float64 `json:"network_line_fee,omitempty"`
	NetworkLineBill         *float64 `json:"network_line_bill,omitempty"`
	NetworkLineFeeOwnerID   *uint64  `json:"network_line_fee_owner_id,omitempty"`
	NodeDeductionFee        *float64 `json:"node_deduction_fee,omitempty"`
	NodeDeductionBill       *float64 `json:"node_deduction_bill,omitempty"`
	NodeDeductionFeeOwnerID *uint64  `json:"node_deduction_fee_owner_id,omitempty"`
	ChannelRate             *float64 `json:"channel_rate,omitempty"`
	ChannelBill             *float64 `json:"channel_bill,omitempty"`
	ChannelOwnerUserID      *uint64  `json:"channel_owner_user_id,omitempty"`
	StockRatio              *float64 `json:"stock_ratio,omitempty"`
	IncrementRatio          *float64 `json:"increment_ratio,omitempty"`
	DailyIncrementValue     *float64 `json:"daily_increment_value,omitempty"`
	Recalculated            bool     `json:"recalculated"`
	LastRecalcTime          *string  `json:"last_recalc_time,omitempty"`
}

// SettlementNodeDaily95 对应 settlement_node_daily95 表
// 节点日95结算金额
type SettlementNodeDaily95 struct {
	ID                         uint64    `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	Region                     string    `gorm:"column:region;size:32;not null" json:"region"`
	CP                         string    `gorm:"column:cp;size:32;not null" json:"cp"`
	CPFee                      *float64  `gorm:"column:cp_fee" json:"cp_fee,omitempty"`
	CPBill                     *float64  `gorm:"column:cp_bill" json:"cp_bill,omitempty"`
	CPFeeOwnerID               *uint64   `gorm:"column:cp_fee_owner_id" json:"cp_fee_owner_id,omitempty"`
	NodeConstructionFee        *float64  `gorm:"column:node_construction_fee" json:"node_construction_fee,omitempty"`
	NodeConstructionBill       *float64  `gorm:"column:node_construction_bill" json:"node_construction_bill,omitempty"`
	NodeConstructionFeeOwnerID *uint64   `gorm:"column:node_construction_fee_owner_id" json:"node_construction_fee_owner_id,omitempty"`
	RackFee                    *float64  `gorm:"column:rack_fee" json:"rack_fee,omitempty"`
	RackBill                   *float64  `gorm:"column:rack_bill" json:"rack_bill,omitempty"`
	RackFeeOwnerID             *uint64   `gorm:"column:rack_fee_owner_id" json:"rack_fee_owner_id,omitempty"`
	OtherFee                   *float64  `gorm:"column:other_fee" json:"other_fee,omitempty"`
	OtherBill                  *float64  `gorm:"column:other_bill" json:"other_bill,omitempty"`
	OtherFeeOwnerID            *uint64   `gorm:"column:other_fee_owner_id" json:"other_fee_owner_id,omitempty"`
	SettlementValue            float64   `gorm:"column:settlement_value;not null" json:"settlement_value"`
	SettlementTime             time.Time `gorm:"column:settlement_time;not null" json:"settlement_time"`
	Daily95Fee                 *float64  `gorm:"column:daily95_fee" json:"daily95_fee,omitempty"`
	Daily95Bill                *float64  `gorm:"column:daily95_bill" json:"daily95_bill,omitempty"`
	CreatedAt                  time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt                  time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

func (SettlementNodeDaily95) TableName() string { return "settlement_node_daily95" }

// SettlementNodeMonthly95 对应 settlement_node_monthly95 表
// 节点月95结算金额
type SettlementNodeMonthly95 struct {
	ID                         uint64    `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	Region                     string    `gorm:"column:region;size:32;not null" json:"region"`
	CP                         string    `gorm:"column:cp;size:32;not null" json:"cp"`
	CPFee                      *float64  `gorm:"column:cp_fee" json:"cp_fee,omitempty"`
	CPBill                     *float64  `gorm:"column:cp_bill" json:"cp_bill,omitempty"`
	CPFeeOwnerID               *uint64   `gorm:"column:cp_fee_owner_id" json:"cp_fee_owner_id,omitempty"`
	NodeConstructionFee        *float64  `gorm:"column:node_construction_fee" json:"node_construction_fee,omitempty"`
	NodeConstructionBill       *float64  `gorm:"column:node_construction_bill" json:"node_construction_bill,omitempty"`
	NodeConstructionFeeOwnerID *uint64   `gorm:"column:node_construction_fee_owner_id" json:"node_construction_fee_owner_id,omitempty"`
	RackFee                    *float64  `gorm:"column:rack_fee" json:"rack_fee,omitempty"`
	RackBill                   *float64  `gorm:"column:rack_bill" json:"rack_bill,omitempty"`
	RackFeeOwnerID             *uint64   `gorm:"column:rack_fee_owner_id" json:"rack_fee_owner_id,omitempty"`
	OtherFee                   *float64  `gorm:"column:other_fee" json:"other_fee,omitempty"`
	OtherBill                  *float64  `gorm:"column:other_bill" json:"other_bill,omitempty"`
	OtherFeeOwnerID            *uint64   `gorm:"column:other_fee_owner_id" json:"other_fee_owner_id,omitempty"`
	SettlementValue            float64   `gorm:"column:settlement_value;not null" json:"settlement_value"`
	SettlementTime             time.Time `gorm:"column:settlement_time;not null" json:"settlement_time"`
	Monthly95Fee               *float64  `gorm:"column:monthly95_fee" json:"monthly95_fee,omitempty"`
	Monthly95Bill              *float64  `gorm:"column:monthly95_bill" json:"monthly95_bill,omitempty"`
	CreatedAt                  time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt                  time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

func (SettlementNodeMonthly95) TableName() string { return "settlement_node_monthly95" }

// RateCustomerCustomFieldDef 对应 rate_customer_custom_field_defs 表
// 自定义字段定义：用于扩展 rate_customer.extra 的结构和校验
type RateCustomerCustomFieldDef struct {
	ID            uint64         `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	FieldKey      string         `gorm:"column:field_key;size:64;not null;unique" json:"field_key"`
	Label         string         `gorm:"column:label;size:64;not null" json:"label"`
	DataType      string         `gorm:"column:data_type;size:16;not null" json:"data_type"`
	Required      bool           `gorm:"column:required;not null" json:"required"`
	DefaultValue  datatypes.JSON `gorm:"column:default_value" json:"default_value,omitempty"`
	ValidateRegex *string        `gorm:"column:validate_regex;size:255" json:"validate_regex,omitempty"`
	Min           *float64       `gorm:"column:min" json:"min,omitempty"`
	Max           *float64       `gorm:"column:max" json:"max,omitempty"`
	Precision     *int           `gorm:"column:precision" json:"precision,omitempty"`
	EnumOptions   datatypes.JSON `gorm:"column:enum_options" json:"enum_options,omitempty"`
	UsableInRules bool           `gorm:"column:usable_in_rules;not null" json:"usable_in_rules"`
	Enabled       bool           `gorm:"column:enabled;not null" json:"enabled"`
	CreatedAt     time.Time      `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt     time.Time      `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

func (RateCustomerCustomFieldDef) TableName() string { return "rate_customer_custom_field_defs" }

// RateCustomerSyncRule 对应 rate_customer_sync_rules 表
// 同步规则：支持范围、条件、字段白名单、覆盖策略与动作（模板/表达式）
type RateCustomerSyncRule struct {
	ID                uint64         `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	Name              string         `gorm:"column:name;size:100;not null" json:"name"`
	Enabled           bool           `gorm:"column:enabled;not null" json:"enabled"`
	Priority          int            `gorm:"column:priority;not null" json:"priority"`
	ScopeRegion       datatypes.JSON `gorm:"column:scope_region" json:"scope_region,omitempty"`
	ScopeCP           datatypes.JSON `gorm:"column:scope_cp" json:"scope_cp,omitempty"`
	ConditionExpr     *string        `gorm:"column:condition_expr" json:"condition_expr,omitempty"`
	FieldsToUpdate    datatypes.JSON `gorm:"column:fields_to_update" json:"fields_to_update,omitempty"`
	OverwriteStrategy string         `gorm:"column:overwrite_strategy;size:16;not null" json:"overwrite_strategy"`
	Actions           datatypes.JSON `gorm:"column:actions;not null" json:"actions"`
	CreatedBy         *uint64        `gorm:"column:created_by" json:"created_by,omitempty"`
	UpdatedBy         *uint64        `gorm:"column:updated_by" json:"updated_by,omitempty"`
	CreatedAt         time.Time      `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt         time.Time      `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

func (RateCustomerSyncRule) TableName() string { return "rate_customer_sync_rules" }

// RateDiscountRule 对应 rate_discount_rule 表
// 客户费率折损规则主表
type RateDiscountRule struct {
	ID        uint64         `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	Name      string         `gorm:"column:name;size:128;not null" json:"name"`
	ScopeType string         `gorm:"column:scope_type;size:32;not null" json:"scope_type"`
	ScopeKey  *string        `gorm:"column:scope_key;size:128" json:"scope_key,omitempty"`
	Fields    datatypes.JSON `gorm:"column:fields" json:"fields,omitempty"`
	Enabled   bool           `gorm:"column:enabled;not null" json:"enabled"`
	Priority  int            `gorm:"column:priority;not null" json:"priority"`
	CreatedAt time.Time      `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt time.Time      `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

func (RateDiscountRule) TableName() string { return "rate_discount_rule" }

// RateDiscountRuleItem 对应 rate_discount_rule_item 表
// 客户费率折损规则明细
type RateDiscountRuleItem struct {
	ID           uint64    `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	RuleID       uint64    `gorm:"column:rule_id" json:"rule_id"`
	FromYear     int       `gorm:"column:from_year" json:"from_year"`
	ToYear       *int      `gorm:"column:to_year" json:"to_year,omitempty"`
	DiscountRate float64   `gorm:"column:discount_rate" json:"discount_rate"`
	CreatedAt    time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt    time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

func (RateDiscountRuleItem) TableName() string { return "rate_discount_rule_item" }
