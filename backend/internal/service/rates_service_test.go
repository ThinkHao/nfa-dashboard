package service

import (
	"testing"
	"time"

	"nfa-dashboard/internal/model"
)

type ratesServiceUserRepoStub struct {
	findByIDsResult                 []model.User
	findByIDsErr                    error
	findActiveByDisplayNamesResult  []model.User
	findActiveByDisplayNamesErr     error
	findActiveByDisplayNamesLastArg []string
	usernameExists                  map[string]bool
	createdUsers                    []*model.User
}

func (s *ratesServiceUserRepoStub) GetByUsername(username string) (*model.User, error) {
	return nil, nil
}

func (s *ratesServiceUserRepoStub) GetByID(id uint64) (*model.User, error) { return nil, nil }

func (s *ratesServiceUserRepoStub) GetUserRoles(userID uint64) ([]model.Role, error) {
	return nil, nil
}

func (s *ratesServiceUserRepoStub) GetUserPermissions(userID uint64) ([]model.Permission, error) {
	return nil, nil
}

func (s *ratesServiceUserRepoStub) List(username string, status *int8, roles []string, page, pageSize int) ([]model.User, int64, error) {
	return nil, 0, nil
}

func (s *ratesServiceUserRepoStub) FindByIDs(ids []uint64) ([]model.User, error) {
	return s.findByIDsResult, s.findByIDsErr
}

func (s *ratesServiceUserRepoStub) FindActiveByDisplayNames(names []string) ([]model.User, error) {
	s.findActiveByDisplayNamesLastArg = append([]string(nil), names...)
	return s.findActiveByDisplayNamesResult, s.findActiveByDisplayNamesErr
}

func (s *ratesServiceUserRepoStub) UsernameExists(username string) (bool, error) {
	return s.usernameExists[username], nil
}

func (s *ratesServiceUserRepoStub) Create(u *model.User) (*model.User, error) {
	cp := *u
	s.createdUsers = append(s.createdUsers, &cp)
	return &cp, nil
}

func (s *ratesServiceUserRepoStub) SetRoles(userID uint64, roleIDs []uint64) error { return nil }

func (s *ratesServiceUserRepoStub) UpdateStatus(userID uint64, status int8) error { return nil }

func (s *ratesServiceUserRepoStub) UpdateAlias(userID uint64, alias *string) error { return nil }

func (s *ratesServiceUserRepoStub) Exists(id uint64) (bool, error) { return false, nil }

func TestNormalizeIncrementConfig_NoIncrementStartForcesDefault(t *testing.T) {
	stock := 0.3
	increment := 0.4
	rate := &model.RateCustomer{
		StockRatio:     &stock,
		IncrementRatio: &increment,
	}

	if err := normalizeIncrementConfig(rate); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if rate.StockRatio == nil || *rate.StockRatio != 1 {
		t.Fatalf("expected stock_ratio=1, got %+v", rate.StockRatio)
	}
	if rate.IncrementRatio == nil || *rate.IncrementRatio != 0 {
		t.Fatalf("expected increment_ratio=0, got %+v", rate.IncrementRatio)
	}
}

func TestNormalizeIncrementConfig_IncrementStartAllowsIndependentRatios(t *testing.T) {
	now := time.Now()
	stock := 0.9
	increment := 0.9
	rate := &model.RateCustomer{
		IncrementStartAt: &now,
		StockRatio:       &stock,
		IncrementRatio:   &increment,
	}

	if err := normalizeIncrementConfig(rate); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestNormalizeIncrementConfig_IncrementStartOneSideMissingAutoFillZero(t *testing.T) {
	now := time.Now()
	stock := 0.3
	rate := &model.RateCustomer{
		IncrementStartAt: &now,
		StockRatio:       &stock,
	}

	if err := normalizeIncrementConfig(rate); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if rate.IncrementRatio == nil || *rate.IncrementRatio != 0 {
		t.Fatalf("expected increment_ratio=0, got %+v", rate.IncrementRatio)
	}
}

func TestNormalizeIncrementConfig_IncrementStartBothMissingShouldFail(t *testing.T) {
	now := time.Now()
	rate := &model.RateCustomer{
		IncrementStartAt: &now,
	}

	if err := normalizeIncrementConfig(rate); err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestNormalizeIncrementConfig_RatioOutOfRangeShouldFail(t *testing.T) {
	now := time.Now()
	stock := 1.1
	increment := 0.2
	rate := &model.RateCustomer{
		IncrementStartAt: &now,
		StockRatio:       &stock,
		IncrementRatio:   &increment,
	}

	if err := normalizeIncrementConfig(rate); err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestResolveCustomerRateOwnerIDsByDisplayName_PrefersVisibleNamesOverHiddenIDs(t *testing.T) {
	repo := &ratesServiceUserRepoStub{
		findActiveByDisplayNamesResult: []model.User{
			{ID: 42, Username: "alice-login"},
			{ID: 77, Username: "line-login"},
		},
	}
	svc := &ratesService{userRepo: repo}

	customerID := uint64(999)
	lineID := uint64(888)
	rate := &model.RateCustomer{
		CustomerFeeOwnerID:    &customerID,
		NetworkLineFeeOwnerID: &lineID,
	}
	err := svc.ResolveCustomerRateOwnerIDsByDisplayName(rate, CustomerRateOwnerNames{
		CustomerFeeOwnerName:    "alice-login",
		NetworkLineFeeOwnerName: "line-login",
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(repo.findActiveByDisplayNamesLastArg) != 2 {
		t.Fatalf("expected 2 names queried, got %v", repo.findActiveByDisplayNamesLastArg)
	}
	if rate.CustomerFeeOwnerID == nil || *rate.CustomerFeeOwnerID != 42 {
		t.Fatalf("expected customer owner id 42, got %+v", rate.CustomerFeeOwnerID)
	}
	if rate.NetworkLineFeeOwnerID == nil || *rate.NetworkLineFeeOwnerID != 77 {
		t.Fatalf("expected line owner id 77, got %+v", rate.NetworkLineFeeOwnerID)
	}
}

func TestResolveCustomerRateOwnerIDsByDisplayName_MissingUserReturnsReadableError(t *testing.T) {
	repo := &ratesServiceUserRepoStub{}
	svc := &ratesService{userRepo: repo}
	rate := &model.RateCustomer{}

	err := svc.ResolveCustomerRateOwnerIDsByDisplayName(rate, CustomerRateOwnerNames{
		CustomerFeeOwnerName: "刘旭阳",
	})
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if got := err.Error(); got != "客户费归属 未匹配到系统用户：刘旭阳" {
		t.Fatalf("unexpected error: %s", got)
	}
}

func TestNormalizeImportUsernameBase_UsesPinyinRule(t *testing.T) {
	if got := normalizeImportUsernameBase("陈金荣"); got != "chenjr" {
		t.Fatalf("expected chenjr, got %s", got)
	}
}

func TestPreviewCustomerRateImportUsers_AppendsSuffixWhenUsernameExists(t *testing.T) {
	repo := &ratesServiceUserRepoStub{
		usernameExists: map[string]bool{
			"chenjr": true,
		},
	}
	svc := &ratesService{userRepo: repo}

	items, err := svc.PreviewCustomerRateImportUsers([]string{"陈金荣"})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
	if items[0].SuggestedUsername != "chenjr2" {
		t.Fatalf("expected chenjr2, got %s", items[0].SuggestedUsername)
	}
}

func TestCreateCustomerRateImportUsers_CreatesEnabledUserWithAliasAndPassword(t *testing.T) {
	repo := &ratesServiceUserRepoStub{usernameExists: map[string]bool{}}
	svc := &ratesService{userRepo: repo}

	created, err := svc.CreateCustomerRateImportUsers([]MissingImportUser{{Alias: "陈金荣"}})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(created) != 1 {
		t.Fatalf("expected 1 created user, got %d", len(created))
	}
	if created[0].Alias != "陈金荣" {
		t.Fatalf("expected alias 陈金荣, got %s", created[0].Alias)
	}
	if created[0].Username != "chenjr" {
		t.Fatalf("expected username chenjr, got %s", created[0].Username)
	}
	if len(created[0].Password) != 12 {
		t.Fatalf("expected password length 12, got %d", len(created[0].Password))
	}
	if len(repo.createdUsers) != 1 {
		t.Fatalf("expected repository create called once, got %d", len(repo.createdUsers))
	}
	if repo.createdUsers[0].Alias == nil || *repo.createdUsers[0].Alias != "陈金荣" {
		t.Fatalf("expected alias copied into created user, got %+v", repo.createdUsers[0].Alias)
	}
	if repo.createdUsers[0].Status != 1 {
		t.Fatalf("expected created user status=1, got %d", repo.createdUsers[0].Status)
	}
	if repo.createdUsers[0].PasswordHash == "" {
		t.Fatalf("expected password hash to be set")
	}
}
