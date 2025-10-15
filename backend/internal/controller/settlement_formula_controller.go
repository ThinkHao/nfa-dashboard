package controller

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"nfa-dashboard/internal/service"
)

type SettlementFormulaController struct {
	service service.SettlementFormulaService
}

func NewSettlementFormulaController(s service.SettlementFormulaService) *SettlementFormulaController {
	return &SettlementFormulaController{service: s}
}

// List 列表
func (c *SettlementFormulaController) List(ctx *gin.Context) {
	limit := parseIntDefault(ctx.DefaultQuery("limit", "20"), 20)
	offset := parseIntDefault(ctx.DefaultQuery("offset", "0"), 0)
	items, total, err := c.service.List(limit, offset)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "获取结算公式列表失败", "error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"code": 200, "message": "OK", "data": gin.H{"total": total, "items": items}})
}

// Get 详情
func (c *SettlementFormulaController) Get(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil || id == 0 {
		ctx.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无效ID"})
		return
	}
	item, err := c.service.GetByID(id)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "获取结算公式失败", "error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"code": 200, "message": "OK", "data": item})
}

// Create 新建
func (c *SettlementFormulaController) Create(ctx *gin.Context) {
	type Req struct {
		Name        string          `json:"name" binding:"required"`
		Description string          `json:"description"`
		Tokens      json.RawMessage `json:"tokens" binding:"required"`
		Enabled     *bool           `json:"enabled"`
	}
	var req Req
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误", "error": err.Error()})
		return
	}
	uid, _ := currentUserID(ctx)
	enabled := true
	if req.Enabled != nil { enabled = *req.Enabled }
	item, err := c.service.Create(req.Name, req.Description, req.Tokens, enabled, strconv.FormatUint(uid, 10))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "创建失败", "error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"code": 200, "message": "创建成功", "data": item})
}

// Update 更新
func (c *SettlementFormulaController) Update(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil || id == 0 {
		ctx.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无效ID"})
		return
	}
	type Req struct {
		Name        string          `json:"name" binding:"required"`
		Description string          `json:"description"`
		Tokens      json.RawMessage `json:"tokens"` // 可选，空则不更新 tokens
		Enabled     *bool           `json:"enabled"`
	}
	var req Req
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误", "error": err.Error()})
		return
	}
	uid, _ := currentUserID(ctx)
	enabled := true
	if req.Enabled != nil { enabled = *req.Enabled }
	if err := c.service.Update(id, req.Name, req.Description, req.Tokens, enabled, strconv.FormatUint(uid, 10)); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "更新失败", "error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"code": 200, "message": "更新成功"})
}

// Delete 删除
func (c *SettlementFormulaController) Delete(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil || id == 0 {
		ctx.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无效ID"})
		return
	}
	if err := c.service.Delete(id); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "删除失败", "error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"code": 200, "message": "删除成功"})
}
