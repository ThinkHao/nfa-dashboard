package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"nfa-dashboard/internal/model"
	"nfa-dashboard/internal/security"
	"nfa-dashboard/internal/service"
)

const (
	ContextUserKey        = "currentUser"
	ContextPermissionsKey = "currentPermissions"
	ContextClaimsKey      = "jwtClaims"
)

type AuthMiddleware struct {
	authSvc service.AuthService
}

func NewAuthMiddleware(authSvc service.AuthService) *AuthMiddleware {
	return &AuthMiddleware{authSvc: authSvc}
}

// AuthRequired validates JWT and loads user & permissions into context
func (m *AuthMiddleware) AuthRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		auth := c.GetHeader("Authorization")
		if auth == "" || !strings.HasPrefix(auth, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"message": "missing or invalid token"})
			return
		}
		tokenStr := strings.TrimPrefix(auth, "Bearer ")
		claims, err := security.ParseToken(tokenStr)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"message": "invalid token"})
			return
		}
		user, err := m.authSvc.GetUserByID(claims.UserID)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"message": "user not found or disabled"})
			return
		}
		perms, _ := m.authSvc.GetUserPermissions(claims.UserID)
		c.Set(ContextUserKey, user)
		c.Set(ContextPermissionsKey, perms)
		c.Set(ContextClaimsKey, claims)
		c.Next()
	}
}

// PermissionRequired checks if current user has all required permissions
func (m *AuthMiddleware) PermissionRequired(required ...string) gin.HandlerFunc {
	reqSet := map[string]struct{}{}
	for _, r := range required { reqSet[r] = struct{}{} }
	return func(c *gin.Context) {
		val, ok := c.Get(ContextPermissionsKey)
		if !ok {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"message": "forbidden"})
			return
		}
		codes := map[string]struct{}{}
		if perms, ok := val.([]model.Permission); ok {
			for _, p := range perms { codes[p.Code] = struct{}{} }
		} else if list, ok := val.([]string); ok {
			for _, s := range list { codes[s] = struct{}{} }
		}
		for r := range reqSet {
			if _, ok := codes[r]; !ok {
				c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"message": "permission denied", "missing": r})
				return
			}
		}
		c.Next()
	}
}
