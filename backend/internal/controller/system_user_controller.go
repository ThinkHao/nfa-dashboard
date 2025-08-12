package controller

import (
	"net/http"
	"strconv"

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
	username := c.Query("username")
	var statusPtr *int8
	if s := c.Query("status"); s != "" {
		if v, err := strconv.Atoi(s); err == nil {
			vv := int8(v)
			statusPtr = &vv
		}
	}
	page := parseIntDefault(c.Query("page"), 1)
	pageSize := parseIntDefault(c.Query("page_size"), 10)
	items, total, err := ctl.userSvc.List(username, statusPtr, page, pageSize)
	if err != nil { c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()}); return }
	// attach roles per user to match frontend display
	type userResp struct {
		model.User `json:",inline"`
		Roles []model.Role `json:"roles"`
	}
	respItems := make([]userResp, 0, len(items))
	for _, u := range items {
		roles, _ := ctl.userSvc.GetUserRoles(u.ID) // ignore error to avoid breaking list
		respItems = append(respItems, userResp{User: u, Roles: roles})
	}
	c.JSON(http.StatusOK, gin.H{"items": respItems, "total": total})
}

// POST /api/v1/system/users
func (ctl *SystemUserController) CreateUser(c *gin.Context) {
	type reqT struct{
		Username string   `json:"username" binding:"required"`
		Password string   `json:"password" binding:"required"`
		Email    *string  `json:"email"`
		Phone    *string  `json:"phone"`
		Status   *int8    `json:"status"`
		RoleIDs  []uint64 `json:"role_ids"`
	}
	var req reqT
	if err := c.ShouldBindJSON(&req); err != nil { c.JSON(http.StatusBadRequest, gin.H{"message":"invalid request"}); return }
	u, err := ctl.userSvc.Create(req.Username, req.Password, req.Email, req.Phone, req.Status, req.RoleIDs)
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
