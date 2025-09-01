package controller

import (
    "encoding/json"
    "net/http"
    "strconv"

    "github.com/gin-gonic/gin"
    "gorm.io/datatypes"
    "nfa-dashboard/internal/model"
    "nfa-dashboard/internal/service"
)

// CustomerFieldsController manages endpoints under /api/v1/settlement/rates/customer-fields

type CustomerFieldsController struct{ svc service.CustomerFieldsService }

func NewCustomerFieldsController(svc service.CustomerFieldsService) *CustomerFieldsController { return &CustomerFieldsController{svc: svc} }

func (ctl *CustomerFieldsController) List(c *gin.Context) {
    page := parseIntDefault(c.Query("page"), 1)
    pageSize := parseIntDefault(c.Query("page_size"), 10)
    fieldKey := c.Query("field_key")
    label := c.Query("label")
    dataType := c.Query("data_type")
    var enabledPtr *bool
    if v := c.Query("enabled"); v != "" {
        b := v == "1" || v == "true"
        enabledPtr = &b
    }
    items, total, err := ctl.svc.List(fieldKey, label, dataType, enabledPtr, page, pageSize)
    if err != nil { c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()}); return }
    c.JSON(http.StatusOK, gin.H{"items": items, "total": total})
}

func (ctl *CustomerFieldsController) Create(c *gin.Context) {
    type reqT struct{
        FieldKey      string           `json:"field_key" binding:"required"`
        Label         string           `json:"label" binding:"required"`
        DataType      string           `json:"data_type" binding:"required"`
        Required      bool             `json:"required"`
        DefaultValue  *json.RawMessage `json:"default_value"`
        ValidateRegex *string          `json:"validate_regex"`
        Min           *float64         `json:"min"`
        Max           *float64         `json:"max"`
        Precision     *int             `json:"precision"`
        EnumOptions   *json.RawMessage `json:"enum_options"`
        UsableInRules bool             `json:"usable_in_rules"`
        Enabled       bool             `json:"enabled"`
    }
    var req reqT
    if err := c.ShouldBindJSON(&req); err != nil { c.JSON(http.StatusBadRequest, gin.H{"message":"invalid request"}); return }
    def := &model.RateCustomerCustomFieldDef{
        FieldKey: req.FieldKey,
        Label: req.Label,
        DataType: req.DataType,
        Required: req.Required,
        ValidateRegex: req.ValidateRegex,
        Min: req.Min,
        Max: req.Max,
        Precision: req.Precision,
        UsableInRules: req.UsableInRules,
        Enabled: req.Enabled,
    }
    if req.DefaultValue != nil { def.DefaultValue = datatypes.JSON(*req.DefaultValue) }
    if req.EnumOptions != nil { def.EnumOptions = datatypes.JSON(*req.EnumOptions) }
    out, err := ctl.svc.Create(def)
    if err != nil {
        if service.IsBadRequest(err) { c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()}); return }
        c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()}); return
    }
    c.JSON(http.StatusOK, out)
}

func (ctl *CustomerFieldsController) Update(c *gin.Context) {
    id, err := strconv.ParseUint(c.Param("id"), 10, 64)
    if err != nil || id == 0 { c.JSON(http.StatusBadRequest, gin.H{"message":"invalid id"}); return }
    type reqT struct{
        Label         *string          `json:"label"`
        DataType      *string          `json:"data_type"`
        Required      *bool            `json:"required"`
        DefaultValue  *json.RawMessage `json:"default_value"`
        ValidateRegex *string          `json:"validate_regex"`
        Min           *float64         `json:"min"`
        Max           *float64         `json:"max"`
        Precision     *int             `json:"precision"`
        EnumOptions   *json.RawMessage `json:"enum_options"`
        UsableInRules *bool            `json:"usable_in_rules"`
        Enabled       *bool            `json:"enabled"`
    }
    var req reqT
    if err := c.ShouldBindJSON(&req); err != nil { c.JSON(http.StatusBadRequest, gin.H{"message":"invalid request"}); return }
    updates := map[string]interface{}{}
    if req.Label != nil { updates["label"] = *req.Label }
    if req.DataType != nil { updates["data_type"] = *req.DataType }
    if req.Required != nil { updates["required"] = *req.Required }
    if req.DefaultValue != nil { b := datatypes.JSON(*req.DefaultValue); updates["default_value"] = b }
    if req.ValidateRegex != nil { updates["validate_regex"] = *req.ValidateRegex }
    if req.Min != nil { updates["min"] = *req.Min }
    if req.Max != nil { updates["max"] = *req.Max }
    if req.Precision != nil { updates["precision"] = *req.Precision }
    if req.EnumOptions != nil { b := datatypes.JSON(*req.EnumOptions); updates["enum_options"] = b }
    if req.UsableInRules != nil { updates["usable_in_rules"] = *req.UsableInRules }
    if req.Enabled != nil { updates["enabled"] = *req.Enabled }
    if err := ctl.svc.Update(id, updates); err != nil {
        if service.IsBadRequest(err) { c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()}); return }
        c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()}); return
    }
    c.Status(http.StatusNoContent)
}

func (ctl *CustomerFieldsController) Delete(c *gin.Context) {
    id, err := strconv.ParseUint(c.Param("id"), 10, 64)
    if err != nil || id == 0 { c.JSON(http.StatusBadRequest, gin.H{"message":"invalid id"}); return }
    if err := ctl.svc.Delete(id); err != nil {
        if service.IsBadRequest(err) { c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()}); return }
        c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()}); return
    }
    c.Status(http.StatusNoContent)
}
