package service

import (
	"errors"
	"fmt"

	"gorm.io/gorm"
	"nfa-dashboard/internal/model"
	"nfa-dashboard/internal/repository"
)

var ErrInvalidTodoRecipients = errors.New("invalid todo recipients")

type TodoUserAccess interface {
	GetByID(id uint64) (*model.User, error)
	GetUserPermissions(userID uint64) ([]model.Permission, error)
}

type TodoService interface {
	GetSettings() (*model.TodoSettingsResponse, error)
	UpdateSettings(ids []uint64) (*model.TodoSettingsResponse, error)
	ReconcileCustomerRateTodos() error
	ListMyTodos(userID uint64, status string, ignored *bool) ([]model.TodoListItem, error)
	CountMyPendingTodos(userID uint64) (int64, error)
	SetIgnored(todoID, userID uint64, ignored bool) error
	ListMyTodoEvents(todoID, userID uint64) ([]model.TodoEventListItem, error)
	SyncCustomerRate(region, cp, schoolName string) error
}

type todoService struct {
	repo     repository.TodoRepository
	userRepo TodoUserAccess
}

func NewTodoService(repo repository.TodoRepository, userRepo TodoUserAccess) TodoService {
	return &todoService{repo: repo, userRepo: userRepo}
}

func (s *todoService) GetSettings() (*model.TodoSettingsResponse, error) {
	ids, err := s.repo.GetCustomerRateUserIDs()
	if err != nil {
		return nil, err
	}
	return &model.TodoSettingsResponse{CustomerRateUserIDs: ids}, nil
}

func (s *todoService) UpdateSettings(ids []uint64) (*model.TodoSettingsResponse, error) {
	if s.userRepo == nil {
		return nil, errors.New("user repository is required")
	}
	ids = normalizeTodoUserIDs(ids)
	for _, id := range ids {
		user, err := s.userRepo.GetByID(id)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("%w: 用户 %d 不存在或已停用", ErrInvalidTodoRecipients, id)
		}
		if err != nil {
			return nil, err
		}
		if user == nil {
			return nil, fmt.Errorf("%w: 用户 %d 不存在或已停用", ErrInvalidTodoRecipients, id)
		}
		permissions, err := s.userRepo.GetUserPermissions(id)
		if err != nil {
			return nil, err
		}
		hasRead, hasWrite := false, false
		for _, permission := range permissions {
			switch permission.Code {
			case "rates.customer.read":
				hasRead = true
			case "rates.customer.write":
				hasWrite = true
			}
		}
		if !hasRead || !hasWrite {
			return nil, fmt.Errorf("%w: 用户 %d 缺少客户业务费率查看或维护权限", ErrInvalidTodoRecipients, id)
		}
	}
	if err := s.repo.SetCustomerRateUserIDs(ids); err != nil {
		return nil, err
	}
	return &model.TodoSettingsResponse{CustomerRateUserIDs: ids}, nil
}

func (s *todoService) ReconcileCustomerRateTodos() error {
	return s.repo.ReconcileCustomerRateTodos()
}

func (s *todoService) ListMyTodos(userID uint64, status string, ignored *bool) ([]model.TodoListItem, error) {
	return s.repo.ListMyCustomerRateTodos(userID, status, ignored)
}

func (s *todoService) CountMyPendingTodos(userID uint64) (int64, error) {
	return s.repo.CountMyPendingCustomerRateTodos(userID)
}

func (s *todoService) SetIgnored(todoID, userID uint64, ignored bool) error {
	return s.repo.SetCustomerRateTodoIgnored(todoID, userID, ignored)
}

func (s *todoService) ListMyTodoEvents(todoID, userID uint64) ([]model.TodoEventListItem, error) {
	return s.repo.ListMyCustomerRateTodoEvents(todoID, userID)
}

func (s *todoService) SyncCustomerRate(region, cp, schoolName string) error {
	return s.repo.SyncCustomerRateTodo(region, cp, schoolName)
}

func normalizeTodoUserIDs(ids []uint64) []uint64 {
	seen := make(map[uint64]struct{}, len(ids))
	result := make([]uint64, 0, len(ids))
	for _, id := range ids {
		if id == 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		result = append(result, id)
	}
	return result
}
