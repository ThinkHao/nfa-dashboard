package repository

import (
	"nfa-dashboard/internal/model"
)

// RateDiscountRepository 管理 rate_discount_rule 与 rate_discount_rule_item 的持久化

type RateDiscountRepository interface {
	ListRules(filter map[string]interface{}, limit, offset int) ([]model.RateDiscountRule, int64, error)
	GetRuleByID(id uint64) (*model.RateDiscountRule, error)
	CreateRule(rule *model.RateDiscountRule) (*model.RateDiscountRule, error)
	UpdateRule(id uint64, updates map[string]interface{}) error
	DeleteRule(id uint64) error

	ListItemsByRuleID(ruleID uint64) ([]model.RateDiscountRuleItem, error)
	ReplaceItems(ruleID uint64, items []model.RateDiscountRuleItem) error
}

type rateDiscountRepository struct{}

func NewRateDiscountRepository() RateDiscountRepository { return &rateDiscountRepository{} }

func (r *rateDiscountRepository) ListRules(filter map[string]interface{}, limit, offset int) ([]model.RateDiscountRule, int64, error) {
	var items []model.RateDiscountRule
	var total int64
	q := model.DB.Model(&model.RateDiscountRule{})
	if v, ok := filter["name"]; ok && v != "" {
		q = q.Where("name LIKE ?", "%"+v.(string)+"%")
	}
	if v, ok := filter["scope_type"]; ok && v != "" {
		q = q.Where("scope_type = ?", v)
	}
	if v, ok := filter["enabled"]; ok {
		q = q.Where("enabled = ?", v)
	}
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if total == 0 {
		return []model.RateDiscountRule{}, 0, nil
	}
	if limit > 0 {
		q = q.Limit(limit)
	}
	if offset > 0 {
		q = q.Offset(offset)
	}
	if err := q.Order("priority ASC").Order("updated_at DESC").Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

func (r *rateDiscountRepository) GetRuleByID(id uint64) (*model.RateDiscountRule, error) {
	if id == 0 {
		return nil, nil
	}
	var rule model.RateDiscountRule
	if err := model.DB.First(&rule, id).Error; err != nil {
		return nil, err
	}
	return &rule, nil
}

func (r *rateDiscountRepository) CreateRule(rule *model.RateDiscountRule) (*model.RateDiscountRule, error) {
	if err := model.DB.Create(rule).Error; err != nil {
		return nil, err
	}
	return rule, nil
}

func (r *rateDiscountRepository) UpdateRule(id uint64, updates map[string]interface{}) error {
	if id == 0 {
		return nil
	}
	if len(updates) == 0 {
		return nil
	}
	return model.DB.Model(&model.RateDiscountRule{}).Where("id = ?", id).Updates(updates).Error
}

func (r *rateDiscountRepository) DeleteRule(id uint64) error {
	if id == 0 {
		return nil
	}
	// 直接物理删除，明细表由外键级联删除
	return model.DB.Where("id = ?", id).Delete(&model.RateDiscountRule{}).Error
}

func (r *rateDiscountRepository) ListItemsByRuleID(ruleID uint64) ([]model.RateDiscountRuleItem, error) {
	var items []model.RateDiscountRuleItem
	if ruleID == 0 {
		return []model.RateDiscountRuleItem{}, nil
	}
	if err := model.DB.Where("rule_id = ?", ruleID).Order("from_year ASC").Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

func (r *rateDiscountRepository) ReplaceItems(ruleID uint64, items []model.RateDiscountRuleItem) error {
	if ruleID == 0 {
		return nil
	}
	tx := model.DB.Begin()
	if err := tx.Where("rule_id = ?", ruleID).Delete(&model.RateDiscountRuleItem{}).Error; err != nil {
		tx.Rollback()
		return err
	}
	if len(items) > 0 {
		for i := range items {
			items[i].RuleID = ruleID
		}
		if err := tx.Create(&items).Error; err != nil {
			tx.Rollback()
			return err
		}
	}
	return tx.Commit().Error
}
