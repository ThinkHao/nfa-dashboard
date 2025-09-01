package repository

import (
    "errors"
    "nfa-dashboard/internal/model"
)

// SyncRulesRepository 管理 rate_customer_sync_rules 的持久化

type SyncRulesRepository interface {
    List(filter map[string]interface{}, limit, offset int) ([]model.RateCustomerSyncRule, int64, error)
    Create(rule *model.RateCustomerSyncRule) (*model.RateCustomerSyncRule, error)
    Update(id uint64, updates map[string]interface{}) error
    Delete(id uint64) error
    UpdatePriority(id uint64, priority int) error
    SetEnabled(id uint64, enabled bool) error
}

type syncRulesRepository struct{}

func NewSyncRulesRepository() SyncRulesRepository { return &syncRulesRepository{} }

func (r *syncRulesRepository) List(filter map[string]interface{}, limit, offset int) ([]model.RateCustomerSyncRule, int64, error) {
    var (
        items []model.RateCustomerSyncRule
        total int64
    )
    q := model.DB.Model(&model.RateCustomerSyncRule{})
    if v, ok := filter["name"]; ok && v != "" {
        s := v.(string)
        q = q.Where("name LIKE ?", "%"+s+"%")
    }
    if v, ok := filter["enabled"]; ok {
        q = q.Where("enabled = ?", v)
    }
    if v, ok := filter["overwrite_strategy"]; ok && v != "" {
        q = q.Where("overwrite_strategy = ?", v)
    }
    if v, ok := filter["priority_gte"]; ok {
        q = q.Where("priority >= ?", v)
    }
    if v, ok := filter["priority_lte"]; ok {
        q = q.Where("priority <= ?", v)
    }
    if err := q.Count(&total).Error; err != nil { return nil, 0, err }
    if total == 0 { return []model.RateCustomerSyncRule{}, 0, nil }
    if limit > 0 { q = q.Limit(limit) }
    if offset > 0 { q = q.Offset(offset) }
    if err := q.Order("priority ASC").Order("updated_at DESC").Find(&items).Error; err != nil { return nil, 0, err }
    return items, total, nil
}

func (r *syncRulesRepository) Create(rule *model.RateCustomerSyncRule) (*model.RateCustomerSyncRule, error) {
    if rule == nil { return nil, errors.New("nil rule") }
    if err := model.DB.Create(rule).Error; err != nil { return nil, err }
    return rule, nil
}

func (r *syncRulesRepository) Update(id uint64, updates map[string]interface{}) error {
    if id == 0 { return errors.New("invalid id") }
    if len(updates) == 0 { return nil }
    // 禁止通过通用更新修改 enabled 与 priority
    delete(updates, "enabled")
    delete(updates, "priority")
    return model.DB.Model(&model.RateCustomerSyncRule{}).Where("id = ?", id).Updates(updates).Error
}

func (r *syncRulesRepository) Delete(id uint64) error {
    if id == 0 { return errors.New("invalid id") }
    return model.DB.Where("id = ?", id).Delete(&model.RateCustomerSyncRule{}).Error
}

func (r *syncRulesRepository) UpdatePriority(id uint64, priority int) error {
    if id == 0 { return errors.New("invalid id") }
    return model.DB.Model(&model.RateCustomerSyncRule{}).Where("id = ?", id).Update("priority", priority).Error
}

func (r *syncRulesRepository) SetEnabled(id uint64, enabled bool) error {
    if id == 0 { return errors.New("invalid id") }
    return model.DB.Model(&model.RateCustomerSyncRule{}).Where("id = ?", id).Update("enabled", enabled).Error
}

