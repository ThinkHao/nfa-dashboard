package service

import (
    "nfa-dashboard/internal/model"
    "nfa-dashboard/internal/repository"
)

// EntitiesService 业务对象服务接口
// 负责入参校验与过滤构建，调用仓储层

type EntitiesService interface {
    List(entityType, entityName string, page, pageSize int) ([]model.BusinessEntity, int64, error)
    ListByIDs(ids []uint64) ([]model.BusinessEntity, int64, error)
    Create(entityType, entityName string, contactInfo *string) (*model.BusinessEntity, error)
    Update(id uint64, entityType, entityName, contactInfo *string) error
    Delete(id uint64) error
}

type entitiesService struct{
    repo   repository.EntitiesRepository
    btRepo repository.BusinessTypeRepository
}

func NewEntitiesService(repo repository.EntitiesRepository, btRepo repository.BusinessTypeRepository) EntitiesService {
    return &entitiesService{repo: repo, btRepo: btRepo}
}

func (s *entitiesService) List(entityType, entityName string, page, pageSize int) ([]model.BusinessEntity, int64, error) {
    filter := map[string]interface{}{}
    if entityType != "" { filter["entity_type"] = entityType }
    if entityName != "" { filter["entity_name"] = entityName }
    if page <= 0 { page = 1 }
    if pageSize <= 0 { pageSize = 10 }
    limit := pageSize
    offset := (page - 1) * pageSize
    return s.repo.List(filter, limit, offset)
}

func (s *entitiesService) ListByIDs(ids []uint64) ([]model.BusinessEntity, int64, error) {
    if len(ids) == 0 { return []model.BusinessEntity{}, 0, nil }
    filter := map[string]interface{}{"ids": ids}
    // 不限制分页，返回全部匹配项
    return s.repo.List(filter, 0, 0)
}

func (s *entitiesService) Create(entityType, entityName string, contactInfo *string) (*model.BusinessEntity, error) {
    if entityType == "" { return nil, NewBadRequest("entity_type is required") }
    if entityName == "" { return nil, NewBadRequest("entity_name is required") }
    // 校验业务类型是否存在且启用
    ok, err := s.btRepo.ExistsEnabled(entityType)
    if err != nil { return nil, err }
    if !ok { return nil, NewBadRequest("invalid entity_type: not found or disabled") }
    e := &model.BusinessEntity{EntityType: entityType, EntityName: entityName, ContactInfo: contactInfo}
    return s.repo.Create(e)
}

func (s *entitiesService) Update(id uint64, entityType, entityName, contactInfo *string) error {
    if id == 0 { return NewBadRequest("invalid id") }
    fields := map[string]interface{}{}
    if entityType != nil {
        // 校验业务类型是否存在且启用
        ok, err := s.btRepo.ExistsEnabled(*entityType)
        if err != nil { return err }
        if !ok { return NewBadRequest("invalid entity_type: not found or disabled") }
        fields["entity_type"] = *entityType
    }
    if entityName != nil { fields["entity_name"] = *entityName }
    if contactInfo != nil { fields["contact_info"] = *contactInfo }
    if len(fields) == 0 { return NewBadRequest("no fields to update") }
    return s.repo.Update(id, fields)
}

func (s *entitiesService) Delete(id uint64) error {
    if id == 0 { return NewBadRequest("invalid id") }
    return s.repo.Delete(id)
}
