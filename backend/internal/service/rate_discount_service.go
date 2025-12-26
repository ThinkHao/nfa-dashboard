package service

import (
	"encoding/json"
	"strings"

	"nfa-dashboard/internal/model"
	"nfa-dashboard/internal/repository"

	"gorm.io/datatypes"
)

// RateDiscountService 定义折损规则的业务接口

type RateDiscountService interface {
	List(name, scopeType string, enabled *bool, page, pageSize int) ([]model.RateDiscountRule, int64, error)
	Get(id uint64) (*model.RateDiscountRule, []model.RateDiscountRuleItem, error)
	Create(rule *model.RateDiscountRule, items []model.RateDiscountRuleItem) (*model.RateDiscountRule, []model.RateDiscountRuleItem, error)
	Update(id uint64, updates map[string]interface{}) error
	Delete(id uint64) error
	ReplaceItems(id uint64, items []model.RateDiscountRuleItem) error
}

type rateDiscountService struct {
	repo repository.RateDiscountRepository
}

func NewRateDiscountService(repo repository.RateDiscountRepository) RateDiscountService {
	return &rateDiscountService{repo: repo}
}

func (s *rateDiscountService) List(name, scopeType string, enabled *bool, page, pageSize int) ([]model.RateDiscountRule, int64, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}
	filter := map[string]interface{}{}
	if name != "" {
		filter["name"] = name
	}
	if scopeType != "" {
		filter["scope_type"] = scopeType
	}
	if enabled != nil {
		filter["enabled"] = *enabled
	}
	limit := pageSize
	offset := (page - 1) * pageSize
	return s.repo.ListRules(filter, limit, offset)
}

func (s *rateDiscountService) Get(id uint64) (*model.RateDiscountRule, []model.RateDiscountRuleItem, error) {
	if id == 0 {
		return nil, nil, NewBadRequest("invalid id")
	}
	rule, err := s.repo.GetRuleByID(id)
	if err != nil {
		return nil, nil, err
	}
	if rule == nil {
		return nil, nil, NewBadRequest("rule not found")
	}
	items, err := s.repo.ListItemsByRuleID(id)
	if err != nil {
		return nil, nil, err
	}
	return rule, items, nil
}

func (s *rateDiscountService) Create(rule *model.RateDiscountRule, items []model.RateDiscountRuleItem) (*model.RateDiscountRule, []model.RateDiscountRuleItem, error) {
	if rule == nil {
		return nil, nil, NewBadRequest("nil rule")
	}
	// 归一化
	rule.Name = strings.TrimSpace(rule.Name)
	rule.ScopeType = strings.TrimSpace(rule.ScopeType)
	if rule.ScopeType == "" {
		rule.ScopeType = "global"
	}
	if rule.Name == "" {
		return nil, nil, NewBadRequest("name is required")
	}
	if rule.Priority < 0 {
		rule.Priority = 100
	}
	// JSON 字段校验：fields 应为字符串数组 JSON
	if err := validateFieldsJSON(rule.Fields); err != nil {
		return nil, nil, err
	}

	created, err := s.repo.CreateRule(rule)
	if err != nil {
		return nil, nil, err
	}
	if err := s.repo.ReplaceItems(created.ID, items); err != nil {
		return nil, nil, err
	}
	newItems, err := s.repo.ListItemsByRuleID(created.ID)
	if err != nil {
		return created, nil, err
	}
	return created, newItems, nil
}

func (s *rateDiscountService) Update(id uint64, updates map[string]interface{}) error {
	if id == 0 {
		return NewBadRequest("invalid id")
	}
	if len(updates) == 0 {
		return NewBadRequest("no fields to update")
	}
	if v, ok := updates["name"]; ok {
		if v == nil {
			return NewBadRequest("name cannot be null")
		}
		name := strings.TrimSpace(v.(string))
		if name == "" {
			return NewBadRequest("name cannot be empty")
		}
		updates["name"] = name
	}
	if v, ok := updates["scope_type"]; ok {
		if v == nil {
			return NewBadRequest("scope_type cannot be null")
		}
		st := strings.TrimSpace(v.(string))
		if st == "" {
			st = "global"
		}
		updates["scope_type"] = st
	}
	if v, ok := updates["fields"]; ok {
		if v == nil {
			return NewBadRequest("fields cannot be null")
		}
		if err := validateFieldsJSONInterface(v); err != nil {
			return err
		}
		// 将数组等通用类型统一转为 JSON，避免直接用数组导致 SQL 语法错误 (Operand should contain 1 column(s))
		if bs, ok2 := toJSONBytes(v); ok2 {
			updates["fields"] = datatypes.JSON(bs)
		}
	}
	return s.repo.UpdateRule(id, updates)
}

func (s *rateDiscountService) Delete(id uint64) error {
	if id == 0 {
		return NewBadRequest("invalid id")
	}
	return s.repo.DeleteRule(id)
}

func (s *rateDiscountService) ReplaceItems(id uint64, items []model.RateDiscountRuleItem) error {
	if id == 0 {
		return NewBadRequest("invalid id")
	}
	// 简单校验：from_year > 0, discount_rate > 0
	for _, it := range items {
		if it.FromYear <= 0 {
			return NewBadRequest("from_year must be > 0")
		}
		if it.DiscountRate <= 0 {
			return NewBadRequest("discount_rate must be > 0")
		}
		if it.ToYear != nil && *it.ToYear < it.FromYear {
			return NewBadRequest("to_year must be >= from_year")
		}
	}
	return s.repo.ReplaceItems(id, items)
}

// ------- 校验辅助 -------

func validateFieldsJSON(data datatypes.JSON) error {
	if len(data) == 0 {
		return nil
	}
	var arr []string
	if err := json.Unmarshal(data, &arr); err != nil {
		return NewBadRequest("fields must be JSON array of strings")
	}
	return nil
}

func validateFieldsJSONInterface(v interface{}) error {
	bs, ok := toJSONBytes(v)
	if !ok {
		return NewBadRequest("fields must be valid JSON")
	}
	var arr []string
	if err := json.Unmarshal(bs, &arr); err != nil {
		return NewBadRequest("fields must be JSON array of strings")
	}
	return nil
}
