package service

import (
	"testing"
	"time"

	"nfa-dashboard/internal/model"
)

type edcRepoStub struct {
	entities      []model.EDCEntity
	trafficFilter model.EDCTrafficFilter
	summaryFilter model.EDCTrafficFilter
}

func (s *edcRepoStub) ListEntities(filter model.EDCEntityFilter) ([]model.EDCEntity, int64, error) {
	out := make([]model.EDCEntity, 0)
	for _, entity := range s.entities {
		if filter.Region != "" && entity.Region != filter.Region {
			continue
		}
		if filter.CP != "" && entity.CP != filter.CP {
			continue
		}
		if len(filter.AllowedEntityIDs) > 0 {
			allowed := false
			for _, id := range filter.AllowedEntityIDs {
				if id == entity.ID {
					allowed = true
					break
				}
			}
			if !allowed {
				continue
			}
		}
		out = append(out, entity)
	}
	return out, int64(len(out)), nil
}

func (s *edcRepoStub) ListRegions(allowedEntityIDs []uint64) ([]string, error) {
	return []string{"天津", "广东"}, nil
}

func (s *edcRepoStub) ListCPs(allowedEntityIDs []uint64) ([]string, error) {
	return []string{"bilibili", "baidu"}, nil
}

func (s *edcRepoStub) ListFilterOptions(allowedEntityIDs []uint64) (model.EDCFilterOptions, error) {
	return model.EDCFilterOptions{EntityTypes: []string{"node", "transmission"}}, nil
}

func (s *edcRepoStub) GetTrafficData(filter model.EDCTrafficFilter) ([]model.EDCTrafficResponse, error) {
	s.trafficFilter = filter
	return []model.EDCTrafficResponse{{EntityID: 10, DisplayName: "TJ-Bilibili", ServiceSize: 300, CacheSize: 100, Total: 400}}, nil
}

func (s *edcRepoStub) GetTrafficSummary(filter model.EDCTrafficFilter) (model.EDCTrafficResponse, error) {
	s.summaryFilter = filter
	return model.EDCTrafficResponse{ServiceSize: 300, CacheSize: 100, Total: 400}, nil
}

type edcScopeRepoStub struct {
	groupsByUser map[uint64][]model.EDCTrafficScopeRuleGroup
	entities     []model.EDCEntity
}

func (s *edcScopeRepoStub) ListByUser(userID uint64) ([]model.EDCTrafficScopeRuleGroup, error) {
	return append([]model.EDCTrafficScopeRuleGroup(nil), s.groupsByUser[userID]...), nil
}

func (s *edcScopeRepoStub) ReplaceByUser(userID uint64, groups []model.EDCTrafficScopeRuleGroup) error {
	copied := make([]model.EDCTrafficScopeRuleGroup, len(groups))
	copy(copied, groups)
	s.groupsByUser[userID] = copied
	return nil
}

func (s *edcScopeRepoStub) MatchEntities(dimension, value string) ([]model.EDCEntity, error) {
	out := make([]model.EDCEntity, 0)
	for _, entity := range s.entities {
		switch dimension {
		case model.EDCTrafficScopeDimensionRegion:
			if entity.Region == value {
				out = append(out, entity)
			}
		case model.EDCTrafficScopeDimensionCP:
			if entity.CP == value {
				out = append(out, entity)
			}
		case model.EDCTrafficScopeDimensionEntity:
			if entity.DisplayName == value || entity.EDCName == value {
				out = append(out, entity)
			}
		}
	}
	return out, nil
}

func (s *edcScopeRepoStub) ListAllEntities() ([]model.EDCEntity, error) {
	return append([]model.EDCEntity(nil), s.entities...), nil
}

type edcUserRepoStub struct {
	rolesByUser map[uint64][]model.Role
}

func (s *edcUserRepoStub) GetByUsername(username string) (*model.User, error) { return nil, nil }
func (s *edcUserRepoStub) GetByID(id uint64) (*model.User, error)             { return nil, nil }
func (s *edcUserRepoStub) GetUserRoles(userID uint64) ([]model.Role, error) {
	return append([]model.Role(nil), s.rolesByUser[userID]...), nil
}
func (s *edcUserRepoStub) GetUserPermissions(userID uint64) ([]model.Permission, error) {
	return nil, nil
}
func (s *edcUserRepoStub) List(username string, status *int8, roles []string, page, pageSize int) ([]model.User, int64, error) {
	return nil, 0, nil
}
func (s *edcUserRepoStub) FindByIDs(ids []uint64) ([]model.User, error) { return nil, nil }
func (s *edcUserRepoStub) FindActiveByDisplayNames(names []string) ([]model.User, error) {
	return nil, nil
}
func (s *edcUserRepoStub) UsernameExists(username string) (bool, error) { return false, nil }
func (s *edcUserRepoStub) Create(u *model.User) (*model.User, error)    { return u, nil }
func (s *edcUserRepoStub) SetRoles(userID uint64, roleIDs []uint64) error {
	return nil
}
func (s *edcUserRepoStub) UpdateStatus(userID uint64, status int8) error { return nil }
func (s *edcUserRepoStub) UpdateAlias(userID uint64, alias *string) error {
	return nil
}
func (s *edcUserRepoStub) UpdatePasswordHash(userID uint64, passwordHash string) error {
	return nil
}
func (s *edcUserRepoStub) Exists(id uint64) (bool, error) { return id != 0, nil }

func TestEDCScopeIntersectsConditionsAcrossDimensions(t *testing.T) {
	scopeRepo := &edcScopeRepoStub{
		groupsByUser: map[uint64][]model.EDCTrafficScopeRuleGroup{
			8: {
				{
					UserID:   8,
					RuleType: model.EDCTrafficScopeRuleTypeAllow,
					Conditions: []model.EDCTrafficScopeCondition{
						{DimensionType: model.EDCTrafficScopeDimensionRegion, DimensionValue: "天津"},
						{DimensionType: model.EDCTrafficScopeDimensionCP, DimensionValue: "bilibili"},
					},
				},
			},
		},
		entities: []model.EDCEntity{
			{ID: 1, DisplayName: "TJ-Bilibili", Region: "天津", CP: "bilibili"},
			{ID: 2, DisplayName: "TJ-Baidu", Region: "天津", CP: "baidu"},
			{ID: 3, DisplayName: "GD-Bilibili", Region: "广东", CP: "bilibili"},
		},
	}
	svc := NewEDCTrafficScopeService(scopeRepo, &edcUserRepoStub{})

	scope, err := svc.ResolveEffectiveScope(8)
	if err != nil {
		t.Fatalf("ResolveEffectiveScope() error = %v", err)
	}
	if scope.Source != model.EDCTrafficScopeSourcePolicyRule {
		t.Fatalf("source=%q, want %q", scope.Source, model.EDCTrafficScopeSourcePolicyRule)
	}
	if len(scope.AllowedEntityIDs) != 1 || scope.AllowedEntityIDs[0] != 1 {
		t.Fatalf("AllowedEntityIDs=%v, want [1]", scope.AllowedEntityIDs)
	}
}

func TestEDCScopeDenyOverridesAllow(t *testing.T) {
	scopeRepo := &edcScopeRepoStub{
		groupsByUser: map[uint64][]model.EDCTrafficScopeRuleGroup{
			9: {
				{
					UserID:   9,
					RuleType: model.EDCTrafficScopeRuleTypeAllow,
					Conditions: []model.EDCTrafficScopeCondition{
						{DimensionType: model.EDCTrafficScopeDimensionRegion, DimensionValue: "天津"},
					},
				},
				{
					UserID:   9,
					RuleType: model.EDCTrafficScopeRuleTypeDeny,
					Conditions: []model.EDCTrafficScopeCondition{
						{DimensionType: model.EDCTrafficScopeDimensionEntity, DimensionValue: "TJ-Baidu"},
					},
				},
			},
		},
		entities: []model.EDCEntity{
			{ID: 1, DisplayName: "TJ-Bilibili", Region: "天津", CP: "bilibili"},
			{ID: 2, DisplayName: "TJ-Baidu", Region: "天津", CP: "baidu"},
		},
	}
	svc := NewEDCTrafficScopeService(scopeRepo, &edcUserRepoStub{})

	scope, err := svc.ResolveEffectiveScope(9)
	if err != nil {
		t.Fatalf("ResolveEffectiveScope() error = %v", err)
	}
	if len(scope.AllowedEntityIDs) != 1 || scope.AllowedEntityIDs[0] != 1 {
		t.Fatalf("AllowedEntityIDs=%v, want [1]", scope.AllowedEntityIDs)
	}
}

func TestEDCServicePassesAllowedEntityIDsToTrafficQueries(t *testing.T) {
	repo := &edcRepoStub{}
	svc := NewEDCService(repo)
	start := time.Date(2026, 5, 1, 0, 0, 0, 0, time.Local)
	end := start.Add(time.Hour)

	_, err := svc.GetTrafficData(model.EDCTrafficFilter{
		StartTime:        start,
		EndTime:          end,
		EntityType:       "transmission",
		SrcRegion:        "北京市",
		DstRegion:        "天津市",
		AllowedEntityIDs: []uint64{10, 20},
	})
	if err != nil {
		t.Fatalf("GetTrafficData() error = %v", err)
	}
	if len(repo.trafficFilter.AllowedEntityIDs) != 2 || repo.trafficFilter.AllowedEntityIDs[0] != 10 || repo.trafficFilter.AllowedEntityIDs[1] != 20 {
		t.Fatalf("AllowedEntityIDs=%v, want [10 20]", repo.trafficFilter.AllowedEntityIDs)
	}
	if repo.trafficFilter.EntityType != "transmission" || repo.trafficFilter.SrcRegion != "北京市" || repo.trafficFilter.DstRegion != "天津市" {
		t.Fatalf("dimensions=%q/%q/%q", repo.trafficFilter.EntityType, repo.trafficFilter.SrcRegion, repo.trafficFilter.DstRegion)
	}
}

func TestEDCServicePassesEntityIDsAndAllowedEntityIDsToTrafficQueries(t *testing.T) {
	repo := &edcRepoStub{}
	svc := NewEDCService(repo)
	start := time.Date(2026, 5, 1, 0, 0, 0, 0, time.Local)
	end := start.Add(time.Hour)

	_, err := svc.GetTrafficData(model.EDCTrafficFilter{
		StartTime:        start,
		EndTime:          end,
		EntityIDs:        []uint64{1, 2},
		AllowedEntityIDs: []uint64{1, 2, 3},
	})
	if err != nil {
		t.Fatalf("GetTrafficData() error = %v", err)
	}
	if len(repo.trafficFilter.EntityIDs) != 2 || repo.trafficFilter.EntityIDs[0] != 1 || repo.trafficFilter.EntityIDs[1] != 2 {
		t.Fatalf("EntityIDs=%v, want [1 2]", repo.trafficFilter.EntityIDs)
	}
	if len(repo.trafficFilter.AllowedEntityIDs) != 3 || repo.trafficFilter.AllowedEntityIDs[0] != 1 || repo.trafficFilter.AllowedEntityIDs[2] != 3 {
		t.Fatalf("AllowedEntityIDs=%v, want [1 2 3]", repo.trafficFilter.AllowedEntityIDs)
	}
}
