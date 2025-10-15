package controller

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"nfa-dashboard/config"
)

// SystemBindingController 暴露与“绑定配置”相关的只读接口
// 例如：允许被绑定为院校可见用户的角色列表
// 路由建议：/api/v1/system/binding/allowed-user-roles
// 受 system.user.manage 保护

type SystemBindingController struct{}

func NewSystemBindingController() *SystemBindingController { return &SystemBindingController{} }
// GET /api/v1/system/binding/allowed-user-roles
func (ctl *SystemBindingController) GetAllowedUserRoles(c *gin.Context) {
	// type: sales | line | ""(compat->sales)
	t := strings.TrimSpace(strings.ToLower(c.Query("type")))
	var roles []string
	switch t {
	case "node":
		roles = config.GetAllowedNodeRoles()
		if len(roles) == 0 { roles = config.GetAllowedLineRoles() }
	case "line":
		roles = config.GetAllowedLineRoles()
		if len(roles) == 0 { roles = config.GetOwnerRoles("network_line_fee") }
	case "sales", "":
		roles = config.GetAllowedSalesRoles()
		if len(roles) == 0 { roles = config.GetOwnerRoles("customer_fee") }
	default:
		roles = config.GetAllowedSalesRoles()
	}
	// 规范化：去空白与小写（前端展示可自行处理大小写，这里只为一致性）
	norm := make([]string, 0, len(roles))
	for _, r := range roles {
		r = strings.TrimSpace(r)
		if r == "" { continue }
		norm = append(norm, r)
	}
	c.JSON(http.StatusOK, gin.H{"items": norm, "total": len(norm)})
}
