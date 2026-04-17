package controller

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"nfa-dashboard/internal/service"
)

type SystemSettingsController struct {
	svc service.SystemSettingsService
}

func NewSystemSettingsController(svc service.SystemSettingsService) *SystemSettingsController {
	return &SystemSettingsController{svc: svc}
}

func (ctl *SystemSettingsController) GetTrafficSettings(c *gin.Context) {
	cfg, err := ctl.svc.GetTrafficSettings()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, cfg)
}

func (ctl *SystemSettingsController) UpdateTrafficSettings(c *gin.Context) {
	var req service.TrafficSettings
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid request"})
		return
	}
	cfg, err := ctl.svc.UpdateTrafficSettings(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, cfg)
}
