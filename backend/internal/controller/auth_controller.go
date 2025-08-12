package controller

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"nfa-dashboard/config"
	"nfa-dashboard/internal/middleware"
	"nfa-dashboard/internal/model"
	"nfa-dashboard/internal/security"
	"nfa-dashboard/internal/service"
)

type AuthController struct {
	authSvc service.AuthService
}

func NewAuthController(authSvc service.AuthService) *AuthController {
	return &AuthController{authSvc: authSvc}
}

// LoginRequest represents login payload
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

func (a *AuthController) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid request"})
		return
	}
	token, user, perms, err := a.authSvc.Login(req.Username, req.Password)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"message": err.Error()})
		return
	}
	// issue refresh token
	refreshToken, err := security.GenerateToken(user.ID, user.Username, config.GetRefreshTokenTTLMinutes())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to issue refresh token"})
		return
	}
	resp := gin.H{
		"token": token,
		"refresh_token": refreshToken,
		"user": gin.H{
			"id": user.ID,
			"username": user.Username,
			"email": user.Email,
			"phone": user.Phone,
		},
		"permissions": toPermissionCodes(perms),
	}
	c.JSON(http.StatusOK, resp)
}

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

// POST /api/v1/auth/refresh
func (a *AuthController) Refresh(c *gin.Context) {
	var req RefreshRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.RefreshToken == "" {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid request"})
		return
	}
	// parse refresh token
	claims, err := security.ParseToken(req.RefreshToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "invalid refresh token"})
		return
	}
	// load user & permissions
	user, err := a.authSvc.GetUserByID(claims.UserID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "user not found"})
		return
	}
	perms, err := a.authSvc.GetUserPermissions(user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "load permissions failed"})
		return
	}
	// issue new tokens (rotate refresh token)
	accessToken, err := security.GenerateToken(user.ID, user.Username, config.GetAccessTokenTTLMinutes())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to issue token"})
		return
	}
	newRefresh, err := security.GenerateToken(user.ID, user.Username, config.GetRefreshTokenTTLMinutes())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to issue refresh token"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"token": accessToken,
		"refresh_token": newRefresh,
		"user": gin.H{
			"id": user.ID,
			"username": user.Username,
			"email": user.Email,
			"phone": user.Phone,
		},
		"permissions": toPermissionCodes(perms),
	})
}

func (a *AuthController) Profile(c *gin.Context) {
	uVal, _ := c.Get(middleware.ContextUserKey)
	pVal, _ := c.Get(middleware.ContextPermissionsKey)
	user, _ := uVal.(*model.User)
	codes := toPermissionCodesFromAny(pVal)
	c.JSON(http.StatusOK, gin.H{
		"user": gin.H{
			"id": user.ID,
			"username": user.Username,
			"email": user.Email,
			"phone": user.Phone,
		},
		"permissions": codes,
	})
}

// helpers
func toPermissionCodes(perms []model.Permission) []string {
	res := make([]string, 0, len(perms))
	for _, p := range perms { res = append(res, p.Code) }
	return res
}

func toPermissionCodesFromAny(v interface{}) []string {
	switch p := v.(type) {
	case []model.Permission:
		return toPermissionCodes(p)
	case []string:
		return p
	default:
		return []string{}
	}
}
