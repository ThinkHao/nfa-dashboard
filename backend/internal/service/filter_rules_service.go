package service

import (
	"encoding/json"
	"strconv"
	"strings"

	"gorm.io/datatypes"
	"nfa-dashboard/internal/model"
	"nfa-dashboard/internal/repository"
)

type FilterRulesService interface {
	List(name string, enabled *bool, page, pageSize int) ([]model.RateCustomerFilterRule, int64, error)
	ListOptions() ([]string, []string, error)
	Create(rule *model.RateCustomerFilterRule) (*model.RateCustomerFilterRule, error)
	Update(id uint64, updates map[string]interface{}) error
	Delete(id uint64) error
	UpdatePriority(id uint64, priority int) error
	SetEnabled(id uint64, enabled bool) error
}

type filterRulesService struct {
	repo repository.FilterRulesRepository
}

func NewFilterRulesService(repo repository.FilterRulesRepository) FilterRulesService {
	return &filterRulesService{repo: repo}
}

func (s *filterRulesService) List(name string, enabled *bool, page, pageSize int) ([]model.RateCustomerFilterRule, int64, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}
	filter := map[string]interface{}{}
	if strings.TrimSpace(name) != "" {
		filter["name"] = strings.TrimSpace(name)
	}
	if enabled != nil {
		filter["enabled"] = *enabled
	}
	limit := pageSize
	offset := (page - 1) * pageSize
	items, total, err := s.repo.List(filter, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	for i := range items {
		count, names, err := s.repo.GetMatchSummary(items[i], 20)
		if err != nil {
			return nil, 0, err
		}
		items[i].MatchCount = count
		items[i].MatchedSchoolNames = names
		items[i].MatchedSummary = summarizeMatchedSchoolNames(names, 10)
	}
	return items, total, nil
}

func (s *filterRulesService) ListOptions() ([]string, []string, error) {
	regions, err := s.repo.ListDistinctCustomerRegions()
	if err != nil {
		return nil, nil, err
	}
	cps, err := s.repo.ListDistinctCustomerCPs()
	if err != nil {
		return nil, nil, err
	}
	return regions, cps, nil
}

func (s *filterRulesService) Create(rule *model.RateCustomerFilterRule) (*model.RateCustomerFilterRule, error) {
	if rule == nil {
		return nil, NewBadRequest("nil rule")
	}
	normalized, err := normalizeFilterRule(rule)
	if err != nil {
		return nil, err
	}
	*rule = normalized
	return s.repo.Create(rule)
}

func (s *filterRulesService) Update(id uint64, updates map[string]interface{}) error {
	if id == 0 {
		return NewBadRequest("invalid id")
	}
	if len(updates) == 0 {
		return NewBadRequest("no fields to update")
	}
	if _, ok := updates["enabled"]; ok {
		return NewBadRequest("enabled cannot be updated here; use SetEnabled")
	}
	if _, ok := updates["priority"]; ok {
		return NewBadRequest("priority cannot be updated here; use UpdatePriority")
	}

	normalized := map[string]interface{}{}
	for k, v := range updates {
		switch k {
		case "name":
			name, ok := v.(string)
			if !ok {
				return NewBadRequest("name must be a string")
			}
			name = strings.TrimSpace(name)
			if name == "" {
				return NewBadRequest("name cannot be empty")
			}
			normalized[k] = name
		case "scope_region", "scope_cp", "school_name_values":
			jsonValue, err := normalizeJSONStringArrayInterface(v, true)
			if err != nil {
				return err
			}
			normalized[k] = jsonValue
		case "school_name_match_type":
			matchType, ok := v.(string)
			if !ok {
				return NewBadRequest("school_name_match_type must be a string")
			}
			matchType = normalizeSchoolNameMatchType(matchType)
			if matchType != "" && !isValidSchoolNameMatchType(matchType) {
				return NewBadRequest("invalid school_name_match_type")
			}
			normalized[k] = matchType
		default:
			normalized[k] = v
		}
	}

	if valuesRaw, ok := normalized["school_name_values"]; ok {
		values := parseStringArrayOrEmpty(valuesRaw)
		matchType, _ := normalized["school_name_match_type"].(string)
		matchType = normalizeSchoolNameMatchType(matchType)
		if len(values) > 0 && matchType == "" {
			matchType = "exact"
			normalized["school_name_match_type"] = matchType
		}
		if len(values) == 0 {
			normalized["school_name_match_type"] = ""
		}
	}

	return s.repo.Update(id, normalized)
}

func (s *filterRulesService) Delete(id uint64) error {
	if id == 0 {
		return NewBadRequest("invalid id")
	}
	return s.repo.Delete(id)
}

func (s *filterRulesService) UpdatePriority(id uint64, priority int) error {
	if id == 0 {
		return NewBadRequest("invalid id")
	}
	if priority < 0 {
		return NewBadRequest("priority must be >= 0")
	}
	return s.repo.UpdatePriority(id, priority)
}

func (s *filterRulesService) SetEnabled(id uint64, enabled bool) error {
	if id == 0 {
		return NewBadRequest("invalid id")
	}
	return s.repo.SetEnabled(id, enabled)
}

func normalizeFilterRule(rule *model.RateCustomerFilterRule) (model.RateCustomerFilterRule, error) {
	out := *rule
	out.Name = strings.TrimSpace(out.Name)
	if out.Name == "" {
		return out, NewBadRequest("name is required")
	}

	scopeRegion, err := normalizeJSONStringArrayInterface(out.ScopeRegion, true)
	if err != nil {
		return out, err
	}
	scopeCP, err := normalizeJSONStringArrayInterface(out.ScopeCP, true)
	if err != nil {
		return out, err
	}
	schoolValues, err := normalizeJSONStringArrayInterface(out.SchoolNameValues, true)
	if err != nil {
		return out, err
	}
	out.ScopeRegion = scopeRegion
	out.ScopeCP = scopeCP
	out.SchoolNameValues = schoolValues

	out.SchoolNameMatchType = normalizeSchoolNameMatchType(out.SchoolNameMatchType)
	values := parseStringArrayOrEmpty(out.SchoolNameValues)
	if len(values) == 0 {
		out.SchoolNameMatchType = ""
	} else {
		if out.SchoolNameMatchType == "" {
			out.SchoolNameMatchType = "exact"
		}
		if !isValidSchoolNameMatchType(out.SchoolNameMatchType) {
			return out, NewBadRequest("invalid school_name_match_type")
		}
	}
	return out, nil
}

func normalizeJSONStringArrayInterface(v interface{}, allowEmpty bool) (datatypes.JSON, error) {
	arr, err := normalizeStringArrayInput(v)
	if err != nil {
		return nil, err
	}
	if !allowEmpty && len(arr) == 0 {
		return nil, NewBadRequest("array cannot be empty")
	}
	bs, err := json.Marshal(arr)
	if err != nil {
		return nil, NewBadRequest("must be JSON array of strings")
	}
	return datatypes.JSON(bs), nil
}

func normalizeStringArrayInput(v interface{}) ([]string, error) {
	if v == nil {
		return []string{}, nil
	}
	switch t := v.(type) {
	case datatypes.JSON:
		return parseNormalizedStringArray([]byte(t))
	case []byte:
		return parseNormalizedStringArray(t)
	case json.RawMessage:
		return parseNormalizedStringArray([]byte(t))
	case string:
		if strings.TrimSpace(t) == "" {
			return []string{}, nil
		}
		return parseNormalizedStringArray([]byte(t))
	case []string:
		return trimAndCompactStrings(t), nil
	case []interface{}:
		out := make([]string, 0, len(t))
		for _, item := range t {
			s, ok := item.(string)
			if !ok {
				return nil, NewBadRequest("must be JSON array of strings")
			}
			out = append(out, s)
		}
		return trimAndCompactStrings(out), nil
	default:
		bs, err := json.Marshal(v)
		if err != nil {
			return nil, NewBadRequest("must be JSON array of strings")
		}
		return parseNormalizedStringArray(bs)
	}
}

func parseNormalizedStringArray(data []byte) ([]string, error) {
	if len(data) == 0 {
		return []string{}, nil
	}
	var arr []string
	if err := json.Unmarshal(data, &arr); err != nil {
		return nil, NewBadRequest("must be JSON array of strings")
	}
	return trimAndCompactStrings(arr), nil
}

func parseStringArrayOrEmpty(v interface{}) []string {
	arr, err := normalizeStringArrayInput(v)
	if err != nil {
		return []string{}
	}
	return arr
}

func trimAndCompactStrings(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		out = append(out, value)
	}
	return out
}

func normalizeSchoolNameMatchType(v string) string {
	return strings.ToLower(strings.TrimSpace(v))
}

func isValidSchoolNameMatchType(v string) bool {
	return v == "exact" || v == "contains"
}

func matchRateCustomerFilterRule(customer model.RateCustomer, rule model.RateCustomerFilterRule) (bool, error) {
	regions, err := normalizeStringArrayInput(rule.ScopeRegion)
	if err != nil {
		return false, err
	}
	if len(regions) > 0 && !containsString(regions, customer.Region) {
		return false, nil
	}
	cps, err := normalizeStringArrayInput(rule.ScopeCP)
	if err != nil {
		return false, err
	}
	if len(cps) > 0 && !containsString(cps, customer.CP) {
		return false, nil
	}
	values, err := normalizeStringArrayInput(rule.SchoolNameValues)
	if err != nil {
		return false, err
	}
	if len(values) == 0 {
		return true, nil
	}

	schoolName := strings.TrimSpace(derefFilterString(customer.SchoolName))
	matchType := normalizeSchoolNameMatchType(rule.SchoolNameMatchType)
	if matchType == "" {
		matchType = "exact"
	}
	switch matchType {
	case "exact":
		return containsString(values, schoolName), nil
	case "contains":
		for _, value := range values {
			if strings.Contains(schoolName, value) {
				return true, nil
			}
		}
		return false, nil
	default:
		return false, NewBadRequest("invalid school_name_match_type")
	}
}

func summarizeMatchedSchoolNames(names []string, limit int) string {
	trimmed := trimAndCompactStrings(names)
	if len(trimmed) == 0 {
		return ""
	}
	if limit <= 0 || len(trimmed) <= limit {
		return strings.Join(trimmed, "、")
	}
	return strings.Join(trimmed[:limit], "、") + " 等 " + itoa(len(trimmed)-limit) + " 所"
}

func containsString(values []string, target string) bool {
	target = strings.TrimSpace(target)
	for _, value := range values {
		if strings.TrimSpace(value) == target {
			return true
		}
	}
	return false
}

func derefFilterString(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}

func itoa(v int) string {
	return strconv.Itoa(v)
}
