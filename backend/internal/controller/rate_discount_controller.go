package controller

import (
	"net/http"
	"strconv"

	"nfa-dashboard/internal/model"
	"nfa-dashboard/internal/service"

	"github.com/gin-gonic/gin"
	"gorm.io/datatypes"
)

// RateDiscountController manages endpoints under /api/v1/settlement/rates/discount-rules

type RateDiscountController struct{ svc service.RateDiscountService }

func NewRateDiscountController(svc service.RateDiscountService) *RateDiscountController {
	return &RateDiscountController{svc: svc}
}

func (ctl *RateDiscountController) List(c *gin.Context) {
	page := parseIntDefault(c.Query("page"), 1)
	pageSize := parseIntDefault(c.Query("page_size"), 10)
	name := c.Query("name")
	scopeType := c.Query("scope_type")
	var enabledPtr *bool
	if v := c.Query("enabled"); v != "" {
		b := v == "1" || v == "true"
		enabledPtr = &b
	}
	items, total, err := ctl.svc.List(name, scopeType, enabledPtr, page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": items, "total": total})
}

func (ctl *RateDiscountController) Get(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid id"})
		return
	}
	rule, items, err := ctl.svc.Get(id)
	if err != nil {
		if service.IsBadRequest(err) {
			c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"rule": rule, "items": items})
}

func (ctl *RateDiscountController) Create(c *gin.Context) {
	type itemReq struct {
		FromYear     int     `json:"from_year" binding:"required"`
		ToYear       *int    `json:"to_year"`
		DiscountRate float64 `json:"discount_rate" binding:"required"`
	}
	type reqT struct {
		Name      string         `json:"name" binding:"required"`
		ScopeType string         `json:"scope_type"`
		ScopeKey  *string        `json:"scope_key"`
		Fields    datatypes.JSON `json:"fields"`
		Enabled   bool           `json:"enabled"`
		Priority  int            `json:"priority"`
		Items     []itemReq      `json:"items"`
	}
	var req reqT
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid request"})
		return
	}
	rule := &model.RateDiscountRule{
		Name:      req.Name,
		ScopeType: req.ScopeType,
		ScopeKey:  req.ScopeKey,
		Fields:    req.Fields,
		Enabled:   req.Enabled,
		Priority:  req.Priority,
	}
	items := make([]model.RateDiscountRuleItem, 0, len(req.Items))
	for _, it := range req.Items {
		items = append(items, model.RateDiscountRuleItem{
			FromYear:     it.FromYear,
			ToYear:       it.ToYear,
			DiscountRate: it.DiscountRate,
		})
	}
	outRule, outItems, err := ctl.svc.Create(rule, items)
	if err != nil {
		if service.IsBadRequest(err) {
			c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"rule": outRule, "items": outItems})
}

func (ctl *RateDiscountController) Update(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid id"})
		return
	}
	var req map[string]interface{}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid request"})
		return
	}
	if err := ctl.svc.Update(id, req); err != nil {
		if service.IsBadRequest(err) {
			c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}

func (ctl *RateDiscountController) Delete(c *gin.Context) {
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

func (ctl *RateDiscountController) ReplaceItems(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid id"})
		return
	}
	type itemReq struct {
		FromYear     int     `json:"from_year" binding:"required"`
		ToYear       *int    `json:"to_year"`
		DiscountRate float64 `json:"discount_rate" binding:"required"`
	}
	var req []itemReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid request"})
		return
	}
	items := make([]model.RateDiscountRuleItem, 0, len(req))
	for _, it := range req {
		items = append(items, model.RateDiscountRuleItem{
			FromYear:     it.FromYear,
			ToYear:       it.ToYear,
			DiscountRate: it.DiscountRate,
		})
	}
	if err := ctl.svc.ReplaceItems(id, items); err != nil {
		if service.IsBadRequest(err) {
			c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}
