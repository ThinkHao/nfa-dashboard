package model

import "time"

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
    ID                        uint64    `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
    Region                    string    `gorm:"column:region;size:32;not null" json:"region"`
    CP                        string    `gorm:"column:cp;size:32;not null" json:"cp"`
    SchoolName                *string   `gorm:"column:school_name;size:128" json:"school_name,omitempty"`
    CustomerFee               *float64  `gorm:"column:customer_fee" json:"customer_fee,omitempty"`
    NetworkLineFee            *float64  `gorm:"column:network_line_fee" json:"network_line_fee,omitempty"`
    GeneralFee                *float64  `gorm:"column:general_fee" json:"general_fee,omitempty"`
    CustomerFeeOwnerID        *uint64   `gorm:"column:customer_fee_owner_id" json:"customer_fee_owner_id,omitempty"`
    NetworkLineFeeOwnerID     *uint64   `gorm:"column:network_line_fee_owner_id" json:"network_line_fee_owner_id,omitempty"`
    CreatedAt                 time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
    UpdatedAt                 time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

func (RateCustomer) TableName() string { return "rate_customer" }

// RateNode 对应 rate_node 表
// 节点业务费率（EDC）
type RateNode struct {
    ID                           uint64    `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
    Region                       string    `gorm:"column:region;size:32;not null" json:"region"`
    CP                           string    `gorm:"column:cp;size:32;not null" json:"cp"`
    CPFee                        *float64  `gorm:"column:cp_fee" json:"cp_fee,omitempty"`
    CPFeeOwnerID                 *uint64   `gorm:"column:cp_fee_owner_id" json:"cp_fee_owner_id,omitempty"`
    NodeConstructionFee          *float64  `gorm:"column:node_construction_fee" json:"node_construction_fee,omitempty"`
    NodeConstructionFeeOwnerID   *uint64   `gorm:"column:node_construction_fee_owner_id" json:"node_construction_fee_owner_id,omitempty"`
    RackFee                      *float64  `gorm:"column:rack_fee" json:"rack_fee,omitempty"`
    RackFeeOwnerID               *uint64   `gorm:"column:rack_fee_owner_id" json:"rack_fee_owner_id,omitempty"`
    OtherFee                     *float64  `gorm:"column:other_fee" json:"other_fee,omitempty"`
    OtherFeeOwnerID              *uint64   `gorm:"column:other_fee_owner_id" json:"other_fee_owner_id,omitempty"`
    SettlementType               string    `gorm:"column:settlement_type;size:16;not null" json:"settlement_type"`
    CreatedAt                    time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
    UpdatedAt                    time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

func (RateNode) TableName() string { return "rate_node" }

// RateFinalCustomer 对应 rate_final_customer 表
// 最终客户费率（手工/自动）
type RateFinalCustomer struct {
    ID                           uint64    `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
    Region                       string    `gorm:"column:region;size:32;not null" json:"region"`
    CP                           string    `gorm:"column:cp;size:32;not null" json:"cp"`
    SchoolName                   string    `gorm:"column:school_name;size:128;not null" json:"school_name"`
    FinalFee                     *float64  `gorm:"column:final_fee" json:"final_fee,omitempty"`
    FeeType                      string    `gorm:"column:fee_type;size:16;not null" json:"fee_type"`
    CustomerFee                  *float64  `gorm:"column:customer_fee" json:"customer_fee,omitempty"`
    CustomerFeeOwnerID           *uint64   `gorm:"column:customer_fee_owner_id" json:"customer_fee_owner_id,omitempty"`
    NetworkLineFee               *float64  `gorm:"column:network_line_fee" json:"network_line_fee,omitempty"`
    NetworkLineFeeOwnerID        *uint64   `gorm:"column:network_line_fee_owner_id" json:"network_line_fee_owner_id,omitempty"`
    NodeDeductionFee             *float64  `gorm:"column:node_deduction_fee" json:"node_deduction_fee,omitempty"`
    NodeDeductionFeeOwnerID      *uint64   `gorm:"column:node_deduction_fee_owner_id" json:"node_deduction_fee_owner_id,omitempty"`
    CreatedAt                    time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
    UpdatedAt                    time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

func (RateFinalCustomer) TableName() string { return "rate_final_customer" }

// SettlementCustomer 对应 settlement_customer 表
// 客户结算金额
type SettlementCustomer struct {
    ID                           uint64    `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
    Region                       string    `gorm:"column:region;size:32;not null" json:"region"`
    CP                           string    `gorm:"column:cp;size:32;not null" json:"cp"`
    SchoolName                   string    `gorm:"column:school_name;size:128;not null" json:"school_name"`
    SettlementValue              float64   `gorm:"column:settlement_value;not null" json:"settlement_value"`
    SettlementTime               time.Time `gorm:"column:settlement_time;not null" json:"settlement_time"`
    CustomerFee                  *float64  `gorm:"column:customer_fee" json:"customer_fee,omitempty"`
    CustomerBill                 *float64  `gorm:"column:customer_bill" json:"customer_bill,omitempty"`
    CustomerFeeOwnerID           *uint64   `gorm:"column:customer_fee_owner_id" json:"customer_fee_owner_id,omitempty"`
    NetworkLineFee               *float64  `gorm:"column:network_line_fee" json:"network_line_fee,omitempty"`
    NetworkLineBill              *float64  `gorm:"column:network_line_bill" json:"network_line_bill,omitempty"`
    NetworkLineFeeOwnerID        *uint64   `gorm:"column:network_line_fee_owner_id" json:"network_line_fee_owner_id,omitempty"`
    NodeDeductionFee             *float64  `gorm:"column:node_deduction_fee" json:"node_deduction_fee,omitempty"`
    NodeDeductionBill            *float64  `gorm:"column:node_deduction_bill" json:"node_deduction_bill,omitempty"`
    NodeDeductionFeeOwnerID      *uint64   `gorm:"column:node_deduction_fee_owner_id" json:"node_deduction_fee_owner_id,omitempty"`
    CreatedAt                    time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
    UpdatedAt                    time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

func (SettlementCustomer) TableName() string { return "settlement_customer" }

// SettlementNodeDaily95 对应 settlement_node_daily95 表
// 节点日95结算金额
type SettlementNodeDaily95 struct {
    ID                           uint64    `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
    Region                       string    `gorm:"column:region;size:32;not null" json:"region"`
    CP                           string    `gorm:"column:cp;size:32;not null" json:"cp"`
    CPFee                        *float64  `gorm:"column:cp_fee" json:"cp_fee,omitempty"`
    CPBill                       *float64  `gorm:"column:cp_bill" json:"cp_bill,omitempty"`
    CPFeeOwnerID                 *uint64   `gorm:"column:cp_fee_owner_id" json:"cp_fee_owner_id,omitempty"`
    NodeConstructionFee          *float64  `gorm:"column:node_construction_fee" json:"node_construction_fee,omitempty"`
    NodeConstructionBill         *float64  `gorm:"column:node_construction_bill" json:"node_construction_bill,omitempty"`
    NodeConstructionFeeOwnerID   *uint64   `gorm:"column:node_construction_fee_owner_id" json:"node_construction_fee_owner_id,omitempty"`
    RackFee                      *float64  `gorm:"column:rack_fee" json:"rack_fee,omitempty"`
    RackBill                     *float64  `gorm:"column:rack_bill" json:"rack_bill,omitempty"`
    RackFeeOwnerID               *uint64   `gorm:"column:rack_fee_owner_id" json:"rack_fee_owner_id,omitempty"`
    OtherFee                     *float64  `gorm:"column:other_fee" json:"other_fee,omitempty"`
    OtherBill                    *float64  `gorm:"column:other_bill" json:"other_bill,omitempty"`
    OtherFeeOwnerID              *uint64   `gorm:"column:other_fee_owner_id" json:"other_fee_owner_id,omitempty"`
    SettlementValue              float64   `gorm:"column:settlement_value;not null" json:"settlement_value"`
    SettlementTime               time.Time `gorm:"column:settlement_time;not null" json:"settlement_time"`
    Daily95Fee                   *float64  `gorm:"column:daily95_fee" json:"daily95_fee,omitempty"`
    Daily95Bill                  *float64  `gorm:"column:daily95_bill" json:"daily95_bill,omitempty"`
    CreatedAt                    time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
    UpdatedAt                    time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

func (SettlementNodeDaily95) TableName() string { return "settlement_node_daily95" }

// SettlementNodeMonthly95 对应 settlement_node_monthly95 表
// 节点月95结算金额
type SettlementNodeMonthly95 struct {
    ID                           uint64    `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
    Region                       string    `gorm:"column:region;size:32;not null" json:"region"`
    CP                           string    `gorm:"column:cp;size:32;not null" json:"cp"`
    CPFee                        *float64  `gorm:"column:cp_fee" json:"cp_fee,omitempty"`
    CPBill                       *float64  `gorm:"column:cp_bill" json:"cp_bill,omitempty"`
    CPFeeOwnerID                 *uint64   `gorm:"column:cp_fee_owner_id" json:"cp_fee_owner_id,omitempty"`
    NodeConstructionFee          *float64  `gorm:"column:node_construction_fee" json:"node_construction_fee,omitempty"`
    NodeConstructionBill         *float64  `gorm:"column:node_construction_bill" json:"node_construction_bill,omitempty"`
    NodeConstructionFeeOwnerID   *uint64   `gorm:"column:node_construction_fee_owner_id" json:"node_construction_fee_owner_id,omitempty"`
    RackFee                      *float64  `gorm:"column:rack_fee" json:"rack_fee,omitempty"`
    RackBill                     *float64  `gorm:"column:rack_bill" json:"rack_bill,omitempty"`
    RackFeeOwnerID               *uint64   `gorm:"column:rack_fee_owner_id" json:"rack_fee_owner_id,omitempty"`
    OtherFee                     *float64  `gorm:"column:other_fee" json:"other_fee,omitempty"`
    OtherBill                    *float64  `gorm:"column:other_bill" json:"other_bill,omitempty"`
    OtherFeeOwnerID              *uint64   `gorm:"column:other_fee_owner_id" json:"other_fee_owner_id,omitempty"`
    SettlementValue              float64   `gorm:"column:settlement_value;not null" json:"settlement_value"`
    SettlementTime               time.Time `gorm:"column:settlement_time;not null" json:"settlement_time"`
    Monthly95Fee                 *float64  `gorm:"column:monthly95_fee" json:"monthly95_fee,omitempty"`
    Monthly95Bill                *float64  `gorm:"column:monthly95_bill" json:"monthly95_bill,omitempty"`
    CreatedAt                    time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
    UpdatedAt                    time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

func (SettlementNodeMonthly95) TableName() string { return "settlement_node_monthly95" }
