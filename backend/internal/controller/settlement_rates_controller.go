package controller

import (
	"encoding/csv"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"nfa-dashboard/internal/model"
	"nfa-dashboard/internal/repository"
	"nfa-dashboard/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/xuri/excelize/v2"
	"gorm.io/datatypes"
)

// SettlementRatesController hosts endpoints under /api/v1/settlement/rates
type SettlementRatesController struct{ svc service.RatesService }

// ExportCustomerRatesXLSX 导出客户业务费率为原生 Excel (.xlsx)
func (ctl *SettlementRatesController) ExportCustomerRatesXLSX(c *gin.Context) {
	region := c.Query("region")
	cp := c.Query("cp")
	schoolName := c.Query("school_name")
	const pageSize = 100000
	items, _, err := ctl.svc.ListCustomerRates(region, cp, schoolName, nil, 1, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		return
	}

	f := excelize.NewFile()
	sheet := f.GetSheetName(0)
	// 可见列：将归属导出为“用户名”列；同时保留隐藏的 *_owner_id 列用于导入兼容
	header := []string{
		"region", "cp", "school_name",
		"customer_fee", "network_line_fee", "general_fee", "channel_rate",
		// 可见的姓名列
		"customer_fee_owner", "network_line_fee_owner", "general_fee_owner", "channel_owner_user",
		"start_at",
		// 隐藏的ID列（用于导入与公式映射）
		"customer_fee_owner_id", "network_line_fee_owner_id", "general_fee_owner_id", "channel_owner_user_id",
	}
	for i, h := range header {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		_ = f.SetCellValue(sheet, cell, h)
	}
	row := 2
	// 预加载系统用户，用于姓名显示与下拉来源
	userRepo := repository.NewUserRepository()
	// 拉取全部启用用户（分页一次取较大数量，通常足够；若超出仍可按 total 分批扩展）
	st := int8(1)
	users, _, uErr := userRepo.List("", &st, nil, 1, 100000)
	if uErr != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": uErr.Error()})
		return
	}
	// 构建 ID -> 显示名 映射与下拉数据
	idToName := map[uint64]string{}
	names := make([]string, 0, len(users))
	ids := make([]uint64, 0, len(users))
	for _, u := range users {
		dn := strings.TrimSpace(u.Username)
		if u.Alias != nil && strings.TrimSpace(*u.Alias) != "" {
			dn = strings.TrimSpace(*u.Alias)
		}
		idToName[u.ID] = dn
		names = append(names, dn)
		ids = append(ids, u.ID)
	}
	// 创建隐藏工作表 Users 作为下拉与 VLOOKUP 数据源：A: display_name, B: id
	if idx, err2 := f.NewSheet("Users"); err2 == nil && idx > 0 {
		for i := 0; i < len(names); i++ {
			cellA, _ := excelize.CoordinatesToCellName(1, i+1) // A1...
			cellB, _ := excelize.CoordinatesToCellName(2, i+1) // B1...
			_ = f.SetCellValue("Users", cellA, names[i])
			_ = f.SetCellValue("Users", cellB, strconv.FormatUint(ids[i], 10))
		}
		// 隐藏 Users 工作表
		_ = f.SetSheetVisible("Users", false)
	}
	// 计算 Users!$A$1:$B$N 与名单区域（去表头，因我们从A1开始，无表头，直接全量）
	usersLastRow := len(names)
	hasUsers := usersLastRow > 0
	usersNameRange := ""
	usersLookupRange := ""
	if hasUsers {
		usersNameRange = "Users!$A$1:$A$" + strconv.Itoa(usersLastRow)
		usersLookupRange = "Users!$A$1:$B$" + strconv.Itoa(usersLastRow)
	}
	// 主要工作表中四个姓名列与四个ID列的列索引（基于 header 顺序）
	// 姓名列: 8..11 ；start_at: 12 ；ID列: 13..16
	nameCols := []int{8, 9, 10, 11}
	idCols := []int{13, 14, 15, 16}
	for _, it := range items {
		// 基本列（到 channel_rate）
		baseVals := []interface{}{
			strings.TrimSpace(it.Region),
			strings.TrimSpace(it.CP),
			func() string {
				if it.SchoolName == nil {
					return ""
				} else {
					return strings.TrimSpace(*it.SchoolName)
				}
			}(),
			func() interface{} {
				if it.CustomerFee == nil {
					return ""
				} else {
					return strconv.FormatFloat(*it.CustomerFee, 'f', 2, 64)
				}
			}(),
			func() interface{} {
				if it.NetworkLineFee == nil {
					return ""
				} else {
					return strconv.FormatFloat(*it.NetworkLineFee, 'f', 2, 64)
				}
			}(),
			func() interface{} {
				if it.GeneralFee == nil {
					return ""
				} else {
					return strconv.FormatFloat(*it.GeneralFee, 'f', 2, 64)
				}
			}(),
			func() interface{} {
				if it.ChannelRate == nil {
					return ""
				} else {
					return strconv.FormatFloat(*it.ChannelRate, 'f', 4, 64)
				}
			}(),
		}
		for i, v := range baseVals {
			cell, _ := excelize.CoordinatesToCellName(i+1, row)
			_ = f.SetCellValue(sheet, cell, v)
		}
		// 姓名列：由 ID->Name 映射（若无则空）
		var cfoName, nfoName, gfoName, choName string
		if it.CustomerFeeOwnerID != nil {
			if nm, ok := idToName[*it.CustomerFeeOwnerID]; ok {
				cfoName = nm
			}
		}
		if it.NetworkLineFeeOwnerID != nil {
			if nm, ok := idToName[*it.NetworkLineFeeOwnerID]; ok {
				nfoName = nm
			}
		}
		if it.GeneralFeeOwnerID != nil {
			if nm, ok := idToName[*it.GeneralFeeOwnerID]; ok {
				gfoName = nm
			}
		}
		if it.ChannelOwnerUserID != nil {
			if nm, ok := idToName[*it.ChannelOwnerUserID]; ok {
				choName = nm
			}
		}
		nameVals := []string{cfoName, nfoName, gfoName, choName}
		for j, v := range nameVals {
			col := nameCols[j]
			cell, _ := excelize.CoordinatesToCellName(col, row)
			_ = f.SetCellValue(sheet, cell, v)
		}
		// start_at
		saCell, _ := excelize.CoordinatesToCellName(12, row)
		if it.StartAt == nil {
			_ = f.SetCellValue(sheet, saCell, "")
		} else {
			_ = f.SetCellValue(sheet, saCell, it.StartAt.Format("2006-01-02"))
		}
		// ID列：若有用户清单，使用 VLOOKUP 公式；否则直接写入原始ID（若存在）
		if hasUsers {
			for j := 0; j < len(idCols); j++ {
				idCol := idCols[j]
				nameCol := nameCols[j]
				idCell, _ := excelize.CoordinatesToCellName(idCol, row)
				nameCell, _ := excelize.CoordinatesToCellName(nameCol, row)
				formula := "IFERROR(VLOOKUP(" + nameCell + "," + usersLookupRange + ",2,FALSE),\"\")"
				_ = f.SetCellFormula(sheet, idCell, formula)
			}
		} else {
			// 无用户清单时，直接将当前记录中的 *_id 写入隐藏列
			rawIDs := []interface{}{
				func() interface{} {
					if it.CustomerFeeOwnerID == nil {
						return ""
					} else {
						return strconv.FormatUint(*it.CustomerFeeOwnerID, 10)
					}
				}(),
				func() interface{} {
					if it.NetworkLineFeeOwnerID == nil {
						return ""
					} else {
						return strconv.FormatUint(*it.NetworkLineFeeOwnerID, 10)
					}
				}(),
				func() interface{} {
					if it.GeneralFeeOwnerID == nil {
						return ""
					} else {
						return strconv.FormatUint(*it.GeneralFeeOwnerID, 10)
					}
				}(),
				func() interface{} {
					if it.ChannelOwnerUserID == nil {
						return ""
					} else {
						return strconv.FormatUint(*it.ChannelOwnerUserID, 10)
					}
				}(),
			}
			for j, v := range rawIDs {
				col := idCols[j]
				cell, _ := excelize.CoordinatesToCellName(col, row)
				_ = f.SetCellValue(sheet, cell, v)
			}
		}
		row++
	}
	// 为姓名列添加下拉数据验证（需有用户清单且表内有数据）
	lastRow := row - 1
	if hasUsers && lastRow >= 2 {
		for _, col := range nameCols {
			startCell, _ := excelize.CoordinatesToCellName(col, 2)
			endCell, _ := excelize.CoordinatesToCellName(col, lastRow)
			rng := startCell + ":" + endCell
			dv := &excelize.DataValidation{Type: "list", AllowBlank: true, Sqref: rng, Formula1: usersNameRange}
			_ = f.AddDataValidation(sheet, dv)
		}
	}
	// 隐藏 *_owner_id 四个列
	_ = f.SetColVisible(sheet, "M", false) // 13
	_ = f.SetColVisible(sheet, "N", false) // 14
	_ = f.SetColVisible(sheet, "O", false) // 15
	_ = f.SetColVisible(sheet, "P", false) // 16
	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Header("Content-Disposition", "attachment; filename=customer_rates.xlsx")
	if err := f.Write(c.Writer); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		return
	}
}

// ExportCustomerRates 导出客户业务费率为 CSV（Excel 可直接打开）
func (ctl *SettlementRatesController) ExportCustomerRates(c *gin.Context) {
	region := c.Query("region")
	cp := c.Query("cp")
	schoolName := c.Query("school_name")
	// 不使用 settlement_ready 过滤，保持与列表一致的基础筛选

	// 简单起见，单次拉取较大页
	const pageSize = 100000
	items, _, err := ctl.svc.ListCustomerRates(region, cp, schoolName, nil, 1, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		return
	}

	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Header("Content-Disposition", "attachment; filename=customer_rates.csv")
	w := csv.NewWriter(c.Writer)
	// 头
	_ = w.Write([]string{
		"region", "cp", "school_name",
		"customer_fee", "network_line_fee", "general_fee", "channel_rate",
		"customer_fee_owner_id", "network_line_fee_owner_id", "general_fee_owner_id", "channel_owner_user_id",
		"start_at",
	})
	toStrF := func(p *float64, prec int) string {
		if p == nil {
			return ""
		}
		return strconv.FormatFloat(*p, 'f', prec, 64)
	}
	// 行
	for _, it := range items {
		var startAt string
		if it.StartAt != nil {
			startAt = it.StartAt.Format("2006-01-02")
		}
		_ = w.Write([]string{
			strings.TrimSpace(it.Region), strings.TrimSpace(it.CP), func() string {
				if it.SchoolName == nil {
					return ""
				} else {
					return strings.TrimSpace(*it.SchoolName)
				}
			}(),
			toStrF(it.CustomerFee, 2), toStrF(it.NetworkLineFee, 2), toStrF(it.GeneralFee, 2), toStrF(it.ChannelRate, 4),
			func() string {
				if it.CustomerFeeOwnerID == nil {
					return ""
				} else {
					return strconv.FormatUint(*it.CustomerFeeOwnerID, 10)
				}
			}(),
			func() string {
				if it.NetworkLineFeeOwnerID == nil {
					return ""
				} else {
					return strconv.FormatUint(*it.NetworkLineFeeOwnerID, 10)
				}
			}(),
			func() string {
				if it.GeneralFeeOwnerID == nil {
					return ""
				} else {
					return strconv.FormatUint(*it.GeneralFeeOwnerID, 10)
				}
			}(),
			func() string {
				if it.ChannelOwnerUserID == nil {
					return ""
				} else {
					return strconv.FormatUint(*it.ChannelOwnerUserID, 10)
				}
			}(),
			startAt,
		})
	}
	w.Flush()
	if err := w.Error(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		return
	}
}

// ImportCustomerRates 从 CSV 导入客户业务费率
// 要求首行包含表头：region,cp,school_name,customer_fee,network_line_fee,general_fee,channel_rate,customer_fee_owner_id,network_line_fee_owner_id,general_fee_owner_id,channel_owner_user_id,start_at
func (ctl *SettlementRatesController) ImportCustomerRates(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "file is required"})
		return
	}
	f, err := file.Open()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "open file failed"})
		return
	}
	defer f.Close()

	parseF := func(s string) *float64 {
		if s == "" {
			return nil
		}
		if v, err := strconv.ParseFloat(s, 64); err == nil {
			return &v
		}
		return nil
	}
	parseU := func(s string) *uint64 {
		if s == "" {
			return nil
		}
		if v, err := strconv.ParseUint(s, 10, 64); err == nil {
			return &v
		}
		return nil
	}
	var affected int
	errs := make([]map[string]interface{}, 0, 16)
	validateOnly := func() bool {
		q := strings.TrimSpace(c.Query("validate_only"))
		if q == "1" || strings.EqualFold(q, "true") {
			return true
		}
		p := strings.TrimSpace(c.PostForm("validate_only"))
		if p == "1" || strings.EqualFold(p, "true") {
			return true
		}
		return false
	}()

	process := func(header []string, rows [][]string) {
		idx := map[string]int{}
		for i, h := range header {
			idx[strings.ToLower(strings.TrimSpace(h))] = i
		}
		get := func(cols []string, key string) string {
			if p, ok := idx[key]; ok && p >= 0 && p < len(cols) {
				return strings.TrimSpace(cols[p])
			}
			return ""
		}
		lineNo := 1
		for _, rec := range rows {
			lineNo++
			region := get(rec, "region")
			cp := get(rec, "cp")
			if region == "" || cp == "" {
				errs = append(errs, map[string]interface{}{"line": lineNo, "message": "region and cp are required"})
				continue
			}
			schoolName := func() *string {
				s := get(rec, "school_name")
				if s == "" {
					return nil
				}
				return &s
			}()
			cfStr := get(rec, "customer_fee")
			nlfStr := get(rec, "network_line_fee")
			gfStr := get(rec, "general_fee")
			crStr := get(rec, "channel_rate")
			cfoStr := get(rec, "customer_fee_owner_id")
			nfoStr := get(rec, "network_line_fee_owner_id")
			gfoStr := get(rec, "general_fee_owner_id")
			choStr := get(rec, "channel_owner_user_id")

			customerFee := parseF(cfStr)
			networkLineFee := parseF(nlfStr)
			generalFee := parseF(gfStr)
			channelRate := parseF(crStr)
			cOwner := parseU(cfoStr)
			nOwner := parseU(nfoStr)
			gOwner := parseU(gfoStr)
			chOwner := parseU(choStr)

			if cfStr != "" && customerFee == nil {
				errs = append(errs, map[string]interface{}{"line": lineNo, "message": "invalid customer_fee"})
				continue
			}
			if nlfStr != "" && networkLineFee == nil {
				errs = append(errs, map[string]interface{}{"line": lineNo, "message": "invalid network_line_fee"})
				continue
			}
			if gfStr != "" && generalFee == nil {
				errs = append(errs, map[string]interface{}{"line": lineNo, "message": "invalid general_fee"})
				continue
			}
			if crStr != "" && channelRate == nil {
				errs = append(errs, map[string]interface{}{"line": lineNo, "message": "invalid channel_rate"})
				continue
			}
			if cfoStr != "" && cOwner == nil {
				errs = append(errs, map[string]interface{}{"line": lineNo, "message": "invalid customer_fee_owner_id"})
				continue
			}
			if nfoStr != "" && nOwner == nil {
				errs = append(errs, map[string]interface{}{"line": lineNo, "message": "invalid network_line_fee_owner_id"})
				continue
			}
			if gfoStr != "" && gOwner == nil {
				errs = append(errs, map[string]interface{}{"line": lineNo, "message": "invalid general_fee_owner_id"})
				continue
			}
			if choStr != "" && chOwner == nil {
				errs = append(errs, map[string]interface{}{"line": lineNo, "message": "invalid channel_owner_user_id"})
				continue
			}

			var startAtPtr *time.Time
			if s := get(rec, "start_at"); s != "" {
				if t, err := time.Parse("2006-01-02", s); err == nil {
					startAtPtr = &t
				} else {
					errs = append(errs, map[string]interface{}{"line": lineNo, "message": "invalid start_at, expect YYYY-MM-DD"})
					continue
				}
			}

			rate := &model.RateCustomer{
				Region:                region,
				CP:                    cp,
				SchoolName:            schoolName,
				CustomerFee:           customerFee,
				NetworkLineFee:        networkLineFee,
				GeneralFee:            generalFee,
				ChannelRate:           channelRate,
				CustomerFeeOwnerID:    cOwner,
				NetworkLineFeeOwnerID: nOwner,
				GeneralFeeOwnerID:     gOwner,
				ChannelOwnerUserID:    chOwner,
				StartAt:               startAtPtr,
			}
			if customerFee != nil || networkLineFee != nil || generalFee != nil {
				rate.FeeMode = "configed"
			}
			if validateOnly {
				affected++
			} else {
				if err := ctl.svc.UpsertCustomerRate(rate); err == nil {
					affected++
				} else {
					errs = append(errs, map[string]interface{}{"line": lineNo, "message": err.Error()})
				}
			}
		}
	}

	nameLower := strings.ToLower(strings.TrimSpace(file.Filename))
	if strings.HasSuffix(nameLower, ".xlsx") || strings.HasSuffix(nameLower, ".xls") {
		xl, err := excelize.OpenReader(f)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"message": "read excel failed"})
			return
		}
		defer func() { _ = xl.Close() }()
		sheets := xl.GetSheetList()
		if len(sheets) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"message": "excel has no sheets"})
			return
		}
		rows, err := xl.GetRows(sheets[0])
		if err != nil || len(rows) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"message": "read excel rows failed or empty"})
			return
		}
		header := rows[0]
		data := [][]string{}
		if len(rows) > 1 {
			data = rows[1:]
		}
		process(header, data)
	} else {
		cr := csv.NewReader(f)
		cr.FieldsPerRecord = -1
		header, err := cr.Read()
		if err != nil {
			if err == io.EOF {
				c.JSON(http.StatusBadRequest, gin.H{"message": "empty file"})
				return
			}
			c.JSON(http.StatusBadRequest, gin.H{"message": "read header failed"})
			return
		}
		rows := make([][]string, 0, 1024)
		for {
			rec, err := cr.Read()
			if err == io.EOF {
				break
			}
			if err != nil {
				rows = append(rows, []string{})
				continue
			}
			rows = append(rows, rec)
		}
		process(header, rows)
	}

	c.JSON(http.StatusOK, gin.H{"affected": affected, "errors": errs, "validate_only": validateOnly})
}

func (ctl *SettlementRatesController) CustomerRatesImportTemplate(c *gin.Context) {
	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Header("Content-Disposition", "attachment; filename=customer_rates_template.csv")
	w := csv.NewWriter(c.Writer)
	_ = w.Write([]string{
		"region", "cp", "school_name",
		"customer_fee", "network_line_fee", "general_fee", "channel_rate",
		"customer_fee_owner_id", "network_line_fee_owner_id", "general_fee_owner_id", "channel_owner_user_id",
		"start_at",
	})
	_ = w.Write([]string{"华东", "CMCC", "示例学校", "100", "50", "10", "0.0200", "1001", "1002", "1003", "1004", "2025-01-01"})
	w.Flush()
}

func NewSettlementRatesController(svc service.RatesService) *SettlementRatesController {
	return &SettlementRatesController{svc: svc}
}

// Customer business rates
func (ctl *SettlementRatesController) ListCustomerRates(c *gin.Context) {
	page := parseIntDefault(c.Query("page"), 1)
	pageSize := parseIntDefault(c.Query("page_size"), 10)
	region := c.Query("region")
	cp := c.Query("cp")
	schoolName := c.Query("school_name")
	// 可选：参与结算筛选
	var settlementReady *bool
	if v := strings.TrimSpace(c.Query("settlement_ready")); v != "" {
		if v == "1" || strings.EqualFold(v, "true") {
			b := true
			settlementReady = &b
		} else if v == "0" || strings.EqualFold(v, "false") {
			b := false
			settlementReady = &b
		}
	}
	items, total, err := ctl.svc.ListCustomerRates(region, cp, schoolName, settlementReady, page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		return
	}
	type customerResp struct {
		model.RateCustomer `json:",inline"`
		SettlementReady    bool     `json:"settlement_ready"`
		MissingFields      []string `json:"missing_fields,omitempty"`
	}
	resp := make([]customerResp, 0, len(items))
	for _, it := range items {
		missing := make([]string, 0, 3)
		name := ""
		if it.SchoolName != nil {
			name = strings.TrimSpace(*it.SchoolName)
		}
		if name == "" {
			missing = append(missing, "school_name")
		}
		if it.CustomerFee == nil {
			missing = append(missing, "customer_fee")
		}
		if it.NetworkLineFee == nil {
			missing = append(missing, "network_line_fee")
		}
		if it.GeneralFee == nil {
			missing = append(missing, "general_fee")
		}
		ready := len(missing) == 0
		resp = append(resp, customerResp{RateCustomer: it, SettlementReady: ready, MissingFields: missing})
	}
	c.JSON(http.StatusOK, gin.H{"items": resp, "total": total})
}

func (ctl *SettlementRatesController) UpsertCustomerRate(c *gin.Context) {
	type reqT struct {
		Region                string          `json:"region" binding:"required"`
		CP                    string          `json:"cp" binding:"required"`
		SchoolName            *string         `json:"school_name"`
		CustomerFee           *float64        `json:"customer_fee"`
		NetworkLineFee        *float64        `json:"network_line_fee"`
		GeneralFee            *float64        `json:"general_fee"`
		CustomerFeeOwnerID    *uint64         `json:"customer_fee_owner_id"`
		NetworkLineFeeOwnerID *uint64         `json:"network_line_fee_owner_id"`
		GeneralFeeOwnerID     *uint64         `json:"general_fee_owner_id"`
		ChannelRate           *float64        `json:"channel_rate"`
		ChannelOwnerUserID    *uint64         `json:"channel_owner_user_id"`
		StartAt               string          `json:"start_at"`
		Extra                 json.RawMessage `json:"extra"`
	}
	var req reqT
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid request"})
		return
	}
	var startAtPtr *time.Time
	if s := strings.TrimSpace(req.StartAt); s != "" {
		if t, err := time.Parse("2006-01-02", s); err == nil {
			startAtPtr = &t
		}
	}

	rate := &model.RateCustomer{
		Region:                req.Region,
		CP:                    req.CP,
		SchoolName:            req.SchoolName,
		CustomerFee:           req.CustomerFee,
		NetworkLineFee:        req.NetworkLineFee,
		GeneralFee:            req.GeneralFee,
		CustomerFeeOwnerID:    req.CustomerFeeOwnerID,
		NetworkLineFeeOwnerID: req.NetworkLineFeeOwnerID,
		GeneralFeeOwnerID:     req.GeneralFeeOwnerID,
		ChannelRate:           req.ChannelRate,
		ChannelOwnerUserID:    req.ChannelOwnerUserID,
		StartAt:               startAtPtr,
	}
	if len(req.Extra) > 0 {
		rate.Extra = datatypes.JSON(req.Extra)
	}
	if req.CustomerFee != nil || req.NetworkLineFee != nil || req.GeneralFee != nil {
		rate.FeeMode = "configed"
	}
	if err := ctl.svc.UpsertCustomerRate(rate); err != nil {
		if service.IsBadRequest(err) {
			c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		return
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
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": items, "total": total})
}

// ListFinalCustomerRatesDiscounted 按服务日期返回折损后的最终客户费率视图
func (ctl *SettlementRatesController) ListFinalCustomerRatesDiscounted(c *gin.Context) {
	page := parseIntDefault(c.Query("page"), 1)
	pageSize := parseIntDefault(c.Query("page_size"), 10)
	region := c.Query("region")
	cp := c.Query("cp")
	schoolName := c.Query("school_name")
	feeType := c.Query("fee_type")
	sd := strings.TrimSpace(c.Query("service_date"))
	if sd == "" {
		c.JSON(http.StatusBadRequest, gin.H{"message": "service_date is required"})
		return
	}
	// 仅按日期解析
	serviceDate, err := time.Parse("2006-01-02", sd)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid service_date, expect YYYY-MM-DD"})
		return
	}
	items, total, err := ctl.svc.ListFinalCustomerRatesDiscounted(region, cp, schoolName, feeType, serviceDate, page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": items, "total": total})
}

func (ctl *SettlementRatesController) UpsertNodeRate(c *gin.Context) {
	type reqT struct {
		Region                     string   `json:"region" binding:"required"`
		CP                         string   `json:"cp" binding:"required"`
		SettlementType             string   `json:"settlement_type" binding:"required"`
		CPFee                      *float64 `json:"cp_fee"`
		CPFeeOwnerID               *uint64  `json:"cp_fee_owner_id"`
		NodeConstructionFee        *float64 `json:"node_construction_fee"`
		NodeConstructionFeeOwnerID *uint64  `json:"node_construction_fee_owner_id"`
		RackFee                    *float64 `json:"rack_fee"`
		RackFeeOwnerID             *uint64  `json:"rack_fee_owner_id"`
		OtherFee                   *float64 `json:"other_fee"`
		OtherFeeOwnerID            *uint64  `json:"other_fee_owner_id"`
	}
	var req reqT
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid request"})
		return
	}
	rate := &model.RateNode{
		Region:                     req.Region,
		CP:                         req.CP,
		SettlementType:             req.SettlementType,
		CPFee:                      req.CPFee,
		CPFeeOwnerID:               req.CPFeeOwnerID,
		NodeConstructionFee:        req.NodeConstructionFee,
		NodeConstructionFeeOwnerID: req.NodeConstructionFeeOwnerID,
		RackFee:                    req.RackFee,
		RackFeeOwnerID:             req.RackFeeOwnerID,
		OtherFee:                   req.OtherFee,
		OtherFeeOwnerID:            req.OtherFeeOwnerID,
	}
	if err := ctl.svc.UpsertNodeRate(rate); err != nil {
		if service.IsBadRequest(err) {
			c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		return
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
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": items, "total": total})
}

func (ctl *SettlementRatesController) UpsertFinalCustomerRate(c *gin.Context) {
	type reqT struct {
		Region                  string   `json:"region" binding:"required"`
		CP                      string   `json:"cp" binding:"required"`
		SchoolName              string   `json:"school_name" binding:"required"`
		FinalFee                *float64 `json:"final_fee"`
		FeeType                 string   `json:"fee_type" binding:"required"`
		CustomerFee             *float64 `json:"customer_fee"`
		CustomerFeeOwnerID      *uint64  `json:"customer_fee_owner_id"`
		NetworkLineFee          *float64 `json:"network_line_fee"`
		NetworkLineFeeOwnerID   *uint64  `json:"network_line_fee_owner_id"`
		NodeDeductionFee        *float64 `json:"node_deduction_fee"`
		NodeDeductionFeeOwnerID *uint64  `json:"node_deduction_fee_owner_id"`
	}
	var req reqT
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid request"})
		return
	}
	rate := &model.RateFinalCustomer{
		Region:                  req.Region,
		CP:                      req.CP,
		SchoolName:              req.SchoolName,
		FinalFee:                req.FinalFee,
		FeeType:                 req.FeeType,
		CustomerFee:             req.CustomerFee,
		CustomerFeeOwnerID:      req.CustomerFeeOwnerID,
		NetworkLineFee:          req.NetworkLineFee,
		NetworkLineFeeOwnerID:   req.NetworkLineFeeOwnerID,
		NodeDeductionFee:        req.NodeDeductionFee,
		NodeDeductionFeeOwnerID: req.NodeDeductionFeeOwnerID,
	}
	if err := ctl.svc.UpsertFinalCustomerRate(rate); err != nil {
		if service.IsBadRequest(err) {
			c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}

// 初始化最终客户费率：从 rate_customer 同步（仅插入缺失或覆盖 auto，保护 config）
func (ctl *SettlementRatesController) InitFinalCustomerRatesFromCustomer(c *gin.Context) {
	affected, err := ctl.svc.InitFinalCustomerRatesFromCustomer()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"affected": affected})
}

// 刷新最终客户费率：仅针对 auto 计算 final_fee
func (ctl *SettlementRatesController) RefreshFinalCustomerRates(c *gin.Context) {
	affected, err := ctl.svc.RefreshFinalCustomerRates()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"affected": affected})
}

// 清理无效最终客户费率：删除 fee_type='auto' 且任一关键费率字段为空的记录
func (ctl *SettlementRatesController) CleanupInvalidFinalCustomerRates(c *gin.Context) {
	affected, err := ctl.svc.CleanupInvalidFinalCustomerRates()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"affected": affected})
}
