package repository

import (
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"nfa-dashboard/internal/model"
)

type EDCNodeSettlementRepository interface {
	ListEnabledEntities() ([]model.EDCEntity, error)
	ListTrafficPoints(entityID uint64, start, end time.Time) ([]model.EDCTraffic5m, error)
	ListTrafficPointsByDisplayNode(start, end time.Time) ([]model.EDCNodeTrafficPoint, error)
	DeleteDailySettlements(start, end time.Time) error
	DeleteMonthlySettlements(serviceMonth string) error
	UpsertDailySettlements(rows []model.SettlementNodeDaily95) error
	UpsertMonthlySettlements(rows []model.SettlementNodeMonthly95) error
	ListDailySettlements(filter map[string]interface{}, limit, offset int) ([]model.SettlementNodeDaily95, int64, error)
	ListMonthlySettlements(filter map[string]interface{}, limit, offset int) ([]model.SettlementNodeMonthly95, int64, error)
}

type edcNodeSettlementRepository struct{}

func NewEDCNodeSettlementRepository() EDCNodeSettlementRepository {
	return &edcNodeSettlementRepository{}
}

func (r *edcNodeSettlementRepository) ListEnabledEntities() ([]model.EDCEntity, error) {
	var entities []model.EDCEntity
	err := model.DB.Where("enabled = ? AND is_backup = ?", true, false).Order("region ASC, cp ASC, display_name ASC").Find(&entities).Error
	return entities, err
}

func (r *edcNodeSettlementRepository) ListTrafficPoints(entityID uint64, start, end time.Time) ([]model.EDCTraffic5m, error) {
	var points []model.EDCTraffic5m
	err := model.DB.Table("edc_traffic_5m AS t").
		Select("t.*").
		Joins("JOIN edc_entities AS e ON e.id = t.entity_id AND e.enabled = ? AND e.is_backup = ?", true, false).
		Where("t.entity_id = ? AND t.bucket_5m >= ? AND t.bucket_5m < ?", entityID, start, end).
		Order("t.bucket_5m ASC").Find(&points).Error
	return points, err
}

func (r *edcNodeSettlementRepository) ListTrafficPointsByDisplayNode(start, end time.Time) ([]model.EDCNodeTrafficPoint, error) {
	var points []model.EDCNodeTrafficPoint
	err := model.DB.Table("edc_traffic_5m AS t").
		Select(`MIN(e.id) AS entity_id, e.region, e.cp, e.display_name, t.bucket_5m,
			SUM(t.service_size) AS service_size,
			SUM(t.cache_size) AS cache_size,
			SUM(t.record_count) AS record_count`).
		Joins("JOIN edc_entities AS e ON e.id = t.entity_id").
		Where("e.enabled = ? AND e.is_backup = ? AND t.bucket_5m >= ? AND t.bucket_5m < ?", true, false, start, end).
		Group("e.region, e.cp, e.display_name, t.bucket_5m").
		Order("e.region ASC, e.cp ASC, e.display_name ASC, t.bucket_5m ASC").
		Scan(&points).Error
	return points, err
}

func (r *edcNodeSettlementRepository) DeleteDailySettlements(start, end time.Time) error {
	return model.DB.Where("settlement_time >= ? AND settlement_time < ?", start, end).
		Delete(&model.SettlementNodeDaily95{}).Error
}

func (r *edcNodeSettlementRepository) DeleteMonthlySettlements(serviceMonth string) error {
	return model.DB.Where("service_month = ?", serviceMonth).
		Delete(&model.SettlementNodeMonthly95{}).Error
}

func (r *edcNodeSettlementRepository) UpsertDailySettlements(rows []model.SettlementNodeDaily95) error {
	if len(rows) == 0 {
		return nil
	}
	return model.DB.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "entity_id"}, {Name: "settlement_time"}, {Name: "settlement_mode"}, {Name: "unit_base"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"display_name", "region", "cp", "service_month", "raw_95", "mbps_95",
			"cp_fee", "cp_bill", "cp_fee_owner_id",
			"node_construction_fee", "node_construction_bill", "node_construction_fee_owner_id",
			"rack_fee", "rack_bill", "rack_fee_owner_id",
			"other_fee", "other_bill", "other_fee_owner_id",
			"settlement_value", "daily95_fee", "daily95_bill", "traffic_bill", "total_bill", "updated_at",
		}),
	}).CreateInBatches(rows, 500).Error
}

func (r *edcNodeSettlementRepository) UpsertMonthlySettlements(rows []model.SettlementNodeMonthly95) error {
	if len(rows) == 0 {
		return nil
	}
	return model.DB.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "entity_id"}, {Name: "service_month"}, {Name: "settlement_mode"}, {Name: "unit_base"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"display_name", "region", "cp", "raw_95", "mbps_95",
			"cp_fee", "cp_bill", "cp_fee_owner_id",
			"node_construction_fee", "node_construction_bill", "node_construction_fee_owner_id",
			"rack_fee", "rack_bill", "rack_fee_owner_id",
			"other_fee", "other_bill", "other_fee_owner_id",
			"settlement_value", "settlement_time", "monthly95_fee", "monthly95_bill", "traffic_bill", "total_bill", "updated_at",
		}),
	}).CreateInBatches(rows, 500).Error
}

func (r *edcNodeSettlementRepository) ListDailySettlements(filter map[string]interface{}, limit, offset int) ([]model.SettlementNodeDaily95, int64, error) {
	var rows []model.SettlementNodeDaily95
	var total int64
	q := model.DB.Model(&model.SettlementNodeDaily95{}).
		Joins("JOIN edc_entities AS e ON e.id = settlement_node_daily95.entity_id AND e.enabled = ? AND e.is_backup = ?", true, false)
	q = applyNodeDailyMonthlyFilters(q, filter, "settlement_node_daily95")
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if total == 0 {
		return []model.SettlementNodeDaily95{}, 0, nil
	}
	err := q.Order("settlement_time DESC, region ASC, cp ASC, display_name ASC").Limit(limit).Offset(offset).Find(&rows).Error
	return rows, total, err
}

func (r *edcNodeSettlementRepository) ListMonthlySettlements(filter map[string]interface{}, limit, offset int) ([]model.SettlementNodeMonthly95, int64, error) {
	var rows []model.SettlementNodeMonthly95
	var total int64
	q := model.DB.Model(&model.SettlementNodeMonthly95{}).
		Joins("JOIN edc_entities AS e ON e.id = settlement_node_monthly95.entity_id AND e.enabled = ? AND e.is_backup = ?", true, false)
	q = applyNodeDailyMonthlyFilters(q, filter, "settlement_node_monthly95")
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if total == 0 {
		return []model.SettlementNodeMonthly95{}, 0, nil
	}
	err := q.Order("service_month DESC, region ASC, cp ASC, display_name ASC").Limit(limit).Offset(offset).Find(&rows).Error
	return rows, total, err
}

func applyNodeDailyMonthlyFilters(db *gorm.DB, filter map[string]interface{}, tableName string) *gorm.DB {
	if v, ok := filter["region"]; ok && v != "" {
		db = db.Where(tableName+".region = ?", v)
	}
	if v, ok := filter["cp"]; ok && v != "" {
		db = db.Where(tableName+".cp = ?", v)
	}
	if v, ok := filter["display_name"]; ok && v != "" {
		db = db.Where(tableName+".display_name LIKE ?", "%"+v.(string)+"%")
	}
	if v, ok := filter["service_month"]; ok && v != "" {
		db = db.Where(tableName+".service_month = ?", v)
	}
	if v, ok := filter["settlement_mode"]; ok && v != "" {
		db = db.Where(tableName+".settlement_mode = ?", v)
	}
	if v, ok := filter["unit_base"]; ok && v != "" {
		db = db.Where(tableName+".unit_base = ?", v)
	}
	if v, ok := filter["start_date"]; ok {
		db = db.Where(tableName+".settlement_time >= ?", v)
	}
	if v, ok := filter["end_date"]; ok {
		db = db.Where(tableName+".settlement_time < ?", v)
	}
	return db
}
