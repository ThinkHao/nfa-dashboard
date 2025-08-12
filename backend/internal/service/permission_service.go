package service

import (
	"regexp"
	"nfa-dashboard/internal/model"
	"nfa-dashboard/internal/repository"
	"nfa-dashboard/internal/authz"
)

type PermissionService interface {
	List(page, pageSize int) ([]model.Permission, int64, error)
	// ListFiltered supports optional keyword search on code/name/description
	ListFiltered(page, pageSize int, keyword string) ([]model.Permission, int64, error)
	Create(code, name string, description *string) (*model.Permission, error)
	GetByID(id uint64) (model.Permission, error)
	Update(id uint64, name *string, description *string) error
	Disable(id uint64) error
	SyncFromCode() error
}

type permissionService struct{ repo repository.PermissionRepository }

func NewPermissionService(repo repository.PermissionRepository) PermissionService {
	return &permissionService{repo: repo}
}

func (s *permissionService) List(page, pageSize int) ([]model.Permission, int64, error) {
	return s.repo.List(page, pageSize)
}

func (s *permissionService) ListFiltered(page, pageSize int, keyword string) ([]model.Permission, int64, error) {
	return s.repo.ListFiltered(page, pageSize, keyword)
}

var codePattern = regexp.MustCompile(`^[a-z0-9_.:]+$`)

func (s *permissionService) Create(code, name string, description *string) (*model.Permission, error) {
	if code == "" || name == "" { return nil, NewBadRequest("code and name are required") }
	if !codePattern.MatchString(code) { return nil, NewBadRequest("invalid code format") }
	if _, err := s.repo.GetByCode(code); err == nil {
		return nil, NewBadRequest("permission code already exists")
	}
	p := &model.Permission{Code: code, Name: name, Description: description, Enabled: true}
	if err := s.repo.Create(p); err != nil { return nil, err }
	return p, nil
}

func (s *permissionService) GetByID(id uint64) (model.Permission, error) {
	return s.repo.GetByID(id)
}

func (s *permissionService) Update(id uint64, name *string, description *string) error {
	if id == 0 { return NewBadRequest("invalid id") }
	// ensure exists
	p, err := s.repo.GetByID(id)
	if err != nil { return err }
	if name != nil { p.Name = *name }
	if description != nil { p.Description = description }
	return s.repo.Update(&p)
}

func (s *permissionService) Disable(id uint64) error {
	if id == 0 { return NewBadRequest("invalid id") }
	return s.repo.Disable(id)
}

// SyncFromCode upserts builtin permissions defined in code without disabling others.
func (s *permissionService) SyncFromCode() error {
	for _, def := range authz.BuiltinPermissions {
		if def.Code == "" || def.Name == "" { continue }
		if !codePattern.MatchString(def.Code) { continue }
		existing, err := s.repo.GetByCode(def.Code)
		if err == nil {
			// update name/description and enable
			existing.Name = def.Name
			existing.Description = def.Description
			tmp := existing
			if err := s.repo.Update(&tmp); err != nil { return err }
			if !existing.Enabled { if err := s.repo.Enable(existing.ID); err != nil { return err } }
		} else {
			p := &model.Permission{Code: def.Code, Name: def.Name, Description: def.Description, Enabled: true}
			if err := s.repo.Create(p); err != nil { return err }
		}
	}
	return nil
}
