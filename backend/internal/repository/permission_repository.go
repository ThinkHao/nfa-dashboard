package repository

import (
	"nfa-dashboard/internal/model"
)

type PermissionRepository interface {
	List(page, pageSize int) ([]model.Permission, int64, error)
	FindByIDs(ids []uint64) ([]model.Permission, error)
	// ListFiltered supports optional keyword search on code/name/description
	ListFiltered(page, pageSize int, keyword string) ([]model.Permission, int64, error)
	// CRUD & helpers
	Create(p *model.Permission) error
	GetByID(id uint64) (model.Permission, error)
	GetByCode(code string) (model.Permission, error)
	Update(p *model.Permission) error
	Disable(id uint64) error
	Enable(id uint64) error
}

func (r *permissionRepository) FindByIDs(ids []uint64) ([]model.Permission, error) {
	out := make([]model.Permission, 0)
	if len(ids) == 0 { return out, nil }
	if err := model.DB.Where("id IN ?", ids).Find(&out).Error; err != nil { return nil, err }
	return out, nil
}

type permissionRepository struct{}

func NewPermissionRepository() PermissionRepository { return &permissionRepository{} }

func (r *permissionRepository) List(page, pageSize int) ([]model.Permission, int64, error) {
	var (
		items []model.Permission
		total int64
	)
	q := model.DB.Model(&model.Permission{}).Where("enabled = ?", true)
	if err := q.Count(&total).Error; err != nil { return nil, 0, err }
	if page > 0 && pageSize > 0 {
		offset := (page - 1) * pageSize
		q = q.Offset(offset).Limit(pageSize)
	}
	if err := q.Order("id DESC").Find(&items).Error; err != nil { return nil, 0, err }
	return items, total, nil
}

func (r *permissionRepository) ListFiltered(page, pageSize int, keyword string) ([]model.Permission, int64, error) {
	var (
		items []model.Permission
		total int64
	)
	q := model.DB.Model(&model.Permission{}).Where("enabled = ?", true)
	if kw := keyword; kw != "" {
		like := "%" + kw + "%"
		q = q.Where("code LIKE ? OR name LIKE ? OR description LIKE ?", like, like, like)
	}
	if err := q.Count(&total).Error; err != nil { return nil, 0, err }
	if page > 0 && pageSize > 0 {
		offset := (page - 1) * pageSize
		q = q.Offset(offset).Limit(pageSize)
	}
	if err := q.Order("id DESC").Find(&items).Error; err != nil { return nil, 0, err }
	return items, total, nil
}

func (r *permissionRepository) Create(p *model.Permission) error {
	return model.DB.Create(p).Error
}

func (r *permissionRepository) GetByID(id uint64) (model.Permission, error) {
	var p model.Permission
	err := model.DB.Where("id = ?", id).First(&p).Error
	return p, err
}

func (r *permissionRepository) GetByCode(code string) (model.Permission, error) {
	var p model.Permission
	err := model.DB.Where("code = ?", code).First(&p).Error
	return p, err
}

func (r *permissionRepository) Update(p *model.Permission) error {
	return model.DB.Model(&model.Permission{}).
		Where("id = ?", p.ID).
		Updates(map[string]interface{}{
			"name":        p.Name,
			"description": p.Description,
		}).Error
}

func (r *permissionRepository) Disable(id uint64) error {
	return model.DB.Model(&model.Permission{}).
		Where("id = ?", id).
		Update("enabled", false).Error
}

func (r *permissionRepository) Enable(id uint64) error {
	return model.DB.Model(&model.Permission{}).
		Where("id = ?", id).
		Update("enabled", true).Error
}
