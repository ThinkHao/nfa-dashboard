package controller

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"nfa-dashboard/internal/service"
)

type SystemRoleController struct {
	roleSvc service.RoleService
}

func NewSystemRoleController(roleSvc service.RoleService) *SystemRoleController {
	return &SystemRoleController{roleSvc: roleSvc}
}

// GET /api/v1/system/roles
func (ctl *SystemRoleController) ListRoles(c *gin.Context) {
	page := parseIntDefault(c.Query("page"), 1)
	pageSize := parseIntDefault(c.Query("page_size"), 10)
	items, total, err := ctl.roleSvc.List(page, pageSize)
	if err != nil { c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()}); return }
	c.JSON(http.StatusOK, gin.H{"items": items, "total": total})
}

// POST /api/v1/system/roles
func (ctl *SystemRoleController) CreateRole(c *gin.Context) {
	type reqT struct{
		Name string `json:"name" binding:"required"`
		Description *string `json:"description"`
	}
	var req reqT
	if err := c.ShouldBindJSON(&req); err != nil { c.JSON(http.StatusBadRequest, gin.H{"message":"invalid request"}); return }
	role, err := ctl.roleSvc.Create(req.Name, req.Description)
	if err != nil { c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()}); return }
	c.JSON(http.StatusOK, role)
}

// PUT /api/v1/system/roles/:id
func (ctl *SystemRoleController) UpdateRole(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 { c.JSON(http.StatusBadRequest, gin.H{"message":"invalid id"}); return }
	type reqT struct{
		Name *string `json:"name"`
		Description *string `json:"description"`
	}
	var req reqT
	if err := c.ShouldBindJSON(&req); err != nil { c.JSON(http.StatusBadRequest, gin.H{"message":"invalid request"}); return }
	if err := ctl.roleSvc.Update(id, req.Name, req.Description); err != nil { c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()}); return }
	c.Status(http.StatusNoContent)
}

// DELETE /api/v1/system/roles/:id
func (ctl *SystemRoleController) DeleteRole(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 { c.JSON(http.StatusBadRequest, gin.H{"message":"invalid id"}); return }
	if err := ctl.roleSvc.Delete(id); err != nil { c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()}); return }
	c.Status(http.StatusNoContent)
}

// GET /api/v1/system/roles/:id/permissions
func (ctl *SystemRoleController) GetRolePermissions(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 { c.JSON(http.StatusBadRequest, gin.H{"message":"invalid id"}); return }
	perms, err := ctl.roleSvc.GetPermissions(id)
	if err != nil { c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()}); return }
	// Return plain array to match frontend expectation (api.system.roles.getPermissions)
	c.JSON(http.StatusOK, perms)
}

// PUT /api/v1/system/roles/:id/permissions
func (ctl *SystemRoleController) SetRolePermissions(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 { c.JSON(http.StatusBadRequest, gin.H{"message":"invalid id"}); return }
	type reqT struct{ PermissionIDs []uint64 `json:"permission_ids"` }
	var req reqT
	if err := c.ShouldBindJSON(&req); err != nil { c.JSON(http.StatusBadRequest, gin.H{"message":"invalid request"}); return }
	if err := ctl.roleSvc.SetPermissions(id, req.PermissionIDs); err != nil {
        if service.IsBadRequest(err) { c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()}); return }
        c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()}); return
    }
	c.Status(http.StatusNoContent)
}

// helpers
func parseIntDefault(s string, def int) int {
	if s == "" { return def }
	if v, err := strconv.Atoi(s); err == nil && v > 0 { return v }
	return def
}
