package controller

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"nfa-dashboard/internal/model"
	"nfa-dashboard/internal/service"
)

// SchoolController 学校控制器
type SchoolController struct {
	schoolService       service.SchoolService
	trafficScopeService service.TrafficScopeService
}

func parseTrafficTimeParam(raw string) (time.Time, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return time.Time{}, fmt.Errorf("time value is empty")
	}
	if t, err := time.Parse(time.RFC3339Nano, value); err == nil {
		return t, nil
	}
	if t, err := time.Parse(time.RFC3339, value); err == nil {
		return t, nil
	}
	if t, err := time.ParseInLocation("2006-01-02 15:04:05", value, time.Local); err == nil {
		return t, nil
	}
	return time.Time{}, fmt.Errorf("invalid time format: %s", value)
}

func validateTrafficTimeRange(start, end time.Time) error {
	if !start.Before(end) {
		return fmt.Errorf("start_time must be earlier than end_time")
	}
	return nil
}

// GetAllRegionsV2 获取所有地区（v2：按 user_id 过滤，普通用户强制为自身；管理员可查看全量或指定 user_id）
func (c *SchoolController) GetAllRegionsV2(ctx *gin.Context) {
	scope, err := c.resolveTrafficScope(ctx)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "解析流量可见范围失败", "error": err.Error()})
		return
	}
	if scope.Source == model.TrafficScopeSourceNone {
		ctx.JSON(http.StatusOK, gin.H{"code": 200, "message": "获取地区列表成功", "data": []string{}})
		return
	}
	regions, err := c.schoolService.GetRegionsWithScope(scope.AllowedSchoolKeys)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "获取地区列表失败", "error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"code": 200, "message": "获取地区列表成功", "data": regions})
}

// GetAllCPsV2 获取所有运营商（v2：按 user_id 过滤，普通用户强制为自身；管理员可查看全量或指定 user_id）
func (c *SchoolController) GetAllCPsV2(ctx *gin.Context) {
	scope, err := c.resolveTrafficScope(ctx)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "解析流量可见范围失败", "error": err.Error()})
		return
	}
	if scope.Source == model.TrafficScopeSourceNone {
		ctx.JSON(http.StatusOK, gin.H{"code": 200, "message": "获取运营商列表成功", "data": []string{}})
		return
	}
	cps, err := c.schoolService.GetCPsWithScope(scope.AllowedSchoolKeys)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "获取运营商列表失败", "error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"code": 200, "message": "获取运营商列表成功", "data": cps})
}

// GetTrafficDataV2 获取流量数据（v2：按 user_id 过滤，普通用户强制为自身）
func (c *SchoolController) GetTrafficDataV2(ctx *gin.Context) {
	var filter model.TrafficFilter
	// 解析时间参数（优先 RFC3339，兼容 YYYY-MM-DD HH:mm:ss）
	startTimeStr := ctx.Query("start_time")
	endTimeStr := ctx.Query("end_time")
	if startTimeStr != "" {
		startTime, err := parseTrafficTimeParam(startTimeStr)
		if err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{
				"code":    400,
				"message": "start_time 格式错误，应为 RFC3339 或 YYYY-MM-DD HH:mm:ss",
				"error":   err.Error(),
			})
			return
		}
		filter.StartTime = startTime
	}
	if endTimeStr != "" {
		endTime, err := parseTrafficTimeParam(endTimeStr)
		if err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{
				"code":    400,
				"message": "end_time 格式错误，应为 RFC3339 或 YYYY-MM-DD HH:mm:ss",
				"error":   err.Error(),
			})
			return
		}
		filter.EndTime = endTime
	}
	if !filter.StartTime.IsZero() && !filter.EndTime.IsZero() {
		if err := validateTrafficTimeRange(filter.StartTime, filter.EndTime); err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{
				"code":    400,
				"message": "时间范围非法：start_time 必须早于 end_time",
				"error":   err.Error(),
			})
			return
		}
	}
	// 其他过滤
	filter.SchoolName = ctx.Query("school_name")
	filter.Region = ctx.Query("region")
	filter.CP = ctx.Query("cp")
	filter.Interval = ctx.DefaultQuery("interval", "hour")
	if v := ctx.DefaultQuery("limit", "100"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			filter.Limit = n
		}
	}
	if v := ctx.DefaultQuery("offset", "0"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			filter.Offset = n
		}
	}
	if g := ctx.Query("granularity"); g != "" {
		filter.Granularity = g
	}

	scope, err := c.resolveTrafficScope(ctx)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "解析流量可见范围失败", "error": err.Error()})
		return
	}
	if scope.Source == model.TrafficScopeSourceNone {
		ctx.JSON(http.StatusOK, gin.H{"code": 200, "message": "获取流量数据成功，但没有符合条件的数据", "data": []interface{}{}, "scope_source": scope.Source})
		return
	}
	filter.AllowedSchoolIDs = scope.AllowedSchoolIDs
	filter.AllowedSchoolKeys = scope.AllowedSchoolKeys
	filter.ScopeSource = scope.Source
	log.Printf("[traffic.v2] parsed time range start=%s end=%s region=%s cp=%s school=%s scope_source=%s allowed_school_count=%d",
		filter.StartTime.Format(time.RFC3339), filter.EndTime.Format(time.RFC3339), filter.Region, filter.CP, filter.SchoolName, scope.Source, len(scope.AllowedSchoolKeys))

	// 调用服务
	trafficData, err := c.schoolService.GetTrafficData(filter)
	if err != nil {
		// 与 v1 行为一致：返回 200 + 空数组
		ctx.JSON(http.StatusOK, gin.H{"code": 200, "message": "获取流量数据成功，但没有符合条件的数据", "data": []interface{}{}, "warning": err.Error()})
		return
	}
	if len(trafficData) == 0 {
		ctx.JSON(http.StatusOK, gin.H{"code": 200, "message": "获取流量数据成功，但没有符合条件的数据", "data": []interface{}{}})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"code": 200, "message": "获取流量数据成功", "data": trafficData})
}

// GetTrafficSummaryV2 获取流量汇总数据（v2：按 user_id 过滤，普通用户强制为自身）
func (c *SchoolController) GetTrafficSummaryV2(ctx *gin.Context) {
	var filter model.TrafficFilter
	// 解析时间
	if s := ctx.Query("start_time"); s != "" {
		if t, err := time.Parse("2006-01-02 15:04:05", s); err == nil {
			filter.StartTime = t
		}
	}
	if s := ctx.Query("end_time"); s != "" {
		if t, err := time.Parse("2006-01-02 15:04:05", s); err == nil {
			filter.EndTime = t
		}
	}
	// 其他过滤
	filter.SchoolName = ctx.Query("school_name")
	filter.Region = ctx.Query("region")
	filter.CP = ctx.Query("cp")
	scope, err := c.resolveTrafficScope(ctx)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "解析流量可见范围失败", "error": err.Error()})
		return
	}
	if scope.Source == model.TrafficScopeSourceNone {
		ctx.JSON(http.StatusOK, gin.H{"code": 200, "message": "获取流量汇总数据成功", "data": model.TrafficResponse{}, "scope_source": scope.Source})
		return
	}
	filter.AllowedSchoolIDs = scope.AllowedSchoolIDs
	filter.AllowedSchoolKeys = scope.AllowedSchoolKeys
	filter.ScopeSource = scope.Source

	summary, err := c.schoolService.GetTrafficSummary(filter)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "获取流量汇总数据失败", "error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"code": 200, "message": "获取流量汇总数据成功", "data": summary})
}

// GetAllSchoolsV2 获取所有学校（v2：可按 user_id 过滤，普通用户强制为自身）
func (c *SchoolController) GetAllSchoolsV2(ctx *gin.Context) {
	// 查询参数
	schoolName := ctx.Query("school_name")
	region := ctx.Query("region")
	cp := ctx.Query("cp")
	sort := ctx.DefaultQuery("sort", "")
	// 分页
	limitStr := ctx.DefaultQuery("limit", "10")
	offsetStr := ctx.DefaultQuery("offset", "0")
	limit, _ := strconv.Atoi(limitStr)
	offset, _ := strconv.Atoi(offsetStr)

	scope, err := c.resolveTrafficScope(ctx)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "解析流量可见范围失败", "error": err.Error()})
		return
	}
	if scope.Source == model.TrafficScopeSourceNone {
		ctx.JSON(http.StatusOK, gin.H{"code": 200, "message": "获取学校列表成功", "data": gin.H{"total": 0, "items": []model.School{}, "limit": limit, "offset": offset}, "scope_source": scope.Source})
		return
	}
	schools, total, err := c.schoolService.GetAllSchoolsWithScope(schoolName, region, cp, sort, scope.AllowedSchoolKeys, limit, offset)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "获取学校列表失败", "error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"code": 200, "message": "获取学校列表成功", "data": gin.H{"total": total, "items": schools, "limit": limit, "offset": offset}, "scope_source": scope.Source})
}

// NewSchoolController 创建学校控制器实例
func NewSchoolController(schoolService service.SchoolService, trafficScopeService service.TrafficScopeService) *SchoolController {
	return &SchoolController{
		schoolService:       schoolService,
		trafficScopeService: trafficScopeService,
	}
}

func (c *SchoolController) resolveTrafficScope(ctx *gin.Context) (model.EffectiveTrafficScope, error) {
	if c.trafficScopeService == nil {
		return model.EffectiveTrafficScope{Source: model.TrafficScopeSourceNone, AllowedSchoolKeys: []model.TrafficScopeSchoolKey{}, AllowedSchoolIDs: []string{}}, nil
	}
	uid, ok := currentUserID(ctx)
	if !ok || uid == 0 {
		return model.EffectiveTrafficScope{Source: model.TrafficScopeSourceNone, AllowedSchoolKeys: []model.TrafficScopeSchoolKey{}, AllowedSchoolIDs: []string{}}, nil
	}
	return c.trafficScopeService.ResolveEffectiveScope(uid)
}

// GetAllSchools 获取所有学校
func (c *SchoolController) GetAllSchools(ctx *gin.Context) {
	// 获取查询参数
	schoolName := ctx.Query("school_name")
	region := ctx.Query("region")
	cp := ctx.Query("cp")

	// 获取分页参数
	limitStr := ctx.DefaultQuery("limit", "10")
	offsetStr := ctx.DefaultQuery("offset", "0")

	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		limit = 10
	}

	offset, err := strconv.Atoi(offsetStr)
	if err != nil {
		offset = 0
	}

	// 获取学校列表
	schools, total, err := c.schoolService.GetAllSchools(schoolName, region, cp, limit, offset)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取学校列表失败",
			"error":   err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "获取学校列表成功",
		"data": gin.H{
			"total":  total,
			"items":  schools,
			"limit":  limit,
			"offset": offset,
		},
	})
}

// GetAllRegions 获取所有地区
func (c *SchoolController) GetAllRegions(ctx *gin.Context) {
	regions, err := c.schoolService.GetAllRegions()
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取地区列表失败",
			"error":   err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "获取地区列表成功",
		"data":    regions,
	})
}

// GetAllCPs 获取所有运营商
func (c *SchoolController) GetAllCPs(ctx *gin.Context) {
	cps, err := c.schoolService.GetAllCPs()
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取运营商列表失败",
			"error":   err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "获取运营商列表成功",
		"data":    cps,
	})
}

// GetTrafficData 获取流量数据
func (c *SchoolController) GetTrafficData(ctx *gin.Context) {
	var filter model.TrafficFilter

	// 解析时间参数
	startTimeStr := ctx.Query("start_time")
	endTimeStr := ctx.Query("end_time")

	if startTimeStr != "" {
		// 尝试多种时间格式
		var startTime time.Time
		var err error

		// 尝试 ISO 8601 格式 (RFC3339)
		startTime, err = time.Parse(time.RFC3339, startTimeStr)
		if err != nil {
			// 尝试标准格式
			startTime, err = time.Parse("2006-01-02T15:04:05Z", startTimeStr)
			if err != nil {
				// 尝试传统格式
				startTime, err = time.Parse("2006-01-02 15:04:05", startTimeStr)
				if err != nil {
					// 记录解析错误
					ctx.Error(err)
					ctx.Set("error", "Invalid start_time format: "+startTimeStr)
				}
			}
		}

		if err == nil {
			filter.StartTime = startTime
			ctx.Set("parsed_start_time", startTime.Format(time.RFC3339))
		}
	}

	if endTimeStr != "" {
		// 尝试多种时间格式
		var endTime time.Time
		var err error

		// 尝试 ISO 8601 格式 (RFC3339)
		endTime, err = time.Parse(time.RFC3339, endTimeStr)
		if err != nil {
			// 尝试标准格式
			endTime, err = time.Parse("2006-01-02T15:04:05Z", endTimeStr)
			if err != nil {
				// 尝试传统格式
				endTime, err = time.Parse("2006-01-02 15:04:05", endTimeStr)
				if err != nil {
					// 记录解析错误
					ctx.Error(err)
					ctx.Set("error", "Invalid end_time format: "+endTimeStr)
				}
			}
		}

		if err == nil {
			filter.EndTime = endTime
			ctx.Set("parsed_end_time", endTime.Format(time.RFC3339))
		}
	}

	// 获取其他过滤参数
	filter.SchoolName = ctx.Query("school_name")
	filter.Region = ctx.Query("region")
	filter.CP = ctx.Query("cp")
	filter.Interval = ctx.DefaultQuery("interval", "hour")

	// 获取分页参数
	limitStr := ctx.DefaultQuery("limit", "100")
	offsetStr := ctx.DefaultQuery("offset", "0")

	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		limit = 100
	}

	// 计算时间范围间隔，根据时间范围自动调整数据点限制
	if !filter.StartTime.IsZero() && !filter.EndTime.IsZero() {
		// 计算时间间隔（小时）
		diffHours := filter.EndTime.Sub(filter.StartTime).Hours()

		// 根据时间范围自动调整限制
		// 每5分钟一个数据点，计算需要的数据点数量
		neededPoints := int(diffHours*60/5) + 1 // 每5分钟一个数据点

		// 设置最小限制为100，最大限制为10000
		if neededPoints > limit {
			limit = neededPoints
			if limit > 10000 {
				limit = 10000 // 设置一个合理的上限
			}
			ctx.Set("adjusted_limit", limit)
		}
	}

	filter.Limit = limit

	offset, err := strconv.Atoi(offsetStr)
	if err != nil {
		offset = 0
	}
	filter.Offset = offset

	// 获取granularity参数
	granularity := ctx.Query("granularity")
	if granularity != "" {
		filter.Granularity = granularity
		ctx.Set("granularity", granularity)
		log.Printf("使用前端指定的粒度: %s", granularity)
	}

	// 获取流量数据
	trafficData, err := c.schoolService.GetTrafficData(filter)
	if err != nil {
		// 记录错误但仍然返回空数组，而不是返回500错误
		log.Printf("获取流量数据时发生错误: %v", err)

		// 返回空数组而不是错误，避免前端崩溃
		ctx.JSON(http.StatusOK, gin.H{
			"code":    200,
			"message": "获取流量数据成功，但没有符合条件的数据",
			"data":    []interface{}{},
			"warning": err.Error(),
		})
		return
	}

	// 如果数据为空，返回空数组
	if len(trafficData) == 0 {
		ctx.JSON(http.StatusOK, gin.H{
			"code":    200,
			"message": "获取流量数据成功，但没有符合条件的数据",
			"data":    []interface{}{},
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "获取流量数据成功",
		"data":    trafficData,
	})
}

// GetTrafficSummary 获取流量汇总数据
func (c *SchoolController) GetTrafficSummary(ctx *gin.Context) {
	var filter model.TrafficFilter

	// 解析时间参数
	startTimeStr := ctx.Query("start_time")
	endTimeStr := ctx.Query("end_time")

	if startTimeStr != "" {
		startTime, err := time.Parse("2006-01-02 15:04:05", startTimeStr)
		if err == nil {
			filter.StartTime = startTime
		}
	}

	if endTimeStr != "" {
		endTime, err := time.Parse("2006-01-02 15:04:05", endTimeStr)
		if err == nil {
			filter.EndTime = endTime
		}
	}

	// 获取其他过滤参数
	filter.SchoolName = ctx.Query("school_name")
	filter.Region = ctx.Query("region")
	filter.CP = ctx.Query("cp")

	// 获取流量汇总数据
	trafficSummary, err := c.schoolService.GetTrafficSummary(filter)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取流量汇总数据失败",
			"error":   err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "获取流量汇总数据成功",
		"data":    trafficSummary,
	})
}
