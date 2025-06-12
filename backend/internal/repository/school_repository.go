package repository

import (
	"context"
	"fmt"
	"log"
	"sort"
	"time"

	"nfa-dashboard/internal/model"
)

// SchoolRepository 学校数据仓库接口
type SchoolRepository interface {
	// 获取所有学校
	GetAllSchools(filter map[string]interface{}, limit, offset int) ([]model.School, int64, error)
	// 获取所有地区
	GetAllRegions() ([]string, error)
	// 获取所有运营商
	GetAllCPs() ([]string, error)
	// 根据过滤条件获取流量数据
	GetTrafficData(filter model.TrafficFilter) ([]model.TrafficResponse, error)
	// 获取流量汇总数据
	GetTrafficSummary(filter model.TrafficFilter) (model.TrafficResponse, error)
}

// schoolRepository 学校数据仓库实现
type schoolRepository struct{}

// NewSchoolRepository 创建学校数据仓库实例
func NewSchoolRepository() SchoolRepository {
	return &schoolRepository{}
}

// GetAllSchools 获取所有学校
func (r *schoolRepository) GetAllSchools(filter map[string]interface{}, limit, offset int) ([]model.School, int64, error) {
	var schools []model.School
	var count int64

	query := model.DB.Model(&model.School{})

	// 应用过滤条件，优化查询性能
	for key, value := range filter {
		if value != "" {
			strValue := value.(string)
			// 根据字段类型选择合适的查询方式
			switch key {
			case "school_id", "primary_hash_uuid", "data_hash":
				// 对于精确匹配的字段，使用等于查询
				query = query.Where(key+" = ?", strValue)
			case "region", "cp":
				// 对于枚举类型的字段，使用等于查询
				query = query.Where(key+" = ?", strValue)
			case "school_name":
				// 对于需要模糊匹配的字段，使用前缀匹配以提高性能
				query = query.Where(key+" LIKE ?", strValue+"%")
			default:
				// 默认使用模糊匹配
				query = query.Where(key+" LIKE ?", "%"+strValue+"%")
			}
		}
	}

	// 添加排序以提高查询性能
	query = query.Order("id ASC")

	// 获取总数
	err := query.Count(&count).Error
	if err != nil {
		return nil, 0, err
	}

	// 获取分页数据
	err = query.Limit(limit).Offset(offset).Find(&schools).Error
	if err != nil {
		return nil, 0, err
	}

	return schools, count, nil
}

// GetAllRegions 获取所有地区
func (r *schoolRepository) GetAllRegions() ([]string, error) {
	var regions []string
	err := model.DB.Model(&model.School{}).Distinct().Pluck("region", &regions).Error
	return regions, err
}

// GetAllCPs 获取所有运营商
func (r *schoolRepository) GetAllCPs() ([]string, error) {
	var cps []string
	err := model.DB.Model(&model.School{}).Distinct().Pluck("cp", &cps).Error
	return cps, err
}

// GetTrafficData 根据过滤条件获取流量数据
func (r *schoolRepository) GetTrafficData(filter model.TrafficFilter) ([]model.TrafficResponse, error) {
	var results []model.TrafficResponse

	// 限制查询时间范围，避免全表扫描
	if filter.StartTime.IsZero() && filter.EndTime.IsZero() {
		// 默认查询最近1天的数据，减少查询范围
		filter.EndTime = time.Now()
		filter.StartTime = filter.EndTime.AddDate(0, 0, -1)
	} else if filter.StartTime.IsZero() {
		// 如果只有结束时间，则设置开始时间为1天前
		filter.StartTime = filter.EndTime.AddDate(0, 0, -1)
	} else if filter.EndTime.IsZero() {
		// 如果只有开始时间，则设置结束时间为当前时间
		filter.EndTime = time.Now()
	}

	// 根据时间范围长度自动调整查询策略，参考 Grafana 的做法
	timeRange := filter.EndTime.Sub(filter.StartTime)

	// 记录查询时间信息
	_ = filter.Interval // 避免未使用警告

	// 计算时间范围分钟数
	timeMinutes := timeRange.Minutes()

	// 如果前端指定了granularity参数，则使用前端指定的粒度
	if filter.Granularity != "" {
		log.Printf("使用前端指定的粒度: %s", filter.Granularity)
		filter.Interval = filter.Granularity
	} else if filter.Interval == "" {
		// 始终使用原始5分钟粒度，不进行自动调整
		filter.Interval = "" // 原始5分钟粒度
		log.Printf("时间范围为%.2f小时(%.2f分钟)，使用原始5分钟粒度", timeRange.Hours(), timeMinutes)

		// 计算预期数据点数量（每5分钟一个点）
		expectedPoints := int(timeMinutes/5) + 10 // 每5分钟一个点，加上缓冲
		log.Printf("预期数据点数量: %d", expectedPoints)

		// 确保返回足够的数据点
		if expectedPoints > filter.Limit {
			filter.Limit = expectedPoints
			log.Printf("增加限制数量为%d，以确保返回足够的数据点", filter.Limit)
		}
	}

	// 不再需要检查是否有过滤条件，因为我们始终使用原始5分钟粒度

	// 计算时间范围差值（分钟和小时）
	timeDiffMinutes := filter.EndTime.Sub(filter.StartTime).Minutes()
	timeDiffHours := timeDiffMinutes / 60
	timeDiffDays := timeDiffHours / 24

	// 记录原始预期数据点数量
	filter.OriginalExpectedPoints = int(timeDiffMinutes / 5) // 每5分钟一个数据点
	log.Printf("预期数据点数量: %d，当前限制: %d", filter.OriginalExpectedPoints, filter.Limit)

	// 检查前端传来的限制值
	log.Printf("前端请求的数据限制为: %d条", filter.Limit)

	// 根据时间范围确保最小数据量，但不覆盖前端请求的更大限制
	minLimit := 0
	if timeDiffDays > 25 { // 超过25天
		minLimit = 8000 // 最少需要8000条数据
		log.Printf("长时间范围查询(%.2f天)，建议最少%d条", timeDiffDays, minLimit)
	} else if timeDiffDays > 14 { // 14-25天
		minLimit = 5000 // 最少需要5000条数据
		log.Printf("中长时间范围查询(%.2f天)，建议最少%d条", timeDiffDays, minLimit)
	} else if timeDiffDays > 7 { // 7-14天
		minLimit = 4000 // 最少需要4000条数据
		log.Printf("中时间范围查询(%.2f天)，建议最少%d条", timeDiffDays, minLimit)
	} else {
		// 对于7天以内的数据，根据时间范围计算最小限制
		minLimit = int(timeDiffMinutes/5) + 100 // 每5分钟一个点，加上缓冲
		log.Printf("短时间范围查询(%.2f天)，建议最少%d条", timeDiffDays, minLimit)
	}

	// 使用前端请求的限制和最小限制中的较大值
	if filter.Limit < minLimit {
		filter.Limit = minLimit
		log.Printf("前端请求的限制值过小，调整为%d条", filter.Limit)
	} else {
		log.Printf("使用前端请求的限制值: %d条", filter.Limit)
	}

	// 对于长时间范围，使用更高效的查询策略
	var query string
	var args []interface{}

	if timeDiffDays > 14 {
		// 对于超过14天的查询，使用平均采样策略

		// 计算时间间隔（小时）
		timeIntervalHours := 1.0 // 默认1小时
		if timeDiffDays > 25 {
			timeIntervalHours = 3.0 // 超过25天用3小时间隔
		} else if timeDiffDays > 20 {
			timeIntervalHours = 2.0 // 20-25天用2小时间隔
		}

		// 创建带时间间隔的查询
		query = fmt.Sprintf(`
			SELECT 
				create_time,
				school_id,
				school_name,
				region,
				cp,
				total_recv,
				total_send
			FROM nfa_school_traffic
			WHERE create_time BETWEEN ? AND ?
				AND MOD(HOUR(create_time), %.1f) < 1
				AND MINUTE(create_time) BETWEEN 0 AND 10`, timeIntervalHours)

		log.Printf("长时间范围查询(%.2f天)，使用%.1f小时间隔采样", timeDiffDays, timeIntervalHours)
	} else {
		// 对于14天以内的查询，使用原始查询
		query = `
			SELECT 
				create_time,
				school_id,
				school_name,
				region,
				cp,
				total_recv,
				total_send
			FROM nfa_school_traffic
			WHERE create_time BETWEEN ? AND ?`

		if timeDiffDays > 7 {
			// 7-14天，每30分钟采样一个点
			query += " AND MINUTE(create_time) % 30 < 5"
			log.Printf("中时间范围查询(%.2f天)，使用每30分钟采样", timeDiffDays)
		} else if timeDiffDays > 3 {
			// 3-7天，每15分钟采样一个点
			query += " AND MINUTE(create_time) % 15 < 5"
			log.Printf("短时间范围查询(%.2f天)，使用每15分钟采样", timeDiffDays)
		}
	}

	// 初始化参数
	args = []interface{}{filter.StartTime, filter.EndTime}

	// 添加过滤条件
	if filter.SchoolName != "" {
		query += " AND school_name LIKE ?"
		args = append(args, filter.SchoolName+"%")
	}
	if filter.Region != "" {
		query += " AND region = ?"
		args = append(args, filter.Region)
	}
	if filter.CP != "" {
		query += " AND cp = ?"
		args = append(args, filter.CP)
	}

	// 不使用FORCE INDEX，因为语法位置有问题
	// 我们的查询已经足够高效，不需要强制指定索引

	// 添加排序
	query += " ORDER BY create_time ASC"

	// 添加限制
	if filter.Limit > 0 {
		query += " LIMIT ?"
		args = append(args, filter.Limit)
	}

	log.Printf("最终查询SQL: %s", query)
	log.Printf("查询参数: %v", args)

	// 执行查询
	log.Printf("查询参数: %v", args)

	// 如果查询的数据量过大，可能需要增加数据库连接超时时间
	// 创建一个带超时的上下文
	backgroundCtx := context.Background()
	ctxWithTimeout, cancel := context.WithTimeout(backgroundCtx, 60*time.Second)
	defer cancel() // 确保资源释放

	// 使用带超时的上下文执行查询
	rows, err := model.DB.WithContext(ctxWithTimeout).Raw(query, args...).Rows()
	if err != nil {
		log.Printf("获取流量数据时发生错误: %v", err)
		return nil, err
	}
	defer rows.Close()

	// 使用批量处理来提高性能
	const batchSize = 1000 // 每批处理的数据量

	// 初始化结果切片，预分配空间以提高性能
	results = make([]model.TrafficResponse, 0, filter.Limit)

	// 批量计数器
	batchCount := 0
	totalCount := 0
	batchStartTime := time.Now()

	// 创建一个临时批次切片
	batch := make([]model.TrafficResponse, 0, batchSize)

	// 处理查询结果
	for rows.Next() {
		var result model.TrafficResponse

		// 直接使用时间类型扫描
		var createTime time.Time
		err := rows.Scan(&createTime, &result.SchoolID, &result.SchoolName, &result.Region, &result.CP, &result.TotalRecv, &result.TotalSend)
		if err != nil {
			log.Printf("扫描查询结果时出错: %v", err)
			continue
		}

		// 设置创建时间
		result.CreateTime = createTime

		// 计算总流量
		result.Total = result.TotalRecv + result.TotalSend

		// 添加到当前批次
		batch = append(batch, result)
		batchCount++
		totalCount++

		// 当批次达到指定大小时，将其添加到结果中
		if batchCount >= batchSize {
			// 将当前批次添加到结果中
			results = append(results, batch...)

			// 记录批处理时间
			batchDuration := time.Since(batchStartTime)
			log.Printf("处理了 %d 条数据，耗时 %.2f 秒", batchCount, batchDuration.Seconds())

			// 重置批次
			batch = make([]model.TrafficResponse, 0, batchSize)
			batchCount = 0
			batchStartTime = time.Now()
		}
	}

	// 处理最后一批不足batchSize的数据
	if len(batch) > 0 {
		results = append(results, batch...)
		log.Printf("处理最后一批 %d 条数据", len(batch))
	}

	// 按照创建时间排序结果
	if len(results) > 0 {
		sort.Slice(results, func(i, j int) bool {
			return results[i].CreateTime.After(results[j].CreateTime)
		})
	}

	// 限制返回结果数量
	if filter.Limit > 0 && len(results) > filter.Limit {
		results = results[:filter.Limit]
	}

	// 记录查询结果数量
	log.Printf("查询到 %d 条数据记录", len(results))

	// 如果没有数据，打印警告
	if len(results) == 0 {
		log.Printf("警告: 没有找到符合条件的数据，时间范围: %v 至 %v", filter.StartTime, filter.EndTime)
		// 检查数据库中是否有任何数据
		var count int64
		model.DB.Table("nfa_school_traffic").Count(&count)
		log.Printf("数据库中共有 %d 条数据记录", count)

		// 检查最早和最晚的数据时间
		var earliest, latest time.Time
		model.DB.Table("nfa_school_traffic").Select("MIN(create_time)").Row().Scan(&earliest)
		model.DB.Table("nfa_school_traffic").Select("MAX(create_time)").Row().Scan(&latest)
		log.Printf("数据库中最早的数据时间: %v", earliest)
		log.Printf("数据库中最晚的数据时间: %v", latest)
	}

	return results, nil
}

// GetTrafficSummary 获取流量汇总数据
func (r *schoolRepository) GetTrafficSummary(filter model.TrafficFilter) (model.TrafficResponse, error) {
	var result model.TrafficResponse

	// 构建查询
	query := model.DB.Table("nfa_school_traffic")

	// 应用过滤条件
	if !filter.StartTime.IsZero() {
		query = query.Where("create_time >= ?", filter.StartTime)
	}
	if !filter.EndTime.IsZero() {
		query = query.Where("create_time <= ?", filter.EndTime)
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

	// 计算总流量
	err := query.Select("SUM(total_recv) as total_recv, SUM(total_send) as total_send").Row().Scan(&result.TotalRecv, &result.TotalSend)
	if err != nil {
		return result, err
	}

	result.Total = result.TotalRecv + result.TotalSend
	return result, nil
}
