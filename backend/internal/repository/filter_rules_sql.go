package repository

import (
	"encoding/json"
	"strings"

	"gorm.io/gorm"
	"nfa-dashboard/internal/model"
)

func excludeFilteredCustomerRates(q *gorm.DB, customerAlias string) *gorm.DB {
	rules, err := loadEnabledCustomerFilterRules()
	if err != nil || len(rules) == 0 {
		return q
	}
	clause, args := buildFilterRulesPredicate(customerAlias, rules)
	if strings.TrimSpace(clause) == "" {
		return q
	}
	return q.Where("NOT ("+clause+")", args...)
}

func applySingleFilterRuleScope(q *gorm.DB, customerAlias string, rule model.RateCustomerFilterRule) *gorm.DB {
	clause, args := buildSingleFilterRulePredicate(customerAlias, rule)
	if strings.TrimSpace(clause) == "" {
		return q
	}
	return q.Where(clause, args...)
}

func enabledFilterRuleExistsClause(customerAlias string) string {
	rules, err := loadEnabledCustomerFilterRules()
	if err != nil || len(rules) == 0 {
		return "SELECT 1 WHERE 0 = 1"
	}
	clause, args := buildFilterRulesPredicate(customerAlias, rules)
	if strings.TrimSpace(clause) == "" {
		return "SELECT 1 WHERE 0 = 1"
	}
	return "SELECT 1 WHERE " + interpolateFilterPredicateForStaticSQL(clause, args)
}

func loadEnabledCustomerFilterRules() ([]model.RateCustomerFilterRule, error) {
	var rules []model.RateCustomerFilterRule
	err := model.DB.
		Where("enabled = ?", true).
		Order("priority ASC").
		Order("updated_at DESC").
		Find(&rules).Error
	return rules, err
}

func buildFilterRulesPredicate(customerAlias string, rules []model.RateCustomerFilterRule) (string, []interface{}) {
	parts := make([]string, 0, len(rules))
	args := make([]interface{}, 0)
	for _, rule := range rules {
		clause, ruleArgs := buildSingleFilterRulePredicate(customerAlias, rule)
		if strings.TrimSpace(clause) == "" {
			continue
		}
		parts = append(parts, "("+clause+")")
		args = append(args, ruleArgs...)
	}
	return strings.Join(parts, " OR "), args
}

func buildSingleFilterRulePredicate(customerAlias string, rule model.RateCustomerFilterRule) (string, []interface{}) {
	parts := make([]string, 0, 3)
	args := make([]interface{}, 0)

	if regions := parseFilterRuleStringArray(rule.ScopeRegion); len(regions) > 0 {
		parts = append(parts, customerAlias+".region IN ?")
		args = append(args, regions)
	}
	if cps := parseFilterRuleStringArray(rule.ScopeCP); len(cps) > 0 {
		parts = append(parts, customerAlias+".cp IN ?")
		args = append(args, cps)
	}
	if values := parseFilterRuleStringArray(rule.SchoolNameValues); len(values) > 0 {
		if strings.EqualFold(strings.TrimSpace(rule.SchoolNameMatchType), "contains") {
			likeParts := make([]string, 0, len(values))
			for _, value := range values {
				likeParts = append(likeParts, customerAlias+".school_name LIKE ?")
				args = append(args, "%"+value+"%")
			}
			parts = append(parts, "("+strings.Join(likeParts, " OR ")+")")
		} else {
			parts = append(parts, customerAlias+".school_name IN ?")
			args = append(args, values)
		}
	}
	if len(parts) == 0 {
		return "", nil
	}
	return strings.Join(parts, " AND "), args
}

func parseFilterRuleStringArray(data []byte) []string {
	if len(data) == 0 {
		return nil
	}
	var arr []string
	if err := json.Unmarshal(data, &arr); err != nil {
		return nil
	}
	out := make([]string, 0, len(arr))
	for _, item := range arr {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		out = append(out, item)
	}
	return out
}

func interpolateFilterPredicateForStaticSQL(clause string, args []interface{}) string {
	out := clause
	for _, arg := range args {
		replacement := "NULL"
		switch v := arg.(type) {
		case []string:
			quoted := make([]string, 0, len(v))
			for _, item := range v {
				quoted = append(quoted, "'"+strings.ReplaceAll(item, "'", "''")+"'")
			}
			replacement = "(" + strings.Join(quoted, ",") + ")"
		case string:
			replacement = "'" + strings.ReplaceAll(v, "'", "''") + "'"
		}
		out = strings.Replace(out, "?", replacement, 1)
	}
	return out
}
