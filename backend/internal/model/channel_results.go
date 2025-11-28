package model

import "time"

// ChannelResultFilter 用于渠道维度结算查询
// 支持按照渠道名称模糊过滤
// 必填：start_date, end_date
// 可选：formula_id, limit, offset
// 时间格式：yyyy-mm-dd
// 注意：该过滤与 SettlementResultFilter 的周期必须一致（同起止日期）
type ChannelResultFilter struct {
    ChannelName string    `form:"channel_name" json:"channel_name"`
    UserName    string    `form:"user_name" json:"user_name"`
    StartDate   time.Time `form:"start_date" time_format:"2006-01-02" json:"start_date"`
    EndDate     time.Time `form:"end_date" time_format:"2006-01-02" json:"end_date"`
    FormulaID   uint64    `form:"formula_id" json:"formula_id"`
    Limit       int       `form:"limit,default=50" json:"limit"`
    Offset      int       `form:"offset,default=0" json:"offset"`
}

// ChannelSettlementResultItem 渠道聚合后的结果
// breakdown_detail 为各分项金额明细的 JSON 字符串
// 注意：金额单位与 SettlementResultItem 保持一致
type ChannelSettlementResultItem struct {
    UserID      uint64    `json:"user_id"`
    UserName    string    `json:"user_name"`
    Amount      float64   `json:"amount"`
    Currency    string    `json:"currency"`
    StartDate   time.Time `json:"start_date"`
    EndDate     time.Time `json:"end_date"`
    FormulaID   uint64    `json:"formula_id"`
    FormulaName string    `json:"formula_name"`
    Breakdown   string    `json:"breakdown_detail"`
    UpdatedAt   time.Time `json:"updated_at"`
}

// ChannelOwnerJoinRow 用于从缓存结果与费率归属联查得到的一行记录
// 其中包含每个学校结算结果 + 三类 owner 归属
// 说明：FinalFee 无 owner 归属
type ChannelOwnerJoinRow struct {
    Result    SettlementResultRecord
    CustOwner *uint64
    NetOwner  *uint64
    NodeOwner *uint64
}
