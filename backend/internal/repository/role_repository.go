package repository

import (
	"errors"
	"gorm.io/gorm"
	"nfa-dashboard/internal/model"
)

type RoleRepository interface {
	List(page, pageSize int) ([]model.Role, int64, error)
	Create(name string, description *string) (*model.Role, error)
	Update(id uint64, name *string, description *string) error
	Delete(id uint64) error
	GetPermissions(roleID uint64) ([]model.Permission, error)
	SetPermissions(roleID uint64, permissionIDs []uint64) error
	Exists(id uint64) (bool, error)
	FindByIDs(ids []uint64) ([]model.Role, error)
}

type roleRepository struct{}

func NewRoleRepository() RoleRepository { return &roleRepository{} }

func (r *roleRepository) List(page, pageSize int) ([]model.Role, int64, error) {
	var (
		items []model.Role
		total int64
	)
	q := model.DB.Model(&model.Role{})
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if page > 0 && pageSize > 0 {
		offset := (page - 1) * pageSize
		q = q.Offset(offset).Limit(pageSize)
	}
	if err := q.Order("id DESC").Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

func (r *roleRepository) Create(name string, description *string) (*model.Role, error) {
	role := &model.Role{Name: name, Description: description}
	if err := model.DB.Create(role).Error; err != nil {
		return nil, err
	}
	return role, nil
}

func (r *roleRepository) Update(id uint64, name *string, description *string) error {
	if id == 0 { return errors.New("invalid id") }
	updates := map[string]interface{}{}
	if name != nil { updates["name"] = *name }
	if description != nil { updates["description"] = *description }
	if len(updates) == 0 { return nil }
	return model.DB.Model(&model.Role{}).Where("id = ?", id).Updates(updates).Error
}

func (r *roleRepository) Delete(id uint64) error {
	if id == 0 { return errors.New("invalid id") }
	// FK with ON DELETE CASCADE will cleanup mappings
	return model.DB.Delete(&model.Role{}, id).Error
}

func (r *roleRepository) GetPermissions(roleID uint64) ([]model.Permission, error) {
	if roleID == 0 { return nil, errors.New("invalid roleID") }
	var perms []model.Permission
	tx := model.DB.Table("permissions p").
		Select("p.*").
		Joins("JOIN role_permissions rp ON rp.permission_id = p.id").
		Where("rp.role_id = ?", roleID)
	if err := tx.Find(&perms).Error; err != nil { return nil, err }
	return perms, nil
}

func (r *roleRepository) SetPermissions(roleID uint64, permissionIDs []uint64) error {
	if roleID == 0 { return errors.New("invalid roleID") }
	return model.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("role_id = ?", roleID).Delete(&model.RolePermission{}).Error; err != nil {
			return err
		}
		if len(permissionIDs) == 0 { return nil }
		batch := make([]model.RolePermission, 0, len(permissionIDs))
		for _, pid := range permissionIDs {
			if pid == 0 { continue }
			batch = append(batch, model.RolePermission{RoleID: roleID, PermissionID: pid})
		}
		if len(batch) == 0 { return nil }
		return tx.Create(&batch).Error
	})
}

func (r *roleRepository) Exists(id uint64) (bool, error) {
	if id == 0 { return false, nil }
	var cnt int64
	if err := model.DB.Model(&model.Role{}).Where("id = ?", id).Count(&cnt).Error; err != nil { return false, err }
	return cnt > 0, nil
}

func (r *roleRepository) FindByIDs(ids []uint64) ([]model.Role, error) {
	roles := make([]model.Role, 0)
	if len(ids) == 0 { return roles, nil }
	if err := model.DB.Where("id IN ?", ids).Find(&roles).Error; err != nil { return nil, err }
	return roles, nil
}
