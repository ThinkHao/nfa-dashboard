package service

import (
	"encoding/json"
	"strings"
	"time"

	"nfa-dashboard/internal/model"
	"nfa-dashboard/internal/repository"
)

// RatesService 费率服务接口
// 封装过滤条件与分页，调用 RatesRepository

type RatesService interface {
	// 客户业务费率
	ListCustomerRates(region, cp, schoolName string, settlementReady *bool, page, pageSize int) ([]model.RateCustomer, int64, error)
	UpsertCustomerRate(rate *model.RateCustomer) error
	ValidateCustomerRate(rate *model.RateCustomer) error
	LookupCustomerRateOwnerIDsByDisplayName(names CustomerRateOwnerNames) (CustomerRateOwnerIDs, []MissingCustomerRateOwner, error)
	ResolveCustomerRateOwnerIDsByDisplayName(rate *model.RateCustomer, names CustomerRateOwnerNames) error
	PreviewCustomerRateImportUsers(aliases []string) ([]MissingImportUser, error)
	CreateCustomerRateImportUsers(missing []MissingImportUser) ([]CreatedImportUser, error)

	// 节点业务费率
	ListNodeRates(region, cp, settlementType string, page, pageSize int) ([]model.RateNode, int64, error)
	UpsertNodeRate(rate *model.RateNode) error

	// 最终客户费率
	ListFinalCustomerRates(region, cp, schoolName, feeType string, page, pageSize int) ([]model.RateFinalCustomer, int64, error)
	UpsertFinalCustomerRate(rate *model.RateFinalCustomer) error

	// 按服务日期返回折损后的最终客户费率视图（基于 rate_customer + 折损规则动态计算）
	ListFinalCustomerRatesDiscounted(region, cp, schoolName, feeType string, serviceDate time.Time, page, pageSize int) ([]DiscountedFinalCustomerRate, int64, error)

	// 初始化最终客户费率（从 rate_customer 同步，保护 config 记录）
	InitFinalCustomerRatesFromCustomer() (int64, error)

	// 刷新最终客户费率，返回受影响行数
	RefreshFinalCustomerRates() (int64, error)

	// 清理无效的最终客户费率（仅 auto；任一关键费率字段为空）
	CleanupInvalidFinalCustomerRates() (int64, error)
}

type ratesService struct {
	repo         repository.RatesRepository
	discountRepo repository.RateDiscountRepository
	userRepo     repository.UserRepository
}

type CustomerRateOwnerNames struct {
	CustomerFeeOwnerName    string
	NetworkLineFeeOwnerName string
	GeneralFeeOwnerName     string
	ChannelOwnerName        string
}

func NewRatesService(repo repository.RatesRepository, discountRepo repository.RateDiscountRepository, userRepo repository.UserRepository) RatesService {
	return &ratesService{repo: repo, discountRepo: discountRepo, userRepo: userRepo}
}

func (s *ratesService) ListCustomerRates(region, cp, schoolName string, settlementReady *bool, page, pageSize int) ([]model.RateCustomer, int64, error) {
	filter := map[string]interface{}{}
	if region != "" {
		filter["region"] = region
	}
	if cp != "" {
		filter["cp"] = cp
	}
	if schoolName != "" {
		filter["school_name"] = schoolName
	}
	if settlementReady != nil {
		filter["settlement_ready"] = *settlementReady
	}
	filter["exclude_filtered"] = true
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}
	limit := pageSize
	offset := (page - 1) * pageSize
	return s.repo.ListCustomerRates(filter, limit, offset)
}

func (s *ratesService) UpsertCustomerRate(rate *model.RateCustomer) error {
	if err := s.ValidateCustomerRate(rate); err != nil {
		return err
	}
	return s.repo.UpsertCustomerRate(rate)
}

func (s *ratesService) ValidateCustomerRate(rate *model.RateCustomer) error {
	if err := normalizeIncrementConfig(rate); err != nil {
		return err
	}
	return s.validateCustomerRateOwnerUsers(rate)
}

func (s *ratesService) validateCustomerRateOwnerUsers(rate *model.RateCustomer) error {
	if rate == nil {
		return NewBadRequest("rate is required")
	}
	if s.userRepo == nil {
		return NewBadRequest("user repository is required")
	}

	type ownerRef struct {
		field string
		id    uint64
	}
	owners := make([]ownerRef, 0, 4)
	push := func(field string, idPtr *uint64) {
		if idPtr == nil || *idPtr == 0 {
			return
		}
		owners = append(owners, ownerRef{field: field, id: *idPtr})
	}
	push("customer_fee_owner_id", rate.CustomerFeeOwnerID)
	push("network_line_fee_owner_id", rate.NetworkLineFeeOwnerID)
	push("general_fee_owner_id", rate.GeneralFeeOwnerID)
	push("channel_owner_user_id", rate.ChannelOwnerUserID)
	if len(owners) == 0 {
		return nil
	}

	uniq := make([]uint64, 0, len(owners))
	seen := make(map[uint64]struct{}, len(owners))
	for _, o := range owners {
		if _, ok := seen[o.id]; ok {
			continue
		}
		seen[o.id] = struct{}{}
		uniq = append(uniq, o.id)
	}

	users, err := s.userRepo.FindByIDs(uniq)
	if err != nil {
		return err
	}
	userSet := make(map[uint64]struct{}, len(users))
	for _, u := range users {
		userSet[u.ID] = struct{}{}
	}

	for _, o := range owners {
		if _, ok := userSet[o.id]; ok {
			continue
		}
		return NewBadRequest(o.field + " must be a valid system user id")
	}
	return nil
}

func normalizeIncrementConfig(rate *model.RateCustomer) error {
	if rate == nil {
		return NewBadRequest("rate is required")
	}
	defaultStock := 1.0
	defaultIncrement := 0.0
	if rate.IncrementStartAt == nil {
		rate.StockRatio = &defaultStock
		rate.IncrementRatio = &defaultIncrement
		return nil
	}

	if rate.StockRatio == nil && rate.IncrementRatio == nil {
		return NewBadRequest("设置增量起算日期后，存量占比和增量占比不能同时为空")
	}
	if rate.StockRatio == nil && rate.IncrementRatio != nil {
		v := 0.0
		rate.StockRatio = &v
	}
	if rate.IncrementRatio == nil && rate.StockRatio != nil {
		v := 0.0
		rate.IncrementRatio = &v
	}

	stock := *rate.StockRatio
	increment := *rate.IncrementRatio
	if stock < 0 || stock > 1 || increment < 0 || increment > 1 {
		return NewBadRequest("存量占比和增量占比必须在 0% 到 100% 之间")
	}
	return nil
}

func (s *ratesService) ListNodeRates(region, cp, settlementType string, page, pageSize int) ([]model.RateNode, int64, error) {
	filter := map[string]interface{}{}
	if region != "" {
		filter["region"] = region
	}
	if cp != "" {
		filter["cp"] = cp
	}
	if settlementType != "" {
		filter["settlement_type"] = settlementType
	}
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}
	limit := pageSize
	offset := (page - 1) * pageSize
	return s.repo.ListNodeRates(filter, limit, offset)
}

func (s *ratesService) UpsertNodeRate(rate *model.RateNode) error { return s.repo.UpsertNodeRate(rate) }

func (s *ratesService) ListFinalCustomerRates(region, cp, schoolName, feeType string, page, pageSize int) ([]model.RateFinalCustomer, int64, error) {
	filter := map[string]interface{}{}
	if region != "" {
		filter["region"] = region
	}
	if cp != "" {
		filter["cp"] = cp
	}
	if schoolName != "" {
		filter["school_name"] = schoolName
	}
	if feeType != "" {
		filter["fee_type"] = feeType
	}
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}
	limit := pageSize
	offset := (page - 1) * pageSize
	return s.repo.ListFinalCustomerRates(filter, limit, offset)
}

func (s *ratesService) UpsertFinalCustomerRate(rate *model.RateFinalCustomer) error {
	return s.repo.UpsertFinalCustomerRate(rate)
}

func (s *ratesService) InitFinalCustomerRatesFromCustomer() (int64, error) {
	return s.repo.InitFinalCustomerRatesFromCustomer()
}

func (s *ratesService) RefreshFinalCustomerRates() (int64, error) {
	return s.repo.RefreshFinalCustomerRates()
}

func (s *ratesService) CleanupInvalidFinalCustomerRates() (int64, error) {
	return s.repo.CleanupInvalidFinalCustomerRates()
}

// DiscountedFinalCustomerRate 表示指定服务日期下、折损后的最终客户费率视图
type DiscountedFinalCustomerRate struct {
	Region                 string   `json:"region"`
	CP                     string   `json:"cp"`
	SchoolName             *string  `json:"school_name,omitempty"`
	ServiceDate            string   `json:"service_date"`
	CustomerFeeBase        *float64 `json:"customer_fee_base,omitempty"`
	CustomerFeeDiscount    *float64 `json:"customer_fee_discount,omitempty"`
	ChannelRateBase        *float64 `json:"channel_rate_base,omitempty"`
	ChannelRateDiscount    *float64 `json:"channel_rate_discount,omitempty"`
	NetworkLineFeeBase     *float64 `json:"network_line_fee_base,omitempty"`
	NetworkLineFeeDiscount *float64 `json:"network_line_fee_discount,omitempty"`
	GeneralFeeBase         *float64 `json:"general_fee_base,omitempty"`
	GeneralFeeDiscount     *float64 `json:"general_fee_discount,omitempty"`
	CustomerFeeOwnerID     *uint64  `json:"customer_fee_owner_id,omitempty"`
	ChannelOwnerUserID     *uint64  `json:"channel_owner_user_id,omitempty"`
	DiscountRuleID         *uint64  `json:"discount_rule_id,omitempty"`
	DiscountRuleName       *string  `json:"discount_rule_name,omitempty"`
	ServiceYearIndex       *int     `json:"service_year_index,omitempty"`
}

// ListFinalCustomerRatesDiscounted 按服务日期返回折损后的最终客户费率
// 当前实现：基于 rate_customer 及折损规则动态计算，不直接依赖 rate_final_customer 表
func (s *ratesService) ListFinalCustomerRatesDiscounted(region, cp, schoolName, feeType string, serviceDate time.Time, page, pageSize int) ([]DiscountedFinalCustomerRate, int64, error) {
	// 复用已有的 ListCustomerRates 过滤与分页逻辑
	filter := map[string]interface{}{}
	if region != "" {
		filter["region"] = region
	}
	if cp != "" {
		filter["cp"] = cp
	}
	if schoolName != "" {
		filter["school_name"] = schoolName
	}
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}
	limit := pageSize
	offset := (page - 1) * pageSize

	filter["exclude_filtered"] = true
	customers, total, err := s.repo.ListCustomerRates(filter, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	if len(customers) == 0 {
		return []DiscountedFinalCustomerRate{}, total, nil
	}

	items := make([]DiscountedFinalCustomerRate, 0, len(customers))
	for _, cst := range customers {
		discounted, rule, yearIdx, err := s.applyDiscountForCustomer(&cst, serviceDate)
		if err != nil {
			// 为了调用方便，这里选择忽略单条错误并继续，其它记录仍可返回
			continue
		}
		var cfBase, chBase, nlBase, gnBase *float64
		if cst.CustomerFee != nil {
			cfBase = new(float64)
			*cfBase = *cst.CustomerFee
		}
		if cst.ChannelRate != nil {
			chBase = new(float64)
			*chBase = *cst.ChannelRate
		}
		if cst.NetworkLineFee != nil {
			nlBase = new(float64)
			*nlBase = *cst.NetworkLineFee
		}
		if cst.GeneralFee != nil {
			gnBase = new(float64)
			*gnBase = *cst.GeneralFee
		}
		var cfDiscount, chDiscount *float64
		if v, ok := discounted["customer_fee"]; ok {
			cfDiscount = v
		} else {
			cfDiscount = cfBase
		}
		if v, ok := discounted["channel_rate"]; ok {
			chDiscount = v
		} else {
			chDiscount = chBase
		}
		var nlDiscount, gnDiscount *float64
		if v, ok := discounted["network_line_fee"]; ok {
			nlDiscount = v
		} else {
			nlDiscount = nlBase
		}
		if v, ok := discounted["general_fee"]; ok {
			gnDiscount = v
		} else {
			gnDiscount = gnBase
		}
		var ruleID *uint64
		var ruleName *string
		if rule != nil {
			if rule.ID > 0 {
				id := rule.ID
				ruleID = &id
			}
			if rule.Name != "" {
				nameCopy := rule.Name
				ruleName = &nameCopy
			}
		}
		var yearIdxPtr *int
		if yearIdx > 0 {
			yearIdxPtr = &yearIdx
		}
		items = append(items, DiscountedFinalCustomerRate{
			Region:                 cst.Region,
			CP:                     cst.CP,
			SchoolName:             cst.SchoolName,
			ServiceDate:            serviceDate.Format("2006-01-02"),
			CustomerFeeBase:        cfBase,
			CustomerFeeDiscount:    cfDiscount,
			ChannelRateBase:        chBase,
			ChannelRateDiscount:    chDiscount,
			NetworkLineFeeBase:     nlBase,
			NetworkLineFeeDiscount: nlDiscount,
			GeneralFeeBase:         gnBase,
			GeneralFeeDiscount:     gnDiscount,
			CustomerFeeOwnerID:     cst.CustomerFeeOwnerID,
			ChannelOwnerUserID:     cst.ChannelOwnerUserID,
			DiscountRuleID:         ruleID,
			DiscountRuleName:       ruleName,
			ServiceYearIndex:       yearIdxPtr,
		})
	}
	return items, total, nil
}

// applyDiscountForCustomer 根据客户费率与服务日期应用折损规则
// 返回：折损后的字段值、命中的规则以及服务年限索引（从1起）
func (s *ratesService) applyDiscountForCustomer(c *model.RateCustomer, serviceDate time.Time) (map[string]*float64, *model.RateDiscountRule, int, error) {
	if c == nil {
		return nil, nil, 0, nil
	}
	if c.StartAt == nil {
		// 未设置起始时间则不折损
		return nil, nil, 0, nil
	}
	start := time.Date(c.StartAt.Year(), c.StartAt.Month(), c.StartAt.Day(), 0, 0, 0, 0, c.StartAt.Location())
	if serviceDate.IsZero() {
		serviceDate = time.Now()
	}
	cur := time.Date(serviceDate.Year(), serviceDate.Month(), serviceDate.Day(), 0, 0, 0, 0, serviceDate.Location())
	// 计算服务年限（自然年区间）
	yearIdx := 1
	if cur.Before(start) {
		yearIdx = 1
	} else {
		for tmp := start; !cur.Before(tmp.AddDate(1, 0, 0)); tmp = tmp.AddDate(1, 0, 0) {
			yearIdx++
		}
	}

	// 按 scope 匹配规则：school > cp > region > global
	scopes := []struct {
		typeVal string
		key     *string
	}{
		{"school", c.SchoolName},
		{"cp", &c.CP},
		{"region", &c.Region},
		{"global", nil},
	}
	var matchedRule *model.RateDiscountRule
	for _, sc := range scopes {
		filter := map[string]interface{}{"scope_type": sc.typeVal, "enabled": true}
		rules, _, err := s.discountRepo.ListRules(filter, 0, 0)
		if err != nil {
			return nil, nil, 0, err
		}
		if len(rules) == 0 {
			continue
		}
		for i := range rules {
			r := &rules[i]
			// global 无需校验 scope_key
			if sc.typeVal == "global" {
				matchedRule = r
				break
			}
			if sc.key == nil || *sc.key == "" {
				continue
			}
			if r.ScopeKey == nil {
				continue
			}
			if strings.TrimSpace(*r.ScopeKey) == strings.TrimSpace(*sc.key) {
				matchedRule = r
				break
			}
		}
		if matchedRule != nil {
			break
		}
	}
	if matchedRule == nil {
		return nil, nil, yearIdx, nil
	}

	items, err := s.discountRepo.ListItemsByRuleID(matchedRule.ID)
	if err != nil {
		return nil, nil, 0, err
	}
	if len(items) == 0 {
		return nil, matchedRule, yearIdx, nil
	}
	var usedItem *model.RateDiscountRuleItem
	for i := range items {
		it := &items[i]
		if yearIdx < it.FromYear {
			continue
		}
		if it.ToYear != nil && yearIdx > *it.ToYear {
			continue
		}
		usedItem = it
		break
	}
	if usedItem == nil {
		return nil, matchedRule, yearIdx, nil
	}

	// 解析 fields
	var fieldKeys []string
	if len(matchedRule.Fields) > 0 {
		if err := json.Unmarshal(matchedRule.Fields, &fieldKeys); err != nil {
			// 若配置非法，视为无折损
			return nil, matchedRule, yearIdx, nil
		}
	}
	if len(fieldKeys) == 0 {
		// 未配置字段则默认仅对 customer_fee 生效
		fieldKeys = []string{"customer_fee"}
	}

	res := map[string]*float64{}
	ratio := usedItem.DiscountRate
	for _, key := range fieldKeys {
		switch key {
		case "customer_fee":
			if c.CustomerFee != nil {
				v := (*c.CustomerFee) * ratio
				res["customer_fee"] = &v
			}
		case "network_line_fee":
			if c.NetworkLineFee != nil {
				v := (*c.NetworkLineFee) * ratio
				res["network_line_fee"] = &v
			}
		case "general_fee":
			if c.GeneralFee != nil {
				v := (*c.GeneralFee) * ratio
				res["general_fee"] = &v
			}
		case "channel_rate":
			if c.ChannelRate != nil {
				v := (*c.ChannelRate) * ratio
				res["channel_rate"] = &v
			}
		}
	}
	return res, matchedRule, yearIdx, nil
}
