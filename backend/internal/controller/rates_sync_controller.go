package controller

import (
    "net/http"

    "github.com/gin-gonic/gin"
    "nfa-dashboard/internal/service"
)

// RatesSyncController 提供执行客户费率同步的接口
// Base path: /api/v1/settlement/rates/sync

type RatesSyncController struct{ svc service.RatesSyncService }

func NewRatesSyncController(svc service.RatesSyncService) *RatesSyncController { return &RatesSyncController{svc: svc} }

// Execute 触发一次同步任务，返回受影响行数
func (ctl *RatesSyncController) Execute(c *gin.Context) {
    affected, err := ctl.svc.ExecuteSync()
    if err != nil {
        if service.IsBadRequest(err) { c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()}); return }
        c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
        return
    }
    c.JSON(http.StatusOK, gin.H{"affected": affected})
}
