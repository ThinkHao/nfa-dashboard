package service

import (
    "nfa-dashboard/internal/model"
    "nfa-dashboard/internal/repository"
)

// RatesService 费率服务接口
// 封装过滤条件与分页，调用 RatesRepository

type RatesService interface {
    // 客户业务费率
    ListCustomerRates(region, cp, schoolName string, page, pageSize int) ([]model.RateCustomer, int64, error)
    UpsertCustomerRate(rate *model.RateCustomer) error

    // 节点业务费率
    ListNodeRates(region, cp, settlementType string, page, pageSize int) ([]model.RateNode, int64, error)
    UpsertNodeRate(rate *model.RateNode) error

    // 最终客户费率
    ListFinalCustomerRates(region, cp, schoolName, feeType string, page, pageSize int) ([]model.RateFinalCustomer, int64, error)
    UpsertFinalCustomerRate(rate *model.RateFinalCustomer) error

    // 初始化最终客户费率（从 rate_customer 同步，保护 config 记录）
    InitFinalCustomerRatesFromCustomer() (int64, error)

    // 刷新最终客户费率，返回受影响行数
    RefreshFinalCustomerRates() (int64, error)
}

type ratesService struct{ repo repository.RatesRepository }

func NewRatesService(repo repository.RatesRepository) RatesService { return &ratesService{repo: repo} }

func (s *ratesService) ListCustomerRates(region, cp, schoolName string, page, pageSize int) ([]model.RateCustomer, int64, error) {
    filter := map[string]interface{}{}
    if region != "" { filter["region"] = region }
    if cp != "" { filter["cp"] = cp }
    if schoolName != "" { filter["school_name"] = schoolName }
    if page <= 0 { page = 1 }
    if pageSize <= 0 { pageSize = 10 }
    limit := pageSize
    offset := (page - 1) * pageSize
    return s.repo.ListCustomerRates(filter, limit, offset)
}

func (s *ratesService) UpsertCustomerRate(rate *model.RateCustomer) error { return s.repo.UpsertCustomerRate(rate) }

func (s *ratesService) ListNodeRates(region, cp, settlementType string, page, pageSize int) ([]model.RateNode, int64, error) {
    filter := map[string]interface{}{}
    if region != "" { filter["region"] = region }
    if cp != "" { filter["cp"] = cp }
    if settlementType != "" { filter["settlement_type"] = settlementType }
    if page <= 0 { page = 1 }
    if pageSize <= 0 { pageSize = 10 }
    limit := pageSize
    offset := (page - 1) * pageSize
    return s.repo.ListNodeRates(filter, limit, offset)
}

func (s *ratesService) UpsertNodeRate(rate *model.RateNode) error { return s.repo.UpsertNodeRate(rate) }

func (s *ratesService) ListFinalCustomerRates(region, cp, schoolName, feeType string, page, pageSize int) ([]model.RateFinalCustomer, int64, error) {
    filter := map[string]interface{}{}
    if region != "" { filter["region"] = region }
    if cp != "" { filter["cp"] = cp }
    if schoolName != "" { filter["school_name"] = schoolName }
    if feeType != "" { filter["fee_type"] = feeType }
    if page <= 0 { page = 1 }
    if pageSize <= 0 { pageSize = 10 }
    limit := pageSize
    offset := (page - 1) * pageSize
    return s.repo.ListFinalCustomerRates(filter, limit, offset)
}

func (s *ratesService) UpsertFinalCustomerRate(rate *model.RateFinalCustomer) error { return s.repo.UpsertFinalCustomerRate(rate) }

func (s *ratesService) InitFinalCustomerRatesFromCustomer() (int64, error) {
    return s.repo.InitFinalCustomerRatesFromCustomer()
}

func (s *ratesService) RefreshFinalCustomerRates() (int64, error) { return s.repo.RefreshFinalCustomerRates() }
