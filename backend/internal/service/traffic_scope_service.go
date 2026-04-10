package service

import (
	"sort"
	"strings"

	"nfa-dashboard/internal/model"
	"nfa-dashboard/internal/repository"
)

type TrafficScopeService interface {
	ListRules(userID uint64) ([]model.TrafficScopeRuleGroup, error)
	ReplaceRules(userID uint64, groups []model.TrafficScopeRuleGroup) error
	ResolveEffectiveScope(userID uint64) (model.EffectiveTrafficScope, error)
	Preview(userID uint64) (model.EffectiveTrafficScopePreview, error)
}

type trafficScopeService struct {
	ruleRepo       repository.TrafficScopeRuleRepository
	schoolRepo     repository.TrafficScopeSchoolRepository
	userSchoolRepo repository.UserSchoolRepository
	userRepo       repository.UserRepository
}

func NewTrafficScopeService(
	ruleRepo repository.TrafficScopeRuleRepository,
	schoolRepo repository.TrafficScopeSchoolRepository,
	userSchoolRepo repository.UserSchoolRepository,
	userRepo repository.UserRepository,
) TrafficScopeService {
	return &trafficScopeService{
		ruleRepo:       ruleRepo,
		schoolRepo:     schoolRepo,
		userSchoolRepo: userSchoolRepo,
		userRepo:       userRepo,
	}
}

func (s *trafficScopeService) ListRules(userID uint64) ([]model.TrafficScopeRuleGroup, error) {
	if userID == 0 {
		return nil, NewBadRequest("invalid user id")
	}
	return s.ruleRepo.ListByUser(userID)
}

func (s *trafficScopeService) ReplaceRules(userID uint64, groups []model.TrafficScopeRuleGroup) error {
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
	normalized := make([]model.TrafficScopeRuleGroup, 0, len(groups))
	for _, group := range groups {
		g, include, err := normalizeTrafficScopeRuleGroup(userID, group)
		if err != nil {
			return err
		}
		if include {
			normalized = append(normalized, g)
		}
	}
	return s.ruleRepo.ReplaceByUser(userID, normalized)
}

func (s *trafficScopeService) ResolveEffectiveScope(userID uint64) (model.EffectiveTrafficScope, error) {
	scope := model.EffectiveTrafficScope{
		UserID:            userID,
		Source:            model.TrafficScopeSourceNone,
		AllowedSchoolKeys: []model.TrafficScopeSchoolKey{},
		AllowedSchoolIDs:  []string{},
	}
	if userID == 0 {
		return scope, nil
	}
	groups, err := s.ruleRepo.ListByUser(userID)
	if err != nil {
		return scope, err
	}
	if len(groups) > 0 {
		allowSet := map[string]model.TrafficScopeSchoolKey{}
		denySet := map[string]model.TrafficScopeSchoolKey{}
		for _, group := range groups {
			schoolKeys, err := s.resolveRuleGroupSchoolKeys(group)
			if err != nil {
				return scope, err
			}
			target := allowSet
			if group.RuleType == model.TrafficScopeRuleTypeDeny {
				target = denySet
			}
			for _, key := range schoolKeys {
				target[schoolKeyString(key)] = key
			}
		}
		keys := make([]model.TrafficScopeSchoolKey, 0, len(allowSet))
		for keyString, key := range allowSet {
			if _, denied := denySet[keyString]; denied {
				continue
			}
			keys = append(keys, key)
		}
		scope.AllowedSchoolKeys = sortTrafficScopeSchoolKeys(keys)
		scope.AllowedSchoolIDs = schoolIDsFromKeys(scope.AllowedSchoolKeys)
		scope.Source = model.TrafficScopeSourcePolicyRule
		return scope, nil
	}
	if s.userSchoolRepo != nil {
		legacyIDs, err := s.userSchoolRepo.GetSchoolIDsByUser(userID)
		if err != nil {
			return scope, err
		}
		if len(legacyIDs) > 0 {
			keys, err := s.schoolRepo.ExpandSchoolIDsToKeys(legacyIDs)
			if err != nil {
				return scope, err
			}
			scope.Source = model.TrafficScopeSourceLegacyUserSchool
			scope.AllowedSchoolKeys = sortTrafficScopeSchoolKeys(keys)
			scope.AllowedSchoolIDs = dedupeSortedStrings(legacyIDs)
			return scope, nil
		}
	}
	isAdmin, err := s.hasDefaultAdminScope(userID)
	if err != nil {
		return scope, err
	}
	if isAdmin {
		schools, err := s.schoolRepo.ListAllSchools()
		if err != nil {
			return scope, err
		}
		keys := make([]model.TrafficScopeSchoolKey, 0, len(schools))
		for _, school := range schools {
			keys = append(keys, model.TrafficScopeSchoolKey{
				SchoolID: school.SchoolID,
				Region:   school.Region,
				CP:       school.CP,
			})
		}
		scope.Source = model.TrafficScopeSourceDefaultAdminRole
		scope.AllowedSchoolKeys = sortTrafficScopeSchoolKeys(keys)
		scope.AllowedSchoolIDs = schoolIDsFromKeys(scope.AllowedSchoolKeys)
	}
	return scope, nil
}

func (s *trafficScopeService) Preview(userID uint64) (model.EffectiveTrafficScopePreview, error) {
	preview := model.EffectiveTrafficScopePreview{
		UserID:          userID,
		Source:          model.TrafficScopeSourceNone,
		Rules:           []model.TrafficScopeRuleGroup{},
		LegacySchoolIDs: []string{},
		AllowedSchools:  []model.School{},
	}
	if userID == 0 {
		return preview, NewBadRequest("invalid user id")
	}
	groups, err := s.ruleRepo.ListByUser(userID)
	if err != nil {
		return preview, err
	}
	preview.Rules = groups
	if s.userSchoolRepo != nil {
		legacyIDs, err := s.userSchoolRepo.GetSchoolIDsByUser(userID)
		if err != nil {
			return preview, err
		}
		preview.LegacySchoolIDs = dedupeSortedStrings(legacyIDs)
	}
	scope, err := s.ResolveEffectiveScope(userID)
	if err != nil {
		return preview, err
	}
	preview.Source = scope.Source
	if len(scope.AllowedSchoolKeys) > 0 {
		schools, err := s.schoolRepo.GetSchoolsByKeys(scope.AllowedSchoolKeys)
		if err != nil {
			return preview, err
		}
		preview.AllowedSchools = schools
	}
	return preview, nil
}

func normalizeTrafficScopeRuleGroup(userID uint64, group model.TrafficScopeRuleGroup) (model.TrafficScopeRuleGroup, bool, error) {
	group.UserID = userID
	group.RuleType = strings.TrimSpace(strings.ToLower(group.RuleType))
	if group.RuleType != model.TrafficScopeRuleTypeAllow && group.RuleType != model.TrafficScopeRuleTypeDeny {
		return model.TrafficScopeRuleGroup{}, false, NewBadRequest("invalid rule_type")
	}
	normalizedConditions := make([]model.TrafficScopeCondition, 0, len(group.Conditions))
	for _, condition := range group.Conditions {
		c, include, err := normalizeTrafficScopeCondition(condition)
		if err != nil {
			return model.TrafficScopeRuleGroup{}, false, err
		}
		if include {
			normalizedConditions = append(normalizedConditions, c)
		}
	}
	if len(normalizedConditions) == 0 {
		return model.TrafficScopeRuleGroup{}, false, nil
	}
	group.Conditions = normalizedConditions
	return group, true, nil
}

func normalizeTrafficScopeCondition(condition model.TrafficScopeCondition) (model.TrafficScopeCondition, bool, error) {
	condition.DimensionType = strings.TrimSpace(strings.ToLower(condition.DimensionType))
	condition.DimensionValue = strings.TrimSpace(condition.DimensionValue)
	switch condition.DimensionType {
	case model.TrafficScopeDimensionRegion, model.TrafficScopeDimensionCP, model.TrafficScopeDimensionSchool:
	default:
		return model.TrafficScopeCondition{}, false, NewBadRequest("invalid dimension_type")
	}
	if condition.DimensionValue == "" {
		return model.TrafficScopeCondition{}, false, nil
	}
	return condition, true, nil
}

func (s *trafficScopeService) resolveRuleGroupSchoolKeys(group model.TrafficScopeRuleGroup) ([]model.TrafficScopeSchoolKey, error) {
	if len(group.Conditions) == 0 {
		return []model.TrafficScopeSchoolKey{}, nil
	}
	perDimension := map[string]map[string]model.TrafficScopeSchoolKey{}
	for _, condition := range group.Conditions {
		schools, err := s.schoolRepo.MatchSchools(condition.DimensionType, condition.DimensionValue)
		if err != nil {
			return nil, err
		}
		if _, ok := perDimension[condition.DimensionType]; !ok {
			perDimension[condition.DimensionType] = map[string]model.TrafficScopeSchoolKey{}
		}
		for _, school := range schools {
			key := model.TrafficScopeSchoolKey{
				SchoolID: strings.TrimSpace(school.SchoolID),
				Region:   strings.TrimSpace(school.Region),
				CP:       strings.TrimSpace(school.CP),
			}
			if key.SchoolID == "" || key.Region == "" || key.CP == "" {
				continue
			}
			perDimension[condition.DimensionType][schoolKeyString(key)] = key
		}
	}
	if len(perDimension) == 0 {
		return []model.TrafficScopeSchoolKey{}, nil
	}
	var intersection map[string]model.TrafficScopeSchoolKey
	for _, schoolSet := range perDimension {
		if intersection == nil {
			intersection = copySchoolKeyMap(schoolSet)
			continue
		}
		intersection = intersectSchoolKeyMaps(intersection, schoolSet)
	}
	out := make([]model.TrafficScopeSchoolKey, 0, len(intersection))
	for _, key := range intersection {
		out = append(out, key)
	}
	return sortTrafficScopeSchoolKeys(out), nil
}

func schoolKeyString(key model.TrafficScopeSchoolKey) string {
	return key.SchoolID + "|" + key.Region + "|" + key.CP
}

func copySchoolKeyMap(source map[string]model.TrafficScopeSchoolKey) map[string]model.TrafficScopeSchoolKey {
	out := make(map[string]model.TrafficScopeSchoolKey, len(source))
	for key, value := range source {
		out[key] = value
	}
	return out
}

func intersectSchoolKeyMaps(left, right map[string]model.TrafficScopeSchoolKey) map[string]model.TrafficScopeSchoolKey {
	if len(left) > len(right) {
		left, right = right, left
	}
	out := map[string]model.TrafficScopeSchoolKey{}
	for key, value := range left {
		if _, ok := right[key]; ok {
			out[key] = value
		}
	}
	return out
}

func sortTrafficScopeSchoolKeys(keys []model.TrafficScopeSchoolKey) []model.TrafficScopeSchoolKey {
	if len(keys) == 0 {
		return []model.TrafficScopeSchoolKey{}
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].SchoolID != keys[j].SchoolID {
			return keys[i].SchoolID < keys[j].SchoolID
		}
		if keys[i].Region != keys[j].Region {
			return keys[i].Region < keys[j].Region
		}
		return keys[i].CP < keys[j].CP
	})
	return keys
}

func schoolIDsFromKeys(keys []model.TrafficScopeSchoolKey) []string {
	values := make([]string, 0, len(keys))
	for _, key := range keys {
		values = append(values, key.SchoolID)
	}
	return dedupeSortedStrings(values)
}

func dedupeSortedStrings(values []string) []string {
	if len(values) == 0 {
		return []string{}
	}
	seen := map[string]struct{}{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		v := strings.TrimSpace(value)
		if v == "" {
			continue
		}
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	sort.Strings(out)
	return out
}

func (s *trafficScopeService) hasDefaultAdminScope(userID uint64) (bool, error) {
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
