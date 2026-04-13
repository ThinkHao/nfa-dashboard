package controller

import (
	"bytes"
	"crypto/rand"
	"encoding/csv"
	"encoding/hex"
	"io"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"nfa-dashboard/internal/model"
	"nfa-dashboard/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/xuri/excelize/v2"
)

type customerRateImportError struct {
	Line    int    `json:"line"`
	Message string `json:"message"`
}

type preparedCustomerRateImportRow struct {
	Line       int
	Rate       *model.RateCustomer
	OwnerNames service.CustomerRateOwnerNames
}

type customerRateImportSession struct {
	Filename     string
	Content      []byte
	ValidateOnly bool
	ExpiresAt    time.Time
}

var customerRateImportSessions = struct {
	mu    sync.Mutex
	items map[string]customerRateImportSession
}{
	items: make(map[string]customerRateImportSession),
}

func readCustomerRateImportRequest(c *gin.Context) (string, []byte, string, bool, bool, error) {
	validateOnly := parseBoolImportFlag(c, "validate_only")
	autoCreateMissingUsers := parseBoolImportFlag(c, "auto_create_missing_users")
	resumableToken := strings.TrimSpace(c.PostForm("resumable_token"))
	if resumableToken == "" {
		resumableToken = strings.TrimSpace(c.Query("resumable_token"))
	}
	if resumableToken != "" {
		session, ok := loadCustomerRateImportSession(resumableToken)
		if !ok {
			return "", nil, "", false, false, service.NewBadRequest("导入会话已失效，请重新上传文件")
		}
		return session.Filename, append([]byte(nil), session.Content...), resumableToken, validateOnly || session.ValidateOnly, autoCreateMissingUsers, nil
	}

	file, err := c.FormFile("file")
	if err != nil {
		return "", nil, "", false, false, err
	}
	f, err := file.Open()
	if err != nil {
		return "", nil, "", false, false, err
	}
	defer f.Close()
	content, err := io.ReadAll(f)
	if err != nil {
		return "", nil, "", false, false, err
	}
	return file.Filename, content, "", validateOnly, autoCreateMissingUsers, nil
}

func parseCustomerRateImportFile(filename string, content []byte) ([]string, [][]string, error) {
	nameLower := strings.ToLower(strings.TrimSpace(filename))
	reader := bytes.NewReader(content)
	if strings.HasSuffix(nameLower, ".xlsx") || strings.HasSuffix(nameLower, ".xls") {
		xl, err := excelize.OpenReader(reader)
		if err != nil {
			return nil, nil, service.NewBadRequest("read excel failed")
		}
		defer func() { _ = xl.Close() }()
		sheets := xl.GetSheetList()
		if len(sheets) == 0 {
			return nil, nil, service.NewBadRequest("excel has no sheets")
		}
		rows, err := xl.GetRows(sheets[0])
		if err != nil || len(rows) == 0 {
			return nil, nil, service.NewBadRequest("read excel rows failed or empty")
		}
		header := rows[0]
		data := [][]string{}
		if len(rows) > 1 {
			data = rows[1:]
		}
		return header, data, nil
	}

	cr := csv.NewReader(reader)
	cr.FieldsPerRecord = -1
	header, err := cr.Read()
	if err != nil {
		if err == io.EOF {
			return nil, nil, service.NewBadRequest("empty file")
		}
		return nil, nil, service.NewBadRequest("read header failed")
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
	return header, rows, nil
}

func prepareCustomerRateImportRows(header []string, rows [][]string) ([]preparedCustomerRateImportRow, []customerRateImportError) {
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
	parseRatio := func(s string) (*float64, bool) {
		if s == "" {
			return nil, true
		}
		vtxt := strings.TrimSpace(strings.TrimSuffix(s, "%"))
		v, err := strconv.ParseFloat(vtxt, 64)
		if err != nil {
			return nil, false
		}
		if strings.HasSuffix(strings.TrimSpace(s), "%") || v > 1 {
			v = v / 100
		}
		return &v, true
	}
	idx := map[string]int{}
	fieldLabel := map[string]string{
		"region":                      "区域",
		"cp":                          "CP",
		"school_name":                 "学校",
		"customer_fee":                "客户费率",
		"network_line_fee":            "线路费率",
		"general_fee":                 "节点通用费率",
		"channel_rate":                "渠道费率",
		"customer_fee_owner_name":     "客户费归属",
		"network_line_fee_owner_name": "线路费归属",
		"general_fee_owner_name":      "节点通用费归属",
		"channel_owner_name":          "渠道费归属",
		"customer_fee_owner_id":       "客户费归属用户ID",
		"network_line_fee_owner_id":   "线路费归属用户ID",
		"general_fee_owner_id":        "节点通用费归属用户ID",
		"channel_owner_user_id":       "渠道费归属用户ID",
		"start_at":                    "存量起算日期",
		"increment_start_at":          "增量起算日期",
		"stock_ratio":                 "存量占比",
		"increment_ratio":             "增量占比",
	}
	labelOf := func(key string) string {
		if v, ok := fieldLabel[key]; ok {
			return v
		}
		return key
	}
	normalizeKey := func(key string) string {
		k := strings.ToLower(strings.TrimSpace(key))
		switch k {
		case "区域", "地区":
			return "region"
		case "cp":
			return "cp"
		case "学校", "学校名称":
			return "school_name"
		case "客户费率", "客户费":
			return "customer_fee"
		case "线路费率", "线路费":
			return "network_line_fee"
		case "节点通用费率", "节点通用费":
			return "general_fee"
		case "渠道费率":
			return "channel_rate"
		case "客户费归属":
			return "customer_fee_owner_name"
		case "客户费归属id":
			return "customer_fee_owner_id"
		case "线路费归属":
			return "network_line_fee_owner_name"
		case "线路费归属id":
			return "network_line_fee_owner_id"
		case "节点通用费归属":
			return "general_fee_owner_name"
		case "节点通用费归属id":
			return "general_fee_owner_id"
		case "渠道费归属":
			return "channel_owner_name"
		case "渠道费归属id":
			return "channel_owner_user_id"
		case "存量起算日期", "起算日期":
			return "start_at"
		case "增量起算日期":
			return "increment_start_at"
		case "存量占比":
			return "stock_ratio"
		case "增量占比":
			return "increment_ratio"
		default:
			return k
		}
	}
	for i, h := range header {
		idx[normalizeKey(h)] = i
	}
	get := func(cols []string, key string) string {
		if p, ok := idx[key]; ok && p >= 0 && p < len(cols) {
			return strings.TrimSpace(cols[p])
		}
		return ""
	}

	prepared := make([]preparedCustomerRateImportRow, 0, len(rows))
	errors := make([]customerRateImportError, 0)
	lineNo := 1
	for _, rec := range rows {
		lineNo++
		if isBlankImportRecord(rec) {
			continue
		}
		region := get(rec, "region")
		cp := get(rec, "cp")
		if region == "" || cp == "" {
			errors = append(errors, customerRateImportError{Line: lineNo, Message: "区域 和 CP 为必填"})
			continue
		}

		schoolName := func() *string {
			s := get(rec, "school_name")
			if s == "" {
				return nil
			}
			return &s
		}()
		cfStr := normalizeImportNumericField(get(rec, "customer_fee"))
		nlfStr := normalizeImportNumericField(get(rec, "network_line_fee"))
		gfStr := normalizeImportNumericField(get(rec, "general_fee"))
		crStr := normalizeImportNumericField(get(rec, "channel_rate"))
		cfoName := get(rec, "customer_fee_owner_name")
		nfoName := get(rec, "network_line_fee_owner_name")
		gfoName := get(rec, "general_fee_owner_name")
		choName := get(rec, "channel_owner_name")
		cfoStr := get(rec, "customer_fee_owner_id")
		nfoStr := get(rec, "network_line_fee_owner_id")
		gfoStr := get(rec, "general_fee_owner_id")
		choStr := get(rec, "channel_owner_user_id")
		incrementStartAtStr := get(rec, "increment_start_at")
		stockRatioStr := get(rec, "stock_ratio")
		incrementRatioStr := get(rec, "increment_ratio")

		customerFee := parseF(cfStr)
		networkLineFee := parseF(nlfStr)
		generalFee := parseF(gfStr)
		channelRate := parseF(crStr)
		cOwner := parseU(cfoStr)
		nOwner := parseU(nfoStr)
		gOwner := parseU(gfoStr)
		chOwner := parseU(choStr)
		stockRatio, okStock := parseRatio(stockRatioStr)
		incrementRatio, okInc := parseRatio(incrementRatioStr)

		switch {
		case cfStr != "" && customerFee == nil:
			errors = append(errors, customerRateImportError{Line: lineNo, Message: labelOf("customer_fee") + " 格式错误"})
			continue
		case nlfStr != "" && networkLineFee == nil:
			errors = append(errors, customerRateImportError{Line: lineNo, Message: labelOf("network_line_fee") + " 格式错误"})
			continue
		case gfStr != "" && generalFee == nil:
			errors = append(errors, customerRateImportError{Line: lineNo, Message: labelOf("general_fee") + " 格式错误"})
			continue
		case crStr != "" && channelRate == nil:
			errors = append(errors, customerRateImportError{Line: lineNo, Message: labelOf("channel_rate") + " 格式错误"})
			continue
		case cfoName == "" && cfoStr != "" && cOwner == nil:
			errors = append(errors, customerRateImportError{Line: lineNo, Message: labelOf("customer_fee_owner_id") + " 格式错误"})
			continue
		case nfoName == "" && nfoStr != "" && nOwner == nil:
			errors = append(errors, customerRateImportError{Line: lineNo, Message: labelOf("network_line_fee_owner_id") + " 格式错误"})
			continue
		case gfoName == "" && gfoStr != "" && gOwner == nil:
			errors = append(errors, customerRateImportError{Line: lineNo, Message: labelOf("general_fee_owner_id") + " 格式错误"})
			continue
		case choName == "" && choStr != "" && chOwner == nil:
			errors = append(errors, customerRateImportError{Line: lineNo, Message: labelOf("channel_owner_user_id") + " 格式错误"})
			continue
		case stockRatioStr != "" && !okStock:
			errors = append(errors, customerRateImportError{Line: lineNo, Message: labelOf("stock_ratio") + " 格式错误"})
			continue
		case incrementRatioStr != "" && !okInc:
			errors = append(errors, customerRateImportError{Line: lineNo, Message: labelOf("increment_ratio") + " 格式错误"})
			continue
		}

		startAtPtr, err := parseDateField(get(rec, "start_at"))
		if err != nil {
			errors = append(errors, customerRateImportError{Line: lineNo, Message: labelOf("start_at") + " 格式错误，期望 YYYY-MM-DD"})
			continue
		}
		incrementStartAtPtr, err := parseDateField(strings.TrimSpace(incrementStartAtStr))
		if err != nil {
			errors = append(errors, customerRateImportError{Line: lineNo, Message: labelOf("increment_start_at") + " 格式错误，期望 YYYY-MM-DD"})
			continue
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
			IncrementStartAt:      incrementStartAtPtr,
			StockRatio:            stockRatio,
			IncrementRatio:        incrementRatio,
		}
		if customerFee != nil || networkLineFee != nil || generalFee != nil {
			rate.FeeMode = "configed"
		}
		prepared = append(prepared, preparedCustomerRateImportRow{
			Line: lineNo,
			Rate: rate,
			OwnerNames: service.CustomerRateOwnerNames{
				CustomerFeeOwnerName:    cfoName,
				NetworkLineFeeOwnerName: nfoName,
				GeneralFeeOwnerName:     gfoName,
				ChannelOwnerName:        choName,
			},
		})
	}
	return prepared, errors
}

func collectMissingCustomerRateImportUsers(svc service.RatesService, rows []preparedCustomerRateImportRow) ([]service.MissingImportUser, []customerRateImportError, error) {
	missingByAlias := make(map[string]*service.MissingImportUser)
	order := make([]string, 0)
	errors := make([]customerRateImportError, 0)
	for _, row := range rows {
		_, missing, err := svc.LookupCustomerRateOwnerIDsByDisplayName(row.OwnerNames)
		if err != nil {
			errors = append(errors, customerRateImportError{Line: row.Line, Message: err.Error()})
			continue
		}
		for _, item := range missing {
			entry, ok := missingByAlias[item.Alias]
			if !ok {
				entry = &service.MissingImportUser{Alias: item.Alias}
				missingByAlias[item.Alias] = entry
				order = append(order, item.Alias)
			}
			appendMissingField(entry, item.Field)
			appendMissingLine(entry, row.Line)
		}
	}
	if len(order) == 0 {
		return nil, errors, nil
	}
	preview, err := svc.PreviewCustomerRateImportUsers(order)
	if err != nil {
		return nil, errors, err
	}
	for _, item := range preview {
		if entry, ok := missingByAlias[item.Alias]; ok {
			entry.SuggestedUsername = item.SuggestedUsername
		}
	}
	items := make([]service.MissingImportUser, 0, len(order))
	for _, alias := range order {
		items = append(items, *missingByAlias[alias])
	}
	return items, errors, nil
}

func executeCustomerRateImportRows(svc service.RatesService, rows []preparedCustomerRateImportRow, validateOnly bool) (int, []customerRateImportError) {
	affected := 0
	errors := make([]customerRateImportError, 0)
	for _, row := range rows {
		rate := clonePreparedImportRate(row.Rate)
		if err := svc.ResolveCustomerRateOwnerIDsByDisplayName(rate, row.OwnerNames); err != nil {
			errors = append(errors, customerRateImportError{Line: row.Line, Message: err.Error()})
			continue
		}
		if err := svc.ValidateCustomerRate(rate); err != nil {
			errors = append(errors, customerRateImportError{Line: row.Line, Message: err.Error()})
			continue
		}
		if validateOnly {
			affected++
			continue
		}
		if err := svc.UpsertCustomerRate(rate); err != nil {
			errors = append(errors, customerRateImportError{Line: row.Line, Message: err.Error()})
			continue
		}
		affected++
	}
	return affected, errors
}

func saveCustomerRateImportSession(filename string, content []byte, validateOnly bool) (string, error) {
	tokenBytes := make([]byte, 16)
	if _, err := rand.Read(tokenBytes); err != nil {
		return "", err
	}
	token := hex.EncodeToString(tokenBytes)
	customerRateImportSessions.mu.Lock()
	defer customerRateImportSessions.mu.Unlock()
	customerRateImportSessions.items[token] = customerRateImportSession{
		Filename:     filename,
		Content:      append([]byte(nil), content...),
		ValidateOnly: validateOnly,
		ExpiresAt:    time.Now().Add(15 * time.Minute),
	}
	return token, nil
}

func loadCustomerRateImportSession(token string) (customerRateImportSession, bool) {
	customerRateImportSessions.mu.Lock()
	defer customerRateImportSessions.mu.Unlock()
	session, ok := customerRateImportSessions.items[token]
	if !ok {
		return customerRateImportSession{}, false
	}
	if time.Now().After(session.ExpiresAt) {
		delete(customerRateImportSessions.items, token)
		return customerRateImportSession{}, false
	}
	return session, true
}

func deleteCustomerRateImportSession(token string) {
	if token == "" {
		return
	}
	customerRateImportSessions.mu.Lock()
	defer customerRateImportSessions.mu.Unlock()
	delete(customerRateImportSessions.items, token)
}

func parseBoolImportFlag(c *gin.Context, key string) bool {
	q := strings.TrimSpace(c.Query(key))
	if q == "1" || strings.EqualFold(q, "true") {
		return true
	}
	p := strings.TrimSpace(c.PostForm(key))
	return p == "1" || strings.EqualFold(p, "true")
}

func appendMissingField(item *service.MissingImportUser, field string) {
	for _, existing := range item.Fields {
		if existing == field {
			return
		}
	}
	item.Fields = append(item.Fields, field)
	sort.Strings(item.Fields)
}

func appendMissingLine(item *service.MissingImportUser, line int) {
	for _, existing := range item.Lines {
		if existing == line {
			return
		}
	}
	item.Lines = append(item.Lines, line)
	sort.Ints(item.Lines)
}

func clonePreparedImportRate(in *model.RateCustomer) *model.RateCustomer {
	if in == nil {
		return nil
	}
	cp := *in
	return &cp
}
