package controller

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"nfa-dashboard/internal/model"
	"nfa-dashboard/internal/service"
)

type SystemUserController struct {
	userSvc service.UserService
}

func NewSystemUserController(userSvc service.UserService) *SystemUserController {
	return &SystemUserController{userSvc: userSvc}
}

// GET /api/v1/system/users
func (ctl *SystemUserController) ListUsers(c *gin.Context) {
    // ids 优先：逗号分隔的用户ID集合，存在时忽略其余过滤与分页
    if idsStr := strings.TrimSpace(c.Query("ids")); idsStr != "" {
        parts := strings.Split(idsStr, ",")
        ids := make([]uint64, 0, len(parts))
        for _, p := range parts {
            p = strings.TrimSpace(p)
            if p == "" { continue }
            if v, err := strconv.ParseUint(p, 10, 64); err == nil && v > 0 { ids = append(ids, v) }
        }
        users, err := ctl.userSvc.FindByIDs(ids)
        if err != nil { c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()}); return }
        type userResp struct {
            model.User `json:",inline"`
            Roles []model.Role `json:"roles"`
            DisplayName string `json:"display_name,omitempty"`
        }
        respItems := make([]userResp, 0, len(users))
        for _, u := range users {
            roles, _ := ctl.userSvc.GetUserRoles(u.ID)
            dn := ""
            if u.Alias != nil && strings.TrimSpace(*u.Alias) != "" { dn = strings.TrimSpace(*u.Alias) } else if strings.TrimSpace(u.Username) != "" { dn = strings.TrimSpace(u.Username) } else { dn = fmt.Sprintf("用户#%d", u.ID) }
            respItems = append(respItems, userResp{User: u, Roles: roles, DisplayName: dn})
        }
        c.JSON(http.StatusOK, gin.H{"items": respItems, "total": len(respItems)})
        return
    }

    username := c.Query("username")
	var statusPtr *int8
	if s := c.Query("status"); s != "" {
		if v, err := strconv.Atoi(s); err == nil {
			vv := int8(v)
			statusPtr = &vv
		}
	}
	// 支持通过角色过滤，roles=逗号分隔或单个 role
	roles := make([]string, 0)
	if r := c.Query("roles"); r != "" {
		for _, p := range strings.Split(r, ",") {
			if v := strings.TrimSpace(p); v != "" { roles = append(roles, v) }
		}
	} else if r := c.Query("role"); r != "" {
		if v := strings.TrimSpace(r); v != "" { roles = append(roles, v) }
	}
	page := parseIntDefault(c.Query("page"), 1)
	pageSize := parseIntDefault(c.Query("page_size"), 10)
	items, total, err := ctl.userSvc.List(username, statusPtr, roles, page, pageSize)
	if err != nil { c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()}); return }
	// attach roles per user to match frontend display
	type userResp struct {
		model.User `json:",inline"`
		Roles []model.Role `json:"roles"`
		DisplayName string `json:"display_name,omitempty"`
	}
	respItems := make([]userResp, 0, len(items))
	for _, u := range items {
		roles, _ := ctl.userSvc.GetUserRoles(u.ID) // ignore error to avoid breaking list
		dn := ""
		if u.Alias != nil && strings.TrimSpace(*u.Alias) != "" { dn = strings.TrimSpace(*u.Alias) } else if strings.TrimSpace(u.Username) != "" { dn = strings.TrimSpace(u.Username) } else { dn = fmt.Sprintf("用户#%d", u.ID) }
		respItems = append(respItems, userResp{User: u, Roles: roles, DisplayName: dn})
	}
	c.JSON(http.StatusOK, gin.H{"items": respItems, "total": total})
}

// POST /api/v1/system/users
func (ctl *SystemUserController) CreateUser(c *gin.Context) {
	type reqT struct{
		Username string   `json:"username" binding:"required"`
		Alias    *string  `json:"alias"`
		Password string   `json:"password" binding:"required"`
		Email    *string  `json:"email"`
		Phone    *string  `json:"phone"`
		Status   *int8    `json:"status"`
		RoleIDs  []uint64 `json:"role_ids"`
	}
	var req reqT
	if err := c.ShouldBindJSON(&req); err != nil { c.JSON(http.StatusBadRequest, gin.H{"message":"invalid request"}); return }
	u, err := ctl.userSvc.Create(req.Username, req.Alias, req.Password, req.Email, req.Phone, req.Status, req.RoleIDs)
	if err != nil {
		if service.IsBadRequest(err) { c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()}); return }
		c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()}); return
	}
	c.JSON(http.StatusOK, u)
}

// PUT /api/v1/system/users/:id/status
func (ctl *SystemUserController) UpdateUserStatus(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 { c.JSON(http.StatusBadRequest, gin.H{"message":"invalid id"}); return }
	type reqT struct{ Status int8 `json:"status" binding:"required"` }
	var req reqT
	if err := c.ShouldBindJSON(&req); err != nil { c.JSON(http.StatusBadRequest, gin.H{"message":"invalid request"}); return }
	if err := ctl.userSvc.UpdateStatus(id, req.Status); err != nil { c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()}); return }
	c.Status(http.StatusNoContent)
}

// PUT /api/v1/system/users/:id/roles
func (ctl *SystemUserController) SetUserRoles(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 { c.JSON(http.StatusBadRequest, gin.H{"message":"invalid id"}); return }
	type reqT struct{ RoleIDs []uint64 `json:"role_ids"` }
	var req reqT
	if err := c.ShouldBindJSON(&req); err != nil { c.JSON(http.StatusBadRequest, gin.H{"message":"invalid request"}); return }
	if err := ctl.userSvc.SetRoles(id, req.RoleIDs); err != nil {
		if service.IsBadRequest(err) { c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()}); return }
		c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()}); return
	}
	c.Status(http.StatusNoContent)
}

// PUT /api/v1/system/users/:id/alias
func (ctl *SystemUserController) UpdateUserAlias(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 { c.JSON(http.StatusBadRequest, gin.H{"message":"invalid id"}); return }
	type reqT struct{ Alias *string `json:"alias"` }
	var req reqT
	if err := c.ShouldBindJSON(&req); err != nil { c.JSON(http.StatusBadRequest, gin.H{"message":"invalid request"}); return }
	if err := ctl.userSvc.UpdateAlias(id, req.Alias); err != nil {
		if service.IsBadRequest(err) { c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()}); return }
		c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()}); return
	}
	c.Status(http.StatusNoContent)
}
