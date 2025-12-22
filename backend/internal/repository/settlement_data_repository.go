package repository

import (
	"errors"
	"math"
	"nfa-dashboard/internal/model"
	"time"

	"gorm.io/gorm"
)

type SettlementDataRepository interface {
	ListSettlementCustomer(filter map[string]interface{}, limit, offset int) ([]model.SettlementCustomer, int64, error)
	UpdateRecalculated(region, cp, school string, start, end time.Time) (int64, error)
	BackfillFromSchoolSettlement(region, cp, school string, start, end time.Time, markRecalc bool) (int64, error)
}

type settlementDataRepository struct{}

func NewSettlementDataRepository() SettlementDataRepository { return &settlementDataRepository{} }

func (r *settlementDataRepository) ListSettlementCustomer(filter map[string]interface{}, limit, offset int) ([]model.SettlementCustomer, int64, error) {
	qb := model.DB.Model(&model.SettlementCustomer{})
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

			// 折损：仅作用于客户费率；计算服务年序并匹配规则（school > cp > region > global）
			var yearIdx int
			if rc.StartAt != nil {
				start := time.Date(rc.StartAt.Year(), rc.StartAt.Month(), rc.StartAt.Day(), 0, 0, 0, 0, rc.StartAt.Location())
				cur := time.Date(sd.Year(), sd.Month(), sd.Day(), 0, 0, 0, 0, sd.Location())
				yearIdx = 1
				if cur.Before(start) {
					yearIdx = 1
				} else {
					for tmp := start; !cur.Before(tmp.AddDate(1, 0, 0)); tmp = tmp.AddDate(1, 0, 0) {
						yearIdx++
					}
				}
			}
			// 匹配规则
			var matchedRule model.RateDiscountRule
			// school
			if rc.SchoolName != nil && *rc.SchoolName != "" {
				if err := tx.Where("enabled = ? AND scope_type = ? AND scope_key = ?", true, "school", *rc.SchoolName).
					Order("priority ASC, updated_at DESC").First(&matchedRule).Error; err == nil {
				}
			}
			// cp
			if matchedRule.ID == 0 {
				if err := tx.Where("enabled = ? AND scope_type = ? AND scope_key = ?", true, "cp", rc.CP).
					Order("priority ASC, updated_at DESC").First(&matchedRule).Error; err == nil {
				}
			}
			// region
			if matchedRule.ID == 0 {
				if err := tx.Where("enabled = ? AND scope_type = ? AND scope_key = ?", true, "region", rc.Region).
					Order("priority ASC, updated_at DESC").First(&matchedRule).Error; err == nil {
				}
			}
			// global
			if matchedRule.ID == 0 {
				_ = tx.Where("enabled = ? AND scope_type = ?", true, "global").
					Order("priority ASC, updated_at DESC").First(&matchedRule).Error
			}
			if matchedRule.ID != 0 && yearIdx > 0 && rec.CustomerFee != nil {
				// 选取明细项
				var items []model.RateDiscountRuleItem
				if err := tx.Where("rule_id = ?", matchedRule.ID).Order("from_year ASC").Find(&items).Error; err == nil {
					var used *model.RateDiscountRuleItem
					for i := range items {
						it := &items[i]
						if yearIdx < it.FromYear {
							continue
						}
						if it.ToYear != nil && yearIdx > *it.ToYear {
							continue
						}
						used = it
						break
					}
					if used != nil && used.DiscountRate > 0 {
						v := (*rec.CustomerFee) * used.DiscountRate
						rec.CustomerFee = &v
						did := matchedRule.ID
						rec.DiscountRuleID = &did
						yi := yearIdx
						rec.ServiceYearIndex = &yi
					}
				}
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
