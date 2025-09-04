package repository

import (
	"fmt"
	"log"
	"math"
	"sort"
	"strings"
	"time"

	"nfa-dashboard/internal/model"
	
	"gorm.io/gorm"
)

// SettlementRepository 结算数据仓库接口
type SettlementRepository interface {
	// 获取结算配置
	GetSettlementConfig() (*model.SettlementConfig, error)
	// 更新结算配置
	UpdateSettlementConfig(config *model.SettlementConfig) error
	// 创建结算任务
	CreateSettlementTask(task *model.SettlementTask) error
	// 更新结算任务
	UpdateSettlementTask(task *model.SettlementTask) error
	// 删除结算任务
	DeleteSettlementTask(id int64) error
	// 获取结算任务列表
	GetSettlementTasks(filter map[string]interface{}, limit, offset int) ([]model.SettlementTask, int64, error)
	// 获取结算任务详情
	GetSettlementTaskByID(id int64) (*model.SettlementTask, error)
	// 创建结算数据
	CreateSettlement(settlement *model.SchoolSettlement) error
	// 批量创建结算数据
	BatchCreateSettlements(settlements []model.SchoolSettlement) error
	// 获取结算数据列表
	GetSettlements(filter model.SettlementFilter) ([]model.SettlementResponse, int64, error)
	// 计算95值
	CalculateDaily95(date time.Time, schoolID string) (*model.SchoolSettlement, error)
	// 按省份和运营商计算95值
	CalculateDaily95WithRegionAndCP(date time.Time, schoolID string, region string, cp string) (*model.SchoolSettlement, error)
	// 为指定学校计算所有区域和运营商的日95值
	CalculateDaily95WithRegionAndCPForAllRegionsAndCPs(date time.Time, schoolID string) ([]model.SchoolSettlement, error)
	// GetDailySettlementDetails 获取日95明细数据列表
	GetDailySettlementDetails(filter model.SettlementFilter) ([]model.DailySettlementDetail, int64, error)
}

// settlementRepository 结算数据仓库实现
type settlementRepository struct{}

// NewSettlementRepository 创建结算数据仓库实例
func NewSettlementRepository() SettlementRepository {
	return &settlementRepository{}
}

// GetSettlementConfig 获取结算配置
func (r *settlementRepository) GetSettlementConfig() (*model.SettlementConfig, error) {
	var config model.SettlementConfig
	result := model.DB.First(&config)
	if result.Error != nil {
		return nil, result.Error
	}
	return &config, nil
}

// UpdateSettlementConfig 更新结算配置
func (r *settlementRepository) UpdateSettlementConfig(config *model.SettlementConfig) error {
    // 确保有有效的ID（前端可能未传递ID）
    if config.ID == 0 {
        var existing model.SettlementConfig
        if err := model.DB.First(&existing).Error; err != nil {
            if err == gorm.ErrRecordNotFound {
                // 如果不存在记录，则创建一条新配置
                toCreate := model.SettlementConfig{
                    DailyTime:  config.DailyTime,
                    WeeklyDay:  config.WeeklyDay,
                    WeeklyTime: config.WeeklyTime,
                    Enabled:    config.Enabled,
                }
                if !config.LastExecuteTime.IsZero() {
                    toCreate.LastExecuteTime = config.LastExecuteTime
                }
                return model.DB.Create(&toCreate).Error
            }
            return err
        }
        config.ID = existing.ID
    }

    // 只更新业务字段；若 LastExecuteTime 为零值则跳过更新该列，避免写入非法时间
    updates := map[string]interface{}{
        "daily_time":  config.DailyTime,
        "weekly_day":  config.WeeklyDay,
        "weekly_time": config.WeeklyTime,
        "enabled":     config.Enabled,
    }

    if !config.LastExecuteTime.IsZero() {
        updates["last_execute_time"] = config.LastExecuteTime
    }

    result := model.DB.Model(&model.SettlementConfig{}).Where("id = ?", config.ID).Updates(updates)
    return result.Error
}

// CreateSettlementTask 创建结算任务
func (r *settlementRepository) CreateSettlementTask(task *model.SettlementTask) error {
	result := model.DB.Create(task)
	return result.Error
}

// UpdateSettlementTask 更新结算任务
func (r *settlementRepository) UpdateSettlementTask(task *model.SettlementTask) error {
	result := model.DB.Save(task)
	return result.Error
}

// DeleteSettlementTask 删除结算任务
func (r *settlementRepository) DeleteSettlementTask(id int64) error {
	result := model.DB.Delete(&model.SettlementTask{}, id)
	return result.Error
}

// GetSettlementTasks 获取结算任务列表
func (r *settlementRepository) GetSettlementTasks(filter map[string]interface{}, limit, offset int) ([]model.SettlementTask, int64, error) {
	var tasks []model.SettlementTask
	var count int64

	query := model.DB.Model(&model.SettlementTask{})

	// 应用过滤条件
	for key, value := range filter {
		if value == nil || value == "" {
			continue
		}
		// 如果 key 中包含操作符或占位符，直接作为条件使用
		// 支持示例："task_date >= ?", "task_date <= ?", "task_type LIKE ?", "id IN (?)" 等
		lowered := strings.ToLower(strings.TrimSpace(key))
		if strings.Contains(key, "?") ||
			strings.ContainsAny(lowered, "<> ") ||
			strings.Contains(lowered, " like ") ||
			strings.Contains(lowered, " in ") ||
			strings.Contains(lowered, " between ") {
			query = query.Where(key, value)
		} else {
			// 默认为等值匹配
			query = query.Where(fmt.Sprintf("%s = ?", key), value)
		}
	}

	// 获取总数
	err := query.Count(&count).Error
	if err != nil {
		return nil, 0, err
	}

	// 获取分页数据
	err = query.Order("id DESC").Limit(limit).Offset(offset).Find(&tasks).Error
	if err != nil {
		return nil, 0, err
	}

	return tasks, count, nil
}

// GetSettlementTaskByID 获取结算任务详情
func (r *settlementRepository) GetSettlementTaskByID(id int64) (*model.SettlementTask, error) {
	var task model.SettlementTask
	result := model.DB.First(&task, id)
	if result.Error != nil {
		return nil, result.Error
	}
	return &task, nil
}

// CreateSettlement 创建结算数据
func (r *settlementRepository) CreateSettlement(settlement *model.SchoolSettlement) error {
	// 打印详细的结算数据信息
	log.Printf("尝试创建结算数据: 学校ID=%s, 学校名=%s, 省份=%s, 运营商=%s, 日期=%s, 值=%d",
		settlement.SchoolID, settlement.SchoolName, settlement.Region, settlement.CP, 
		settlement.SettlementDate.Format("2006-01-02"), settlement.SettlementValue)
	
	// 检查必要字段是否为空
	if settlement.SchoolID == "" || settlement.Region == "" || settlement.CP == "" {
		log.Printf("结算数据缺少必要字段: 学校ID=%s, 省份=%s, 运营商=%s",
			settlement.SchoolID, settlement.Region, settlement.CP)
		return fmt.Errorf("结算数据缺少必要字段")
	}
	
	// 先查询是否已存在相同省份、运营商、学校和相同日期的结算数据
	var existingSettlement model.SchoolSettlement
	query := "region = ? AND cp = ? AND school_id = ? AND DATE(settlement_date) = ?"
	queryParams := []interface{}{settlement.Region, settlement.CP, settlement.SchoolID, settlement.SettlementDate.Format("2006-01-02")}
	log.Printf("查询条件: %s, 参数: %v", query, queryParams)
	
	result := model.DB.Where(query, queryParams...).First(&existingSettlement)

	// 如果已存在，则更新该数据
	if result.Error == nil {
		log.Printf("发现已存在的结算数据(ID=%d)，进行更新: 省份=%s, 运营商=%s, 学校ID=%s, 日期=%s",
			existingSettlement.ID, settlement.Region, settlement.CP, settlement.SchoolID, settlement.SettlementDate.Format("2006-01-02"))

		// 更新字段
		existingSettlement.SettlementValue = settlement.SettlementValue
		existingSettlement.SettlementTime = settlement.SettlementTime

		// 保存更新
		result = model.DB.Save(&existingSettlement)
		if result.Error != nil {
			log.Printf("更新结算数据失败: %v", result.Error)
		} else {
			log.Printf("更新结算数据成功: ID=%d", existingSettlement.ID)
		}
		return result.Error
	} else if result.Error != gorm.ErrRecordNotFound {
		log.Printf("查询结算数据时发生错误: %v", result.Error)
		return result.Error
	}

	// 如果不存在，则创建新数据
	log.Printf("创建新的结算数据: 省份=%s, 运营商=%s, 学校ID=%s, 日期=%s",
		settlement.Region, settlement.CP, settlement.SchoolID, settlement.SettlementDate.Format("2006-01-02"))
	result = model.DB.Create(settlement)
	if result.Error != nil {
		log.Printf("创建结算数据失败: %v", result.Error)
	} else {
		log.Printf("创建结算数据成功: ID=%d", settlement.ID)
	}
	return result.Error
}

// BatchCreateSettlements 批量创建结算数据
func (r *settlementRepository) BatchCreateSettlements(settlements []model.SchoolSettlement) error {
	log.Printf("开始批量创建结算数据，总数量: %d", len(settlements))
	
	if len(settlements) == 0 {
		log.Printf("没有结算数据需要保存")
		return nil
	}

	// 将数据分为需要更新和需要插入的两组
	// 1. 首先收集所有关键字段组合，用于一次性查询
	type SettlementKey struct {
		Region     string
		CP         string
		SchoolID   string
		DateString string
	}

	// 收集所有日期字符串
	keys := make([]SettlementKey, 0, len(settlements))
	for _, s := range settlements {
		keys = append(keys, SettlementKey{
			Region:     s.Region,
			CP:         s.CP,
			SchoolID:   s.SchoolID,
			DateString: s.SettlementDate.Format("2006-01-02"),
		})
	}

	// 2. 构建查询条件，使用IN操作一次性查询所有可能存在的记录
	// 将查询条件分批处理，避免查询条件过长
	existingMap := make(map[string]model.SchoolSettlement)
	batchSize := 500 // 每批查询的最大记录数

	for i := 0; i < len(keys); i += batchSize {
		end := i + batchSize
		if end > len(keys) {
			end = len(keys)
		}

		batchKeys := keys[i:end]
		var conditions []string
		var params []interface{}

		// 构建批量查询条件
		for _, key := range batchKeys {
			conditions = append(conditions, "(region = ? AND cp = ? AND school_id = ? AND DATE(settlement_date) = ?)")
			params = append(params, key.Region, key.CP, key.SchoolID, key.DateString)
		}

		if len(conditions) == 0 {
			continue
		}

		// 执行批量查询
		var existingSettlements []model.SchoolSettlement
		query := model.DB.Where(strings.Join(conditions, " OR "), params...)
		result := query.Find(&existingSettlements)

		if result.Error != nil && result.Error != gorm.ErrRecordNotFound {
			log.Printf("批量查询结算数据时发生错误: %v", result.Error)
			return result.Error
		}

		// 将现有记录添加到map中
		for _, existing := range existingSettlements {
			key := fmt.Sprintf("%s_%s_%s_%s", 
				existing.Region, 
				existing.CP, 
				existing.SchoolID, 
				existing.SettlementDate.Format("2006-01-02"))
			existingMap[key] = existing
		}
	}

	log.Printf("找到 %d 条现有结算数据需要更新", len(existingMap))

	// 3. 分离需要更新和需要新增的记录
	var toUpdate []model.SchoolSettlement
	var toInsert []model.SchoolSettlement

	for _, settlement := range settlements {
		key := fmt.Sprintf("%s_%s_%s_%s", 
			settlement.Region, 
			settlement.CP, 
			settlement.SchoolID, 
			settlement.SettlementDate.Format("2006-01-02"))

		if existing, found := existingMap[key]; found {
			// 需要更新
			existing.SettlementValue = settlement.SettlementValue
			existing.SettlementTime = settlement.SettlementTime
			toUpdate = append(toUpdate, existing)
		} else {
			// 需要新增
			toInsert = append(toInsert, settlement)
		}
	}

	// 4. 批量更新现有记录
	if len(toUpdate) > 0 {
		log.Printf("开始批量更新 %d 条结算数据", len(toUpdate))
		
		// 分批更新
		updateBatchSize := 100
		for i := 0; i < len(toUpdate); i += updateBatchSize {
			end := i + updateBatchSize
			if end > len(toUpdate) {
				end = len(toUpdate)
			}
			
			batch := toUpdate[i:end]
			for _, item := range batch {
				result := model.DB.Model(&model.SchoolSettlement{}).Where("id = ?", item.ID).Updates(map[string]interface{}{
					"settlement_value": item.SettlementValue,
					"settlement_time": item.SettlementTime,
				})
				
				if result.Error != nil {
					log.Printf("更新结算数据失败 ID=%d: %v", item.ID, result.Error)
					return result.Error
				}
			}
			log.Printf("已更新 %d/%d 条结算数据", end, len(toUpdate))
		}
	}

	// 5. 批量插入新记录
	if len(toInsert) > 0 {
		log.Printf("开始批量插入 %d 条新结算数据", len(toInsert))
		
		// 分批插入
		insertBatchSize := 100
		for i := 0; i < len(toInsert); i += insertBatchSize {
			end := i + insertBatchSize
			if end > len(toInsert) {
				end = len(toInsert)
			}
			
			batch := toInsert[i:end]
			result := model.DB.CreateInBatches(batch, len(batch))
			if result.Error != nil {
				log.Printf("批量插入结算数据失败: %v", result.Error)
				return result.Error
			}
			log.Printf("已插入 %d/%d 条新结算数据", end, len(toInsert))
		}
	}
	
	log.Printf("批量处理结算数据完成: 更新 %d 条, 新增 %d 条", len(toUpdate), len(toInsert))
	return nil
}

// GetSettlements 获取结算数据列表
func (r *settlementRepository) GetSettlements(filter model.SettlementFilter) ([]model.SettlementResponse, int64, error) {
	var count int64
	var responses []model.SettlementResponse

	// 打印过滤条件以便于调试
	log.Printf("结算数据查询过滤条件: StartDate=%v, EndDate=%v, SchoolName=%s, Region=%s, CP=%s, Limit=%d, Offset=%d",
		filter.StartDate, filter.EndDate, filter.SchoolName, filter.Region, filter.CP, filter.Limit, filter.Offset)

	// 检查是否需要进行聚合
	isMultiDayQuery := false
	if !filter.StartDate.IsZero() && !filter.EndDate.IsZero() {
		// 如果时间跨度超过1天，则认为是多日查询
		daysDiff := int(filter.EndDate.Sub(filter.StartDate).Hours() / 24)
		isMultiDayQuery = daysDiff > 0
		log.Printf("时间范围跨度: %d 天, 是否多日查询: %v", daysDiff, isMultiDayQuery)
	}

	// 如果是多日查询，我们需要聚合数据
	if isMultiDayQuery {
		return r.getAggregatedSettlements(filter)
	}

	// 如果不是按月查询，使用原来的日结算查询逻辑
	var settlements []model.SchoolSettlement
	query := model.DB.Model(&model.SchoolSettlement{})

	// 应用过滤条件
	if !filter.StartDate.IsZero() {
		// 将时间转换为当天的开始时间，使用本地时区
		loc, _ := time.LoadLocation("Asia/Shanghai")
		startDate := time.Date(filter.StartDate.Year(), filter.StartDate.Month(), filter.StartDate.Day(), 0, 0, 0, 0, loc)
		// 直接使用日期字符串进行查询，避免时区转换问题
		startDateStr := startDate.Format("2006-01-02")
		query = query.Where("DATE(settlement_date) >= ?", startDateStr)
		log.Printf("应用开始日期过滤: %v (格式化为: %s)", startDate, startDateStr)
	}

	if !filter.EndDate.IsZero() {
		// 将时间转换为当天的结束时间，使用本地时区
		loc, _ := time.LoadLocation("Asia/Shanghai")
		endDate := time.Date(filter.EndDate.Year(), filter.EndDate.Month(), filter.EndDate.Day(), 23, 59, 59, 999999999, loc)
		// 直接使用日期字符串进行查询，避免时区转换问题
		endDateStr := endDate.Format("2006-01-02")
		query = query.Where("DATE(settlement_date) <= ?", endDateStr)
		log.Printf("应用结束日期过滤: %v (格式化为: %s)", endDate, endDateStr)
	}

	if filter.SchoolID != "" {
		query = query.Where("school_id = ?", filter.SchoolID)
		log.Printf("应用学校ID过滤: %s", filter.SchoolID)
	}

	if filter.SchoolName != "" {
		query = query.Where("school_name LIKE ?", "%"+filter.SchoolName+"%")
		log.Printf("应用学校名称过滤: %s", filter.SchoolName)
	}

	if filter.Region != "" {
		query = query.Where("region = ?", filter.Region)
		log.Printf("应用地区过滤: %s", filter.Region)
	}

	if filter.CP != "" {
		query = query.Where("cp = ?", filter.CP)
		log.Printf("应用运营商过滤: %s", filter.CP)
	}

	// 获取总数
	err := query.Count(&count).Error
	if err != nil {
		log.Printf("获取结算数据总数失败: %v", err)
		return nil, 0, err
	}
	log.Printf("结算数据总数: %d", count)

	// 如果没有数据，直接返回空列表
	if count == 0 {
		log.Printf("没有找到符合条件的结算数据")
		return []model.SettlementResponse{}, 0, nil
	}

	// 获取分页数据
	err = query.Order("settlement_date DESC").Limit(filter.Limit).Offset(filter.Offset).Find(&settlements).Error
	if err != nil {
		log.Printf("获取结算数据失败: %v", err)
		return nil, 0, err
	}
	log.Printf("查询到 %d 条结算数据记录", len(settlements))

	// 转换为响应结构
	responses = make([]model.SettlementResponse, 0, len(settlements))
	for _, s := range settlements {
		responses = append(responses, model.SettlementResponse{
			ID:              s.ID,
			SchoolID:        s.SchoolID,
			SchoolName:      s.SchoolName,
			Region:          s.Region,
			CP:              s.CP,
			SettlementValue: s.SettlementValue,
			SettlementTime:  s.SettlementTime,
			SettlementDate:  s.SettlementDate,
			CreateTime:      s.CreateTime,
		})
	}

	log.Printf("返回 %d 条结算数据响应", len(responses))
	return responses, count, nil
}

// getAggregatedSettlements 获取聚合的结算数据
// GetDailySettlementDetails 获取日95明细数据列表
func (r *settlementRepository) GetDailySettlementDetails(filter model.SettlementFilter) ([]model.DailySettlementDetail, int64, error) {
	var details []model.DailySettlementDetail
	var count int64

	query := model.DB.Model(&model.SchoolSettlement{})

	// 应用过滤条件
	if !filter.StartDate.IsZero() {
		query = query.Where("DATE(settlement_date) >= ?", filter.StartDate.Format("2006-01-02"))
	}
	if !filter.EndDate.IsZero() {
		query = query.Where("DATE(settlement_date) <= ?", filter.EndDate.Format("2006-01-02"))
	}
	if filter.SchoolID != "" {
		query = query.Where("school_id = ?", filter.SchoolID)
	}
	if filter.SchoolName != "" {
		query = query.Where("school_name LIKE ?", "%"+filter.SchoolName+"%")
	}
	if filter.Region != "" {
		query = query.Where("region = ?", filter.Region)
	}
	if filter.CP != "" {
		query = query.Where("cp = ?", filter.CP)
	}

	// 获取总数
	err := query.Count(&count).Error
	if err != nil {
		return nil, 0, err
	}

	// 获取分页数据
	// 注意：这里需要将 SchoolSettlement 的字段映射到 DailySettlementDetail
	// GORM 可以通过 .Scan() 实现这一点，或者如果字段名和类型兼容，可以直接 Find
	// 为了清晰，我们显式地 Select 字段并 Scan
	// 注意：gorm tag 在 DailySettlementDetail 中已经定义了 column 映射
	result := query.Order("settlement_date DESC, id DESC").
		Limit(filter.Limit).
		Offset(filter.Offset).
		Find(&details) // GORM 会自动将 nfa_school_settlement 的列映射到 details 的字段

	if result.Error != nil {
		return nil, 0, result.Error
	}

	return details, count, nil
}

func (r *settlementRepository) getAggregatedSettlements(filter model.SettlementFilter) ([]model.SettlementResponse, int64, error) {
	log.Printf("开始聚合查询结算数据")
	
	// 构建基本查询
	query := model.DB.Model(&model.SchoolSettlement{})
	
	// 应用过滤条件
	if !filter.StartDate.IsZero() {
		loc, _ := time.LoadLocation("Asia/Shanghai")
		startDate := time.Date(filter.StartDate.Year(), filter.StartDate.Month(), filter.StartDate.Day(), 0, 0, 0, 0, loc)
		startDateStr := startDate.Format("2006-01-02")
		query = query.Where("DATE(settlement_date) >= ?", startDateStr)
	}
	
	if !filter.EndDate.IsZero() {
		loc, _ := time.LoadLocation("Asia/Shanghai")
		endDate := time.Date(filter.EndDate.Year(), filter.EndDate.Month(), filter.EndDate.Day(), 23, 59, 59, 999999999, loc)
		endDateStr := endDate.Format("2006-01-02")
		query = query.Where("DATE(settlement_date) <= ?", endDateStr)
	}
	
	if filter.SchoolID != "" {
		query = query.Where("school_id = ?", filter.SchoolID)
	}
	
	if filter.SchoolName != "" {
		query = query.Where("school_name LIKE ?", "%"+filter.SchoolName+"%")
	}
	
	if filter.Region != "" {
		query = query.Where("region = ?", filter.Region)
	}
	
	if filter.CP != "" {
		query = query.Where("cp = ?", filter.CP)
	}
	
	// 使用原生SQL来聚合数据
	// 我们需要将结算数据按学校、地区、运营商进行分组
	// 然后计算每组的总和除以天数作为聚合结算值
	sql := `
		SELECT 
			MAX(id) as id,
			school_id,
			school_name,
			region,
			cp,
			SUM(settlement_value) / COUNT(*) as settlement_value,
			MAX(settlement_time) as settlement_time,
			MIN(settlement_date) as settlement_date,
			MAX(create_time) as create_time,
			COUNT(*) as records_count
		FROM nfa_school_settlement
		WHERE 1=1
	`
	
	// 添加过滤条件
	args := []interface{}{}
	
	if !filter.StartDate.IsZero() {
		sql += " AND DATE(settlement_date) >= ?"
		args = append(args, filter.StartDate.Format("2006-01-02"))
	}
	
	if !filter.EndDate.IsZero() {
		sql += " AND DATE(settlement_date) <= ?"
		args = append(args, filter.EndDate.Format("2006-01-02"))
	}
	
	if filter.SchoolID != "" {
		sql += " AND school_id = ?"
		args = append(args, filter.SchoolID)
	}
	
	if filter.SchoolName != "" {
		sql += " AND school_name LIKE ?"
		args = append(args, "%"+filter.SchoolName+"%")
	}
	
	if filter.Region != "" {
		sql += " AND region = ?"
		args = append(args, filter.Region)
	}
	
	if filter.CP != "" {
		sql += " AND cp = ?"
		args = append(args, filter.CP)
	}
	
	// 添加分组和排序
	sql += " GROUP BY school_id, school_name, region, cp"
	sql += " ORDER BY settlement_date DESC"
	
	// 添加分页
	countSql := "SELECT COUNT(*) FROM (" + sql + ") as t"
	var totalCount int64
	err := model.DB.Raw(countSql, args...).Count(&totalCount).Error
	if err != nil {
		log.Printf("获取按月聚合的结算数据总数失败: %v", err)
		return nil, 0, err
	}
	
	// 如果没有数据，直接返回空列表
	if totalCount == 0 {
		log.Printf("没有找到符合条件的按月聚合的结算数据")
		return []model.SettlementResponse{}, 0, nil
	}
	
	// 添加分页限制
	sql += " LIMIT ? OFFSET ?"
	args = append(args, filter.Limit, filter.Offset)
	
	// 执行查询
	type MonthlySettlement struct {
		ID              int64     `gorm:"column:id"`
		SchoolID        string    `gorm:"column:school_id"`
		SchoolName      string    `gorm:"column:school_name"`
		Region          string    `gorm:"column:region"`
		CP              string    `gorm:"column:cp"`
		SettlementValue float64   `gorm:"column:settlement_value"` // 注意这里是浮点数，因为是平均值
		SettlementTime  time.Time `gorm:"column:settlement_time"`
		SettlementDate  time.Time `gorm:"column:settlement_date"`
		CreateTime      time.Time `gorm:"column:create_time"`
		RecordsCount    int       `gorm:"column:records_count"`
	}
	
	var monthlySettlements []MonthlySettlement
	err = model.DB.Raw(sql, args...).Scan(&monthlySettlements).Error
	if err != nil {
		log.Printf("查询按月聚合的结算数据失败: %v", err)
		return nil, 0, err
	}
	
	log.Printf("查询到 %d 条按月聚合的结算数据记录", len(monthlySettlements))
	
	// 转换为响应结构
	responses := make([]model.SettlementResponse, 0, len(monthlySettlements))
	for _, ms := range monthlySettlements {
		// 将浮点数四舍五入为整数
		settlementValue := int64(math.Round(ms.SettlementValue))
		
		responses = append(responses, model.SettlementResponse{
			ID:              ms.ID,
			SchoolID:        ms.SchoolID,
			SchoolName:      ms.SchoolName,
			Region:          ms.Region,
			CP:              ms.CP,
			SettlementValue: settlementValue,
			SettlementTime:  ms.SettlementTime,
			SettlementDate:  ms.SettlementDate,
			CreateTime:      ms.CreateTime,
		})
	}
	
	log.Printf("返回 %d 条按月聚合的结算数据响应", len(responses))
	return responses, totalCount, nil
}

// CalculateDaily95 计算指定日期和学校的日95值
func (r *settlementRepository) CalculateDaily95(date time.Time, schoolID string) (*model.SchoolSettlement, error) {
	// 调用新的函数，传入空的省份和运营商，表示不进行筛选
	return r.CalculateDaily95WithRegionAndCP(date, schoolID, "", "")
}

// CalculateDaily95WithRegionAndCP 计算指定日期、学校、省份和运营商的日95值
func (r *settlementRepository) CalculateDaily95WithRegionAndCP(date time.Time, schoolID string, region string, cp string) (*model.SchoolSettlement, error) {
	// 获取指定日期的开始和结束时间
	startTime := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
	endTime := time.Date(date.Year(), date.Month(), date.Day(), 23, 59, 59, 999999999, date.Location())

	// 获取学校信息
	var school model.School
	query := model.DB.Where("school_id = ?", schoolID)
	
	// 如果指定了省份，添加到查询条件
	if region != "" {
		query = query.Where("region = ?", region)
	}
	
	// 如果指定了运营商，添加到查询条件
	if cp != "" {
		query = query.Where("cp = ?", cp)
	}
	
	err := query.First(&school).Error
	if err != nil {
		log.Printf("没有找到匹配的学校信息: schoolID=%s, region=%s, cp=%s, error=%v", schoolID, region, cp, err)
		return nil, fmt.Errorf("获取学校信息失败: %v", err)
	}

	// 构建流量数据查询
	trafficQuery := model.DB.Where("school_id = ? AND create_time BETWEEN ? AND ?", schoolID, startTime, endTime)
	
	// 如果指定了省份，添加到查询条件
	if region != "" {
		trafficQuery = trafficQuery.Where("region = ?", region)
	}
	
	// 如果指定了运营商，添加到查询条件
	if cp != "" {
		trafficQuery = trafficQuery.Where("cp = ?", cp)
	}
	
	// 获取指定日期的流量数据
	var trafficData []model.SchoolTraffic
	err = trafficQuery.Find(&trafficData).Error
	if err != nil {
		return nil, fmt.Errorf("获取流量数据失败: %v", err)
	}

	if len(trafficData) == 0 {
		log.Printf("没有找到流量数据: schoolID=%s, region=%s, cp=%s", schoolID, region, cp)
		return nil, fmt.Errorf("没有找到流量数据")
	}

	// 计算每个时间点的接收流量，保留原始值
	type TrafficPoint struct {
		Time  time.Time
		Value int64
	}
	var trafficPoints []TrafficPoint

	for _, data := range trafficData {
		// 只使用TotalRecv，保留原始值，不进行单位换算
		// 前端会使用公式: 流量*8/60 进行单位换算
		trafficPoints = append(trafficPoints, TrafficPoint{
			Time:  data.CreateTime,
			Value: data.TotalRecv,
		})
	}

	// 按流量从大到小排序
	sort.Slice(trafficPoints, func(i, j int) bool {
		return trafficPoints[i].Value > trafficPoints[j].Value
	})

	// 计算95百分位的索引
	totalPoints := len(trafficPoints)
	if totalPoints == 0 {
		return nil, fmt.Errorf("没有有效的流量数据点")
	}

	// 计算需要排除的数据点数量（前5%）
	excludeCount := int(math.Ceil(float64(totalPoints) * 0.05))
	if excludeCount >= totalPoints {
		excludeCount = totalPoints - 1
	}

	// 获取95百分位的流量值和对应的时间
	index95 := excludeCount
	if index95 >= len(trafficPoints) {
		index95 = len(trafficPoints) - 1
	}
	settlement95Value := trafficPoints[index95].Value
	settlement95Time := trafficPoints[index95].Time

	// 创建结算数据
	settlementRegion := school.Region
	if region != "" {
		settlementRegion = region
	}
	
	settlementCP := school.CP
	if cp != "" {
		settlementCP = cp
	}
	
	settlement := &model.SchoolSettlement{
		SchoolID:        school.SchoolID,
		SchoolName:      school.SchoolName,
		Region:          settlementRegion,
		CP:              settlementCP,
		SettlementValue: settlement95Value,
		SettlementTime:  settlement95Time,
		SettlementDate:  date,
	}

	return settlement, nil
}

// CalculateDaily95WithRegionAndCPForAllRegionsAndCPs 为指定学校计算所有区域和运营商的日95值
func (r *settlementRepository) CalculateDaily95WithRegionAndCPForAllRegionsAndCPs(date time.Time, schoolID string) ([]model.SchoolSettlement, error) {
	log.Printf("开始计算学校 %s 在 %s 的日95值", schoolID, date.Format("2006-01-02"))

	// 获取学校信息
	var school model.School
	err := model.DB.Where("school_id = ?", schoolID).First(&school).Error
	if err != nil {
		log.Printf("获取学校信息失败: schoolID=%s, 错误=%v", schoolID, err)
		return nil, fmt.Errorf("获取学校信息失败: %v", err)
	}
	log.Printf("获取学校信息成功: schoolID=%s, 学校名=%s, 区域=%s, CP=%s", 
		school.SchoolID, school.SchoolName, school.Region, school.CP)
	
	// 获取指定日期的开始和结束时间
	startTime := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
	endTime := time.Date(date.Year(), date.Month(), date.Day(), 23, 59, 59, 999999999, date.Location())
	log.Printf("查询时间范围: %s 至 %s", startTime.Format("2006-01-02 15:04:05"), endTime.Format("2006-01-02 15:04:05"))
	
	// 使用DISTINCT查询获取所有不同的区域和运营商组合
	type RegionCPPair struct {
		Region string
		CP     string
	}
	
	var regionCPPairs []RegionCPPair
	query := "school_id = ? AND create_time BETWEEN ? AND ?"
	queryParams := []interface{}{schoolID, startTime, endTime}
	log.Printf("查询区域和运营商组合: 条件=%s, 参数=%v", query, queryParams)
	
	rows, err := model.DB.Table("nfa_school_traffic").Select("DISTINCT region, cp").Where(query, queryParams...).Rows()
	
	if err != nil {
		log.Printf("获取区域和运营商组合失败: %v", err)
		return nil, fmt.Errorf("获取区域和运营商组合失败: %v", err)
	}
	defer rows.Close()
	
	for rows.Next() {
		var pair RegionCPPair
		if err := model.DB.ScanRows(rows, &pair); err != nil {
			log.Printf("解析区域和运营商数据失败: %v", err)
			return nil, fmt.Errorf("解析区域和运营商数据失败: %v", err)
		}
		log.Printf("找到区域和运营商组合: 区域=%s, CP=%s", pair.Region, pair.CP)
		regionCPPairs = append(regionCPPairs, pair)
	}
	
	// 如果没有找到有效的区域和运营商组合，尝试使用学校的默认区域和运营商
	if len(regionCPPairs) == 0 {
		log.Printf("没有找到区域和运营商组合，将使用学校默认值: 区域=%s, CP=%s", 
			school.Region, school.CP)
		
		// 检查学校的区域和运营商是否有效
		if school.Region == "" || school.CP == "" {
			log.Printf("学校的默认区域或运营商为空，无法计算结算数据: schoolID=%s", schoolID)
			return nil, fmt.Errorf("学校的默认区域或运营商为空")
		}
		
		regionCPPairs = append(regionCPPairs, RegionCPPair{
			Region: school.Region,
			CP:     school.CP,
		})
	} else {
		log.Printf("找到 %d 个区域和运营商组合", len(regionCPPairs))
	}
	
	// 一次性获取所有流量数据，避免重复查询
	var allTrafficData []model.SchoolTraffic
	log.Printf("获取学校流量数据: schoolID=%s, 日期=%s", schoolID, date.Format("2006-01-02"))
	err = model.DB.Where("school_id = ? AND create_time BETWEEN ? AND ?", schoolID, startTime, endTime).Find(&allTrafficData).Error
	if err != nil {
		log.Printf("获取流量数据失败: %v", err)
		return nil, fmt.Errorf("获取流量数据失败: %v", err)
	}
	log.Printf("获取到 %d 条流量数据记录", len(allTrafficData))
	
	// 按区域和运营商分组流量数据
	trafficMap := make(map[string][]model.SchoolTraffic)
	for _, traffic := range allTrafficData {
		key := traffic.Region + "-" + traffic.CP
		trafficMap[key] = append(trafficMap[key], traffic)
	}
	
	// 打印分组结果
	for key, trafficList := range trafficMap {
		log.Printf("区域-运营商组合 %s 有 %d 条流量数据", key, len(trafficList))
	}
	
	// 为每个区域和运营商组合计算95值
	var settlements []model.SchoolSettlement
	log.Printf("开始为每个区域和运营商组合计算95值, 共 %d 个组合", len(regionCPPairs))
	
	for i, pair := range regionCPPairs {
		log.Printf("处理第 %d 个区域和运营商组合: 区域=%s, CP=%s", i+1, pair.Region, pair.CP)
		key := pair.Region + "-" + pair.CP
		trafficData, exists := trafficMap[key]
		
		// 如果没有该组合的流量数据，跳过
		if !exists || len(trafficData) == 0 {
			log.Printf("没有找到流量数据: schoolID=%s, region=%s, cp=%s", schoolID, pair.Region, pair.CP)
			continue
		}
		log.Printf("找到 %d 条流量数据记录: schoolID=%s, region=%s, cp=%s", len(trafficData), schoolID, pair.Region, pair.CP)
		
		// 计算每个时间点的总流量 (bits/s)
		type TrafficPoint struct {
			Time  time.Time
			Value int64
		}
		var trafficPoints []TrafficPoint

		for j, data := range trafficData {
			// 将字节转换为比特，并除以时间间隔（5分钟 = 300秒）得到bits/s
			totalBits := (data.TotalRecv + data.TotalSend) * 8
			bitsPerSecond := totalBits / 300
			trafficPoints = append(trafficPoints, TrafficPoint{
				Time:  data.CreateTime,
				Value: bitsPerSecond,
			})
			
			// 打印前5条流量数据作为示例
			if j < 5 {
				log.Printf("流量数据示例[%d]: 时间=%s, 接收=%d, 发送=%d, 总比特=%d, 比特每秒=%d", 
					j, data.CreateTime.Format("2006-01-02 15:04:05"), data.TotalRecv, data.TotalSend, totalBits, bitsPerSecond)
			}
		}
		log.Printf("共生成 %d 个流量数据点", len(trafficPoints))

		// 按流量从大到小排序
		log.Printf("开始按流量从大到小排序 %d 个数据点", len(trafficPoints))
		sort.Slice(trafficPoints, func(i, j int) bool {
			return trafficPoints[i].Value > trafficPoints[j].Value
		})

		// 打印排序后的前5个流量数据点
		for j := 0; j < 5 && j < len(trafficPoints); j++ {
			log.Printf("排序后数据点[%d]: 时间=%s, 值=%d", 
				j, trafficPoints[j].Time.Format("2006-01-02 15:04:05"), trafficPoints[j].Value)
		}

		// 计算95百分位的索引
		totalPoints := len(trafficPoints)
		if totalPoints == 0 {
			log.Printf("没有有效的流量数据点，跳过计算")
			continue
		}

		// 计算需要排除的数据点数量（前5%）
		excludeCount := int(math.Ceil(float64(totalPoints) * 0.05))
		if excludeCount >= totalPoints {
			excludeCount = totalPoints - 1
		}
		log.Printf("总数据点数量=%d, 排除前 %d 个数据点", totalPoints, excludeCount)

		// 获取95百分位的流量值和对应的时间
		index95 := excludeCount
		if index95 >= len(trafficPoints) {
			index95 = len(trafficPoints) - 1
		}
		settlement95Value := trafficPoints[index95].Value
		settlement95Time := trafficPoints[index95].Time
		log.Printf("计算得到日95值: 值=%d, 时间=%s", 
			settlement95Value, settlement95Time.Format("2006-01-02 15:04:05"))

		// 创建结算数据
		settlement := model.SchoolSettlement{
			SchoolID:        school.SchoolID,
			SchoolName:      school.SchoolName,
			Region:          pair.Region,
			CP:              pair.CP,
			SettlementValue: settlement95Value,
			SettlementTime:  settlement95Time,
			SettlementDate:  date,
		}
		log.Printf("创建结算数据: 学校ID=%s, 学校名=%s, 区域=%s, CP=%s, 日期=%s, 值=%d", 
			settlement.SchoolID, settlement.SchoolName, settlement.Region, settlement.CP, 
			settlement.SettlementDate.Format("2006-01-02"), settlement.SettlementValue)

		settlements = append(settlements, settlement)
	}
	
	log.Printf("计算完成，共生成 %d 条结算数据", len(settlements))
	if len(settlements) > 0 {
		for i := 0; i < 5 && i < len(settlements); i++ {
			log.Printf("结算数据示例[%d]: 学校ID=%s, 学校名=%s, 区域=%s, CP=%s, 日期=%s, 值=%d", 
				i, settlements[i].SchoolID, settlements[i].SchoolName, settlements[i].Region, settlements[i].CP, 
				settlements[i].SettlementDate.Format("2006-01-02"), settlements[i].SettlementValue)
		}
	} else {
		log.Printf("警告: 没有生成任何结算数据")
	}
	return settlements, nil
}
