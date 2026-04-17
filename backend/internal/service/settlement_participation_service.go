package service

import (
	"context"
	"sort"
	"strings"
	"sync"
	"time"

	"nfa-dashboard/internal/model"
	"nfa-dashboard/internal/repository"
)

type SettlementParticipationService interface {
	ListParticipatingSchoolKeys(ctx context.Context) ([]model.TrafficScopeSchoolKey, error)
	InvalidateCache()
}

type settlementParticipationService struct {
	schoolRepo      repository.TrafficScopeSchoolRepository
	filterRulesRepo repository.FilterRulesRepository
	cacheTTL        time.Duration

	mu       sync.RWMutex
	cachedAt time.Time
	cached   []model.TrafficScopeSchoolKey
}

func NewSettlementParticipationService(
	schoolRepo repository.TrafficScopeSchoolRepository,
	filterRulesRepo repository.FilterRulesRepository,
	cacheTTL time.Duration,
) SettlementParticipationService {
	if cacheTTL <= 0 {
		cacheTTL = 2 * time.Minute
	}
	return &settlementParticipationService{
		schoolRepo:      schoolRepo,
		filterRulesRepo: filterRulesRepo,
		cacheTTL:        cacheTTL,
		cached:          []model.TrafficScopeSchoolKey{},
	}
}

func (s *settlementParticipationService) ListParticipatingSchoolKeys(ctx context.Context) ([]model.TrafficScopeSchoolKey, error) {
	if keys, ok := s.readCache(); ok {
		return keys, nil
	}

	schools, err := s.schoolRepo.ListAllSchools()
	if err != nil {
		return nil, err
	}
	rules, err := s.filterRulesRepo.ListEnabled()
	if err != nil {
		return nil, err
	}
	normalizedRules := make([]model.RateCustomerFilterRule, 0, len(rules))
	for _, rule := range rules {
		normalized, normalizeErr := normalizeFilterRule(&rule)
		if normalizeErr != nil {
			return nil, normalizeErr
		}
		normalizedRules = append(normalizedRules, normalized)
	}

	result := make([]model.TrafficScopeSchoolKey, 0, len(schools))
	seen := map[string]struct{}{}
	for _, school := range schools {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}
		excluded, matchErr := isSchoolExcludedByRules(school, normalizedRules)
		if matchErr != nil {
			return nil, matchErr
		}
		if excluded {
			continue
		}
		key := model.TrafficScopeSchoolKey{
			SchoolID: school.SchoolID,
			Region:   school.Region,
			CP:       school.CP,
		}
		keyStr := schoolKeyString(key)
		if _, ok := seen[keyStr]; ok {
			continue
		}
		seen[keyStr] = struct{}{}
		result = append(result, key)
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].SchoolID != result[j].SchoolID {
			return result[i].SchoolID < result[j].SchoolID
		}
		if result[i].Region != result[j].Region {
			return result[i].Region < result[j].Region
		}
		return result[i].CP < result[j].CP
	})

	s.writeCache(result)
	return append([]model.TrafficScopeSchoolKey(nil), result...), nil
}

func (s *settlementParticipationService) InvalidateCache() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cachedAt = time.Time{}
	s.cached = []model.TrafficScopeSchoolKey{}
}

func (s *settlementParticipationService) readCache() ([]model.TrafficScopeSchoolKey, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.cachedAt.IsZero() || time.Since(s.cachedAt) > s.cacheTTL {
		return nil, false
	}
	return append([]model.TrafficScopeSchoolKey(nil), s.cached...), true
}

func (s *settlementParticipationService) writeCache(keys []model.TrafficScopeSchoolKey) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cachedAt = time.Now()
	s.cached = append([]model.TrafficScopeSchoolKey(nil), keys...)
}

func isSchoolExcludedByRules(school model.School, rules []model.RateCustomerFilterRule) (bool, error) {
	if len(rules) == 0 {
		return false, nil
	}
	schoolName := strings.TrimSpace(school.SchoolName)
	customer := model.RateCustomer{
		Region: school.Region,
		CP:     school.CP,
	}
	customer.SchoolName = &schoolName

	for _, rule := range rules {
		matched, err := matchRateCustomerFilterRule(customer, rule)
		if err != nil {
			return false, err
		}
		if matched {
			return true, nil
		}
	}
	return false, nil
}
