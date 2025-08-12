package repository

import (
	"errors"
	"nfa-dashboard/internal/model"
	"gorm.io/gorm"
)

type UserRepository interface {
	GetByUsername(username string) (*model.User, error)
	GetByID(id uint64) (*model.User, error)
	GetUserRoles(userID uint64) ([]model.Role, error)
	GetUserPermissions(userID uint64) ([]model.Permission, error)
	List(username string, status *int8, page, pageSize int) ([]model.User, int64, error)
	Create(u *model.User) (*model.User, error)
	SetRoles(userID uint64, roleIDs []uint64) error
	UpdateStatus(userID uint64, status int8) error
	Exists(id uint64) (bool, error)
}

type userRepository struct{}

func NewUserRepository() UserRepository { return &userRepository{} }

func (r *userRepository) GetByUsername(username string) (*model.User, error) {
	var u model.User
	if err := model.DB.Where("username = ? AND status = 1", username).First(&u).Error; err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *userRepository) GetByID(id uint64) (*model.User, error) {
	var u model.User
	if err := model.DB.Where("id = ? AND status = 1", id).First(&u).Error; err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *userRepository) GetUserRoles(userID uint64) ([]model.Role, error) {
	var roles []model.Role
	if userID == 0 {
		return nil, errors.New("invalid userID")
	}
	tx := model.DB.Table("roles r").
		Select("r.*").
		Joins("JOIN user_roles ur ON ur.role_id = r.id").
		Where("ur.user_id = ?", userID)
	if err := tx.Find(&roles).Error; err != nil {
		return nil, err
	}
	return roles, nil
}

func (r *userRepository) GetUserPermissions(userID uint64) ([]model.Permission, error) {
	var perms []model.Permission
	if userID == 0 {
		return nil, errors.New("invalid userID")
	}
	tx := model.DB.Table("permissions p").
		Select("p.*").
		Joins("JOIN role_permissions rp ON rp.permission_id = p.id").
		Joins("JOIN user_roles ur ON ur.role_id = rp.role_id").
		Where("ur.user_id = ?", userID)
	if err := tx.Distinct().Find(&perms).Error; err != nil {
		return nil, err
	}
	return perms, nil
}

// List users with optional filters and pagination
func (r *userRepository) List(username string, status *int8, page, pageSize int) ([]model.User, int64, error) {
	var (
		items []model.User
		total int64
	)
	q := model.DB.Model(&model.User{})
	if username != "" { q = q.Where("username LIKE ?", "%"+username+"%") }
	if status != nil { q = q.Where("status = ?", *status) }
	if err := q.Count(&total).Error; err != nil { return nil, 0, err }
	if page > 0 && pageSize > 0 {
		offset := (page - 1) * pageSize
		q = q.Offset(offset).Limit(pageSize)
	}
	if err := q.Order("id DESC").Find(&items).Error; err != nil { return nil, 0, err }
	return items, total, nil
}

// Create inserts a new user record
func (r *userRepository) Create(u *model.User) (*model.User, error) {
	if u == nil { return nil, errors.New("nil user") }
	if err := model.DB.Create(u).Error; err != nil { return nil, err }
	return u, nil
}

// SetRoles replaces user's roles with given roleIDs
func (r *userRepository) SetRoles(userID uint64, roleIDs []uint64) error {
	if userID == 0 { return errors.New("invalid userID") }
	return model.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("user_id = ?", userID).Delete(&model.UserRole{}).Error; err != nil { return err }
		if len(roleIDs) == 0 { return nil }
		batch := make([]model.UserRole, 0, len(roleIDs))
		for _, rid := range roleIDs {
			if rid == 0 { continue }
			batch = append(batch, model.UserRole{UserID: userID, RoleID: rid})
		}
		if len(batch) == 0 { return nil }
		return tx.Create(&batch).Error
	})
}

func (r *userRepository) UpdateStatus(userID uint64, status int8) error {
	if userID == 0 { return errors.New("invalid userID") }
	return model.DB.Model(&model.User{}).Where("id = ?", userID).Update("status", status).Error
}

func (r *userRepository) Exists(id uint64) (bool, error) {
	if id == 0 { return false, nil }
	var cnt int64
	if err := model.DB.Model(&model.User{}).Where("id = ?", id).Count(&cnt).Error; err != nil { return false, err }
	return cnt > 0, nil
}
