package controller

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"nfa-dashboard/internal/service"
)

type SystemPermissionController struct {
	permSvc service.PermissionService
}

func NewSystemPermissionController(permSvc service.PermissionService) *SystemPermissionController {
	return &SystemPermissionController{permSvc: permSvc}
}

// GET /api/v1/system/permissions
func (ctl *SystemPermissionController) ListPermissions(c *gin.Context) {
	page := parseIntDefault(c.Query("page"), 1)
	pageSize := parseIntDefault(c.Query("page_size"), 100)
	keyword := c.Query("keyword")
	items, total, err := ctl.permSvc.ListFiltered(page, pageSize, keyword)
	if err != nil { c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()}); return }
	c.JSON(http.StatusOK, gin.H{"items": items, "total": total})
}

// POST /api/v1/system/permissions
func (ctl *SystemPermissionController) CreatePermission(c *gin.Context) {
	type reqT struct {
		Code        string `json:"code" binding:"required"`
		Name        string `json:"name" binding:"required"`
		Description *string `json:"description"`
	}
	var req reqT
	if err := c.ShouldBindJSON(&req); err != nil { c.JSON(http.StatusBadRequest, gin.H{"message": "invalid request"}); return }
	p, err := ctl.permSvc.Create(req.Code, req.Name, req.Description)
	if err != nil {
		if service.IsBadRequest(err) { c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()}); return }
		c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()}); return
	}
	c.JSON(http.StatusOK, p)
}

// GET /api/v1/system/permissions/:id
func (ctl *SystemPermissionController) GetPermission(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 { c.JSON(http.StatusBadRequest, gin.H{"message": "invalid id"}); return }
	p, err := ctl.permSvc.GetByID(id)
	if err != nil { c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()}); return }
	c.JSON(http.StatusOK, p)
}

// PUT /api/v1/system/permissions/:id
func (ctl *SystemPermissionController) UpdatePermission(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 { c.JSON(http.StatusBadRequest, gin.H{"message": "invalid id"}); return }
	type reqT struct {
		Name        *string `json:"name"`
		Description *string `json:"description"`
	}
	var req reqT
	if err := c.ShouldBindJSON(&req); err != nil { c.JSON(http.StatusBadRequest, gin.H{"message": "invalid request"}); return }
	if err := ctl.permSvc.Update(id, req.Name, req.Description); err != nil {
		if service.IsBadRequest(err) { c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()}); return }
		c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()}); return
	}
	c.Status(http.StatusNoContent)
}

// DELETE /api/v1/system/permissions/:id
func (ctl *SystemPermissionController) DisablePermission(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 { c.JSON(http.StatusBadRequest, gin.H{"message": "invalid id"}); return }
	if err := ctl.permSvc.Disable(id); err != nil {
		if service.IsBadRequest(err) { c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()}); return }
		c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()}); return
	}
	c.Status(http.StatusNoContent)
}

// POST /api/v1/system/permissions/sync
func (ctl *SystemPermissionController) SyncPermissions(c *gin.Context) {
	if err := ctl.permSvc.SyncFromCode(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()}); return
	}
	c.Status(http.StatusNoContent)
}
