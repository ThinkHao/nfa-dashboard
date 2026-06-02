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
		Date string `json:"date"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"message": "invalid request"})
		return
	}
	day, err := parseNodeTaskDate(req.Date, time.Now().AddDate(0, 0, -1))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"message": "invalid date, expect YYYY-MM-DD"})
		return
	}
	task, err := c.settlementSvc.CreateSettlementTask("node_daily95", day)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		return
	}
	go func() { _ = c.nodeSvc.ExecuteDailyTask(task.ID, day) }()
	ctx.JSON(http.StatusOK, gin.H{"task": task})
}

func (c *EDCNodeSettlementController) CreateNodeMonthlyTask(ctx *gin.Context) {
	var req struct {
		Month string `json:"month"`
		Date  string `json:"date"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"message": "invalid request"})
		return
	}
	monthText := req.Month
	if monthText == "" {
		monthText = req.Date
	}
	month, err := parseNodeTaskMonth(monthText, time.Now().AddDate(0, -1, 0))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"message": "invalid month, expect YYYY-MM"})
		return
	}
	task, err := c.settlementSvc.CreateSettlementTask("node_monthly95", month)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		return
	}
	go func() { _ = c.nodeSvc.ExecuteMonthlyTask(task.ID, month) }()
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
