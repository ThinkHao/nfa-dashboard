package service

import (
	"testing"

	"nfa-dashboard/internal/model"
)

type trafficScopeRuleRepoStub struct {
	groupsByUser map[uint64][]model.TrafficScopeRuleGroup
}

func (s *trafficScopeRuleRepoStub) ListByUser(userID uint64) ([]model.TrafficScopeRuleGroup, error) {
	return append([]model.TrafficScopeRuleGroup(nil), s.groupsByUser[userID]...), nil
}

func (s *trafficScopeRuleRepoStub) ReplaceByUser(userID uint64, groups []model.TrafficScopeRuleGroup) error {
	copied := make([]model.TrafficScopeRuleGroup, len(groups))
	copy(copied, groups)
	if s.groupsByUser == nil {
		s.groupsByUser = map[uint64][]model.TrafficScopeRuleGroup{}
	}
	s.groupsByUser[userID] = copied
	return nil
}

type trafficScopeSchoolRepoStub struct {
	matchesByConditionKey map[string][]model.School
	allSchools            []model.School
}

func (s *trafficScopeSchoolRepoStub) MatchSchools(dimension, value string) ([]model.School, error) {
	key := dimension + "|" + value
	return append([]model.School(nil), s.matchesByConditionKey[key]...), nil
}

func (s *trafficScopeSchoolRepoStub) GetSchoolsByKeys(keys []model.TrafficScopeSchoolKey) ([]model.School, error) {
	out := make([]model.School, 0, len(keys))
	known := map[string]model.School{}
	for _, school := range s.allSchools {
		known[school.SchoolID+"|"+school.Region+"|"+school.CP] = school
	}
	for _, schools := range s.matchesByConditionKey {
		for _, school := range schools {
			known[school.SchoolID+"|"+school.Region+"|"+school.CP] = school
		}
	}
	for _, key := range keys {
		school, ok := known[key.SchoolID+"|"+key.Region+"|"+key.CP]
		if !ok {
			continue
		}
		out = append(out, school)
	}
	return out, nil
}

func (s *trafficScopeSchoolRepoStub) ListAllSchools() ([]model.School, error) {
	return append([]model.School(nil), s.allSchools...), nil
}

func (s *trafficScopeSchoolRepoStub) ExpandSchoolIDsToKeys(ids []string) ([]model.TrafficScopeSchoolKey, error) {
	out := make([]model.TrafficScopeSchoolKey, 0)
	seen := map[string]struct{}{}
	for _, school := range s.allSchools {
		for _, id := range ids {
			if school.SchoolID != id {
				continue
			}
			key := school.SchoolID + "|" + school.Region + "|" + school.CP
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			out = append(out, model.TrafficScopeSchoolKey{
				SchoolID: school.SchoolID,
				Region:   school.Region,
				CP:       school.CP,
			})
		}
	}
	return out, nil
}

type trafficScopeLegacyRepoStub struct {
	legacy map[uint64][]string
}

func (s *trafficScopeLegacyRepoStub) SetSchoolOwner(schoolID string, userID *uint64) error {
	return nil
}

func (s *trafficScopeLegacyRepoStub) GetSchoolIDsByUser(userID uint64) ([]string, error) {
	return append([]string(nil), s.legacy[userID]...), nil
}

type trafficScopeUserRepoStub struct {
	existsByUser map[uint64]bool
	rolesByUser  map[uint64][]model.Role
}

func (s *trafficScopeUserRepoStub) GetByUsername(username string) (*model.User, error) {
	return nil, nil
}
func (s *trafficScopeUserRepoStub) GetByID(id uint64) (*model.User, error) { return nil, nil }
func (s *trafficScopeUserRepoStub) GetUserRoles(userID uint64) ([]model.Role, error) {
	return append([]model.Role(nil), s.rolesByUser[userID]...), nil
}
func (s *trafficScopeUserRepoStub) GetUserPermissions(userID uint64) ([]model.Permission, error) {
	return nil, nil
}
func (s *trafficScopeUserRepoStub) List(username string, status *int8, roles []string, page, pageSize int) ([]model.User, int64, error) {
	return nil, 0, nil
}
func (s *trafficScopeUserRepoStub) FindByIDs(ids []uint64) ([]model.User, error) { return nil, nil }
func (s *trafficScopeUserRepoStub) FindActiveByDisplayNames(names []string) ([]model.User, error) {
	return nil, nil
}
func (s *trafficScopeUserRepoStub) UsernameExists(username string) (bool, error) { return false, nil }
func (s *trafficScopeUserRepoStub) Create(u *model.User) (*model.User, error)    { return u, nil }
func (s *trafficScopeUserRepoStub) SetRoles(userID uint64, roleIDs []uint64) error {
	return nil
}
func (s *trafficScopeUserRepoStub) UpdateStatus(userID uint64, status int8) error { return nil }
func (s *trafficScopeUserRepoStub) UpdateAlias(userID uint64, alias *string) error {
	return nil
}
func (s *trafficScopeUserRepoStub) Exists(id uint64) (bool, error) {
	return s.existsByUser[id], nil
}

func TestResolveTrafficScope_DenyOverridesAllow(t *testing.T) {
	svc := NewTrafficScopeService(
		&trafficScopeRuleRepoStub{
			groupsByUser: map[uint64][]model.TrafficScopeRuleGroup{
				7: {
					{
						UserID:   7,
						RuleType: model.TrafficScopeRuleTypeAllow,
						Conditions: []model.TrafficScopeCondition{
							{DimensionType: model.TrafficScopeDimensionRegion, DimensionValue: "华东"},
						},
					},
					{
						UserID:   7,
						RuleType: model.TrafficScopeRuleTypeDeny,
						Conditions: []model.TrafficScopeCondition{
							{DimensionType: model.TrafficScopeDimensionSchool, DimensionValue: "school-b"},
						},
					},
				},
			},
		},
		&trafficScopeSchoolRepoStub{
			matchesByConditionKey: map[string][]model.School{
				"region|华东": {
					{SchoolID: "school-a", SchoolName: "A", Region: "华东", CP: "cmcc"},
					{SchoolID: "school-b", SchoolName: "B", Region: "华东", CP: "ctcc"},
				},
				"school|school-b": {
					{SchoolID: "school-b", SchoolName: "B", Region: "华东", CP: "ctcc"},
				},
			},
		},
		&trafficScopeLegacyRepoStub{},
		nil,
	)

	scope, err := svc.ResolveEffectiveScope(7)
	if err != nil {
		t.Fatalf("ResolveEffectiveScope() error = %v", err)
	}

	if scope.Source != model.TrafficScopeSourcePolicyRule {
		t.Fatalf("expected source %q, got %q", model.TrafficScopeSourcePolicyRule, scope.Source)
	}
	if len(scope.AllowedSchoolKeys) != 1 || scope.AllowedSchoolKeys[0].SchoolID != "school-a" || scope.AllowedSchoolKeys[0].CP != "cmcc" {
		t.Fatalf("expected only school-a/cmcc to remain after deny override, got %#v", scope.AllowedSchoolKeys)
	}
}

func TestResolveTrafficScope_IntersectsConditionsAcrossDimensions(t *testing.T) {
	svc := NewTrafficScopeService(
		&trafficScopeRuleRepoStub{
			groupsByUser: map[uint64][]model.TrafficScopeRuleGroup{
				9: {
					{
						UserID:   9,
						RuleType: model.TrafficScopeRuleTypeAllow,
						Conditions: []model.TrafficScopeCondition{
							{DimensionType: model.TrafficScopeDimensionRegion, DimensionValue: "华东"},
							{DimensionType: model.TrafficScopeDimensionCP, DimensionValue: "CMCC"},
						},
					},
				},
			},
		},
		&trafficScopeSchoolRepoStub{
			matchesByConditionKey: map[string][]model.School{
				"region|华东": {
					{SchoolID: "school-a", Region: "华东", CP: "ctcc"},
					{SchoolID: "school-b", Region: "华东", CP: "cmcc"},
					{SchoolID: "school-b", Region: "华东", CP: "bilibili"},
				},
				"cp|CMCC": {
					{SchoolID: "school-b", Region: "华东", CP: "cmcc"},
					{SchoolID: "school-c", Region: "华南", CP: "cmcc"},
				},
			},
		},
		&trafficScopeLegacyRepoStub{},
		nil,
	)

	scope, err := svc.ResolveEffectiveScope(9)
	if err != nil {
		t.Fatalf("ResolveEffectiveScope() error = %v", err)
	}

	if len(scope.AllowedSchoolKeys) != 1 || scope.AllowedSchoolKeys[0].SchoolID != "school-b" || scope.AllowedSchoolKeys[0].CP != "cmcc" {
		t.Fatalf("expected region/cp intersection to keep only school-b/cmcc, got %#v", scope.AllowedSchoolKeys)
	}
}

func TestResolveTrafficScope_UnionsValuesWithinSameDimension(t *testing.T) {
	svc := NewTrafficScopeService(
		&trafficScopeRuleRepoStub{
			groupsByUser: map[uint64][]model.TrafficScopeRuleGroup{
				10: {
					{
						UserID:   10,
						RuleType: model.TrafficScopeRuleTypeAllow,
						Conditions: []model.TrafficScopeCondition{
							{DimensionType: model.TrafficScopeDimensionRegion, DimensionValue: "华东"},
							{DimensionType: model.TrafficScopeDimensionRegion, DimensionValue: "华南"},
							{DimensionType: model.TrafficScopeDimensionCP, DimensionValue: "CMCC"},
							{DimensionType: model.TrafficScopeDimensionCP, DimensionValue: "CTCC"},
						},
					},
				},
			},
		},
		&trafficScopeSchoolRepoStub{
			matchesByConditionKey: map[string][]model.School{
				"region|华东": {
					{SchoolID: "school-a", Region: "华东", CP: "bilibili"},
					{SchoolID: "school-b", Region: "华东", CP: "cmcc"},
				},
				"region|华南": {
					{SchoolID: "school-c", Region: "华南", CP: "ctcc"},
				},
				"cp|CMCC": {
					{SchoolID: "school-b", Region: "华东", CP: "cmcc"},
				},
				"cp|CTCC": {
					{SchoolID: "school-c", Region: "华南", CP: "ctcc"},
					{SchoolID: "school-d", Region: "华北", CP: "ctcc"},
				},
			},
		},
		&trafficScopeLegacyRepoStub{},
		nil,
	)

	scope, err := svc.ResolveEffectiveScope(10)
	if err != nil {
		t.Fatalf("ResolveEffectiveScope() error = %v", err)
	}

	if len(scope.AllowedSchoolKeys) != 2 || scope.AllowedSchoolKeys[0].SchoolID != "school-b" || scope.AllowedSchoolKeys[1].SchoolID != "school-c" {
		t.Fatalf("expected same-dimension unions then cross-dimension intersection, got %#v", scope.AllowedSchoolKeys)
	}
}

func TestResolveTrafficScope_PolicyRulesOverrideLegacyBindings(t *testing.T) {
	svc := NewTrafficScopeService(
		&trafficScopeRuleRepoStub{
			groupsByUser: map[uint64][]model.TrafficScopeRuleGroup{
				11: {
					{
						UserID:   11,
						RuleType: model.TrafficScopeRuleTypeAllow,
						Conditions: []model.TrafficScopeCondition{
							{DimensionType: model.TrafficScopeDimensionSchool, DimensionValue: "school-c"},
						},
					},
				},
			},
		},
		&trafficScopeSchoolRepoStub{
			matchesByConditionKey: map[string][]model.School{
				"school|school-c": {
					{SchoolID: "school-c", SchoolName: "C", Region: "华北", CP: "cmcc"},
				},
			},
		},
		&trafficScopeLegacyRepoStub{
			legacy: map[uint64][]string{
				11: {"school-legacy"},
			},
		},
		nil,
	)

	scope, err := svc.ResolveEffectiveScope(11)
	if err != nil {
		t.Fatalf("ResolveEffectiveScope() error = %v", err)
	}

	if scope.Source != model.TrafficScopeSourcePolicyRule {
		t.Fatalf("expected policy source, got %q", scope.Source)
	}
	if len(scope.AllowedSchoolKeys) != 1 || scope.AllowedSchoolKeys[0].SchoolID != "school-c" {
		t.Fatalf("expected policy groups to override legacy schools, got %#v", scope.AllowedSchoolKeys)
	}
}

func TestResolveTrafficScope_FallsBackToLegacyBindingsWhenNoPolicy(t *testing.T) {
	svc := NewTrafficScopeService(
		&trafficScopeRuleRepoStub{
			groupsByUser: map[uint64][]model.TrafficScopeRuleGroup{},
		},
		&trafficScopeSchoolRepoStub{
			allSchools: []model.School{
				{SchoolID: "school-x", Region: "山东省", CP: "bilibili"},
				{SchoolID: "school-x", Region: "山东省", CP: "ctcc"},
				{SchoolID: "school-y", Region: "北京市", CP: "cmcc"},
			},
		},
		&trafficScopeLegacyRepoStub{
			legacy: map[uint64][]string{
				12: {"school-x", "school-y"},
			},
		},
		nil,
	)

	scope, err := svc.ResolveEffectiveScope(12)
	if err != nil {
		t.Fatalf("ResolveEffectiveScope() error = %v", err)
	}

	if scope.Source != model.TrafficScopeSourceLegacyUserSchool {
		t.Fatalf("expected legacy source, got %q", scope.Source)
	}
	if len(scope.AllowedSchoolKeys) != 3 {
		t.Fatalf("expected legacy school ids to expand into all composite rows, got %#v", scope.AllowedSchoolKeys)
	}
}

func TestResolveTrafficScope_FallsBackToAdminDefaultScope(t *testing.T) {
	svc := NewTrafficScopeService(
		&trafficScopeRuleRepoStub{
			groupsByUser: map[uint64][]model.TrafficScopeRuleGroup{},
		},
		&trafficScopeSchoolRepoStub{
			allSchools: []model.School{
				{SchoolID: "school-a", SchoolName: "A", Region: "山东省", CP: "bilibili"},
				{SchoolID: "school-b", SchoolName: "B", Region: "北京市", CP: "cmcc"},
			},
		},
		&trafficScopeLegacyRepoStub{},
		&trafficScopeUserRepoStub{
			rolesByUser: map[uint64][]model.Role{
				1: {
					{Name: "admin"},
				},
			},
		},
	)

	scope, err := svc.ResolveEffectiveScope(1)
	if err != nil {
		t.Fatalf("ResolveEffectiveScope() error = %v", err)
	}

	if scope.Source != model.TrafficScopeSourceDefaultAdminRole {
		t.Fatalf("expected admin default source, got %q", scope.Source)
	}
	if len(scope.AllowedSchoolKeys) != 2 {
		t.Fatalf("expected all schools for admin default scope, got %#v", scope.AllowedSchoolKeys)
	}
}

func TestResolveTrafficScope_ReturnsNoneWhenNoPolicyLegacyOrAdminDefault(t *testing.T) {
	svc := NewTrafficScopeService(
		&trafficScopeRuleRepoStub{
			groupsByUser: map[uint64][]model.TrafficScopeRuleGroup{},
		},
		&trafficScopeSchoolRepoStub{},
		&trafficScopeLegacyRepoStub{},
		&trafficScopeUserRepoStub{},
	)

	scope, err := svc.ResolveEffectiveScope(99)
	if err != nil {
		t.Fatalf("ResolveEffectiveScope() error = %v", err)
	}

	if scope.Source != model.TrafficScopeSourceNone {
		t.Fatalf("expected none source, got %q", scope.Source)
	}
	if len(scope.AllowedSchoolKeys) != 0 {
		t.Fatalf("expected no allowed school keys, got %#v", scope.AllowedSchoolKeys)
	}
}

func TestPreview_UsesCompositeKeysInsteadOfSchoolIDExpansion(t *testing.T) {
	svc := NewTrafficScopeService(
		&trafficScopeRuleRepoStub{
			groupsByUser: map[uint64][]model.TrafficScopeRuleGroup{
				88: {
					{
						UserID:   88,
						RuleType: model.TrafficScopeRuleTypeAllow,
						Conditions: []model.TrafficScopeCondition{
							{DimensionType: model.TrafficScopeDimensionRegion, DimensionValue: "山东省"},
							{DimensionType: model.TrafficScopeDimensionCP, DimensionValue: "bilibili"},
						},
					},
				},
			},
		},
		&trafficScopeSchoolRepoStub{
			matchesByConditionKey: map[string][]model.School{
				"region|山东省": {
					{SchoolID: "1138", SchoolName: "中国海洋大学", Region: "山东省", CP: "bilibili"},
					{SchoolID: "1138", SchoolName: "中国海洋大学", Region: "山东省", CP: "cnc"},
				},
				"cp|bilibili": {
					{SchoolID: "1138", SchoolName: "中国海洋大学", Region: "山东省", CP: "bilibili"},
				},
			},
			allSchools: []model.School{
				{SchoolID: "1138", SchoolName: "中国海洋大学", Region: "山东省", CP: "bilibili"},
				{SchoolID: "1138", SchoolName: "中国海洋大学", Region: "山东省", CP: "cnc"},
			},
		},
		&trafficScopeLegacyRepoStub{},
		nil,
	)

	preview, err := svc.Preview(88)
	if err != nil {
		t.Fatalf("Preview() error = %v", err)
	}

	if len(preview.AllowedSchools) != 1 || preview.AllowedSchools[0].CP != "bilibili" {
		t.Fatalf("expected preview to keep only 山东省+bilibili row, got %#v", preview.AllowedSchools)
	}
}

func TestReplaceRules_NormalizesAndDropsEmptyGroups(t *testing.T) {
	repo := &trafficScopeRuleRepoStub{}
	svc := NewTrafficScopeService(
		repo,
		&trafficScopeSchoolRepoStub{},
		&trafficScopeLegacyRepoStub{},
		&trafficScopeUserRepoStub{
			existsByUser: map[uint64]bool{77: true},
		},
	)

	err := svc.ReplaceRules(77, []model.TrafficScopeRuleGroup{
		{
			RuleType: " allow ",
			Conditions: []model.TrafficScopeCondition{
				{DimensionType: " region ", DimensionValue: " 华东 "},
				{DimensionType: "cp", DimensionValue: "   "},
			},
		},
		{
			RuleType: "deny",
			Conditions: []model.TrafficScopeCondition{
				{DimensionType: "school", DimensionValue: "  "},
			},
		},
	})
	if err != nil {
		t.Fatalf("ReplaceRules() error = %v", err)
	}

	stored := repo.groupsByUser[77]
	if len(stored) != 1 {
		t.Fatalf("expected one normalized group, got %#v", stored)
	}
	if stored[0].RuleType != model.TrafficScopeRuleTypeAllow {
		t.Fatalf("expected normalized rule type allow, got %q", stored[0].RuleType)
	}
	if len(stored[0].Conditions) != 1 || stored[0].Conditions[0].DimensionType != model.TrafficScopeDimensionRegion || stored[0].Conditions[0].DimensionValue != "华东" {
		t.Fatalf("expected trimmed region condition, got %#v", stored[0].Conditions)
	}
}
