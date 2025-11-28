package service

import (
	"nfa-dashboard/internal/model"
	"nfa-dashboard/internal/repository"
	"time"
)

type SettlementDataService interface {
	List(filter SettlementCustomerFilter, page, pageSize int) ([]model.SettlementCustomer, int64, error)
	ListAll(filter SettlementCustomerFilter) ([]model.SettlementCustomer, error)
	Recalculate(filter SettlementCustomerFilter) (int64, error)
}

type SettlementCustomerFilter struct {
	Region string
	CP     string
	School string
	Start  *time.Time
	End    *time.Time
}

type settlementDataService struct {
	repo repository.SettlementDataRepository
}

func NewSettlementDataService(repo repository.SettlementDataRepository) SettlementDataService {
	return &settlementDataService{repo: repo}
}

func (s *settlementDataService) List(filter SettlementCustomerFilter, page, pageSize int) ([]model.SettlementCustomer, int64, error) {
	m := map[string]interface{}{}
	if filter.Region != "" {
		m["region"] = filter.Region
	}
	if filter.CP != "" {
		m["cp"] = filter.CP
	}
	if filter.School != "" {
		m["school_name"] = filter.School
	}
	if filter.Start != nil {
		m["start_service_date"] = *filter.Start
	}
	if filter.End != nil {
		m["end_service_date"] = *filter.End
	}
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}
	limit := pageSize
	offset := (page - 1) * pageSize
	return s.repo.ListSettlementCustomer(m, limit, offset)
}

func (s *settlementDataService) ListAll(filter SettlementCustomerFilter) ([]model.SettlementCustomer, error) {
	m := map[string]interface{}{}
	if filter.Region != "" {
		m["region"] = filter.Region
	}
	if filter.CP != "" {
		m["cp"] = filter.CP
	}
	if filter.School != "" {
		m["school_name"] = filter.School
	}
	if filter.Start != nil {
		m["start_service_date"] = *filter.Start
	}
	if filter.End != nil {
		m["end_service_date"] = *filter.End
	}
	rows, _, err := s.repo.ListSettlementCustomer(m, 100000, 0)
	return rows, err
}

// Recalculate 轻量实现：按筛选范围从 nfa_school_settlement 回填/覆盖 settlement_customer 基础字段
func (s *settlementDataService) Recalculate(filter SettlementCustomerFilter) (int64, error) {
	var start, end time.Time
	if filter.Start != nil {
		start = *filter.Start
	}
	if filter.End != nil {
		end = *filter.End
	}
	return s.repo.BackfillFromSchoolSettlement(filter.Region, filter.CP, filter.School, start, end, true)
}
