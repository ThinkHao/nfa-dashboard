package repository

import (
	"fmt"
	"log"
	"math"
	"sort"
	"time"

	"nfa-dashboard/internal/model"
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
	result := model.DB.Save(config)
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
		if value != nil && value != "" {
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
	// 先查询是否已存在相同省份、运营商、学校和相同日期的结算数据
	var existingSettlement model.SchoolSettlement
	result := model.DB.Where("region = ? AND cp = ? AND school_id = ? AND DATE(settlement_date) = ?",
		settlement.Region, settlement.CP, settlement.SchoolID, settlement.SettlementDate.Format("2006-01-02")).First(&existingSettlement)

	// 如果已存在，则更新该数据
	if result.Error == nil {
		log.Printf("发现已存在的结算数据，进行更新: 省份=%s, 运营商=%s, 学校ID=%s, 日期=%s",
			settlement.Region, settlement.CP, settlement.SchoolID, settlement.SettlementDate.Format("2006-01-02"))

		// 更新字段
		existingSettlement.SettlementValue = settlement.SettlementValue
		existingSettlement.SettlementTime = settlement.SettlementTime

		// 保存更新
		result = model.DB.Save(&existingSettlement)
		return result.Error
	}

	// 如果不存在，则创建新数据
	log.Printf("创建新的结算数据: 省份=%s, 运营商=%s, 学校ID=%s, 日期=%s",
		settlement.Region, settlement.CP, settlement.SchoolID, settlement.SettlementDate.Format("2006-01-02"))
	result = model.DB.Create(settlement)
	return result.Error
}

// BatchCreateSettlements 批量创建结算数据
func (r *settlementRepository) BatchCreateSettlements(settlements []model.SchoolSettlement) error {
	// 逐个处理每条数据，确保同一天同一学校只有一条记录
	for _, settlement := range settlements {
		// 使用单条创建方法，其中已包含重复检查逻辑
		err := r.CreateSettlement(&settlement)
		if err != nil {
			log.Printf("创建结算数据失败: %v", err)
			return err
		}
	}
	return nil
}

// GetSettlements 获取结算数据列表
func (r *settlementRepository) GetSettlements(filter model.SettlementFilter) ([]model.SettlementResponse, int64, error) {
	var settlements []model.SchoolSettlement
	var count int64
	var responses []model.SettlementResponse

	// 打印过滤条件以便于调试
	log.Printf("结算数据查询过滤条件: StartDate=%v, EndDate=%v, SchoolName=%s, Region=%s, CP=%s, Limit=%d, Offset=%d",
		filter.StartDate, filter.EndDate, filter.SchoolName, filter.Region, filter.CP, filter.Limit, filter.Offset)

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

	// 计算每个时间点的总流量 (bits/s)
	type TrafficPoint struct {
		Time  time.Time
		Value int64
	}
	var trafficPoints []TrafficPoint

	for _, data := range trafficData {
		// 将字节转换为比特，并除以时间间隔（5分钟 = 300秒）得到bits/s
		totalBits := (data.TotalRecv + data.TotalSend) * 8
		bitsPerSecond := totalBits / 300
		trafficPoints = append(trafficPoints, TrafficPoint{
			Time:  data.CreateTime,
			Value: bitsPerSecond,
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
