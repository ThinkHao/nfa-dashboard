package controller

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"nfa-dashboard/internal/model"
	"nfa-dashboard/internal/service"

	"github.com/gin-gonic/gin"
)

// SettlementController 结算控制器
type SettlementController struct {
	settlementService service.SettlementService
}

// GetDailySettlementDetailsV2 获取日95明细数据列表（v2：按 user_id 过滤，普通用户强制为自身）
func (c *SettlementController) GetDailySettlementDetailsV2(ctx *gin.Context) {
    var filter model.SettlementFilter

    // 查询参数
    startDateStr := ctx.Query("start_date")
    endDateStr := ctx.Query("end_date")
    filter.SchoolID = ctx.Query("school_id")
    filter.Region = ctx.Query("region")
    filter.CP = ctx.Query("cp")
    limitStr := ctx.DefaultQuery("limit", "10")
    offsetStr := ctx.DefaultQuery("offset", "0")

    // 解析日期
    if startDateStr != "" { if t, err := time.Parse("2006-01-02", startDateStr); err == nil { filter.StartDate = t } }
    if endDateStr != "" { if t, err := time.Parse("2006-01-02", endDateStr); err == nil { filter.EndDate = t } }

    // 分页
    if n, err := strconv.Atoi(limitStr); err == nil { filter.Limit = n } else { filter.Limit = 10 }
    if n, err := strconv.Atoi(offsetStr); err == nil { filter.Offset = n } else { filter.Offset = 0 }

    // v2：user_id 解析与权限覆盖
    var reqUserID *uint64
    if v := ctx.Query("user_id"); v != "" { if uv, err := strconv.ParseUint(v, 10, 64); err == nil && uv > 0 { reqUserID = &uv } }
    if !hasAnyPermission(ctx, "system.user.manage") { if uid, ok := currentUserID(ctx); ok { reqUserID = &uid } }
    filter.UserID = reqUserID

    // 调用服务
    details, total, err := c.settlementService.GetDailySettlementDetails(filter)
    if err != nil {
        ctx.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "获取日95明细数据列表失败", "error": err.Error()})
        return
    }
    ctx.JSON(http.StatusOK, gin.H{"code": 200, "message": "获取日95明细数据列表成功", "data": gin.H{"total": total, "items": details}})
}

// GetSettlementsV2 获取结算数据列表（v2：按 user_id 过滤，普通用户强制为自身）
func (c *SettlementController) GetSettlementsV2(ctx *gin.Context) {
    var filter model.SettlementFilter

    // 获取查询参数
    startDateStr := ctx.Query("start_date")
    endDateStr := ctx.Query("end_date")
    filter.SchoolID = ctx.Query("school_id")
    filter.SchoolName = ctx.Query("school_name")
    filter.Region = ctx.Query("region")
    filter.CP = ctx.Query("cp")
    limitStr := ctx.DefaultQuery("limit", "10")
    offsetStr := ctx.DefaultQuery("offset", "0")

    // 解析日期
    if startDateStr != "" { if t, err := time.Parse("2006-01-02", startDateStr); err == nil { filter.StartDate = t } }
    if endDateStr != "" { if t, err := time.Parse("2006-01-02", endDateStr); err == nil { filter.EndDate = t } }

    // 分页
    if n, err := strconv.Atoi(limitStr); err == nil { filter.Limit = n } else { filter.Limit = 10 }
    if n, err := strconv.Atoi(offsetStr); err == nil { filter.Offset = n } else { filter.Offset = 0 }

    // v2：user_id 解析与权限覆盖
    var reqUserID *uint64
    if v := ctx.Query("user_id"); v != "" { if uv, err := strconv.ParseUint(v, 10, 64); err == nil && uv > 0 { reqUserID = &uv } }
    if !hasAnyPermission(ctx, "system.user.manage") { if uid, ok := currentUserID(ctx); ok { reqUserID = &uid } }
    filter.UserID = reqUserID

    settlements, total, err := c.settlementService.GetSettlements(filter)
    if err != nil {
        ctx.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "获取结算数据列表失败", "error": err.Error()})
        return
    }
    ctx.JSON(http.StatusOK, gin.H{"code": 200, "message": "获取结算数据列表成功", "data": gin.H{"total": total, "items": settlements}})
}

// NewSettlementController 创建结算控制器实例
func NewSettlementController(settlementService service.SettlementService) *SettlementController {
	return &SettlementController{
		settlementService: settlementService,
	}
}

// GetSettlementConfig 获取结算配置
func (c *SettlementController) GetSettlementConfig(ctx *gin.Context) {
	config, err := c.settlementService.GetSettlementConfig()
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取结算配置失败",
			"error":   err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "获取结算配置成功",
		"data":    config,
	})
}

// UpdateSettlementConfig 更新结算配置
func (c *SettlementController) UpdateSettlementConfig(ctx *gin.Context) {
	var config model.SettlementConfig
	if err := ctx.ShouldBindJSON(&config); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "请求参数错误",
			"error":   err.Error(),
		})
		return
	}

	// 验证时间格式
	if len(config.DailyTime) != 5 || len(config.WeeklyTime) != 5 {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "时间格式错误，应为HH:MM格式",
		})
		return
	}

	// 验证周几的值
	if config.WeeklyDay < 1 || config.WeeklyDay > 7 {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "周几的值应为1-7",
		})
		return
	}

	err := c.settlementService.UpdateSettlementConfig(&config)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "更新结算配置失败",
			"error":   err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "更新结算配置成功",
		"data":    config,
	})
}

// GetSettlementTasks 获取结算任务列表
func (c *SettlementController) GetSettlementTasks(ctx *gin.Context) {
	// 获取查询参数
	taskType := ctx.Query("task_type")
	status := ctx.Query("status")
	startDateStr := ctx.Query("start_date")
	endDateStr := ctx.Query("end_date")
	limitStr := ctx.DefaultQuery("limit", "10")
	offsetStr := ctx.DefaultQuery("offset", "0")

	// 解析日期
	var startDate, endDate time.Time
	var err error
	if startDateStr != "" {
		startDate, err = time.Parse("2006-01-02", startDateStr)
		if err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{
				"code":    400,
				"message": "开始日期格式错误，应为YYYY-MM-DD",
				"error":   err.Error(),
			})
			return
		}
	}

	if endDateStr != "" {
		endDate, err = time.Parse("2006-01-02", endDateStr)
		if err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{
				"code":    400,
				"message": "结束日期格式错误，应为YYYY-MM-DD",
				"error":   err.Error(),
			})
			return
		}
	}

	// 解析分页参数
	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		limit = 10
	}

	offset, err := strconv.Atoi(offsetStr)
	if err != nil {
		offset = 0
	}

	// 获取任务列表
	tasks, total, err := c.settlementService.GetSettlementTasks(taskType, status, startDate, endDate, limit, offset)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取结算任务列表失败",
			"error":   err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "获取结算任务列表成功",
		"data": gin.H{
			"total": total,
			"items": tasks,
		},
	})
}

// GetSettlementTaskByID 获取结算任务详情
func (c *SettlementController) GetSettlementTaskByID(ctx *gin.Context) {
	// 获取任务ID
	idStr := ctx.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "任务ID格式错误",
			"error":   err.Error(),
		})
		return
	}

	// 获取任务详情
	task, err := c.settlementService.GetSettlementTaskByID(id)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取结算任务详情失败",
			"error":   err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "获取结算任务详情成功",
		"data":    task,
	})
}

// DeleteSettlementTask 删除结算任务
func (c *SettlementController) DeleteSettlementTask(ctx *gin.Context) {
	// 从路径参数中获取任务ID
	idStr := ctx.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "无效的任务ID",
			"error":   err.Error(),
		})
		return
	}

	// 删除任务
	err = c.settlementService.DeleteSettlementTask(id)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "删除结算任务失败",
			"error":   err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "删除结算任务成功",
	})
}

// GetSettlements 获取结算数据列表
func (c *SettlementController) GetSettlements(ctx *gin.Context) {
	var filter model.SettlementFilter

	// 获取查询参数
	startDateStr := ctx.Query("start_date")
	endDateStr := ctx.Query("end_date")
	filter.SchoolID = ctx.Query("school_id")     // 添加学校ID参数
	filter.SchoolName = ctx.Query("school_name")
	filter.Region = ctx.Query("region")
	filter.CP = ctx.Query("cp")

	limitStr := ctx.DefaultQuery("limit", "10")
	offsetStr := ctx.DefaultQuery("offset", "0")

	// 解析日期
	var err error
	if startDateStr != "" {
		filter.StartDate, err = time.Parse("2006-01-02", startDateStr)
		if err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{
				"code":    400,
				"message": "开始日期格式错误，应为YYYY-MM-DD",
				"error":   err.Error(),
			})
			return
		}
	}

	if endDateStr != "" {
		filter.EndDate, err = time.Parse("2006-01-02", endDateStr)
		if err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{
				"code":    400,
				"message": "结束日期格式错误，应为YYYY-MM-DD",
				"error":   err.Error(),
			})
			return
		}
	}

	// 解析分页参数
	filter.Limit, err = strconv.Atoi(limitStr)
	if err != nil {
		filter.Limit = 10
	}

	filter.Offset, err = strconv.Atoi(offsetStr)
	if err != nil {
		filter.Offset = 0
	}

	// 获取结算数据列表
	settlements, total, err := c.settlementService.GetSettlements(filter)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取结算数据列表失败",
			"error":   err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "获取结算数据列表成功",
		"data": gin.H{
			"total": total,
			"items": settlements,
		},
	})
}

// GetDailySettlementDetails 获取日95明细数据列表
func (c *SettlementController) GetDailySettlementDetails(ctx *gin.Context) {
    var filter model.SettlementFilter // 可以复用 SettlementFilter，或者为其创建一个新的类型

    // 获取查询参数
    startDateStr := ctx.Query("start_date")
    endDateStr := ctx.Query("end_date")
    filter.SchoolID = ctx.Query("school_id")
    filter.Region = ctx.Query("region")
    filter.CP = ctx.Query("cp")

    limitStr := ctx.DefaultQuery("limit", "10")
    offsetStr := ctx.DefaultQuery("offset", "0")

    // 解析日期
    var err error
    if startDateStr != "" {
        filter.StartDate, err = time.Parse("2006-01-02", startDateStr)
        if err != nil {
            ctx.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "开始日期格式错误，应为YYYY-MM-DD", "error": err.Error()})
            return
        }
    }
    if endDateStr != "" {
        filter.EndDate, err = time.Parse("2006-01-02", endDateStr)
        if err != nil {
            ctx.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "结束日期格式错误，应为YYYY-MM-DD", "error": err.Error()})
            return
        }
    }

    // 解析分页参数
    filter.Limit, err = strconv.Atoi(limitStr)
    if err != nil { filter.Limit = 10 }
    filter.Offset, err = strconv.Atoi(offsetStr)
    if err != nil { filter.Offset = 0 }

    // 获取日95明细数据列表
    dailyDetails, total, err := c.settlementService.GetDailySettlementDetails(filter)
    if err != nil {
        ctx.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "获取日95明细数据列表失败", "error": err.Error()})
        return
    }
    ctx.JSON(http.StatusOK, gin.H{"code": 200, "message": "获取日95明细数据列表成功", "data": gin.H{"total": total, "items": dailyDetails}})
}

// CreateDailySettlementTask 创建日结算任务
func (c *SettlementController) CreateDailySettlementTask(ctx *gin.Context) {
    // 获取日期参数
    dateStr := ctx.DefaultQuery("date", "")
    var settlementDate time.Time
    var err error

	if dateStr == "" {
		// 默认计算前一天的数据
		yesterday := time.Now().AddDate(0, 0, -1)
		// 设置为前一天的零点
		settlementDate = time.Date(yesterday.Year(), yesterday.Month(), yesterday.Day(), 0, 0, 0, 0, yesterday.Location())
	} else {
		// 解析日期字符串
		parsedDate, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{
				"code":    400,
				"message": "日期格式错误，应为YYYY-MM-DD",
				"error":   err.Error(),
			})
			return
		}
		// 设置为指定日期的零点
		settlementDate = time.Date(parsedDate.Year(), parsedDate.Month(), parsedDate.Day(), 0, 0, 0, 0, parsedDate.Location())
	}

	// 创建结算任务
	task, err := c.settlementService.CreateSettlementTask("daily", settlementDate)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "创建日结算任务失败",
			"error":   err.Error(),
		})
		return
	}

	// 在返回前等待一下，确保数据库操作完成
	time.Sleep(100 * time.Millisecond)

	// 异步执行结算任务
	go func() {
		err := c.settlementService.ExecuteDailySettlement(task.ID, settlementDate)
		if err != nil {
			log.Printf("执行日结算任务失败: %v", err)
		}
	}()

	ctx.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "创建日结算任务成功",
		"data":    task,
	})
}

// CreateWeeklySettlementTask 创建周结算任务
func (c *SettlementController) CreateWeeklySettlementTask(ctx *gin.Context) {
	// 从请求体中获取参数
	type TaskParams struct {
		StartDate string `json:"start_date"`
		EndDate   string `json:"end_date"`
	}
	
	var params TaskParams
	if err := ctx.ShouldBindJSON(&params); err != nil {
		// 如果解析JSON失败，尝试从查询参数获取
		params.StartDate = ctx.DefaultQuery("start_date", "")
		params.EndDate = ctx.DefaultQuery("end_date", "")
	}
	
	log.Printf("接收到周结算任务参数: start_date=%s, end_date=%s", params.StartDate, params.EndDate)
	
	var startDate, endDate time.Time
	var err error

	// 处理开始日期
	if params.StartDate == "" {
		// 默认计算上一周的数据（从上周一开始）
		now := time.Now()
		daysToLastMonday := (int(now.Weekday()) + 6) % 7
		if daysToLastMonday == 0 {
			daysToLastMonday = 7
		}
		startDate = now.AddDate(0, 0, -daysToLastMonday-7)
	} else {
		startDate, err = time.Parse("2006-01-02", params.StartDate)
		if err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{
				"code":    400,
				"message": "开始日期格式错误，应为YYYY-MM-DD",
				"error":   err.Error(),
			})
			return
		}
	}
	
	// 处理结束日期
	if params.EndDate == "" {
		// 默认为开始日期后的6天（周日）
		endDate = startDate.AddDate(0, 0, 6)
	} else {
		endDate, err = time.Parse("2006-01-02", params.EndDate)
		if err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{
				"code":    400,
				"message": "结束日期格式错误，应为YYYY-MM-DD",
				"error":   err.Error(),
			})
			return
		}
	}
	
	// 检查日期范围是否有效
	if endDate.Before(startDate) {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "结束日期不能早于开始日期",
		})
		return
	}

	// 将日期范围信息存储在任务的错误信息字段中（临时存储，不影响实际使用）
	dateRangeInfo := fmt.Sprintf("%s,%s", startDate.Format("2006-01-02"), endDate.Format("2006-01-02"))
	
	// 创建结算任务，使用开始日期作为任务日期
	task, err := c.settlementService.CreateSettlementTask("weekly", startDate)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "创建周结算任务失败",
			"error":   err.Error(),
		})
		return
	}
	
	// 更新任务信息，将日期范围信息保存到任务中
	c.settlementService.UpdateSettlementTaskStatus(task.ID, "pending", dateRangeInfo)

	// 异步执行结算任务
	go func() {
		// 使用开始日期和结束日期执行周结算
		err := c.settlementService.ExecuteWeeklySettlementWithDateRange(task.ID, startDate, endDate)
		if err != nil {
			log.Printf("执行周结算任务失败: %v", err)
		}
	}()

	ctx.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "创建周结算任务成功",
		"data":    task,
	})
}
