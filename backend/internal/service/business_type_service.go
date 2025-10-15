package service

import (
    "nfa-dashboard/internal/model"
    "nfa-dashboard/internal/repository"
)

// BusinessTypeService 提供业务类型的应用逻辑
// 主要是参数校验与调用仓储

type BusinessTypeService interface {
    List(code, name string, enabled *bool, page, pageSize int) ([]model.BusinessType, int64, error)
    Create(code, name string, description *string, enabled *bool) (*model.BusinessType, error)
    Update(id uint64, name, description *string, enabled *bool) error
    Delete(id uint64) error
}

type businessTypeService struct { repo repository.BusinessTypeRepository }

func NewBusinessTypeService(repo repository.BusinessTypeRepository) BusinessTypeService { return &businessTypeService{repo: repo} }

func (s *businessTypeService) List(code, name string, enabled *bool, page, pageSize int) ([]model.BusinessType, int64, error) {
    if page <= 0 { page = 1 }
    if pageSize <= 0 { pageSize = 10 }
    filter := map[string]interface{}{}
    if code != "" { filter["code"] = code }
    if name != "" { filter["name"] = name }
    if enabled != nil { filter["enabled"] = *enabled }
    limit := pageSize
    offset := (page - 1) * pageSize
    return s.repo.List(filter, limit, offset)
}

func (s *businessTypeService) Create(code, name string, description *string, enabled *bool) (*model.BusinessType, error) {
    if code == "" { return nil, NewBadRequest("code is required") }
    if name == "" { return nil, NewBadRequest("name is required") }
    bt := &model.BusinessType{ Code: code, Name: name, Description: description }
    if enabled != nil { bt.Enabled = *enabled }
    return s.repo.Create(bt)
}

func (s *businessTypeService) Update(id uint64, name, description *string, enabled *bool) error {
    if id == 0 { return NewBadRequest("invalid id") }
    fields := map[string]interface{}{}
    if name != nil { fields["name"] = *name }
    if description != nil { fields["description"] = *description }
    if enabled != nil { fields["enabled"] = *enabled }
    if len(fields) == 0 { return NewBadRequest("no fields to update") }
    return s.repo.Update(id, fields)
}

func (s *businessTypeService) Delete(id uint64) error {
    if id == 0 { return NewBadRequest("invalid id") }
    return s.repo.Delete(id)
}
