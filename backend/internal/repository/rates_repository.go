package repository

import (
	"fmt"
	"nfa-dashboard/internal/model"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// RatesRepository 费率仓储接口
type RatesRepository interface {
	// 客户业务费率
	ListCustomerRates(filter map[string]interface{}, limit, offset int) ([]model.RateCustomer, int64, error)
	ListCustomerRateKeys(filter map[string]interface{}) (map[string]struct{}, error)
	UpsertCustomerRate(rate *model.RateCustomer) error
	CreateCustomerRateIfMissing(rate *model.RateCustomer) (bool, error)
	UpdateCustomerByID(id uint64, updates map[string]interface{}) error

	// 节点业务费率
	ListNodeRates(filter map[string]interface{}, limit, offset int) ([]model.RateNode, int64, error)
	UpsertNodeRate(rate *model.RateNode) error
	ListNodeSettlementGroups(filter map[string]interface{}, limit, offset int) ([]model.EDCNodeSettlementGroup, int64, error)
	SaveNodeSettlementGroup(group *model.EDCNodeSettlementGroup, memberIDs []uint64) error
	DisableNodeSettlementGroup(id uint64) error
	ListEnabledNodeSettlementGroups() ([]model.EDCNodeSettlementGroup, error)

	// 最终客户费率
	ListFinalCustomerRates(filter map[string]interface{}, limit, offset int) ([]model.RateFinalCustomer, int64, error)
	UpsertFinalCustomerRate(rate *model.RateFinalCustomer) error

	// 最终节点费率
	ListFinalNodeRates(filter map[string]interface{}, limit, offset int) ([]model.RateFinalNode, int64, error)
	UpsertFinalNodeRate(rate *model.RateFinalNode) error
	SyncFinalNodeRateFromNode(rate *model.RateNode) (bool, error)
	InitFinalNodeRatesFromNode() (int64, error)
	RefreshFinalNodeRates() (int64, error)
	ListAllFinalNodeRates() ([]model.RateFinalNode, error)

	// 初始化最终客户费率（从 rate_customer 同步，保护 config 记录）
	InitFinalCustomerRatesFromCustomer() (int64, error)

	// 刷新最终客户费率（重算 final_fee，仅 auto）
	RefreshFinalCustomerRates() (int64, error)

	// 清理无效的最终客户费率（仅 auto；任一关键费率字段为空）
	CleanupInvalidFinalCustomerRates() (int64, error)

	// 根据 region+cp+school_name 获取单条最终客户费率
	GetFinalCustomerRate(region, cp, schoolName string) (*model.RateFinalCustomer, error)
	ListDistinctCustomerRegions() ([]string, error)
	ListDistinctCustomerCPs() ([]string, error)
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

func normalizeDistinctOptionValues(items []string) []string {
	out := make([]string, 0, len(items))
	seen := make(map[string]struct{}, len(items))
	for _, item := range items {
		value := strings.TrimSpace(item)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

func (r *ratesRepository) listDistinctCustomerColumn(column string) ([]string, error) {
	values := make([]string, 0)
	if err := model.DB.Model(&model.RateCustomer{}).
		Where(column+" IS NOT NULL").
		Where("TRIM("+column+") <> ''").
		Distinct(column).
		Order(column+" ASC").
		Pluck(column, &values).Error; err != nil {
		return nil, err
	}
	return normalizeDistinctOptionValues(values), nil
}

func (r *ratesRepository) ListDistinctCustomerRegions() ([]string, error) {
	return r.listDistinctCustomerColumn("region")
}

func (r *ratesRepository) ListDistinctCustomerCPs() ([]string, error) {
	return r.listDistinctCustomerColumn("cp")
}

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
	if v, ok := filter["exclude_filtered"]; ok {
		if b, ok2 := v.(bool); ok2 && b {
			q = excludeFilteredCustomerRates(q, "rate_customer")
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

func (r *ratesRepository) ListCustomerRateKeys(filter map[string]interface{}) (map[string]struct{}, error) {
	type rateKeyRow struct {
		Region     string
		CP         string
		SchoolName *string
	}
	var rows []rateKeyRow
	q := model.DB.Model(&model.RateCustomer{}).Select("region, cp, school_name")
	if v, ok := filter["region"]; ok && v != "" {
		q = q.Where("region = ?", v)
	}
	if v, ok := filter["cp"]; ok && v != "" {
		q = q.Where("cp = ?", v)
	}
	if v, ok := filter["school_name"]; ok && v != "" {
		q = q.Where("school_name = ?", v)
	}
	if err := q.Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make(map[string]struct{}, len(rows))
	for _, row := range rows {
		out[rateCustomerKey(row.Region, row.CP, derefRateCustomerSchool(row.SchoolName))] = struct{}{}
	}
	return out, nil
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
		"channel_rate":              rate.ChannelRate,
		"channel_owner_user_id":     rate.ChannelOwnerUserID,
		"start_at":                  rate.StartAt,
		"increment_start_at":        rate.IncrementStartAt,
		"stock_ratio":               rate.StockRatio,
		"increment_ratio":           rate.IncrementRatio,
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

func (r *ratesRepository) CreateCustomerRateIfMissing(rate *model.RateCustomer) (bool, error) {
	if rate == nil {
		return false, gorm.ErrInvalidData
	}
	if rate.FeeMode == "" {
		rate.FeeMode = "auto"
	}
	res := model.DB.Clauses(clause.Insert{Modifier: "IGNORE"}).Create(rate)
	if res.Error != nil {
		return false, res.Error
	}
	return res.RowsAffected > 0, nil
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

func rateCustomerKey(region, cp, schoolName string) string {
	return region + "\x00" + cp + "\x00" + schoolName
}

func derefRateCustomerSchool(p *string) string {
	if p == nil {
		return ""
	}
	return *p
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
		q = q.Where("settlement_type = ? OR settlement_mode = ?", v, v)
	}
	if v, ok := filter["settlement_mode"]; ok && v != "" {
		q = q.Where("settlement_mode = ?", v)
	}
	if v, ok := filter["unit_base"]; ok && v != "" {
		q = q.Where("unit_base = ?", v)
	}
	if v, ok := filter["entity_id"]; ok && v != "" {
		q = q.Where("entity_id = ?", v)
	}
	if v, ok := filter["billing_subject_type"]; ok && v != "" {
		q = q.Where("billing_subject_type = ?", v)
	}
	if v, ok := filter["billing_subject_id"]; ok && v != "" {
		q = q.Where("billing_subject_id = ?", v)
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

// UpsertNodeRate 基于 entity_id 或 region/cp/settlement_mode/unit_base 进行插入或更新
func (r *ratesRepository) UpsertNodeRate(rate *model.RateNode) error {
	if rate.SettlementMode == "" {
		rate.SettlementMode = rate.SettlementType
	}
	if rate.SettlementMode == "" {
		rate.SettlementMode = "daily_95_avg"
	}
	if rate.SettlementType == "" {
		rate.SettlementType = rate.SettlementMode
	}
	if rate.UnitBase == 0 {
		rate.UnitBase = 1000
	}
	normalizeRateNodeBillingSubject(rate)
	updates := map[string]interface{}{
		"display_name":                   rate.DisplayName,
		"billing_subject_type":           rate.BillingSubjectType,
		"billing_subject_id":             rate.BillingSubjectID,
		"billing_display_name":           rate.BillingDisplayName,
		"region":                         rate.Region,
		"cp":                             rate.CP,
		"settlement_type":                rate.SettlementType,
		"settlement_mode":                rate.SettlementMode,
		"unit_base":                      rate.UnitBase,
		"cp_fee":                         rate.CPFee,
		"cp_fee_owner_id":                rate.CPFeeOwnerID,
		"node_construction_fee":          rate.NodeConstructionFee,
		"node_construction_fee_owner_id": rate.NodeConstructionFeeOwnerID,
		"rack_fee":                       rate.RackFee,
		"rack_fee_owner_id":              rate.RackFeeOwnerID,
		"other_fee":                      rate.OtherFee,
		"other_fee_owner_id":             rate.OtherFeeOwnerID,
		"updated_at":                     gorm.Expr("NOW()"),
	}
	q := model.DB.Model(&model.RateNode{})
	if rate.BillingSubjectID != nil && *rate.BillingSubjectID > 0 {
		q = q.Where("billing_subject_type = ? AND billing_subject_id = ? AND settlement_mode = ? AND unit_base = ?", rate.BillingSubjectType, *rate.BillingSubjectID, rate.SettlementMode, rate.UnitBase)
	} else {
		q = q.Where("billing_subject_type = ? AND billing_subject_id IS NULL AND region = ? AND cp = ? AND settlement_mode = ? AND unit_base = ?", rate.BillingSubjectType, rate.Region, rate.CP, rate.SettlementMode, rate.UnitBase)
	}
	var existing model.RateNode
	if err := q.Limit(1).Find(&existing).Error; err != nil {
		return err
	}
	if existing.ID > 0 {
		return model.DB.Model(&model.RateNode{}).Where("id = ?", existing.ID).Updates(updates).Error
	}
	return model.DB.Create(rate).Error
}

func normalizeRateNodeBillingSubject(rate *model.RateNode) {
	if rate.BillingSubjectType != "group" {
		rate.BillingSubjectType = "node"
	}
	if rate.BillingSubjectType == "node" {
		rate.BillingSubjectID = rate.EntityID
		if rate.BillingDisplayName == nil {
			rate.BillingDisplayName = rate.DisplayName
		}
		return
	}
	rate.EntityID = nil
	if rate.BillingDisplayName == nil {
		rate.BillingDisplayName = rate.DisplayName
	}
	if rate.DisplayName == nil {
		rate.DisplayName = rate.BillingDisplayName
	}
}

func normalizeRateFinalNodeBillingSubject(rate *model.RateFinalNode) {
	if rate.BillingSubjectType != "group" {
		rate.BillingSubjectType = "node"
	}
	if rate.BillingSubjectType == "node" {
		rate.BillingSubjectID = rate.EntityID
		if rate.BillingDisplayName == "" {
			rate.BillingDisplayName = rate.DisplayName
		}
		return
	}
	rate.EntityID = nil
	if rate.BillingDisplayName == "" {
		rate.BillingDisplayName = rate.DisplayName
	}
	if rate.DisplayName == "" {
		rate.DisplayName = rate.BillingDisplayName
	}
}

func (r *ratesRepository) ListNodeSettlementGroups(filter map[string]interface{}, limit, offset int) ([]model.EDCNodeSettlementGroup, int64, error) {
	var items []model.EDCNodeSettlementGroup
	var count int64
	q := model.DB.Model(&model.EDCNodeSettlementGroup{})
	if v, ok := filter["region"]; ok && v != "" {
		q = q.Where("region = ?", v)
	}
	if v, ok := filter["cp"]; ok && v != "" {
		q = q.Where("cp = ?", v)
	}
	if v, ok := filter["enabled"]; ok {
		if b, ok2 := v.(bool); ok2 {
			q = q.Where("enabled = ?", b)
		}
	}
	if v, ok := filter["group_name"]; ok && v != "" {
		q = q.Where("group_name LIKE ?", "%"+v.(string)+"%")
	}
	if err := q.Count(&count).Error; err != nil {
		return nil, 0, err
	}
	if count == 0 {
		return []model.EDCNodeSettlementGroup{}, 0, nil
	}
	err := q.Preload("Members").Preload("Members.Entity").
		Order("enabled DESC, region ASC, cp ASC, group_name ASC").
		Limit(limit).Offset(offset).Find(&items).Error
	return items, count, err
}

func (r *ratesRepository) SaveNodeSettlementGroup(group *model.EDCNodeSettlementGroup, memberIDs []uint64) error {
	if group == nil {
		return gorm.ErrInvalidData
	}
	seen := map[uint64]struct{}{}
	cleanIDs := make([]uint64, 0, len(memberIDs))
	for _, id := range memberIDs {
		if id == 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			return fmt.Errorf("同一节点不能重复加入结算分组")
		}
		seen[id] = struct{}{}
		cleanIDs = append(cleanIDs, id)
	}
	return model.DB.Transaction(func(tx *gorm.DB) error {
		if len(cleanIDs) > 0 {
			var count int64
			if err := tx.Model(&model.EDCEntity{}).
				Where("id IN ? AND enabled = ? AND is_backup = ?", cleanIDs, true, false).
				Count(&count).Error; err != nil {
				return err
			}
			if count != int64(len(cleanIDs)) {
				return fmt.Errorf("分组成员必须是启用的非备份 EDC 节点")
			}
			conflictQ := tx.Model(&model.EDCNodeSettlementGroupMember{}).Where("entity_id IN ?", cleanIDs)
			if group.ID > 0 {
				conflictQ = conflictQ.Where("group_id <> ?", group.ID)
			}
			var conflictCount int64
			if err := conflictQ.Count(&conflictCount).Error; err != nil {
				return err
			}
			if conflictCount > 0 {
				return fmt.Errorf("同一节点只能属于一个结算分组")
			}
		}
		if group.ID > 0 {
			if err := tx.Model(&model.EDCNodeSettlementGroup{}).Where("id = ?", group.ID).Updates(map[string]interface{}{
				"group_name": group.GroupName,
				"region":     group.Region,
				"cp":         group.CP,
				"enabled":    group.Enabled,
				"remark":     group.Remark,
				"updated_at": gorm.Expr("NOW()"),
			}).Error; err != nil {
				return err
			}
		} else if err := tx.Create(group).Error; err != nil {
			return err
		}
		if err := tx.Where("group_id = ?", group.ID).Delete(&model.EDCNodeSettlementGroupMember{}).Error; err != nil {
			return err
		}
		if len(cleanIDs) == 0 {
			return nil
		}
		members := make([]model.EDCNodeSettlementGroupMember, 0, len(cleanIDs))
		for _, id := range cleanIDs {
			members = append(members, model.EDCNodeSettlementGroupMember{GroupID: group.ID, EntityID: id})
		}
		return tx.Create(&members).Error
	})
}

func (r *ratesRepository) DisableNodeSettlementGroup(id uint64) error {
	return model.DB.Model(&model.EDCNodeSettlementGroup{}).Where("id = ?", id).Updates(map[string]interface{}{
		"enabled":    false,
		"updated_at": gorm.Expr("NOW()"),
	}).Error
}

func (r *ratesRepository) ListEnabledNodeSettlementGroups() ([]model.EDCNodeSettlementGroup, error) {
	var groups []model.EDCNodeSettlementGroup
	err := model.DB.Preload("Members").Where("enabled = ?", true).
		Order("region ASC, cp ASC, group_name ASC").Find(&groups).Error
	return groups, err
}

func (r *ratesRepository) ListFinalNodeRates(filter map[string]interface{}, limit, offset int) ([]model.RateFinalNode, int64, error) {
	var items []model.RateFinalNode
	var count int64
	q := model.DB.Model(&model.RateFinalNode{})
	if v, ok := filter["region"]; ok && v != "" {
		q = q.Where("region = ?", v)
	}
	if v, ok := filter["cp"]; ok && v != "" {
		q = q.Where("cp = ?", v)
	}
	if v, ok := filter["display_name"]; ok && v != "" {
		q = q.Where("display_name LIKE ?", "%"+v.(string)+"%")
	}
	if v, ok := filter["settlement_mode"]; ok && v != "" {
		q = q.Where("settlement_mode = ?", v)
	}
	if v, ok := filter["unit_base"]; ok && v != "" {
		q = q.Where("unit_base = ?", v)
	}
	if v, ok := filter["billing_subject_type"]; ok && v != "" {
		q = q.Where("billing_subject_type = ?", v)
	}
	if v, ok := filter["billing_subject_id"]; ok && v != "" {
		q = q.Where("billing_subject_id = ?", v)
	}
	if err := q.Count(&count).Error; err != nil {
		return nil, 0, err
	}
	if count == 0 {
		return []model.RateFinalNode{}, 0, nil
	}
	if err := q.Order("updated_at DESC").Limit(limit).Offset(offset).Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, count, nil
}

func (r *ratesRepository) ListAllFinalNodeRates() ([]model.RateFinalNode, error) {
	var items []model.RateFinalNode
	err := model.DB.Order("entity_id DESC, region ASC, cp ASC").Find(&items).Error
	return items, err
}

func (r *ratesRepository) UpsertFinalNodeRate(rate *model.RateFinalNode) error {
	if rate.SettlementMode == "" {
		rate.SettlementMode = "daily_95_avg"
	}
	if rate.UnitBase == 0 {
		rate.UnitBase = 1000
	}
	if rate.FeeType == "" {
		rate.FeeType = "config"
	}
	normalizeRateFinalNodeBillingSubject(rate)
	updates := map[string]interface{}{
		"display_name":                   rate.DisplayName,
		"billing_subject_type":           rate.BillingSubjectType,
		"billing_subject_id":             rate.BillingSubjectID,
		"billing_display_name":           rate.BillingDisplayName,
		"region":                         rate.Region,
		"cp":                             rate.CP,
		"settlement_mode":                rate.SettlementMode,
		"unit_base":                      rate.UnitBase,
		"final_fee":                      rate.FinalFee,
		"fee_type":                       rate.FeeType,
		"cp_fee":                         rate.CPFee,
		"cp_fee_owner_id":                rate.CPFeeOwnerID,
		"node_construction_fee":          rate.NodeConstructionFee,
		"node_construction_fee_owner_id": rate.NodeConstructionFeeOwnerID,
		"rack_fee":                       rate.RackFee,
		"rack_fee_owner_id":              rate.RackFeeOwnerID,
		"other_fee":                      rate.OtherFee,
		"other_fee_owner_id":             rate.OtherFeeOwnerID,
		"updated_at":                     gorm.Expr("NOW()"),
	}
	q := model.DB.Model(&model.RateFinalNode{})
	if rate.BillingSubjectID != nil && *rate.BillingSubjectID > 0 {
		q = q.Where("billing_subject_type = ? AND billing_subject_id = ? AND settlement_mode = ? AND unit_base = ?", rate.BillingSubjectType, *rate.BillingSubjectID, rate.SettlementMode, rate.UnitBase)
	} else {
		q = q.Where("billing_subject_type = ? AND billing_subject_id IS NULL AND region = ? AND cp = ? AND settlement_mode = ? AND unit_base = ?", rate.BillingSubjectType, rate.Region, rate.CP, rate.SettlementMode, rate.UnitBase)
	}
	var existing model.RateFinalNode
	if err := q.Limit(1).Find(&existing).Error; err != nil {
		return err
	}
	if existing.ID > 0 {
		return model.DB.Model(&model.RateFinalNode{}).Where("id = ?", existing.ID).Updates(updates).Error
	}
	return model.DB.Create(rate).Error
}

func (r *ratesRepository) SyncFinalNodeRateFromNode(rate *model.RateNode) (bool, error) {
	if rate == nil {
		return false, gorm.ErrInvalidData
	}
	mode := rate.SettlementMode
	if mode == "" {
		mode = rate.SettlementType
	}
	if mode == "" {
		mode = "daily_95_avg"
	}
	base := rate.UnitBase
	if base == 0 {
		base = 1000
	}
	normalizeRateNodeBillingSubject(rate)
	displayName := ""
	if rate.DisplayName != nil {
		displayName = *rate.DisplayName
	}
	item := &model.RateFinalNode{
		EntityID:                   rate.EntityID,
		DisplayName:                displayName,
		BillingSubjectType:         rate.BillingSubjectType,
		BillingSubjectID:           rate.BillingSubjectID,
		BillingDisplayName:         "",
		Region:                     rate.Region,
		CP:                         rate.CP,
		SettlementMode:             mode,
		UnitBase:                   base,
		FinalFee:                   rate.NodeConstructionFee,
		FeeType:                    "auto",
		CPFee:                      rate.CPFee,
		CPFeeOwnerID:               rate.CPFeeOwnerID,
		NodeConstructionFee:        rate.NodeConstructionFee,
		NodeConstructionFeeOwnerID: rate.NodeConstructionFeeOwnerID,
		RackFee:                    rate.RackFee,
		RackFeeOwnerID:             rate.RackFeeOwnerID,
		OtherFee:                   rate.OtherFee,
		OtherFeeOwnerID:            rate.OtherFeeOwnerID,
		LastSyncTime:               ptrTime(time.Now()),
	}
	if rate.BillingDisplayName != nil {
		item.BillingDisplayName = *rate.BillingDisplayName
	}
	if item.BillingDisplayName == "" {
		item.BillingDisplayName = item.DisplayName
	}
	q := model.DB.Model(&model.RateFinalNode{})
	if item.BillingSubjectID != nil && *item.BillingSubjectID > 0 {
		q = q.Where("billing_subject_type = ? AND billing_subject_id = ? AND settlement_mode = ? AND unit_base = ?", item.BillingSubjectType, *item.BillingSubjectID, item.SettlementMode, item.UnitBase)
	} else {
		q = q.Where("billing_subject_type = ? AND billing_subject_id IS NULL AND region = ? AND cp = ? AND settlement_mode = ? AND unit_base = ?", item.BillingSubjectType, item.Region, item.CP, item.SettlementMode, item.UnitBase)
	}
	var existing model.RateFinalNode
	if err := q.Limit(1).Find(&existing).Error; err != nil {
		return false, err
	}
	if existing.ID > 0 {
		if existing.FeeType != "" && existing.FeeType != "auto" {
			return false, nil
		}
		updates := map[string]interface{}{
			"display_name":                   item.DisplayName,
			"billing_subject_type":           item.BillingSubjectType,
			"billing_subject_id":             item.BillingSubjectID,
			"billing_display_name":           item.BillingDisplayName,
			"region":                         item.Region,
			"cp":                             item.CP,
			"final_fee":                      item.FinalFee,
			"fee_type":                       "auto",
			"cp_fee":                         item.CPFee,
			"cp_fee_owner_id":                item.CPFeeOwnerID,
			"node_construction_fee":          item.NodeConstructionFee,
			"node_construction_fee_owner_id": item.NodeConstructionFeeOwnerID,
			"rack_fee":                       item.RackFee,
			"rack_fee_owner_id":              item.RackFeeOwnerID,
			"other_fee":                      item.OtherFee,
			"other_fee_owner_id":             item.OtherFeeOwnerID,
			"last_sync_time":                 gorm.Expr("NOW()"),
			"updated_at":                     gorm.Expr("NOW()"),
		}
		res := model.DB.Model(&model.RateFinalNode{}).Where("id = ?", existing.ID).Updates(updates)
		return res.RowsAffected > 0, res.Error
	}
	if err := model.DB.Create(item).Error; err != nil {
		return false, err
	}
	return true, nil
}

func (r *ratesRepository) InitFinalNodeRatesFromNode() (int64, error) {
	var rates []model.RateNode
	if err := model.DB.Order("entity_id DESC, region ASC, cp ASC").Find(&rates).Error; err != nil {
		return 0, err
	}
	var affected int64
	for _, rn := range rates {
		ok, err := r.SyncFinalNodeRateFromNode(&rn)
		if err != nil {
			return affected, err
		}
		if ok {
			affected++
		}
	}
	return affected, nil
}

func (r *ratesRepository) RefreshFinalNodeRates() (int64, error) {
	var rates []model.RateNode
	if err := model.DB.Find(&rates).Error; err != nil {
		return 0, err
	}
	var affected int64
	for _, rn := range rates {
		ok, err := r.SyncFinalNodeRateFromNode(&rn)
		if err != nil {
			return affected, err
		}
		if ok {
			affected++
		}
	}
	return affected, nil
}

func ptrTime(t time.Time) *time.Time {
	return &t
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
  AND NOT EXISTS (
` + enabledFilterRuleExistsClause("rc") + `
  )
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
  AND rc.network_line_fee IS NOT NULL
  AND NOT EXISTS (
` + enabledFilterRuleExistsClause("rc") + `
  )`
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
  AND rc.network_line_fee IS NOT NULL
  AND NOT EXISTS (
` + enabledFilterRuleExistsClause("rc") + `
  )`
	if err := model.DB.Exec(updateSQL).Error; err != nil {
		return 0, err
	}
	return matched, nil
}
