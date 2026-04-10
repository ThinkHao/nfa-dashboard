package repository

import (
	"context"
	"log"
	"strings"
	"time"

	"nfa-dashboard/internal/model"
	"gorm.io/gorm"
)

// SchoolRepository 学校数据仓库接口
type SchoolRepository interface {
	// 获取所有学校
	GetAllSchools(filter map[string]interface{}, limit, offset int) ([]model.School, int64, error)
	// 获取所有地区
	GetAllRegions() ([]string, error)
	// 获取所有运营商
	GetAllCPs() ([]string, error)
	// v2：按院校范围过滤的地区列表
	GetRegionsWithScope(allowedSchoolKeys []model.TrafficScopeSchoolKey) ([]string, error)
	// v2：按院校范围过滤的运营商列表
	GetCPsWithScope(allowedSchoolKeys []model.TrafficScopeSchoolKey) ([]string, error)
	// 根据过滤条件获取流量数据
	GetTrafficData(filter model.TrafficFilter) ([]model.TrafficResponse, error)
	// 获取流量汇总数据
	GetTrafficSummary(filter model.TrafficFilter) (model.TrafficResponse, error)
	// ExistsBySchoolID 检查学校是否存在
	ExistsBySchoolID(schoolID string) (bool, error)
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
	orderBy := "id ASC"

	// 应用过滤条件，优化查询性能
	for key, value := range filter {
		if key == "sort" {
			if sortValue, ok := value.(string); ok {
				switch strings.ToLower(strings.TrimSpace(sortValue)) {
				case "id_desc":
					orderBy = "id DESC"
				case "id_asc":
					orderBy = "id ASC"
				}
			}
			continue
		}
		// 特殊处理 allowed_school_keys（用于 v2 范围过滤）
		if key == "allowed_school_keys" {
			if keys, ok := value.([]model.TrafficScopeSchoolKey); ok && len(keys) > 0 {
				query = applyAllowedSchoolKeysQuery(query, keys)
			}
			continue
		}
		if value == nil {
			continue
		}
		// 仅对字符串类型按原逻辑处理
		if strValue, ok := value.(string); ok && strValue != "" {
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

	// 添加排序（默认 id ASC，v2 可选 id_desc / id_asc）
	query = query.Order(orderBy)

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

// GetRegionsWithScope v2：按院校范围过滤可见地区列表
func (r *schoolRepository) GetRegionsWithScope(allowedSchoolKeys []model.TrafficScopeSchoolKey) ([]string, error) {
	var regions []string
	q := model.DB.Model(&model.School{})
	if len(allowedSchoolKeys) > 0 {
		q = applyAllowedSchoolKeysQuery(q, allowedSchoolKeys)
	}
	if err := q.Distinct().Pluck("region", &regions).Error; err != nil {
		return nil, err
	}
	return regions, nil
}

// GetCPsWithScope v2：按院校范围过滤可见运营商列表
func (r *schoolRepository) GetCPsWithScope(allowedSchoolKeys []model.TrafficScopeSchoolKey) ([]string, error) {
	var cps []string
	q := model.DB.Model(&model.School{})
	if len(allowedSchoolKeys) > 0 {
		q = applyAllowedSchoolKeysQuery(q, allowedSchoolKeys)
	}
	if err := q.Distinct().Pluck("cp", &cps).Error; err != nil {
		return nil, err
	}
	return cps, nil
}

// GetTrafficData 根据过滤条件获取流量数据
func (r *schoolRepository) GetTrafficData(filter model.TrafficFilter) ([]model.TrafficResponse, error) {
	var results []model.TrafficResponse

	// 统一时间到本地时区，避免前端使用 RFC3339(Z/UTC) 传参时与库内 DATETIME(+08:00) 比较产生错位
	if !filter.StartTime.IsZero() {
		filter.StartTime = filter.StartTime.In(time.Local)
	}
	if !filter.EndTime.IsZero() {
		filter.EndTime = filter.EndTime.In(time.Local)
	}
	// 防御：如果时间范围反了，交换
	if !filter.StartTime.IsZero() && !filter.EndTime.IsZero() && filter.EndTime.Before(filter.StartTime) {
		filter.StartTime, filter.EndTime = filter.EndTime, filter.StartTime
	}

	// 限制查询时间范围，避免全表扫描
	if filter.StartTime.IsZero() && filter.EndTime.IsZero() {
		filter.EndTime = time.Now()
		filter.StartTime = filter.EndTime.AddDate(0, 0, -1)
	} else if filter.StartTime.IsZero() {
		filter.StartTime = filter.EndTime.AddDate(0, 0, -1)
	} else if filter.EndTime.IsZero() {
		filter.EndTime = time.Now()
	}

	// 始终使用原始 5 分钟粒度（不降采样），并在 SQL 侧按时间桶去重聚合。
	groupBy := []string{"create_time"}
	selectSchoolID := "'' AS school_id"
	selectSchoolName := "'' AS school_name"
	selectRegion := "'' AS region"
	selectCP := "'' AS cp"

	// school_name 存在时，保留 school+region+cp 维度，支持“同院校不同 CP”对比
	if filter.SchoolName != "" {
		groupBy = append(groupBy, "school_name", "region", "cp")
		selectSchoolID = "MIN(school_id) AS school_id"
		selectSchoolName = "school_name"
		selectRegion = "region"
		selectCP = "cp"
	} else {
		// 未指定 school_name 时，根据筛选项决定聚合维度，兼顾灵活查询与返回体积。
		if filter.Region != "" {
			groupBy = append(groupBy, "region")
			selectRegion = "region"
		}
		if filter.CP != "" {
			groupBy = append(groupBy, "cp")
			selectCP = "cp"
		}
	}

	query := `
        SELECT
            create_time,
            ` + selectSchoolID + `,
            ` + selectSchoolName + `,
            ` + selectRegion + `,
            ` + selectCP + `,
            SUM(total_recv) AS total_recv,
            SUM(total_send) AS total_send
        FROM nfa_school_traffic
        WHERE create_time >= ? AND create_time < ?`

	args := []interface{}{filter.StartTime, filter.EndTime}

	if filter.SchoolName != "" {
		if strings.ContainsAny(filter.SchoolName, "%_") {
			query += " AND school_name LIKE ?"
			args = append(args, filter.SchoolName)
		} else {
			query += " AND school_name = ?"
			args = append(args, filter.SchoolName)
		}
	}
	if filter.Region != "" {
		query += " AND region = ?"
		args = append(args, filter.Region)
	}
	if filter.CP != "" {
		query += " AND cp = ?"
		args = append(args, filter.CP)
	}
	if len(filter.AllowedSchoolKeys) > 0 {
		query, args = appendAllowedSchoolKeysSQL(query, args, filter.AllowedSchoolKeys)
	}

	query += " GROUP BY " + strings.Join(groupBy, ", ")
	query += " ORDER BY create_time ASC, region ASC, cp ASC, school_name ASC"

	log.Printf("[traffic.repo] query window start=%s end=%s region=%s cp=%s school=%s scope_source=%s allowed_school_count=%d",
		filter.StartTime.Format(time.RFC3339), filter.EndTime.Format(time.RFC3339), filter.Region, filter.CP, filter.SchoolName, filter.ScopeSource, len(filter.AllowedSchoolKeys))
	log.Printf("最终查询SQL: %s", query)
	log.Printf("查询参数: %v", args)

	backgroundCtx := context.Background()
	ctxWithTimeout, cancel := context.WithTimeout(backgroundCtx, 60*time.Second)
	defer cancel()

	rows, err := model.DB.WithContext(ctxWithTimeout).Raw(query, args...).Rows()
	if err != nil {
		log.Printf("获取流量数据时发生错误: %v", err)
		return nil, err
	}
	defer rows.Close()

	const batchSize = 1000
	results = make([]model.TrafficResponse, 0, 1024)
	batchCount := 0
	totalCount := 0
	batchStartTime := time.Now()
	batch := make([]model.TrafficResponse, 0, batchSize)

	for rows.Next() {
		var result model.TrafficResponse
		var createTime time.Time
		err := rows.Scan(&createTime, &result.SchoolID, &result.SchoolName, &result.Region, &result.CP, &result.TotalRecv, &result.TotalSend)
		if err != nil {
			log.Printf("扫描查询结果时出错: %v", err)
			continue
		}

		result.CreateTime = createTime
		result.Total = result.TotalRecv + result.TotalSend

		batch = append(batch, result)
		batchCount++
		totalCount++

		if batchCount >= batchSize {
			results = append(results, batch...)
			batchDuration := time.Since(batchStartTime)
			log.Printf("处理了 %d 条数据，耗时 %.2f 秒", batchCount, batchDuration.Seconds())
			batch = make([]model.TrafficResponse, 0, batchSize)
			batchCount = 0
			batchStartTime = time.Now()
		}
	}

	if len(batch) > 0 {
		results = append(results, batch...)
		log.Printf("处理最后一批 %d 条数据", len(batch))
	}

	log.Printf("查询到 %d 条数据记录(总处理行数: %d)", len(results), totalCount)
	if len(results) == 0 {
		log.Printf("警告: 没有找到符合条件的数据，时间范围: %v 至 %v", filter.StartTime, filter.EndTime)
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
	// v2：按范围过滤可见院校范围
	if len(filter.AllowedSchoolKeys) > 0 {
		query = applyAllowedSchoolKeysTrafficQuery(query, filter.AllowedSchoolKeys)
	}

	// 计算总流量
	err := query.Select("SUM(total_recv) as total_recv, SUM(total_send) as total_send").Row().Scan(&result.TotalRecv, &result.TotalSend)
	if err != nil {
		return result, err
	}

	result.Total = result.TotalRecv + result.TotalSend
	return result, nil
}

func (r *schoolRepository) ExistsBySchoolID(schoolID string) (bool, error) {
	if strings.TrimSpace(schoolID) == "" {
		return false, nil
	}
	var cnt int64
	if err := model.DB.Model(&model.School{}).Where("school_id = ?", schoolID).Count(&cnt).Error; err != nil {
		return false, err
	}
	return cnt > 0, nil
}

func applyAllowedSchoolKeysQuery(query *gorm.DB, keys []model.TrafficScopeSchoolKey) *gorm.DB {
	if len(keys) == 0 {
		return query
	}
	clauses := make([]string, 0, len(keys))
	args := make([]interface{}, 0, len(keys)*3)
	for _, key := range keys {
		clauses = append(clauses, "(school_id = ? AND region = ? AND cp = ?)")
		args = append(args, key.SchoolID, key.Region, key.CP)
	}
	return query.Where("("+strings.Join(clauses, " OR ")+")", args...)
}

func applyAllowedSchoolKeysTrafficQuery(query *gorm.DB, keys []model.TrafficScopeSchoolKey) *gorm.DB {
	if len(keys) == 0 {
		return query
	}
	clauses := make([]string, 0, len(keys))
	args := make([]interface{}, 0, len(keys)*3)
	for _, key := range keys {
		clauses = append(clauses, "(school_id = ? AND region = ? AND cp = ?)")
		args = append(args, key.SchoolID, key.Region, key.CP)
	}
	return query.Where("("+strings.Join(clauses, " OR ")+")", args...)
}

func appendAllowedSchoolKeysSQL(query string, args []interface{}, keys []model.TrafficScopeSchoolKey) (string, []interface{}) {
	if len(keys) == 0 {
		return query, args
	}
	clauses := make([]string, 0, len(keys))
	for _, key := range keys {
		clauses = append(clauses, "(school_id = ? AND region = ? AND cp = ?)")
		args = append(args, key.SchoolID, key.Region, key.CP)
	}
	query += " AND (" + strings.Join(clauses, " OR ") + ")"
	return query, args
}
