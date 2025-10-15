package repository

import (
    "errors"
    "nfa-dashboard/internal/model"
)

// BusinessTypeRepository 业务类型仓储接口
// 提供基础的列表、创建、更新、删除与存在性检查

type BusinessTypeRepository interface {
    List(filter map[string]interface{}, limit, offset int) ([]model.BusinessType, int64, error)
    Create(bt *model.BusinessType) (*model.BusinessType, error)
    Update(id uint64, fields map[string]interface{}) error
    Delete(id uint64) error
    GetByCode(code string) (*model.BusinessType, error)
    ExistsEnabled(code string) (bool, error)
}

type businessTypeRepository struct{}

func NewBusinessTypeRepository() BusinessTypeRepository { return &businessTypeRepository{} }

func (r *businessTypeRepository) List(filter map[string]interface{}, limit, offset int) ([]model.BusinessType, int64, error) {
    var (
        items []model.BusinessType
        total int64
    )
    q := model.DB.Model(&model.BusinessType{})
    if v, ok := filter["code"]; ok && v != "" { q = q.Where("code LIKE ?", "%"+v.(string)+"%") }
    if v, ok := filter["name"]; ok && v != "" { q = q.Where("name LIKE ?", "%"+v.(string)+"%") }
    if v, ok := filter["enabled"]; ok { q = q.Where("enabled = ?", v) }
    if err := q.Count(&total).Error; err != nil { return nil, 0, err }
    if total == 0 { return []model.BusinessType{}, 0, nil }
    if limit > 0 { q = q.Limit(limit) }
    if offset > 0 { q = q.Offset(offset) }
    if err := q.Order("updated_at DESC").Find(&items).Error; err != nil { return nil, 0, err }
    return items, total, nil
}

func (r *businessTypeRepository) Create(bt *model.BusinessType) (*model.BusinessType, error) {
    if bt == nil { return nil, errors.New("nil business type") }
    if err := model.DB.Create(bt).Error; err != nil { return nil, err }
    return bt, nil
}

func (r *businessTypeRepository) Update(id uint64, fields map[string]interface{}) error {
    if id == 0 { return errors.New("invalid id") }
    if len(fields) == 0 { return nil }
    // 防止 code 被误改
    delete(fields, "code")
    return model.DB.Model(&model.BusinessType{}).Where("id = ?", id).Updates(fields).Error
}

func (r *businessTypeRepository) Delete(id uint64) error {
    if id == 0 { return errors.New("invalid id") }
    return model.DB.Where("id = ?", id).Delete(&model.BusinessType{}).Error
}

func (r *businessTypeRepository) GetByCode(code string) (*model.BusinessType, error) {
    var bt model.BusinessType
    if err := model.DB.Where("code = ?", code).First(&bt).Error; err != nil { return nil, err }
    return &bt, nil
}

func (r *businessTypeRepository) ExistsEnabled(code string) (bool, error) {
    var cnt int64
    if err := model.DB.Model(&model.BusinessType{}).Where("code = ? AND enabled = ?", code, true).Count(&cnt).Error; err != nil { return false, err }
    return cnt > 0, nil
}
