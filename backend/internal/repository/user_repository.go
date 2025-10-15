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
	List(username string, status *int8, roles []string, page, pageSize int) ([]model.User, int64, error)
	FindByIDs(ids []uint64) ([]model.User, error)
	Create(u *model.User) (*model.User, error)
	SetRoles(userID uint64, roleIDs []uint64) error
	UpdateStatus(userID uint64, status int8) error
	UpdateAlias(userID uint64, alias *string) error
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
func (r *userRepository) List(username string, status *int8, roles []string, page, pageSize int) ([]model.User, int64, error) {
    var (
        items []model.User
        total int64
    )
    // 基础查询（显式使用 users 表别名，避免列名歧义）
    base := model.DB.Table("users")
    if len(roles) > 0 {
        base = base.Joins("JOIN user_roles ur ON ur.user_id = users.id").Joins("JOIN roles r ON r.id = ur.role_id").Where("r.name IN ?", roles)
    }
    if username != "" { base = base.Where("users.username LIKE ?", "%"+username+"%") }
    if status != nil { base = base.Where("users.status = ?", *status) }

    // 先计算总数：按去重的用户ID计数（保持在同一 base 上，确保已设置表）
    if err := base.Distinct("users.id").Count(&total).Error; err != nil {
        return nil, 0, err
    }

    // 再查询明细：DISTINCT + users.*，避免只查出 id 的问题
    q := base.Select("users.*").Distinct()
    if page > 0 && pageSize > 0 {
        offset := (page - 1) * pageSize
        q = q.Offset(offset).Limit(pageSize)
    }
    if err := q.Order("users.id DESC").Find(&items).Error; err != nil { return nil, 0, err }
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

func (r *userRepository) UpdateAlias(userID uint64, alias *string) error {
	if userID == 0 { return errors.New("invalid userID") }
	// Using single-column Update will set NULL when alias is nil
	return model.DB.Model(&model.User{}).Where("id = ?", userID).Update("alias", alias).Error
}

func (r *userRepository) Exists(id uint64) (bool, error) {
	if id == 0 { return false, nil }
	var cnt int64
	if err := model.DB.Model(&model.User{}).Where("id = ?", id).Count(&cnt).Error; err != nil { return false, err }
	return cnt > 0, nil
}

// FindByIDs returns users by ids
func (r *userRepository) FindByIDs(ids []uint64) ([]model.User, error) {
    if len(ids) == 0 { return []model.User{}, nil }
    var users []model.User
    // 使用全新的会话并显式列出字段，避免上游 SELECT/DISTINCT 泄漏上下文造成字段缺失
    tx := model.DB.Session(&gorm.Session{NewDB: true}).Debug().
        Table("users").
        Select("users.id, users.username, users.alias, users.email, users.phone, users.status, users.last_login_at, users.created_at, users.updated_at").
        Where("users.id IN ?", ids)
    if err := tx.Find(&users).Error; err != nil { return nil, err }
    return users, nil
}
