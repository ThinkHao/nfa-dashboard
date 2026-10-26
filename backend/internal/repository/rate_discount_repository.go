package repository

import (
	"nfa-dashboard/internal/model"
	"strings"
)

// RateDiscountRepository 管理 rate_discount_rule 与 rate_discount_rule_item 的持久化

type RateDiscountRepository interface {
	ListRules(filter map[string]interface{}, limit, offset int) ([]model.RateDiscountRule, int64, error)
	GetRuleByID(id uint64) (*model.RateDiscountRule, error)
	CreateRule(rule *model.RateDiscountRule) (*model.RateDiscountRule, error)
	UpdateRule(id uint64, updates map[string]interface{}) error
	DeleteRule(id uint64) error

	ListItemsByRuleID(ruleID uint64) ([]model.RateDiscountRuleItem, error)
	ReplaceItems(ruleID uint64, items []model.RateDiscountRuleItem) error
	ListScopesByRuleIDs(ruleIDs []uint64) ([]model.RateDiscountRuleScope, error)
	ListSchoolTargets(regions []string) ([]model.RateDiscountRuleScope, error)
	FindMatchedRule(region, cp, schoolName string) (*model.RateDiscountRule, error)
	CreateRulesBatch(drafts []model.RateDiscountRuleDraft) ([]model.RateDiscountRule, error)
	UpdateRuleWithScopes(id uint64, updates map[string]interface{}, scopes []model.RateDiscountRuleScope) error
}

type rateDiscountRepository struct{}

func NewRateDiscountRepository() RateDiscountRepository { return &rateDiscountRepository{} }

func (r *rateDiscountRepository) ListRules(filter map[string]interface{}, limit, offset int) ([]model.RateDiscountRule, int64, error) {
	var items []model.RateDiscountRule
	var total int64
	q := model.DB.Model(&model.RateDiscountRule{})
	if v, ok := filter["name"]; ok && v != "" {
		q = q.Where("name LIKE ?", "%"+v.(string)+"%")
	}
	if v, ok := filter["scope_type"]; ok && v != "" {
		q = q.Where("scope_type = ?", v)
	}
	if v, ok := filter["enabled"]; ok {
		q = q.Where("enabled = ?", v)
	}
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if total == 0 {
		return []model.RateDiscountRule{}, 0, nil
	}
	if limit > 0 {
		q = q.Limit(limit)
	}
	if offset > 0 {
		q = q.Offset(offset)
	}
	if err := q.Order("priority ASC").Order("updated_at DESC").Order("id DESC").Find(&items).Error; err != nil {
		return nil, 0, err
	}
	ids := make([]uint64, 0, len(items))
	for _, item := range items {
		ids = append(ids, item.ID)
	}
	scopes, err := r.ListScopesByRuleIDs(ids)
	if err != nil {
		return nil, 0, err
	}
	byRule := make(map[uint64][]model.RateDiscountRuleScope, len(items))
	for _, scope := range scopes {
		byRule[scope.RuleID] = append(byRule[scope.RuleID], scope)
	}
	for i := range items {
		items[i].Scopes = byRule[items[i].ID]
	}
	return items, total, nil
}

func (r *rateDiscountRepository) GetRuleByID(id uint64) (*model.RateDiscountRule, error) {
	if id == 0 {
		return nil, nil
	}
	var rule model.RateDiscountRule
	if err := model.DB.First(&rule, id).Error; err != nil {
		return nil, err
	}
	scopes, err := r.ListScopesByRuleIDs([]uint64{id})
	if err != nil {
		return nil, err
	}
	rule.Scopes = scopes
	return &rule, nil
}

func (r *rateDiscountRepository) CreateRule(rule *model.RateDiscountRule) (*model.RateDiscountRule, error) {
	tx := model.DB.Begin()
	if tx.Error != nil {
		return nil, tx.Error
	}
	if err := tx.Create(rule).Error; err != nil {
		tx.Rollback()
		return nil, err
	}
	scope := model.RateDiscountRuleScope{RuleID: rule.ID, ScopeType: rule.ScopeType, ScopeKey: rule.ScopeKey}
	switch rule.ScopeType {
	case "region":
		scope.Region = rule.ScopeKey
	case "cp":
		scope.CP = rule.ScopeKey
	case "school":
		scope.SchoolName = rule.ScopeKey
	}
	if err := tx.Create(&scope).Error; err != nil {
		tx.Rollback()
		return nil, err
	}
	if err := tx.Commit().Error; err != nil {
		return nil, err
	}
	rule.Scopes = []model.RateDiscountRuleScope{scope}
	return rule, nil
}

func (r *rateDiscountRepository) UpdateRule(id uint64, updates map[string]interface{}) error {
	if id == 0 {
		return nil
	}
	if len(updates) == 0 {
		return nil
	}
	return model.DB.Model(&model.RateDiscountRule{}).Where("id = ?", id).Updates(updates).Error
}

func (r *rateDiscountRepository) DeleteRule(id uint64) error {
	if id == 0 {
		return nil
	}
	// 直接物理删除，明细表由外键级联删除
	return model.DB.Where("id = ?", id).Delete(&model.RateDiscountRule{}).Error
}

func (r *rateDiscountRepository) ListItemsByRuleID(ruleID uint64) ([]model.RateDiscountRuleItem, error) {
	var items []model.RateDiscountRuleItem
	if ruleID == 0 {
		return []model.RateDiscountRuleItem{}, nil
	}
	if err := model.DB.Where("rule_id = ?", ruleID).Order("from_year ASC").Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

func (r *rateDiscountRepository) ReplaceItems(ruleID uint64, items []model.RateDiscountRuleItem) error {
	if ruleID == 0 {
		return nil
	}
	tx := model.DB.Begin()
	if err := tx.Where("rule_id = ?", ruleID).Delete(&model.RateDiscountRuleItem{}).Error; err != nil {
		tx.Rollback()
		return err
	}
	if len(items) > 0 {
		for i := range items {
			items[i].RuleID = ruleID
		}
		if err := tx.Create(&items).Error; err != nil {
			tx.Rollback()
			return err
		}
	}
	return tx.Commit().Error
}

func (r *rateDiscountRepository) ListScopesByRuleIDs(ruleIDs []uint64) ([]model.RateDiscountRuleScope, error) {
	if len(ruleIDs) == 0 {
		return []model.RateDiscountRuleScope{}, nil
	}
	var scopes []model.RateDiscountRuleScope
	if err := model.DB.Where("rule_id IN ?", ruleIDs).Order("scope_type ASC, region ASC, cp ASC, school_name ASC, scope_key ASC").Find(&scopes).Error; err != nil {
		return nil, err
	}
	return scopes, nil
}

func (r *rateDiscountRepository) ListSchoolTargets(regions []string) ([]model.RateDiscountRuleScope, error) {
	var scopes []model.RateDiscountRuleScope
	q := model.DB.Model(&model.RateCustomer{}).
		Select("region, cp, school_name").
		Where("school_name IS NOT NULL AND TRIM(school_name) <> ''")
	if len(regions) > 0 {
		q = q.Where("region IN ?", regions)
	}
	if err := q.Group("region, cp, school_name").Order("region ASC, cp ASC, school_name ASC").Scan(&scopes).Error; err != nil {
		return nil, err
	}
	for i := range scopes {
		scopes[i].ScopeType = "school"
	}
	return scopes, nil
}

func (r *rateDiscountRepository) FindMatchedRule(region, cp, schoolName string) (*model.RateDiscountRule, error) {
	var rule model.RateDiscountRule
	q := model.DB.Table("rate_discount_rule r").
		Select("r.*").
		Joins("JOIN rate_discount_rule_scope s ON s.rule_id = r.id").
		Where("r.enabled = ?", true).
		Where(`(
			(s.scope_type = 'school' AND s.school_name = ? AND (
				(s.region IS NULL AND s.cp IS NULL) OR (s.region = ? AND s.cp = ?)
			)) OR
			(s.scope_type = 'cp' AND s.scope_key = ?) OR
			(s.scope_type = 'region' AND s.scope_key = ?) OR
			s.scope_type = 'global'
		)`, strings.TrimSpace(schoolName), strings.TrimSpace(region), strings.TrimSpace(cp), strings.TrimSpace(cp), strings.TrimSpace(region)).
		Order("CASE s.scope_type WHEN 'school' THEN 1 WHEN 'cp' THEN 2 WHEN 'region' THEN 3 WHEN 'global' THEN 4 ELSE 5 END ASC").
		Order("r.priority ASC").Order("r.updated_at DESC").Order("r.id DESC")
	if err := q.Limit(1).Scan(&rule).Error; err != nil {
		return nil, err
	}
	if rule.ID == 0 {
		return nil, nil
	}
	return &rule, nil
}

func (r *rateDiscountRepository) CreateRulesBatch(drafts []model.RateDiscountRuleDraft) ([]model.RateDiscountRule, error) {
	tx := model.DB.Begin()
	if tx.Error != nil {
		return nil, tx.Error
	}
	created := make([]model.RateDiscountRule, 0, len(drafts))
	for _, draft := range drafts {
		rule := draft.Rule
		if err := tx.Create(&rule).Error; err != nil {
			tx.Rollback()
			return nil, err
		}
		items := append([]model.RateDiscountRuleItem(nil), draft.Items...)
		if len(items) > 0 {
			for i := range items {
				items[i].RuleID = rule.ID
			}
			if err := tx.Create(&items).Error; err != nil {
				tx.Rollback()
				return nil, err
			}
		}
		scopes := append([]model.RateDiscountRuleScope(nil), draft.Scopes...)
		if len(scopes) > 0 {
			for i := range scopes {
				scopes[i].ID = 0
				scopes[i].RuleID = rule.ID
			}
			if err := tx.Create(&scopes).Error; err != nil {
				tx.Rollback()
				return nil, err
			}
		}
		rule.Scopes = scopes
		created = append(created, rule)
	}
	if err := tx.Commit().Error; err != nil {
		return nil, err
	}
	return created, nil
}

func (r *rateDiscountRepository) UpdateRuleWithScopes(id uint64, updates map[string]interface{}, scopes []model.RateDiscountRuleScope) error {
	tx := model.DB.Begin()
	if tx.Error != nil {
		return tx.Error
	}
	if len(updates) > 0 {
		if err := tx.Model(&model.RateDiscountRule{}).Where("id = ?", id).Updates(updates).Error; err != nil {
			tx.Rollback()
			return err
		}
	}
	if err := tx.Where("rule_id = ?", id).Delete(&model.RateDiscountRuleScope{}).Error; err != nil {
		tx.Rollback()
		return err
	}
	if len(scopes) > 0 {
		for i := range scopes {
			scopes[i].ID = 0
			scopes[i].RuleID = id
		}
		if err := tx.Create(&scopes).Error; err != nil {
			tx.Rollback()
			return err
		}
	}
	return tx.Commit().Error
}
