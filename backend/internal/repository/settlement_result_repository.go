package repository

import (
    "strings"

    "nfa-dashboard/internal/model"

    "gorm.io/gorm"
)

// SettlementResultRepository 提供结算结果相关的数据访问
// 负责：
// 1. 聚合日95结算数据，返回校区粒度的流量与费率信息
// 2. 将计算后的结算结果写入缓存表（幂等 Upsert）
// 3. 查询/删除缓存表中的结算结果
type SettlementResultRepository interface {
    ListAggregatedFlows(filter model.SettlementResultFilter) ([]model.AggregatedFlowRecord, int64, error)
    UpsertResults(records []model.SettlementResultRecord) error
    ListResults(filter model.SettlementResultFilter) ([]model.SettlementResultRecord, int64, error)
    DeleteByID(id uint64) error
}

type settlementResultRepository struct{}

func NewSettlementResultRepository() SettlementResultRepository {
    return &settlementResultRepository{}
}

func (r *settlementResultRepository) ListAggregatedFlows(filter model.SettlementResultFilter) ([]model.AggregatedFlowRecord, int64, error) {
    if filter.Limit <= 0 {
        filter.Limit = 50
    }
    if filter.Offset < 0 {
        filter.Offset = 0
    }

    baseSQL := strings.Builder{}
    args := make([]interface{}, 0, 8)

    baseSQL.WriteString(
        " FROM nfa_school_settlement s\n" +
            " JOIN rate_final_customer fc ON fc.region COLLATE utf8mb4_unicode_ci = s.region COLLATE utf8mb4_unicode_ci" +
            " AND fc.cp COLLATE utf8mb4_unicode_ci = s.cp COLLATE utf8mb4_unicode_ci" +
            " AND fc.school_name COLLATE utf8mb4_unicode_ci = s.school_name COLLATE utf8mb4_unicode_ci\n" +
            " WHERE DATE(s.settlement_date) BETWEEN ? AND ?",
    )
    args = append(args,
        filter.StartDate.Format("2006-01-02"),
        filter.EndDate.Format("2006-01-02"),
    )

    if filter.Region != "" {
        baseSQL.WriteString(" AND s.region = ?")
        args = append(args, filter.Region)
    }
    if filter.CP != "" {
        baseSQL.WriteString(" AND s.cp = ?")
        args = append(args, filter.CP)
    }
    if filter.SchoolID != "" {
        baseSQL.WriteString(" AND s.school_id = ?")
        args = append(args, filter.SchoolID)
    }
    if filter.SchoolName != "" {
        baseSQL.WriteString(" AND s.school_name LIKE ?")
        args = append(args, "%"+filter.SchoolName+"%")
    }
    if filter.UserID != nil && *filter.UserID > 0 {
        baseSQL.WriteString(" AND s.school_id IN (SELECT school_id FROM user_schools WHERE user_id = ?)")
        args = append(args, *filter.UserID)
    }

    // 先统计总量
    countSQL := "SELECT COUNT(*) FROM (SELECT s.school_id" + baseSQL.String() + " GROUP BY s.region, s.cp, s.school_id, s.school_name) AS agg"
    var total int64
    if err := model.DB.Raw(countSQL, args...).Scan(&total).Error; err != nil {
        return nil, 0, err
    }
    if total == 0 {
        return []model.AggregatedFlowRecord{}, 0, nil
    }

    dataSQL := "SELECT\n" +
        " s.region,\n" +
        " s.cp,\n" +
        " s.school_id,\n" +
        " s.school_name,\n" +
        " COUNT(*) AS day_count,\n" +
        " SUM(s.settlement_value) AS total_flow,\n" +
        " MIN(s.settlement_date) AS min_date,\n" +
        " MAX(s.settlement_date) AS max_date,\n" +
        " MAX(s.update_time) AS latest_update,\n" +
        " MAX(fc.customer_fee) AS customer_fee,\n" +
        " MAX(fc.network_line_fee) AS network_line_fee,\n" +
        " MAX(fc.node_deduction_fee) AS node_deduction_fee,\n" +
        " MAX(fc.final_fee) AS final_fee" +
        baseSQL.String() +
        " GROUP BY s.region, s.cp, s.school_id, s.school_name\n" +
        " ORDER BY total_flow DESC\n" +
        " LIMIT ? OFFSET ?"

    dataArgs := append(append([]interface{}{}, args...), filter.Limit, filter.Offset)
    var records []model.AggregatedFlowRecord
    if err := model.DB.Raw(dataSQL, dataArgs...).Scan(&records).Error; err != nil {
        return nil, 0, err
    }
    return records, total, nil
}

func (r *settlementResultRepository) UpsertResults(records []model.SettlementResultRecord) error {
    if len(records) == 0 {
        return nil
    }
    return model.DB.Transaction(func(tx *gorm.DB) error {
        insertSQL := `INSERT INTO nfa_settlement_results
            (formula_id, formula_name, formula_tokens, region, cp, school_id, school_name,
             start_date, end_date, billing_days, total_95_flow, average_95_flow,
             customer_fee, network_line_fee, node_deduction_fee, final_fee,
             amount, currency, missing_days, missing_fields, calculation_detail, calculated_by)
            VALUES
            (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
            ON DUPLICATE KEY UPDATE
             formula_name = VALUES(formula_name),
             formula_tokens = VALUES(formula_tokens),
             billing_days = VALUES(billing_days),
             total_95_flow = VALUES(total_95_flow),
             average_95_flow = VALUES(average_95_flow),
             customer_fee = VALUES(customer_fee),
             network_line_fee = VALUES(network_line_fee),
             node_deduction_fee = VALUES(node_deduction_fee),
             final_fee = VALUES(final_fee),
             amount = VALUES(amount),
             currency = VALUES(currency),
             missing_days = VALUES(missing_days),
             missing_fields = VALUES(missing_fields),
             calculation_detail = VALUES(calculation_detail),
             calculated_by = VALUES(calculated_by),
             updated_at = NOW()`

        for _, record := range records {
            args := []interface{}{
                record.FormulaID,
                record.FormulaName,
                []byte(record.FormulaTokens),
                record.Region,
                record.CP,
                record.SchoolID,
                record.SchoolName,
                record.StartDate,
                record.EndDate,
                record.BillingDays,
                record.Total95Flow,
                record.Average95Flow,
                record.CustomerFee,
                record.NetworkLineFee,
                record.NodeDeductionFee,
                record.FinalFee,
                record.Amount,
                record.Currency,
                record.MissingDays,
                []byte(record.MissingFields),
                []byte(record.CalculationDetail),
                record.CalculatedBy,
            }
            if err := tx.Exec(insertSQL, args...).Error; err != nil {
                return err
            }
        }
        return nil
    })
}

func (r *settlementResultRepository) ListResults(filter model.SettlementResultFilter) ([]model.SettlementResultRecord, int64, error) {
    if filter.Limit <= 0 {
        filter.Limit = 50
    }
    if filter.Offset < 0 {
        filter.Offset = 0
    }

    db := model.DB.Model(&model.SettlementResultRecord{})

    if filter.ID > 0 {
        db = db.Where("id = ?", filter.ID)
    }

    if !filter.StartDate.IsZero() {
        db = db.Where("DATE(start_date) = ?", filter.StartDate.Format("2006-01-02"))
    }
    if !filter.EndDate.IsZero() {
        db = db.Where("DATE(end_date) = ?", filter.EndDate.Format("2006-01-02"))
    }
    if filter.Region != "" {
        db = db.Where("region = ?", filter.Region)
    }
    if filter.CP != "" {
        db = db.Where("cp = ?", filter.CP)
    }
    if filter.SchoolID != "" {
        db = db.Where("school_id = ?", filter.SchoolID)
    }
    if filter.SchoolName != "" {
        db = db.Where("school_name LIKE ?", "%"+filter.SchoolName+"%")
    }
    if filter.FormulaID > 0 {
        db = db.Where("formula_id = ?", filter.FormulaID)
    }

    if filter.UserID != nil && *filter.UserID > 0 {
        subQuery := model.DB.Table("user_schools").Select("school_id").Where("user_id = ?", *filter.UserID)
        db = db.Where("school_id IN (?)", subQuery)
    }

    var total int64
    if err := db.Count(&total).Error; err != nil {
        return nil, 0, err
    }
    if total == 0 {
        return []model.SettlementResultRecord{}, 0, nil
    }

    db = db.Order("updated_at DESC, amount DESC").Limit(filter.Limit).Offset(filter.Offset)

    var records []model.SettlementResultRecord
    if err := db.Find(&records).Error; err != nil {
        return nil, 0, err
    }
    return records, total, nil
}

func (r *settlementResultRepository) DeleteByID(id uint64) error {
    if id == 0 {
        return nil
    }
    return model.DB.Where("id = ?", id).Delete(&model.SettlementResultRecord{}).Error
}
