package service

import (
	"context"
	"strings"
	"time"

	"nfa-dashboard/internal/model"
	"nfa-dashboard/internal/repository"
)

type EDCNFAComparisonService interface {
	ListGroups(ctx context.Context, allowedEntityIDs []uint64) ([]model.EDCNFAComparisonGroup, error)
	ListMappings(ctx context.Context) ([]model.EDCNFAComparisonGroup, error)
	ListMappingEntities(ctx context.Context) ([]model.EDCEntity, error)
	ListMappingSchoolNames(ctx context.Context, srcRegion, cp string) ([]string, error)
	CreateMapping(ctx context.Context, input model.EDCNFAComparisonGroupInput) (model.EDCNFAComparisonGroup, error)
	UpdateMapping(ctx context.Context, id uint64, input model.EDCNFAComparisonGroupInput) (model.EDCNFAComparisonGroup, error)
	SetMappingEnabled(ctx context.Context, id uint64, enabled bool) error
	GetComparison(ctx context.Context, filter model.EDCNFAComparisonFilter) ([]model.EDCNFAComparisonPoint, error)
}

type edcNFAComparisonService struct {
	repo repository.EDCNFAComparisonRepository
}

func (s *edcNFAComparisonService) mappingRepo() (repository.EDCNFAComparisonMappingRepository, error) {
	repo, ok := s.repo.(repository.EDCNFAComparisonMappingRepository)
	if !ok {
		return nil, NewBadRequest("EDC/NFA 映射仓储未启用")
	}
	return repo, nil
}

func NewEDCNFAComparisonService(repo repository.EDCNFAComparisonRepository) EDCNFAComparisonService {
	return &edcNFAComparisonService{repo: repo}
}

func (s *edcNFAComparisonService) ListGroups(ctx context.Context, allowedEntityIDs []uint64) ([]model.EDCNFAComparisonGroup, error) {
	return s.repo.ListGroups(ctx, allowedEntityIDs)
}

func (s *edcNFAComparisonService) ListMappings(ctx context.Context) ([]model.EDCNFAComparisonGroup, error) {
	repo, err := s.mappingRepo()
	if err != nil {
		return nil, err
	}
	return repo.ListMappings(ctx)
}

func (s *edcNFAComparisonService) ListMappingEntities(ctx context.Context) ([]model.EDCEntity, error) {
	repo, err := s.mappingRepo()
	if err != nil {
		return nil, err
	}
	return repo.ListMappingEntities(ctx)
}

func (s *edcNFAComparisonService) ListMappingSchoolNames(ctx context.Context, srcRegion, cp string) ([]string, error) {
	srcRegion = strings.TrimSpace(srcRegion)
	cp = strings.TrimSpace(cp)
	if srcRegion == "" || cp == "" {
		return nil, NewBadRequest("NFA 节点源区域和 CP 均为必填")
	}
	repo, err := s.mappingRepo()
	if err != nil {
		return nil, err
	}
	names, err := repo.ListMappingSchoolNames(ctx, srcRegion, cp)
	if err != nil {
		return nil, err
	}
	return normalizeComparisonSchoolNames(names), nil
}

func (s *edcNFAComparisonService) CreateMapping(ctx context.Context, input model.EDCNFAComparisonGroupInput) (model.EDCNFAComparisonGroup, error) {
	return s.saveMapping(ctx, 0, input)
}

func (s *edcNFAComparisonService) UpdateMapping(ctx context.Context, id uint64, input model.EDCNFAComparisonGroupInput) (model.EDCNFAComparisonGroup, error) {
	if id == 0 {
		return model.EDCNFAComparisonGroup{}, NewBadRequest("id 必须是正整数")
	}
	return s.saveMapping(ctx, id, input)
}

func (s *edcNFAComparisonService) saveMapping(ctx context.Context, id uint64, input model.EDCNFAComparisonGroupInput) (model.EDCNFAComparisonGroup, error) {
	input.GroupName = strings.TrimSpace(input.GroupName)
	input.NFASrcRegion = strings.TrimSpace(input.NFASrcRegion)
	input.NFACP = strings.TrimSpace(input.NFACP)
	input.NFASchoolNames = normalizeComparisonSchoolNames(input.NFASchoolNames)
	input.Remark = strings.TrimSpace(input.Remark)
	if input.GroupName == "" || input.NFASrcRegion == "" || input.NFACP == "" {
		return model.EDCNFAComparisonGroup{}, NewBadRequest("比较组名称、NFA 节点源区域和 CP 均为必填")
	}
	if len(input.Members) == 0 {
		return model.EDCNFAComparisonGroup{}, NewBadRequest("至少选择一个 EDC 成员")
	}
	repo, err := s.mappingRepo()
	if err != nil {
		return model.EDCNFAComparisonGroup{}, err
	}
	if len(input.NFASchoolNames) > 0 {
		availableSchoolNames, err := s.ListMappingSchoolNames(ctx, input.NFASrcRegion, input.NFACP)
		if err != nil {
			return model.EDCNFAComparisonGroup{}, err
		}
		available := make(map[string]struct{}, len(availableSchoolNames))
		for _, name := range availableSchoolNames {
			available[name] = struct{}{}
		}
		for _, name := range input.NFASchoolNames {
			if _, ok := available[name]; !ok {
				return model.EDCNFAComparisonGroup{}, NewBadRequestf("NFA 院校“%s”不属于节点源区域 %s / %s", name, input.NFASrcRegion, input.NFACP)
			}
		}
	}
	entities, err := repo.ListMappingEntities(ctx)
	if err != nil {
		return model.EDCNFAComparisonGroup{}, err
	}
	available := make(map[uint64]struct{}, len(entities))
	for _, entity := range entities {
		available[entity.ID] = struct{}{}
	}
	seen := make(map[uint64]struct{}, len(input.Members))
	activeIDs := make([]uint64, 0, len(input.Members))
	for _, member := range input.Members {
		if member.EntityID == 0 {
			return model.EDCNFAComparisonGroup{}, NewBadRequest("EDC 成员 ID 必须是正整数")
		}
		if _, ok := available[member.EntityID]; !ok {
			return model.EDCNFAComparisonGroup{}, NewBadRequestf("EDC 实体 %d 不存在、已禁用或为备份节点", member.EntityID)
		}
		if _, ok := seen[member.EntityID]; ok {
			return model.EDCNFAComparisonGroup{}, NewBadRequestf("EDC 实体 %d 在同一比较组中重复", member.EntityID)
		}
		seen[member.EntityID] = struct{}{}
		if member.ValidFrom != nil && member.ValidTo != nil && !member.ValidFrom.Before(*member.ValidTo) {
			return model.EDCNFAComparisonGroup{}, NewBadRequestf("EDC 实体 %d 的生效时间必须早于失效时间", member.EntityID)
		}
		if member.Enabled == nil || *member.Enabled {
			activeIDs = append(activeIDs, member.EntityID)
		}
	}
	conflicts, err := repo.FindActiveMappingMembers(ctx, activeIDs, id)
	if err != nil {
		return model.EDCNFAComparisonGroup{}, err
	}
	for _, old := range conflicts {
		for _, next := range input.Members {
			if old.EntityID != next.EntityID || (next.Enabled != nil && !*next.Enabled) {
				continue
			}
			if intervalsOverlap(old.ValidFrom, old.ValidTo, next.ValidFrom, next.ValidTo) {
				return model.EDCNFAComparisonGroup{}, NewBadRequestf("EDC 实体 %d 已映射到其他比较组，生效时间存在重叠", next.EntityID)
			}
		}
	}
	return repo.SaveMapping(ctx, id, input)
}

func normalizeComparisonSchoolNames(names []string) []string {
	result := make([]string, 0, len(names))
	seen := make(map[string]struct{}, len(names))
	for _, name := range names {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		result = append(result, name)
	}
	return result
}

func (s *edcNFAComparisonService) SetMappingEnabled(ctx context.Context, id uint64, enabled bool) error {
	if id == 0 {
		return NewBadRequest("id 必须是正整数")
	}
	repo, err := s.mappingRepo()
	if err != nil {
		return err
	}
	return repo.SetMappingEnabled(ctx, id, enabled)
}

func intervalsOverlap(aFrom, aTo, bFrom, bTo *time.Time) bool {
	if aTo != nil && bFrom != nil && !aTo.After(*bFrom) {
		return false
	}
	if bTo != nil && aFrom != nil && !bTo.After(*aFrom) {
		return false
	}
	return true
}

func (s *edcNFAComparisonService) GetComparison(ctx context.Context, filter model.EDCNFAComparisonFilter) ([]model.EDCNFAComparisonPoint, error) {
	filter.StartTime = filter.StartTime.In(time.Local)
	filter.EndTime = filter.EndTime.In(time.Local)
	if filter.StartTime.IsZero() && filter.EndTime.IsZero() {
		filter.EndTime = time.Now()
		filter.StartTime = filter.EndTime.Add(-24 * time.Hour)
	} else if filter.StartTime.IsZero() {
		filter.StartTime = filter.EndTime.Add(-24 * time.Hour)
	} else if filter.EndTime.IsZero() {
		filter.EndTime = time.Now()
	}
	if !filter.StartTime.Before(filter.EndTime) {
		return nil, NewBadRequest("start_time 必须早于 end_time")
	}
	if filter.EndTime.Sub(filter.StartTime) > 366*24*time.Hour {
		return nil, NewBadRequest("单次比较时间范围不能超过 366 天")
	}
	return s.repo.GetComparison(ctx, filter)
}
