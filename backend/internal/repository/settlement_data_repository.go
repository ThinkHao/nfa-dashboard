package repository

import (
	"context"
	"crypto/sha1"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"nfa-dashboard/internal/model"
	"sort"
	"strings"
	"sync"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type SettlementDataRepository interface {
	ListSettlementCustomer(ctx context.Context, filter map[string]interface{}, limit, offset int) ([]model.SettlementCustomer, int64, error)
	ListSettlementCustomerMonthly(ctx context.Context, filter map[string]interface{}, limit, offset int) ([]model.SettlementCustomerMonthly, int64, error)
	ListDistinctOwnerUserIDs(ctx context.Context, filter map[string]interface{}) ([]uint64, error)
	RebuildSettlementCustomerMonthly(start, end time.Time) (int64, error)
	UpdateRecalculated(region, cp, school string, start, end time.Time) (int64, error)
	CountSchoolSettlementRows(region, cp, school string, start, end time.Time) (int64, error)
	BackfillFromSchoolSettlement(region, cp, school string, start, end time.Time, markRecalc bool, progress func(processed int64, stageMetrics map[string]int64)) (int64, error)
}

type settlementDataRepository struct{}

var (
	slotTableOnce      sync.Once
	slotTableSupported bool
)

func NewSettlementDataRepository() SettlementDataRepository { return &settlementDataRepository{} }

func isSlotTableSupported() bool {
	slotTableOnce.Do(func() {
		slotTableSupported = model.DB.Migrator().HasTable("settlement_customer_v") &&
			model.DB.Migrator().HasTable("settlement_customer_monthly_v") &&
			model.DB.Migrator().HasTable("settlement_month_slot_pointer")
	})
	return slotTableSupported
}

func withActiveSlot(qb *gorm.DB, alias string) *gorm.DB {
	return qb.Joins(
		"JOIN settlement_month_slot_pointer p ON p.service_month = " + alias + ".service_month AND p.active_slot = " + alias + ".slot",
	)
}

func resolveScopeHash(region, cp, school string, start, end time.Time) string {
	payload := fmt.Sprintf("%s|%s|%s|%s|%s", region, cp, school, start.Format("2006-01-02"), end.Format("2006-01-02"))
	sum := sha1.Sum([]byte(payload))
	return fmt.Sprintf("%x", sum)
}

func normalizeDayBounds(start, end time.Time) (*time.Time, *time.Time) {
	var startBound *time.Time
	var endExclusive *time.Time
	if !start.IsZero() {
		s := time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, start.Location())
		startBound = &s
	}
	if !end.IsZero() {
		e := time.Date(end.Year(), end.Month(), end.Day(), 0, 0, 0, 0, end.Location()).AddDate(0, 0, 1)
		endExclusive = &e
	}
	return startBound, endExclusive
}

func normalizeSettlementResultUnitBase(base int) int {
	if base == 1000 || base == 1024 {
		return base
	}
	return 1024
}

func loadSettlementResultUnitBaseFromSystemSettings() int {
	var cfg model.SystemSettings
	err := model.DB.Select("settlement_result_unit_base").First(&cfg).Error
	if err != nil {
		return 1024
	}
	return normalizeSettlementResultUnitBase(cfg.SettlementResultUnitBase)
}

func settlementValueToGbps(settlementValue float64, unitBase int) float64 {
	normalizedBase := normalizeSettlementResultUnitBase(unitBase)
	bitsPerSecond := settlementValue * 8.0 / 60.0
	return bitsPerSecond / float64(normalizedBase*normalizedBase*normalizedBase)
}

func buildChunkRanges(total, chunkSize int) [][2]int {
	if total <= 0 {
		return [][2]int{}
	}
	if chunkSize <= 0 {
		chunkSize = total
	}
	ranges := make([][2]int, 0, (total+chunkSize-1)/chunkSize)
	for start := 0; start < total; start += chunkSize {
		end := start + chunkSize
		if end > total {
			end = total
		}
		ranges = append(ranges, [2]int{start, end})
	}
	return ranges
}

func settlementCustomerKey(region, cp, school string, serviceDate time.Time) string {
	return fmt.Sprintf("%s|%s|%s|%s", region, cp, school, serviceDate.Format("2006-01-02"))
}

func buildExistingSettlementMap(rows []model.SettlementCustomer) map[string]uint64 {
	m := make(map[string]uint64, len(rows))
	for i := range rows {
		if rows[i].ServiceDate == nil {
			continue
		}
		k := settlementCustomerKey(rows[i].Region, rows[i].CP, rows[i].SchoolName, *rows[i].ServiceDate)
		m[k] = rows[i].ID
	}
	return m
}

func applySettlementCustomerFilters(qb *gorm.DB, filter map[string]interface{}, alias string) *gorm.DB {
	if strings.TrimSpace(alias) == "" {
		alias = "settlement_customer"
	}
	col := func(name string) string {
		if alias == "settlement_customer" {
			return name
		}
		return alias + "." + name
	}
	if v, ok := filter["region"]; ok && v != "" {
		qb = qb.Where(col("region")+" = ?", v)
	}
	if v, ok := filter["src_region"]; ok && v != "" {
		qb = qb.Where(col("src_region")+" = ?", v)
	}
	if v, ok := filter["cp"]; ok && v != "" {
		qb = qb.Where(col("cp")+" = ?", v)
	}
	if v, ok := filter["school_name"]; ok && v != "" {
		qb = qb.Where(col("school_name")+" LIKE ?", "%"+v.(string)+"%")
	}
	if v, ok := filter["start_service_date"]; ok && v != nil {
		if t, ok2 := v.(time.Time); ok2 {
			dayStart := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
			qb = qb.Where(col("service_date")+" >= ?", dayStart)
		} else {
			qb = qb.Where(col("service_date")+" >= ?", v)
		}
	}
	if v, ok := filter["end_service_date"]; ok && v != nil {
		if t, ok2 := v.(time.Time); ok2 {
			dayEndExclusive := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location()).AddDate(0, 0, 1)
			qb = qb.Where(col("service_date")+" < ?", dayEndExclusive)
		} else {
			qb = qb.Where(col("service_date")+" <= ?", v)
		}
	}

	// 费用归属业务对象过滤（客户费/线路费/节点通用费 任一匹配）
	if v, ok := filter["owner_entity_id"]; ok && v != nil {
		qb = qb.Where("("+col("customer_fee_owner_id")+" = ? OR "+col("network_line_fee_owner_id")+" = ? OR "+col("node_deduction_fee_owner_id")+" = ?)", v, v, v)
	}
	// 渠道归属用户过滤（统一为用户ID后，需在四个归属字段中任一匹配）
	if v, ok := filter["channel_owner_user_id"]; ok && v != nil {
		qb = qb.Where("("+col("customer_fee_owner_id")+" = ? OR "+col("network_line_fee_owner_id")+" = ? OR "+col("node_deduction_fee_owner_id")+" = ? OR "+col("channel_owner_user_id")+" = ?)", v, v, v, v)
	}
	return applyRateFilterRulesIfEnabled(qb, alias)
}

func shouldApplySettlementFilterRules() bool {
	var cfg model.SystemSettings
	if err := model.DB.Select("hide_non_settlement_schools_in_traffic").First(&cfg).Error; err != nil {
		return false
	}
	return cfg.HideNonSettlementSchoolsInTraffic
}

func applyRateFilterRulesIfEnabled(qb *gorm.DB, alias string) *gorm.DB {
	if !shouldApplySettlementFilterRules() {
		return qb
	}
	return excludeFilteredCustomerRates(qb, alias)
}

func extractServiceMonthRange(filter map[string]interface{}) (string, string) {
	startMonth := ""
	endMonth := ""
	if v, ok := filter["start_service_date"]; ok && v != nil {
		if ym, ok2 := toYearMonth(v); ok2 {
			startMonth = ym
		}
	}
	if v, ok := filter["end_service_date"]; ok && v != nil {
		if ym, ok2 := toYearMonth(v); ok2 {
			endMonth = ym
		}
	}
	return startMonth, endMonth
}

func applyServiceMonthRangeToDailyQB(qb *gorm.DB, filter map[string]interface{}) *gorm.DB {
	startMonth, endMonth := extractServiceMonthRange(filter)
	if startMonth != "" {
		qb = qb.Where("DATE_FORMAT(service_date, '%Y-%m') >= ?", startMonth)
	}
	if endMonth != "" {
		qb = qb.Where("DATE_FORMAT(service_date, '%Y-%m') <= ?", endMonth)
	}
	return qb
}

func (r *settlementDataRepository) ListSettlementCustomer(ctx context.Context, filter map[string]interface{}, limit, offset int) ([]model.SettlementCustomer, int64, error) {
	qb := model.DB.WithContext(ctx).Model(&model.SettlementCustomer{})
	if isSlotTableSupported() {
		qb = withActiveSlot(model.DB.WithContext(ctx).Table("settlement_customer_v scv"), "scv")
	}
	alias := "settlement_customer"
	if isSlotTableSupported() {
		alias = "scv"
	}
	qb = applySettlementCustomerFilters(qb, filter, alias)

	var total int64
	if err := qb.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var rows []model.SettlementCustomer
	if limit <= 0 {
		limit = 10
	}
	if offset < 0 {
		offset = 0
	}
	if err := qb.Order("service_date DESC, settlement_time DESC").Limit(limit).Offset(offset).Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}

func (r *settlementDataRepository) ListDistinctOwnerUserIDs(ctx context.Context, filter map[string]interface{}) ([]uint64, error) {
	qb := model.DB.WithContext(ctx).Model(&model.SettlementCustomer{})
	alias := "settlement_customer"
	if isSlotTableSupported() {
		qb = withActiveSlot(model.DB.WithContext(ctx).Table("settlement_customer_v scv"), "scv")
		alias = "scv"
	}
	qb = applySettlementCustomerFilters(qb, filter, alias)

	ownerExpr := fmt.Sprintf(`CASE owner_slots.n
		WHEN 1 THEN %s.customer_fee_owner_id
		WHEN 2 THEN %s.network_line_fee_owner_id
		WHEN 3 THEN %s.node_deduction_fee_owner_id
		ELSE %s.channel_owner_user_id
	END`, alias, alias, alias, alias)
	var rows []struct {
		OwnerID uint64 `gorm:"column:owner_id"`
	}
	if err := qb.
		Joins("JOIN (SELECT 1 AS n UNION ALL SELECT 2 UNION ALL SELECT 3 UNION ALL SELECT 4) owner_slots ON 1=1").
		Select("DISTINCT " + ownerExpr + " AS owner_id").
		Where("(" + ownerExpr + ") IS NOT NULL AND (" + ownerExpr + ") > 0").
		Order("owner_id ASC").
		Scan(&rows).Error; err != nil {
		return nil, err
	}
	ids := make([]uint64, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, row.OwnerID)
	}
	return ids, nil
}

func (r *settlementDataRepository) listSettlementCustomerMonthlyFromDaily(ctx context.Context, filter map[string]interface{}, limit, offset int) ([]model.SettlementCustomerMonthly, int64, error) {
	base := model.DB.WithContext(ctx).Model(&model.SettlementCustomer{})
	if isSlotTableSupported() {
		base = withActiveSlot(model.DB.WithContext(ctx).Table("settlement_customer_v scv"), "scv")
	}
	alias := "settlement_customer"
	if isSlotTableSupported() {
		alias = "scv"
	}
	base = applySettlementCustomerFilters(base, filter, alias).Where(alias + ".service_date IS NOT NULL")
	base = applyServiceMonthRangeToDailyQB(base, filter)

	// 统计分组后的总条数
	countSubQuery := base.
		Select("region, cp, school_name, DATE_FORMAT(service_date, '%Y-%m') AS service_month").
		Group("region, cp, school_name, DATE_FORMAT(service_date, '%Y-%m')")
	var total int64
	if err := model.DB.WithContext(ctx).Table("(?) AS grouped", countSubQuery).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if limit <= 0 {
		limit = 10
	}
	if offset < 0 {
		offset = 0
	}

	selectSQL := `
		region,
		CASE WHEN COUNT(DISTINCT src_region) = 1 THEN MAX(src_region) ELSE NULL END AS src_region,
		cp,
		school_name,
		DATE_FORMAT(service_date, '%Y-%m') AS service_date,
		ROUND(AVG(settlement_value), 6) AS settlement_value,
		ROUND(AVG(stock_ratio), 6) AS stock_ratio,
		ROUND(AVG(increment_ratio), 6) AS increment_ratio,
		ROUND(AVG(daily_increment_value), 6) AS daily_increment_value,
		ROUND(AVG(customer_fee), 6) AS customer_fee,
		ROUND(AVG(network_line_fee), 6) AS network_line_fee,
		ROUND(AVG(node_deduction_fee), 6) AS node_deduction_fee,
		ROUND(AVG(channel_rate), 6) AS channel_rate,
		CASE WHEN SUM(CASE WHEN recalculated = 1 THEN 1 ELSE 0 END) > 0 THEN 1 ELSE 0 END AS recalculated,
		DATE_FORMAT(MAX(last_recalc_time), '%Y-%m-%d %H:%i:%s') AS last_recalc_time,
		CASE WHEN COUNT(DISTINCT customer_fee_owner_id) = 1 THEN MAX(customer_fee_owner_id) ELSE NULL END AS customer_fee_owner_id,
		CASE WHEN COUNT(DISTINCT network_line_fee_owner_id) = 1 THEN MAX(network_line_fee_owner_id) ELSE NULL END AS network_line_fee_owner_id,
		CASE WHEN COUNT(DISTINCT node_deduction_fee_owner_id) = 1 THEN MAX(node_deduction_fee_owner_id) ELSE NULL END AS node_deduction_fee_owner_id,
		CASE WHEN COUNT(DISTINCT channel_owner_user_id) = 1 THEN MAX(channel_owner_user_id) ELSE NULL END AS channel_owner_user_id,
		ROUND(SUM(customer_bill), 2) AS customer_bill,
		ROUND(SUM(network_line_bill), 2) AS network_line_bill,
		ROUND(SUM(node_deduction_bill), 2) AS node_deduction_bill,
		ROUND(SUM(channel_bill), 2) AS channel_bill
	`
	args := []interface{}{}
	if v, ok := filter["channel_owner_user_id"]; ok && v != nil {
		selectSQL = `
			region,
			CASE WHEN COUNT(DISTINCT src_region) = 1 THEN MAX(src_region) ELSE NULL END AS src_region,
			cp,
			school_name,
			DATE_FORMAT(service_date, '%Y-%m') AS service_date,
			ROUND(AVG(settlement_value), 6) AS settlement_value,
			ROUND(AVG(stock_ratio), 6) AS stock_ratio,
			ROUND(AVG(increment_ratio), 6) AS increment_ratio,
			ROUND(AVG(daily_increment_value), 6) AS daily_increment_value,
			ROUND(AVG(customer_fee), 6) AS customer_fee,
			ROUND(AVG(network_line_fee), 6) AS network_line_fee,
			ROUND(AVG(node_deduction_fee), 6) AS node_deduction_fee,
			ROUND(AVG(channel_rate), 6) AS channel_rate,
			CASE WHEN SUM(CASE WHEN recalculated = 1 THEN 1 ELSE 0 END) > 0 THEN 1 ELSE 0 END AS recalculated,
			DATE_FORMAT(MAX(last_recalc_time), '%Y-%m-%d %H:%i:%s') AS last_recalc_time,
			CASE WHEN SUM(CASE WHEN customer_fee_owner_id = ? THEN 1 ELSE 0 END) > 0 THEN ? ELSE NULL END AS customer_fee_owner_id,
			CASE WHEN SUM(CASE WHEN network_line_fee_owner_id = ? THEN 1 ELSE 0 END) > 0 THEN ? ELSE NULL END AS network_line_fee_owner_id,
			CASE WHEN SUM(CASE WHEN node_deduction_fee_owner_id = ? THEN 1 ELSE 0 END) > 0 THEN ? ELSE NULL END AS node_deduction_fee_owner_id,
			CASE WHEN SUM(CASE WHEN channel_owner_user_id = ? THEN 1 ELSE 0 END) > 0 THEN ? ELSE NULL END AS channel_owner_user_id,
			CASE
				WHEN SUM(CASE WHEN customer_fee_owner_id = ? THEN 1 ELSE 0 END) > 0
				THEN ROUND(SUM(CASE WHEN customer_fee_owner_id = ? THEN COALESCE(customer_bill, 0) ELSE 0 END), 2)
				ELSE NULL
			END AS customer_bill,
			CASE
				WHEN SUM(CASE WHEN network_line_fee_owner_id = ? THEN 1 ELSE 0 END) > 0
				THEN ROUND(SUM(CASE WHEN network_line_fee_owner_id = ? THEN COALESCE(network_line_bill, 0) ELSE 0 END), 2)
				ELSE NULL
			END AS network_line_bill,
			CASE
				WHEN SUM(CASE WHEN node_deduction_fee_owner_id = ? THEN 1 ELSE 0 END) > 0
				THEN ROUND(SUM(CASE WHEN node_deduction_fee_owner_id = ? THEN COALESCE(node_deduction_bill, 0) ELSE 0 END), 2)
				ELSE NULL
			END AS node_deduction_bill,
			CASE
				WHEN SUM(CASE WHEN channel_owner_user_id = ? THEN 1 ELSE 0 END) > 0
				THEN ROUND(SUM(CASE WHEN channel_owner_user_id = ? THEN COALESCE(channel_bill, 0) ELSE 0 END), 2)
				ELSE NULL
			END AS channel_bill
		`
		args = []interface{}{
			v, v, v, v, v, v, v, v,
			v, v, v, v, v, v, v, v,
		}
	}

	var rows []model.SettlementCustomerMonthly
	query := base.Select(selectSQL, args...).
		Group("region, cp, school_name, DATE_FORMAT(service_date, '%Y-%m')").
		Order("service_date DESC, region ASC, cp ASC, school_name ASC").
		Limit(limit).
		Offset(offset)
	if err := query.Scan(&rows).Error; err != nil {
		return nil, 0, err
	}
	for i := range rows {
		rows[i].DataSource = "realtime"
	}
	return rows, total, nil
}

func (r *settlementDataRepository) ListSettlementCustomerMonthly(ctx context.Context, filter map[string]interface{}, limit, offset int) ([]model.SettlementCustomerMonthly, int64, error) {
	// 带用户过滤时，需要按用户掩码金额，必须从日表实时聚合保证精确
	if shouldUseDailyMonthlyAggregation(filter) {
		return r.listSettlementCustomerMonthlyFromDaily(ctx, filter, limit, offset)
	}

	if limit <= 0 {
		limit = 10
	}
	if offset < 0 {
		offset = 0
	}

	snapshotTable := "settlement_customer_monthly"
	snapshotAlias := "settlement_customer_monthly"
	baseQB := model.DB.WithContext(ctx).Table(snapshotTable)
	if isSlotTableSupported() {
		snapshotTable = "settlement_customer_monthly_v scmv"
		snapshotAlias = "scmv"
		baseQB = withActiveSlot(model.DB.WithContext(ctx).Table(snapshotTable), "scmv")
	}
	qb := applyMonthlySnapshotFilters(baseQB, filter, snapshotAlias)

	var total int64
	if err := qb.Count(&total).Error; err != nil {
		// 月表不存在时，回退实时聚合，避免功能中断
		return r.listSettlementCustomerMonthlyFromDaily(ctx, filter, limit, offset)
	}
	if stale, ferr := isMonthlySnapshotStale(ctx, filter); ferr != nil || stale {
		return r.listSettlementCustomerMonthlyFromDaily(ctx, filter, limit, offset)
	}

	var rows []model.SettlementCustomerMonthly
	err := qb.Select(`
			region,
			src_region,
			cp,
			school_name,
			service_month AS service_date,
			settlement_value,
			stock_ratio,
			increment_ratio,
			daily_increment_value,
			customer_fee,
			customer_bill,
			customer_fee_owner_id,
			network_line_fee,
			network_line_bill,
			network_line_fee_owner_id,
			node_deduction_fee,
			node_deduction_bill,
			node_deduction_fee_owner_id,
			channel_rate,
			channel_bill,
			channel_owner_user_id,
			recalculated,
			DATE_FORMAT(last_recalc_time, '%Y-%m-%d %H:%i:%s') AS last_recalc_time,
			'snapshot' AS data_source
		`).
		Order("service_month DESC, region ASC, cp ASC, school_name ASC").
		Limit(limit).
		Offset(offset).
		Scan(&rows).Error
	if err != nil {
		return r.listSettlementCustomerMonthlyFromDaily(ctx, filter, limit, offset)
	}
	return rows, total, nil
}

func applyMonthlySnapshotFilters(qb *gorm.DB, filter map[string]interface{}, alias string) *gorm.DB {
	if strings.TrimSpace(alias) == "" {
		alias = "settlement_customer_monthly"
	}
	col := func(name string) string { return alias + "." + name }
	if v, ok := filter["region"]; ok && v != "" {
		qb = qb.Where(col("region")+" = ?", v)
	}
	if v, ok := filter["src_region"]; ok && v != "" {
		qb = qb.Where(col("src_region")+" = ?", v)
	}
	if v, ok := filter["cp"]; ok && v != "" {
		qb = qb.Where(col("cp")+" = ?", v)
	}
	if v, ok := filter["school_name"]; ok && v != "" {
		qb = qb.Where(col("school_name")+" LIKE ?", "%"+v.(string)+"%")
	}
	if v, ok := filter["start_service_date"]; ok && v != nil {
		if m, ok2 := toYearMonth(v); ok2 {
			qb = qb.Where(col("service_month")+" >= ?", m)
		}
	}
	if v, ok := filter["end_service_date"]; ok && v != nil {
		if m, ok2 := toYearMonth(v); ok2 {
			qb = qb.Where(col("service_month")+" <= ?", m)
		}
	}
	return applyRateFilterRulesIfEnabled(qb, alias)
}

func toYearMonth(v interface{}) (string, bool) {
	switch t := v.(type) {
	case time.Time:
		return t.Format("2006-01"), true
	case string:
		s := strings.TrimSpace(strings.Split(t, " ")[0])
		if len(s) >= 7 {
			return s[:7], true
		}
	}
	return "", false
}

func isMonthlySnapshotStale(ctx context.Context, filter map[string]interface{}) (bool, error) {
	dailyAlias := "settlement_customer"
	dailyQB := model.DB.WithContext(ctx).Model(&model.SettlementCustomer{}).Where("service_date IS NOT NULL")
	if isSlotTableSupported() {
		dailyAlias = "scv"
		dailyQB = withActiveSlot(model.DB.WithContext(ctx).Table("settlement_customer_v scv"), "scv").Where("scv.service_date IS NOT NULL")
	}
	col := func(name string) string {
		if dailyAlias == "settlement_customer" {
			return name
		}
		return dailyAlias + "." + name
	}
	if v, ok := filter["region"]; ok && v != "" {
		dailyQB = dailyQB.Where(col("region")+" = ?", v)
	}
	if v, ok := filter["src_region"]; ok && v != "" {
		dailyQB = dailyQB.Where(col("src_region")+" = ?", v)
	}
	if v, ok := filter["cp"]; ok && v != "" {
		dailyQB = dailyQB.Where(col("cp")+" = ?", v)
	}
	if v, ok := filter["school_name"]; ok && v != "" {
		dailyQB = dailyQB.Where(col("school_name")+" LIKE ?", "%"+v.(string)+"%")
	}
	if v, ok := filter["start_service_date"]; ok && v != nil {
		if t, ok2 := v.(time.Time); ok2 {
			dailyQB = dailyQB.Where("DATE("+col("service_date")+") >= ?", t.Format("2006-01-02"))
		}
	}
	if v, ok := filter["end_service_date"]; ok && v != nil {
		if t, ok2 := v.(time.Time); ok2 {
			dailyQB = dailyQB.Where("DATE("+col("service_date")+") <= ?", t.Format("2006-01-02"))
		}
	}
	dailyQB = applyServiceMonthRangeToDailyQB(dailyQB, filter)
	var maxDailyUpdatedAt *time.Time
	if err := dailyQB.Select("MAX(updated_at)").Scan(&maxDailyUpdatedAt).Error; err != nil {
		return true, err
	}
	if maxDailyUpdatedAt == nil {
		return false, nil
	}

	snapQB := model.DB.WithContext(ctx).Table("settlement_customer_monthly")
	if isSlotTableSupported() {
		snapQB = withActiveSlot(model.DB.WithContext(ctx).Table("settlement_customer_monthly_v scmv"), "scmv")
	}
	snapQB = applyMonthlySnapshotFilters(snapQB, filter, func() string {
		if isSlotTableSupported() {
			return "scmv"
		}
		return "settlement_customer_monthly"
	}())
	var maxSnapUpdatedAt *time.Time
	if err := snapQB.Select("MAX(updated_at)").Scan(&maxSnapUpdatedAt).Error; err != nil {
		return true, err
	}
	if maxSnapUpdatedAt == nil {
		return true, nil
	}
	return maxSnapUpdatedAt.Before(*maxDailyUpdatedAt), nil
}

func shouldUseDailyMonthlyAggregation(filter map[string]interface{}) bool {
	v, ok := filter["channel_owner_user_id"]
	if !ok || v == nil {
		return false
	}
	switch t := v.(type) {
	case uint64:
		return t > 0
	case uint32:
		return t > 0
	case uint:
		return t > 0
	case int64:
		return t > 0
	case int32:
		return t > 0
	case int:
		return t > 0
	case string:
		s := strings.TrimSpace(t)
		return s != "" && s != "0"
	default:
		return true
	}
}

func (r *settlementDataRepository) RebuildSettlementCustomerMonthly(start, end time.Time) (int64, error) {
	if isSlotTableSupported() {
		months := buildMonthList(start, end)
		if len(months) == 0 {
			var pointerMonths []string
			if err := model.DB.Table("settlement_month_slot_pointer").Pluck("service_month", &pointerMonths).Error; err != nil {
				return 0, err
			}
			months = pointerMonths
		}
		var total int64
		for _, month := range months {
			tx := model.DB.Begin()
			if tx.Error != nil {
				return total, tx.Error
			}
			activeSlot, err := activeSlotForMonth(tx, month, 0)
			if err != nil {
				tx.Rollback()
				return total, err
			}
			if err := tx.Exec("DELETE FROM settlement_customer_monthly_v WHERE service_month = ? AND slot = ?", month, activeSlot).Error; err != nil {
				tx.Rollback()
				return total, err
			}
			res := tx.Exec(`
				INSERT INTO settlement_customer_monthly_v (
					region, src_region, cp, school_name, service_month, slot,
					settlement_value, stock_ratio, increment_ratio, daily_increment_value,
					customer_fee, customer_bill, customer_fee_owner_id,
					network_line_fee, network_line_bill, network_line_fee_owner_id,
					node_deduction_fee, node_deduction_bill, node_deduction_fee_owner_id,
					channel_rate, channel_bill, channel_owner_user_id,
					recalculated, last_recalc_time, created_at, updated_at
				)
				SELECT
					region, CASE WHEN COUNT(DISTINCT src_region) = 1 THEN MAX(src_region) ELSE NULL END, cp, school_name, service_month, ?,
					ROUND(AVG(settlement_value), 6),
					ROUND(AVG(stock_ratio), 6),
					ROUND(AVG(increment_ratio), 6),
					ROUND(AVG(daily_increment_value), 6),
					ROUND(AVG(customer_fee), 6),
					ROUND(SUM(customer_bill), 2),
					CASE WHEN COUNT(DISTINCT customer_fee_owner_id) = 1 THEN MAX(customer_fee_owner_id) ELSE NULL END,
					ROUND(AVG(network_line_fee), 6),
					ROUND(SUM(network_line_bill), 2),
					CASE WHEN COUNT(DISTINCT network_line_fee_owner_id) = 1 THEN MAX(network_line_fee_owner_id) ELSE NULL END,
					ROUND(AVG(node_deduction_fee), 6),
					ROUND(SUM(node_deduction_bill), 2),
					CASE WHEN COUNT(DISTINCT node_deduction_fee_owner_id) = 1 THEN MAX(node_deduction_fee_owner_id) ELSE NULL END,
					ROUND(AVG(channel_rate), 6),
					ROUND(SUM(channel_bill), 2),
					CASE WHEN COUNT(DISTINCT channel_owner_user_id) = 1 THEN MAX(channel_owner_user_id) ELSE NULL END,
					CASE WHEN SUM(CASE WHEN recalculated = 1 THEN 1 ELSE 0 END) > 0 THEN 1 ELSE 0 END,
					MAX(last_recalc_time),
					NOW(), NOW()
				FROM settlement_customer_v
				WHERE service_month = ? AND slot = ?
				GROUP BY region, cp, school_name, service_month
			`, activeSlot, month, activeSlot)
			if res.Error != nil {
				tx.Rollback()
				return total, res.Error
			}
			total += res.RowsAffected
			if err := tx.Commit().Error; err != nil {
				return total, err
			}
		}
		return total, nil
	}

	qb := model.DB.Model(&model.SettlementCustomer{}).Where("service_date IS NOT NULL")
	startBound, endExclusive := normalizeDayBounds(start, end)
	if !start.IsZero() {
		qb = qb.Where("service_date >= ?", *startBound)
	}
	if !end.IsZero() {
		qb = qb.Where("service_date < ?", *endExclusive)
	}

	base := qb.Select(`
		region,
		CASE WHEN COUNT(DISTINCT src_region) = 1 THEN MAX(src_region) ELSE NULL END AS src_region,
		cp,
		school_name,
		DATE_FORMAT(service_date, '%Y-%m') AS service_month,
		ROUND(AVG(settlement_value), 6) AS settlement_value,
		ROUND(AVG(stock_ratio), 6) AS stock_ratio,
		ROUND(AVG(increment_ratio), 6) AS increment_ratio,
		ROUND(AVG(daily_increment_value), 6) AS daily_increment_value,
		ROUND(AVG(customer_fee), 6) AS customer_fee,
		ROUND(SUM(customer_bill), 2) AS customer_bill,
		CASE WHEN COUNT(DISTINCT customer_fee_owner_id) = 1 THEN MAX(customer_fee_owner_id) ELSE NULL END AS customer_fee_owner_id,
		ROUND(AVG(network_line_fee), 6) AS network_line_fee,
		ROUND(SUM(network_line_bill), 2) AS network_line_bill,
		CASE WHEN COUNT(DISTINCT network_line_fee_owner_id) = 1 THEN MAX(network_line_fee_owner_id) ELSE NULL END AS network_line_fee_owner_id,
		ROUND(AVG(node_deduction_fee), 6) AS node_deduction_fee,
		ROUND(SUM(node_deduction_bill), 2) AS node_deduction_bill,
		CASE WHEN COUNT(DISTINCT node_deduction_fee_owner_id) = 1 THEN MAX(node_deduction_fee_owner_id) ELSE NULL END AS node_deduction_fee_owner_id,
		ROUND(AVG(channel_rate), 6) AS channel_rate,
		ROUND(SUM(channel_bill), 2) AS channel_bill,
		CASE WHEN COUNT(DISTINCT channel_owner_user_id) = 1 THEN MAX(channel_owner_user_id) ELSE NULL END AS channel_owner_user_id,
		CASE WHEN SUM(CASE WHEN recalculated = 1 THEN 1 ELSE 0 END) > 0 THEN 1 ELSE 0 END AS recalculated,
		MAX(last_recalc_time) AS last_recalc_time
	`).Group("region, cp, school_name, DATE_FORMAT(service_date, '%Y-%m')")

	sql := `
		INSERT INTO settlement_customer_monthly (
			region, src_region, cp, school_name, service_month,
			settlement_value, stock_ratio, increment_ratio, daily_increment_value,
			customer_fee, customer_bill, customer_fee_owner_id,
			network_line_fee, network_line_bill, network_line_fee_owner_id,
			node_deduction_fee, node_deduction_bill, node_deduction_fee_owner_id,
			channel_rate, channel_bill, channel_owner_user_id,
			recalculated, last_recalc_time, created_at, updated_at
		)
		SELECT
			region, src_region, cp, school_name, service_month,
			settlement_value, stock_ratio, increment_ratio, daily_increment_value,
			customer_fee, customer_bill, customer_fee_owner_id,
			network_line_fee, network_line_bill, network_line_fee_owner_id,
			node_deduction_fee, node_deduction_bill, node_deduction_fee_owner_id,
			channel_rate, channel_bill, channel_owner_user_id,
			recalculated, last_recalc_time, NOW(), NOW()
		FROM ( ? ) AS agg
		ON DUPLICATE KEY UPDATE
			src_region = COALESCE(settlement_customer_monthly.src_region, VALUES(src_region)),
			settlement_value = VALUES(settlement_value),
			stock_ratio = VALUES(stock_ratio),
			increment_ratio = VALUES(increment_ratio),
			daily_increment_value = VALUES(daily_increment_value),
			customer_fee = VALUES(customer_fee),
			customer_bill = VALUES(customer_bill),
			customer_fee_owner_id = VALUES(customer_fee_owner_id),
			network_line_fee = VALUES(network_line_fee),
			network_line_bill = VALUES(network_line_bill),
			network_line_fee_owner_id = VALUES(network_line_fee_owner_id),
			node_deduction_fee = VALUES(node_deduction_fee),
			node_deduction_bill = VALUES(node_deduction_bill),
			node_deduction_fee_owner_id = VALUES(node_deduction_fee_owner_id),
			channel_rate = VALUES(channel_rate),
			channel_bill = VALUES(channel_bill),
			channel_owner_user_id = VALUES(channel_owner_user_id),
			recalculated = VALUES(recalculated),
			last_recalc_time = VALUES(last_recalc_time),
			updated_at = NOW()
	`

	res := model.DB.Exec(sql, base)
	return res.RowsAffected, res.Error
}

func (r *settlementDataRepository) UpdateRecalculated(region, cp, school string, start, end time.Time) (int64, error) {
	// 使用 DATE(service_date) BETWEEN 起止日期（YYYY-MM-DD），确保边界包含当天整日
	qb := model.DB.Model(&model.SettlementCustomer{}).Where("service_date IS NOT NULL")
	if !start.IsZero() && !end.IsZero() {
		qb = qb.Where("DATE(service_date) BETWEEN ? AND ?", start.Format("2006-01-02"), end.Format("2006-01-02"))
	} else if !start.IsZero() {
		qb = qb.Where("DATE(service_date) >= ?", start.Format("2006-01-02"))
	} else if !end.IsZero() {
		qb = qb.Where("DATE(service_date) <= ?", end.Format("2006-01-02"))
	}
	if region != "" {
		qb = qb.Where("region = ?", region)
	}
	if cp != "" {
		qb = qb.Where("cp = ?", cp)
	}
	if school != "" {
		qb = qb.Where("school_name = ?", school)
	}
	now := time.Now()
	res := qb.Updates(map[string]interface{}{
		"recalculated":     true,
		"last_recalc_time": now,
	})
	return res.RowsAffected, res.Error
}

func calcServiceYearIndex(startAt *time.Time, serviceDate time.Time) int {
	if startAt == nil {
		return 0
	}
	start := time.Date(startAt.Year(), startAt.Month(), startAt.Day(), 0, 0, 0, 0, startAt.Location())
	cur := time.Date(serviceDate.Year(), serviceDate.Month(), serviceDate.Day(), 0, 0, 0, 0, serviceDate.Location())
	yearIdx := 1
	if cur.Before(start) {
		return yearIdx
	}
	for tmp := start; !cur.Before(tmp.AddDate(1, 0, 0)); tmp = tmp.AddDate(1, 0, 0) {
		yearIdx++
	}
	return yearIdx
}

func findMatchedDiscountRule(tx *gorm.DB, rc *model.RateCustomer) (*model.RateDiscountRule, error) {
	if rc == nil {
		return nil, nil
	}
	var matched model.RateDiscountRule
	if rc.SchoolName != nil && *rc.SchoolName != "" {
		if err := tx.Where("enabled = ? AND scope_type = ? AND scope_key = ?", true, "school", *rc.SchoolName).
			Order("priority ASC, updated_at DESC").First(&matched).Error; err == nil {
			return &matched, nil
		}
	}
	if err := tx.Where("enabled = ? AND scope_type = ? AND scope_key = ?", true, "cp", rc.CP).
		Order("priority ASC, updated_at DESC").First(&matched).Error; err == nil {
		return &matched, nil
	}
	if err := tx.Where("enabled = ? AND scope_type = ? AND scope_key = ?", true, "region", rc.Region).
		Order("priority ASC, updated_at DESC").First(&matched).Error; err == nil {
		return &matched, nil
	}
	if err := tx.Where("enabled = ? AND scope_type = ?", true, "global").
		Order("priority ASC, updated_at DESC").First(&matched).Error; err == nil {
		return &matched, nil
	}
	return nil, nil
}

func findDiscountRatioByYear(tx *gorm.DB, ruleID uint64, yearIdx int) float64 {
	if ruleID == 0 || yearIdx <= 0 {
		return 1
	}
	var items []model.RateDiscountRuleItem
	if err := tx.Where("rule_id = ?", ruleID).Order("from_year ASC").Find(&items).Error; err != nil {
		return 1
	}
	for i := range items {
		it := &items[i]
		if yearIdx < it.FromYear {
			continue
		}
		if it.ToYear != nil && yearIdx > *it.ToYear {
			continue
		}
		if it.DiscountRate > 0 {
			return it.DiscountRate
		}
	}
	return 1
}

func discountRuleFieldSet(rule *model.RateDiscountRule) map[string]bool {
	set := map[string]bool{}
	if rule == nil {
		return set
	}
	var fields []string
	if len(rule.Fields) > 0 {
		if err := json.Unmarshal(rule.Fields, &fields); err == nil {
			for _, f := range fields {
				key := strings.TrimSpace(f)
				if key != "" {
					set[key] = true
				}
			}
		}
	}
	if len(set) == 0 {
		set["customer_fee"] = true
	}
	return set
}

func (r *settlementDataRepository) CountSchoolSettlementRows(region, cp, school string, start, end time.Time) (int64, error) {
	src := model.DB.Model(&model.SchoolSettlement{})
	startBound, endExclusive := normalizeDayBounds(start, end)
	if startBound != nil {
		src = src.Where("settlement_date >= ?", *startBound)
	}
	if endExclusive != nil {
		src = src.Where("settlement_date < ?", *endExclusive)
	}
	if region != "" {
		src = src.Where("region = ?", region)
	}
	if cp != "" {
		src = src.Where("cp = ?", cp)
	}
	if school != "" {
		src = src.Where("school_name = ?", school)
	}
	var cnt int64
	if err := src.Count(&cnt).Error; err != nil {
		return 0, err
	}
	return cnt, nil
}

// BackfillFromSchoolSettlement 按条件从 nfa_school_settlement 回填/覆盖 settlement_customer 的基础字段
// 说明：
// - 不做费用计算，仅复制 95 值与服务日期/时间等基础信息
// - 以 (region, cp, school_name, service_date) 作为匹配键，存在则更新，不存在则插入
// - markRecalc: 为 true 时表示“复算”，会设置 recalculated 与 last_recalc_time；为 false 表示“初算”，不设置上述标记
func (r *settlementDataRepository) BackfillFromSchoolSettlement(region, cp, school string, start, end time.Time, markRecalc bool, progress func(processed int64, stageMetrics map[string]int64)) (int64, error) {
	if isSlotTableSupported() {
		return r.backfillFromSchoolSettlementWithSlot(region, cp, school, start, end, markRecalc, progress)
	}

	var (
		affected  int64
		processed int64
		lastID    int64
	)
	const sourceChunkSize = 2000
	startBound, endExclusive := normalizeDayBounds(start, end)
	settlementResultUnitBase := loadSettlementResultUnitBaseFromSystemSettings()
	type rateLookup struct {
		rc    model.RateCustomer
		found bool
	}
	rateCache := map[string]rateLookup{}
	ruleCache := map[string]*model.RateDiscountRule{}
	ruleItemCache := map[uint64][]model.RateDiscountRuleItem{}

	for {
		tx := model.DB.Begin()
		if tx.Error != nil {
			return affected, tx.Error
		}
		now := time.Now()

		src := tx.Model(&model.SchoolSettlement{})
		if startBound != nil {
			src = src.Where("settlement_date >= ?", *startBound)
		}
		if endExclusive != nil {
			src = src.Where("settlement_date < ?", *endExclusive)
		}
		if region != "" {
			src = src.Where("region = ?", region)
		}
		if cp != "" {
			src = src.Where("cp = ?", cp)
		}
		if school != "" {
			src = src.Where("school_name = ?", school)
		}
		if lastID > 0 {
			src = src.Where("id > ?", lastID)
		}

		var chunkRows []model.SchoolSettlement
		if err := src.Order("id ASC").Limit(sourceChunkSize).Find(&chunkRows).Error; err != nil {
			tx.Rollback()
			return affected, err
		}
		if len(chunkRows) == 0 {
			tx.Rollback()
			break
		}

		upserts := make([]model.SettlementCustomer, 0, len(chunkRows))
		pendingRateIncrement := map[uint64]float64{}

		for _, it := range chunkRows {
			sd := it.SettlementDate
			rec := model.SettlementCustomer{
				Region:          it.Region,
				SrcRegion:       it.SrcRegion,
				CP:              it.CP,
				SchoolName:      it.SchoolName,
				SettlementValue: float64(it.SettlementValue),
				SettlementTime:  it.SettlementTime,
				ServiceDate:     &sd,
				// 初算不标记复算；复算才设置标记
				Recalculated: markRecalc,
				LastRecalcTime: func() *time.Time {
					if markRecalc {
						return &now
					}
					return nil
				}(),
			}

			// 尝试填充费率与归属（取 service_date 生效的最新一条）
			day := time.Date(sd.Year(), sd.Month(), sd.Day(), 0, 0, 0, 0, sd.Location())
			rateKey := settlementCustomerKey(it.Region, it.CP, it.SchoolName, day)
			rateHit, ok := rateCache[rateKey]
			if !ok {
				var rc model.RateCustomer
				rcq := tx.Model(&model.RateCustomer{}).
					Where("region = ? AND cp = ? AND school_name = ?", it.Region, it.CP, it.SchoolName)
				if !sd.IsZero() {
					rcq = rcq.Where("(start_at IS NULL OR start_at <= ?)", sd)
				}
				err := rcq.Order("start_at DESC, id DESC").First(&rc).Error
				if err != nil {
					if errors.Is(err, gorm.ErrRecordNotFound) {
						rateCache[rateKey] = rateLookup{found: false}
					} else {
						tx.Rollback()
						return affected, err
					}
				} else {
					rateCache[rateKey] = rateLookup{rc: rc, found: true}
				}
				rateHit = rateCache[rateKey]
			}

			if rateHit.found {
				rc := rateHit.rc
				// 费率
				if rc.CustomerFee != nil {
					rec.CustomerFee = rc.CustomerFee
				}
				if rc.NetworkLineFee != nil {
					rec.NetworkLineFee = rc.NetworkLineFee
				}
				// 一般费率映射到节点通用费
				if rc.GeneralFee != nil {
					rec.NodeDeductionFee = rc.GeneralFee
				}
				if rc.ChannelRate != nil {
					rec.ChannelRate = rc.ChannelRate
				}
				// 归属
				if rc.CustomerFeeOwnerID != nil {
					rec.CustomerFeeOwnerID = rc.CustomerFeeOwnerID
				}
				if rc.NetworkLineFeeOwnerID != nil {
					rec.NetworkLineFeeOwnerID = rc.NetworkLineFeeOwnerID
				}
				if rc.GeneralFeeOwnerID != nil {
					rec.NodeDeductionFeeOwnerID = rc.GeneralFeeOwnerID
				}
				if rc.ChannelOwnerUserID != nil {
					rec.ChannelOwnerUserID = rc.ChannelOwnerUserID
				}

				stockRatio := 1.0
				incrementRatio := 0.0
				if rc.StockRatio != nil {
					stockRatio = *rc.StockRatio
				}
				if rc.IncrementRatio != nil {
					incrementRatio = *rc.IncrementRatio
				}
				// 增量分段仅在“服务日期 >= 增量起算日期”时生效；
				// 在增量起算日前，按纯存量处理（100%/0%）。
				incrementNotStarted := false
				if rc.IncrementStartAt != nil {
					startInc := time.Date(rc.IncrementStartAt.Year(), rc.IncrementStartAt.Month(), rc.IncrementStartAt.Day(), 0, 0, 0, 0, rc.IncrementStartAt.Location())
					curDay := time.Date(sd.Year(), sd.Month(), sd.Day(), 0, 0, 0, 0, sd.Location())
					incrementNotStarted = curDay.Before(startInc)
				}
				if rc.IncrementStartAt == nil || incrementNotStarted {
					stockRatio = 1
					incrementRatio = 0
				}
				rec.StockRatio = &stockRatio
				rec.IncrementRatio = &incrementRatio

				incrementValue := rec.SettlementValue * incrementRatio
				incrementValueRounded := math.Round(incrementValue*1_000_000) / 1_000_000
				rec.DailyIncrementValue = &incrementValueRounded

				stockYearIdx := calcServiceYearIndex(rc.StartAt, sd)
				incrementYearIdx := calcServiceYearIndex(rc.IncrementStartAt, sd)
				rateSchoolName := ""
				if rc.SchoolName != nil {
					rateSchoolName = *rc.SchoolName
				}
				ruleKey := rateSchoolName + "|" + rc.CP + "|" + rc.Region
				matchedRule, hasRuleKey := ruleCache[ruleKey]
				if !hasRuleKey {
					matchedRule, _ = findMatchedDiscountRule(tx, &rc)
					ruleCache[ruleKey] = matchedRule
				}
				stockDiscountRatio := 1.0
				incrementDiscountRatio := 1.0
				if matchedRule != nil {
					items, okItems := ruleItemCache[matchedRule.ID]
					if !okItems {
						var loaded []model.RateDiscountRuleItem
						if err := tx.Where("rule_id = ?", matchedRule.ID).Order("from_year ASC").Find(&loaded).Error; err != nil {
							tx.Rollback()
							return affected, err
						}
						ruleItemCache[matchedRule.ID] = loaded
						items = loaded
					}
					stockDiscountRatio = 1
					incrementDiscountRatio = 1
					for i := range items {
						itm := items[i]
						if stockYearIdx >= itm.FromYear && (itm.ToYear == nil || stockYearIdx <= *itm.ToYear) && itm.DiscountRate > 0 {
							stockDiscountRatio = itm.DiscountRate
							break
						}
					}
					for i := range items {
						itm := items[i]
						if incrementYearIdx >= itm.FromYear && (itm.ToYear == nil || incrementYearIdx <= *itm.ToYear) && itm.DiscountRate > 0 {
							incrementDiscountRatio = itm.DiscountRate
							break
						}
					}
					did := matchedRule.ID
					rec.DiscountRuleID = &did
				}
				if stockYearIdx > 0 {
					yi := stockYearIdx
					rec.ServiceYearIndex = &yi
				}
				affectedFields := discountRuleFieldSet(matchedRule)
				blendByField := func(base *float64, field string) *float64 {
					if base == nil {
						return nil
					}
					if !affectedFields[field] {
						return base
					}
					v := (*base)*stockRatio*stockDiscountRatio + (*base)*incrementRatio*incrementDiscountRatio
					return &v
				}
				if v := blendByField(rec.CustomerFee, "customer_fee"); v != nil {
					rec.CustomerFee = v
				}
				if v := blendByField(rec.NetworkLineFee, "network_line_fee"); v != nil {
					rec.NetworkLineFee = v
				}
				if v := blendByField(rec.NodeDeductionFee, "general_fee"); v != nil {
					rec.NodeDeductionFee = v
				}
				if v := blendByField(rec.ChannelRate, "channel_rate"); v != nil {
					rec.ChannelRate = v
				}

				// 金额计算：费率单位 元/Gbps（客户金额使用折损后的费率），Gbps 进制与系统设置保持一致（1000/1024）。
				gbps := settlementValueToGbps(rec.SettlementValue, settlementResultUnitBase)
				// 当日金额按所在月份天数分摊
				daysInMonth := 30
				if rec.ServiceDate != nil {
					y, m, _ := rec.ServiceDate.Date()
					last := time.Date(y, m+1, 0, 0, 0, 0, 0, rec.ServiceDate.Location())
					daysInMonth = last.Day()
				}
				if rec.CustomerFee != nil {
					v := gbps * (*rec.CustomerFee) / float64(daysInMonth)
					vv := math.Round(v*100) / 100
					rec.CustomerBill = &vv
				}
				if rec.NetworkLineFee != nil {
					v := gbps * (*rec.NetworkLineFee) / float64(daysInMonth)
					vv := math.Round(v*100) / 100
					rec.NetworkLineBill = &vv
				}
				if rec.ChannelRate != nil {
					v := gbps * (*rec.ChannelRate) / float64(daysInMonth)
					vv := math.Round(v*100) / 100
					rec.ChannelBill = &vv
				}
				if rec.NodeDeductionFee != nil {
					v := gbps * (*rec.NodeDeductionFee) / float64(daysInMonth)
					vv := math.Round(v*100) / 100
					rec.NodeDeductionBill = &vv
				}
				// 回写客户费率条目的“当日增量值”快照：按批去重后统一写
				if rec.DailyIncrementValue != nil {
					pendingRateIncrement[rc.ID] = *rec.DailyIncrementValue
				}
			}
			rec.CreatedAt = now
			rec.UpdatedAt = now
			upserts = append(upserts, rec)
			lastID = it.ID
		}

		if len(upserts) > 0 {
			if err := tx.Clauses(clause.OnConflict{
				Columns: []clause.Column{
					{Name: "region"},
					{Name: "cp"},
					{Name: "school_name"},
					{Name: "service_date"},
				},
				DoUpdates: clause.Assignments(buildSettlementCustomerUpsertAssignments(markRecalc)),
			}).CreateInBatches(upserts, 500).Error; err != nil {
				tx.Rollback()
				return affected, err
			}
			affected += int64(len(upserts))
		}

		if err := updateRateCustomerIncrementBatch(tx, pendingRateIncrement); err != nil {
			tx.Rollback()
			return affected, err
		}

		if err := tx.Commit().Error; err != nil {
			return affected, err
		}
		processed += int64(len(chunkRows))
		if progress != nil {
			progress(processed, nil)
		}
	}
	return affected, nil
}

func buildSettlementCustomerUpsertAssignments(markRecalc bool) map[string]interface{} {
	assignments := map[string]interface{}{
		"settlement_value":            clause.Expr{SQL: "VALUES(settlement_value)"},
		"settlement_time":             clause.Expr{SQL: "VALUES(settlement_time)"},
		"customer_fee":                clause.Expr{SQL: "VALUES(customer_fee)"},
		"network_line_fee":            clause.Expr{SQL: "VALUES(network_line_fee)"},
		"node_deduction_fee":          clause.Expr{SQL: "VALUES(node_deduction_fee)"},
		"channel_rate":                clause.Expr{SQL: "VALUES(channel_rate)"},
		"customer_fee_owner_id":       clause.Expr{SQL: "VALUES(customer_fee_owner_id)"},
		"network_line_fee_owner_id":   clause.Expr{SQL: "VALUES(network_line_fee_owner_id)"},
		"node_deduction_fee_owner_id": clause.Expr{SQL: "VALUES(node_deduction_fee_owner_id)"},
		"channel_owner_user_id":       clause.Expr{SQL: "VALUES(channel_owner_user_id)"},
		"stock_ratio":                 clause.Expr{SQL: "VALUES(stock_ratio)"},
		"increment_ratio":             clause.Expr{SQL: "VALUES(increment_ratio)"},
		"daily_increment_value":       clause.Expr{SQL: "VALUES(daily_increment_value)"},
		"discount_rule_id":            clause.Expr{SQL: "VALUES(discount_rule_id)"},
		"service_year_index":          clause.Expr{SQL: "VALUES(service_year_index)"},
		"customer_bill":               clause.Expr{SQL: "VALUES(customer_bill)"},
		"network_line_bill":           clause.Expr{SQL: "VALUES(network_line_bill)"},
		"channel_bill":                clause.Expr{SQL: "VALUES(channel_bill)"},
		"node_deduction_bill":         clause.Expr{SQL: "VALUES(node_deduction_bill)"},
		"updated_at":                  clause.Expr{SQL: "NOW()"},
	}
	if markRecalc {
		assignments["recalculated"] = true
		assignments["last_recalc_time"] = clause.Expr{SQL: "VALUES(last_recalc_time)"}
	}
	return assignments
}

func updateRateCustomerIncrementBatch(tx *gorm.DB, pending map[uint64]float64) error {
	if len(pending) == 0 {
		return nil
	}
	ids := make([]int, 0, len(pending))
	for id := range pending {
		ids = append(ids, int(id))
	}
	sort.Ints(ids)
	var sqlBuilder strings.Builder
	sqlBuilder.WriteString("UPDATE rate_customer SET daily_increment_value = CASE id ")
	args := make([]interface{}, 0, len(ids)*3)
	for _, idInt := range ids {
		id := uint64(idInt)
		sqlBuilder.WriteString("WHEN ? THEN ? ")
		args = append(args, id, pending[id])
	}
	sqlBuilder.WriteString("ELSE daily_increment_value END WHERE id IN (")
	for i := range ids {
		if i > 0 {
			sqlBuilder.WriteString(",")
		}
		sqlBuilder.WriteString("?")
	}
	sqlBuilder.WriteString(")")
	for _, idInt := range ids {
		args = append(args, uint64(idInt))
	}
	return tx.Exec(sqlBuilder.String(), args...).Error
}

func buildMonthList(start, end time.Time) []string {
	if start.IsZero() || end.IsZero() {
		return nil
	}
	from := time.Date(start.Year(), start.Month(), 1, 0, 0, 0, 0, start.Location())
	to := time.Date(end.Year(), end.Month(), 1, 0, 0, 0, 0, end.Location())
	if to.Before(from) {
		return nil
	}
	months := make([]string, 0, 6)
	for cur := from; !cur.After(to); cur = cur.AddDate(0, 1, 0) {
		months = append(months, cur.Format("2006-01"))
	}
	return months
}

func activeSlotForMonth(tx *gorm.DB, serviceMonth string, taskID int64) (int8, error) {
	var pointer model.SettlementMonthSlotPointer
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("service_month = ?", serviceMonth).First(&pointer).Error
	if err == nil {
		return pointer.ActiveSlot, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return 0, err
	}
	pointer = model.SettlementMonthSlotPointer{
		ServiceMonth: serviceMonth,
		ActiveSlot:   0,
		TaskID:       &taskID,
		UpdatedAt:    time.Now(),
	}
	if cerr := tx.Create(&pointer).Error; cerr != nil {
		return 0, cerr
	}
	return 0, nil
}

func publishSlotForMonth(tx *gorm.DB, serviceMonth string, slot int8, taskID int64) error {
	now := time.Now()
	return tx.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "service_month"}},
		DoUpdates: clause.Assignments(map[string]interface{}{
			"active_slot": slot,
			"task_id":     taskID,
			"updated_at":  now,
		}),
	}).Create(&model.SettlementMonthSlotPointer{
		ServiceMonth: serviceMonth,
		ActiveSlot:   slot,
		TaskID:       &taskID,
		UpdatedAt:    now,
	}).Error
}

func buildTmpSourceWhere(region, cp, school string) (string, []interface{}) {
	parts := []string{"settlement_date >= ?", "settlement_date < ?"}
	args := make([]interface{}, 0, 5)
	if strings.TrimSpace(region) != "" {
		parts = append(parts, "region = ?")
	}
	if strings.TrimSpace(cp) != "" {
		parts = append(parts, "cp = ?")
	}
	if strings.TrimSpace(school) != "" {
		parts = append(parts, "school_name = ?")
	}
	if strings.TrimSpace(region) != "" {
		args = append(args, region)
	}
	if strings.TrimSpace(cp) != "" {
		args = append(args, cp)
	}
	if strings.TrimSpace(school) != "" {
		args = append(args, school)
	}
	return strings.Join(parts, " AND "), args
}

func prepareTmpTables(tx *gorm.DB) error {
	stmts := []string{
		"DROP TEMPORARY TABLE IF EXISTS tmp_source",
		"DROP TEMPORARY TABLE IF EXISTS tmp_key",
		"DROP TEMPORARY TABLE IF EXISTS tmp_rate",
		"DROP TEMPORARY TABLE IF EXISTS tmp_rule_applied",
		"DROP TEMPORARY TABLE IF EXISTS tmp_result",
	}
	for _, sql := range stmts {
		if err := tx.Exec(sql).Error; err != nil {
			return err
		}
	}
	return nil
}

func countTempRows(tx *gorm.DB, table string) (int64, error) {
	var cnt int64
	if err := tx.Table(table).Count(&cnt).Error; err != nil {
		return 0, err
	}
	return cnt, nil
}

func createTmpSourceAndKeys(
	tx *gorm.DB,
	monthStart, monthEnd time.Time,
	region, cp, school string,
) (rowsSource, rowsKey int64, err error) {
	whereClause, dynamicArgs := buildTmpSourceWhere(region, cp, school)
	args := make([]interface{}, 0, 2+len(dynamicArgs))
	args = append(args, monthStart, monthEnd.AddDate(0, 0, 1))
	args = append(args, dynamicArgs...)

	if err = tx.Exec(`
		CREATE TEMPORARY TABLE tmp_source AS
		SELECT
			CONVERT(region USING utf8mb4) COLLATE utf8mb4_unicode_ci AS region,
			CONVERT(src_region USING utf8mb4) COLLATE utf8mb4_unicode_ci AS src_region,
			CONVERT(cp USING utf8mb4) COLLATE utf8mb4_unicode_ci AS cp,
			CONVERT(school_name USING utf8mb4) COLLATE utf8mb4_unicode_ci AS school_name,
			DATE(settlement_date) AS service_date,
			settlement_value,
			settlement_time
		FROM nfa_school_settlement
		WHERE `+whereClause, args...).Error; err != nil {
		return
	}
	if err = tx.Exec(`
		ALTER TABLE tmp_source
		ADD INDEX idx_tmp_source_key (region, cp, school_name, service_date)
	`).Error; err != nil {
		return
	}
	if rowsSource, err = countTempRows(tx, "tmp_source"); err != nil {
		return
	}
	if rowsSource == 0 {
		return
	}
	if err = tx.Exec(`
		CREATE TEMPORARY TABLE tmp_key AS
		SELECT DISTINCT region, cp, school_name, service_date
		FROM tmp_source
	`).Error; err != nil {
		return
	}
	if err = tx.Exec(`
		ALTER TABLE tmp_key
		ADD PRIMARY KEY (region, cp, school_name, service_date)
	`).Error; err != nil {
		return
	}
	if rowsKey, err = countTempRows(tx, "tmp_key"); err != nil {
		return
	}
	return
}

func copyFullMonthFromActiveSlot(tx *gorm.DB, month string, activeSlot, inactiveSlot int8) (rowsDeleted, rowsCopied int64, err error) {
	delRes := tx.Exec(`
		DELETE FROM settlement_customer_v
		WHERE service_month = ? AND slot = ?
	`, month, inactiveSlot)
	if delRes.Error != nil {
		err = delRes.Error
		return
	}
	rowsDeleted = delRes.RowsAffected

	copyRes := tx.Exec(`
		INSERT INTO settlement_customer_v (
			region,src_region,cp,school_name,service_month,slot,settlement_value,settlement_time,service_date,
			recalculated,last_recalc_time,customer_fee,customer_bill,customer_fee_owner_id,
			network_line_fee,network_line_bill,network_line_fee_owner_id,node_deduction_fee,node_deduction_bill,node_deduction_fee_owner_id,
			channel_rate,channel_bill,channel_owner_user_id,stock_ratio,increment_ratio,daily_increment_value,discount_rule_id,service_year_index,
			created_at,updated_at
		)
		SELECT
			scv.region,scv.src_region,scv.cp,scv.school_name,scv.service_month,?,
			scv.settlement_value,scv.settlement_time,scv.service_date,
			scv.recalculated,scv.last_recalc_time,scv.customer_fee,scv.customer_bill,scv.customer_fee_owner_id,
			scv.network_line_fee,scv.network_line_bill,scv.network_line_fee_owner_id,scv.node_deduction_fee,scv.node_deduction_bill,scv.node_deduction_fee_owner_id,
			scv.channel_rate,scv.channel_bill,scv.channel_owner_user_id,scv.stock_ratio,scv.increment_ratio,scv.daily_increment_value,scv.discount_rule_id,scv.service_year_index,
			scv.created_at,scv.updated_at
		FROM settlement_customer_v scv FORCE INDEX (idx_scv_month_slot_service_date_id)
		WHERE scv.service_month = ?
		  AND scv.slot = ?
	`, inactiveSlot, month, activeSlot)
	if copyRes.Error != nil {
		err = copyRes.Error
		return
	}
	rowsCopied = copyRes.RowsAffected
	return
}

func countMonthSlotRows(tx *gorm.DB, month string, slot int8) (int64, error) {
	var cnt int64
	if err := tx.Raw(`
		SELECT COUNT(1)
		FROM settlement_customer_v FORCE INDEX (idx_scv_month_slot_service_date_id)
		WHERE service_month = ? AND slot = ?
	`, month, slot).Scan(&cnt).Error; err != nil {
		return 0, err
	}
	return cnt, nil
}

func runTempPipelineForMonth(
	tx *gorm.DB,
	month string,
	inactiveSlot int8,
	markRecalc bool,
) (rowsRate, rowsUpsert int64, err error) {
	settlementResultUnitBase := loadSettlementResultUnitBaseFromSystemSettings()
	unitDiv := float64(settlementResultUnitBase * settlementResultUnitBase * settlementResultUnitBase)

	if err = tx.Exec(`
		CREATE TEMPORARY TABLE tmp_rate AS
		SELECT
			k.region,
			k.cp,
			k.school_name,
			k.service_date,
			rc.id AS rate_customer_id,
			rc.customer_fee,
			rc.network_line_fee,
			rc.general_fee,
			rc.channel_rate,
			rc.customer_fee_owner_id,
			rc.network_line_fee_owner_id,
			rc.general_fee_owner_id,
			rc.channel_owner_user_id,
			rc.start_at,
			rc.increment_start_at,
			rc.stock_ratio,
			rc.increment_ratio
		FROM tmp_key k
		LEFT JOIN rate_customer rc
			ON rc.region = k.region
			AND rc.cp = k.cp
			AND rc.school_name = k.school_name
			AND (rc.start_at IS NULL OR DATE(rc.start_at) <= k.service_date)
		LEFT JOIN rate_customer rc2
			ON rc.id IS NOT NULL
			AND rc2.region = k.region
			AND rc2.cp = k.cp
			AND rc2.school_name = k.school_name
			AND (rc2.start_at IS NULL OR DATE(rc2.start_at) <= k.service_date)
			AND (
				COALESCE(rc2.start_at, '1000-01-01 00:00:00') > COALESCE(rc.start_at, '1000-01-01 00:00:00')
				OR (
					COALESCE(rc2.start_at, '1000-01-01 00:00:00') = COALESCE(rc.start_at, '1000-01-01 00:00:00')
					AND rc2.id > rc.id
				)
			)
		WHERE rc2.id IS NULL
	`).Error; err != nil {
		return
	}
	if rowsRate, err = countTempRows(tx, "tmp_rate"); err != nil {
		return
	}

	if err = tx.Exec(`
		CREATE TEMPORARY TABLE tmp_rule_applied AS
		SELECT
			tr.*,
			(
				SELECT r.id
				FROM rate_discount_rule r
				WHERE r.enabled = 1
				  AND (
					(r.scope_type = 'school' AND r.scope_key = CONVERT(tr.school_name USING utf8mb4) COLLATE utf8mb4_unicode_ci)
					OR (r.scope_type = 'cp' AND r.scope_key = CONVERT(tr.cp USING utf8mb4) COLLATE utf8mb4_unicode_ci)
					OR (r.scope_type = 'region' AND r.scope_key = CONVERT(tr.region USING utf8mb4) COLLATE utf8mb4_unicode_ci)
					OR (r.scope_type = 'global')
				  )
				ORDER BY
				  CASE r.scope_type
					WHEN 'school' THEN 1
					WHEN 'cp' THEN 2
					WHEN 'region' THEN 3
					WHEN 'global' THEN 4
					ELSE 5
				  END ASC,
				  r.priority ASC,
				  r.updated_at DESC,
				  r.id DESC
				LIMIT 1
			) AS discount_rule_id,
			(
				SELECT r.fields
				FROM rate_discount_rule r
				WHERE r.enabled = 1
				  AND (
					(r.scope_type = 'school' AND r.scope_key = CONVERT(tr.school_name USING utf8mb4) COLLATE utf8mb4_unicode_ci)
					OR (r.scope_type = 'cp' AND r.scope_key = CONVERT(tr.cp USING utf8mb4) COLLATE utf8mb4_unicode_ci)
					OR (r.scope_type = 'region' AND r.scope_key = CONVERT(tr.region USING utf8mb4) COLLATE utf8mb4_unicode_ci)
					OR (r.scope_type = 'global')
				  )
				ORDER BY
				  CASE r.scope_type
					WHEN 'school' THEN 1
					WHEN 'cp' THEN 2
					WHEN 'region' THEN 3
					WHEN 'global' THEN 4
					ELSE 5
				  END ASC,
				  r.priority ASC,
				  r.updated_at DESC,
				  r.id DESC
				LIMIT 1
			) AS discount_fields,
			CASE
				WHEN tr.start_at IS NULL THEN NULL
				ELSE GREATEST(1, TIMESTAMPDIFF(YEAR, DATE(tr.start_at), tr.service_date) + 1)
			END AS stock_year_idx,
			CASE
				WHEN tr.increment_start_at IS NULL THEN NULL
				ELSE GREATEST(1, TIMESTAMPDIFF(YEAR, DATE(tr.increment_start_at), tr.service_date) + 1)
			END AS increment_year_idx,
			CASE
				WHEN tr.rate_customer_id IS NULL THEN NULL
				WHEN tr.increment_start_at IS NULL OR tr.service_date < DATE(tr.increment_start_at) THEN 1
				ELSE COALESCE(tr.stock_ratio, 1)
			END AS stock_ratio_eff,
			CASE
				WHEN tr.rate_customer_id IS NULL THEN NULL
				WHEN tr.increment_start_at IS NULL OR tr.service_date < DATE(tr.increment_start_at) THEN 0
				ELSE COALESCE(tr.increment_ratio, 0)
			END AS increment_ratio_eff
		FROM tmp_rate tr
	`).Error; err != nil {
		return
	}
	if err = tx.Exec(`ALTER TABLE tmp_rule_applied ADD COLUMN stock_discount_ratio DECIMAL(10,6) NULL, ADD COLUMN increment_discount_ratio DECIMAL(10,6) NULL`).Error; err != nil {
		return
	}
	if err = tx.Exec(`
		UPDATE tmp_rule_applied t
		SET
			t.stock_discount_ratio = CASE
				WHEN t.discount_rule_id IS NULL OR t.stock_year_idx IS NULL THEN 1
				ELSE COALESCE((
					SELECT i.discount_rate
					FROM rate_discount_rule_item i
					WHERE i.rule_id = t.discount_rule_id
					  AND t.stock_year_idx >= i.from_year
					  AND (i.to_year IS NULL OR t.stock_year_idx <= i.to_year)
					  AND i.discount_rate > 0
					ORDER BY i.from_year ASC
					LIMIT 1
				), 1)
			END,
			t.increment_discount_ratio = CASE
				WHEN t.discount_rule_id IS NULL OR t.increment_year_idx IS NULL THEN 1
				ELSE COALESCE((
					SELECT i.discount_rate
					FROM rate_discount_rule_item i
					WHERE i.rule_id = t.discount_rule_id
					  AND t.increment_year_idx >= i.from_year
					  AND (i.to_year IS NULL OR t.increment_year_idx <= i.to_year)
					  AND i.discount_rate > 0
					ORDER BY i.from_year ASC
					LIMIT 1
				), 1)
			END
	`).Error; err != nil {
		return
	}

	if err = tx.Exec(`
		CREATE TEMPORARY TABLE tmp_result AS
		SELECT
			s.region,
			s.src_region,
			s.cp,
			s.school_name,
			s.service_date,
			s.settlement_value,
			s.settlement_time,
			r.rate_customer_id,
			r.customer_fee_owner_id,
			r.network_line_fee_owner_id,
			r.general_fee_owner_id AS node_deduction_fee_owner_id,
			r.channel_owner_user_id,
			r.discount_rule_id,
			r.stock_year_idx AS service_year_index,
			r.stock_ratio_eff AS stock_ratio,
			r.increment_ratio_eff AS increment_ratio,
			ROUND(s.settlement_value * COALESCE(r.increment_ratio_eff, 0), 6) AS daily_increment_value,
			CASE
				WHEN r.customer_fee IS NULL THEN NULL
				WHEN r.discount_rule_id IS NULL THEN r.customer_fee
				WHEN (r.discount_fields IS NULL OR JSON_LENGTH(r.discount_fields) = 0 OR JSON_CONTAINS(r.discount_fields, JSON_QUOTE('customer_fee')))
					THEN (r.customer_fee * COALESCE(r.stock_ratio_eff, 1) * COALESCE(r.stock_discount_ratio, 1))
					   + (r.customer_fee * COALESCE(r.increment_ratio_eff, 0) * COALESCE(r.increment_discount_ratio, 1))
				ELSE r.customer_fee
			END AS customer_fee,
			CASE
				WHEN r.network_line_fee IS NULL THEN NULL
				WHEN r.discount_rule_id IS NOT NULL AND JSON_CONTAINS(COALESCE(r.discount_fields, JSON_ARRAY()), JSON_QUOTE('network_line_fee'))
					THEN (r.network_line_fee * COALESCE(r.stock_ratio_eff, 1) * COALESCE(r.stock_discount_ratio, 1))
					   + (r.network_line_fee * COALESCE(r.increment_ratio_eff, 0) * COALESCE(r.increment_discount_ratio, 1))
				ELSE r.network_line_fee
			END AS network_line_fee,
			CASE
				WHEN r.general_fee IS NULL THEN NULL
				WHEN r.discount_rule_id IS NOT NULL AND JSON_CONTAINS(COALESCE(r.discount_fields, JSON_ARRAY()), JSON_QUOTE('general_fee'))
					THEN (r.general_fee * COALESCE(r.stock_ratio_eff, 1) * COALESCE(r.stock_discount_ratio, 1))
					   + (r.general_fee * COALESCE(r.increment_ratio_eff, 0) * COALESCE(r.increment_discount_ratio, 1))
				ELSE r.general_fee
			END AS node_deduction_fee,
			CASE
				WHEN r.channel_rate IS NULL THEN NULL
				WHEN r.discount_rule_id IS NOT NULL AND JSON_CONTAINS(COALESCE(r.discount_fields, JSON_ARRAY()), JSON_QUOTE('channel_rate'))
					THEN (r.channel_rate * COALESCE(r.stock_ratio_eff, 1) * COALESCE(r.stock_discount_ratio, 1))
					   + (r.channel_rate * COALESCE(r.increment_ratio_eff, 0) * COALESCE(r.increment_discount_ratio, 1))
				ELSE r.channel_rate
			END AS channel_rate
		FROM tmp_source s
		LEFT JOIN tmp_rule_applied r
		  ON r.region = s.region
		 AND r.cp = s.cp
		 AND r.school_name = s.school_name
		 AND r.service_date = s.service_date
	`).Error; err != nil {
		return
	}

	now := time.Now()
	recalcFlag := 0
	if markRecalc {
		recalcFlag = 1
	}
	lastRecalc := interface{}(nil)
	if markRecalc {
		lastRecalc = now
	}
	res := tx.Exec(`
		INSERT INTO settlement_customer_v (
			region, src_region, cp, school_name, service_month, slot,
			settlement_value, settlement_time, service_date,
			recalculated, last_recalc_time,
			customer_fee, customer_bill, customer_fee_owner_id,
			network_line_fee, network_line_bill, network_line_fee_owner_id,
			node_deduction_fee, node_deduction_bill, node_deduction_fee_owner_id,
			channel_rate, channel_bill, channel_owner_user_id,
			stock_ratio, increment_ratio, daily_increment_value, discount_rule_id, service_year_index,
			created_at, updated_at
		)
		SELECT
			r.region, r.src_region, r.cp, r.school_name, ?, ?,
			r.settlement_value, r.settlement_time, r.service_date,
			?, ?,
			r.customer_fee,
			CASE WHEN r.customer_fee IS NULL THEN NULL ELSE ROUND((((r.settlement_value * 8.0 / 60.0) / ?) * r.customer_fee) / DAY(LAST_DAY(r.service_date)), 2) END,
			r.customer_fee_owner_id,
			r.network_line_fee,
			CASE WHEN r.network_line_fee IS NULL THEN NULL ELSE ROUND((((r.settlement_value * 8.0 / 60.0) / ?) * r.network_line_fee) / DAY(LAST_DAY(r.service_date)), 2) END,
			r.network_line_fee_owner_id,
			r.node_deduction_fee,
			CASE WHEN r.node_deduction_fee IS NULL THEN NULL ELSE ROUND((((r.settlement_value * 8.0 / 60.0) / ?) * r.node_deduction_fee) / DAY(LAST_DAY(r.service_date)), 2) END,
			r.node_deduction_fee_owner_id,
			r.channel_rate,
			CASE WHEN r.channel_rate IS NULL THEN NULL ELSE ROUND((((r.settlement_value * 8.0 / 60.0) / ?) * r.channel_rate) / DAY(LAST_DAY(r.service_date)), 2) END,
			r.channel_owner_user_id,
			r.stock_ratio, r.increment_ratio, r.daily_increment_value, r.discount_rule_id, r.service_year_index,
			NOW(), NOW()
		FROM tmp_result r
		ON DUPLICATE KEY UPDATE
			src_region = COALESCE(settlement_customer_v.src_region, VALUES(src_region)),
			settlement_value = VALUES(settlement_value),
			settlement_time = VALUES(settlement_time),
			customer_fee = VALUES(customer_fee),
			network_line_fee = VALUES(network_line_fee),
			node_deduction_fee = VALUES(node_deduction_fee),
			channel_rate = VALUES(channel_rate),
			customer_fee_owner_id = VALUES(customer_fee_owner_id),
			network_line_fee_owner_id = VALUES(network_line_fee_owner_id),
			node_deduction_fee_owner_id = VALUES(node_deduction_fee_owner_id),
			channel_owner_user_id = VALUES(channel_owner_user_id),
			stock_ratio = VALUES(stock_ratio),
			increment_ratio = VALUES(increment_ratio),
			daily_increment_value = VALUES(daily_increment_value),
			discount_rule_id = VALUES(discount_rule_id),
			service_year_index = VALUES(service_year_index),
			customer_bill = VALUES(customer_bill),
			network_line_bill = VALUES(network_line_bill),
			channel_bill = VALUES(channel_bill),
			node_deduction_bill = VALUES(node_deduction_bill),
			recalculated = IF(? = 1, 1, recalculated),
			last_recalc_time = IF(? = 1, VALUES(last_recalc_time), last_recalc_time),
			updated_at = NOW()
	`, month, inactiveSlot, recalcFlag, lastRecalc, unitDiv, unitDiv, unitDiv, unitDiv, recalcFlag, recalcFlag)
	if res.Error != nil {
		err = res.Error
		return
	}
	rowsUpsert = res.RowsAffected

	if err = tx.Exec(`
		UPDATE rate_customer rc
		JOIN (
			SELECT rate_customer_id, ROUND(AVG(daily_increment_value), 6) AS new_increment_value
			FROM tmp_result
			WHERE rate_customer_id IS NOT NULL
			GROUP BY rate_customer_id
		) x ON x.rate_customer_id = rc.id
		SET rc.daily_increment_value = x.new_increment_value
	`).Error; err != nil {
		return
	}
	return
}

func cloneStageMetrics(metrics map[string]int64) map[string]int64 {
	cloned := make(map[string]int64, len(metrics))
	for k, v := range metrics {
		cloned[k] = v
	}
	return cloned
}

func (r *settlementDataRepository) backfillFromSchoolSettlementWithSlot(region, cp, school string, start, end time.Time, markRecalc bool, progress func(processed int64, stageMetrics map[string]int64)) (int64, error) {
	months := buildMonthList(start, end)
	if len(months) == 0 {
		return 0, nil
	}
	var totalAffected int64
	var processed int64
	stageMetrics := map[string]int64{
		"rows_hit_key":            0,
		"rows_source":             0,
		"rows_rate":               0,
		"rows_active_month":       0,
		"rows_deleted_inactive":   0,
		"rows_copied_from_active": 0,
		"rows_upsert":             0,
		"copy_mode_full_month":    0,
		"copy_mode_hit_key":       0,
		"copy_slot_ms":            0,
		"compute_ms":              0,
		"publish_ms":              0,
	}
	var taskID int64
	if markRecalc {
		taskID = int64(time.Now().Unix())
	}
	for _, month := range months {
		monthStart, _ := time.ParseInLocation("2006-01-02", month+"-01", start.Location())
		monthEnd := monthStart.AddDate(0, 1, -1)
		if monthStart.Before(start) {
			monthStart = start
		}
		if monthEnd.After(end) {
			monthEnd = end
		}
		tx := model.DB.Begin()
		if tx.Error != nil {
			return totalAffected, tx.Error
		}

		if err := prepareTmpTables(tx); err != nil {
			tx.Rollback()
			return totalAffected, err
		}
		rowsSource, rowsKey, err := createTmpSourceAndKeys(tx, monthStart, monthEnd, region, cp, school)
		if err != nil {
			tx.Rollback()
			return totalAffected, err
		}
		if rowsSource == 0 {
			tx.Rollback()
			continue
		}

		activeSlot, err := activeSlotForMonth(tx, month, taskID)
		if err != nil {
			tx.Rollback()
			return totalAffected, err
		}
		inactiveSlot := int8(1 - activeSlot)
		activeMonthRows, err := countMonthSlotRows(tx, month, activeSlot)
		if err != nil {
			tx.Rollback()
			return totalAffected, err
		}
		stageMetrics["rows_active_month"] += activeMonthRows

		copyStageStart := time.Now()
		// The inactive slot must be a COMPLETE mirror of the active slot before we
		// overlay the changed keys via the pipeline below. Under ping-pong double
		// buffering the inactive slot is always one generation stale (it is missing
		// whatever was written during the previous flip), so a partial "hit-key"
		// copy that only brings over the current day's keys silently drops the days
		// added in earlier flips — producing alternating-date gaps in the published
		// data. Always rebuild the inactive slot as a full month copy.
		rowsDeletedInactive, rowsCopiedFromActive, err := copyFullMonthFromActiveSlot(tx, month, activeSlot, inactiveSlot)
		if err != nil {
			tx.Rollback()
			return totalAffected, err
		}
		stageMetrics["copy_mode_full_month"]++
		stageMetrics["copy_slot_ms"] += time.Since(copyStageStart).Milliseconds()

		computeStageStart := time.Now()
		rowsRate, rowsUpsert, err := runTempPipelineForMonth(tx, month, inactiveSlot, markRecalc)
		if err != nil {
			tx.Rollback()
			return totalAffected, err
		}
		stageMetrics["compute_ms"] += time.Since(computeStageStart).Milliseconds()
		if rowsUpsert > 0 {
			totalAffected += rowsUpsert
		}

		stageMetrics["rows_source"] += rowsSource
		stageMetrics["rows_hit_key"] += rowsKey
		stageMetrics["rows_rate"] += rowsRate
		stageMetrics["rows_deleted_inactive"] += rowsDeletedInactive
		stageMetrics["rows_copied_from_active"] += rowsCopiedFromActive
		stageMetrics["rows_upsert"] += rowsUpsert

		publishStageStart := time.Now()
		if err := tx.Exec("DELETE FROM settlement_customer_monthly_v WHERE service_month = ? AND slot = ?", month, inactiveSlot).Error; err != nil {
			tx.Rollback()
			return totalAffected, err
		}
		if err := tx.Exec(`
			INSERT INTO settlement_customer_monthly_v (
				region, src_region, cp, school_name, service_month, slot,
				settlement_value, stock_ratio, increment_ratio, daily_increment_value,
				customer_fee, customer_bill, customer_fee_owner_id,
				network_line_fee, network_line_bill, network_line_fee_owner_id,
				node_deduction_fee, node_deduction_bill, node_deduction_fee_owner_id,
				channel_rate, channel_bill, channel_owner_user_id,
				recalculated, last_recalc_time, created_at, updated_at
			)
			SELECT
				region, CASE WHEN COUNT(DISTINCT src_region) = 1 THEN MAX(src_region) ELSE NULL END, cp, school_name, service_month, ?,
				ROUND(AVG(settlement_value), 6),
				ROUND(AVG(stock_ratio), 6),
				ROUND(AVG(increment_ratio), 6),
				ROUND(AVG(daily_increment_value), 6),
				ROUND(AVG(customer_fee), 6),
				ROUND(SUM(customer_bill), 2),
				CASE WHEN COUNT(DISTINCT customer_fee_owner_id) = 1 THEN MAX(customer_fee_owner_id) ELSE NULL END,
				ROUND(AVG(network_line_fee), 6),
				ROUND(SUM(network_line_bill), 2),
				CASE WHEN COUNT(DISTINCT network_line_fee_owner_id) = 1 THEN MAX(network_line_fee_owner_id) ELSE NULL END,
				ROUND(AVG(node_deduction_fee), 6),
				ROUND(SUM(node_deduction_bill), 2),
				CASE WHEN COUNT(DISTINCT node_deduction_fee_owner_id) = 1 THEN MAX(node_deduction_fee_owner_id) ELSE NULL END,
				ROUND(AVG(channel_rate), 6),
				ROUND(SUM(channel_bill), 2),
				CASE WHEN COUNT(DISTINCT channel_owner_user_id) = 1 THEN MAX(channel_owner_user_id) ELSE NULL END,
				CASE WHEN SUM(CASE WHEN recalculated = 1 THEN 1 ELSE 0 END) > 0 THEN 1 ELSE 0 END,
				MAX(last_recalc_time),
				NOW(),
				NOW()
			FROM settlement_customer_v
			WHERE service_month = ? AND slot = ?
			GROUP BY region, cp, school_name, service_month
		`, inactiveSlot, month, inactiveSlot).Error; err != nil {
			tx.Rollback()
			return totalAffected, err
		}
		if err := publishSlotForMonth(tx, month, inactiveSlot, taskID); err != nil {
			tx.Rollback()
			return totalAffected, err
		}
		stageMetrics["publish_ms"] += time.Since(publishStageStart).Milliseconds()
		if err := tx.Commit().Error; err != nil {
			return totalAffected, err
		}
		processed += rowsSource
		if progress != nil {
			progress(processed, cloneStageMetrics(stageMetrics))
		}
	}
	return totalAffected, nil
}
