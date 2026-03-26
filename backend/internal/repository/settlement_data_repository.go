package repository

import (
	"encoding/json"
	"errors"
	"math"
	"nfa-dashboard/internal/model"
	"strings"
	"time"

	"gorm.io/gorm"
)

type SettlementDataRepository interface {
	ListSettlementCustomer(filter map[string]interface{}, limit, offset int) ([]model.SettlementCustomer, int64, error)
	ListSettlementCustomerMonthly(filter map[string]interface{}, limit, offset int) ([]model.SettlementCustomerMonthly, int64, error)
	RebuildSettlementCustomerMonthly(start, end time.Time) (int64, error)
	UpdateRecalculated(region, cp, school string, start, end time.Time) (int64, error)
	BackfillFromSchoolSettlement(region, cp, school string, start, end time.Time, markRecalc bool) (int64, error)
}

type settlementDataRepository struct{}

func NewSettlementDataRepository() SettlementDataRepository { return &settlementDataRepository{} }

func applySettlementCustomerFilters(qb *gorm.DB, filter map[string]interface{}) *gorm.DB {
	if v, ok := filter["region"]; ok && v != "" {
		qb = qb.Where("region = ?", v)
	}
	if v, ok := filter["cp"]; ok && v != "" {
		qb = qb.Where("cp = ?", v)
	}
	if v, ok := filter["school_name"]; ok && v != "" {
		qb = qb.Where("school_name LIKE ?", "%"+v.(string)+"%")
	}
	if v, ok := filter["start_service_date"]; ok && v != nil {
		// 使用 DATE(service_date) 与 YYYY-MM-DD 比较，确保包含起始日期整天并避免时区影响
		if t, ok2 := v.(time.Time); ok2 {
			qb = qb.Where("DATE(service_date) >= ?", t.Format("2006-01-02"))
		} else {
			qb = qb.Where("DATE(service_date) >= ?", v)
		}
	}
	if v, ok := filter["end_service_date"]; ok && v != nil {
		// 使用 DATE(service_date) 与 YYYY-MM-DD 比较，确保包含结束日期整天
		if t, ok2 := v.(time.Time); ok2 {
			qb = qb.Where("DATE(service_date) <= ?", t.Format("2006-01-02"))
		} else {
			qb = qb.Where("DATE(service_date) <= ?", v)
		}
	}

	// 费用归属业务对象过滤（客户费/线路费/节点通用费 任一匹配）
	if v, ok := filter["owner_entity_id"]; ok && v != nil {
		qb = qb.Where("(customer_fee_owner_id = ? OR network_line_fee_owner_id = ? OR node_deduction_fee_owner_id = ?)", v, v, v)
	}
	// 渠道归属用户过滤（统一为用户ID后，需在四个归属字段中任一匹配）
	if v, ok := filter["channel_owner_user_id"]; ok && v != nil {
		qb = qb.Where("(customer_fee_owner_id = ? OR network_line_fee_owner_id = ? OR node_deduction_fee_owner_id = ? OR channel_owner_user_id = ?)", v, v, v, v)
	}
	return qb
}

func (r *settlementDataRepository) ListSettlementCustomer(filter map[string]interface{}, limit, offset int) ([]model.SettlementCustomer, int64, error) {
	qb := applySettlementCustomerFilters(model.DB.Model(&model.SettlementCustomer{}), filter)

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

func (r *settlementDataRepository) listSettlementCustomerMonthlyFromDaily(filter map[string]interface{}, limit, offset int) ([]model.SettlementCustomerMonthly, int64, error) {
	base := applySettlementCustomerFilters(model.DB.Model(&model.SettlementCustomer{}), filter).Where("service_date IS NOT NULL")

	// 统计分组后的总条数
	countSubQuery := base.
		Select("region, cp, school_name, DATE_FORMAT(service_date, '%Y-%m') AS service_month").
		Group("region, cp, school_name, DATE_FORMAT(service_date, '%Y-%m')")
	var total int64
	if err := model.DB.Table("(?) AS grouped", countSubQuery).Count(&total).Error; err != nil {
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

func (r *settlementDataRepository) ListSettlementCustomerMonthly(filter map[string]interface{}, limit, offset int) ([]model.SettlementCustomerMonthly, int64, error) {
	// 带用户过滤时，需要按用户掩码金额，必须从日表实时聚合保证精确
	if shouldUseDailyMonthlyAggregation(filter) {
		return r.listSettlementCustomerMonthlyFromDaily(filter, limit, offset)
	}

	if limit <= 0 {
		limit = 10
	}
	if offset < 0 {
		offset = 0
	}

	qb := applyMonthlySnapshotFilters(model.DB.Table("settlement_customer_monthly"), filter)

	var total int64
	if err := qb.Count(&total).Error; err != nil {
		// 月表不存在时，回退实时聚合，避免功能中断
		return r.listSettlementCustomerMonthlyFromDaily(filter, limit, offset)
	}
	if stale, ferr := isMonthlySnapshotStale(filter); ferr != nil || stale {
		return r.listSettlementCustomerMonthlyFromDaily(filter, limit, offset)
	}

	var rows []model.SettlementCustomerMonthly
	err := qb.Select(`
			region,
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
		return r.listSettlementCustomerMonthlyFromDaily(filter, limit, offset)
	}
	return rows, total, nil
}

func applyMonthlySnapshotFilters(qb *gorm.DB, filter map[string]interface{}) *gorm.DB {
	if v, ok := filter["region"]; ok && v != "" {
		qb = qb.Where("region = ?", v)
	}
	if v, ok := filter["cp"]; ok && v != "" {
		qb = qb.Where("cp = ?", v)
	}
	if v, ok := filter["school_name"]; ok && v != "" {
		qb = qb.Where("school_name LIKE ?", "%"+v.(string)+"%")
	}
	if v, ok := filter["start_service_date"]; ok && v != nil {
		if m, ok2 := toYearMonth(v); ok2 {
			qb = qb.Where("service_month >= ?", m)
		}
	}
	if v, ok := filter["end_service_date"]; ok && v != nil {
		if m, ok2 := toYearMonth(v); ok2 {
			qb = qb.Where("service_month <= ?", m)
		}
	}
	return qb
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

func isMonthlySnapshotStale(filter map[string]interface{}) (bool, error) {
	dailyQB := model.DB.Model(&model.SettlementCustomer{}).Where("service_date IS NOT NULL")
	if v, ok := filter["region"]; ok && v != "" {
		dailyQB = dailyQB.Where("region = ?", v)
	}
	if v, ok := filter["cp"]; ok && v != "" {
		dailyQB = dailyQB.Where("cp = ?", v)
	}
	if v, ok := filter["school_name"]; ok && v != "" {
		dailyQB = dailyQB.Where("school_name LIKE ?", "%"+v.(string)+"%")
	}
	if v, ok := filter["start_service_date"]; ok && v != nil {
		if t, ok2 := v.(time.Time); ok2 {
			dailyQB = dailyQB.Where("DATE(service_date) >= ?", t.Format("2006-01-02"))
		}
	}
	if v, ok := filter["end_service_date"]; ok && v != nil {
		if t, ok2 := v.(time.Time); ok2 {
			dailyQB = dailyQB.Where("DATE(service_date) <= ?", t.Format("2006-01-02"))
		}
	}
	var maxDailyUpdatedAt *time.Time
	if err := dailyQB.Select("MAX(updated_at)").Scan(&maxDailyUpdatedAt).Error; err != nil {
		return true, err
	}
	if maxDailyUpdatedAt == nil {
		return false, nil
	}

	snapQB := applyMonthlySnapshotFilters(model.DB.Table("settlement_customer_monthly"), filter)
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
	qb := model.DB.Model(&model.SettlementCustomer{}).Where("service_date IS NOT NULL")
	if !start.IsZero() {
		qb = qb.Where("DATE(service_date) >= ?", start.Format("2006-01-02"))
	}
	if !end.IsZero() {
		qb = qb.Where("DATE(service_date) <= ?", end.Format("2006-01-02"))
	}

	base := qb.Select(`
		region,
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
			region, cp, school_name, service_month,
			settlement_value, stock_ratio, increment_ratio, daily_increment_value,
			customer_fee, customer_bill, customer_fee_owner_id,
			network_line_fee, network_line_bill, network_line_fee_owner_id,
			node_deduction_fee, node_deduction_bill, node_deduction_fee_owner_id,
			channel_rate, channel_bill, channel_owner_user_id,
			recalculated, last_recalc_time, created_at, updated_at
		)
		SELECT
			region, cp, school_name, service_month,
			settlement_value, stock_ratio, increment_ratio, daily_increment_value,
			customer_fee, customer_bill, customer_fee_owner_id,
			network_line_fee, network_line_bill, network_line_fee_owner_id,
			node_deduction_fee, node_deduction_bill, node_deduction_fee_owner_id,
			channel_rate, channel_bill, channel_owner_user_id,
			recalculated, last_recalc_time, NOW(), NOW()
		FROM ( ? ) AS agg
		ON DUPLICATE KEY UPDATE
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

// BackfillFromSchoolSettlement 按条件从 nfa_school_settlement 回填/覆盖 settlement_customer 的基础字段
// 说明：
// - 不做费用计算，仅复制 95 值与服务日期/时间等基础信息
// - 以 (region, cp, school_name, service_date) 作为匹配键，存在则更新，不存在则插入
// - markRecalc: 为 true 时表示“复算”，会设置 recalculated 与 last_recalc_time；为 false 表示“初算”，不设置上述标记
func (r *settlementDataRepository) BackfillFromSchoolSettlement(region, cp, school string, start, end time.Time, markRecalc bool) (int64, error) {
	// 查询来源数据
	src := model.DB.Model(&model.SchoolSettlement{})
	if !start.IsZero() {
		// 使用 DATE(settlement_date) 与 YYYY-MM-DD 比较，确保包含起始日期整天
		src = src.Where("DATE(settlement_date) >= ?", start.Format("2006-01-02"))
	}
	if !end.IsZero() {
		// 使用 DATE(settlement_date) 与 YYYY-MM-DD 比较，确保包含结束日期整天
		src = src.Where("DATE(settlement_date) <= ?", end.Format("2006-01-02"))
	}
	if region != "" {
		src = src.Where("region = ?", region)
	}
	if cp != "" {
		src = src.Where("cp = ?", cp)
	}
	if school != "" {
		// 完全匹配学校名（与前端筛选一致）；如需模糊可改为 LIKE
		src = src.Where("school_name = ?", school)
	}

	var rows []model.SchoolSettlement
	if err := src.Order("settlement_date ASC, id ASC").Find(&rows).Error; err != nil {
		return 0, err
	}

	if len(rows) == 0 {
		return 0, nil
	}

	tx := model.DB.Begin()
	if tx.Error != nil {
		return 0, tx.Error
	}

	now := time.Now()
	var affected int64 = 0

	for _, it := range rows {
		sd := it.SettlementDate
		rec := model.SettlementCustomer{
			Region:          it.Region,
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
		var rc model.RateCustomer
		rcq := tx.Model(&model.RateCustomer{}).
			Where("region = ? AND cp = ? AND school_name = ?", it.Region, it.CP, it.SchoolName)
		if !sd.IsZero() {
			rcq = rcq.Where("(start_at IS NULL OR start_at <= ?)", sd)
		}
		if err := rcq.Order("start_at DESC, id DESC").First(&rc).Error; err == nil {
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
			matchedRule, _ := findMatchedDiscountRule(tx, &rc)
			stockDiscountRatio := 1.0
			incrementDiscountRatio := 1.0
			if matchedRule != nil {
				stockDiscountRatio = findDiscountRatioByYear(tx, matchedRule.ID, stockYearIdx)
				incrementDiscountRatio = findDiscountRatioByYear(tx, matchedRule.ID, incrementYearIdx)
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

			// 金额计算：费率单位 元/Gbps（客户金额使用折损后的费率）
			// 参考前端换算逻辑：bitsPerSecond = settlement_value * 8 / 60
			// 展示为 Mbps = bitsPerSecond / 1e6；则 Gbps = bitsPerSecond / 1e9
			bitsPerSecond := rec.SettlementValue * 8.0 / 60.0
			gbps := bitsPerSecond / 1_000_000_000.0
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
			// 回写客户费率条目的“当日增量值”快照，便于费率列表直接展示
			if rec.DailyIncrementValue != nil {
				_ = tx.Model(&model.RateCustomer{}).Where("id = ?", rc.ID).Update("daily_increment_value", *rec.DailyIncrementValue).Error
			}
		}

		var existing model.SettlementCustomer
		err := tx.Where("region = ? AND cp = ? AND school_name = ? AND service_date = ?",
			it.Region, it.CP, it.SchoolName, it.SettlementDate,
		).First(&existing).Error

		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				if cerr := tx.Create(&rec).Error; cerr != nil {
					tx.Rollback()
					return affected, cerr
				}
				affected++
			} else {
				tx.Rollback()
				return affected, err
			}
		} else {
			// 更新基础字段与复算标记
			upd := map[string]interface{}{
				"settlement_value": rec.SettlementValue,
				"settlement_time":  rec.SettlementTime,
				"service_date":     rec.ServiceDate,
			}
			if markRecalc {
				upd["recalculated"] = true
				upd["last_recalc_time"] = now
			}
			// 同步费率与归属（如有）
			if rec.CustomerFee != nil {
				upd["customer_fee"] = rec.CustomerFee
			}
			if rec.NetworkLineFee != nil {
				upd["network_line_fee"] = rec.NetworkLineFee
			}
			if rec.NodeDeductionFee != nil {
				upd["node_deduction_fee"] = rec.NodeDeductionFee
			}
			if rec.ChannelRate != nil {
				upd["channel_rate"] = rec.ChannelRate
			}
			if rec.CustomerFeeOwnerID != nil {
				upd["customer_fee_owner_id"] = rec.CustomerFeeOwnerID
			}
			if rec.NetworkLineFeeOwnerID != nil {
				upd["network_line_fee_owner_id"] = rec.NetworkLineFeeOwnerID
			}
			if rec.NodeDeductionFeeOwnerID != nil {
				upd["node_deduction_fee_owner_id"] = rec.NodeDeductionFeeOwnerID
			}
			if rec.ChannelOwnerUserID != nil {
				upd["channel_owner_user_id"] = rec.ChannelOwnerUserID
			}
			if rec.StockRatio != nil {
				upd["stock_ratio"] = rec.StockRatio
			}
			if rec.IncrementRatio != nil {
				upd["increment_ratio"] = rec.IncrementRatio
			}
			if rec.DailyIncrementValue != nil {
				upd["daily_increment_value"] = rec.DailyIncrementValue
			}
			// 同步金额（如计算出）
			if rec.CustomerBill != nil {
				upd["customer_bill"] = rec.CustomerBill
			}
			if rec.NetworkLineBill != nil {
				upd["network_line_bill"] = rec.NetworkLineBill
			}
			if rec.ChannelBill != nil {
				upd["channel_bill"] = rec.ChannelBill
			}
			if rec.NodeDeductionBill != nil {
				upd["node_deduction_bill"] = rec.NodeDeductionBill
			}
			if uerr := tx.Model(&existing).Updates(upd).Error; uerr != nil {
				tx.Rollback()
				return affected, uerr
			}
			affected++
		}
	}

	if err := tx.Commit().Error; err != nil {
		return affected, err
	}
	return affected, nil
}
