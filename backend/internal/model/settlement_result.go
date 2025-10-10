package model

import (
    "time"

    "gorm.io/datatypes"
)

// SettlementResultFilter 定义结算结果的查询条件
// region/cp/school_name: 精确匹配；start_date/end_date: 闭区间，必填
// formula_id: 若传递则指定公式；未传则默认使用启用的标准公式
// limit/offset: 分页
// 注意：时间格式使用 yyyy-mm-dd
//
// swagger:parameters settlementResultQuery
//
// 约定：start_date 和 end_date 均为自然日；后端会根据该范围聚合日流量
// 其中 average_95_flow = sum(daily_95_value) / days
// days = 包含首尾的总天数（若中间缺少流量数据，后端视为 0 并提示）
type SettlementResultFilter struct {
    ID         uint64    `form:"id" json:"id"`
    Region     string    `form:"region" json:"region"`
    CP         string    `form:"cp" json:"cp"`
    SchoolName string    `form:"school_name" json:"school_name"`
    SchoolID   string    `form:"school_id" json:"school_id"`
    StartDate  time.Time `form:"start_date" time_format:"2006-01-02" json:"start_date"`
    EndDate    time.Time `form:"end_date" time_format:"2006-01-02" json:"end_date"`
    FormulaID  uint64    `form:"formula_id" json:"formula_id"`
    Limit      int       `form:"limit,default=50" json:"limit"`
    Offset     int       `form:"offset,default=0" json:"offset"`
    UserID     *uint64   `form:"-" json:"-"`
    UnitBase   int       `form:"unit_base" json:"-"`
}

// SettlementResultRecord 对应 nfa_settlement_results 表
type SettlementResultRecord struct {
	ID                uint64         `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	FormulaID         uint64         `gorm:"column:formula_id;not null" json:"formula_id"`
	FormulaName       string         `gorm:"column:formula_name;size:128;not null" json:"formula_name"`
	FormulaTokens     datatypes.JSON `gorm:"column:formula_tokens;type:json;not null" json:"formula_tokens"`
	Region            string         `gorm:"column:region;size:64;not null" json:"region"`
	CP                string         `gorm:"column:cp;size:64;not null" json:"cp"`
	SchoolID          string         `gorm:"column:school_id;size:64;not null" json:"school_id"`
	SchoolName        string         `gorm:"column:school_name;size:255;not null" json:"school_name"`
	StartDate         time.Time      `gorm:"column:start_date;type:date;not null" json:"start_date"`
	EndDate           time.Time      `gorm:"column:end_date;type:date;not null" json:"end_date"`
	BillingDays       int            `gorm:"column:billing_days;not null" json:"billing_days"`
	Total95Flow       float64        `gorm:"column:total_95_flow;not null" json:"total_95_flow"`
	Average95Flow     float64        `gorm:"column:average_95_flow;not null" json:"average_95_flow"`
	CustomerFee       *float64       `gorm:"column:customer_fee" json:"customer_fee,omitempty"`
	NetworkLineFee    *float64       `gorm:"column:network_line_fee" json:"network_line_fee,omitempty"`
	NodeDeductionFee  *float64       `gorm:"column:node_deduction_fee" json:"node_deduction_fee,omitempty"`
	FinalFee          *float64       `gorm:"column:final_fee" json:"final_fee,omitempty"`
	Amount            *float64       `gorm:"column:amount" json:"amount,omitempty"`
	Currency          string         `gorm:"column:currency;size:8;not null" json:"currency"`
	MissingDays       int            `gorm:"column:missing_days;not null" json:"missing_days"`
	MissingFields     datatypes.JSON `gorm:"column:missing_fields;type:json" json:"missing_fields"`
	CalculationDetail datatypes.JSON `gorm:"column:calculation_detail;type:json" json:"calculation_detail"`
	CalculatedBy      *uint64        `gorm:"column:calculated_by" json:"calculated_by,omitempty"`
	CreatedAt         time.Time      `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt         time.Time      `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

func (SettlementResultRecord) TableName() string { return "nfa_settlement_results" }

// SettlementResultItem 表示某个“校区+运营商”的结算结果
// Amount = 根据公式对费率字段/平均流量求值的结果
// FormulaTokens 使用 JSON 字符串存储，便于前端展示公式内容
// MissingFields / MissingDays 用于提示数据不完整
//
// swagger:model SettlementResultItem
type SettlementResultItem struct {
    Region            string    `json:"region"`
    CP                string    `json:"cp"`
    SchoolID          string    `json:"school_id"`
    SchoolName        string    `json:"school_name"`
    BillingDays       int       `json:"billing_days"`
    Average95Flow     float64   `json:"average_95_flow"`
    Total95Flow       float64   `json:"total_95_flow"`
    MissingDays       int       `json:"missing_days"`
    FormulaID         uint64    `json:"formula_id"`
    FormulaName       string    `json:"formula_name"`
    FormulaTokens     string    `json:"formula_tokens"`
    CustomerFee       float64   `json:"customer_fee"`
    NetworkLineFee    float64   `json:"network_line_fee"`
    NodeDeductionFee  float64   `json:"node_deduction_fee"`
    FinalFee          float64   `json:"final_fee"`
    Amount            float64   `json:"amount"`
    Currency          string    `json:"currency"`
    StartDate         time.Time `json:"start_date"`
    EndDate           time.Time `json:"end_date"`
    UpdatedAt         time.Time `json:"updated_at"`
    MissingFields     []string  `json:"missing_fields"`
    CalculationDetail string    `json:"calculation_detail"`
}

// SettlementFormulaToken 用于解析公式 JSON
type SettlementFormulaToken struct {
    ID    string `json:"id"`
    Type  string `json:"type"`
    Value string `json:"value"`
    Label string `json:"label"`
}

// AggregatedFlowRecord 用于承载按校区聚合的日95流量数据
type AggregatedFlowRecord struct {
    Region       string    `json:"region"`
    CP           string    `json:"cp"`
    SchoolID     string    `json:"school_id"`
    SchoolName   string    `json:"school_name"`
    DayCount     int       `json:"day_count"`
    TotalFlow    float64   `json:"total_flow"`
    MinDate      time.Time `json:"min_date"`
    MaxDate      time.Time `json:"max_date"`
    LatestUpdate time.Time `json:"latest_update"`
    CustomerFee      *float64 `json:"customer_fee,omitempty"`
    NetworkLineFee   *float64 `json:"network_line_fee,omitempty"`
    NodeDeductionFee *float64 `json:"node_deduction_fee,omitempty"`
    FinalFee         *float64 `json:"final_fee,omitempty"`
}
