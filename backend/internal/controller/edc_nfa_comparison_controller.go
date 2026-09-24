package controller

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"nfa-dashboard/internal/model"
	"nfa-dashboard/internal/service"
)

type EDCNFAComparisonController struct {
	comparisonService service.EDCNFAComparisonService
	edcScopeService   service.EDCTrafficScopeService
	trafficScope      service.TrafficScopeService
	settingsService   service.SystemSettingsService
	participationSvc  service.SettlementParticipationService
}

// ListMappings returns the complete mapping configuration for the management page.
func (c *EDCNFAComparisonController) ListMappings(ctx *gin.Context) {
	mappings, err := c.comparisonService.ListMappings(ctx.Request.Context())
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "获取 EDC/NFA 映射配置失败", "error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"code": 200, "message": "获取 EDC/NFA 映射配置成功", "data": mappings})
}

func (c *EDCNFAComparisonController) ListMappingEntities(ctx *gin.Context) {
	entities, err := c.comparisonService.ListMappingEntities(ctx.Request.Context())
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "获取可映射 EDC 实体失败", "error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"code": 200, "message": "获取可映射 EDC 实体成功", "data": entities})
}

func (c *EDCNFAComparisonController) CreateMapping(ctx *gin.Context) {
	input, ok := parseMappingInput(ctx)
	if !ok {
		return
	}
	mapping, err := c.comparisonService.CreateMapping(ctx.Request.Context(), input)
	if err != nil {
		c.writeMappingError(ctx, err)
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"code": 200, "message": "创建 EDC/NFA 映射成功", "data": mapping})
}

func (c *EDCNFAComparisonController) UpdateMapping(ctx *gin.Context) {
	id, err := strconv.ParseUint(ctx.Param("id"), 10, 64)
	if err != nil || id == 0 {
		ctx.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "id 必须是正整数"})
		return
	}
	input, ok := parseMappingInput(ctx)
	if !ok {
		return
	}
	mapping, err := c.comparisonService.UpdateMapping(ctx.Request.Context(), id, input)
	if err != nil {
		c.writeMappingError(ctx, err)
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"code": 200, "message": "更新 EDC/NFA 映射成功", "data": mapping})
}

type mappingInputPayload struct {
	GroupName    string `json:"group_name"`
	NFASrcRegion string `json:"nfa_src_region"`
	NFACP        string `json:"nfa_cp"`
	Enabled      *bool  `json:"enabled"`
	Remark       string `json:"remark"`
	Members      []struct {
		EntityID  uint64  `json:"entity_id"`
		ValidFrom *string `json:"valid_from"`
		ValidTo   *string `json:"valid_to"`
		Enabled   *bool   `json:"enabled"`
	} `json:"members"`
}

func parseMappingInput(ctx *gin.Context) (model.EDCNFAComparisonGroupInput, bool) {
	var payload mappingInputPayload
	if err := ctx.ShouldBindJSON(&payload); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "映射配置格式错误", "error": err.Error()})
		return model.EDCNFAComparisonGroupInput{}, false
	}
	input := model.EDCNFAComparisonGroupInput{
		GroupName:    payload.GroupName,
		NFASrcRegion: payload.NFASrcRegion,
		NFACP:        payload.NFACP,
		Enabled:      payload.Enabled,
		Remark:       payload.Remark,
		Members:      make([]model.EDCNFAComparisonMemberInput, 0, len(payload.Members)),
	}
	for _, item := range payload.Members {
		from, err := parseOptionalMappingTime(item.ValidFrom)
		if err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "valid_from 格式错误", "error": err.Error()})
			return model.EDCNFAComparisonGroupInput{}, false
		}
		to, err := parseOptionalMappingTime(item.ValidTo)
		if err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "valid_to 格式错误", "error": err.Error()})
			return model.EDCNFAComparisonGroupInput{}, false
		}
		input.Members = append(input.Members, model.EDCNFAComparisonMemberInput{EntityID: item.EntityID, ValidFrom: from, ValidTo: to, Enabled: item.Enabled})
	}
	return input, true
}

func parseOptionalMappingTime(raw *string) (*time.Time, error) {
	if raw == nil || strings.TrimSpace(*raw) == "" {
		return nil, nil
	}
	parsed, err := parseTrafficTimeParam(*raw)
	if err != nil {
		return nil, err
	}
	return &parsed, nil
}

func (c *EDCNFAComparisonController) SetMappingEnabled(ctx *gin.Context) {
	id, err := strconv.ParseUint(ctx.Param("id"), 10, 64)
	if err != nil || id == 0 {
		ctx.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "id 必须是正整数"})
		return
	}
	var input struct {
		Enabled bool `json:"enabled"`
	}
	if err := ctx.ShouldBindJSON(&input); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "enabled 必须是布尔值", "error": err.Error()})
		return
	}
	if err := c.comparisonService.SetMappingEnabled(ctx.Request.Context(), id, input.Enabled); err != nil {
		c.writeMappingError(ctx, err)
		return
	}
	ctx.Status(http.StatusNoContent)
}

func (c *EDCNFAComparisonController) writeMappingError(ctx *gin.Context, err error) {
	if service.IsBadRequest(err) {
		ctx.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	ctx.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "保存 EDC/NFA 映射失败", "error": err.Error()})
}

func NewEDCNFAComparisonController(
	comparisonService service.EDCNFAComparisonService,
	edcScopeService service.EDCTrafficScopeService,
	trafficScope service.TrafficScopeService,
	settingsService service.SystemSettingsService,
	participationSvc service.SettlementParticipationService,
) *EDCNFAComparisonController {
	return &EDCNFAComparisonController{
		comparisonService: comparisonService,
		edcScopeService:   edcScopeService,
		trafficScope:      trafficScope,
		settingsService:   settingsService,
		participationSvc:  participationSvc,
	}
}

func (c *EDCNFAComparisonController) ListGroups(ctx *gin.Context) {
	edcScope, ok := c.resolveEDCScope(ctx)
	if !ok {
		return
	}
	trafficScope, err := resolveTrafficScopeForRequest(ctx, c.trafficScope, c.settingsService, c.participationSvc)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "解析流量可见范围失败", "error": err.Error()})
		return
	}
	if edcScope.Source == model.EDCTrafficScopeSourceNone || trafficScope.Source == model.TrafficScopeSourceNone {
		ctx.JSON(http.StatusOK, gin.H{"code": 200, "message": "获取 EDC/NFA 比较组成功", "data": []model.EDCNFAComparisonGroup{}, "scope_source": gin.H{"edc": edcScope.Source, "nfa": trafficScope.Source}})
		return
	}
	groups, err := c.comparisonService.ListGroups(ctx.Request.Context(), edcScope.AllowedEntityIDs)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "获取 EDC/NFA 比较组失败", "error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"code": 200, "message": "获取 EDC/NFA 比较组成功", "data": groups, "scope_source": gin.H{"edc": edcScope.Source, "nfa": trafficScope.Source}})
}

func (c *EDCNFAComparisonController) GetComparison(ctx *gin.Context) {
	groupID, err := strconv.ParseUint(ctx.Query("group_id"), 10, 64)
	if err != nil || groupID == 0 {
		ctx.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "group_id 必须是正整数"})
		return
	}
	start, end, ok := parseComparisonTimeRange(ctx)
	if !ok {
		return
	}
	edcScope, ok := c.resolveEDCScope(ctx)
	if !ok {
		return
	}
	trafficScope, err := resolveTrafficScopeForRequest(ctx, c.trafficScope, c.settingsService, c.participationSvc)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "解析流量可见范围失败", "error": err.Error()})
		return
	}
	if edcScope.Source == model.EDCTrafficScopeSourceNone || trafficScope.Source == model.TrafficScopeSourceNone {
		ctx.JSON(http.StatusOK, gin.H{"code": 200, "message": "获取 EDC/NFA 比较数据成功，但没有符合权限的数据", "data": []model.EDCNFAComparisonPoint{}, "scope_source": gin.H{"edc": edcScope.Source, "nfa": trafficScope.Source}})
		return
	}
	points, err := c.comparisonService.GetComparison(ctx.Request.Context(), model.EDCNFAComparisonFilter{
		GroupID:           groupID,
		StartTime:         start,
		EndTime:           end,
		AllowedEntityIDs:  edcScope.AllowedEntityIDs,
		AllowedSchoolKeys: trafficScope.AllowedSchoolKeys,
	})
	if err != nil {
		if errors.Is(err, context.Canceled) {
			return
		}
		if service.IsBadRequest(err) {
			ctx.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
			return
		}
		ctx.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "获取 EDC/NFA 比较数据失败", "error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "获取 EDC/NFA 比较数据成功",
		"data": gin.H{
			"points":                   points,
			"edc_raw_field":            "service_size",
			"edc_raw_interval_seconds": 300,
			"nfa_raw_field":            "total_recv",
			"nfa_raw_interval_seconds": 60,
		},
		"scope_source": gin.H{"edc": edcScope.Source, "nfa": trafficScope.Source},
	})
}

func (c *EDCNFAComparisonController) resolveEDCScope(ctx *gin.Context) (model.EffectiveEDCTrafficScope, bool) {
	if c.edcScopeService == nil {
		return model.EffectiveEDCTrafficScope{Source: model.EDCTrafficScopeSourceNone}, true
	}
	uid, ok := currentUserID(ctx)
	if !ok || uid == 0 {
		return model.EffectiveEDCTrafficScope{Source: model.EDCTrafficScopeSourceNone}, true
	}
	scope, err := c.edcScopeService.ResolveEffectiveScope(uid)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "解析 EDC 可见范围失败", "error": err.Error()})
		return model.EffectiveEDCTrafficScope{}, false
	}
	return scope, true
}

func parseComparisonTimeRange(ctx *gin.Context) (start, end time.Time, ok bool) {
	if raw := ctx.Query("start_time"); raw != "" {
		var err error
		start, err = parseTrafficTimeParam(raw)
		if err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "start_time 格式错误，应为 RFC3339 或 YYYY-MM-DD HH:mm:ss", "error": err.Error()})
			return time.Time{}, time.Time{}, false
		}
	}
	if raw := ctx.Query("end_time"); raw != "" {
		var err error
		end, err = parseTrafficTimeParam(raw)
		if err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "end_time 格式错误，应为 RFC3339 或 YYYY-MM-DD HH:mm:ss", "error": err.Error()})
			return time.Time{}, time.Time{}, false
		}
	}
	if !start.IsZero() && !end.IsZero() {
		if err := validateTrafficTimeRange(start, end); err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "时间范围非法", "error": err.Error()})
			return time.Time{}, time.Time{}, false
		}
	}
	return start, end, true
}
