package service

import (
	"sort"
	"strings"
	"time"

	"nfa-dashboard/internal/model"
	"nfa-dashboard/internal/repository"
)

type EDCService interface {
	ListEntities(filter model.EDCEntityFilter) ([]model.EDCEntity, int64, error)
	ListRegions(allowedEntityIDs []uint64) ([]string, error)
	ListCPs(allowedEntityIDs []uint64) ([]string, error)
	ListFilterOptions(allowedEntityIDs []uint64) (model.EDCFilterOptions, error)
	GetTrafficData(filter model.EDCTrafficFilter) ([]model.EDCTrafficResponse, error)
	GetTrafficSummary(filter model.EDCTrafficFilter) (model.EDCTrafficResponse, error)
}

type edcService struct {
	repo repository.EDCRepository
}

func NewEDCService(repo repository.EDCRepository) EDCService {
	return &edcService{repo: repo}
}

func (s *edcService) ListEntities(filter model.EDCEntityFilter) ([]model.EDCEntity, int64, error) {
	if filter.Limit <= 0 {
		filter.Limit = 100
	}
	filter.EnabledOnly = true
	return s.repo.ListEntities(filter)
}

func (s *edcService) ListRegions(allowedEntityIDs []uint64) ([]string, error) {
	return s.repo.ListRegions(allowedEntityIDs)
}

func (s *edcService) ListCPs(allowedEntityIDs []uint64) ([]string, error) {
	return s.repo.ListCPs(allowedEntityIDs)
}

func (s *edcService) ListFilterOptions(allowedEntityIDs []uint64) (model.EDCFilterOptions, error) {
	return s.repo.ListFilterOptions(allowedEntityIDs)
}

func (s *edcService) GetTrafficData(filter model.EDCTrafficFilter) ([]model.EDCTrafficResponse, error) {
	filter = defaultEDCTrafficFilter(filter)
	return s.repo.GetTrafficData(filter)
}

func (s *edcService) GetTrafficSummary(filter model.EDCTrafficFilter) (model.EDCTrafficResponse, error) {
	filter = defaultEDCTrafficFilter(filter)
	return s.repo.GetTrafficSummary(filter)
}

func defaultEDCTrafficFilter(filter model.EDCTrafficFilter) model.EDCTrafficFilter {
	if filter.StartTime.IsZero() {
		filter.StartTime = time.Now().AddDate(0, 0, -1)
	}
	if filter.EndTime.IsZero() {
		filter.EndTime = time.Now()
	}
	return filter
}

type EDCTrafficScopeService interface {
	ListRules(userID uint64) ([]model.EDCTrafficScopeRuleGroup, error)
	ReplaceRules(userID uint64, groups []model.EDCTrafficScopeRuleGroup) error
	ResolveEffectiveScope(userID uint64) (model.EffectiveEDCTrafficScope, error)
}

type edcTrafficScopeService struct {
	repo     repository.EDCTrafficScopeRepository
	userRepo repository.UserRepository
}

func NewEDCTrafficScopeService(repo repository.EDCTrafficScopeRepository, userRepo repository.UserRepository) EDCTrafficScopeService {
	return &edcTrafficScopeService{repo: repo, userRepo: userRepo}
}

func (s *edcTrafficScopeService) ListRules(userID uint64) ([]model.EDCTrafficScopeRuleGroup, error) {
	if userID == 0 {
		return nil, NewBadRequest("invalid user id")
	}
	return s.repo.ListByUser(userID)
}

func (s *edcTrafficScopeService) ReplaceRules(userID uint64, groups []model.EDCTrafficScopeRuleGroup) error {
	if userID == 0 {
		return NewBadRequest("invalid user id")
	}
	if s.userRepo != nil {
		ok, err := s.userRepo.Exists(userID)
		if err != nil {
			return err
		}
		if !ok {
			return NewBadRequest("user not found")
		}
	}
	normalized := make([]model.EDCTrafficScopeRuleGroup, 0, len(groups))
	for _, group := range groups {
		next, include, err := normalizeEDCScopeRuleGroup(userID, group)
		if err != nil {
			return err
		}
		if include {
			normalized = append(normalized, next)
		}
	}
	return s.repo.ReplaceByUser(userID, normalized)
}

func (s *edcTrafficScopeService) ResolveEffectiveScope(userID uint64) (model.EffectiveEDCTrafficScope, error) {
	scope := model.EffectiveEDCTrafficScope{
		Source:           model.EDCTrafficScopeSourceNone,
		AllowedEntityIDs: []uint64{},
	}
	if userID == 0 {
		return scope, nil
	}
	groups, err := s.repo.ListByUser(userID)
	if err != nil {
		return scope, err
	}
	if len(groups) > 0 {
		allowSet := map[uint64]struct{}{}
		denySet := map[uint64]struct{}{}
		for _, group := range groups {
			ids, err := s.resolveRuleGroupEntityIDs(group)
			if err != nil {
				return scope, err
			}
			target := allowSet
			if group.RuleType == model.EDCTrafficScopeRuleTypeDeny {
				target = denySet
			}
			for _, id := range ids {
				target[id] = struct{}{}
			}
		}
		ids := make([]uint64, 0, len(allowSet))
		for id := range allowSet {
			if _, denied := denySet[id]; denied {
				continue
			}
			ids = append(ids, id)
		}
		scope.AllowedEntityIDs = sortUint64s(ids)
		scope.Source = model.EDCTrafficScopeSourcePolicyRule
		return scope, nil
	}
	if ok, err := s.hasDefaultAdminScope(userID); err != nil {
		return scope, err
	} else if ok {
		entities, err := s.repo.ListAllEntities()
		if err != nil {
			return scope, err
		}
		ids := make([]uint64, 0, len(entities))
		for _, entity := range entities {
			ids = append(ids, entity.ID)
		}
		scope.AllowedEntityIDs = sortUint64s(ids)
		scope.Source = model.EDCTrafficScopeSourceDefaultAdminRole
	}
	return scope, nil
}

func normalizeEDCScopeRuleGroup(userID uint64, group model.EDCTrafficScopeRuleGroup) (model.EDCTrafficScopeRuleGroup, bool, error) {
	group.UserID = userID
	group.RuleType = strings.TrimSpace(strings.ToLower(group.RuleType))
	if group.RuleType != model.EDCTrafficScopeRuleTypeAllow && group.RuleType != model.EDCTrafficScopeRuleTypeDeny {
		return model.EDCTrafficScopeRuleGroup{}, false, NewBadRequest("invalid rule_type")
	}
	conditions := make([]model.EDCTrafficScopeCondition, 0, len(group.Conditions))
	for _, condition := range group.Conditions {
		condition.DimensionType = strings.TrimSpace(strings.ToLower(condition.DimensionType))
		condition.DimensionValue = strings.TrimSpace(condition.DimensionValue)
		switch condition.DimensionType {
		case model.EDCTrafficScopeDimensionRegion, model.EDCTrafficScopeDimensionCP, model.EDCTrafficScopeDimensionEntity:
		default:
			return model.EDCTrafficScopeRuleGroup{}, false, NewBadRequest("invalid dimension_type")
		}
		if condition.DimensionValue == "" {
			continue
		}
		conditions = append(conditions, condition)
	}
	if len(conditions) == 0 {
		return model.EDCTrafficScopeRuleGroup{}, false, nil
	}
	group.Conditions = conditions
	return group, true, nil
}

func (s *edcTrafficScopeService) resolveRuleGroupEntityIDs(group model.EDCTrafficScopeRuleGroup) ([]uint64, error) {
	if len(group.Conditions) == 0 {
		return []uint64{}, nil
	}
	var intersection map[uint64]struct{}
	for _, condition := range group.Conditions {
		entities, err := s.repo.MatchEntities(condition.DimensionType, condition.DimensionValue)
		if err != nil {
			return nil, err
		}
		current := map[uint64]struct{}{}
		for _, entity := range entities {
			if entity.ID != 0 {
				current[entity.ID] = struct{}{}
			}
		}
		if intersection == nil {
			intersection = current
			continue
		}
		next := map[uint64]struct{}{}
		for id := range intersection {
			if _, ok := current[id]; ok {
				next[id] = struct{}{}
			}
		}
		intersection = next
	}
	ids := make([]uint64, 0, len(intersection))
	for id := range intersection {
		ids = append(ids, id)
	}
	return sortUint64s(ids), nil
}

func (s *edcTrafficScopeService) hasDefaultAdminScope(userID uint64) (bool, error) {
	if s.userRepo == nil || userID == 0 {
		return false, nil
	}
	roles, err := s.userRepo.GetUserRoles(userID)
	if err != nil {
		return false, err
	}
	for _, role := range roles {
		if strings.EqualFold(strings.TrimSpace(role.Name), "admin") {
			return true, nil
		}
	}
	return false, nil
}

func sortUint64s(values []uint64) []uint64 {
	sort.Slice(values, func(i, j int) bool { return values[i] < values[j] })
	return values
}
