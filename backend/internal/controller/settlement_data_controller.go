package controller

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"nfa-dashboard/internal/service"

	"github.com/gin-gonic/gin"
)

type SettlementDataController struct {
	dataSvc service.SettlementDataService
}

// ListUsedChannelOwners GET /api/v1/settlement/data/customer/channel-owners
// 返回在结算明细中实际被使用过的“渠道归属用户”（系统用户）去重列表
func (c *SettlementDataController) ListUsedChannelOwners(ctx *gin.Context) {
	filter := parseSettlementFilter(ctx)
	items, err := c.dataSvc.ListUsedChannelOwners(ctx.Request.Context(), filter)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			return
		}
		ctx.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "查询渠道归属用户失败", "error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"code": 200, "message": "OK", "data": gin.H{"items": items}})
}

// ListUsedOwnerSubjects GET /api/v1/settlement/data/customer/owner-subjects
// 统一返回费用归属主体的去重列表（仅 system user）
func (c *SettlementDataController) ListUsedOwnerSubjects(ctx *gin.Context) {
	filter := parseSettlementFilter(ctx)
	items, err := c.dataSvc.ListUsedOwnerSubjects(ctx.Request.Context(), filter)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			return
		}
		ctx.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "查询费用归属主体失败", "error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"code": 200, "message": "OK", "data": gin.H{"items": items}})
}

// ListUsedOwnerEntities GET /api/v1/settlement/data/customer/owners
// 返回在结算明细中实际被使用过的业务对象（费用归属：客户费/线路费/节点通用费）去重列表
func (c *SettlementDataController) ListUsedOwnerEntities(ctx *gin.Context) {
	filter := parseSettlementFilter(ctx)
	items, err := c.dataSvc.ListUsedOwnerEntities(ctx.Request.Context(), filter)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			return
		}
		ctx.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "查询费用归属对象失败", "error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"code": 200, "message": "OK", "data": gin.H{"items": items}})
}

func mapGetEntityName(m map[uint64]string, id *uint64) string {
	if id == nil {
		return ""
	}
	if v, ok := m[*id]; ok && v != "" {
		return v
	}
	return fmt.Sprintf("#%d", *id)
}
func mapGetUserName(m map[uint64]string, id *uint64) string {
	if id == nil {
		return ""
	}
	if v, ok := m[*id]; ok && v != "" {
		return v
	}
	return fmt.Sprintf("#%d", *id)
}

func NewSettlementDataController(svc service.SettlementDataService) *SettlementDataController {
	return &SettlementDataController{dataSvc: svc}
}

// ListCustomerData GET /api/v1/settlement/data/customer
func (c *SettlementDataController) ListCustomerData(ctx *gin.Context) {
	var (
		region   = ctx.Query("region")
		cp       = ctx.Query("cp")
		school   = ctx.Query("school_name")
		startS   = ctx.Query("start_service_date")
		endS     = ctx.Query("end_service_date")
		ownerS   = ctx.Query("owner_entity_id")
		channelS = ctx.Query("channel_owner_user_id")
		page     = intFrom(ctx.DefaultQuery("page", "1"), 1)
		size     = intFrom(ctx.DefaultQuery("page_size", "10"), 10)
	)
	var (
		startPtr *time.Time
		endPtr   *time.Time
	)
	if startS != "" {
		if parsed, err := parseOptionalDateBoundary(startS, false); err == nil {
			startPtr = parsed
		}
	}
	if endS != "" {
		if parsed, err := parseOptionalDateBoundary(endS, true); err == nil {
			endPtr = parsed
		}
	}
	var ownerPtr *uint64
	if ownerS != "" {
		if uv, err := strconv.ParseUint(ownerS, 10, 64); err == nil && uv > 0 {
			ownerPtr = &uv
		}
	}
	var channelPtr *uint64
	if channelS != "" {
		if cuv, err := strconv.ParseUint(channelS, 10, 64); err == nil && cuv > 0 {
			channelPtr = &cuv
		}
	}
	items, total, err := c.dataSvc.List(ctx.Request.Context(), service.SettlementCustomerFilter{Region: region, CP: cp, School: school, Start: startPtr, End: endPtr, OwnerEntityID: ownerPtr, ChannelOwnerUserID: channelPtr}, page, size)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			return
		}
		ctx.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "查询结算数据失败", "error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"code": 200, "message": "OK", "data": gin.H{"items": items, "total": total}})
}

// ListCustomerMonthlyData GET /api/v1/settlement/data/customer/monthly
func (c *SettlementDataController) ListCustomerMonthlyData(ctx *gin.Context) {
	var (
		region   = ctx.Query("region")
		cp       = ctx.Query("cp")
		school   = ctx.Query("school_name")
		startS   = ctx.Query("start_service_date")
		endS     = ctx.Query("end_service_date")
		ownerS   = ctx.Query("owner_entity_id")
		channelS = ctx.Query("channel_owner_user_id")
		page     = intFrom(ctx.DefaultQuery("page", "1"), 1)
		size     = intFrom(ctx.DefaultQuery("page_size", "10"), 10)
	)
	var (
		startPtr *time.Time
		endPtr   *time.Time
	)
	if startS != "" {
		if parsed, err := parseOptionalDateBoundary(startS, false); err == nil {
			startPtr = parsed
		}
	}
	if endS != "" {
		if parsed, err := parseOptionalDateBoundary(endS, true); err == nil {
			endPtr = parsed
		}
	}
	var ownerPtr *uint64
	if ownerS != "" {
		if uv, err := strconv.ParseUint(ownerS, 10, 64); err == nil && uv > 0 {
			ownerPtr = &uv
		}
	}
	var channelPtr *uint64
	if channelS != "" {
		if cuv, err := strconv.ParseUint(channelS, 10, 64); err == nil && cuv > 0 {
			channelPtr = &cuv
		}
	}
	items, total, err := c.dataSvc.ListMonthly(ctx.Request.Context(), service.SettlementCustomerFilter{
		Region:             region,
		CP:                 cp,
		School:             school,
		Start:              startPtr,
		End:                endPtr,
		OwnerEntityID:      ownerPtr,
		ChannelOwnerUserID: channelPtr,
	}, page, size)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			return
		}
		ctx.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "查询月度结算数据失败", "error": err.Error()})
		return
	}
	effectiveMonthRange := ""
	if startPtr != nil || endPtr != nil {
		effectiveMonthRange = fmt.Sprintf("%s~%s", formatYearMonthPtr(startPtr), formatYearMonthPtr(endPtr))
		ctx.Header("X-Effective-Month-Range", effectiveMonthRange)
	}
	// 无用户筛选时若本次结果回退到了实时聚合，后台异步增量回写对应月份，收敛口径差异
	if channelPtr == nil {
		needRebuild := false
		for _, it := range items {
			if strings.EqualFold(strings.TrimSpace(it.DataSource), "realtime") {
				needRebuild = true
				break
			}
		}
		if needRebuild {
			go func(s, e *time.Time) {
				_, _ = c.dataSvc.RebuildMonthlySnapshot(s, e)
			}(startPtr, endPtr)
		}
	}
	data := gin.H{"items": items, "total": total}
	if effectiveMonthRange != "" {
		data["effective_month_range"] = effectiveMonthRange
	}
	ctx.JSON(http.StatusOK, gin.H{"code": 200, "message": "OK", "data": data})
}

// ExportCustomerData GET /api/v1/settlement/data/customer/export
func (c *SettlementDataController) ExportCustomerData(ctx *gin.Context) {
	var (
		region   = ctx.Query("region")
		cp       = ctx.Query("cp")
		school   = ctx.Query("school_name")
		startS   = ctx.Query("start_service_date")
		endS     = ctx.Query("end_service_date")
		ownerS   = ctx.Query("owner_entity_id")
		channelS = ctx.Query("channel_owner_user_id")
	)
	var (
		startPtr *time.Time
		endPtr   *time.Time
	)
	if startS != "" {
		if parsed, err := parseOptionalDateBoundary(startS, false); err == nil {
			startPtr = parsed
		}
	}
	if endS != "" {
		if parsed, err := parseOptionalDateBoundary(endS, true); err == nil {
			endPtr = parsed
		}
	}
	var ownerPtr *uint64
	if ownerS != "" {
		if uv, err := strconv.ParseUint(ownerS, 10, 64); err == nil && uv > 0 {
			ownerPtr = &uv
		}
	}
	var channelPtr *uint64
	if channelS != "" {
		if cuv, err := strconv.ParseUint(channelS, 10, 64); err == nil && cuv > 0 {
			channelPtr = &cuv
		}
	}
	rows, err := c.dataSvc.ListAll(ctx.Request.Context(), service.SettlementCustomerFilter{Region: region, CP: cp, School: school, Start: startPtr, End: endPtr, OwnerEntityID: ownerPtr, ChannelOwnerUserID: channelPtr})
	if err != nil {
		if errors.Is(err, context.Canceled) {
			return
		}
		ctx.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "导出失败", "error": err.Error()})
		return
	}
	// 批量构建费用归属名称映射（业务对象与系统用户）
	entityMap, userMap, mapErr := c.dataSvc.BuildOwnerNameMaps(rows)
	if mapErr != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "导出失败", "error": mapErr.Error()})
		return
	}
	// CSV 导出
	ctx.Header("Content-Type", "text/csv; charset=utf-8")
	ctx.Header("Content-Disposition", "attachment; filename=settlement_customer.csv; filename*=UTF-8''%E7%BB%93%E7%AE%97%E6%95%B0%E6%8D%AE%E6%98%8E%E7%BB%86.csv")
	w := ctx.Writer
	_, _ = w.Write([]byte{0xEF, 0xBB, 0xBF})
	w.Write([]byte("区域,CP,学校,服务日期,客户费率,客户金额,线路费率,线路金额,渠道费率,渠道金额,客户费归属,线路费归属,渠道费归属,是否复算,最近复算时间\n"))
	for _, r := range rows {
		var sd, lrt string
		if r.ServiceDate != nil {
			sd = r.ServiceDate.Format("2006-01-02")
		}
		if r.LastRecalcTime != nil {
			lrt = r.LastRecalcTime.Format("2006-01-02 15:04:05")
		}
		writeCSVLine(w,
			r.Region, r.CP, r.SchoolName, sd,
			fmtFloat(r.CustomerFee), fmtFloat(r.CustomerBill),
			fmtFloat(r.NetworkLineFee), fmtFloat(r.NetworkLineBill),
			fmtFloat(r.ChannelRate), fmtFloat(r.ChannelBill),
			mapGetEntityName(entityMap, r.CustomerFeeOwnerID), mapGetEntityName(entityMap, r.NetworkLineFeeOwnerID), mapGetUserName(userMap, r.ChannelOwnerUserID),
			boolToCN(r.Recalculated), lrt,
		)
	}
}

// RecalculateCustomerData POST /api/v1/settlement/data/customer/recalculate
// 注：此为占位实现，仅更新标记，后续将补充实际重算逻辑（按服务年序分段计算并覆盖写入）
func (c *SettlementDataController) RecalculateCustomerData(ctx *gin.Context) {
	type Req struct {
		Region string `json:"region"`
		CP     string `json:"cp"`
		School string `json:"school_name"`
		Start  string `json:"start_service_date"` // YYYY-MM-DD
		End    string `json:"end_service_date"`   // YYYY-MM-DD
	}
	var req Req
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误", "error": err.Error()})
		return
	}
	if req.Start == "" || req.End == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "start_service_date 与 end_service_date 必填"})
		return
	}
	start, err := parseDateBoundary(req.Start, false)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "start_service_date 格式错误，应为 YYYY-MM-DD 或 YYYY-MM-DD HH:mm:ss", "error": err.Error()})
		return
	}
	end, err := parseDateBoundary(req.End, true)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "end_service_date 格式错误，应为 YYYY-MM-DD 或 YYYY-MM-DD HH:mm:ss", "error": err.Error()})
		return
	}
	taskID, err := c.dataSvc.CreateRecalculateTask(start, end)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "创建复算任务失败", "error": err.Error()})
		return
	}
	// 异步执行
	go func(tid int64, region, cp, school string, s, e time.Time) {
		filter := service.SettlementCustomerFilter{Region: region, CP: cp, School: school, Start: &s, End: &e}
		jobStart := time.Now()
		durations := map[string]int64{}

		stageStart := time.Now()
		total, _ := c.dataSvc.EstimateRecalculateTotal(filter)
		durations["estimate_ms"] = time.Since(stageStart).Milliseconds()
		_ = c.dataSvc.MarkTaskRunning(tid, total)
		_ = c.dataSvc.MarkTaskStage(tid, "recalculating", 0, map[string]interface{}{
			"total":        total,
			"durations_ms": durations,
		})

		stageStart = time.Now()
		affected, recErr := c.dataSvc.RecalculateWithProgress(filter, func(processed int64) {
			_ = c.dataSvc.MarkTaskProgress(tid, processed)
		})
		durations["recalculate_ms"] = time.Since(stageStart).Milliseconds()
		if recErr != nil {
			_ = c.dataSvc.MarkTaskStage(tid, "failed", -1, map[string]interface{}{
				"durations_ms": durations,
				"error_stage":  "recalculate",
			})
			_ = c.dataSvc.MarkTaskFailed(tid, recErr.Error())
			return
		}

		_ = c.dataSvc.MarkTaskStage(tid, "rebuilding_monthly", affected, map[string]interface{}{
			"durations_ms": durations,
		})
		stageStart = time.Now()
		if _, snapErr := c.dataSvc.RebuildMonthlySnapshot(&s, &e); snapErr != nil {
			durations["rebuild_monthly_ms"] = time.Since(stageStart).Milliseconds()
			durations["total_ms"] = time.Since(jobStart).Milliseconds()
			_ = c.dataSvc.MarkTaskStage(tid, "failed", affected, map[string]interface{}{
				"durations_ms": durations,
				"error_stage":  "rebuild_monthly",
			})
			_ = c.dataSvc.MarkTaskFailed(tid, "复算成功但月表回写失败: "+snapErr.Error())
			return
		}
		durations["rebuild_monthly_ms"] = time.Since(stageStart).Milliseconds()
		durations["total_ms"] = time.Since(jobStart).Milliseconds()
		_ = c.dataSvc.MarkTaskStage(tid, "finalizing", affected, map[string]interface{}{
			"durations_ms": durations,
		})
		_ = c.dataSvc.MarkTaskSuccess(tid, affected)
	}(taskID, req.Region, req.CP, req.School, start, end)

	ctx.JSON(http.StatusOK, gin.H{"code": 200, "message": "已创建复算任务", "data": gin.H{"task_id": taskID}})
}

// RebuildMonthlyData POST /api/v1/settlement/data/customer/monthly/rebuild
func (c *SettlementDataController) RebuildMonthlyData(ctx *gin.Context) {
	type Req struct {
		Start string `json:"start_service_date"` // YYYY-MM-DD
		End   string `json:"end_service_date"`   // YYYY-MM-DD
	}
	var req Req
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误", "error": err.Error()})
		return
	}
	var startPtr *time.Time
	var endPtr *time.Time
	if strings.TrimSpace(req.Start) != "" {
		t, err := parseDateBoundary(req.Start, false)
		if err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "start_service_date 格式错误，应为 YYYY-MM-DD 或 YYYY-MM-DD HH:mm:ss"})
			return
		}
		startPtr = &t
	}
	if strings.TrimSpace(req.End) != "" {
		t, err := parseDateBoundary(req.End, true)
		if err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "end_service_date 格式错误，应为 YYYY-MM-DD 或 YYYY-MM-DD HH:mm:ss"})
			return
		}
		endPtr = &t
	}
	affected, err := c.dataSvc.RebuildMonthlySnapshot(startPtr, endPtr)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "月度回写失败", "error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"code": 200, "message": "月度回写完成", "data": gin.H{"affected": affected}})
}

func parseSettlementFilter(ctx *gin.Context) service.SettlementCustomerFilter {
	filter := service.SettlementCustomerFilter{
		Region: ctx.Query("region"),
		CP:     ctx.Query("cp"),
		School: ctx.Query("school_name"),
	}
	if startS := ctx.Query("start_service_date"); startS != "" {
		if parsed, err := parseOptionalDateBoundary(startS, false); err == nil {
			filter.Start = parsed
		}
	}
	if endS := ctx.Query("end_service_date"); endS != "" {
		if parsed, err := parseOptionalDateBoundary(endS, true); err == nil {
			filter.End = parsed
		}
	}
	return filter
}

// 工具函数
func intFrom(s string, def int) int {
	if v, err := strconv.Atoi(s); err == nil {
		return v
	}
	return def
}

func formatYearMonthPtr(v *time.Time) string {
	if v == nil {
		return ""
	}
	return v.Format("2006-01")
}
func fmtFloat(p *float64) string {
	if p == nil {
		return ""
	}
	return strconv.FormatFloat(*p, 'f', -1, 64)
}
func fmtUint(p *uint64) string {
	if p == nil {
		return ""
	}
	return strconv.FormatUint(*p, 10)
}
func boolToCN(b bool) string {
	if b {
		return "是"
	}
	return "否"
}
func writeCSVLine(w http.ResponseWriter, cols ...string) {
	for i, c := range cols {
		if i > 0 {
			w.Write([]byte(","))
		}
		if strings.ContainsAny(c, ",\n\"") {
			w.Write([]byte("\"" + strings.ReplaceAll(c, "\"", "\"\"") + "\""))
		} else {
			w.Write([]byte(c))
		}
	}
	w.Write([]byte("\n"))
}
