package controller

import (
    "encoding/json"
    "net/http"
    "strconv"

    "github.com/gin-gonic/gin"
    "gorm.io/datatypes"
    "nfa-dashboard/internal/model"
    "nfa-dashboard/internal/service"
)

// SyncRulesController manages endpoints under /api/v1/settlement/rates/sync-rules

type SyncRulesController struct{ svc service.SyncRulesService }

func NewSyncRulesController(svc service.SyncRulesService) *SyncRulesController { return &SyncRulesController{svc: svc} }

func (ctl *SyncRulesController) List(c *gin.Context) {
    page := parseIntDefault(c.Query("page"), 1)
    pageSize := parseIntDefault(c.Query("page_size"), 10)
    name := c.Query("name")
    var enabledPtr *bool
    if v := c.Query("enabled"); v != "" {
        b := v == "1" || v == "true"
        enabledPtr = &b
    }
    items, total, err := ctl.svc.List(name, enabledPtr, page, pageSize)
    if err != nil { c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()}); return }
    c.JSON(http.StatusOK, gin.H{"items": items, "total": total})
}

func (ctl *SyncRulesController) Create(c *gin.Context) {
    type reqT struct{
        Name              string           `json:"name" binding:"required"`
        Enabled           bool             `json:"enabled"`
        Priority          int              `json:"priority"`
        ScopeRegion       *json.RawMessage `json:"scope_region"`
        ScopeCP           *json.RawMessage `json:"scope_cp"`
        ConditionExpr     *string          `json:"condition_expr"`
        FieldsToUpdate    *json.RawMessage `json:"fields_to_update"`
        OverwriteStrategy string           `json:"overwrite_strategy" binding:"required"`
        Actions           json.RawMessage  `json:"actions" binding:"required"`
    }
    var req reqT
    if err := c.ShouldBindJSON(&req); err != nil { c.JSON(http.StatusBadRequest, gin.H{"message":"invalid request"}); return }
    rule := &model.RateCustomerSyncRule{
        Name: req.Name,
        Enabled: req.Enabled,
        Priority: req.Priority,
        ConditionExpr: req.ConditionExpr,
        OverwriteStrategy: req.OverwriteStrategy,
        Actions: datatypes.JSON(req.Actions),
    }
    if req.ScopeRegion != nil { rule.ScopeRegion = datatypes.JSON(*req.ScopeRegion) }
    if req.ScopeCP != nil { rule.ScopeCP = datatypes.JSON(*req.ScopeCP) }
    if req.FieldsToUpdate != nil { rule.FieldsToUpdate = datatypes.JSON(*req.FieldsToUpdate) }
    out, err := ctl.svc.Create(rule)
    if err != nil {
        if service.IsBadRequest(err) { c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()}); return }
        c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()}); return
    }
    c.JSON(http.StatusOK, out)
}

func (ctl *SyncRulesController) Update(c *gin.Context) {
    id, err := strconv.ParseUint(c.Param("id"), 10, 64)
    if err != nil || id == 0 { c.JSON(http.StatusBadRequest, gin.H{"message":"invalid id"}); return }
    type reqT struct{
        Name              *string          `json:"name"`
        Enabled           *bool            `json:"enabled"`
        Priority          *int             `json:"priority"`
        ScopeRegion       *json.RawMessage `json:"scope_region"`
        ScopeCP           *json.RawMessage `json:"scope_cp"`
        ConditionExpr     *string          `json:"condition_expr"`
        FieldsToUpdate    *json.RawMessage `json:"fields_to_update"`
        OverwriteStrategy *string          `json:"overwrite_strategy"`
        Actions           *json.RawMessage `json:"actions"`
    }
    var req reqT
    if err := c.ShouldBindJSON(&req); err != nil { c.JSON(http.StatusBadRequest, gin.H{"message":"invalid request"}); return }
    updates := map[string]interface{}{}
    if req.Name != nil { updates["name"] = *req.Name }
    if req.Enabled != nil { updates["enabled"] = *req.Enabled }
    if req.Priority != nil { updates["priority"] = *req.Priority }
    if req.ScopeRegion != nil { b := datatypes.JSON(*req.ScopeRegion); updates["scope_region"] = b }
    if req.ScopeCP != nil { b := datatypes.JSON(*req.ScopeCP); updates["scope_cp"] = b }
    if req.ConditionExpr != nil { updates["condition_expr"] = *req.ConditionExpr }
    if req.FieldsToUpdate != nil { b := datatypes.JSON(*req.FieldsToUpdate); updates["fields_to_update"] = b }
    if req.OverwriteStrategy != nil { updates["overwrite_strategy"] = *req.OverwriteStrategy }
    if req.Actions != nil { b := datatypes.JSON(*req.Actions); updates["actions"] = b }
    if err := ctl.svc.Update(id, updates); err != nil {
        if service.IsBadRequest(err) { c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()}); return }
        c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()}); return
    }
    c.Status(http.StatusNoContent)
}

func (ctl *SyncRulesController) Delete(c *gin.Context) {
    id, err := strconv.ParseUint(c.Param("id"), 10, 64)
    if err != nil || id == 0 { c.JSON(http.StatusBadRequest, gin.H{"message":"invalid id"}); return }
    if err := ctl.svc.Delete(id); err != nil {
        if service.IsBadRequest(err) { c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()}); return }
        c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()}); return
    }
    c.Status(http.StatusNoContent)
}

func (ctl *SyncRulesController) UpdatePriority(c *gin.Context) {
    id, err := strconv.ParseUint(c.Param("id"), 10, 64)
    if err != nil || id == 0 { c.JSON(http.StatusBadRequest, gin.H{"message":"invalid id"}); return }
    type reqT struct{ Priority int `json:"priority" binding:"required"` }
    var req reqT
    if err := c.ShouldBindJSON(&req); err != nil { c.JSON(http.StatusBadRequest, gin.H{"message":"invalid request"}); return }
    if err := ctl.svc.UpdatePriority(id, req.Priority); err != nil {
        if service.IsBadRequest(err) { c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()}); return }
        c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()}); return
    }
    c.Status(http.StatusNoContent)
}

func (ctl *SyncRulesController) SetEnabled(c *gin.Context) {
    id, err := strconv.ParseUint(c.Param("id"), 10, 64)
    if err != nil || id == 0 { c.JSON(http.StatusBadRequest, gin.H{"message":"invalid id"}); return }
    type reqT struct{ Enabled bool `json:"enabled" binding:"required"` }
    var req reqT
    if err := c.ShouldBindJSON(&req); err != nil { c.JSON(http.StatusBadRequest, gin.H{"message":"invalid request"}); return }
    if err := ctl.svc.SetEnabled(id, req.Enabled); err != nil {
        if service.IsBadRequest(err) { c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()}); return }
        c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()}); return
    }
    c.Status(http.StatusNoContent)
}
