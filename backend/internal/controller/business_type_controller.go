package controller

import (
    "net/http"
    "strconv"

    "github.com/gin-gonic/gin"
    "nfa-dashboard/internal/service"
)

// BusinessTypeController provides endpoints for business types management
// Base path: /api/v1/settlement/business-types

type BusinessTypeController struct{ svc service.BusinessTypeService }

func NewBusinessTypeController(svc service.BusinessTypeService) *BusinessTypeController { return &BusinessTypeController{svc: svc} }

func (ctl *BusinessTypeController) List(c *gin.Context) {
    page := parseIntDefault(c.Query("page"), 1)
    pageSize := parseIntDefault(c.Query("page_size"), 10)
    code := c.Query("code")
    name := c.Query("name")
    var enabledPtr *bool
    if v := c.Query("enabled"); v != "" {
        b := v == "1" || v == "true"
        enabledPtr = &b
    }
    items, total, err := ctl.svc.List(code, name, enabledPtr, page, pageSize)
    if err != nil { c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()}); return }
    c.JSON(http.StatusOK, gin.H{"items": items, "total": total})
}

func (ctl *BusinessTypeController) Create(c *gin.Context) {
    type reqT struct {
        Code        string  `json:"code" binding:"required"`
        Name        string  `json:"name" binding:"required"`
        Description *string `json:"description"`
        Enabled     *bool   `json:"enabled"`
    }
    var req reqT
    if err := c.ShouldBindJSON(&req); err != nil { c.JSON(http.StatusBadRequest, gin.H{"message": "invalid request"}); return }
    bt, err := ctl.svc.Create(req.Code, req.Name, req.Description, req.Enabled)
    if err != nil {
        if service.IsBadRequest(err) { c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()}); return }
        c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()}); return
    }
    c.JSON(http.StatusOK, bt)
}

func (ctl *BusinessTypeController) Update(c *gin.Context) {
    id, err := strconv.ParseUint(c.Param("id"), 10, 64)
    if err != nil || id == 0 { c.JSON(http.StatusBadRequest, gin.H{"message": "invalid id"}); return }
    type reqT struct {
        Name        *string `json:"name"`
        Description *string `json:"description"`
        Enabled     *bool   `json:"enabled"`
    }
    var req reqT
    if err := c.ShouldBindJSON(&req); err != nil { c.JSON(http.StatusBadRequest, gin.H{"message": "invalid request"}); return }
    if err := ctl.svc.Update(id, req.Name, req.Description, req.Enabled); err != nil {
        if service.IsBadRequest(err) { c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()}); return }
        c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()}); return
    }
    c.Status(http.StatusNoContent)
}

func (ctl *BusinessTypeController) Delete(c *gin.Context) {
    id, err := strconv.ParseUint(c.Param("id"), 10, 64)
    if err != nil || id == 0 { c.JSON(http.StatusBadRequest, gin.H{"message": "invalid id"}); return }
    if err := ctl.svc.Delete(id); err != nil {
        if service.IsBadRequest(err) { c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()}); return }
        c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()}); return
    }
    c.Status(http.StatusNoContent)
}
