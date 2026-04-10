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

// FilterRulesController manages endpoints under /api/v1/settlement/rates/filter-rules
type FilterRulesController struct{ svc service.FilterRulesService }

func NewFilterRulesController(svc service.FilterRulesService) *FilterRulesController {
	return &FilterRulesController{svc: svc}
}

func (ctl *FilterRulesController) ListOptions(c *gin.Context) {
	regions, cps, err := ctl.svc.ListOptions()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"regions": regions, "cps": cps})
}

func (ctl *FilterRulesController) List(c *gin.Context) {
	page := parseIntDefault(c.Query("page"), 1)
	pageSize := parseIntDefault(c.Query("page_size"), 10)
	name := c.Query("name")
	var enabledPtr *bool
	if v := c.Query("enabled"); v != "" {
		b := v == "1" || v == "true"
		enabledPtr = &b
	}
	items, total, err := ctl.svc.List(name, enabledPtr, page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": items, "total": total})
}

func (ctl *FilterRulesController) Create(c *gin.Context) {
	type reqT struct {
		Name                string           `json:"name" binding:"required"`
		Enabled             bool             `json:"enabled"`
		Priority            int              `json:"priority"`
		ScopeRegion         *json.RawMessage `json:"scope_region"`
		ScopeCP             *json.RawMessage `json:"scope_cp"`
		SchoolNameMatchType string           `json:"school_name_match_type"`
		SchoolNameValues    *json.RawMessage `json:"school_name_values"`
	}
	var req reqT
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid request"})
		return
	}
	rule := &model.RateCustomerFilterRule{
		Name:                req.Name,
		Enabled:             req.Enabled,
		Priority:            req.Priority,
		SchoolNameMatchType: req.SchoolNameMatchType,
	}
	if req.ScopeRegion != nil {
		rule.ScopeRegion = datatypes.JSON(*req.ScopeRegion)
	}
	if req.ScopeCP != nil {
		rule.ScopeCP = datatypes.JSON(*req.ScopeCP)
	}
	if req.SchoolNameValues != nil {
		rule.SchoolNameValues = datatypes.JSON(*req.SchoolNameValues)
	}
	out, err := ctl.svc.Create(rule)
	if err != nil {
		if service.IsBadRequest(err) {
			c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, out)
}

func (ctl *FilterRulesController) Update(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid id"})
		return
	}
	type reqT struct {
		Name                *string          `json:"name"`
		Enabled             *bool            `json:"enabled"`
		Priority            *int             `json:"priority"`
		ScopeRegion         *json.RawMessage `json:"scope_region"`
		ScopeCP             *json.RawMessage `json:"scope_cp"`
		SchoolNameMatchType *string          `json:"school_name_match_type"`
		SchoolNameValues    *json.RawMessage `json:"school_name_values"`
	}
	var req reqT
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid request"})
		return
	}
	updates := map[string]interface{}{}
	if req.Name != nil {
		updates["name"] = *req.Name
	}
	if req.Enabled != nil {
		updates["enabled"] = *req.Enabled
	}
	if req.Priority != nil {
		updates["priority"] = *req.Priority
	}
	if req.ScopeRegion != nil {
		updates["scope_region"] = datatypes.JSON(*req.ScopeRegion)
	}
	if req.ScopeCP != nil {
		updates["scope_cp"] = datatypes.JSON(*req.ScopeCP)
	}
	if req.SchoolNameMatchType != nil {
		updates["school_name_match_type"] = *req.SchoolNameMatchType
	}
	if req.SchoolNameValues != nil {
		updates["school_name_values"] = datatypes.JSON(*req.SchoolNameValues)
	}
	if err := ctl.svc.Update(id, updates); err != nil {
		if service.IsBadRequest(err) {
			c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}

func (ctl *FilterRulesController) Delete(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid id"})
		return
	}
	if err := ctl.svc.Delete(id); err != nil {
		if service.IsBadRequest(err) {
			c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}

func (ctl *FilterRulesController) UpdatePriority(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid id"})
		return
	}
	type reqT struct {
		Priority int `json:"priority" binding:"required"`
	}
	var req reqT
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid request"})
		return
	}
	if err := ctl.svc.UpdatePriority(id, req.Priority); err != nil {
		if service.IsBadRequest(err) {
			c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}

func (ctl *FilterRulesController) SetEnabled(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid id"})
		return
	}
	type reqT struct {
		Enabled bool `json:"enabled" binding:"required"`
	}
	var req reqT
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid request"})
		return
	}
	if err := ctl.svc.SetEnabled(id, req.Enabled); err != nil {
		if service.IsBadRequest(err) {
			c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}
