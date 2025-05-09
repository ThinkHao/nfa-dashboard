package repository

import (
	"fmt"
	"log"
	"nfa-dashboard/internal/model"
	"sort"
	"time"
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

	// 记录原始时间范围，用于日志
	originalInterval := filter.Interval

	// 根据时间范围长度自动调整查询策略
	// 计算时间范围天数
	timeDays := timeRange.Hours() / 24

	// 检查是否指定了过滤条件（学校、地区或运营商）
	hasFilter := filter.SchoolName != "" || filter.Region != "" || filter.CP != ""

	// 如果没有指定任何过滤条件，强制使用更严格的限制
	if !hasFilter {
		// 计算时间范围差值（分钟）
		timeDiffMinutes := filter.EndTime.Sub(filter.StartTime).Minutes()

		// 如果时间范围小于3小时，使用原始数据点
		if timeDiffMinutes <= 180 { // 3小时 = 180分钟
			log.Printf("未指定过滤条件，但时间范围较短（%.2f分钟），使用原始数据点", timeDiffMinutes)
			filter.Interval = ""
			filter.Limit = int(timeDiffMinutes/5) + 10 // 每5分钟一个数据点，加上缓冲
		} else {
			// 当没有过滤条件且时间范围较长时，强制使用按天分组并限制返回数据量
			filter.Interval = "day"
			filter.Limit = 50 // 限制返回最多50条数据
			log.Printf("未指定过滤条件且时间范围较长，强制按天分组，限制返回最多50条数据")

			// 如果时间范围超过3天，则限制时间范围为最近3天
			if timeDays > 3 {
				filter.StartTime = filter.EndTime.AddDate(0, 0, -3)
				log.Printf("未指定过滤条件且时间范围超过3天，限制时间范围为最近3天")
			}
		}
	} else {
		// 有过滤条件时的处理
		// 计算时间范围差值（分钟）
		timeDiffMinutes := filter.EndTime.Sub(filter.StartTime).Minutes()

		// 如果时间范围小于6小时，使用原始数据点
		if timeDiffMinutes <= 360 { // 6小时 = 360分钟
			log.Printf("有过滤条件，且时间范围较短（%.2f分钟），使用原始数据点", timeDiffMinutes)
			filter.Interval = ""
			// 根据时间范围计算需要的数据点数量
			expectedPoints := int(timeDiffMinutes/5) + 10 // 每5分钟一个数据点，加上缓冲
			if expectedPoints > filter.Limit {
				filter.Limit = expectedPoints
				log.Printf("增加限制数量为%d，以确保返回足够的数据点", filter.Limit)
			}
		} else if timeDays > 30 {
			// 如果时间范围超过30天，强制使用按天分组
			filter.Interval = "day"
			// 限制返回数据量，每天只返回一个数据点
			filter.Limit = int(timeDays) + 10 // 每天一个数据点，加上一些缓冲
			log.Printf("时间范围超过30天（%.2f天），强制按天分组，限制返回%d条数据", timeDays, filter.Limit)
		} else if timeDays > 7 {
			// 如果时间范围超过7天，强制使用按小时分组
			filter.Interval = "hour"
			// 限制返回数据量，每小时只返回一个数据点
			filter.Limit = int(timeDays*24) + 10 // 每小时一个数据点，加上一些缓冲
			// 确保数据量不超过上限
			if filter.Limit > 500 {
				filter.Limit = 500
			}
			log.Printf("时间范围超过7天（%.2f天），强制按小时分组，限制返回%d条数据", timeDays, filter.Limit)
		} else {
			// 对于短时间范围，限制数据量不超过300条
			if filter.Limit > 300 {
				filter.Limit = 300
				log.Printf("短时间范围（%.2f天），限制返回最多300条数据", timeDays)
			}
		}
	}

	// 如果时间间隔发生了变化，记录日志
	if originalInterval != filter.Interval {
		log.Printf("时间间隔从 %s 调整为 %s，时间范围: %s 至 %s",
			originalInterval, filter.Interval, filter.StartTime.Format(time.RFC3339), filter.EndTime.Format(time.RFC3339))
	}

	// 直接查询符合条件的数据，不再分两步查询
	var query string
	var args []interface{}

	// 根据时间间隔决定查询方式
	var timeField string
	var needsGrouping bool = true

	// 计算时间范围差值（分钟）
	timeDiff := filter.EndTime.Sub(filter.StartTime).Minutes()

	// 如果时间范围小于6小时，强制使用原始数据点而不分组
	if timeDiff <= 360 { // 6小时 = 360分钟
		log.Printf("时间范围较短（%.2f分钟），强制使用原始数据点而不分组", timeDiff)
		timeField = "create_time"
		needsGrouping = false
		filter.Interval = "" // 清空时间间隔，确保不会被其他逻辑覆盖

		// 对于短时间范围，增加限制数量以确保返回足够的数据点
		expectedPoints := int(timeDiff/5) + 10 // 每5分钟一个点，加上一些缓冲
		if expectedPoints > filter.Limit {
			filter.Limit = expectedPoints
			log.Printf("增加限制数量为%d，以确保返回足够的数据点", filter.Limit)
		}
	} else {
		// 对于较长时间范围，使用分组
		switch filter.Interval {
		case "day":
			// 按天分组
			// 注意：返回字符串而不是时间类型，避免扫描错误
			timeField = "DATE_FORMAT(DATE(create_time), '%Y-%m-%d') as create_time"
			needsGrouping = true
		case "week":
			// 按周分组
			timeField = "DATE_FORMAT(DATE(DATE_ADD(create_time, INTERVAL(-WEEKDAY(create_time)) DAY)), '%Y-%m-%d') as create_time"
			needsGrouping = true
		case "month":
			// 按月分组
			timeField = "DATE_FORMAT(create_time, '%Y-%m-01') as create_time"
			needsGrouping = true
		case "hour":
			// 按小时分组
			timeField = "DATE_FORMAT(create_time, '%Y-%m-%d %H:00:00') as create_time"
			needsGrouping = true
		default:
			// 不分组，使用原始数据
			timeField = "create_time"
			needsGrouping = false
		}
	}

	// 根据是否需要分组构建不同的查询
	if needsGrouping {
		// 构建查询SQL
		query = fmt.Sprintf(`
			SELECT 
				%s,
				school_id,
				school_name,
				region,
				cp,
				SUM(total_recv) as total_recv,
				SUM(total_send) as total_send
			FROM nfa_school_traffic
			WHERE create_time BETWEEN ? AND ?
		`, timeField)
		args = []interface{}{filter.StartTime, filter.EndTime}
	} else {
		// 不分组，直接查询原始数据
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
			WHERE create_time BETWEEN ? AND ?
		`
		args = []interface{}{filter.StartTime, filter.EndTime}
	}

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

	// 添加分组和排序
	if needsGrouping {
		// 根据时间间隔添加分组字段
		// 注意：这里的分组字段必须与前面的timeField中的格式保持一致
		var groupByField string
		switch filter.Interval {
		case "day":
			groupByField = "DATE_FORMAT(DATE(create_time), '%Y-%m-%d')"
		case "week":
			groupByField = "DATE_FORMAT(DATE(DATE_ADD(create_time, INTERVAL(-WEEKDAY(create_time)) DAY)), '%Y-%m-%d')"
		case "month":
			groupByField = "DATE_FORMAT(create_time, '%Y-%m-01')"
		case "hour":
			groupByField = "DATE_FORMAT(create_time, '%Y-%m-%d %H:00:00')"
		default:
			groupByField = "DATE_FORMAT(create_time, '%Y-%m-%d %H:00:00')"
		}
		query += " GROUP BY " + groupByField + ", school_id, school_name, region, cp"
	}

	// 对于短时间范围查询，使用升序排序以确保数据点按时间顺序排列
	if !needsGrouping {
		query += " ORDER BY create_time ASC"
		log.Printf("短时间范围查询使用升序排序")
	} else {
		query += " ORDER BY create_time DESC"
	}

	// 限制返回数量
	query += " LIMIT ?"
	args = append(args, filter.Limit)

	// 执行查询
	rows, err := model.DB.Raw(query, args...).Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// 处理查询结果
	for rows.Next() {
		var result model.TrafficResponse

		// 添加错误处理和调试日志
		try := func() error {
			defer func() {
				if r := recover(); r != nil {
					log.Printf("处理查询结果时发生错误: %v", r)
				}
			}()

			// 根据是否分组使用不同的扫描方式
			var scanErr error
			if needsGrouping {
				// 分组查询返回的是字符串时间
				// 注意：这里的字段名必须与SQL查询中的别名一致，都是create_time
				var createTimeStr string
				scanErr = rows.Scan(&createTimeStr, &result.SchoolID, &result.SchoolName, &result.Region, &result.CP, &result.TotalRecv, &result.TotalSend)
				if scanErr == nil {
					// 尝试将字符串转换为时间
					var parseErr error
					var parsedTime time.Time

					// 尝试不同的时间格式
					formats := []string{
						"2006-01-02",
						"2006-01-02 15:04:05",
						"2006-01-02 15:04",
					}

					for _, format := range formats {
						parsedTime, parseErr = time.Parse(format, createTimeStr)
						if parseErr == nil {
							break
						}
					}

					if parseErr != nil {
						log.Printf("无法解析时间字符串 '%s': %v", createTimeStr, parseErr)
						// 使用当前时间作为后备
						parsedTime = time.Now()
					}

					result.CreateTime = parsedTime
				}
			} else {
				// 非分组查询返回的是时间类型
				var createTime time.Time
				scanErr = rows.Scan(&createTime, &result.SchoolID, &result.SchoolName, &result.Region, &result.CP, &result.TotalRecv, &result.TotalSend)
				if scanErr == nil {
					result.CreateTime = createTime
				}
			}

			if scanErr != nil {
				log.Printf("扫描查询结果时出错: %v", scanErr)
				return scanErr
			}
			return nil
		}

		if err := try(); err != nil {
			// 如果发生错误，跳过当前行，继续处理下一行
			continue
		}

		// 计算总流量
		result.Total = result.TotalRecv + result.TotalSend
		results = append(results, result)
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
