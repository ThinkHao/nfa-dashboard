package repository

import (
    "errors"
    "nfa-dashboard/internal/model"
)

// EntitiesRepository 业务对象仓储接口
// 对应表：business_entities
// 仅提供基础的列表、创建、更新、删除
// 按照现有仓储风格实现分页与条件过滤

type EntitiesRepository interface {
    List(filter map[string]interface{}, limit, offset int) ([]model.BusinessEntity, int64, error)
    Create(e *model.BusinessEntity) (*model.BusinessEntity, error)
    Update(id uint64, fields map[string]interface{}) error
    Delete(id uint64) error
}

type entitiesRepository struct{}

func NewEntitiesRepository() EntitiesRepository { return &entitiesRepository{} }

func (r *entitiesRepository) List(filter map[string]interface{}, limit, offset int) ([]model.BusinessEntity, int64, error) {
    var (
        items []model.BusinessEntity
        total int64
    )
    q := model.DB.Model(&model.BusinessEntity{})
    // 支持按ID集合过滤
    if v, ok := filter["ids"]; ok {
        // 期望为切片类型（[]uint64/[]int64/[]int/[]interface{}）
        q = q.Where("id IN ?", v)
    }
    if v, ok := filter["entity_type"]; ok && v != "" { q = q.Where("entity_type = ?", v) }
    if v, ok := filter["entity_name"]; ok && v != "" { q = q.Where("entity_name LIKE ?", "%"+v.(string)+"%") }
    if err := q.Count(&total).Error; err != nil { return nil, 0, err }
    if total == 0 { return []model.BusinessEntity{}, 0, nil }
    if limit > 0 { q = q.Limit(limit) }
    if offset > 0 { q = q.Offset(offset) }
    if err := q.Order("updated_at DESC").Find(&items).Error; err != nil { return nil, 0, err }
    return items, total, nil
}

func (r *entitiesRepository) Create(e *model.BusinessEntity) (*model.BusinessEntity, error) {
    if e == nil { return nil, errors.New("nil entity") }
    if err := model.DB.Create(e).Error; err != nil { return nil, err }
    return e, nil
}

func (r *entitiesRepository) Update(id uint64, fields map[string]interface{}) error {
    if id == 0 { return errors.New("invalid id") }
    if len(fields) == 0 { return nil }
    return model.DB.Model(&model.BusinessEntity{}).Where("id = ?", id).Updates(fields).Error
}

func (r *entitiesRepository) Delete(id uint64) error {
    if id == 0 { return errors.New("invalid id") }
    return model.DB.Where("id = ?", id).Delete(&model.BusinessEntity{}).Error
}
