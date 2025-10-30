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

	// 清理无效的最终客户费率（仅 auto；任一关键费率字段为空）
	CleanupInvalidFinalCustomerRates() (int64, error)

	// 根据 region+cp+school_name 获取单条最终客户费率
	GetFinalCustomerRate(region, cp, schoolName string) (*model.RateFinalCustomer, error)
}

// CleanupInvalidFinalCustomerRates 清理无效数据：
// 仅针对 fee_type='auto' 且 (final_fee IS NULL OR customer_fee IS NULL OR network_line_fee IS NULL)
// 不强制 node_deduction_fee 非空，因其可选
func (r *ratesRepository) CleanupInvalidFinalCustomerRates() (int64, error) {
    sql := `DELETE FROM rate_final_customer
WHERE fee_type = 'auto'
  AND (final_fee IS NULL OR customer_fee IS NULL OR network_line_fee IS NULL)`
    res := model.DB.Exec(sql)
    return res.RowsAffected, res.Error
}

type ratesRepository struct{}

func NewRatesRepository() RatesRepository { return &ratesRepository{} }

// ListCustomerRates 列表查询客户业务费率
func (r *ratesRepository) ListCustomerRates(filter map[string]interface{}, limit, offset int) ([]model.RateCustomer, int64, error) {
	var items []model.RateCustomer
	var count int64
	q := model.DB.Model(&model.RateCustomer{})
	if v, ok := filter["region"]; ok && v != "" {
		q = q.Where("region = ?", v)
	}
	if v, ok := filter["cp"]; ok && v != "" {
		q = q.Where("cp = ?", v)
	}
	if v, ok := filter["school_name"]; ok && v != "" {
		q = q.Where("school_name = ?", v)
	}
	if v, ok := filter["settlement_ready"]; ok {
		if b, ok2 := v.(bool); ok2 {
			if b {
				q = q.Where("school_name IS NOT NULL AND school_name <> ''").
					Where("customer_fee IS NOT NULL").
					Where("network_line_fee IS NOT NULL").
					Where("general_fee IS NOT NULL")
			} else {
				q = q.Where("(school_name IS NULL OR school_name = '' OR customer_fee IS NULL OR network_line_fee IS NULL OR general_fee IS NULL)")
			}
		}
	}
	if err := q.Count(&count).Error; err != nil {
		return nil, 0, err
	}
	if count == 0 {
		return []model.RateCustomer{}, 0, nil
	}
	if err := q.Order("updated_at DESC").Limit(limit).Offset(offset).Find(&items).Error; err != nil {
        return nil, 0, err
    }
    return items, count, nil
}

// UpsertCustomerRate 基于唯一键(region,cp,school_name)进行插入或更新
func (r *ratesRepository) UpsertCustomerRate(rate *model.RateCustomer) error {
    updates := map[string]interface{}{
        "customer_fee":              rate.CustomerFee,
        "network_line_fee":          rate.NetworkLineFee,
        "general_fee":               rate.GeneralFee,
        "general_fee_owner_id":      rate.GeneralFeeOwnerID,
        "customer_fee_owner_id":     rate.CustomerFeeOwnerID,
        "network_line_fee_owner_id": rate.NetworkLineFeeOwnerID,
        "extra":                     rate.Extra,
        "updated_at":                gorm.Expr("NOW()"),
    }
    if rate.FeeMode != "" {
        updates["fee_mode"] = rate.FeeMode
    }
    // 确保新插入时 fee_mode 有默认值（auto）
    if rate.FeeMode == "" {
        rate.FeeMode = "auto"
    }
    return model.DB.Clauses(clause.OnConflict{
        Columns:   []clause.Column{{Name: "region"}, {Name: "cp"}, {Name: "school_name"}},
        DoUpdates: clause.Assignments(updates),
    }).Create(rate).Error
}

// UpdateCustomerByID 基于主键进行局部字段更新
func (r *ratesRepository) UpdateCustomerByID(id uint64, updates map[string]interface{}) error {
    if id == 0 {
        return gorm.ErrInvalidData
    }
    if len(updates) == 0 {
        return nil
    }
    return model.DB.Model(&model.RateCustomer{}).Where("id = ?", id).Updates(updates).Error
}

// ListNodeRates 列表查询节点业务费率
func (r *ratesRepository) ListNodeRates(filter map[string]interface{}, limit, offset int) ([]model.RateNode, int64, error) {
	var items []model.RateNode
	var count int64
	q := model.DB.Model(&model.RateNode{})
	if v, ok := filter["region"]; ok && v != "" {
		q = q.Where("region = ?", v)
	}
	if v, ok := filter["cp"]; ok && v != "" {
		q = q.Where("cp = ?", v)
	}
	if v, ok := filter["settlement_type"]; ok && v != "" {
		q = q.Where("settlement_type = ?", v)
	}
	if err := q.Count(&count).Error; err != nil {
		return nil, 0, err
	}
	if count == 0 {
		return []model.RateNode{}, 0, nil
	}
	if err := q.Order("updated_at DESC").Limit(limit).Offset(offset).Find(&items).Error; err != nil {
		return nil, 0, err
	}
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
	if v, ok := filter["region"]; ok && v != "" {
		q = q.Where("region = ?", v)
	}
	if v, ok := filter["cp"]; ok && v != "" {
		q = q.Where("cp = ?", v)
	}
	if v, ok := filter["school_name"]; ok && v != "" {
		q = q.Where("school_name = ?", v)
	}
	if v, ok := filter["fee_type"]; ok && v != "" {
		q = q.Where("fee_type = ?", v)
	}
	if err := q.Count(&count).Error; err != nil {
		return nil, 0, err
	}
	if count == 0 {
		return []model.RateFinalCustomer{}, 0, nil
	}
	if err := q.Order("updated_at DESC").Limit(limit).Offset(offset).Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, count, nil
}

// UpsertFinalCustomerRate 基于唯一键(region,cp,school_name)进行插入或更新
func (r *ratesRepository) UpsertFinalCustomerRate(rate *model.RateFinalCustomer) error {
	return model.DB.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "region"}, {Name: "cp"}, {Name: "school_name"}},
		DoUpdates: clause.AssignmentColumns([]string{"final_fee", "fee_type", "customer_fee", "customer_fee_owner_id", "network_line_fee", "network_line_fee_owner_id", "node_deduction_fee", "node_deduction_fee_owner_id", "updated_at"}),
	}).Create(rate).Error
}

// GetFinalCustomerRate 根据 region+cp+school_name 获取单条最终客户费率
func (r *ratesRepository) GetFinalCustomerRate(region, cp, schoolName string) (*model.RateFinalCustomer, error) {
	if region == "" || cp == "" || schoolName == "" {
		return nil, gorm.ErrRecordNotFound
	}
	var item model.RateFinalCustomer
	if err := model.DB.Where("region = ? AND cp = ? AND school_name = ?", region, cp, schoolName).Limit(1).Find(&item).Error; err != nil {
		return nil, err
	}
	if item.ID == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	return &item, nil
}

// InitFinalCustomerRatesFromCustomer 从 rate_customer 初始化/同步到 rate_final_customer（保护 config 不被覆盖）
func (r *ratesRepository) InitFinalCustomerRatesFromCustomer() (int64, error) {
    sql := `
INSERT INTO rate_final_customer
  (region, cp, school_name, fee_type,
   customer_fee, customer_fee_owner_id,
   network_line_fee, network_line_fee_owner_id,
   node_deduction_fee, node_deduction_fee_owner_id,
   created_at, updated_at)
SELECT
  rc.region,
  rc.cp,
  rc.school_name,
  'auto' AS fee_type,
  rc.customer_fee,
  rc.customer_fee_owner_id,
  rc.network_line_fee,
  rc.network_line_fee_owner_id,
  rc.general_fee AS node_deduction_fee,
  rc.general_fee_owner_id AS node_deduction_fee_owner_id,
  NOW(), NOW()
FROM rate_customer rc
WHERE rc.school_name IS NOT NULL AND rc.school_name <> ''
  AND rc.customer_fee IS NOT NULL
  AND rc.network_line_fee IS NOT NULL
ON DUPLICATE KEY UPDATE
  fee_type = IF(rate_final_customer.fee_type = 'config', rate_final_customer.fee_type, 'auto'),
  customer_fee = IF(rate_final_customer.fee_type = 'config', rate_final_customer.customer_fee, VALUES(customer_fee)),
  customer_fee_owner_id = IF(rate_final_customer.fee_type = 'config', rate_final_customer.customer_fee_owner_id, VALUES(customer_fee_owner_id)),
  network_line_fee = IF(rate_final_customer.fee_type = 'config', rate_final_customer.network_line_fee, VALUES(network_line_fee)),
  network_line_fee_owner_id = IF(rate_final_customer.fee_type = 'config', rate_final_customer.network_line_fee_owner_id, VALUES(network_line_fee_owner_id)),
  node_deduction_fee = IF(rate_final_customer.fee_type = 'config', rate_final_customer.node_deduction_fee, VALUES(node_deduction_fee)),
  node_deduction_fee_owner_id = IF(rate_final_customer.fee_type = 'config', rate_final_customer.node_deduction_fee_owner_id, VALUES(node_deduction_fee_owner_id)),
  updated_at = NOW();`
    res := model.DB.Exec(sql)
    return res.RowsAffected, res.Error
}

// RefreshFinalCustomerRates 按公式重算 final_fee（仅 auto）
// 公式：final_fee = COALESCE(customer_fee,0) + COALESCE(network_line_fee,0) - COALESCE(node_deduction_fee,0)
func (r *ratesRepository) RefreshFinalCustomerRates() (int64, error) {
    // 仅针对“参与结算”的记录刷新（与 rate_customer 条件保持一致）：
    // 条件：rc.school_name 非空 且 rc.customer_fee 与 rc.network_line_fee 均非 NULL；并且仅刷新 fee_type='auto'
    // 先统计匹配行数（使用 JOIN），避免因值未变化导致 RowsAffected=0 的错觉
    var matched int64
    countSQL := `
SELECT COUNT(*)
FROM rate_final_customer fc
JOIN rate_customer rc
  ON fc.region = rc.region AND fc.cp = rc.cp AND fc.school_name = rc.school_name
WHERE (fc.fee_type = 'auto' OR fc.fee_type IS NULL OR fc.fee_type = '')
  AND rc.school_name IS NOT NULL AND rc.school_name <> ''
  AND rc.customer_fee IS NOT NULL
  AND rc.network_line_fee IS NOT NULL`
    if err := model.DB.Raw(countSQL).Scan(&matched).Error; err != nil {
        return 0, err
    }
    // 执行更新计算，仅更新匹配的“参与结算 + auto”记录
    updateSQL := `
UPDATE rate_final_customer fc
JOIN rate_customer rc
  ON fc.region = rc.region AND fc.cp = rc.cp AND fc.school_name = rc.school_name
SET 
    fc.customer_fee = rc.customer_fee,
    fc.customer_fee_owner_id = rc.customer_fee_owner_id,
    fc.network_line_fee = rc.network_line_fee,
    fc.network_line_fee_owner_id = rc.network_line_fee_owner_id,
    fc.node_deduction_fee = rc.general_fee,
    fc.node_deduction_fee_owner_id = rc.general_fee_owner_id,
    fc.final_fee = COALESCE(rc.customer_fee,0) + COALESCE(rc.network_line_fee,0) - COALESCE(rc.general_fee,0),
    fc.updated_at = NOW()
WHERE (fc.fee_type = 'auto' OR fc.fee_type IS NULL OR fc.fee_type = '')
  AND rc.school_name IS NOT NULL AND rc.school_name <> ''
  AND rc.customer_fee IS NOT NULL
  AND rc.network_line_fee IS NOT NULL`
    if err := model.DB.Exec(updateSQL).Error; err != nil {
        return 0, err
    }
    return matched, nil
}
