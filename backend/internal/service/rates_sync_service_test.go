package service

import (
	"context"
	"testing"

	"nfa-dashboard/internal/model"
)

func TestParseActionsSet_TemplateFinalFeeMapsToChannelRate(t *testing.T) {
	data := []byte(`{"type":"template","values":{"final_fee":0.12}}`)

	got, err := parseActionsSet(data)
	if err != nil {
		t.Fatalf("parseActionsSet returned error: %v", err)
	}

	if _, ok := got["final_fee"]; ok {
		t.Fatalf("expected final_fee to be normalized away, got map: %#v", got)
	}
	if got["channel_rate"] != 0.12 {
		t.Fatalf("expected channel_rate=0.12, got %#v", got["channel_rate"])
	}
}

func TestParseActionsSet_TemplateChannelRatePreferredOverFinalFee(t *testing.T) {
	data := []byte(`{"type":"template","values":{"channel_rate":0.23,"final_fee":0.11}}`)

	got, err := parseActionsSet(data)
	if err != nil {
		t.Fatalf("parseActionsSet returned error: %v", err)
	}

	if _, ok := got["final_fee"]; ok {
		t.Fatalf("expected final_fee to be normalized away, got map: %#v", got)
	}
	if got["channel_rate"] != 0.23 {
		t.Fatalf("expected channel_rate to keep explicit value 0.23, got %#v", got["channel_rate"])
	}
}

func TestApplyRuleToCustomer_ChannelRateAlwaysUpdatesTopField(t *testing.T) {
	svc := &ratesSyncService{}
	rc := &model.RateCustomer{}
	rule := model.RateCustomerSyncRule{OverwriteStrategy: "always"}

	updated, updates, err := svc.applyRuleToCustomer(rc, rule, nil, map[string]interface{}{"channel_rate": 0.66}, "r", "c", "s")
	if err != nil {
		t.Fatalf("applyRuleToCustomer returned error: %v", err)
	}
	if !updated {
		t.Fatalf("expected update=true")
	}
	if rc.ChannelRate == nil || *rc.ChannelRate != 0.66 {
		t.Fatalf("expected rc.ChannelRate=0.66, got %+v", rc.ChannelRate)
	}
	if v, ok := updates["channel_rate"]; !ok || v != 0.66 {
		t.Fatalf("expected updates[channel_rate]=0.66, got %#v", updates)
	}
}

func TestApplyRuleToCustomer_ChannelRateIfEmptyRespectsExistingValue(t *testing.T) {
	svc := &ratesSyncService{}
	current := 0.31
	rc := &model.RateCustomer{ChannelRate: &current}
	rule := model.RateCustomerSyncRule{OverwriteStrategy: "if_empty"}

	updated, _, err := svc.applyRuleToCustomer(rc, rule, nil, map[string]interface{}{"channel_rate": 0.99}, "r", "c", "s")
	if err != nil {
		t.Fatalf("applyRuleToCustomer returned error: %v", err)
	}
	if updated {
		t.Fatalf("expected update=false when channel_rate already exists")
	}
	if rc.ChannelRate == nil || *rc.ChannelRate != 0.31 {
		t.Fatalf("expected rc.ChannelRate unchanged=0.31, got %+v", rc.ChannelRate)
	}
}

func TestExecuteSync_InsertsMissingRatesWithoutUpdatingExistingRates(t *testing.T) {
	ruleActions := []byte(`{"type":"template","values":{"customer_fee":0.88}}`)
	schoolName := "existing-school"
	existingFee := 0.11
	ratesRepo := &ratesSyncRatesRepoStub{
		existing: map[string]model.RateCustomer{
			"north|ct|existing-school": {
				ID:          7,
				Region:      "north",
				CP:          "ct",
				SchoolName:  &schoolName,
				CustomerFee: &existingFee,
			},
		},
	}
	svc := NewRatesSyncService(
		&ratesSyncRulesRepoStub{rules: []model.RateCustomerSyncRule{{
			ID:                3,
			Name:              "template",
			Enabled:           true,
			OverwriteStrategy: "always",
			Actions:           ruleActions,
		}}},
		ratesRepo,
		&ratesSyncSchoolRepoStub{schools: []model.School{
			{Region: "north", CP: "ct", SchoolName: "existing-school"},
			{Region: "north", CP: "ct", SchoolName: "new-school"},
		}},
	)

	affected, err := svc.ExecuteSync()
	if err != nil {
		t.Fatalf("ExecuteSync returned error: %v", err)
	}
	if affected != 1 {
		t.Fatalf("expected only one inserted row to be affected, got %d", affected)
	}
	if ratesRepo.updateCalls != 0 {
		t.Fatalf("expected existing rows not to be updated, got %d update calls", ratesRepo.updateCalls)
	}
	if len(ratesRepo.inserted) != 1 {
		t.Fatalf("expected one inserted row, got %d", len(ratesRepo.inserted))
	}
	if got := ratesRepo.inserted[0]; got.Region != "north" || got.CP != "ct" || got.SchoolName == nil || *got.SchoolName != "new-school" {
		t.Fatalf("unexpected inserted row: %+v", got)
	}
}

type ratesSyncRulesRepoStub struct {
	rules []model.RateCustomerSyncRule
}

func (s *ratesSyncRulesRepoStub) List(filter map[string]interface{}, limit, offset int) ([]model.RateCustomerSyncRule, int64, error) {
	return s.rules, int64(len(s.rules)), nil
}
func (s *ratesSyncRulesRepoStub) ListDistinctCustomerRegions() ([]string, error) { return nil, nil }
func (s *ratesSyncRulesRepoStub) ListDistinctCustomerCPs() ([]string, error)     { return nil, nil }
func (s *ratesSyncRulesRepoStub) Create(rule *model.RateCustomerSyncRule) (*model.RateCustomerSyncRule, error) {
	return rule, nil
}
func (s *ratesSyncRulesRepoStub) Update(id uint64, updates map[string]interface{}) error { return nil }
func (s *ratesSyncRulesRepoStub) Delete(id uint64) error                                 { return nil }
func (s *ratesSyncRulesRepoStub) UpdatePriority(id uint64, priority int) error           { return nil }
func (s *ratesSyncRulesRepoStub) SetEnabled(id uint64, enabled bool) error               { return nil }

type ratesSyncSchoolRepoStub struct {
	schools []model.School
}

func (s *ratesSyncSchoolRepoStub) GetAllSchools(filter map[string]interface{}, limit, offset int) ([]model.School, int64, error) {
	if offset >= len(s.schools) {
		return []model.School{}, int64(len(s.schools)), nil
	}
	end := offset + limit
	if end > len(s.schools) {
		end = len(s.schools)
	}
	return s.schools[offset:end], int64(len(s.schools)), nil
}
func (s *ratesSyncSchoolRepoStub) GetAllRegions() ([]string, error) { return nil, nil }
func (s *ratesSyncSchoolRepoStub) GetAllCPs() ([]string, error)     { return nil, nil }
func (s *ratesSyncSchoolRepoStub) GetRegionsWithScope(allowedSchoolKeys []model.TrafficScopeSchoolKey) ([]string, error) {
	return nil, nil
}
func (s *ratesSyncSchoolRepoStub) GetCPsWithScope(allowedSchoolKeys []model.TrafficScopeSchoolKey) ([]string, error) {
	return nil, nil
}
func (s *ratesSyncSchoolRepoStub) GetTrafficData(ctx context.Context, filter model.TrafficFilter) ([]model.TrafficResponse, error) {
	return nil, nil
}
func (s *ratesSyncSchoolRepoStub) GetTrafficSummary(ctx context.Context, filter model.TrafficFilter) (model.TrafficResponse, error) {
	return model.TrafficResponse{}, nil
}
func (s *ratesSyncSchoolRepoStub) ExistsBySchoolID(schoolID string) (bool, error) { return false, nil }

type ratesSyncRatesRepoStub struct {
	existing    map[string]model.RateCustomer
	inserted    []model.RateCustomer
	updateCalls int
}

func (s *ratesSyncRatesRepoStub) ListCustomerRates(filter map[string]interface{}, limit, offset int) ([]model.RateCustomer, int64, error) {
	key := ratesSyncKey(filter["region"], filter["cp"], filter["school_name"])
	if item, ok := s.existing[key]; ok {
		return []model.RateCustomer{item}, 1, nil
	}
	return []model.RateCustomer{}, 0, nil
}
func (s *ratesSyncRatesRepoStub) ListCustomerRateKeys(filter map[string]interface{}) (map[string]struct{}, error) {
	out := map[string]struct{}{}
	for key := range s.existing {
		out[key] = struct{}{}
	}
	return out, nil
}
func (s *ratesSyncRatesRepoStub) UpsertCustomerRate(rate *model.RateCustomer) error {
	if rate != nil {
		s.inserted = append(s.inserted, *rate)
	}
	return nil
}
func (s *ratesSyncRatesRepoStub) CreateCustomerRateIfMissing(rate *model.RateCustomer) (bool, error) {
	if rate == nil || rate.SchoolName == nil {
		return false, nil
	}
	key := ratesSyncKey(rate.Region, rate.CP, *rate.SchoolName)
	if _, ok := s.existing[key]; ok {
		return false, nil
	}
	s.existing[key] = *rate
	s.inserted = append(s.inserted, *rate)
	return true, nil
}
func (s *ratesSyncRatesRepoStub) UpdateCustomerByID(id uint64, updates map[string]interface{}) error {
	s.updateCalls++
	return nil
}
func (s *ratesSyncRatesRepoStub) ListNodeRates(filter map[string]interface{}, limit, offset int) ([]model.RateNode, int64, error) {
	return nil, 0, nil
}
func (s *ratesSyncRatesRepoStub) UpsertNodeRate(rate *model.RateNode) error { return nil }
func (s *ratesSyncRatesRepoStub) ListNodeSettlementGroups(filter map[string]interface{}, limit, offset int) ([]model.EDCNodeSettlementGroup, int64, error) {
	return nil, 0, nil
}
func (s *ratesSyncRatesRepoStub) SaveNodeSettlementGroup(group *model.EDCNodeSettlementGroup, memberIDs []uint64) error {
	return nil
}
func (s *ratesSyncRatesRepoStub) DisableNodeSettlementGroup(id uint64) error { return nil }
func (s *ratesSyncRatesRepoStub) ListEnabledNodeSettlementGroups() ([]model.EDCNodeSettlementGroup, error) {
	return nil, nil
}
func (s *ratesSyncRatesRepoStub) ListFinalCustomerRates(filter map[string]interface{}, limit, offset int) ([]model.RateFinalCustomer, int64, error) {
	return nil, 0, nil
}
func (s *ratesSyncRatesRepoStub) UpsertFinalCustomerRate(rate *model.RateFinalCustomer) error {
	return nil
}
func (s *ratesSyncRatesRepoStub) ListFinalNodeRates(filter map[string]interface{}, limit, offset int) ([]model.RateFinalNode, int64, error) {
	return nil, 0, nil
}
func (s *ratesSyncRatesRepoStub) UpsertFinalNodeRate(rate *model.RateFinalNode) error { return nil }
func (s *ratesSyncRatesRepoStub) SyncFinalNodeRateFromNode(rate *model.RateNode) (bool, error) {
	return true, nil
}
func (s *ratesSyncRatesRepoStub) InitFinalNodeRatesFromNode() (int64, error) { return 0, nil }
func (s *ratesSyncRatesRepoStub) RefreshFinalNodeRates() (int64, error)      { return 0, nil }
func (s *ratesSyncRatesRepoStub) ListAllFinalNodeRates() ([]model.RateFinalNode, error) {
	return nil, nil
}
func (s *ratesSyncRatesRepoStub) InitFinalCustomerRatesFromCustomer() (int64, error) {
	return 0, nil
}
func (s *ratesSyncRatesRepoStub) RefreshFinalCustomerRates() (int64, error) { return 0, nil }
func (s *ratesSyncRatesRepoStub) CleanupInvalidFinalCustomerRates() (int64, error) {
	return 0, nil
}
func (s *ratesSyncRatesRepoStub) GetFinalCustomerRate(region, cp, schoolName string) (*model.RateFinalCustomer, error) {
	return nil, nil
}
func (s *ratesSyncRatesRepoStub) ListDistinctCustomerRegions() ([]string, error) { return nil, nil }
func (s *ratesSyncRatesRepoStub) ListDistinctCustomerCPs() ([]string, error)     { return nil, nil }

func ratesSyncKey(region, cp, school interface{}) string {
	return region.(string) + "|" + cp.(string) + "|" + school.(string)
}
