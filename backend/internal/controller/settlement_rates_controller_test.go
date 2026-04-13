package controller

import (
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"nfa-dashboard/internal/model"
	"nfa-dashboard/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/xuri/excelize/v2"
)

type settlementRatesServiceStub struct {
	resolvedNames       service.CustomerRateOwnerNames
	resolvedRate        *model.RateCustomer
	resolveErr          error
	validateErr         error
	upsertErr           error
	upsertedRates       []*model.RateCustomer
	validateOnlyCalled  int
	lookupMissingByName map[string]bool
	previewUsers        []service.MissingImportUser
	createdUsers        []service.CreatedImportUser
	previewAliases      []string
	createMissingUsers  []service.MissingImportUser
}

func (s *settlementRatesServiceStub) ListCustomerRates(region, cp, schoolName string, settlementReady *bool, page, pageSize int) ([]model.RateCustomer, int64, error) {
	return nil, 0, nil
}

func (s *settlementRatesServiceStub) UpsertCustomerRate(rate *model.RateCustomer) error {
	s.upsertedRates = append(s.upsertedRates, cloneRateCustomer(rate))
	return s.upsertErr
}

func (s *settlementRatesServiceStub) ValidateCustomerRate(rate *model.RateCustomer) error {
	s.validateOnlyCalled++
	return s.validateErr
}

func (s *settlementRatesServiceStub) ResolveCustomerRateOwnerIDsByDisplayName(rate *model.RateCustomer, names service.CustomerRateOwnerNames) error {
	s.resolvedNames = names
	s.resolvedRate = cloneRateCustomer(rate)
	ids, missing, err := s.LookupCustomerRateOwnerIDsByDisplayName(names)
	if err != nil {
		return err
	}
	if len(missing) > 0 {
		return service.NewBadRequest(missing[0].Field + " 未匹配到系统用户：" + missing[0].Alias)
	}
	if ids.CustomerFeeOwnerID != nil {
		rate.CustomerFeeOwnerID = ids.CustomerFeeOwnerID
	}
	return nil
}

func (s *settlementRatesServiceStub) LookupCustomerRateOwnerIDsByDisplayName(names service.CustomerRateOwnerNames) (service.CustomerRateOwnerIDs, []service.MissingCustomerRateOwner, error) {
	if s.resolveErr != nil {
		return service.CustomerRateOwnerIDs{}, nil, s.resolveErr
	}
	ids := service.CustomerRateOwnerIDs{}
	missing := make([]service.MissingCustomerRateOwner, 0)
	appendMissing := func(field, alias string) {
		if alias == "" {
			return
		}
		if s.lookupMissingByName != nil && s.lookupMissingByName[alias] {
			missing = append(missing, service.MissingCustomerRateOwner{Field: field, Alias: alias})
			return
		}
		id := uint64(42)
		switch field {
		case "客户费归属":
			ids.CustomerFeeOwnerID = &id
		case "线路费归属":
			ids.NetworkLineFeeOwnerID = &id
		case "节点通用费归属":
			ids.GeneralFeeOwnerID = &id
		case "渠道费归属":
			ids.ChannelOwnerUserID = &id
		}
	}
	appendMissing("客户费归属", names.CustomerFeeOwnerName)
	appendMissing("线路费归属", names.NetworkLineFeeOwnerName)
	appendMissing("节点通用费归属", names.GeneralFeeOwnerName)
	appendMissing("渠道费归属", names.ChannelOwnerName)
	return ids, missing, nil
}

func (s *settlementRatesServiceStub) PreviewCustomerRateImportUsers(aliases []string) ([]service.MissingImportUser, error) {
	s.previewAliases = append([]string(nil), aliases...)
	if len(s.previewUsers) > 0 {
		return append([]service.MissingImportUser(nil), s.previewUsers...), nil
	}
	items := make([]service.MissingImportUser, 0, len(aliases))
	for _, alias := range aliases {
		items = append(items, service.MissingImportUser{Alias: alias, SuggestedUsername: "user1"})
	}
	return items, nil
}

func (s *settlementRatesServiceStub) CreateCustomerRateImportUsers(missing []service.MissingImportUser) ([]service.CreatedImportUser, error) {
	s.createMissingUsers = append([]service.MissingImportUser(nil), missing...)
	for _, item := range missing {
		delete(s.lookupMissingByName, item.Alias)
	}
	return append([]service.CreatedImportUser(nil), s.createdUsers...), nil
}

func (s *settlementRatesServiceStub) ListNodeRates(region, cp, settlementType string, page, pageSize int) ([]model.RateNode, int64, error) {
	return nil, 0, nil
}

func (s *settlementRatesServiceStub) UpsertNodeRate(rate *model.RateNode) error { return nil }

func (s *settlementRatesServiceStub) ListFinalCustomerRates(region, cp, schoolName, feeType string, page, pageSize int) ([]model.RateFinalCustomer, int64, error) {
	return nil, 0, nil
}

func (s *settlementRatesServiceStub) UpsertFinalCustomerRate(rate *model.RateFinalCustomer) error {
	return nil
}

func (s *settlementRatesServiceStub) ListFinalCustomerRatesDiscounted(region, cp, schoolName, feeType string, serviceDate time.Time, page, pageSize int) ([]service.DiscountedFinalCustomerRate, int64, error) {
	return nil, 0, nil
}

func (s *settlementRatesServiceStub) InitFinalCustomerRatesFromCustomer() (int64, error) {
	return 0, nil
}

func (s *settlementRatesServiceStub) RefreshFinalCustomerRates() (int64, error) {
	return 0, nil
}

func (s *settlementRatesServiceStub) CleanupInvalidFinalCustomerRates() (int64, error) {
	return 0, nil
}

func TestImportCustomerRates_XLSXSupportsFormattedNumbersDatesAndOwnerNames(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &settlementRatesServiceStub{}
	ctl := NewSettlementRatesController(svc)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", "customer_rates.xlsx")
	if err != nil {
		t.Fatalf("create form file: %v", err)
	}
	if err := writeCustomerRatesWorkbook(part); err != nil {
		t.Fatalf("write workbook: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close writer: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/settlement/rates/customer/import", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	rec := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(rec)
	ctx.Request = req

	ctl.ImportCustomerRates(ctx)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	var resp struct {
		Affected int `json:"affected"`
		Errors   []struct {
			Line    int    `json:"line"`
			Message string `json:"message"`
		} `json:"errors"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp.Affected != 1 {
		t.Fatalf("expected affected=1, got %d body=%s", resp.Affected, rec.Body.String())
	}
	if len(resp.Errors) != 0 {
		t.Fatalf("expected no errors, got %+v", resp.Errors)
	}
	if svc.resolvedNames.CustomerFeeOwnerName != "Alice" {
		t.Fatalf("expected visible owner name to be used, got %+v", svc.resolvedNames)
	}
	if len(svc.upsertedRates) != 1 {
		t.Fatalf("expected one upserted rate, got %d", len(svc.upsertedRates))
	}
	rate := svc.upsertedRates[0]
	assertFloatPtrEquals(t, rate.CustomerFee, 1200)
	assertFloatPtrEquals(t, rate.NetworkLineFee, 1300)
	assertFloatPtrEquals(t, rate.GeneralFee, 2500)
	assertFloatPtrEquals(t, rate.ChannelRate, 500)
	if rate.StartAt == nil || rate.StartAt.Format("2006-01-02") != "2025-07-01" {
		t.Fatalf("expected start_at 2025-07-01, got %+v", rate.StartAt)
	}
	if rate.IncrementStartAt == nil || rate.IncrementStartAt.Format("2006-01-02") != "2025-01-01" {
		t.Fatalf("expected increment_start_at 2025-01-01, got %+v", rate.IncrementStartAt)
	}
	if rate.CustomerFeeOwnerID == nil || *rate.CustomerFeeOwnerID != 42 {
		t.Fatalf("expected resolved customer owner id 42, got %+v", rate.CustomerFeeOwnerID)
	}
}

func TestImportCustomerRates_ReturnsMissingUsersThenAutoCreatesAndResumes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &settlementRatesServiceStub{
		lookupMissingByName: map[string]bool{"陈金荣": true},
		previewUsers: []service.MissingImportUser{
			{Alias: "陈金荣", SuggestedUsername: "chenjr"},
		},
		createdUsers: []service.CreatedImportUser{
			{Alias: "陈金荣", Username: "chenjr", Password: "Pwd123456"},
		},
	}
	ctl := NewSettlementRatesController(svc)

	firstBody, firstContentType := buildImportMultipartBody(t, "customer_rates.xlsx", writeCustomerRatesWorkbookWithOwner("陈金荣"))
	firstReq := httptest.NewRequest(http.MethodPost, "/api/v1/settlement/rates/customer/import", firstBody)
	firstReq.Header.Set("Content-Type", firstContentType)
	firstRec := httptest.NewRecorder()
	firstCtx, _ := gin.CreateTestContext(firstRec)
	firstCtx.Request = firstReq

	ctl.ImportCustomerRates(firstCtx)

	if firstRec.Code != http.StatusOK {
		t.Fatalf("expected first response 200, got %d body=%s", firstRec.Code, firstRec.Body.String())
	}
	var firstResp struct {
		Stage              string                      `json:"stage"`
		CanAutoCreateUsers bool                        `json:"can_auto_create_users"`
		ResumableToken     string                      `json:"resumable_token"`
		MissingUsers       []service.MissingImportUser `json:"missing_users"`
	}
	if err := json.Unmarshal(firstRec.Body.Bytes(), &firstResp); err != nil {
		t.Fatalf("unmarshal first response: %v", err)
	}
	if firstResp.Stage != "needs_user_creation" {
		t.Fatalf("expected stage needs_user_creation, got %s body=%s", firstResp.Stage, firstRec.Body.String())
	}
	if !firstResp.CanAutoCreateUsers {
		t.Fatalf("expected can_auto_create_users=true")
	}
	if firstResp.ResumableToken == "" {
		t.Fatalf("expected resumable token")
	}
	if len(firstResp.MissingUsers) != 1 || firstResp.MissingUsers[0].Alias != "陈金荣" {
		t.Fatalf("unexpected missing users: %+v", firstResp.MissingUsers)
	}

	secondBody := &bytes.Buffer{}
	secondWriter := multipart.NewWriter(secondBody)
	if err := secondWriter.WriteField("resumable_token", firstResp.ResumableToken); err != nil {
		t.Fatalf("write resumable token: %v", err)
	}
	if err := secondWriter.WriteField("auto_create_missing_users", "1"); err != nil {
		t.Fatalf("write auto create flag: %v", err)
	}
	if err := secondWriter.Close(); err != nil {
		t.Fatalf("close second writer: %v", err)
	}
	secondReq := httptest.NewRequest(http.MethodPost, "/api/v1/settlement/rates/customer/import", secondBody)
	secondReq.Header.Set("Content-Type", secondWriter.FormDataContentType())
	secondRec := httptest.NewRecorder()
	secondCtx, _ := gin.CreateTestContext(secondRec)
	secondCtx.Request = secondReq

	ctl.ImportCustomerRates(secondCtx)

	if secondRec.Code != http.StatusOK {
		t.Fatalf("expected second response 200, got %d body=%s", secondRec.Code, secondRec.Body.String())
	}
	var secondResp struct {
		Stage        string                      `json:"stage"`
		Affected     int                         `json:"affected"`
		CreatedUsers []service.CreatedImportUser `json:"created_users"`
	}
	if err := json.Unmarshal(secondRec.Body.Bytes(), &secondResp); err != nil {
		t.Fatalf("unmarshal second response: %v", err)
	}
	if secondResp.Stage != "completed" {
		t.Fatalf("expected stage completed, got %s body=%s", secondResp.Stage, secondRec.Body.String())
	}
	if secondResp.Affected != 1 {
		t.Fatalf("expected affected=1, got %d body=%s", secondResp.Affected, secondRec.Body.String())
	}
	if len(secondResp.CreatedUsers) != 1 || secondResp.CreatedUsers[0].Username != "chenjr" {
		t.Fatalf("unexpected created users: %+v", secondResp.CreatedUsers)
	}
	if len(svc.createMissingUsers) != 1 || svc.createMissingUsers[0].Alias != "陈金荣" {
		t.Fatalf("expected create called with 陈金荣, got %+v", svc.createMissingUsers)
	}
	if len(svc.upsertedRates) != 1 {
		t.Fatalf("expected resumed import to upsert 1 rate, got %d", len(svc.upsertedRates))
	}
}

func writeCustomerRatesWorkbook(w io.Writer) error {
	return writeCustomerRatesWorkbookWithOwner("Alice")(w)
}

func writeCustomerRatesWorkbookWithOwner(owner string) func(io.Writer) error {
	return func(w io.Writer) error {
		f := excelize.NewFile()
		sheet := f.GetSheetName(0)
		header := []string{
			"区域", "CP", "学校",
			"客户费率", "线路费率", "节点通用费率", "渠道费率",
			"客户费归属", "线路费归属", "节点通用费归属", "渠道费归属",
			"存量起算日期", "增量起算日期", "存量占比", "增量占比",
			"客户费归属用户ID", "线路费归属用户ID", "节点通用费归属用户ID", "渠道费归属用户ID",
		}
		for i, h := range header {
			cell, _ := excelize.CoordinatesToCellName(i+1, 1)
			f.SetCellValue(sheet, cell, h)
		}
		row := []string{
			"河南省", "bilibili", "中原工学院",
			"1,200.00", "1,300.00", "2,500.00", "500.00 ",
			owner, "", "", "",
			"07-01-25", "01-01-25", "70%", "30%",
			"999", "", "", "",
		}
		for i, v := range row {
			cell, _ := excelize.CoordinatesToCellName(i+1, 2)
			f.SetCellValue(sheet, cell, v)
		}
		blankRow := 3
		for i := 1; i <= len(header); i++ {
			cell, _ := excelize.CoordinatesToCellName(i, blankRow)
			f.SetCellValue(sheet, cell, "")
		}
		return f.Write(w)
	}
}

func buildImportMultipartBody(t *testing.T, filename string, writeFile func(io.Writer) error) (*bytes.Buffer, string) {
	t.Helper()
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		t.Fatalf("create form file: %v", err)
	}
	if err := writeFile(part); err != nil {
		t.Fatalf("write file: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close writer: %v", err)
	}
	return body, writer.FormDataContentType()
}

func cloneRateCustomer(in *model.RateCustomer) *model.RateCustomer {
	if in == nil {
		return nil
	}
	cp := *in
	return &cp
}

func assertFloatPtrEquals(t *testing.T, ptr *float64, want float64) {
	t.Helper()
	if ptr == nil {
		t.Fatalf("expected %v, got nil", want)
	}
	if *ptr != want {
		t.Fatalf("expected %v, got %v", want, *ptr)
	}
}

var _ service.RatesService = (*settlementRatesServiceStub)(nil)
