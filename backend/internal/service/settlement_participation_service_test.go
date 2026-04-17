package service

import (
	"context"
	"testing"
	"time"

	"gorm.io/datatypes"
	"nfa-dashboard/internal/model"
)

type settlementParticipationFilterRulesRepoStub struct {
	enabled []model.RateCustomerFilterRule
}

func (s *settlementParticipationFilterRulesRepoStub) List(filter map[string]interface{}, limit, offset int) ([]model.RateCustomerFilterRule, int64, error) {
	return nil, 0, nil
}
func (s *settlementParticipationFilterRulesRepoStub) ListDistinctCustomerRegions() ([]string, error) {
	return nil, nil
}
func (s *settlementParticipationFilterRulesRepoStub) ListDistinctCustomerCPs() ([]string, error) {
	return nil, nil
}
func (s *settlementParticipationFilterRulesRepoStub) Create(rule *model.RateCustomerFilterRule) (*model.RateCustomerFilterRule, error) {
	return rule, nil
}
func (s *settlementParticipationFilterRulesRepoStub) Update(id uint64, updates map[string]interface{}) error {
	return nil
}
func (s *settlementParticipationFilterRulesRepoStub) Delete(id uint64) error {
	return nil
}
func (s *settlementParticipationFilterRulesRepoStub) UpdatePriority(id uint64, priority int) error {
	return nil
}
func (s *settlementParticipationFilterRulesRepoStub) SetEnabled(id uint64, enabled bool) error {
	return nil
}
func (s *settlementParticipationFilterRulesRepoStub) GetMatchSummary(rule model.RateCustomerFilterRule, previewLimit int) (int64, []string, error) {
	return 0, nil, nil
}
func (s *settlementParticipationFilterRulesRepoStub) ListEnabled() ([]model.RateCustomerFilterRule, error) {
	out := make([]model.RateCustomerFilterRule, len(s.enabled))
	copy(out, s.enabled)
	return out, nil
}

func TestSettlementParticipationService_ListParticipatingSchoolKeys_ExcludesMatchedSchools(t *testing.T) {
	schoolRepo := &trafficScopeSchoolRepoStub{
		allSchools: []model.School{
			{SchoolID: "school-a", SchoolName: "中国海洋大学", Region: "华东", CP: "cmcc"},
			{SchoolID: "school-b", SchoolName: "北京大学", Region: "华东", CP: "ctcc"},
		},
	}
	filterRepo := &settlementParticipationFilterRulesRepoStub{
		enabled: []model.RateCustomerFilterRule{
			{
				Name:                "exclude-haijiang",
				Enabled:             true,
				ScopeRegion:         datatypes.JSON([]byte(`["华东"]`)),
				ScopeCP:             datatypes.JSON([]byte(`[]`)),
				SchoolNameMatchType: "contains",
				SchoolNameValues:    datatypes.JSON([]byte(`["海洋"]`)),
			},
		},
	}

	svc := NewSettlementParticipationService(schoolRepo, filterRepo, 5*time.Minute)
	keys, err := svc.ListParticipatingSchoolKeys(context.Background())
	if err != nil {
		t.Fatalf("ListParticipatingSchoolKeys() error = %v", err)
	}
	if len(keys) != 1 || keys[0].SchoolID != "school-b" || keys[0].CP != "ctcc" {
		t.Fatalf("expected only school-b to remain, got %#v", keys)
	}
}

func TestSettlementParticipationService_InvalidateCache(t *testing.T) {
	schoolRepo := &trafficScopeSchoolRepoStub{
		allSchools: []model.School{
			{SchoolID: "school-a", SchoolName: "中国海洋大学", Region: "华东", CP: "cmcc"},
			{SchoolID: "school-b", SchoolName: "北京大学", Region: "华东", CP: "ctcc"},
		},
	}
	filterRepo := &settlementParticipationFilterRulesRepoStub{
		enabled: []model.RateCustomerFilterRule{
			{
				Name:                "exclude-haijiang",
				Enabled:             true,
				ScopeRegion:         datatypes.JSON([]byte(`["华东"]`)),
				SchoolNameMatchType: "contains",
				SchoolNameValues:    datatypes.JSON([]byte(`["海洋"]`)),
			},
		},
	}

	svc := NewSettlementParticipationService(schoolRepo, filterRepo, 30*time.Minute)

	first, err := svc.ListParticipatingSchoolKeys(context.Background())
	if err != nil {
		t.Fatalf("first ListParticipatingSchoolKeys() error = %v", err)
	}
	if len(first) != 1 {
		t.Fatalf("expected first call to return 1 key, got %#v", first)
	}

	filterRepo.enabled = nil

	cached, err := svc.ListParticipatingSchoolKeys(context.Background())
	if err != nil {
		t.Fatalf("cached ListParticipatingSchoolKeys() error = %v", err)
	}
	if len(cached) != 1 {
		t.Fatalf("expected cached call to remain 1 key before invalidation, got %#v", cached)
	}

	svc.InvalidateCache()
	refreshed, err := svc.ListParticipatingSchoolKeys(context.Background())
	if err != nil {
		t.Fatalf("refreshed ListParticipatingSchoolKeys() error = %v", err)
	}
	if len(refreshed) != 2 {
		t.Fatalf("expected invalidation to refresh cache and return 2 keys, got %#v", refreshed)
	}
}
