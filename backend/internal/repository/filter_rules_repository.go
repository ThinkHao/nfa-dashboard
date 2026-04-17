package repository

import (
	"errors"

	"nfa-dashboard/internal/model"
)

// FilterRulesRepository 管理 rate_customer_filter_rules 的持久化
type FilterRulesRepository interface {
	List(filter map[string]interface{}, limit, offset int) ([]model.RateCustomerFilterRule, int64, error)
	ListEnabled() ([]model.RateCustomerFilterRule, error)
	ListDistinctCustomerRegions() ([]string, error)
	ListDistinctCustomerCPs() ([]string, error)
	Create(rule *model.RateCustomerFilterRule) (*model.RateCustomerFilterRule, error)
	Update(id uint64, updates map[string]interface{}) error
	Delete(id uint64) error
	UpdatePriority(id uint64, priority int) error
	SetEnabled(id uint64, enabled bool) error
	GetMatchSummary(rule model.RateCustomerFilterRule, previewLimit int) (int64, []string, error)
}

type filterRulesRepository struct{}

func NewFilterRulesRepository() FilterRulesRepository { return &filterRulesRepository{} }

func (r *filterRulesRepository) ListDistinctCustomerRegions() ([]string, error) {
	return (&ratesRepository{}).ListDistinctCustomerRegions()
}

func (r *filterRulesRepository) ListDistinctCustomerCPs() ([]string, error) {
	return (&ratesRepository{}).ListDistinctCustomerCPs()
}

func (r *filterRulesRepository) List(filter map[string]interface{}, limit, offset int) ([]model.RateCustomerFilterRule, int64, error) {
	var (
		items []model.RateCustomerFilterRule
		total int64
	)
	q := model.DB.Model(&model.RateCustomerFilterRule{})
	if v, ok := filter["name"]; ok && v != "" {
		q = q.Where("name LIKE ?", "%"+v.(string)+"%")
	}
	if v, ok := filter["enabled"]; ok {
		q = q.Where("enabled = ?", v)
	}
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if total == 0 {
		return []model.RateCustomerFilterRule{}, 0, nil
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

func (r *filterRulesRepository) ListEnabled() ([]model.RateCustomerFilterRule, error) {
	items := make([]model.RateCustomerFilterRule, 0)
	err := model.DB.
		Model(&model.RateCustomerFilterRule{}).
		Where("enabled = ?", true).
		Order("priority ASC").
		Order("updated_at DESC").
		Find(&items).Error
	return items, err
}

func (r *filterRulesRepository) Create(rule *model.RateCustomerFilterRule) (*model.RateCustomerFilterRule, error) {
	if rule == nil {
		return nil, errors.New("nil rule")
	}
	if err := model.DB.Create(rule).Error; err != nil {
		return nil, err
	}
	return rule, nil
}

func (r *filterRulesRepository) Update(id uint64, updates map[string]interface{}) error {
	if id == 0 {
		return errors.New("invalid id")
	}
	if len(updates) == 0 {
		return nil
	}
	delete(updates, "enabled")
	delete(updates, "priority")
	return model.DB.Model(&model.RateCustomerFilterRule{}).Where("id = ?", id).Updates(updates).Error
}

func (r *filterRulesRepository) Delete(id uint64) error {
	if id == 0 {
		return errors.New("invalid id")
	}
	return model.DB.Where("id = ?", id).Delete(&model.RateCustomerFilterRule{}).Error
}

func (r *filterRulesRepository) UpdatePriority(id uint64, priority int) error {
	if id == 0 {
		return errors.New("invalid id")
	}
	return model.DB.Model(&model.RateCustomerFilterRule{}).Where("id = ?", id).Update("priority", priority).Error
}

func (r *filterRulesRepository) SetEnabled(id uint64, enabled bool) error {
	if id == 0 {
		return errors.New("invalid id")
	}
	return model.DB.Model(&model.RateCustomerFilterRule{}).Where("id = ?", id).Update("enabled", enabled).Error
}

func (r *filterRulesRepository) GetMatchSummary(rule model.RateCustomerFilterRule, previewLimit int) (int64, []string, error) {
	if previewLimit <= 0 {
		previewLimit = 20
	}
	q := model.DB.Model(&model.RateCustomer{})
	q = applySingleFilterRuleScope(q, "rate_customer", rule)
	q = q.Where("school_name IS NOT NULL AND school_name <> ''")

	var total int64
	if err := q.Distinct("school_name").Count(&total).Error; err != nil {
		return 0, nil, err
	}
	if total == 0 {
		return 0, []string{}, nil
	}

	names := make([]string, 0, previewLimit)
	if err := q.
		Distinct("school_name").
		Order("school_name ASC").
		Limit(previewLimit).
		Pluck("school_name", &names).Error; err != nil {
		return 0, nil, err
	}
	return total, names, nil
}
