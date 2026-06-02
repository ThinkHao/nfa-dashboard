package repository

import (
	"context"
	"strings"
	"time"

	"gorm.io/gorm"
	"nfa-dashboard/internal/model"
)

type EDCRepository interface {
	ListEntities(filter model.EDCEntityFilter) ([]model.EDCEntity, int64, error)
	ListRegions(allowedEntityIDs []uint64) ([]string, error)
	ListCPs(allowedEntityIDs []uint64) ([]string, error)
	GetTrafficData(filter model.EDCTrafficFilter) ([]model.EDCTrafficResponse, error)
	GetTrafficSummary(filter model.EDCTrafficFilter) (model.EDCTrafficResponse, error)
}

type edcRepository struct{}

func NewEDCRepository() EDCRepository { return &edcRepository{} }

func (r *edcRepository) ListEntities(filter model.EDCEntityFilter) ([]model.EDCEntity, int64, error) {
	var entities []model.EDCEntity
	var total int64
	q := model.DB.Model(&model.EDCEntity{})
	q = applyEDCEntityFilter(q, filter)
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	limit := filter.Limit
	if limit <= 0 {
		limit = 100
	}
	if err := q.Order("region ASC, cp ASC, display_name ASC, edc_name ASC").
		Limit(limit).Offset(filter.Offset).Find(&entities).Error; err != nil {
		return nil, 0, err
	}
	return entities, total, nil
}

func (r *edcRepository) ListRegions(allowedEntityIDs []uint64) ([]string, error) {
	var regions []string
	q := model.DB.Model(&model.EDCEntity{}).Where("enabled = ? AND is_backup = ?", true, false)
	q = applyAllowedEDCEntityIDs(q, allowedEntityIDs)
	if err := q.Distinct().Order("region ASC").Pluck("region", &regions).Error; err != nil {
		return nil, err
	}
	return regions, nil
}

func (r *edcRepository) ListCPs(allowedEntityIDs []uint64) ([]string, error) {
	var cps []string
	q := model.DB.Model(&model.EDCEntity{}).Where("enabled = ? AND is_backup = ?", true, false)
	q = applyAllowedEDCEntityIDs(q, allowedEntityIDs)
	if err := q.Distinct().Order("cp ASC").Pluck("cp", &cps).Error; err != nil {
		return nil, err
	}
	return cps, nil
}

func (r *edcRepository) GetTrafficData(filter model.EDCTrafficFilter) ([]model.EDCTrafficResponse, error) {
	filter = normalizeEDCTimeFilter(filter)
	selectDims := []string{"t.bucket_5m"}
	selectExpr := []string{
		"t.bucket_5m",
		"CAST(0 AS UNSIGNED) AS entity_id",
		"'' AS display_name",
		"'' AS region",
		"'' AS cp",
		"SUM(t.service_size) AS service_size",
		"SUM(t.cache_size) AS cache_size",
	}
	groupBy := []string{"t.bucket_5m"}
	orderBy := "t.bucket_5m ASC"

	if filter.DisplayName != "" {
		selectExpr[1] = "t.entity_id"
		selectExpr[2] = "t.display_name"
		selectExpr[3] = "t.region"
		selectExpr[4] = "t.cp"
		groupBy = append(groupBy, "t.entity_id", "t.display_name", "t.region", "t.cp")
		orderBy += ", t.region ASC, t.cp ASC, t.display_name ASC"
	} else {
		if filter.Region != "" {
			selectExpr[3] = "t.region"
			groupBy = append(groupBy, "t.region")
			selectDims = append(selectDims, "t.region")
		}
		if filter.CP != "" {
			selectExpr[4] = "t.cp"
			groupBy = append(groupBy, "t.cp")
			selectDims = append(selectDims, "t.cp")
		}
		if len(selectDims) > 1 {
			orderBy += ", t.region ASC, t.cp ASC"
		}
	}

	q := model.DB.Table("edc_traffic_5m AS t").
		Joins("JOIN edc_entities AS e ON e.id = t.entity_id AND e.enabled = ? AND e.is_backup = ?", true, false).
		Select(strings.Join(selectExpr, ", "))
	q = applyEDCTrafficFilter(q, filter)
	q = q.Group(strings.Join(groupBy, ", ")).Order(orderBy)

	var rows []model.EDCTrafficResponse
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	if err := q.WithContext(ctx).Scan(&rows).Error; err != nil {
		return nil, err
	}
	for i := range rows {
		rows[i].Total = rows[i].ServiceSize + rows[i].CacheSize
	}
	return rows, nil
}

func (r *edcRepository) GetTrafficSummary(filter model.EDCTrafficFilter) (model.EDCTrafficResponse, error) {
	filter = normalizeEDCTimeFilter(filter)
	var row model.EDCTrafficResponse
	q := model.DB.Table("edc_traffic_5m AS t").
		Joins("JOIN edc_entities AS e ON e.id = t.entity_id AND e.enabled = ? AND e.is_backup = ?", true, false).
		Select("SUM(t.service_size) AS service_size, SUM(t.cache_size) AS cache_size")
	q = applyEDCTrafficFilter(q, filter)
	if err := q.Scan(&row).Error; err != nil {
		return row, err
	}
	row.Total = row.ServiceSize + row.CacheSize
	return row, nil
}

type EDCTrafficScopeRepository interface {
	ListByUser(userID uint64) ([]model.EDCTrafficScopeRuleGroup, error)
	ReplaceByUser(userID uint64, groups []model.EDCTrafficScopeRuleGroup) error
	MatchEntities(dimension, value string) ([]model.EDCEntity, error)
	ListAllEntities() ([]model.EDCEntity, error)
}

type edcTrafficScopeRepository struct{}

func NewEDCTrafficScopeRepository() EDCTrafficScopeRepository {
	return &edcTrafficScopeRepository{}
}

func (r *edcTrafficScopeRepository) ListByUser(userID uint64) ([]model.EDCTrafficScopeRuleGroup, error) {
	var groups []model.EDCTrafficScopeRuleGroup
	if err := model.DB.Preload("Conditions").Where("user_id = ?", userID).Order("id ASC").Find(&groups).Error; err != nil {
		return nil, err
	}
	return groups, nil
}

func (r *edcTrafficScopeRepository) ReplaceByUser(userID uint64, groups []model.EDCTrafficScopeRuleGroup) error {
	return model.DB.Transaction(func(tx *gorm.DB) error {
		var existingIDs []uint64
		if err := tx.Model(&model.EDCTrafficScopeRuleGroup{}).Where("user_id = ?", userID).Pluck("id", &existingIDs).Error; err != nil {
			return err
		}
		if len(existingIDs) > 0 {
			if err := tx.Where("group_id IN ?", existingIDs).Delete(&model.EDCTrafficScopeCondition{}).Error; err != nil {
				return err
			}
		}
		if err := tx.Where("user_id = ?", userID).Delete(&model.EDCTrafficScopeRuleGroup{}).Error; err != nil {
			return err
		}
		for _, group := range groups {
			next := model.EDCTrafficScopeRuleGroup{UserID: userID, RuleType: group.RuleType}
			if err := tx.Create(&next).Error; err != nil {
				return err
			}
			conditions := make([]model.EDCTrafficScopeCondition, 0, len(group.Conditions))
			for _, condition := range group.Conditions {
				conditions = append(conditions, model.EDCTrafficScopeCondition{
					GroupID:        next.ID,
					DimensionType:  condition.DimensionType,
					DimensionValue: strings.TrimSpace(condition.DimensionValue),
				})
			}
			if len(conditions) > 0 {
				if err := tx.Create(&conditions).Error; err != nil {
					return err
				}
			}
		}
		return nil
	})
}

func (r *edcTrafficScopeRepository) MatchEntities(dimension, value string) ([]model.EDCEntity, error) {
	q := model.DB.Model(&model.EDCEntity{}).Where("enabled = ? AND is_backup = ?", true, false)
	switch dimension {
	case model.EDCTrafficScopeDimensionRegion:
		q = q.Where("region = ?", value)
	case model.EDCTrafficScopeDimensionCP:
		q = q.Where("cp = ?", value)
	case model.EDCTrafficScopeDimensionEntity:
		q = q.Where("display_name = ? OR edc_name = ? OR CAST(id AS CHAR) = ?", value, value, value)
	default:
		return []model.EDCEntity{}, nil
	}
	var entities []model.EDCEntity
	if err := q.Find(&entities).Error; err != nil {
		return nil, err
	}
	return entities, nil
}

func (r *edcTrafficScopeRepository) ListAllEntities() ([]model.EDCEntity, error) {
	var entities []model.EDCEntity
	if err := model.DB.Where("enabled = ? AND is_backup = ?", true, false).Order("region ASC, cp ASC, display_name ASC").Find(&entities).Error; err != nil {
		return nil, err
	}
	return entities, nil
}

func normalizeEDCTimeFilter(filter model.EDCTrafficFilter) model.EDCTrafficFilter {
	if !filter.StartTime.IsZero() {
		filter.StartTime = filter.StartTime.In(time.Local)
	}
	if !filter.EndTime.IsZero() {
		filter.EndTime = filter.EndTime.In(time.Local)
	}
	if filter.StartTime.IsZero() && filter.EndTime.IsZero() {
		filter.EndTime = time.Now()
		filter.StartTime = filter.EndTime.AddDate(0, 0, -1)
	} else if filter.StartTime.IsZero() {
		filter.StartTime = filter.EndTime.AddDate(0, 0, -1)
	} else if filter.EndTime.IsZero() {
		filter.EndTime = time.Now()
	}
	return filter
}

func applyEDCEntityFilter(q *gorm.DB, filter model.EDCEntityFilter) *gorm.DB {
	q = q.Where("is_backup = ?", false)
	if filter.EnabledOnly {
		q = q.Where("enabled = ?", true)
	}
	if filter.DisplayName != "" {
		if strings.ContainsAny(filter.DisplayName, "%_") {
			q = q.Where("display_name LIKE ? OR edc_name LIKE ?", filter.DisplayName, filter.DisplayName)
		} else {
			q = q.Where("display_name = ? OR edc_name = ?", filter.DisplayName, filter.DisplayName)
		}
	}
	if filter.Region != "" {
		q = q.Where("region = ?", filter.Region)
	}
	if filter.CP != "" {
		q = q.Where("cp = ?", filter.CP)
	}
	return applyAllowedEDCEntityIDs(q, filter.AllowedEntityIDs)
}

func applyEDCTrafficFilter(q *gorm.DB, filter model.EDCTrafficFilter) *gorm.DB {
	if !filter.StartTime.IsZero() {
		q = q.Where("t.bucket_5m >= ?", filter.StartTime)
	}
	if !filter.EndTime.IsZero() {
		q = q.Where("t.bucket_5m < ?", filter.EndTime)
	}
	if filter.DisplayName != "" {
		if strings.ContainsAny(filter.DisplayName, "%_") {
			q = q.Where("t.display_name LIKE ?", filter.DisplayName)
		} else {
			q = q.Where("t.display_name = ?", filter.DisplayName)
		}
	}
	if filter.Region != "" {
		q = q.Where("t.region = ?", filter.Region)
	}
	if filter.CP != "" {
		q = q.Where("t.cp = ?", filter.CP)
	}
	return applyAllowedEDCTrafficEntityIDs(q, filter.AllowedEntityIDs)
}

func applyAllowedEDCEntityIDs(q *gorm.DB, ids []uint64) *gorm.DB {
	if len(ids) == 0 {
		return q
	}
	return q.Where("id IN ?", ids)
}

func applyAllowedEDCTrafficEntityIDs(q *gorm.DB, ids []uint64) *gorm.DB {
	if len(ids) == 0 {
		return q
	}
	return q.Where("t.entity_id IN ?", ids)
}
