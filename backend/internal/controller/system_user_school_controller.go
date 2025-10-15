package controller

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"nfa-dashboard/internal/service"
)

// SystemUserSchoolController 提供用户-院校绑定控制器
type SystemUserSchoolController struct {
	svc service.UserSchoolService
}

func NewSystemUserSchoolController(svc service.UserSchoolService) *SystemUserSchoolController {
	return &SystemUserSchoolController{svc: svc}
}

// POST /api/v1/system/user-schools/owner
// body: { "school_id": "SCHOOL_001", "user_id": 123 }  // user_id 可省略或为0表示解绑
func (ctl *SystemUserSchoolController) SetOwner(c *gin.Context) {
	type reqT struct {
		SchoolID string  `json:"school_id" binding:"required"`
		UserID   *uint64 `json:"user_id"`
	}
	var req reqT
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid request"})
		return
	}
	if err := ctl.svc.SetOwner(req.SchoolID, req.UserID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}
