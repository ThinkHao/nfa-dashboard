package service

import (
	"nfa-dashboard/internal/model"
	"nfa-dashboard/internal/repository"
)

type RoleService interface {
	List(page, pageSize int) ([]model.Role, int64, error)
	Create(name string, description *string) (*model.Role, error)
	Update(id uint64, name *string, description *string) error
	Delete(id uint64) error
	GetPermissions(roleID uint64) ([]model.Permission, error)
	SetPermissions(roleID uint64, permissionIDs []uint64) error
}

type roleService struct{
    roleRepo repository.RoleRepository
    permRepo repository.PermissionRepository
}

func NewRoleService(roleRepo repository.RoleRepository, permRepo repository.PermissionRepository) RoleService { 
    return &roleService{roleRepo: roleRepo, permRepo: permRepo} 
}

func (s *roleService) List(page, pageSize int) ([]model.Role, int64, error) {
    return s.roleRepo.List(page, pageSize)
}
func (s *roleService) Create(name string, description *string) (*model.Role, error) {
    return s.roleRepo.Create(name, description)
}
func (s *roleService) Update(id uint64, name *string, description *string) error {
    return s.roleRepo.Update(id, name, description)
}
func (s *roleService) Delete(id uint64) error { return s.roleRepo.Delete(id) }
func (s *roleService) GetPermissions(roleID uint64) ([]model.Permission, error) {
    return s.roleRepo.GetPermissions(roleID)
}
func (s *roleService) SetPermissions(roleID uint64, permissionIDs []uint64) error {
    // validate role exists
    exists, err := s.roleRepo.Exists(roleID)
    if err != nil { return err }
    if !exists { return NewBadRequestf("role %d not found", roleID) }

    // dedup permissionIDs
    uniq := make([]uint64, 0, len(permissionIDs))
    seen := make(map[uint64]struct{}, len(permissionIDs))
    for _, id := range permissionIDs {
        if id == 0 { continue }
        if _, ok := seen[id]; ok { continue }
        seen[id] = struct{}{}
        uniq = append(uniq, id)
    }
    if len(uniq) == 0 { return s.roleRepo.SetPermissions(roleID, nil) }

    // fetch permissions to verify existence
    perms, err := s.permRepo.FindByIDs(uniq)
    if err != nil { return err }
    if len(perms) != len(uniq) {
        // find missing ids
        present := make(map[uint64]struct{}, len(perms))
        for _, p := range perms { present[p.ID] = struct{}{} }
        missing := make([]uint64, 0)
        for _, id := range uniq { if _, ok := present[id]; !ok { missing = append(missing, id) } }
        return NewBadRequestf("permissions not found: %v", missing)
    }
    return s.roleRepo.SetPermissions(roleID, uniq)
}
