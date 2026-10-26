package service

import (
	"encoding/json"
	"fmt"
	"sort"
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
	PreviewBatch(drafts []model.RateDiscountRuleDraft) (*RateDiscountBatchPreview, error)
	CreateBatch(drafts []model.RateDiscountRuleDraft) ([]model.RateDiscountRule, error)
}

type RateDiscountBatchPreview struct {
	Groups   []model.RateDiscountRuleBatchPreview `json:"groups"`
	CanApply bool                                 `json:"can_apply"`
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
	for _, item := range items {
		if item.FromYear <= 0 || item.DiscountRate <= 0 {
			return nil, nil, NewBadRequest("from_year and discount_rate must be > 0")
		}
		if item.ToYear != nil && *item.ToYear < item.FromYear {
			return nil, nil, NewBadRequest("to_year must be >= from_year")
		}
	}

	drafts := []model.RateDiscountRuleDraft{{
		Rule:   *rule,
		Items:  items,
		Scopes: legacyDiscountRuleScopes(rule.ScopeType, rule.ScopeKey),
	}}
	createdRules, err := s.repo.CreateRulesBatch(drafts)
	if err != nil {
		return nil, nil, err
	}
	created := &createdRules[0]
	created.Scopes = drafts[0].Scopes
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
	current, err := s.repo.GetRuleByID(id)
	if err != nil {
		return err
	}
	if current == nil {
		return NewBadRequest("rule not found")
	}
	scopeChanged := false
	scopeType := current.ScopeType
	scopeKey := current.ScopeKey
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
		scopeType = st
		scopeChanged = true
	}
	if v, ok := updates["scope_key"]; ok {
		if v == nil {
			scopeKey = nil
		} else {
			key, ok := v.(string)
			if !ok {
				return NewBadRequest("scope_key must be a string")
			}
			key = strings.TrimSpace(key)
			if key == "" {
				scopeKey = nil
				updates["scope_key"] = nil
			} else {
				scopeKey = &key
				updates["scope_key"] = key
			}
		}
		scopeChanged = true
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
	if scopeChanged {
		if len(current.Scopes) > 1 || (len(current.Scopes) == 1 && current.Scopes[0].ScopeType == "school" && current.Scopes[0].Region != nil && current.Scopes[0].CP != nil) {
			return NewBadRequest("该规则使用批量适用范围，请通过批量设置入口调整")
		}
		return s.repo.UpdateRuleWithScopes(id, updates, legacyDiscountRuleScopes(scopeType, scopeKey))
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

func (s *rateDiscountService) PreviewBatch(drafts []model.RateDiscountRuleDraft) (*RateDiscountBatchPreview, error) {
	normalized, err := normalizeDiscountRuleDrafts(drafts)
	if err != nil {
		return nil, err
	}
	regions := make([]string, 0)
	for _, draft := range normalized {
		for _, scope := range draft.Scopes {
			if scope.ScopeType == "region" && scope.ScopeKey != nil {
				regions = append(regions, *scope.ScopeKey)
			}
			if scope.ScopeType == "school" && scope.Region != nil {
				regions = append(regions, *scope.Region)
			}
		}
	}
	candidates, err := s.repo.ListSchoolTargets(regions)
	if err != nil {
		return nil, err
	}
	candidateSet := make(map[string]bool, len(candidates))
	for _, target := range candidates {
		candidateSet[discountSchoolKey(target)] = true
	}
	// Check enabled and disabled rules so a later re-enable cannot introduce a same-level overlap.
	existingRules, _, err := s.repo.ListRules(map[string]interface{}{}, 0, 0)
	if err != nil {
		return nil, err
	}
	preview := &RateDiscountBatchPreview{Groups: make([]model.RateDiscountRuleBatchPreview, 0, len(normalized)), CanApply: true}
	for i, draft := range normalized {
		group := model.RateDiscountRuleBatchPreview{
			Index:       i,
			Name:        draft.Rule.Name,
			ScopeType:   draft.Rule.ScopeType,
			TargetCount: len(draft.Scopes),
		}
		conflictDetails := map[string]model.RateDiscountRuleConflict{}
		for _, scope := range draft.Scopes {
			if scope.ScopeType == "region" && scope.ScopeKey != nil {
				for _, candidate := range candidates {
					if candidate.Region != nil && *candidate.Region == *scope.ScopeKey {
						group.MatchedSchoolCount++
					}
				}
				continue
			}
			if candidateSet[discountSchoolKey(scope)] {
				group.MatchedSchoolCount++
			} else {
				group.UnmatchedScopes = append(group.UnmatchedScopes, scope)
			}
		}
		for _, old := range existingRules {
			for _, oldScope := range old.Scopes {
				for _, newScope := range draft.Scopes {
					if discountScopesConflict(oldScope, newScope) {
						key := fmt.Sprintf("%d|%s", old.ID, discountScopeIdentity(oldScope))
						conflictDetails[key] = model.RateDiscountRuleConflict{
							RuleID:        old.ID,
							RuleName:      old.Name,
							Enabled:       old.Enabled,
							ExistingScope: oldScope,
							TargetScope:   newScope,
						}
					}
				}
			}
		}
		for _, conflict := range conflictDetails {
			group.Conflicts = append(group.Conflicts, conflict)
		}
		sort.Slice(group.Conflicts, func(i, j int) bool {
			if group.Conflicts[i].RuleID != group.Conflicts[j].RuleID {
				return group.Conflicts[i].RuleID < group.Conflicts[j].RuleID
			}
			return discountScopeIdentity(group.Conflicts[i].ExistingScope) < discountScopeIdentity(group.Conflicts[j].ExistingScope)
		})
		if len(group.Conflicts) > 0 {
			preview.CanApply = false
		}
		preview.Groups = append(preview.Groups, group)
	}
	return preview, nil
}

func (s *rateDiscountService) CreateBatch(drafts []model.RateDiscountRuleDraft) ([]model.RateDiscountRule, error) {
	normalized, err := normalizeDiscountRuleDrafts(drafts)
	if err != nil {
		return nil, err
	}
	preview, err := s.PreviewBatch(normalized)
	if err != nil {
		return nil, err
	}
	if !preview.CanApply {
		return nil, NewBadRequest("存在与已启用规则重叠的适用范围，请先处理冲突规则")
	}
	return s.repo.CreateRulesBatch(normalized)
}

func normalizeDiscountRuleDrafts(drafts []model.RateDiscountRuleDraft) ([]model.RateDiscountRuleDraft, error) {
	if len(drafts) == 0 {
		return nil, NewBadRequest("至少需要一组规则")
	}
	seenScopes := map[string]bool{}
	for i := range drafts {
		draft := &drafts[i]
		draft.Rule.Name = strings.TrimSpace(draft.Rule.Name)
		draft.Rule.ScopeType = strings.TrimSpace(draft.Rule.ScopeType)
		if draft.Rule.Name == "" {
			return nil, NewBadRequest(fmt.Sprintf("第 %d 组规则名称必填", i+1))
		}
		if draft.Rule.ScopeType != "region" && draft.Rule.ScopeType != "school" {
			return nil, NewBadRequest(fmt.Sprintf("第 %d 组规则仅支持省份或院校范围", i+1))
		}
		if len(draft.Scopes) == 0 {
			return nil, NewBadRequest(fmt.Sprintf("第 %d 组规则未选择适用范围", i+1))
		}
		if draft.Rule.Priority < 0 {
			draft.Rule.Priority = 100
		}
		if err := validateFieldsJSON(draft.Rule.Fields); err != nil {
			return nil, err
		}
		if err := validateDiscountItems(draft.Items); err != nil {
			return nil, err
		}
		localScopes := map[string]bool{}
		for j := range draft.Scopes {
			scope := &draft.Scopes[j]
			scope.ScopeType = draft.Rule.ScopeType
			scope.RuleID = 0
			scope.ID = 0
			key := ""
			if scope.ScopeType == "region" {
				region := ""
				if scope.ScopeKey != nil {
					region = strings.TrimSpace(*scope.ScopeKey)
				}
				if region == "" && scope.Region != nil {
					region = strings.TrimSpace(*scope.Region)
				}
				if region == "" {
					return nil, NewBadRequest(fmt.Sprintf("第 %d 组规则包含空省份", i+1))
				}
				scope.ScopeKey = &region
				scope.Region = &region
				scope.CP = nil
				scope.SchoolName = nil
				key = "region|" + region
			} else {
				region, cp, name := scopeString(scope.Region), scopeString(scope.CP), scopeString(scope.SchoolName)
				if region == "" || cp == "" || name == "" {
					return nil, NewBadRequest(fmt.Sprintf("第 %d 组规则的院校范围必须包含省份、CP 和院校名称", i+1))
				}
				scope.Region, scope.CP, scope.SchoolName = &region, &cp, &name
				scope.ScopeKey = nil
				key = "school|" + region + "|" + cp + "|" + name
			}
			if localScopes[key] || seenScopes[key] {
				return nil, NewBadRequest("同一适用范围不能分配给多个规则组")
			}
			localScopes[key] = true
			seenScopes[key] = true
		}
		if len(draft.Scopes) == 1 && draft.Rule.ScopeType == "region" {
			draft.Rule.ScopeKey = draft.Scopes[0].ScopeKey
		} else {
			draft.Rule.ScopeKey = nil
		}
		draft.Rule.Scopes = nil
	}
	return drafts, nil
}

func validateDiscountItems(items []model.RateDiscountRuleItem) error {
	ordered := append([]model.RateDiscountRuleItem(nil), items...)
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].FromYear < ordered[j].FromYear })
	var previousEnd *int
	for i, item := range ordered {
		if item.FromYear <= 0 || item.DiscountRate <= 0 || item.DiscountRate > 1 {
			return NewBadRequest("折损条目要求 from_year > 0 且 discount_rate 在 0~1 之间")
		}
		if item.ToYear != nil && *item.ToYear < item.FromYear {
			return NewBadRequest("to_year must be >= from_year")
		}
		if previousEnd == nil && i > 0 || previousEnd != nil && item.FromYear <= *previousEnd {
			return NewBadRequest("折损条目年份区间不能重叠")
		}
		previousEnd = item.ToYear
	}
	return nil
}

func legacyDiscountRuleScopes(scopeType string, scopeKey *string) []model.RateDiscountRuleScope {
	scope := model.RateDiscountRuleScope{ScopeType: scopeType, ScopeKey: scopeKey}
	switch scopeType {
	case "region":
		scope.Region = scopeKey
	case "cp":
		scope.CP = scopeKey
	case "school":
		scope.SchoolName = scopeKey
	}
	return []model.RateDiscountRuleScope{scope}
}

func scopeString(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}

func discountSchoolKey(scope model.RateDiscountRuleScope) string {
	return fmt.Sprintf("%s|%s|%s", scopeString(scope.Region), scopeString(scope.CP), scopeString(scope.SchoolName))
}

func discountScopeIdentity(scope model.RateDiscountRuleScope) string {
	if scope.ScopeType == "region" {
		return "region|" + scopeString(scope.ScopeKey)
	}
	return scope.ScopeType + "|" + discountSchoolKey(scope)
}

func discountScopesConflict(existing, incoming model.RateDiscountRuleScope) bool {
	if existing.ScopeType != incoming.ScopeType {
		return false
	}
	if incoming.ScopeType == "region" {
		return existing.ScopeKey != nil && incoming.ScopeKey != nil && strings.EqualFold(strings.TrimSpace(*existing.ScopeKey), strings.TrimSpace(*incoming.ScopeKey))
	}
	if incoming.ScopeType == "school" {
		if !strings.EqualFold(scopeString(existing.SchoolName), scopeString(incoming.SchoolName)) {
			return false
		}
		legacyWildcard := existing.Region == nil && existing.CP == nil
		return legacyWildcard || (strings.EqualFold(scopeString(existing.Region), scopeString(incoming.Region)) && strings.EqualFold(scopeString(existing.CP), scopeString(incoming.CP)))
	}
	return false
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
