package controller

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"nfa-dashboard/internal/model"
	"nfa-dashboard/internal/service"

	"github.com/gin-gonic/gin"
)

type SettlementDataController struct {
	dataSvc service.SettlementDataService
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
		region = ctx.Query("region")
		cp     = ctx.Query("cp")
		school = ctx.Query("school_name")
		startS = ctx.Query("start_service_date")
		endS   = ctx.Query("end_service_date")
		page   = intFrom(ctx.DefaultQuery("page", "1"), 1)
		size   = intFrom(ctx.DefaultQuery("page_size", "10"), 10)
	)
	var (
		startPtr *time.Time
		endPtr   *time.Time
	)
	if startS != "" {
		if t, err := time.Parse("2006-01-02", startS); err == nil {
			startPtr = &t
		}
	}
	if endS != "" {
		if t, err := time.Parse("2006-01-02", endS); err == nil {
			endPtr = &t
		}
	}
	items, total, err := c.dataSvc.List(service.SettlementCustomerFilter{Region: region, CP: cp, School: school, Start: startPtr, End: endPtr}, page, size)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "查询结算数据失败", "error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"code": 200, "message": "OK", "data": gin.H{"items": items, "total": total}})
}

// ExportCustomerData GET /api/v1/settlement/data/customer/export
func (c *SettlementDataController) ExportCustomerData(ctx *gin.Context) {
	var (
		region = ctx.Query("region")
		cp     = ctx.Query("cp")
		school = ctx.Query("school_name")
		startS = ctx.Query("start_service_date")
		endS   = ctx.Query("end_service_date")
	)
	var (
		startPtr *time.Time
		endPtr   *time.Time
	)
	if startS != "" {
		if t, err := time.Parse("2006-01-02", startS); err == nil {
			startPtr = &t
		}
	}
	if endS != "" {
		if t, err := time.Parse("2006-01-02", endS); err == nil {
			endPtr = &t
		}
	}
	rows, err := c.dataSvc.ListAll(service.SettlementCustomerFilter{Region: region, CP: cp, School: school, Start: startPtr, End: endPtr})
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "导出失败", "error": err.Error()})
		return
	}
	// 批量构建费用归属名称映射（业务对象与系统用户）
	entityIDSet := map[uint64]struct{}{}
	userIDSet := map[uint64]struct{}{}
	for _, r := range rows {
		if r.CustomerFeeOwnerID != nil {
			entityIDSet[*r.CustomerFeeOwnerID] = struct{}{}
		}
		if r.NetworkLineFeeOwnerID != nil {
			entityIDSet[*r.NetworkLineFeeOwnerID] = struct{}{}
		}
		if r.ChannelOwnerUserID != nil {
			userIDSet[*r.ChannelOwnerUserID] = struct{}{}
		}
	}
	entityIDs := make([]uint64, 0, len(entityIDSet))
	for id := range entityIDSet {
		entityIDs = append(entityIDs, id)
	}
	userIDs := make([]uint64, 0, len(userIDSet))
	for id := range userIDSet {
		userIDs = append(userIDs, id)
	}
	entityMap := map[uint64]string{}
	if len(entityIDs) > 0 {
		var ents []model.BusinessEntity
		if err := model.DB.Where("id IN ?", entityIDs).Find(&ents).Error; err == nil {
			for _, e := range ents {
				entityMap[e.ID] = strings.TrimSpace(e.EntityName)
			}
		}
	}
	userMap := map[uint64]string{}
	if len(userIDs) > 0 {
		var us []model.User
		if err := model.DB.Where("id IN ?", userIDs).Find(&us).Error; err == nil {
			for _, u := range us {
				dn := ""
				if u.Alias != nil && strings.TrimSpace(*u.Alias) != "" {
					dn = strings.TrimSpace(*u.Alias)
				} else if strings.TrimSpace(u.Username) != "" {
					dn = strings.TrimSpace(u.Username)
				} else {
					dn = fmt.Sprintf("用户#%d", u.ID)
				}
				userMap[u.ID] = dn
			}
		}
	}
	// CSV 导出
	ctx.Header("Content-Type", "text/csv; charset=utf-8")
	ctx.Header("Content-Disposition", "attachment; filename=settlement_customer.csv")
	w := ctx.Writer
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
	var (
		start, _ = time.Parse("2006-01-02", req.Start)
		end, _   = time.Parse("2006-01-02", req.End)
	)
	// 创建任务记录
	task := &model.SettlementTask{
		TaskType:       "customer_recalc",
		TaskDate:       start,
		Status:         "pending",
		ProcessedCount: 0,
		CreateTime:     time.Now(),
		UpdateTime:     time.Now(),
		ErrorMessage:   req.Start + "," + req.End,
	}
	if err := model.DB.Create(task).Error; err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "创建复算任务失败", "error": err.Error()})
		return
	}
	// 异步执行
	go func(tid int64, region, cp, school string, s, e time.Time) {
		// running
		model.DB.Model(&model.SettlementTask{}).Where("id = ?", tid).Updates(map[string]interface{}{"status": "running", "start_time": time.Now()})
		affected, err := c.dataSvc.Recalculate(service.SettlementCustomerFilter{Region: region, CP: cp, School: school, Start: &s, End: &e})
		if err != nil {
			model.DB.Model(&model.SettlementTask{}).Where("id = ?", tid).Updates(map[string]interface{}{"status": "failed", "end_time": time.Now(), "error_message": err.Error()})
			return
		}
		model.DB.Model(&model.SettlementTask{}).Where("id = ?", tid).Updates(map[string]interface{}{"status": "success", "end_time": time.Now(), "processed_count": affected})
	}(task.ID, req.Region, req.CP, req.School, start, end)

	ctx.JSON(http.StatusOK, gin.H{"code": 200, "message": "已创建复算任务", "data": gin.H{"task_id": task.ID}})
}

// 工具函数
func intFrom(s string, def int) int {
	if v, err := strconv.Atoi(s); err == nil {
		return v
	}
	return def
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
