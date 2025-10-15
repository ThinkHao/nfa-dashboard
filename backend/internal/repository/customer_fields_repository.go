package repository

import (
    "errors"
    "nfa-dashboard/internal/model"
)

// CustomerFieldsRepository 管理 rate_customer_custom_field_defs 的持久化
// 提供列表、创建、更新、删除

type CustomerFieldsRepository interface {
    List(filter map[string]interface{}, limit, offset int) ([]model.RateCustomerCustomFieldDef, int64, error)
    Create(def *model.RateCustomerCustomFieldDef) (*model.RateCustomerCustomFieldDef, error)
    Update(id uint64, updates map[string]interface{}) error
    Delete(id uint64) error
    // GetByID 根据主键查询
    GetByID(id uint64) (*model.RateCustomerCustomFieldDef, error)
    // ExistsByFieldKey 判断 field_key 是否已存在（精确匹配）
    ExistsByFieldKey(fieldKey string) (bool, error)
}

type customerFieldsRepository struct{}

func NewCustomerFieldsRepository() CustomerFieldsRepository { return &customerFieldsRepository{} }

func (r *customerFieldsRepository) List(filter map[string]interface{}, limit, offset int) ([]model.RateCustomerCustomFieldDef, int64, error) {
    var (
        items []model.RateCustomerCustomFieldDef
        total int64
    )
    q := model.DB.Model(&model.RateCustomerCustomFieldDef{})
    if v, ok := filter["field_key"]; ok && v != "" {
        // field_key 支持前缀匹配更友好
        s := v.(string)
        q = q.Where("field_key LIKE ?", s+"%")
    }
    if v, ok := filter["label"]; ok && v != "" {
        s := v.(string)
        q = q.Where("label LIKE ?", "%"+s+"%")
    }
    if v, ok := filter["data_type"]; ok && v != "" {
        q = q.Where("data_type = ?", v)
    }
    if v, ok := filter["enabled"]; ok {
        q = q.Where("enabled = ?", v)
    }
    if err := q.Count(&total).Error; err != nil { return nil, 0, err }
    if total == 0 { return []model.RateCustomerCustomFieldDef{}, 0, nil }
    if limit > 0 { q = q.Limit(limit) }
    if offset > 0 { q = q.Offset(offset) }
    if err := q.Order("updated_at DESC").Find(&items).Error; err != nil { return nil, 0, err }
    return items, total, nil
}

func (r *customerFieldsRepository) Create(def *model.RateCustomerCustomFieldDef) (*model.RateCustomerCustomFieldDef, error) {
    if def == nil { return nil, errors.New("nil def") }
    if err := model.DB.Create(def).Error; err != nil { return nil, err }
    return def, nil
}

func (r *customerFieldsRepository) Update(id uint64, updates map[string]interface{}) error {
    if id == 0 { return errors.New("invalid id") }
    if len(updates) == 0 { return nil }
    // 禁止修改 field_key
    delete(updates, "field_key")
    return model.DB.Model(&model.RateCustomerCustomFieldDef{}).Where("id = ?", id).Updates(updates).Error
}

func (r *customerFieldsRepository) Delete(id uint64) error {
    if id == 0 { return errors.New("invalid id") }
    return model.DB.Where("id = ?", id).Delete(&model.RateCustomerCustomFieldDef{}).Error
}

func (r *customerFieldsRepository) GetByID(id uint64) (*model.RateCustomerCustomFieldDef, error) {
    if id == 0 { return nil, errors.New("invalid id") }
    var m model.RateCustomerCustomFieldDef
    if err := model.DB.Where("id = ?", id).First(&m).Error; err != nil { return nil, err }
    return &m, nil
}

func (r *customerFieldsRepository) ExistsByFieldKey(fieldKey string) (bool, error) {
    if fieldKey == "" { return false, errors.New("empty fieldKey") }
    var c int64
    if err := model.DB.Model(&model.RateCustomerCustomFieldDef{}).Where("field_key = ?", fieldKey).Count(&c).Error; err != nil { return false, err }
    return c > 0, nil
}


