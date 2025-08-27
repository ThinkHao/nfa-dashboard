package controller

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"nfa-dashboard/internal/service"
)

// SettlementEntitiesController hosts endpoints under /api/v1/settlement/entities
type SettlementEntitiesController struct{ svc service.EntitiesService }

func NewSettlementEntitiesController(svc service.EntitiesService) *SettlementEntitiesController { return &SettlementEntitiesController{svc: svc} }

func (ctl *SettlementEntitiesController) ListEntities(c *gin.Context) {
	page := parseIntDefault(c.Query("page"), 1)
	pageSize := parseIntDefault(c.Query("page_size"), 10)
	entityType := c.Query("entity_type")
	entityName := c.Query("entity_name")
	items, total, err := ctl.svc.List(entityType, entityName, page, pageSize)
	if err != nil { c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()}); return }
	c.JSON(http.StatusOK, gin.H{"items": items, "total": total})
}

func (ctl *SettlementEntitiesController) CreateEntity(c *gin.Context) {
	type reqT struct{
		EntityType string `json:"entity_type" binding:"required"`
		EntityName string `json:"entity_name" binding:"required"`
		ContactInfo *string `json:"contact_info"`
	}
	var req reqT
	if err := c.ShouldBindJSON(&req); err != nil { c.JSON(http.StatusBadRequest, gin.H{"message":"invalid request"}); return }
	e, err := ctl.svc.Create(req.EntityType, req.EntityName, req.ContactInfo)
	if err != nil {
		if service.IsBadRequest(err) { c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()}); return }
		c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()}); return
	}
	c.JSON(http.StatusOK, e)
}

func (ctl *SettlementEntitiesController) UpdateEntity(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 { c.JSON(http.StatusBadRequest, gin.H{"message":"invalid id"}); return }
	type reqT struct{
		EntityType *string `json:"entity_type"`
		EntityName *string `json:"entity_name"`
		ContactInfo *string `json:"contact_info"`
	}
	var req reqT
	if err := c.ShouldBindJSON(&req); err != nil { c.JSON(http.StatusBadRequest, gin.H{"message":"invalid request"}); return }
	if err := ctl.svc.Update(id, req.EntityType, req.EntityName, req.ContactInfo); err != nil {
		if service.IsBadRequest(err) { c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()}); return }
		c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()}); return
	}
	c.Status(http.StatusNoContent)
}

func (ctl *SettlementEntitiesController) DeleteEntity(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 { c.JSON(http.StatusBadRequest, gin.H{"message":"invalid id"}); return }
	if err := ctl.svc.Delete(id); err != nil {
		if service.IsBadRequest(err) { c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()}); return }
		c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()}); return
	}
	c.Status(http.StatusNoContent)
}
