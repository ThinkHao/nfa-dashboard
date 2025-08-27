package controller

import (
    "net/http"

    "github.com/gin-gonic/gin"
    "nfa-dashboard/internal/model"
    "nfa-dashboard/internal/service"
)

// SettlementRatesController hosts endpoints under /api/v1/settlement/rates
type SettlementRatesController struct{ svc service.RatesService }

func NewSettlementRatesController(svc service.RatesService) *SettlementRatesController { return &SettlementRatesController{svc: svc} }

// Customer business rates
func (ctl *SettlementRatesController) ListCustomerRates(c *gin.Context) {
    page := parseIntDefault(c.Query("page"), 1)
    pageSize := parseIntDefault(c.Query("page_size"), 10)
    region := c.Query("region")
    cp := c.Query("cp")
    schoolName := c.Query("school_name")
    items, total, err := ctl.svc.ListCustomerRates(region, cp, schoolName, page, pageSize)
    if err != nil { c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()}); return }
    c.JSON(http.StatusOK, gin.H{"items": items, "total": total})
}

func (ctl *SettlementRatesController) UpsertCustomerRate(c *gin.Context) {
    type reqT struct{
        Region string   `json:"region" binding:"required"`
        CP     string   `json:"cp" binding:"required"`
        SchoolName *string  `json:"school_name"`
        CustomerFee *float64 `json:"customer_fee"`
        NetworkLineFee *float64 `json:"network_line_fee"`
        GeneralFee *float64 `json:"general_fee"`
        CustomerFeeOwnerID *uint64 `json:"customer_fee_owner_id"`
        NetworkLineFeeOwnerID *uint64 `json:"network_line_fee_owner_id"`
    }
    var req reqT
    if err := c.ShouldBindJSON(&req); err != nil { c.JSON(http.StatusBadRequest, gin.H{"message":"invalid request"}); return }
    rate := &model.RateCustomer{
        Region: req.Region,
        CP: req.CP,
        SchoolName: req.SchoolName,
        CustomerFee: req.CustomerFee,
        NetworkLineFee: req.NetworkLineFee,
        GeneralFee: req.GeneralFee,
        CustomerFeeOwnerID: req.CustomerFeeOwnerID,
        NetworkLineFeeOwnerID: req.NetworkLineFeeOwnerID,
    }
    if err := ctl.svc.UpsertCustomerRate(rate); err != nil {
        if service.IsBadRequest(err) { c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()}); return }
        c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()}); return
    }
    c.Status(http.StatusNoContent)
}

// Node business rates
func (ctl *SettlementRatesController) ListNodeRates(c *gin.Context) {
    page := parseIntDefault(c.Query("page"), 1)
    pageSize := parseIntDefault(c.Query("page_size"), 10)
    region := c.Query("region")
    cp := c.Query("cp")
    settlementType := c.Query("settlement_type")
    items, total, err := ctl.svc.ListNodeRates(region, cp, settlementType, page, pageSize)
    if err != nil { c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()}); return }
    c.JSON(http.StatusOK, gin.H{"items": items, "total": total})
}

func (ctl *SettlementRatesController) UpsertNodeRate(c *gin.Context) {
    type reqT struct{
        Region string `json:"region" binding:"required"`
        CP string `json:"cp" binding:"required"`
        SettlementType string `json:"settlement_type" binding:"required"`
        CPFee *float64 `json:"cp_fee"`
        CPFeeOwnerID *uint64 `json:"cp_fee_owner_id"`
        NodeConstructionFee *float64 `json:"node_construction_fee"`
        NodeConstructionFeeOwnerID *uint64 `json:"node_construction_fee_owner_id"`
        RackFee *float64 `json:"rack_fee"`
        RackFeeOwnerID *uint64 `json:"rack_fee_owner_id"`
        OtherFee *float64 `json:"other_fee"`
        OtherFeeOwnerID *uint64 `json:"other_fee_owner_id"`
    }
    var req reqT
    if err := c.ShouldBindJSON(&req); err != nil { c.JSON(http.StatusBadRequest, gin.H{"message":"invalid request"}); return }
    rate := &model.RateNode{
        Region: req.Region,
        CP: req.CP,
        SettlementType: req.SettlementType,
        CPFee: req.CPFee,
        CPFeeOwnerID: req.CPFeeOwnerID,
        NodeConstructionFee: req.NodeConstructionFee,
        NodeConstructionFeeOwnerID: req.NodeConstructionFeeOwnerID,
        RackFee: req.RackFee,
        RackFeeOwnerID: req.RackFeeOwnerID,
        OtherFee: req.OtherFee,
        OtherFeeOwnerID: req.OtherFeeOwnerID,
    }
    if err := ctl.svc.UpsertNodeRate(rate); err != nil {
        if service.IsBadRequest(err) { c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()}); return }
        c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()}); return
    }
    c.Status(http.StatusNoContent)
}

// Final customer rates
func (ctl *SettlementRatesController) ListFinalCustomerRates(c *gin.Context) {
    page := parseIntDefault(c.Query("page"), 1)
    pageSize := parseIntDefault(c.Query("page_size"), 10)
    region := c.Query("region")
    cp := c.Query("cp")
    schoolName := c.Query("school_name")
    feeType := c.Query("fee_type")
    items, total, err := ctl.svc.ListFinalCustomerRates(region, cp, schoolName, feeType, page, pageSize)
    if err != nil { c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()}); return }
    c.JSON(http.StatusOK, gin.H{"items": items, "total": total})
}

func (ctl *SettlementRatesController) UpsertFinalCustomerRate(c *gin.Context) {
    type reqT struct{
        Region string `json:"region" binding:"required"`
        CP string `json:"cp" binding:"required"`
        SchoolName string `json:"school_name" binding:"required"`
        FinalFee *float64 `json:"final_fee"`
        FeeType string `json:"fee_type" binding:"required"`
        CustomerFee *float64 `json:"customer_fee"`
        CustomerFeeOwnerID *uint64 `json:"customer_fee_owner_id"`
        NetworkLineFee *float64 `json:"network_line_fee"`
        NetworkLineFeeOwnerID *uint64 `json:"network_line_fee_owner_id"`
        NodeDeductionFee *float64 `json:"node_deduction_fee"`
        NodeDeductionFeeOwnerID *uint64 `json:"node_deduction_fee_owner_id"`
    }
    var req reqT
    if err := c.ShouldBindJSON(&req); err != nil { c.JSON(http.StatusBadRequest, gin.H{"message":"invalid request"}); return }
    rate := &model.RateFinalCustomer{
        Region: req.Region,
        CP: req.CP,
        SchoolName: req.SchoolName,
        FinalFee: req.FinalFee,
        FeeType: req.FeeType,
        CustomerFee: req.CustomerFee,
        CustomerFeeOwnerID: req.CustomerFeeOwnerID,
        NetworkLineFee: req.NetworkLineFee,
        NetworkLineFeeOwnerID: req.NetworkLineFeeOwnerID,
        NodeDeductionFee: req.NodeDeductionFee,
        NodeDeductionFeeOwnerID: req.NodeDeductionFeeOwnerID,
    }
    if err := ctl.svc.UpsertFinalCustomerRate(rate); err != nil {
        if service.IsBadRequest(err) { c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()}); return }
        c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()}); return
    }
    c.Status(http.StatusNoContent)
}

func (ctl *SettlementRatesController) RefreshFinalCustomerRates(c *gin.Context) {
    c.JSON(http.StatusNotImplemented, gin.H{"message": "Not Implemented"})
}
