package controller

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"nfa-dashboard/internal/model"
	"nfa-dashboard/internal/service"
)

type EDCController struct {
	edcService   service.EDCService
	scopeService service.EDCTrafficScopeService
}

func NewEDCController(edcService service.EDCService, scopeService service.EDCTrafficScopeService) *EDCController {
	return &EDCController{edcService: edcService, scopeService: scopeService}
}

func (c *EDCController) ListEntities(ctx *gin.Context) {
	scope, ok := c.resolveEDCScopeOrRespond(ctx)
	if !ok {
		return
	}
	limit, _ := strconv.Atoi(ctx.DefaultQuery("limit", "100"))
	offset, _ := strconv.Atoi(ctx.DefaultQuery("offset", "0"))
	if scope.Source == model.EDCTrafficScopeSourceNone {
		ctx.JSON(http.StatusOK, gin.H{"code": 200, "message": "获取 EDC 实体列表成功", "data": gin.H{"total": 0, "items": []model.EDCEntity{}, "limit": limit, "offset": offset}, "scope_source": scope.Source})
		return
	}
	items, total, err := c.edcService.ListEntities(model.EDCEntityFilter{
		DisplayName:      ctx.Query("display_name"),
		Region:           ctx.Query("region"),
		CP:               ctx.Query("cp"),
		EntityType:       ctx.Query("entity_type"),
		SrcRegion:        ctx.Query("src_region"),
		DstRegion:        ctx.Query("dst_region"),
		AllowedEntityIDs: scope.AllowedEntityIDs,
		Limit:            limit,
		Offset:           offset,
	})
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "获取 EDC 实体列表失败", "error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"code": 200, "message": "获取 EDC 实体列表成功", "data": gin.H{"total": total, "items": items, "limit": limit, "offset": offset}, "scope_source": scope.Source})
}

func (c *EDCController) ListRegions(ctx *gin.Context) {
	scope, ok := c.resolveEDCScopeOrRespond(ctx)
	if !ok {
		return
	}
	if scope.Source == model.EDCTrafficScopeSourceNone {
		ctx.JSON(http.StatusOK, gin.H{"code": 200, "message": "获取 EDC 地区列表成功", "data": []string{}, "scope_source": scope.Source})
		return
	}
	regions, err := c.edcService.ListRegions(scope.AllowedEntityIDs)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "获取 EDC 地区列表失败", "error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"code": 200, "message": "获取 EDC 地区列表成功", "data": regions, "scope_source": scope.Source})
}

func (c *EDCController) ListCPs(ctx *gin.Context) {
	scope, ok := c.resolveEDCScopeOrRespond(ctx)
	if !ok {
		return
	}
	if scope.Source == model.EDCTrafficScopeSourceNone {
		ctx.JSON(http.StatusOK, gin.H{"code": 200, "message": "获取 EDC 内容方列表成功", "data": []string{}, "scope_source": scope.Source})
		return
	}
	cps, err := c.edcService.ListCPs(scope.AllowedEntityIDs)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "获取 EDC 内容方列表失败", "error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"code": 200, "message": "获取 EDC 内容方列表成功", "data": cps, "scope_source": scope.Source})
}

func (c *EDCController) ListFilterOptions(ctx *gin.Context) {
	scope, ok := c.resolveEDCScopeOrRespond(ctx)
	if !ok {
		return
	}
	if scope.Source == model.EDCTrafficScopeSourceNone {
		ctx.JSON(http.StatusOK, gin.H{"code": 200, "message": "获取 EDC 筛选项成功", "data": model.EDCFilterOptions{}, "scope_source": scope.Source})
		return
	}
	options, err := c.edcService.ListFilterOptions(scope.AllowedEntityIDs)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "获取 EDC 筛选项失败", "error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"code": 200, "message": "获取 EDC 筛选项成功", "data": options, "scope_source": scope.Source})
}

func (c *EDCController) GetTrafficData(ctx *gin.Context) {
	filter, ok := parseEDCTrafficFilter(ctx)
	if !ok {
		return
	}
	scope, ok := c.resolveEDCScopeOrRespond(ctx)
	if !ok {
		return
	}
	if scope.Source == model.EDCTrafficScopeSourceNone {
		ctx.JSON(http.StatusOK, gin.H{"code": 200, "message": "获取 EDC 流量数据成功，但没有符合条件的数据", "data": []interface{}{}, "scope_source": scope.Source})
		return
	}
	filter.AllowedEntityIDs = scope.AllowedEntityIDs
	filter.ScopeSource = scope.Source
	data, err := c.edcService.GetTrafficData(filter)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			return
		}
		ctx.JSON(http.StatusOK, gin.H{"code": 200, "message": "获取 EDC 流量数据成功，但没有符合条件的数据", "data": []interface{}{}, "warning": err.Error()})
		return
	}
	if len(data) == 0 {
		ctx.JSON(http.StatusOK, gin.H{"code": 200, "message": "获取 EDC 流量数据成功，但没有符合条件的数据", "data": []interface{}{}})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"code": 200, "message": "获取 EDC 流量数据成功", "data": data})
}

func (c *EDCController) GetTrafficSummary(ctx *gin.Context) {
	filter, ok := parseEDCTrafficFilter(ctx)
	if !ok {
		return
	}
	scope, ok := c.resolveEDCScopeOrRespond(ctx)
	if !ok {
		return
	}
	if scope.Source == model.EDCTrafficScopeSourceNone {
		ctx.JSON(http.StatusOK, gin.H{"code": 200, "message": "获取 EDC 流量汇总成功", "data": model.EDCTrafficResponse{}, "scope_source": scope.Source})
		return
	}
	filter.AllowedEntityIDs = scope.AllowedEntityIDs
	filter.ScopeSource = scope.Source
	summary, err := c.edcService.GetTrafficSummary(filter)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			return
		}
		ctx.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "获取 EDC 流量汇总失败", "error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"code": 200, "message": "获取 EDC 流量汇总成功", "data": summary})
}

func parseEDCTrafficFilter(ctx *gin.Context) (model.EDCTrafficFilter, bool) {
	var filter model.EDCTrafficFilter
	if s := ctx.Query("start_time"); s != "" {
		t, err := parseTrafficTimeParam(s)
		if err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "start_time 格式错误，应为 RFC3339 或 YYYY-MM-DD HH:mm:ss", "error": err.Error()})
			return filter, false
		}
		filter.StartTime = t
	}
	if s := ctx.Query("end_time"); s != "" {
		t, err := parseTrafficTimeParam(s)
		if err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "end_time 格式错误，应为 RFC3339 或 YYYY-MM-DD HH:mm:ss", "error": err.Error()})
			return filter, false
		}
		filter.EndTime = t
	}
	if !filter.StartTime.IsZero() && !filter.EndTime.IsZero() {
		if err := validateTrafficTimeRange(filter.StartTime, filter.EndTime); err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "时间范围非法：start_time 必须早于 end_time", "error": err.Error()})
			return filter, false
		}
	}
	filter.DisplayName = ctx.Query("display_name")
	if s := ctx.Query("entity_ids"); s != "" {
		ids, err := parseEDCEntityIDsParam(s)
		if err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "entity_ids 格式错误，应为逗号分隔的正整数", "error": err.Error()})
			return filter, false
		}
		filter.EntityIDs = ids
	}
	filter.Region = ctx.Query("region")
	filter.CP = ctx.Query("cp")
	filter.EntityType = ctx.Query("entity_type")
	if filter.EntityType != "" && filter.EntityType != model.EDCEntityTypeNode && filter.EntityType != model.EDCEntityTypeTransmission {
		ctx.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "entity_type 只能是 node 或 transmission"})
		return filter, false
	}
	filter.SrcRegion = ctx.Query("src_region")
	filter.DstRegion = ctx.Query("dst_region")
	return filter, true
}

func parseEDCEntityIDsParam(value string) ([]uint64, error) {
	parts := strings.Split(value, ",")
	ids := make([]uint64, 0, len(parts))
	for _, part := range parts {
		s := strings.TrimSpace(part)
		if s == "" {
			continue
		}
		id, err := strconv.ParseUint(s, 10, 64)
		if err != nil || id == 0 {
			return nil, errors.New("entity id must be a positive integer")
		}
		ids = append(ids, id)
	}
	return ids, nil
}

func (c *EDCController) resolveEDCScopeOrRespond(ctx *gin.Context) (model.EffectiveEDCTrafficScope, bool) {
	if c.scopeService == nil {
		return model.EffectiveEDCTrafficScope{Source: model.EDCTrafficScopeSourceNone, AllowedEntityIDs: []uint64{}}, true
	}
	uid, ok := currentUserID(ctx)
	if !ok || uid == 0 {
		return model.EffectiveEDCTrafficScope{Source: model.EDCTrafficScopeSourceNone, AllowedEntityIDs: []uint64{}}, true
	}
	scope, err := c.scopeService.ResolveEffectiveScope(uid)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "解析 EDC 可见范围失败", "error": err.Error()})
		return model.EffectiveEDCTrafficScope{}, false
	}
	return scope, true
}

type SystemEDCTrafficScopeController struct {
	scopeService service.EDCTrafficScopeService
	userService  service.UserService
}

func NewSystemEDCTrafficScopeController(scopeService service.EDCTrafficScopeService, userService service.UserService) *SystemEDCTrafficScopeController {
	return &SystemEDCTrafficScopeController{scopeService: scopeService, userService: userService}
}

func (c *SystemEDCTrafficScopeController) ListRules(ctx *gin.Context) {
	userID, err := strconv.ParseUint(ctx.Param("user_id"), 10, 64)
	if err != nil || userID == 0 {
		ctx.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "用户ID无效"})
		return
	}
	rules, err := c.scopeService.ListRules(userID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "获取 EDC 可见范围失败", "error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"items": rules, "total": len(rules)})
}

func (c *SystemEDCTrafficScopeController) ReplaceRules(ctx *gin.Context) {
	userID, err := strconv.ParseUint(ctx.Param("user_id"), 10, 64)
	if err != nil || userID == 0 {
		ctx.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "用户ID无效"})
		return
	}
	var payload struct {
		Rules []model.EDCTrafficScopeRuleGroup `json:"rules"`
	}
	if err := ctx.ShouldBindJSON(&payload); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "请求体格式错误", "error": err.Error()})
		return
	}
	if err := c.scopeService.ReplaceRules(userID, payload.Rules); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "保存 EDC 可见范围失败", "error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"code": 200, "message": "保存 EDC 可见范围成功"})
}
