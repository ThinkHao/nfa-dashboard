package controller

import (
    "strconv"
    "github.com/gin-gonic/gin"
    "nfa-dashboard/internal/middleware"
    "nfa-dashboard/internal/model"
)

// parseIntDefault 解析正整数，非法或空返回默认值
func parseIntDefault(s string, def int) int {
    if s == "" { return def }
    if v, err := strconv.Atoi(s); err == nil && v > 0 { return v }
    return def
}

// hasPermission 检查当前上下文是否包含指定权限码
func hasPermission(c *gin.Context, code string) bool {
    val, ok := c.Get(middleware.ContextPermissionsKey)
    if !ok { return false }
    if perms, ok := val.([]model.Permission); ok {
        for _, p := range perms { if p.Code == code { return true } }
        return false
    }
    if list, ok := val.([]string); ok {
        for _, s := range list { if s == code { return true } }
    }
    return false
}

// hasAnyPermission 检查是否拥有任意一个权限码
func hasAnyPermission(c *gin.Context, codes ...string) bool {
    for _, code := range codes {
        if hasPermission(c, code) { return true }
    }
    return false
}

// currentUserID 从上下文中获取当前用户ID
func currentUserID(c *gin.Context) (uint64, bool) {
    val, ok := c.Get(middleware.ContextUserKey)
    if !ok || val == nil { return 0, false }
    if u, ok := val.(*model.User); ok && u != nil { return u.ID, true }
    return 0, false
}
