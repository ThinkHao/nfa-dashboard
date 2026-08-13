package controller

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"nfa-dashboard/internal/model"
	"nfa-dashboard/internal/service"
)

type settlementServiceForNodeTaskTest struct {
	service.SettlementService
	createdTypes []string
	createdDates []time.Time
}

func (s *settlementServiceForNodeTaskTest) GetSettlementConfig() (*model.SettlementConfig, error) {
	return nil, nil
}
func (s *settlementServiceForNodeTaskTest) UpdateSettlementConfig(config *model.SettlementConfig) error {
	return nil
}
func (s *settlementServiceForNodeTaskTest) CreateSettlementTask(taskType string, taskDate time.Time) (*model.SettlementTask, error) {
	s.createdTypes = append(s.createdTypes, taskType)
	s.createdDates = append(s.createdDates, taskDate)
	return &model.SettlementTask{ID: 99, TaskType: taskType, TaskDate: taskDate, Status: "pending"}, nil
}
func (s *settlementServiceForNodeTaskTest) UpdateSettlementTaskStatus(taskID int64, status string, errorMsg string) error {
	return nil
}
func (s *settlementServiceForNodeTaskTest) DeleteSettlementTask(id int64) error { return nil }
func (s *settlementServiceForNodeTaskTest) GetSettlementTasks(taskType, status string, startDate, endDate time.Time, limit, offset int) ([]model.SettlementTaskResponse, int64, error) {
	return nil, 0, nil
}
func (s *settlementServiceForNodeTaskTest) GetSettlementTaskByID(id int64) (*model.SettlementTaskResponse, error) {
	return nil, nil
}
func (s *settlementServiceForNodeTaskTest) GetSettlements(filter model.SettlementFilter) ([]model.SettlementResponse, int64, error) {
	return nil, 0, nil
}
func (s *settlementServiceForNodeTaskTest) ExecuteDailySettlement(taskID int64, date time.Time) error {
	return nil
}
func (s *settlementServiceForNodeTaskTest) ExecuteWeeklySettlement(taskID int64, weekStartDate time.Time) error {
	return nil
}
func (s *settlementServiceForNodeTaskTest) ExecuteWeeklySettlementWithDateRange(taskID int64, startDate, endDate time.Time) error {
	return nil
}
func (s *settlementServiceForNodeTaskTest) GetValidSchoolComboCount(userID *uint64) (int64, error) {
	return 0, nil
}

type edcNodeSettlementServiceForTaskTest struct {
	hasTrafficCalls       int
	executedTaskID        int64
	executedStart         time.Time
	executedEnd           time.Time
	executedMonthlyTaskID int64
	executedMonthStart    time.Time
	executedMonthEnd      time.Time
	executed              chan struct{}
	executedMonthly       chan struct{}
	settlementReady       bool
}

func (s *edcNodeSettlementServiceForTaskTest) HasSettlementTraffic(start, end time.Time) (bool, error) {
	s.hasTrafficCalls++
	return true, nil
}
func (s *edcNodeSettlementServiceForTaskTest) CheckSettlementReadiness(start, end time.Time) (bool, error) {
	return s.settlementReady, nil
}
func (s *edcNodeSettlementServiceForTaskTest) ExecuteDailyTask(taskID int64, day time.Time) error {
	return nil
}
func (s *edcNodeSettlementServiceForTaskTest) ExecuteDailyRangeTask(taskID int64, start, end time.Time) error {
	s.executedTaskID = taskID
	s.executedStart = start
	s.executedEnd = end
	close(s.executed)
	return nil
}
func (s *edcNodeSettlementServiceForTaskTest) ExecuteMonthlyTask(taskID int64, month time.Time) error {
	return nil
}
func (s *edcNodeSettlementServiceForTaskTest) ExecuteMonthlyRangeTask(taskID int64, start, end time.Time) error {
	s.executedMonthlyTaskID = taskID
	s.executedMonthStart = start
	s.executedMonthEnd = end
	close(s.executedMonthly)
	return nil
}
func (s *edcNodeSettlementServiceForTaskTest) ListDailySettlements(filter map[string]interface{}, page, pageSize int) ([]model.SettlementNodeDaily95, int64, error) {
	return nil, 0, nil
}
func (s *edcNodeSettlementServiceForTaskTest) ListMonthlySettlements(filter map[string]interface{}, page, pageSize int) ([]model.SettlementNodeMonthly95, int64, error) {
	return nil, 0, nil
}

func TestCreateNodeDailyTaskRangeCreatesSingleTask(t *testing.T) {
	gin.SetMode(gin.TestMode)
	settlementSvc := &settlementServiceForNodeTaskTest{}
	nodeSvc := &edcNodeSettlementServiceForTaskTest{executed: make(chan struct{}), executedMonthly: make(chan struct{}), settlementReady: true}
	controller := NewEDCNodeSettlementController(settlementSvc, nodeSvc)

	rec := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(rec)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/api/v1/settlement/tasks/node-daily95", strings.NewReader(`{"start_date":"2026-06-01","end_date":"2026-06-30"}`))
	ctx.Request.Header.Set("Content-Type", "application/json")

	controller.CreateNodeDailyTask(ctx)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s, want 200", rec.Code, rec.Body.String())
	}
	if len(settlementSvc.createdTypes) != 1 {
		t.Fatalf("created task count=%d, want 1", len(settlementSvc.createdTypes))
	}
	if settlementSvc.createdTypes[0] != "node_daily95" {
		t.Fatalf("created task type=%s, want node_daily95", settlementSvc.createdTypes[0])
	}
	if got := settlementSvc.createdDates[0].Format("2006-01-02"); got != "2026-06-01" {
		t.Fatalf("task date=%s, want 2026-06-01", got)
	}
	select {
	case <-nodeSvc.executed:
	case <-time.After(time.Second):
		t.Fatalf("range task was not executed")
	}
	if nodeSvc.executedTaskID != 99 {
		t.Fatalf("executed task id=%d, want 99", nodeSvc.executedTaskID)
	}
	if got := nodeSvc.executedStart.Format("2006-01-02"); got != "2026-06-01" {
		t.Fatalf("executed start=%s, want 2026-06-01", got)
	}
	if got := nodeSvc.executedEnd.Format("2006-01-02"); got != "2026-06-30" {
		t.Fatalf("executed end=%s, want 2026-06-30", got)
	}
	if nodeSvc.hasTrafficCalls != 1 {
		t.Fatalf("traffic precheck calls=%d, want 1", nodeSvc.hasTrafficCalls)
	}
}

func TestCreateNodeMonthlyTaskRangeCreatesSingleTask(t *testing.T) {
	gin.SetMode(gin.TestMode)
	settlementSvc := &settlementServiceForNodeTaskTest{}
	nodeSvc := &edcNodeSettlementServiceForTaskTest{executed: make(chan struct{}), executedMonthly: make(chan struct{}), settlementReady: true}
	controller := NewEDCNodeSettlementController(settlementSvc, nodeSvc)

	rec := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(rec)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/api/v1/settlement/tasks/node-monthly95", strings.NewReader(`{"start_month":"2026-01","end_month":"2026-12"}`))
	ctx.Request.Header.Set("Content-Type", "application/json")

	controller.CreateNodeMonthlyTask(ctx)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s, want 200", rec.Code, rec.Body.String())
	}
	if len(settlementSvc.createdTypes) != 1 {
		t.Fatalf("created task count=%d, want 1", len(settlementSvc.createdTypes))
	}
	if settlementSvc.createdTypes[0] != "node_monthly95" {
		t.Fatalf("created task type=%s, want node_monthly95", settlementSvc.createdTypes[0])
	}
	if got := settlementSvc.createdDates[0].Format("2006-01"); got != "2026-01" {
		t.Fatalf("task month=%s, want 2026-01", got)
	}
	select {
	case <-nodeSvc.executedMonthly:
	case <-time.After(time.Second):
		t.Fatalf("monthly range task was not executed")
	}
	if nodeSvc.executedMonthlyTaskID != 99 {
		t.Fatalf("executed monthly task id=%d, want 99", nodeSvc.executedMonthlyTaskID)
	}
	if got := nodeSvc.executedMonthStart.Format("2006-01"); got != "2026-01" {
		t.Fatalf("executed month start=%s, want 2026-01", got)
	}
	if got := nodeSvc.executedMonthEnd.Format("2006-01"); got != "2026-12" {
		t.Fatalf("executed month end=%s, want 2026-12", got)
	}
	if nodeSvc.hasTrafficCalls != 1 {
		t.Fatalf("traffic precheck calls=%d, want 1", nodeSvc.hasTrafficCalls)
	}
}

func TestCreateNodeDailyTaskBlocksWhenEDCBackfillIsNotReady(t *testing.T) {
	gin.SetMode(gin.TestMode)
	settlementSvc := &settlementServiceForNodeTaskTest{}
	nodeSvc := &edcNodeSettlementServiceForTaskTest{
		executed: make(chan struct{}), executedMonthly: make(chan struct{}), settlementReady: false,
	}
	controller := NewEDCNodeSettlementController(settlementSvc, nodeSvc)

	rec := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(rec)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/api/v1/settlement/tasks/node-daily95", strings.NewReader(`{"date":"2026-06-01"}`))
	ctx.Request.Header.Set("Content-Type", "application/json")

	controller.CreateNodeDailyTask(ctx)

	if rec.Code != http.StatusConflict {
		t.Fatalf("status=%d body=%s, want 409", rec.Code, rec.Body.String())
	}
	if len(settlementSvc.createdTypes) != 0 {
		t.Fatalf("created task count=%d, want 0", len(settlementSvc.createdTypes))
	}
}
