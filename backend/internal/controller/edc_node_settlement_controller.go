package controller

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"nfa-dashboard/internal/service"
)

type EDCNodeSettlementController struct {
	settlementSvc service.SettlementService
	nodeSvc       service.EDCNodeSettlementService
}

func NewEDCNodeSettlementController(settlementSvc service.SettlementService, nodeSvc service.EDCNodeSettlementService) *EDCNodeSettlementController {
	return &EDCNodeSettlementController{settlementSvc: settlementSvc, nodeSvc: nodeSvc}
}

func (c *EDCNodeSettlementController) CreateNodeDailyTask(ctx *gin.Context) {
	var req struct {
		Date      string `json:"date"`
		StartDate string `json:"start_date"`
		EndDate   string `json:"end_date"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"message": "invalid request"})
		return
	}
	startText := req.StartDate
	if startText == "" {
		startText = req.Date
	}
	start, err := parseNodeTaskDate(startText, time.Now().AddDate(0, 0, -1))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"message": "invalid date, expect YYYY-MM-DD"})
		return
	}
	endText := req.EndDate
	if endText == "" {
		endText = startText
	}
	end, err := parseNodeTaskDate(endText, start)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"message": "invalid date, expect YYYY-MM-DD"})
		return
	}
	if end.Before(start) {
		ctx.JSON(http.StatusBadRequest, gin.H{"message": "end_date must not be before start_date"})
		return
	}
	hasTraffic, err := c.nodeSvc.HasSettlementTraffic(start, end.AddDate(0, 0, 1))
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		return
	}
	if !hasTraffic {
		ctx.JSON(http.StatusBadRequest, gin.H{"message": "任务周期内没有可结算的 EDC 节点日95流量数据"})
		return
	}
	task, err := c.settlementSvc.CreateSettlementTask("node_daily95", start)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		return
	}
	go func() { _ = c.nodeSvc.ExecuteDailyRangeTask(task.ID, start, end) }()
	ctx.JSON(http.StatusOK, gin.H{"task": task})
}

func (c *EDCNodeSettlementController) CreateNodeMonthlyTask(ctx *gin.Context) {
	var req struct {
		Month      string `json:"month"`
		Date       string `json:"date"`
		StartMonth string `json:"start_month"`
		EndMonth   string `json:"end_month"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"message": "invalid request"})
		return
	}
	startText := req.StartMonth
	if startText == "" {
		startText = req.Month
	}
	if startText == "" {
		startText = req.Date
	}
	start, err := parseNodeTaskMonth(startText, time.Now().AddDate(0, -1, 0))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"message": "invalid month, expect YYYY-MM"})
		return
	}
	endText := req.EndMonth
	if endText == "" {
		endText = startText
	}
	end, err := parseNodeTaskMonth(endText, start)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"message": "invalid month, expect YYYY-MM"})
		return
	}
	if end.Before(start) {
		ctx.JSON(http.StatusBadRequest, gin.H{"message": "end_month must not be before start_month"})
		return
	}
	hasTraffic, err := c.nodeSvc.HasSettlementTraffic(start, end.AddDate(0, 1, 0))
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		return
	}
	if !hasTraffic {
		ctx.JSON(http.StatusBadRequest, gin.H{"message": "任务周期内没有可结算的 EDC 节点月95流量数据"})
		return
	}
	task, err := c.settlementSvc.CreateSettlementTask("node_monthly95", start)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		return
	}
	go func() { _ = c.nodeSvc.ExecuteMonthlyRangeTask(task.ID, start, end) }()
	ctx.JSON(http.StatusOK, gin.H{"task": task})
}

func (c *EDCNodeSettlementController) ListNodeDaily(ctx *gin.Context) {
	filter := buildNodeSettlementFilter(ctx)
	page := parseIntDefault(ctx.Query("page"), 1)
	pageSize := parseIntDefault(ctx.Query("page_size"), 10)
	items, total, err := c.nodeSvc.ListDailySettlements(filter, page, pageSize)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"items": items, "total": total})
}

func (c *EDCNodeSettlementController) ListNodeMonthly(ctx *gin.Context) {
	filter := buildNodeSettlementFilter(ctx)
	page := parseIntDefault(ctx.Query("page"), 1)
	pageSize := parseIntDefault(ctx.Query("page_size"), 10)
	items, total, err := c.nodeSvc.ListMonthlySettlements(filter, page, pageSize)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"items": items, "total": total})
}

func buildNodeSettlementFilter(ctx *gin.Context) map[string]interface{} {
	filter := map[string]interface{}{}
	if v := ctx.Query("region"); v != "" {
		filter["region"] = v
	}
	if v := ctx.Query("cp"); v != "" {
		filter["cp"] = v
	}
	if v := ctx.Query("display_name"); v != "" {
		filter["display_name"] = v
	}
	if v := ctx.Query("service_month"); v != "" {
		filter["service_month"] = v
	}
	if v := ctx.Query("settlement_mode"); v != "" {
		filter["settlement_mode"] = v
	}
	if v := ctx.Query("unit_base"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			filter["unit_base"] = n
		}
	}
	if v := ctx.Query("start_date"); v != "" {
		if t, err := parseDateBoundary(v, false); err == nil {
			filter["start_date"] = t
		}
	}
	if v := ctx.Query("end_date"); v != "" {
		if t, err := parseDateBoundary(v, true); err == nil {
			filter["end_date"] = t.Add(time.Second)
		}
	}
	return filter
}

func parseNodeTaskDate(value string, fallback time.Time) (time.Time, error) {
	if value == "" {
		return startOfLocalDay(fallback), nil
	}
	t, err := time.ParseInLocation("2006-01-02", value, time.Local)
	if err != nil {
		return time.Time{}, err
	}
	return startOfLocalDay(t), nil
}

func parseNodeTaskMonth(value string, fallback time.Time) (time.Time, error) {
	if value == "" {
		return time.Date(fallback.Year(), fallback.Month(), 1, 0, 0, 0, 0, time.Local), nil
	}
	t, err := time.ParseInLocation("2006-01", value, time.Local)
	if err != nil {
		return time.Time{}, err
	}
	return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.Local), nil
}

func startOfLocalDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.Local)
}
