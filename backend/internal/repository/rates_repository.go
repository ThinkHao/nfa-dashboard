package repository

import (
    "nfa-dashboard/internal/model"

    "gorm.io/gorm"
    "gorm.io/gorm/clause"
)

// RatesRepository 费率仓储接口
type RatesRepository interface {
    // 客户业务费率
    ListCustomerRates(filter map[string]interface{}, limit, offset int) ([]model.RateCustomer, int64, error)
    UpsertCustomerRate(rate *model.RateCustomer) error
    UpdateCustomerByID(id uint64, updates map[string]interface{}) error

    // 节点业务费率
    ListNodeRates(filter map[string]interface{}, limit, offset int) ([]model.RateNode, int64, error)
    UpsertNodeRate(rate *model.RateNode) error

    // 最终客户费率
    ListFinalCustomerRates(filter map[string]interface{}, limit, offset int) ([]model.RateFinalCustomer, int64, error)
    UpsertFinalCustomerRate(rate *model.RateFinalCustomer) error

    // 初始化最终客户费率（从 rate_customer 同步，保护 config 记录）
    InitFinalCustomerRatesFromCustomer() (int64, error)

    // 刷新最终客户费率（重算 final_fee，仅 auto）
    RefreshFinalCustomerRates() (int64, error)
}

type ratesRepository struct{}

func NewRatesRepository() RatesRepository { return &ratesRepository{} }

// ListCustomerRates 列表查询客户业务费率
func (r *ratesRepository) ListCustomerRates(filter map[string]interface{}, limit, offset int) ([]model.RateCustomer, int64, error) {
    var items []model.RateCustomer
    var count int64
    q := model.DB.Model(&model.RateCustomer{})
    if v, ok := filter["region"]; ok && v != "" { q = q.Where("region = ?", v) }
    if v, ok := filter["cp"]; ok && v != "" { q = q.Where("cp = ?", v) }
    if v, ok := filter["school_name"]; ok && v != "" { q = q.Where("school_name = ?", v) }
    if err := q.Count(&count).Error; err != nil { return nil, 0, err }
    if count == 0 { return []model.RateCustomer{}, 0, nil }
    if err := q.Order("updated_at DESC").Limit(limit).Offset(offset).Find(&items).Error; err != nil { return nil, 0, err }
    return items, count, nil
}

// UpsertCustomerRate 基于唯一键(region,cp,school_name)进行插入或更新
func (r *ratesRepository) UpsertCustomerRate(rate *model.RateCustomer) error {
    return model.DB.Clauses(clause.OnConflict{
        Columns:   []clause.Column{{Name: "region"}, {Name: "cp"}, {Name: "school_name"}},
        DoUpdates: clause.AssignmentColumns([]string{"customer_fee", "network_line_fee", "general_fee", "customer_fee_owner_id", "network_line_fee_owner_id", "updated_at"}),
    }).Create(rate).Error
}

// UpdateCustomerByID 基于主键进行局部字段更新
func (r *ratesRepository) UpdateCustomerByID(id uint64, updates map[string]interface{}) error {
    if id == 0 { return gorm.ErrInvalidData }
    if len(updates) == 0 { return nil }
    return model.DB.Model(&model.RateCustomer{}).Where("id = ?", id).Updates(updates).Error
}

// ListNodeRates 列表查询节点业务费率
func (r *ratesRepository) ListNodeRates(filter map[string]interface{}, limit, offset int) ([]model.RateNode, int64, error) {
    var items []model.RateNode
    var count int64
    q := model.DB.Model(&model.RateNode{})
    if v, ok := filter["region"]; ok && v != "" { q = q.Where("region = ?", v) }
    if v, ok := filter["cp"]; ok && v != "" { q = q.Where("cp = ?", v) }
    if v, ok := filter["settlement_type"]; ok && v != "" { q = q.Where("settlement_type = ?", v) }
    if err := q.Count(&count).Error; err != nil { return nil, 0, err }
    if count == 0 { return []model.RateNode{}, 0, nil }
    if err := q.Order("updated_at DESC").Limit(limit).Offset(offset).Find(&items).Error; err != nil { return nil, 0, err }
    return items, count, nil
}

// UpsertNodeRate 基于唯一键(region,cp,settlement_type)进行插入或更新
func (r *ratesRepository) UpsertNodeRate(rate *model.RateNode) error {
    return model.DB.Clauses(clause.OnConflict{
        Columns:   []clause.Column{{Name: "region"}, {Name: "cp"}, {Name: "settlement_type"}},
        DoUpdates: clause.AssignmentColumns([]string{"cp_fee", "cp_fee_owner_id", "node_construction_fee", "node_construction_fee_owner_id", "rack_fee", "rack_fee_owner_id", "other_fee", "other_fee_owner_id", "updated_at"}),
    }).Create(rate).Error
}

// ListFinalCustomerRates 列表查询最终客户费率
func (r *ratesRepository) ListFinalCustomerRates(filter map[string]interface{}, limit, offset int) ([]model.RateFinalCustomer, int64, error) {
    var items []model.RateFinalCustomer
    var count int64
    q := model.DB.Model(&model.RateFinalCustomer{})
    if v, ok := filter["region"]; ok && v != "" { q = q.Where("region = ?", v) }
    if v, ok := filter["cp"]; ok && v != "" { q = q.Where("cp = ?", v) }
    if v, ok := filter["school_name"]; ok && v != "" { q = q.Where("school_name = ?", v) }
    if v, ok := filter["fee_type"]; ok && v != "" { q = q.Where("fee_type = ?", v) }
    if err := q.Count(&count).Error; err != nil { return nil, 0, err }
    if count == 0 { return []model.RateFinalCustomer{}, 0, nil }
    if err := q.Order("updated_at DESC").Limit(limit).Offset(offset).Find(&items).Error; err != nil { return nil, 0, err }
    return items, count, nil
}

// UpsertFinalCustomerRate 基于唯一键(region,cp,school_name)进行插入或更新
func (r *ratesRepository) UpsertFinalCustomerRate(rate *model.RateFinalCustomer) error {
    return model.DB.Clauses(clause.OnConflict{
        Columns:   []clause.Column{{Name: "region"}, {Name: "cp"}, {Name: "school_name"}},
        DoUpdates: clause.AssignmentColumns([]string{"final_fee", "fee_type", "customer_fee", "customer_fee_owner_id", "network_line_fee", "network_line_fee_owner_id", "node_deduction_fee", "node_deduction_fee_owner_id", "updated_at"}),
    }).Create(rate).Error
}

// InitFinalCustomerRatesFromCustomer 从 rate_customer 初始化/同步到 rate_final_customer（保护 config 不被覆盖）
func (r *ratesRepository) InitFinalCustomerRatesFromCustomer() (int64, error) {
    sql := `
INSERT INTO rate_final_customer
  (region, cp, school_name, fee_type,
   customer_fee, customer_fee_owner_id,
   network_line_fee, network_line_fee_owner_id,
   created_at, updated_at)
SELECT
  rc.region,
  rc.cp,
  COALESCE(rc.school_name, 'not_a_school') AS school_name,
  'auto' AS fee_type,
  rc.customer_fee,
  rc.customer_fee_owner_id,
  rc.network_line_fee,
  rc.network_line_fee_owner_id,
  NOW(), NOW()
FROM rate_customer rc
ON DUPLICATE KEY UPDATE
  fee_type = IF(rate_final_customer.fee_type = 'config', rate_final_customer.fee_type, 'auto'),
  customer_fee = IF(rate_final_customer.fee_type = 'config', rate_final_customer.customer_fee, VALUES(customer_fee)),
  customer_fee_owner_id = IF(rate_final_customer.fee_type = 'config', rate_final_customer.customer_fee_owner_id, VALUES(customer_fee_owner_id)),
  network_line_fee = IF(rate_final_customer.fee_type = 'config', rate_final_customer.network_line_fee, VALUES(network_line_fee)),
  network_line_fee_owner_id = IF(rate_final_customer.fee_type = 'config', rate_final_customer.network_line_fee_owner_id, VALUES(network_line_fee_owner_id)),
  updated_at = NOW();`
    res := model.DB.Exec(sql)
    return res.RowsAffected, res.Error
}

// RefreshFinalCustomerRates 按公式重算 final_fee（仅 auto）
// 公式：final_fee = COALESCE(customer_fee,0) + COALESCE(network_line_fee,0) - COALESCE(node_deduction_fee,0)
func (r *ratesRepository) RefreshFinalCustomerRates() (int64, error) {
    res := model.DB.Model(&model.RateFinalCustomer{}).
        Where("fee_type = ?", "auto").
        Updates(map[string]interface{}{
            "final_fee": gorm.Expr("COALESCE(customer_fee,0) + COALESCE(network_line_fee,0) - COALESCE(node_deduction_fee,0)"),
            "updated_at": gorm.Expr("NOW()"),
        })
    return res.RowsAffected, res.Error
}
