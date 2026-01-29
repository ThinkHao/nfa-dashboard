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
	"gorm.io/gorm"
)

type SettlementDataController struct {
	dataSvc service.SettlementDataService
}

// ListUsedChannelOwners GET /api/v1/settlement/data/customer/channel-owners
// 返回在结算明细中实际被使用过的“渠道归属用户”（系统用户）去重列表
func (c *SettlementDataController) ListUsedChannelOwners(ctx *gin.Context) {
	// 1) 取去重的用户ID
	var uids []uint64
	_ = model.DB.Model(&model.SettlementCustomer{}).
		Where("channel_owner_user_id IS NOT NULL").
		Distinct().Pluck("channel_owner_user_id", &uids).Error

	if len(uids) == 0 {
		ctx.JSON(http.StatusOK, gin.H{"code": 200, "message": "OK", "data": gin.H{"items": []interface{}{}}})
		return
	}

	// 2) 查询用户展示名
	type item struct {
		ID          uint64 `json:"id"`
		DisplayName string `json:"display_name"`
	}
	var users []model.User
	if err := model.DB.Where("id IN ?", uids).Find(&users).Error; err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "查询渠道归属用户失败", "error": err.Error()})
		return
	}
	out := make([]item, 0, len(users))
	for _, u := range users {
		name := ""
		if u.Alias != nil && strings.TrimSpace(*u.Alias) != "" {
			name = strings.TrimSpace(*u.Alias)
		} else if strings.TrimSpace(u.Username) != "" {
			name = strings.TrimSpace(u.Username)
		} else {
			name = fmt.Sprintf("用户#%d", u.ID)
		}
		out = append(out, item{ID: u.ID, DisplayName: name})
	}
	ctx.JSON(http.StatusOK, gin.H{"code": 200, "message": "OK", "data": gin.H{"items": out}})
}

// ListUsedOwnerSubjects GET /api/v1/settlement/data/customer/owner-subjects
// 统一返回四类归属主体的去重列表：entity(业务对象) 与 user(系统用户)
func (c *SettlementDataController) ListUsedOwnerSubjects(ctx *gin.Context) {
	region := ctx.Query("region")
	cp := ctx.Query("cp")
	school := ctx.Query("school_name")
	startS := ctx.Query("start_service_date")
	endS := ctx.Query("end_service_date")

	var startPtr *time.Time
	var endPtr *time.Time
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

	qbBase := func() *gorm.DB {
		q := model.DB.Model(&model.SettlementCustomer{})
		if region != "" {
			q = q.Where("region = ?", region)
		}
		if cp != "" {
			q = q.Where("cp = ?", cp)
		}
		if school != "" {
			q = q.Where("school_name LIKE ?", "%"+school+"%")
		}
		if startPtr != nil {
			q = q.Where("DATE(service_date) >= ?", startPtr.Format("2006-01-02"))
		}
		if endPtr != nil {
			q = q.Where("DATE(service_date) <= ?", endPtr.Format("2006-01-02"))
		}
		return q
	}

	// 统一按“用户”聚合：四列均视为用户ID来源
	var uids1, uids2, uids3, uids4 []uint64
	_ = qbBase().Where("customer_fee_owner_id IS NOT NULL").Distinct().Pluck("customer_fee_owner_id", &uids1).Error
	_ = qbBase().Where("network_line_fee_owner_id IS NOT NULL").Distinct().Pluck("network_line_fee_owner_id", &uids2).Error
	_ = qbBase().Where("node_deduction_fee_owner_id IS NOT NULL").Distinct().Pluck("node_deduction_fee_owner_id", &uids3).Error
	_ = qbBase().Where("channel_owner_user_id IS NOT NULL").Distinct().Pluck("channel_owner_user_id", &uids4).Error

	userSet := make(map[uint64]struct{})
	for _, id := range uids1 {
		if id > 0 {
			userSet[id] = struct{}{}
		}
	}
	for _, id := range uids2 {
		if id > 0 {
			userSet[id] = struct{}{}
		}
	}
	for _, id := range uids3 {
		if id > 0 {
			userSet[id] = struct{}{}
		}
	}
	for _, id := range uids4 {
		if id > 0 {
			userSet[id] = struct{}{}
		}
	}

	type outItem struct {
		Type  string `json:"type"`
		ID    uint64 `json:"id"`
		Label string `json:"label"`
	}

	out := make([]outItem, 0, len(userSet))
	if len(userSet) > 0 {
		ids := make([]uint64, 0, len(userSet))
		for id := range userSet {
			ids = append(ids, id)
		}
		var users []model.User
		if err := model.DB.Where("id IN ?", ids).Find(&users).Error; err == nil {
			for _, u := range users {
				name := ""
				if u.Alias != nil && strings.TrimSpace(*u.Alias) != "" {
					name = strings.TrimSpace(*u.Alias)
				} else if strings.TrimSpace(u.Username) != "" {
					name = strings.TrimSpace(u.Username)
				} else {
					name = fmt.Sprintf("用户#%d", u.ID)
				}
				out = append(out, outItem{Type: "user", ID: u.ID, Label: name})
			}
		}
	}

	ctx.JSON(http.StatusOK, gin.H{"code": 200, "message": "OK", "data": gin.H{"items": out}})
}

// ListUsedOwnerEntities GET /api/v1/settlement/data/customer/owners
// 返回在结算明细中实际被使用过的业务对象（费用归属：客户费/线路费/节点通用费）去重列表
func (c *SettlementDataController) ListUsedOwnerEntities(ctx *gin.Context) {
	// 1) 分别取三个 owner 列的去重ID
	var ids1, ids2, ids3 []uint64
	_ = model.DB.Model(&model.SettlementCustomer{}).
		Where("customer_fee_owner_id IS NOT NULL").
		Distinct().Pluck("customer_fee_owner_id", &ids1).Error
	_ = model.DB.Model(&model.SettlementCustomer{}).
		Where("network_line_fee_owner_id IS NOT NULL").
		Distinct().Pluck("network_line_fee_owner_id", &ids2).Error
	_ = model.DB.Model(&model.SettlementCustomer{}).
		Where("node_deduction_fee_owner_id IS NOT NULL").
		Distinct().Pluck("node_deduction_fee_owner_id", &ids3).Error

	// 2) 合并去重
	uniq := map[uint64]struct{}{}
	for _, id := range ids1 {
		if id > 0 {
			uniq[id] = struct{}{}
		}
	}
	for _, id := range ids2 {
		if id > 0 {
			uniq[id] = struct{}{}
		}
	}
	for _, id := range ids3 {
		if id > 0 {
			uniq[id] = struct{}{}
		}
	}
	if len(uniq) == 0 {
		ctx.JSON(http.StatusOK, gin.H{"code": 200, "message": "OK", "data": gin.H{"items": []interface{}{}}})
		return
	}
	ids := make([]uint64, 0, len(uniq))
	for id := range uniq {
		ids = append(ids, id)
	}

	// 3) 查询业务对象名称
	type item struct {
		ID         uint64 `json:"id"`
		EntityName string `json:"entity_name"`
	}
	var ents []model.BusinessEntity
	if err := model.DB.Where("id IN ?", ids).Find(&ents).Error; err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "查询费用归属对象失败", "error": err.Error()})
		return
	}
	out := make([]item, 0, len(ents))
	for _, e := range ents {
		out = append(out, item{ID: e.ID, EntityName: strings.TrimSpace(e.EntityName)})
	}
	ctx.JSON(http.StatusOK, gin.H{"code": 200, "message": "OK", "data": gin.H{"items": out}})
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
		if t, err := time.Parse("2006-01-02", startS); err == nil {
			startPtr = &t
		}
	}
	if endS != "" {
		if t, err := time.Parse("2006-01-02", endS); err == nil {
			endPtr = &t
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
	items, total, err := c.dataSvc.List(service.SettlementCustomerFilter{Region: region, CP: cp, School: school, Start: startPtr, End: endPtr, OwnerEntityID: ownerPtr, ChannelOwnerUserID: channelPtr}, page, size)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "查询结算数据失败", "error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"code": 200, "message": "OK", "data": gin.H{"items": items, "total": total}})
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
		if t, err := time.Parse("2006-01-02", startS); err == nil {
			startPtr = &t
		}
	}
	if endS != "" {
		if t, err := time.Parse("2006-01-02", endS); err == nil {
			endPtr = &t
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
	rows, err := c.dataSvc.ListAll(service.SettlementCustomerFilter{Region: region, CP: cp, School: school, Start: startPtr, End: endPtr, OwnerEntityID: ownerPtr, ChannelOwnerUserID: channelPtr})
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
