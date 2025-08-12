package repository

import (
	"nfa-dashboard/internal/model"
	"time"
)

type OperationLogRepository interface {
	List(userID *uint64, method, path, keyword *string, statusCode *int, success *int8, start, end *time.Time, page, pageSize int) ([]model.OperationLog, int64, error)
}

type operationLogRepository struct{}

func NewOperationLogRepository() OperationLogRepository { return &operationLogRepository{} }

func (r *operationLogRepository) List(userID *uint64, method, path, keyword *string, statusCode *int, success *int8, start, end *time.Time, page, pageSize int) ([]model.OperationLog, int64, error) {
	items := make([]model.OperationLog, 0)
	var total int64

	q := model.DB.Model(&model.OperationLog{})
	if userID != nil && *userID > 0 { q = q.Where("user_id = ?", *userID) }
	if method != nil && *method != "" { q = q.Where("method = ?", *method) }
	if path != nil && *path != "" { q = q.Where("path LIKE ?", "%"+*path+"%") }
	if statusCode != nil { q = q.Where("status_code = ?", *statusCode) }
	if success != nil { q = q.Where("success = ?", *success) }
	if start != nil { q = q.Where("created_at >= ?", *start) }
	if end != nil { q = q.Where("created_at <= ?", *end) }
	if keyword != nil && *keyword != "" {
		kw := "%" + *keyword + "%"
		q = q.Where("(error_message LIKE ? OR path LIKE ?)", kw, kw)
	}

	if err := q.Count(&total).Error; err != nil { return nil, 0, err }
	if page > 0 && pageSize > 0 {
		offset := (page - 1) * pageSize
		q = q.Offset(offset).Limit(pageSize)
	}
	if err := q.Order("id DESC").Find(&items).Error; err != nil { return nil, 0, err }
	return items, total, nil
}
