package repository

import (
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"nfa-dashboard/internal/model"
)

type EDCNodeSettlementRepository interface {
	ListEnabledEntities() ([]model.EDCEntity, error)
	ListTrafficPoints(entityID uint64, start, end time.Time) ([]model.EDCTraffic5m, error)
	ListTrafficPointsByEntity(start, end time.Time) ([]model.EDCNodeTrafficPoint, error)
	ListTrafficPointsByEntities(entityIDs []uint64, start, end time.Time) ([]model.EDCNodeTrafficPoint, error)
	ListTrafficPointsByDisplayNode(start, end time.Time) ([]model.EDCNodeTrafficPoint, error)
	ExistsTrafficPointByDisplayNode(start, end time.Time) (bool, error)
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
	err := model.DB.Where("enabled = ? AND is_backup = ? AND (entity_type = ? OR entity_type IS NULL)", true, false, "node").Order("region ASC, cp ASC, display_name ASC").Find(&entities).Error
	return entities, err
}

func (r *edcNodeSettlementRepository) ListTrafficPoints(entityID uint64, start, end time.Time) ([]model.EDCTraffic5m, error) {
	var points []model.EDCTraffic5m
	err := model.DB.Table("edc_traffic_5m AS t").
		Select("t.*").
		Joins("JOIN edc_entities AS e ON e.id = t.entity_id AND e.enabled = ? AND e.is_backup = ? AND (e.entity_type = ? OR e.entity_type IS NULL)", true, false, "node").
		Where("t.entity_id = ? AND t.bucket_5m >= ? AND t.bucket_5m < ?", entityID, start, end).
		Order("t.bucket_5m ASC").Find(&points).Error
	for i := range points {
		points[i].ServiceSize = nonNegativeEDCSize(points[i].ServiceSize)
		points[i].CacheSize = nonNegativeEDCSize(points[i].CacheSize)
	}
	return points, err
}

func (r *edcNodeSettlementRepository) ListTrafficPointsByEntity(start, end time.Time) ([]model.EDCNodeTrafficPoint, error) {
	return r.ListTrafficPointsByEntities(nil, start, end)
}

func (r *edcNodeSettlementRepository) ListTrafficPointsByEntities(entityIDs []uint64, start, end time.Time) ([]model.EDCNodeTrafficPoint, error) {
	var points []model.EDCNodeTrafficPoint
	q := model.DB.Table("edc_traffic_5m AS t").
		Select(`e.id AS entity_id, e.region, e.cp, e.display_name, t.bucket_5m,
			SUM(GREATEST(t.service_size, 0)) AS service_size,
			SUM(GREATEST(t.cache_size, 0)) AS cache_size,
			SUM(t.record_count) AS record_count`).
		Joins("JOIN edc_entities AS e ON e.id = t.entity_id").
		Where("e.enabled = ? AND e.is_backup = ? AND (e.entity_type = ? OR e.entity_type IS NULL) AND t.bucket_5m >= ? AND t.bucket_5m < ?", true, false, "node", start, end)
	if len(entityIDs) > 0 {
		q = q.Where("t.entity_id IN ?", entityIDs)
	}
	err := q.Group("e.id, e.region, e.cp, e.display_name, t.bucket_5m").
		Order("e.region ASC, e.cp ASC, e.display_name ASC, t.bucket_5m ASC").Scan(&points).Error
	return points, err
}

func (r *edcNodeSettlementRepository) ListTrafficPointsByDisplayNode(start, end time.Time) ([]model.EDCNodeTrafficPoint, error) {
	var points []model.EDCNodeTrafficPoint
	err := model.DB.Table("edc_traffic_5m AS t").
		Select(`MIN(e.id) AS entity_id, e.region, e.cp, e.display_name, t.bucket_5m,
			SUM(GREATEST(t.service_size, 0)) AS service_size,
			SUM(GREATEST(t.cache_size, 0)) AS cache_size,
			SUM(t.record_count) AS record_count`).
		Joins("JOIN edc_entities AS e ON e.id = t.entity_id").
		Where("e.enabled = ? AND e.is_backup = ? AND (e.entity_type = ? OR e.entity_type IS NULL) AND t.bucket_5m >= ? AND t.bucket_5m < ?", true, false, "node", start, end).
		Group("e.region, e.cp, e.display_name, t.bucket_5m").
		Order("e.region ASC, e.cp ASC, e.display_name ASC, t.bucket_5m ASC").
		Scan(&points).Error
	return points, err
}

func nonNegativeEDCSize(value int64) int64 {
	if value < 0 {
		return 0
	}
	return value
}

func (r *edcNodeSettlementRepository) ExistsTrafficPointByDisplayNode(start, end time.Time) (bool, error) {
	var hit int
	tx := model.DB.Table("edc_traffic_5m AS t").
		Select("1").
		Joins("JOIN edc_entities AS e ON e.id = t.entity_id").
		Where("e.enabled = ? AND e.is_backup = ? AND (e.entity_type = ? OR e.entity_type IS NULL) AND t.bucket_5m >= ? AND t.bucket_5m < ?", true, false, "node", start, end).
		Limit(1).
		Scan(&hit)
	if tx.Error != nil {
		return false, tx.Error
	}
	return tx.RowsAffected > 0, nil
}

// HasUnreadyEntityCandidates reports whether an unknown or failed EDC mapping
// overlaps the requested settlement period. The candidate table is created by
// edc-extractor; older dashboard databases continue to operate when it is absent.
func (r *edcNodeSettlementRepository) HasUnreadyEntityCandidates(start, end time.Time) (bool, error) {
	var hit int
	tx := model.DB.Table("edc_entity_candidates").
		Select("1").
		Where("is_backup = ?", false).
		Where("status IN ?", []string{"pending", "backfill_pending", "backfilling", "failed"}).
		Where("(latest_seen_at IS NULL OR latest_seen_at >= ?) AND (first_seen_at IS NULL OR first_seen_at < ?)", start, end).
		Limit(1).
		Scan(&hit)
	if tx.Error != nil {
		message := strings.ToLower(tx.Error.Error())
		if strings.Contains(message, "doesn't exist") || strings.Contains(message, "unknown table") {
			return false, nil
		}
		return false, tx.Error
	}
	return tx.RowsAffected > 0, nil
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
		Columns: []clause.Column{{Name: "billing_subject_type"}, {Name: "billing_subject_id"}, {Name: "settlement_time"}, {Name: "settlement_mode"}, {Name: "unit_base"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"entity_id", "display_name", "billing_display_name", "region", "cp", "service_month", "raw_95", "mbps_95",
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
		Columns: []clause.Column{{Name: "billing_subject_type"}, {Name: "billing_subject_id"}, {Name: "service_month"}, {Name: "settlement_mode"}, {Name: "unit_base"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"entity_id", "display_name", "billing_display_name", "region", "cp", "raw_95", "mbps_95",
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
	q := model.DB.Model(&model.SettlementNodeDaily95{})
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
	q := model.DB.Model(&model.SettlementNodeMonthly95{})
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
		db = db.Where(tableName+".billing_display_name LIKE ?", "%"+v.(string)+"%")
	}
	if v, ok := filter["billing_subject_type"]; ok && v != "" {
		db = db.Where(tableName+".billing_subject_type = ?", v)
	}
	if v, ok := filter["billing_subject_id"]; ok && v != "" {
		db = db.Where(tableName+".billing_subject_id = ?", v)
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
