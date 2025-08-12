package service

import (
	"nfa-dashboard/internal/model"
	"nfa-dashboard/internal/repository"
	"time"
)

type OperationLogService interface {
	List(userID *uint64, method, path, keyword *string, statusCode *int, success *int8, start, end *time.Time, page, pageSize int) ([]model.OperationLog, int64, error)
}

type operationLogService struct{ repo repository.OperationLogRepository }

func NewOperationLogService(repo repository.OperationLogRepository) OperationLogService { return &operationLogService{repo: repo} }

func (s *operationLogService) List(userID *uint64, method, path, keyword *string, statusCode *int, success *int8, start, end *time.Time, page, pageSize int) ([]model.OperationLog, int64, error) {
	return s.repo.List(userID, method, path, keyword, statusCode, success, start, end, page, pageSize)
}
