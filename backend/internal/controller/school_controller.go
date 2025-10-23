package controller

import (
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"nfa-dashboard/internal/model"
	"nfa-dashboard/internal/service"
)

// SchoolController 学校控制器
type SchoolController struct {
	schoolService service.SchoolService
}

// GetAllRegionsV2 获取所有地区（v2：按 user_id 过滤，普通用户强制为自身；管理员可查看全量或指定 user_id）
func (c *SchoolController) GetAllRegionsV2(ctx *gin.Context) {
    var reqUserID *uint64
    if v := ctx.Query("user_id"); v != "" { if uv, err := strconv.ParseUint(v, 10, 64); err == nil && uv > 0 { reqUserID = &uv } }
    if !hasAnyPermission(ctx, "system.user.manage") { if uid, ok := currentUserID(ctx); ok { reqUserID = &uid } }
    regions, err := c.schoolService.GetRegionsWithUser(reqUserID)
    if err != nil {
        ctx.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "获取地区列表失败", "error": err.Error()})
        return
    }
    ctx.JSON(http.StatusOK, gin.H{"code": 200, "message": "获取地区列表成功", "data": regions})
}

// GetAllCPsV2 获取所有运营商（v2：按 user_id 过滤，普通用户强制为自身；管理员可查看全量或指定 user_id）
func (c *SchoolController) GetAllCPsV2(ctx *gin.Context) {
    var reqUserID *uint64
    if v := ctx.Query("user_id"); v != "" { if uv, err := strconv.ParseUint(v, 10, 64); err == nil && uv > 0 { reqUserID = &uv } }
    if !hasAnyPermission(ctx, "system.user.manage") { if uid, ok := currentUserID(ctx); ok { reqUserID = &uid } }
    cps, err := c.schoolService.GetCPsWithUser(reqUserID)
    if err != nil {
        ctx.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "获取运营商列表失败", "error": err.Error()})
        return
    }
    ctx.JSON(http.StatusOK, gin.H{"code": 200, "message": "获取运营商列表成功", "data": cps})
}

// GetTrafficDataV2 获取流量数据（v2：按 user_id 过滤，普通用户强制为自身）
func (c *SchoolController) GetTrafficDataV2(ctx *gin.Context) {
    var filter model.TrafficFilter
    // 解析时间参数（与 v1 保持一致）
    startTimeStr := ctx.Query("start_time")
    endTimeStr := ctx.Query("end_time")
    if startTimeStr != "" {
        var startTime time.Time
        var err error
        startTime, err = time.Parse(time.RFC3339, startTimeStr)
        if err != nil {
            startTime, err = time.Parse("2006-01-02T15:04:05Z", startTimeStr)
            if err != nil { startTime, _ = time.Parse("2006-01-02 15:04:05", startTimeStr) }
        }
        if !startTime.IsZero() { filter.StartTime = startTime }
    }
    if endTimeStr != "" {
        var endTime time.Time
        var err error
        endTime, err = time.Parse(time.RFC3339, endTimeStr)
        if err != nil {
            endTime, err = time.Parse("2006-01-02T15:04:05Z", endTimeStr)
            if err != nil { endTime, _ = time.Parse("2006-01-02 15:04:05", endTimeStr) }
        }
        if !endTime.IsZero() { filter.EndTime = endTime }
    }
    // 其他过滤
    filter.SchoolName = ctx.Query("school_name")
    filter.Region = ctx.Query("region")
    filter.CP = ctx.Query("cp")
    filter.Interval = ctx.DefaultQuery("interval", "hour")
    if v := ctx.DefaultQuery("limit", "100"); v != "" { if n, err := strconv.Atoi(v); err == nil { filter.Limit = n } }
    if v := ctx.DefaultQuery("offset", "0"); v != "" { if n, err := strconv.Atoi(v); err == nil { filter.Offset = n } }
    if g := ctx.Query("granularity"); g != "" { filter.Granularity = g }

    // v2：处理 user_id
    var reqUserID *uint64
    if v := ctx.Query("user_id"); v != "" { if uv, err := strconv.ParseUint(v, 10, 64); err == nil && uv > 0 { reqUserID = &uv } }
    if !hasAnyPermission(ctx, "system.user.manage") { if uid, ok := currentUserID(ctx); ok { reqUserID = &uid } }
    filter.UserID = reqUserID

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
    if s := ctx.Query("start_time"); s != "" { if t, err := time.Parse("2006-01-02 15:04:05", s); err == nil { filter.StartTime = t } }
    if s := ctx.Query("end_time"); s != "" { if t, err := time.Parse("2006-01-02 15:04:05", s); err == nil { filter.EndTime = t } }
    // 其他过滤
    filter.SchoolName = ctx.Query("school_name")
    filter.Region = ctx.Query("region")
    filter.CP = ctx.Query("cp")
    // v2：user_id
    var reqUserID *uint64
    if v := ctx.Query("user_id"); v != "" { if uv, err := strconv.ParseUint(v, 10, 64); err == nil && uv > 0 { reqUserID = &uv } }
    if !hasAnyPermission(ctx, "system.user.manage") { if uid, ok := currentUserID(ctx); ok { reqUserID = &uid } }
    filter.UserID = reqUserID

    summary, err := c.schoolService.GetTrafficSummary(filter)
    if err != nil { ctx.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "获取流量汇总数据失败", "error": err.Error()}); return }
    ctx.JSON(http.StatusOK, gin.H{"code": 200, "message": "获取流量汇总数据成功", "data": summary})
}

// GetAllSchoolsV2 获取所有学校（v2：可按 user_id 过滤，普通用户强制为自身）
func (c *SchoolController) GetAllSchoolsV2(ctx *gin.Context) {
    // 查询参数
    schoolName := ctx.Query("school_name")
    region := ctx.Query("region")
    cp := ctx.Query("cp")
    // user_id 可选（仅特权用户可自定义），普通用户将被覆盖
    var reqUserID *uint64
    if v := ctx.Query("user_id"); v != "" {
        if uv, err := strconv.ParseUint(v, 10, 64); err == nil && uv > 0 {
            reqUserID = &uv
        }
    }

    // 分页
    limitStr := ctx.DefaultQuery("limit", "10")
    offsetStr := ctx.DefaultQuery("offset", "0")
    limit, _ := strconv.Atoi(limitStr)
    offset, _ := strconv.Atoi(offsetStr)

    // 权限判断：无管理权限则强制使用自身 user_id
    // 选用较高权限作为“特权”：system.user.manage
    if !hasAnyPermission(ctx, "system.user.manage") {
        if uid, ok := currentUserID(ctx); ok {
            reqUserID = &uid
        }
    }

    schools, total, err := c.schoolService.GetAllSchoolsWithUser(schoolName, region, cp, reqUserID, limit, offset)
    if err != nil {
        ctx.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "获取学校列表失败", "error": err.Error()})
        return
    }
    ctx.JSON(http.StatusOK, gin.H{"code": 200, "message": "获取学校列表成功", "data": gin.H{"total": total, "items": schools, "limit": limit, "offset": offset}})
}

// NewSchoolController 创建学校控制器实例
func NewSchoolController(schoolService service.SchoolService) *SchoolController {
	return &SchoolController{
		schoolService: schoolService,
	}
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
		neededPoints := int(diffHours * 60 / 5) + 1 // 每5分钟一个数据点
		
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
