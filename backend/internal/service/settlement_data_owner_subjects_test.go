package service

import (
	"context"
	"nfa-dashboard/internal/model"
	"nfa-dashboard/internal/repository"
	"testing"
	"time"
)

type ownerIDsRepoStub struct {
	repository.SettlementDataRepository
	ids              []uint64
	filter           map[string]interface{}
	legacyListCalled bool
}

func (s *ownerIDsRepoStub) ListDistinctOwnerUserIDs(_ context.Context, filter map[string]interface{}) ([]uint64, error) {
	s.filter = filter
	return s.ids, nil
}

func (s *ownerIDsRepoStub) ListSettlementCustomer(_ context.Context, _ map[string]interface{}, _, _ int) ([]model.SettlementCustomer, int64, error) {
	s.legacyListCalled = true
	return []model.SettlementCustomer{}, 0, nil
}

type ownerUsersRepoStub struct {
	repository.UserRepository
	users     []model.User
	requested []uint64
}

func (s *ownerUsersRepoStub) FindByIDs(ids []uint64) ([]model.User, error) {
	s.requested = append([]uint64(nil), ids...)
	return s.users, nil
}

func TestListUsedOwnerSubjectsUsesDistinctOwnerIDs(t *testing.T) {
	start := time.Date(2026, 5, 1, 0, 0, 0, 0, time.Local)
	end := time.Date(2026, 7, 31, 23, 59, 59, 0, time.Local)
	ownerEntityID := uint64(11)
	channelOwnerUserID := uint64(13)
	alias := "张三"
	repo := &ownerIDsRepoStub{ids: []uint64{7, 9}}
	users := &ownerUsersRepoStub{users: []model.User{
		{ID: 7, Username: "zhangsan", Alias: &alias},
		{ID: 9, Username: "lisi"},
	}}
	svc := NewSettlementDataService(repo, users, nil, nil)

	items, err := svc.ListUsedOwnerSubjects(context.Background(), SettlementCustomerFilter{
		Region: "北京市", CP: "bilibili", School: "大学", Start: &start, End: &end,
		OwnerEntityID: &ownerEntityID, ChannelOwnerUserID: &channelOwnerUserID,
	})
	if err != nil {
		t.Fatalf("ListUsedOwnerSubjects returned error: %v", err)
	}
	if repo.legacyListCalled {
		t.Fatal("expected lightweight distinct owner query, but legacy full settlement list was used")
	}
	if len(users.requested) != 2 || users.requested[0] != 7 || users.requested[1] != 9 {
		t.Fatalf("FindByIDs called with %#v, want [7 9]", users.requested)
	}
	if len(items) != 2 || items[0].Label != "张三" || items[1].Label != "lisi" {
		t.Fatalf("unexpected owner subjects: %#v", items)
	}
	if repo.filter["region"] != "北京市" || repo.filter["cp"] != "bilibili" || repo.filter["school_name"] != "大学" {
		t.Fatalf("unexpected filter: %#v", repo.filter)
	}
	if repo.filter["start_service_date"] != start || repo.filter["end_service_date"] != end {
		t.Fatalf("date range missing from filter: %#v", repo.filter)
	}
	if repo.filter["owner_entity_id"] != ownerEntityID || repo.filter["channel_owner_user_id"] != channelOwnerUserID {
		t.Fatalf("owner filters missing from filter: %#v", repo.filter)
	}
}

func TestListUsedOwnerSubjectsSkipsUserLookupForEmptyIDs(t *testing.T) {
	repo := &ownerIDsRepoStub{ids: []uint64{}}
	users := &ownerUsersRepoStub{}
	svc := NewSettlementDataService(repo, users, nil, nil)

	items, err := svc.ListUsedOwnerSubjects(context.Background(), SettlementCustomerFilter{})
	if err != nil {
		t.Fatalf("ListUsedOwnerSubjects returned error: %v", err)
	}
	if len(items) != 0 {
		t.Fatalf("expected no owner subjects, got %#v", items)
	}
	if users.requested != nil {
		t.Fatalf("FindByIDs should not be called, got %#v", users.requested)
	}
}
