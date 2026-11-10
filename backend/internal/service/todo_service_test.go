package service

import (
	"errors"
	"testing"

	"gorm.io/gorm"
	"nfa-dashboard/internal/model"
)

func TestTodoServiceUpdateSettingsDeduplicatesEligibleUsers(t *testing.T) {
	repo := &todoRepositoryStub{}
	users := &todoUserAccessStub{
		users:       map[uint64]*model.User{7: {ID: 7}},
		permissions: map[uint64][]model.Permission{7: {{Code: "rates.customer.read"}, {Code: "rates.customer.write"}}},
	}
	svc := NewTodoService(repo, users)

	settings, err := svc.UpdateSettings([]uint64{7, 7, 0})
	if err != nil {
		t.Fatalf("UpdateSettings returned error: %v", err)
	}
	if len(settings.CustomerRateUserIDs) != 1 || settings.CustomerRateUserIDs[0] != 7 {
		t.Fatalf("unexpected normalized recipient IDs: %#v", settings.CustomerRateUserIDs)
	}
	if len(repo.savedIDs) != 1 || repo.savedIDs[0] != 7 {
		t.Fatalf("unexpected saved recipient IDs: %#v", repo.savedIDs)
	}
}

func TestTodoServiceUpdateSettingsRejectsUsersWithoutRateWritePermission(t *testing.T) {
	svc := NewTodoService(&todoRepositoryStub{}, &todoUserAccessStub{
		users:       map[uint64]*model.User{8: {ID: 8}},
		permissions: map[uint64][]model.Permission{8: {{Code: "rates.customer.read"}}},
	})

	if _, err := svc.UpdateSettings([]uint64{8}); !errors.Is(err, ErrInvalidTodoRecipients) {
		t.Fatalf("expected invalid recipient error, got %v", err)
	}
}

func TestTodoServiceReconcileCustomerRateTodos(t *testing.T) {
	repo := &todoRepositoryStub{}
	svc := NewTodoService(repo, nil)

	if err := svc.ReconcileCustomerRateTodos(); err != nil {
		t.Fatalf("ReconcileCustomerRateTodos returned error: %v", err)
	}
	if repo.reconcileCalls != 1 {
		t.Fatalf("expected one repository reconciliation, got %d", repo.reconcileCalls)
	}
}

type todoRepositoryStub struct {
	savedIDs       []uint64
	reconcileCalls int
}

func (s *todoRepositoryStub) GetCustomerRateUserIDs() ([]uint64, error) { return s.savedIDs, nil }
func (s *todoRepositoryStub) SetCustomerRateUserIDs(ids []uint64) error {
	s.savedIDs = append([]uint64(nil), ids...)
	return nil
}
func (s *todoRepositoryStub) ReconcileCustomerRateTodos() error {
	s.reconcileCalls++
	return nil
}
func (s *todoRepositoryStub) ListMyCustomerRateTodos(uint64, string, *bool) ([]model.TodoListItem, error) {
	return nil, nil
}
func (s *todoRepositoryStub) CountMyPendingCustomerRateTodos(uint64) (int64, error) { return 0, nil }
func (s *todoRepositoryStub) SetCustomerRateTodoIgnored(uint64, uint64, bool) error { return nil }
func (s *todoRepositoryStub) ListMyCustomerRateTodoEvents(uint64, uint64) ([]model.TodoEventListItem, error) {
	return nil, nil
}
func (s *todoRepositoryStub) SyncCustomerRateTodo(string, string, string) error { return nil }

type todoUserAccessStub struct {
	users       map[uint64]*model.User
	permissions map[uint64][]model.Permission
}

func (s *todoUserAccessStub) GetByID(id uint64) (*model.User, error) {
	user, ok := s.users[id]
	if !ok {
		return nil, gorm.ErrRecordNotFound
	}
	return user, nil
}
func (s *todoUserAccessStub) GetUserPermissions(id uint64) ([]model.Permission, error) {
	return s.permissions[id], nil
}
